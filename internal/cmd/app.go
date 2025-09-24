package cmd

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/leighmacdonald/gbans/internal/anticheat"
	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/auth"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/contest"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/discord"
	discordoauth "github.com/leighmacdonald/gbans/internal/discord_oauth"
	"github.com/leighmacdonald/gbans/internal/forum"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/metrics"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/network/scp"
	"github.com/leighmacdonald/gbans/internal/news"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/patreon"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/playerqueue"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/internal/votes"
	"github.com/leighmacdonald/gbans/internal/wiki"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
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

func Version() BuildInfo {
	return BuildInfo{
		BuildVersion: BuildVersion,
		Commit:       BuildCommit,
		Date:         BuildDate,
	}
}

type GBans struct {
	anticheat      anticheat.AntiCheat
	appeals        ban.Appeals
	assets         asset.Assets
	auth           *auth.Authentication
	banExpirations *ban.ExpirationMonitor
	bans           ban.Bans
	blocklists     network.Blocklists
	chat           *chat.Chat
	chatRepo       *chat.Repository
	config         *config.Configuration
	contests       contest.Contests
	database       database.Database
	demos          servers.Demos
	forums         forum.Forums
	discordOAuth   discordoauth.DiscordOAuth
	memberships    *ban.Memberships
	metrics        metrics.Metrics
	networks       network.Networks
	news           news.News
	notifications  notification.Notifications
	patreon        patreon.Patreon
	persons        *person.Persons
	playerQueue    *playerqueue.Playerqueue
	reports        ban.Reports
	servers        servers.Servers
	speedruns      servers.Speedruns
	srcds          *servers.SRCDS
	states         *servers.State
	staticConfig   config.Static
	tfapiClient    *thirdparty.TFAPI
	votes          votes.Votes
	wiki           wiki.Wiki
	wordFilters    chat.WordFilters
	sentry         *sentry.Client
	bot            *discord.Discord

	broadcaster *fp.Broadcaster[logparse.EventType, logparse.ServerEvent]

	logCloser func()
}

func NewGBans() (*GBans, error) {
	staticConfig, errStatic := config.ReadStaticConfig()
	if errStatic != nil {
		slog.Error("Failed to read static config", log.ErrAttr(errStatic))

		return nil, errStatic
	}

	return &GBans{
		staticConfig: staticConfig,
		broadcaster:  fp.NewBroadcaster[logparse.EventType, logparse.ServerEvent](),
	}, nil
}

func (g *GBans) Init(ctx context.Context) error {
	dbConn := database.New(g.staticConfig.DatabaseDSN, g.staticConfig.DatabaseAutoMigrate, g.staticConfig.DatabaseLogQueries)
	if errConnect := dbConn.Connect(ctx); errConnect != nil {
		slog.Error("Cannot initialize database", log.ErrAttr(errConnect))

		return errConnect
	}
	g.database = dbConn

	g.config = config.NewConfiguration(g.staticConfig, config.NewRepository(g.database))
	if err := g.config.Init(ctx); err != nil {
		slog.Error("Failed to init config", log.ErrAttr(err))

		return err
	}

	if errConfig := g.config.Reload(ctx); errConfig != nil {
		slog.Error("Failed to read config", log.ErrAttr(errConfig))

		return errConfig
	}

	// This is normally set by build time flags, but can be overwritten by the env var.
	if SentryDSN == "" {
		if value, found := os.LookupEnv("SENTRY_DSN"); found && value != "" {
			SentryDSN = value
		}
	}

	conf := g.config.Config()

	g.setupSentry()

	g.logCloser = log.MustCreateLogger(ctx, conf.Log.File, conf.Log.Level, SentryDSN != "", BuildVersion)

	slog.Info("Starting gbans...",
		slog.String("version", BuildVersion),
		slog.String("commit", BuildCommit),
		slog.String("date", BuildDate))

	// weaponsMap := fp.NewMutexMap[logparse.Weapon, int]()

	g.notifications = notification.NewNotifications(notification.NewRepository(g.database), g.bot)

	wordFilters := chat.NewWordFilters(chat.NewWordFilterRepository(g.database), g.notifications, g.config)
	if err := wordFilters.Import(ctx); err != nil {
		slog.Error("Failed to load word filters", log.ErrAttr(err))

		return err
	}
	g.wordFilters = wordFilters

	tfapiClient, errClient := thirdparty.NewTFAPI("https://tf-api.roto.lol", &http.Client{Timeout: time.Second * 15})
	if errClient != nil {
		return errClient
	}
	g.tfapiClient = tfapiClient

	g.persons = person.NewPersons(person.NewRepository(conf, g.database), g.config, g.tfapiClient)
	g.networks = network.NewNetworks(g.broadcaster, network.NewRepository(g.database, g.persons), g.config)

	assetRepo := asset.NewLocalRepository(g.database, conf.LocalStore.PathRoot)
	if err := assetRepo.Init(ctx); err != nil {
		slog.Error("Failed to init local asset repo", log.ErrAttr(err))

		return err
	}

	bot, errDiscord := discord.NewDiscord(conf.Discord.AppID, conf.Discord.GuildID, conf.Discord.Token, conf.ExternalURL)
	if errDiscord != nil {
		return errDiscord
	}
	g.bot = bot

	g.assets = asset.NewAssets(assetRepo)
	g.servers = servers.NewServers(servers.NewRepository(g.database))
	g.demos = servers.NewDemos(asset.BucketDemo, servers.NewDemoRepository(g.database), g.assets, g.config)
	g.reports = ban.NewReports(ban.NewReportRepository(g.database), g.config, g.persons, g.demos, g.tfapiClient, g.notifications)
	g.states = servers.NewState(g.broadcaster, servers.NewStateRepository(servers.NewCollector(g.servers)), g.config, g.servers)
	g.bans = ban.NewBans(ban.NewRepository(g.database, g.persons, g.networks), g.persons, g.config, g.reports, g.states, g.tfapiClient, g.notifications)
	g.blocklists = network.NewBlocklists(network.NewBlocklistRepository(g.database), g.bans) // TODO Does THE & work here?
	g.discordOAuth = discordoauth.NewOAuth(discordoauth.NewRepository(g.database), g.config)
	g.appeals = ban.NewAppeals(ban.NewAppealRepository(g.database), g.bans, g.persons, g.config, g.notifications)
	g.chatRepo = chat.NewRepository(g.database, g.persons, g.wordFilters, g.broadcaster)
	g.chat = chat.NewChat(g.config, g.chatRepo, g.wordFilters, g.states, g.bans, g.persons)
	g.forums = forum.NewForums(forum.NewRepository(g.database), g.config, g.notifications)
	g.metrics = metrics.NewMetrics(g.broadcaster)
	g.news = news.NewNews(news.NewRepository(g.database))
	g.patreon = patreon.NewPatreon(patreon.NewRepository(g.database), g.config)
	g.srcds = servers.NewSRCDS(servers.NewSRCDSRepository(g.database), g.config, g.servers, g.persons, g.tfapiClient, g.notifications)
	g.wiki = wiki.NewWiki(wiki.NewRepository(g.database))
	g.auth = auth.NewAuthentication(auth.NewRepository(g.database), g.config, g.persons, g.bans, g.servers, SentryDSN)
	g.anticheat = anticheat.NewAntiCheat(anticheat.NewRepository(g.database), g.bans, g.config, g.persons, g.notifications)
	g.votes = votes.NewVotes(votes.NewRepository(g.database), g.broadcaster, g.notifications, g.config, g.persons)
	g.contests = contest.NewContests(contest.NewRepository(g.database))
	g.speedruns = servers.NewSpeedruns(servers.NewSpeedrunRepository(g.database, g.persons))
	g.memberships = ban.NewMemberships(ban.NewRepository(g.database, g.persons, g.networks), g.tfapiClient)
	g.banExpirations = ban.NewExpirationMonitor(g.bans, g.persons, g.notifications, g.config)

	if err := g.firstTimeSetup(ctx); err != nil {
		slog.Error("Failed to run first time setup", log.ErrAttr(err))

		return err
	}

	// If we are using Valve SDR network, optionally enable the dynamic DNS update support to automatically
	// update the A record when a change is detected with the new public SDR IP.
	// if conf.Network.SDREnabled && conf.Network.SDRDNSEnabled {
	// 	// go dns.MonitorChanges(ctx, conf, stateUsecase, serversUC)
	// }

	// Config
	g.setupPlayerQueue(ctx)

	return nil
}

func (g *GBans) startBot(ctx context.Context) error {
	if !g.config.Config().Discord.Enabled {
		return nil
	}

	anticheat.RegisterDiscordCommands(g.bot, g.anticheat, g.config)
	ban.RegisterDiscordCommands(g.bot, g.bans)
	chat.RegisterDiscordCommands(g.bot, g.wordFilters)

	servers.RegisterDiscordCommands(g.bot, *g.states, g.persons, g.servers, g.networks, g.config.Config())

	if err := g.bot.Start(ctx); err != nil {
		return err
	}

	return nil
}

func (g *GBans) setupPlayerQueue(ctx context.Context) {
	playerqueueRepo := playerqueue.NewRepository(g.database, g.persons)
	// Pre-load some messages into queue message cache
	chatlogs, errChatlogs := playerqueueRepo.Query(ctx, playerqueue.QueryOpts{Filter: query.Filter{Limit: 100}})
	if errChatlogs != nil {
		slog.Error("Failed to warm playerqueue chatlogs", log.ErrAttr(errChatlogs))
		chatlogs = []playerqueue.ChatLog{}
	}
	g.playerQueue = playerqueue.NewPlayerqueue(playerqueueRepo, g.persons, g.servers, g.states, chatlogs, g.config, g.notifications)
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
		go g.states.LogAddressAdd(ctx, conf.Debug.AddRCONLogAddress)
	}

	go g.chatRepo.Start(ctx)
	go g.chat.Start(ctx)
	go g.forums.Start(ctx)
	go g.metrics.Start(ctx)
	go g.votes.Start(ctx)
	go g.playerQueue.Start(ctx)
	go g.networks.Start(ctx)
	go g.notifications.Sender(ctx)
	go g.downloadManager(ctx, time.Minute*5)

	go func() {
		if err := g.states.Start(ctx); err != nil {
			slog.Error("Failed to start state tracker", log.ErrAttr(err))
		}
	}()

	go func() {
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
					slog.Error("Failed to generate meta stats", log.ErrAttr(errMeta))
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
	}()
}

func (g *GBans) Serve(rootCtx context.Context) error {
	ctx, stop := signal.NotifyContext(rootCtx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if errStart := g.startBot(ctx); errStart != nil {
			slog.Error("Failed to start bot", slog.String("error", errStart.Error()))
		}
	}()

	conf := g.config.Config()

	router, err := CreateRouter(conf, Version())
	if err != nil {
		slog.Error("Could not setup router", log.ErrAttr(err))

		return err
	}

	// Register all our handlers with router
	anticheat.NewAnticheatHandler(router, g.auth, g.anticheat)
	asset.NewAssetHandler(router, g.assets, g.auth)
	auth.NewAuthHandler(router, g.auth, g.config, g.persons, g.tfapiClient)
	ban.NewAppealHandler(router, g.appeals, g.auth)
	ban.NewReportHandler(router, g.reports, g.auth)
	ban.NewHandlerSteam(router, g.bans, g.config, g.auth)
	chat.NewChatHandler(router, g.chat, g.auth)
	chat.NewWordFilterHandler(router, g.config, g.wordFilters, g.chat, g.auth)
	config.NewHandler(router, g.config, g.auth, BuildVersion)
	contest.NewContestHandler(router, g.contests, g.assets, g.auth)
	discordoauth.NewDiscordOAuthHandler(router, g.auth, g.config, g.persons, g.discordOAuth)
	forum.NewForumHandler(router, g.forums, g.auth)
	// match.NewMatchHandler(ctx, router, matchUsecase, serversUC, authUsecase, configUsecase)
	metrics.NewMetricsHandler(router)
	network.NewNetworkHandler(router, g.networks, g.auth)
	network.NewBlocklistHandler(router, g.blocklists, g.networks, g.auth)
	news.NewNewsHandler(router, g.news, g.notifications, g.auth)
	notification.NewNotificationHandler(router, g.notifications, g.auth)
	patreon.NewPatreonHandler(router, g.patreon, g.auth, g.config)
	person.NewPersonHandler(router, g.config, g.persons, g.auth)
	playerqueue.NewPlayerqueueHandler(router, g.auth, g.config, g.playerQueue)
	servers.NewDemoHandler(router, g.demos, g.auth)
	servers.NewServersHandler(router, g.servers, g.states, g.auth)
	servers.NewSpeedrunsHandler(router, g.speedruns, g.auth, g.config, g.servers, SentryDSN)
	servers.NewSRCDSHandler(router, g.srcds, g.servers, g.persons, g.assets, g.bans,
		g.networks, g.auth, g.config, g.states, g.blocklists, SentryDSN)
	votes.NewVotesHandler(router, g.votes, g.auth)
	wiki.NewWikiHandler(router, g.wiki, g.auth)

	httpServer := httphelper.NewServer(conf.Addr(), router)

	go func() {
		<-ctx.Done()

		slog.Info("Shutting down HTTP service")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		if errShutdown := httpServer.Shutdown(shutdownCtx); errShutdown != nil { //nolint:contextcheck
			slog.Error("Error shutting down http service", log.ErrAttr(errShutdown))
		}
	}()

	slog.Info("Starting HTTP server", slog.String("address", conf.Addr()), slog.String("url", conf.ExternalURL))

	errServe := httpServer.ListenAndServe()
	if errServe != nil && !errors.Is(errServe, http.ErrServerClosed) {
		slog.Error("HTTP server returned error", log.ErrAttr(errServe))
	}

	<-ctx.Done()

	slog.Info("Exiting...")

	return nil
}

func (g *GBans) Close(ctx context.Context) error {
	conf := g.config.Config()
	if conf.Debug.AddRCONLogAddress != "" {
		g.states.LogAddressDel(ctx, conf.Debug.AddRCONLogAddress)
	}

	if g.bot != nil {
		g.bot.Shutdown()
	}

	if g.database != nil {
		if errClose := g.database.Close(); errClose != nil {
			slog.Error("Failed to close database cleanly", log.ErrAttr(errClose))
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

func (g GBans) firstTimeSetup(ctx context.Context) error {
	conf := g.config.Config()
	_, errRootUser := g.persons.GetPersonBySteamID(ctx, nil, steamid.New(conf.Owner))
	if errRootUser == nil {
		return nil
	}

	if !errors.Is(errRootUser, database.ErrNoResult) {
		return errRootUser
	}

	owner := person.New(steamid.New(conf.Owner))
	owner.PermissionLevel = permission.Admin

	if errSave := g.persons.SavePerson(ctx, nil, &owner); errSave != nil {
		slog.Error("Failed create new owner", log.ErrAttr(errSave))
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
		Slug:      wiki.RootSlug,
		BodyMD:    "# Welcome to the wiki",
		Revision:  1,
		CreatedOn: time.Now(),
		UpdatedOn: time.Now(),
	}

	_, errSave := g.wiki.Save(ctx, owner, page.Slug, page.BodyMD, page.PermissionLevel)
	if errSave != nil {
		slog.Error("Failed save example wiki entry", log.ErrAttr(errSave))
	}

	return nil
}

// downloadManager is responsible for connecting to the remote servers via ssh/scp and executing instructions.
// Multiple handlers can be registered that will be ran for every update call.
func (g GBans) downloadManager(ctx context.Context, freq time.Duration) {
	var (
		timeout  time.Duration
		handlers []scp.Connection
		repo     = scp.NewRepository(g.database)
		ticker   = time.NewTicker(freq)
	)

	defer func() {
		for _, handler := range handlers {
			handler.Close()
		}
	}()

	for {
		select {
		case <-ticker.C:
			servers, errServers := repo.Servers(ctx)
			if errServers != nil {
				if errors.Is(errServers, database.ErrNoResult) {
					continue
				}

				slog.Error("Failed to query download servers", slog.String("error", errServers.Error()))

				continue
			}

			for _, server := range servers {
				actualAddr := scp.HostPart(server.Address)
				exists := false
				for _, handler := range handlers {
					if handler.Address() == actualAddr {
						exists = true

						break
					}
				}

				if !exists {
					handler := scp.NewSCPHandler(repo, g.config.Config().SSH)
					handler.AddHandler(g.demos)
					handler.AddHandler(g.anticheat)
					handlers = append(handlers, handler)
				}
			}

			slog.Debug("Updating SCP handlers")
			start := time.Now()
			lCtx, cancel := context.WithTimeout(ctx, timeout)

			// No errgroup since we want to continue on errors.
			waitGroup := &sync.WaitGroup{}

			for _, handler := range handlers {
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
