package cmd

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"connectrpc.com/authn"
	"github.com/getsentry/sentry-go"
	"github.com/leighmacdonald/gbans/internal/anticheat"
	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/auth"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/ban/bantype"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	"github.com/leighmacdonald/gbans/internal/blocklist"
	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/contest"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/demo"
	"github.com/leighmacdonald/gbans/internal/discord"
	discordoauth "github.com/leighmacdonald/gbans/internal/discord/oauth"
	"github.com/leighmacdonald/gbans/internal/forum"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/log"
	"github.com/leighmacdonald/gbans/internal/maps"
	"github.com/leighmacdonald/gbans/internal/metrics"
	"github.com/leighmacdonald/gbans/internal/mge"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/network/scp"
	"github.com/leighmacdonald/gbans/internal/news"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/sourcemod"
	"github.com/leighmacdonald/gbans/internal/speedruns"
	"github.com/leighmacdonald/gbans/internal/stats"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/internal/votes"
	"github.com/leighmacdonald/gbans/internal/wiki"
	"github.com/leighmacdonald/gbans/pkg/broadcaster"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/sosodev/duration"
)

var (
	BuildVersion = "master" //nolint:gochecknoglobals
	BuildCommit  = ""       //nolint:gochecknoglobals
	BuildDate    = ""       //nolint:gochecknoglobals
)

type GBans struct {
	anticheat      anticheat.AntiCheat
	assets         asset.Assets
	appeals        ban.Appeals
	banExpirations *ban.ExpirationMonitor
	bans           ban.Bans
	blocklists     blocklist.Blocklists
	chat           *chat.Chat
	config         *config.Configuration
	contests       contest.Contests
	database       database.Database
	demos          demo.Demos
	forums         forum.Forums
	discordOAuth   discordoauth.DiscordOAuth
	memberships    *ban.Memberships
	metrics        metrics.Metrics
	mge            mge.MGE
	networks       network.Networks
	news           news.News
	notifications  *notification.Notifications
	persons        *person.Persons
	reports        ban.Reports
	servers        *servers.Servers
	speedruns      speedruns.Speedruns
	sourcemod      sourcemod.Sourcemod
	stats          stats.Stats
	staticConfig   config.Static
	tfapiClient    thirdparty.APIProvider
	votes          votes.Votes
	wiki           wiki.Wiki
	wordFilters    chat.WordFilters
	sentry         *sentry.Client
	bot            discord.Connection

	broadcaster *broadcaster.Broadcaster[logparse.EventType, logparse.ServerEvent]

	logCloser func()
}

func New() (*GBans, error) {
	staticConfig, errStatic := config.ReadStaticConfig()
	if errStatic != nil {
		slog.Error("Failed to read static config", slog.String("error", errStatic.Error()))

		return nil, errStatic
	}

	return &GBans{
		staticConfig: staticConfig,
		broadcaster:  broadcaster.New[logparse.EventType, logparse.ServerEvent](),
		database:     database.New(staticConfig.DatabaseDSN, staticConfig.DatabaseAutoMigrate, staticConfig.DatabaseLogQueries),
	}, nil
}

func (g *GBans) Init(ctx context.Context) error {
	if errConnect := g.database.Connect(ctx); errConnect != nil {
		slog.Error("Cannot initialize database", slog.String("error", errConnect.Error()))

		return errConnect
	}

	configuration, errConfig := g.createConfig(ctx)
	if errConfig != nil {
		return errConfig
	}
	g.config = configuration

	conf := g.config.Config()

	g.setupSentry()
	if conf.General.Mode == config.TestMode {
		slog.SetDefault(slog.New(slog.DiscardHandler))
		g.logCloser = func() {}
	} else {
		g.logCloser = log.MustCreateLogger(ctx, conf.Log.File, conf.Log.Level, conf.General.SentryDSN != "", BuildVersion)
	}
	slog.Debug("Starting gbans...",
		slog.String("version", BuildVersion),
		slog.String("commit", BuildCommit),
		slog.String("date", BuildDate))

	tfapiClient, errClient := g.createAPIClient()
	if errClient != nil {
		return errClient
	}
	g.tfapiClient = tfapiClient

	g.persons = person.NewPersons(person.NewRepository(g.database, conf.Clientprefs.CenterProjectiles), steamid.New(conf.Owner), g.tfapiClient)
	g.bot = g.mustCreateBot(conf.Discord)
	g.notifications = notification.NewNotifications(notification.NewRepository(g.database), g.bot)

	wordFilters := chat.NewWordFilters(chat.NewWordFilterRepository(g.database), g.notifications, conf.Filters)
	if err := wordFilters.Import(ctx); err != nil {
		slog.Error("Failed to load word filters", slog.String("error", err.Error()))

		return err
	}
	g.wordFilters = wordFilters

	g.networks = network.NewNetworks(g.broadcaster, network.NewRepository(g.database, g.persons), conf.Network, conf.GeoLocation)

	assetRepo := asset.NewLocalRepository(g.database, conf.LocalStore.PathRoot)
	if err := assetRepo.Init(ctx); err != nil {
		slog.Error("Failed to init local asset repo", slog.String("error", err.Error()))

		return err
	}

	g.assets = asset.NewAssets(assetRepo)

	var errServer error
	if g.servers, errServer = servers.New(servers.NewRepository(g.database), g.broadcaster, conf.General.SrcdsLogAddr); errServer != nil {
		return errServer
	}

	mapsSvc := maps.New(maps.NewRepository(g.database))

	g.stats = stats.New(stats.NewRepository(g.database), mapsSvc)

	g.chat = chat.New(chat.NewRepository(g.database), conf.Filters, g.wordFilters, g.persons, g.notifications, g.chatHandler, conf.Discord.SafeChatLogChannelID())
	g.demos = demo.NewDemos(asset.BucketDemo, demo.NewRepository(g.database), g.assets, g.stats, g.chat, g.persons, conf.Demo, steamid.New(conf.Owner))
	g.reports = ban.NewReports(ban.NewReportRepository(g.database), g.persons, g.demos, g.tfapiClient, g.notifications,
		conf.Discord.SafeAppealLogChannelID())

	g.bans = ban.New(ban.NewRepository(g.database), g.persons, conf.Discord.SafeBanLogChannelID(),
		conf.Discord.SafeKickLogChannelID(), steamid.New(conf.Owner), g.reports, g.notifications, g.servers, g.networks)
	g.blocklists = blocklist.NewBlocklists(blocklist.NewRepository(g.database),
		ban.NewGroupMemberships(tfapiClient, ban.NewRepository(g.database)))
	g.discordOAuth = discordoauth.NewOAuth(discordoauth.NewRepository(g.database), conf.Discord)
	g.forums = forum.New(forum.NewRepository(g.database), g.notifications, g.persons, "")
	g.metrics = metrics.New(g.broadcaster)
	g.news = news.New(news.NewRepository(g.database), g.notifications, conf.Discord.SafePublicLogChannelID())
	g.sourcemod = sourcemod.New(sourcemod.NewRepository(g.database), g.persons, g.notifications, conf.Discord.SafeSeedChannelID(), conf.Discord.LogChannelID, conf.Discord.SafeModPingRoleID(), g.servers)
	g.wiki = wiki.New(wiki.NewRepository(g.database), g.notifications, conf.Discord.SafePublicLogChannelID(), conf.Discord.LogChannelID)
	g.anticheat = anticheat.New(anticheat.NewRepository(g.database), conf.Anticheat, g.notifications, g.onAnticheatBan, g.persons)
	g.votes = votes.New(votes.NewRepository(g.database), g.broadcaster, g.notifications,
		conf.Discord.SafeVoteLogChannelID(), g.persons)

	g.speedruns = speedruns.NewSpeedruns(speedruns.NewSpeedrunRepository(g.database, g.persons), mapsSvc)
	g.memberships = ban.NewMemberships(ban.NewRepository(g.database), g.tfapiClient)
	g.banExpirations = ban.NewExpirationMonitor(g.bans, g.persons, g.notifications)
	g.mge = mge.NewMGE(mge.NewRepository(g.database))
	g.appeals = ban.NewAppeals(ban.NewAppealRepository(g.database), g.bans, g.persons, g.notifications, conf.Discord.SafeAppealLogChannelID())

	if conf.Discord.Enabled {
		anticheat.RegisterDiscordCommands(g.bot, g.anticheat)
		auth.RegisterDiscordCommands(g.bot)
		ban.RegisterDiscordCommands(g.bot, g.bans, g.persons, g.persons)
		chat.RegisterDiscordCommands(g.bot, g.wordFilters)
		forum.RegisterDiscordCommands(g.bot)
		news.RegisterDiscordCommands(g.bot)
		servers.RegisterDiscordCommands(g.bot, g.persons, g.servers, g.networks, g.notifications, conf.Discord.SafeKickLogChannelID())
		sourcemod.RegisterDiscordCommands(g.bot, g.sourcemod, g.servers)
		votes.RegisterDiscordCommands(g.bot)
		wiki.RegisterDiscordCommands(g.bot)
	}

	if err := g.firstTimeSetup(ctx); err != nil {
		slog.Error("Failed to run first time setup", slog.String("error", err.Error()))

		return err
	}

	if errRoles := g.createDiscordRoles(ctx); errRoles != nil {
		slog.Error("Failed to register discord roles", slog.String("error", errRoles.Error()))
	}

	return nil
}

func (g *GBans) createDiscordRoles(ctx context.Context) error {
	conf := g.config.Config().Discord
	if !conf.BotEnabled {
		return nil
	}

	curServers, errServers := g.servers.Servers(ctx, servers.Query{})
	if errServers != nil {
		return errServers
	}

	curRoles, errRoles := g.bot.Roles()
	if errRoles != nil {
		return errRoles
	}

	names := map[string]string{}
	for _, server := range curServers {
		name := "seeder-" + servers.ShortNamePrefix(server.ShortName)
		existingID := ""
		for _, role := range curRoles {
			if strings.EqualFold(role.Name, name) {
				existingID = role.ID
			}
		}

		roleID, found := names[name]
		if !found || roleID == "" {
			if existingID == "" {
				newRoleID, err := g.bot.CreateRole(name)
				if err != nil {
					return err
				}
				roleID = newRoleID
			} else {
				roleID = existingID
			}
			names[name] = roleID
		}

		server.DiscordSeedRoleIDs = []string{roleID}
		if _, errSave := g.servers.Save(ctx, server); errSave != nil {
			return errSave
		}
	}

	return nil
}

func (g *GBans) chatHandler(ctx context.Context, exceeded bool, newWarning chat.NewUserWarning) error {
	if !newWarning.MatchedFilter.IsEnabled {
		return nil
	}

	if !exceeded {
		const msg = "[WARN] Please refrain from using slurs/toxicity (see: rules & MOTD). " +
			"Further offenses will result in mutes/bans"
		if result, found := g.servers.FindPlayer(servers.FindOpts{SteamID: newWarning.UserMessage.SteamID}); found {
			opts := servers.SayOpts{
				Type:    servers.PSay,
				Message: msg,
				Targets: []steamid.SteamID{newWarning.UserMessage.SteamID},
			}
			if errPSay := result.Server.Say(ctx, opts); errPSay != nil {
				return errPSay
			}
		}

		return nil
	}

	slog.Info("Warn limit exceeded",
		slog.String("sid64", newWarning.UserMessage.SteamID.String()),
		slog.Int("weight", int(newWarning.CurrentTotal)))

	return nil
}

func (g *GBans) createConfig(ctx context.Context) (*config.Configuration, error) {
	conf, errConfig := config.NewConfiguration(ctx, g.staticConfig, config.NewRepository(g.database))
	if errConfig != nil {
		return nil, errConfig
	}

	return conf, nil
}

func (g *GBans) createAPIClient() (thirdparty.APIProvider, error) {
	apiURL := os.Getenv("TFAPI_URL")
	if apiURL == "" {
		apiURL = "https://tf-api.roto.lol"
	}

	tfapiClient, errClient := thirdparty.NewTFAPI(apiURL, &http.Client{Timeout: time.Second * 15})
	if errClient != nil {
		return nil, errClient
	}

	return tfapiClient, nil
}

func (g *GBans) mustCreateBot(conf *discord.Config) discord.Connection {
	if conf.BotEnabled {
		discordBot, errDiscord := discord.New(discord.Opts{
			Token:   conf.Token,
			AppID:   conf.AppID,
			GuildID: conf.GuildID,
		})
		if errDiscord != nil {
			panic(errDiscord)
		}

		return discordBot
	}

	return discord.Discard{}
}

func (g *GBans) startBot() {
	if errStart := g.bot.Start(); errStart != nil {
		slog.Error("Failed to start bot", slog.String("error", errStart.Error()))
	}
}

func (g *GBans) setupSentry() {
	dsn := g.config.Config().General.SentryDSN
	if dsn != "" {
		sentryClient, err := log.NewSentryClient(dsn, true, 0.25, BuildVersion, string(g.config.Config().General.Mode))
		if err != nil {
			slog.Error("Failed to setup sentry client")
		} else {
			slog.Info("Sentry.io support is enabled.")
			g.sentry = sentryClient
		}
	} else {
		slog.Debug("Sentry.io support is disabled. To enable at runtime, set SENTRY_DSN.")
	}
}

func (g *GBans) StartBackground(ctx context.Context) {
	conf := g.config.Config()

	if conf.Debug.AddRCONLogAddress != "" {
		g.servers.Each(func(server *servers.Server) error {
			return server.LogAddressAdd(ctx, conf.Debug.AddRCONLogAddress)
		})
	}

	go g.chat.Start(ctx, g.broadcaster)
	go g.forums.Start(ctx)
	go g.metrics.Start(ctx)
	go g.votes.Start(ctx)
	go g.networks.Start(ctx)
	go g.notifications.Sender(ctx)

	go downloadManager(ctx, g.database, conf.SSH, g.demos, g.anticheat)

	go func() {
		if err := g.servers.Start(ctx, servers.DefaultStatusUpdateFreq); err != nil {
			slog.Error("Failed to start state tracker", slog.String("error", err.Error()))
		}
	}()

	if errSync := g.anticheat.SyncDemoIDs(ctx, 100); errSync != nil {
		slog.Error("failed to sync anticheat demos")
	}

	go g.memberships.Update(ctx)
	go g.banExpirations.Update(ctx)
	go g.blocklists.Sync(ctx)
	go g.demos.Cleanup(ctx)

	membershipsTicker := time.NewTicker(12 * time.Hour)
	defer membershipsTicker.Stop()
	expirationsTicker := time.NewTicker(60 * time.Second)
	defer expirationsTicker.Stop()
	reportIntoTicker := time.NewTicker(24 * time.Hour)
	defer reportIntoTicker.Stop()
	blocklistTicker := time.NewTicker(6 * time.Hour)
	defer blocklistTicker.Stop()
	demoTicker := time.NewTicker(15 * time.Minute)
	defer demoTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-membershipsTicker.C:
			go g.memberships.Update(ctx)
		case <-expirationsTicker.C:
			go g.banExpirations.Update(ctx)
		case <-reportIntoTicker.C:
			go func() {
				if errMeta := g.reports.MetaStats(ctx); errMeta != nil {
					slog.Error("Failed to generate meta stats", slog.String("error", errMeta.Error()))
				}
			}()
		case <-blocklistTicker.C:
			go g.blocklists.Sync(ctx)
		case <-demoTicker.C:
			go g.demos.Cleanup(ctx)
			go func() {
				if errSync := g.anticheat.SyncDemoIDs(ctx, 100); errSync != nil {
					slog.Error("failed to sync anticheat demos")
				}
			}()
		}
	}
}

func (g *GBans) createAPI(authMiddleware *rpc.Middleware) *http.ServeMux {
	interceptors := rpc.CreateInterceptors()
	api := http.NewServeMux()
	conf := g.config.Config()

	services := []rpc.Service{
		anticheat.NewService(g.anticheat, authMiddleware, interceptors),
		asset.NewService(g.assets, authMiddleware, interceptors),
		auth.NewService(authMiddleware, interceptors),
		ban.NewAppealService(g.appeals, authMiddleware, interceptors),
		ban.NewBanService(g.bans, authMiddleware, interceptors),
		ban.NewExportService(g.bans, strings.Split(conf.Exports.AuthorizedKeys, ","), conf.General.SiteName),
		ban.NewReportService(g.reports, authMiddleware, interceptors),
		chat.NewService(g.chat, authMiddleware, interceptors),
		chat.NewWordfilterService(g.wordFilters, g.chat, g.config.Config().Filters, authMiddleware, interceptors),
		config.NewService(g.config, BuildVersion, authMiddleware, interceptors),
		contest.NewService(g.contests, g.assets, authMiddleware, interceptors),
		discord.NewService(g.bot, authMiddleware, interceptors),
		forum.NewService(g.forums, authMiddleware, interceptors),
		mge.NewService(g.mge, authMiddleware, interceptors),
		blocklist.NewService(g.blocklists, authMiddleware, interceptors),
		network.NewNetworkService(g.networks, authMiddleware, interceptors),
		news.NewService(g.news, authMiddleware, interceptors),
		notification.NewService(g.notifications, authMiddleware, interceptors),
		person.NewPersonService(g.persons, authMiddleware, interceptors),
		servers.NewServersService(g.servers, authMiddleware, interceptors),
		demo.NewService(g.demos, authMiddleware, interceptors),
		speedruns.NewService(g.speedruns, authMiddleware, interceptors),
		sourcemod.NewPluginService(g.sourcemod, g.persons, g.servers, g.bans,
			rpc.NewServerTokenGenerator(conf.General.SiteName, []byte(conf.HTTPCookieKey)), g.notifications, conf.Discord.LogChannelID, authMiddleware, interceptors),
		sourcemod.NewSourcemodService(g.sourcemod, authMiddleware, interceptors),
		stats.NewService(g.stats, g.servers, authMiddleware, interceptors),
		votes.NewService(g.votes, authMiddleware, interceptors),
		wiki.NewService(g.wiki, authMiddleware, interceptors),
	}

	for _, service := range services {
		api.Handle(service.Pattern, service.Handler)
	}

	return api
}

func (g *GBans) Serve(rootCtx context.Context) error {
	ctx, stop := signal.NotifyContext(rootCtx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	conf := g.config.Config()

	if conf.Discord.Enabled {
		go g.startBot()
	}

	mux, router, err := httphelper.CreateRouter(httphelper.RouterOpts{
		HTTPLogEnabled:    conf.Log.HTTPEnabled,
		LogLevel:          conf.Log.Level,
		HTTPOtelEnabled:   conf.Log.HTTPOtelEnabled,
		SentryDSN:         g.config.Config().General.SentryDSN,
		Version:           BuildVersion,
		PProfEnabled:      conf.PProfEnabled,
		PrometheusEnabled: conf.PrometheusEnabled,
		FrontendEnable:    conf.General.Mode != config.TestMode,
		StaticPath:        conf.HTTPStaticPath,
		HTTPCORSEnabled:   conf.HTTPCORSEnabled,
		CORSOrigins:       conf.HTTPCorsOrigins,
	})
	if err != nil {
		slog.Error("Could not setup router", slog.String("error", err.Error()))

		return err
	}

	userAuth := auth.NewAuthentication(auth.NewRepository(g.database), conf.General.SiteName, conf.HTTPCookieKey, g.persons, g.bans, g.servers, g.config.Config().General.SentryDSN)

	authMiddleware := rpc.NewMiddleware(conf.General.SiteName, conf.HTTPCookieKey)

	asset.NewAssetHandler(mux, g.assets)
	auth.NewAuthHandler(mux, userAuth, g.config, g.tfapiClient, g.notifications, authMiddleware)
	discordoauth.NewDiscordOAuthHandler(mux, g.config, g.persons, g.discordOAuth)

	mux.HandleFunc("GET /health", g.healthCheck)

	apiHandler := g.createAPI(authMiddleware)

	topMux := http.NewServeMux()

	mw := authn.NewMiddleware(authMiddleware.Authenticate)

	topMux.Handle("/connect/", http.StripPrefix("/connect", mw.Wrap(apiHandler)))
	topMux.Handle("/", router)

	httpServer := httphelper.NewServer(conf.Addr(), topMux)

	go func() { //nolint:gosec
		<-ctx.Done()

		slog.Debug("Shutting down HTTP service")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*10) //nolint:contextcheck
		defer cancel()

		if errShutdown := httpServer.Shutdown(shutdownCtx); errShutdown != nil {
			slog.Error("Error shutting down http service", slog.String("error", errShutdown.Error()))
		}
	}()

	slog.Info("Starting HTTP server", slog.String("address", conf.Addr()), slog.String("url", conf.ExternalURL))

	if errServe := httpServer.ListenAndServe(); errServe != nil && !errors.Is(errServe, http.ErrServerClosed) {
		slog.Error("HTTP server returned error", slog.String("error", errServe.Error()))
	}

	<-ctx.Done()

	return nil
}

func (g *GBans) Shutdown(ctx context.Context) error {
	conf := g.config.Config()
	if conf.Debug.AddRCONLogAddress != "" {
		g.servers.Each(func(server *servers.Server) error {
			return server.LogAddressDel(ctx, conf.Debug.AddRCONLogAddress)
		})
	}

	if g.bot != nil {
		g.bot.Close()
	}

	if g.database != nil {
		if errClose := g.database.Close(); errClose != nil {
			slog.Error("Failed to close database cleanly", slog.String("error", errClose.Error()))
		}
	}

	if g.sentry != nil {
		g.sentry.Flush(2 * time.Second)
	}

	if g.logCloser != nil {
		g.logCloser()
	}

	return nil
}

func (g *GBans) firstTimeSetup(ctx context.Context) error {
	conf := g.config.Config()
	_, errRootUser := g.persons.BySteamID(ctx, steamid.New(conf.Owner))
	if errRootUser == nil {
		return nil
	}

	if !errors.Is(errRootUser, person.ErrPlayerDoesNotExist) {
		return errRootUser
	}

	owner := person.New(steamid.New(conf.Owner))
	owner.PermissionLevel = permission.Admin

	if errSave := g.persons.Save(ctx, &owner); errSave != nil {
		slog.Error("Failed create new owner", slog.String("error", errSave.Error()))
	}

	article := news.Article{
		Title:       "Welcome to gbans",
		BodyMD:      "This is an *example* **news** entry.",
		IsPublished: true,
		CreatedOn:   time.Now(),
		UpdatedOn:   time.Now(),
	}

	if errSave := g.news.Save(ctx, &article); errSave != nil {
		return errSave
	}

	page := wiki.Page{
		PermissionLevel: permission.Banned,
		Slug:            wiki.RootSlug,
		BodyMD:          "# Welcome to the wiki",
		Revision:        1,
		CreatedOn:       time.Now(),
		UpdatedOn:       time.Now(),
	}
	_, errSave := g.wiki.Save(ctx, page)
	if errSave != nil {
		slog.Error("Failed save example wiki entry", slog.String("error", errSave.Error()))
	}

	return nil
}

func (g *GBans) onChatBan(ctx context.Context, warning chat.NewUserWarning) error {
	var validUntil time.Time
	if warning.MatchedFilter.Action == chat.FilterActionBan || warning.MatchedFilter.Action == chat.FilterActionMute {
		parsedDur, errDur := duration.Parse(warning.MatchedFilter.Duration)
		if errDur != nil {
			return errors.Join(errDur, chat.ErrInvalidActionDuration)
		}
		validUntil = time.Now().Add(parsedDur.ToTimeDuration())
	}

	var (
		errBan error
		newBan ban.Ban
		req    = ban.Opts{
			TargetID:   warning.UserMessage.SteamID,
			Reason:     warning.WarnReason,
			ReasonText: "",
			Note:       "Automatic warning ban",
			ValidUntil: validUntil,
		}
	)
	switch warning.MatchedFilter.Action {
	case chat.FilterActionMute:
		req.BanType = bantype.NoComm
		newBan, errBan = g.bans.Create(ctx, req)
	case chat.FilterActionBan:
		req.BanType = bantype.Banned
		newBan, errBan = g.bans.Create(ctx, req)
	case chat.FilterActionKick:
		if result, found := g.servers.FindPlayer(servers.FindOpts{SteamID: warning.UserMessage.SteamID}); found {
			errBan = result.Server.Kick(ctx, result.Player.SID, warning.WarnReason.String())
		}
	}

	if errBan != nil {
		return errBan
	}

	admin, err := g.persons.GetOrCreatePersonBySteamID(ctx, steamid.New(g.config.Config().Owner))
	if err != nil {
		return err
	}

	_, errSave := g.wordFilters.Edit(ctx, admin, warning.MatchedFilter.FilterID, warning.MatchedFilter)
	if errSave != nil {
		return errSave
	}

	if !g.config.Config().Filters.PingDiscord {
		return nil
	}

	go g.notifications.Send(notification.NewDiscord(g.config.Config().Discord.SafeWordFilterLogChannelID(),
		chat.WarningMessage(warning, newBan.ValidUntil)))

	return nil
}

func (g *GBans) onAnticheatBan(ctx context.Context, entry logparse.StacEntry, dur time.Duration, count int32) error {
	conf := g.config.Config()
	demoFile, errDemo := g.demos.GetDemoByName(ctx, entry.DemoName)
	if errDemo != nil || entry.DemoID == nil || *entry.DemoID <= 0 {
		return errDemo
	}
	newBan, err := g.bans.Create(ctx, ban.Opts{
		Origin:      ban.System,
		SourceID:    steamid.New(conf.Owner),
		TargetID:    entry.SteamID,
		ValidUntil:  time.Now().Add(dur),
		BanType:     bantype.Banned,
		Reason:      reason.Cheating,
		ReasonText:  "",
		Note:        "```\n" + entry.Summary + "\n\nRaw log:\n" + entry.RawLog + "\n```",
		DemoID:      &demoFile.DemoID,
		DemoTick:    &entry.DemoTick,
		AnticheatID: &entry.AnticheatID,
		EvadeOk:     false,
		Name:        entry.Name,
	})
	if err != nil && !errors.Is(err, database.ErrDuplicate) {
		slog.Error("Failed to ban cheater", slog.String("detection", string(entry.Detection)),
			slog.Int64("steam_id", entry.SteamID.Int64()), slog.String("error", err.Error()))

		return err
	} else if newBan.BanID > 0 {
		slog.Info("Banned cheater", slog.String("detection", string(entry.Detection)),
			slog.String("steam_id", entry.SteamID.String()))
		g.notifications.Send(notification.NewDiscord(conf.Discord.AnticheatChannelID,
			anticheat.NewAnticheatTrigger(newBan.Note, conf.Anticheat.Action, entry, count)))
	}

	return nil
}

func (g *GBans) healthCheck(res http.ResponseWriter, _ *http.Request) {
	serverStates := g.servers.Current()
	if len(serverStates) > 0 {
		for _, server := range serverStates {
			if server.MaxPlayers > 0 {
				res.WriteHeader(http.StatusOK)
				_, _ = res.Write([]byte("😎"))

				return
			}
		}
		res.WriteHeader(http.StatusServiceUnavailable)
		_, _ = res.Write([]byte("🙅🏻‍♀️"))
	} else {
		res.WriteHeader(http.StatusOK)
		_, _ = res.Write([]byte("😎"))
	}
}

func downloadManager(ctx context.Context, store database.Database, conf *scp.Config, handlers ...scp.ConnectionHandler) {
	var (
		connections []scp.Connection
		repo        = scp.NewRepository(store)
		ticker      = time.NewTicker(time.Duration(conf.UpdateInterval) * time.Second)
	)

	defer func() {
		for _, handler := range connections {
			handler.Close()
		}
	}()

	for {
		select {
		case <-ticker.C:
			if !conf.Enabled {
				return
			}
			knownServers, errServers := repo.Servers(ctx)
			if errServers != nil {
				if errors.Is(errServers, database.ErrNoResult) {
					continue
				}

				slog.Error("Failed to query download servers", slog.String("error", errServers.Error()))

				continue
			}

			for _, server := range knownServers {
				actualAddr := scp.HostPart(server.Address)
				exists := false
				for _, conn := range connections {
					if conn.Address() == actualAddr {
						exists = true

						break
					}
				}

				if !exists {
					connection := scp.NewConnection(repo, conf, server)
					for _, handler := range handlers {
						connection.AddHandler(handler)
					}
					connections = append(connections, connection)
				}
			}

			start := time.Now()

			waitGroup := &sync.WaitGroup{}

			for _, handler := range connections {
				waitGroup.Go(func() {
					if err := handler.Update(ctx); err != nil {
						slog.Error("Error running scp handler", slog.String("error", err.Error()))
					}
				})
			}

			waitGroup.Wait()

			slog.Debug("SCP Update complete", slog.Duration("duration", time.Since(start)))
		case <-ctx.Done():
			return
		}
	}
}
