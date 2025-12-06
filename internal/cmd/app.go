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

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/anticheat"
	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/auth"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/ban/bantype"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/contest"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/discord"
	discordoauth "github.com/leighmacdonald/gbans/internal/discord/oauth"
	"github.com/leighmacdonald/gbans/internal/forum"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/log"
	"github.com/leighmacdonald/gbans/internal/metrics"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/network/asn"
	"github.com/leighmacdonald/gbans/internal/network/scp"
	"github.com/leighmacdonald/gbans/internal/news"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/patreon"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/playerqueue"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/sourcemod"
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
	SentryDSN    = ""       //nolint:gochecknoglobals
)

type BuildInfo struct {
	BuildVersion string
	Commit       string
	Date         string
}

type GBans struct {
	anticheat      anticheat.AntiCheat
	assets         asset.Assets
	banExpirations *ban.ExpirationMonitor
	bans           ban.Bans
	blocklists     network.Blocklists
	chat           *chat.Chat
	config         *config.Configuration
	database       database.Database
	demos          servers.Demos
	forums         forum.Forums
	discordOAuth   discordoauth.DiscordOAuth
	memberships    *ban.Memberships
	metrics        metrics.Metrics
	networks       network.Networks
	news           news.News
	notifications  *notification.Notifications
	persons        *person.Persons
	playerQueue    *playerqueue.Playerqueue
	reports        ban.Reports
	servers        *servers.Servers
	speedruns      servers.Speedruns
	sourcemod      sourcemod.Sourcemod
	staticConfig   config.Static
	tfapiClient    thirdparty.APIProvider
	votes          votes.Votes
	wiki           wiki.Wiki
	wordFilters    chat.WordFilters
	sentry         *sentry.Client
	bot            discord.Service

	broadcaster *broadcaster.Broadcaster[logparse.EventType, logparse.ServerEvent]

	logCloser func()
}

// New creates a new application instance.
func New() (*GBans, error) {
	staticConfig, errStatic := config.ReadStaticConfig()
	if errStatic != nil {
		slog.Error("Failed to read static config", slog.String("error", errStatic.Error()))

		return nil, errStatic
	}

	return &GBans{
		staticConfig: staticConfig,
		broadcaster:  broadcaster.New[logparse.EventType, logparse.ServerEvent](),
	}, nil
}

func (g *GBans) Init(ctx context.Context) error {
	dbConn := database.New(g.staticConfig.DatabaseDSN, g.staticConfig.DatabaseAutoMigrate, g.staticConfig.DatabaseLogQueries)
	if errConnect := dbConn.Connect(ctx); errConnect != nil {
		slog.Error("Cannot initialize database", slog.String("error", errConnect.Error()))

		return errConnect
	}
	g.database = dbConn

	configuration, errConfig := g.createConfig(ctx)
	if errConfig != nil {
		return errConfig
	}
	g.config = configuration

	// This is normally set by build time flags, but can be overwritten by the env var.
	if SentryDSN == "" {
		if value, found := os.LookupEnv("SENTRY_DSN"); found && value != "" {
			SentryDSN = value
		}
	}

	conf := g.config.Config()

	g.setupSentry()
	if conf.General.Mode == config.TestMode {
		slog.SetDefault(slog.New(slog.DiscardHandler))
		g.logCloser = func() {}
	} else {
		g.logCloser = log.MustCreateLogger(ctx, conf.Log.File, conf.Log.Level, SentryDSN != "", BuildVersion)
	}
	slog.Info("Starting gbans...",
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
	g.demos = servers.NewDemos(asset.BucketDemo, servers.NewDemoRepository(g.database), g.assets, conf.Demo, steamid.New(conf.Owner))
	g.reports = ban.NewReports(ban.NewReportRepository(g.database), g.persons, g.demos, g.tfapiClient, g.notifications,
		conf.Discord.SafeAppealLogChannelID())
	g.bans = ban.New(ban.NewRepository(g.database), g.persons, conf.Discord.SafeBanLogChannelID(),
		conf.Discord.SafeKickLogChannelID(), steamid.New(conf.Owner), g.reports, g.notifications, g.servers, g.networks)
	g.blocklists = network.NewBlocklists(network.NewBlocklistRepository(g.database),
		ban.NewGroupMemberships(tfapiClient, ban.NewRepository(g.database)))
	g.discordOAuth = discordoauth.NewOAuth(discordoauth.NewRepository(g.database), conf.Discord)
	g.chat = chat.New(chat.NewRepository(g.database), conf.Filters, g.wordFilters, g.persons, g.notifications, g.chatHandler, conf.Discord.SafeChatLogChannelID())
	g.forums = forum.New(forum.NewRepository(g.database), g.config, g.notifications)
	g.metrics = metrics.New(g.broadcaster)
	g.news = news.New(news.NewRepository(g.database), g.notifications, conf.Discord.SafePublicLogChannelID())
	g.sourcemod = sourcemod.New(sourcemod.NewRepository(g.database), g.persons, g.notifications, conf.Discord.SafeSeedChannelID(), g.servers)
	g.wiki = wiki.New(wiki.NewRepository(g.database), g.notifications, conf.Discord.SafePublicLogChannelID(), conf.Discord.LogChannelID)
	g.anticheat = anticheat.New(anticheat.NewRepository(g.database), conf.Anticheat, g.notifications, g.onAnticheatBan, g.persons)
	g.votes = votes.New(votes.NewRepository(g.database), g.broadcaster, g.notifications,
		conf.Discord.SafeVoteLogChannelID(), g.persons)
	g.speedruns = servers.NewSpeedruns(servers.NewSpeedrunRepository(g.database, g.persons))
	g.memberships = ban.NewMemberships(ban.NewRepository(g.database), g.tfapiClient)
	g.banExpirations = ban.NewExpirationMonitor(g.bans, g.persons, g.notifications)

	if conf.Discord.Enabled {
		anticheat.RegisterDiscordCommands(g.bot, g.anticheat)
		ban.RegisterDiscordCommands(g.bot, g.bans, g.persons, g.persons)
		chat.RegisterDiscordCommands(g.bot, g.wordFilters)
		servers.RegisterDiscordCommands(g.bot, g.persons, g.servers, g.networks, g.notifications, conf.Discord.SafeKickLogChannelID())
		sourcemod.RegisterDiscordCommands(g.bot, g.sourcemod, g.servers)
	}

	if err := g.firstTimeSetup(ctx); err != nil {
		slog.Error("Failed to run first time setup", slog.String("error", err.Error()))

		return err
	}

	// If we are using Valve SDR network, optionally enable the dynamic DNS update support to automatically
	// update the A record when a change is detected with the new public SDR IP.
	// if conf.Network.SDREnabled && conf.Network.SDRDNSEnabled {
	// 	// go dns.MonitorChanges(ctx, conf, stateUsecase, serversUC)
	// }

	// Config
	g.setupPlayerQueue(ctx)

	if errRoles := g.createDiscordRoles(ctx); errRoles != nil {
		slog.Error("Failed to register discord roles", slog.String("error", errRoles.Error()))
	}

	asnBlocker := asn.NewBlocker(asn.NewRepository(g.database))
	if err := asnBlocker.Save(ctx, asn.NewBlock(13335, "idk")); err != nil {
		panic(err)
	}

	return nil
}

// createDiscordRoles handles creating discord roles used for seeding requests from servers.
// Names are normalized, removing the trailing digit, so that a single region shares the same single role.
// Given a list of short server names, eg: xyz-1, zyz-2, abc-1, abc-2, tuv-1
// It will create the following, normalized set of roles: zyz, abc, tuv.
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
		slog.Int("weight", newWarning.CurrentTotal))

	return nil
}

func (g *GBans) createConfig(ctx context.Context) (*config.Configuration, error) {
	conf := config.NewConfiguration(g.staticConfig, config.NewRepository(g.database))
	if err := conf.Init(ctx); err != nil {
		slog.Error("Failed to init config", slog.String("error", err.Error()))

		return nil, err
	}

	if errConfig := conf.Reload(ctx); errConfig != nil {
		slog.Error("Failed to read config", slog.String("error", errConfig.Error()))

		return nil, errConfig
	}

	return conf, nil
}

func (g *GBans) createAPIClient() (thirdparty.APIProvider, error) { //noling:ireturn
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

func (g *GBans) mustCreateBot(conf discord.Config) discord.Service { //nolint:ireturn
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

func (g *GBans) setupPlayerQueue(ctx context.Context) {
	playerQueueRepo := playerqueue.NewRepository(g.database, g.persons)
	// Pre-load some messages into queue message cache
	chatlogs, errChatlogs := playerQueueRepo.Query(ctx, playerqueue.QueryOpts{Filter: query.Filter{Limit: 100}})
	if errChatlogs != nil {
		slog.Error("Failed to warm playerqueue chatlogs", slog.String("error", errChatlogs.Error()))
		chatlogs = []playerqueue.ChatLog{}
	}
	g.playerQueue = playerqueue.NewPlayerqueue(ctx, playerQueueRepo, g.persons, g.servers, chatlogs, g.config.Config().Discord.PlayerqueueChannelID, g.notifications)
}

func (g *GBans) setupSentry() {
	if SentryDSN != "" {
		sentryClient, err := log.NewSentryClient(SentryDSN, true, 0.25, BuildVersion, string(g.config.Config().General.Mode))
		if err != nil {
			slog.Error("Failed to setup sentry client")
		} else {
			slog.Info("Sentry.io support is enabled.")
			g.sentry = sentryClient
		}
	} else {
		slog.Info("Sentry.io support is disabled. To enable at runtime, set SENTRY_DSN.")
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
	go g.playerQueue.Start(ctx)
	go g.networks.Start(ctx)
	go g.notifications.Sender(ctx)

	go downloadManager(ctx, time.Minute*5, g.database, conf.SSH, g.demos, g.anticheat)

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
	expirationsTicker := time.NewTicker(60 * time.Second)
	reportIntoTicker := time.NewTicker(24 * time.Hour)
	blocklistTicker := time.NewTicker(6 * time.Hour)
	demoTicker := time.NewTicker(15 * time.Minute)

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
		if errSync := g.anticheat.SyncDemoIDs(ctx, 100); errSync != nil {
			slog.Error("failed to sync anticheat demos")
		}
	}
}

func (g *GBans) Serve(rootCtx context.Context) error {
	ctx, stop := signal.NotifyContext(rootCtx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	conf := g.config.Config()

	if conf.Discord.Enabled {
		go g.startBot()
	}

	router, err := httphelper.CreateRouter(httphelper.RouterOpts{
		HTTPLogEnabled:    conf.Log.HTTPEnabled,
		LogLevel:          conf.Log.Level,
		HTTPOtelEnabled:   conf.Log.HTTPOtelEnabled,
		SentryDSN:         SentryDSN,
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

	// Create authentication middlewares
	userAuth := auth.NewAuthentication(auth.NewRepository(g.database), conf.General.SiteName, conf.HTTPCookieKey, g.persons, g.bans, g.servers, SentryDSN)
	serverAuth := servers.NewServerAuth(g.servers, SentryDSN)

	// Register all our handlers with router
	anticheat.NewAnticheatHandler(router, userAuth, g.anticheat)
	asset.NewAssetHandler(router, userAuth, g.assets)
	auth.NewAuthHandler(router, userAuth, g.config, g.tfapiClient, g.notifications)
	ban.NewAppealHandler(router, userAuth, ban.NewAppeals(ban.NewAppealRepository(g.database), g.bans, g.persons, g.notifications, conf.Discord.LogChannelID))
	ban.NewReportHandler(router, userAuth, g.reports)
	ban.NewHandlerBans(router, userAuth, g.bans, conf.Exports, conf.General.SiteName)
	chat.NewChatHandler(router, g.chat, userAuth)
	chat.NewWordFilterHandler(router, userAuth, conf.Filters, g.wordFilters, g.chat)
	config.NewHandler(router, userAuth, g.config, BuildVersion)
	contest.NewContestHandler(router, userAuth, contest.NewContests(contest.NewRepository(g.database)), g.assets)
	discordoauth.NewDiscordOAuthHandler(router, userAuth, g.config, g.persons, g.discordOAuth)
	forum.NewForumHandler(router, userAuth, g.forums)
	// match.NewMatchHandler(ctx, router, matchUsecase, serversUC, authUsecase, configUsecase)
	metrics.NewMetricsHandler(router)
	network.NewHandler(router, userAuth, g.networks)
	network.NewBlocklistHandler(router, userAuth, g.blocklists, g.networks)
	news.NewNewsHandler(router, g.news, userAuth)
	notification.NewNotificationHandler(router, userAuth, g.notifications)
	patreon.NewPatreonHandler(router, userAuth, patreon.NewPatreon(patreon.NewRepository(g.database), conf.Patreon), g.config.Config().Patreon)
	person.NewPersonHandler(router, userAuth, g.persons)
	playerqueue.NewPlayerqueueHandler(router, userAuth, g.playerQueue)
	servers.NewDemoHandler(router, userAuth, g.demos)
	servers.NewServersHandler(router, userAuth, g.servers)
	servers.NewSpeedrunsHandler(router, userAuth, serverAuth, g.speedruns)
	sourcemod.NewHandler(router, userAuth, serverAuth, g.sourcemod, g.notifications, conf.Discord.SafeKickLogChannelID(), g.persons)
	votes.NewVotesHandler(router, userAuth, g.votes)
	wiki.NewWikiHandler(router, userAuth, g.wiki)

	router.GET("/health", g.healthCheck)

	httpServer := httphelper.NewServer(conf.Addr(), router)

	go func() {
		<-ctx.Done()

		slog.Info("Shutting down HTTP service")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		if errShutdown := httpServer.Shutdown(shutdownCtx); errShutdown != nil { //nolint:contextcheck
			slog.Error("Error shutting down http service", slog.String("error", errShutdown.Error()))
		}
	}()

	slog.Info("Starting HTTP server", slog.String("address", conf.Addr()), slog.String("url", conf.ExternalURL))

	if errServe := httpServer.ListenAndServe(); errServe != nil && !errors.Is(errServe, http.ErrServerClosed) {
		slog.Error("HTTP server returned error", slog.String("error", errServe.Error()))
	}

	<-ctx.Done()

	slog.Info("Exiting...")

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
	var dur *duration.Duration
	if warning.MatchedFilter.Action == chat.FilterActionBan || warning.MatchedFilter.Action == chat.FilterActionMute {
		parsedDur, errDur := duration.Parse(warning.MatchedFilter.Duration)
		if errDur != nil {
			return errors.Join(errDur, chat.ErrInvalidActionDuration)
		}
		dur = parsedDur
	}

	var (
		errBan error
		newBan ban.Ban
		req    = ban.Opts{
			TargetID:   warning.UserMessage.SteamID,
			Reason:     warning.WarnReason,
			ReasonText: "",
			Note:       "Automatic warning ban",
			Duration:   dur,
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
		// Kicks are temporary, so should be done by Player ID to avoid
		// missing players who weren't in the latest state update
		// (otherwise, kicking players very shortly after they connect
		// will usually fail).
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

func (g *GBans) onAnticheatBan(ctx context.Context, entry logparse.StacEntry, dur time.Duration, count int) error {
	conf := g.config.Config()
	newBan, err := g.bans.Create(ctx, ban.Opts{
		Origin:     ban.System,
		SourceID:   steamid.New(conf.Owner),
		TargetID:   entry.SteamID,
		Duration:   duration.FromTimeDuration(dur),
		BanType:    bantype.Banned,
		Reason:     reason.Cheating,
		ReasonText: "",
		Note:       entry.Summary + "\n\nRaw log:\n" + entry.RawLog,
		DemoName:   entry.DemoName,
		DemoTick:   entry.DemoTick,
		EvadeOk:    false,
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

func (g *GBans) healthCheck(ctx *gin.Context) {
	serverStates := g.servers.Current()
	if len(serverStates) > 0 {
		for _, server := range serverStates {
			if server.MaxPlayers > 0 {
				ctx.String(http.StatusOK, "😎")

				return
			}
		}
		ctx.String(http.StatusServiceUnavailable, "🙅🏻‍♀️")
	} else {
		ctx.String(http.StatusOK, "😎")
	}
}

// downloadManager is responsible for connecting to the remote servers via ssh/scp and executing instructions.
// Multiple handlers can be registered that will be run for every update call.
func downloadManager(ctx context.Context, freq time.Duration, store database.Database, conf scp.Config, handlers ...scp.ConnectionHandler) {
	var (
		timeout     = time.Second * 120
		connections []scp.Connection
		repo        = scp.NewRepository(store)
		ticker      = time.NewTicker(freq)
	)

	defer func() {
		for _, handler := range connections {
			handler.Close()
		}
	}()

	for {
		select {
		case <-ticker.C:
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

			slog.Debug("Updating SCP handlers")
			start := time.Now()
			lCtx, cancel := context.WithTimeout(ctx, timeout)

			// No errgroup since we want to continue on errors.
			waitGroup := &sync.WaitGroup{}

			for _, handler := range connections {
				waitGroup.Go(func() {
					if err := handler.Update(lCtx); err != nil {
						slog.Error("Error running scp handler", slog.String("error", err.Error()))
					}
				})
			}

			waitGroup.Wait()

			slog.Debug("SCP Update complete", slog.Duration("duration", time.Since(start)))
			cancel()
		case <-ctx.Done():
			return
		}
	}
}
