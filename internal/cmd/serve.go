package cmd

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
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
	"github.com/spf13/cobra"
)

func firstTimeSetup(ctx context.Context, persons person.Persons, newsUC news.News,
	wikiUC wiki.Wiki, conf config.Config,
) error {
	_, errRootUser := persons.GetPersonBySteamID(ctx, nil, steamid.New(conf.Owner))
	if errRootUser == nil {
		return nil
	}

	if !errors.Is(errRootUser, database.ErrNoResult) {
		return errRootUser
	}

	newOwner := person.NewPerson(steamid.New(conf.Owner))
	newOwner.PermissionLevel = permission.PAdmin

	if errSave := persons.SavePerson(ctx, nil, &newOwner); errSave != nil {
		slog.Error("Failed create new owner", log.ErrAttr(errSave))
	}

	newsEntry := news.Article{
		Title:       "Welcome to gbans",
		BodyMD:      "This is an *example* **news** entry.",
		IsPublished: true,
		CreatedOn:   time.Now(),
		UpdatedOn:   time.Now(),
	}

	if errSave := newsUC.Save(ctx, &newsEntry); errSave != nil {
		return errSave
	}

	page := wiki.Page{
		Slug:      wiki.RootSlug,
		BodyMD:    "# Welcome to the wiki",
		Revision:  1,
		CreatedOn: time.Now(),
		UpdatedOn: time.Now(),
	}

	_, errSave := wikiUC.Save(ctx, newOwner, page.Slug, page.BodyMD, page.PermissionLevel)
	if errSave != nil {
		slog.Error("Failed save example wiki entry", log.ErrAttr(errSave))
	}

	return nil
}

// func createQueueWorkers(people person.PersonUsecase, notifications notification.NotificationPayload,
// 	discordUC *discord.Discord, authRepo auth.AuthRepository,
// 	patreonUC patreon.PatreonCredential, reports ban.ReportUsecase, discordOAuth discordoauth.DiscordOAuthUsecase,
// ) *river.Workers {
// 	workers := river.NewWorkers()

// 	river.AddWorker[notification.SenderArgs](workers, notification.NewSenderWorker(people, notifications, discordUC))
// 	river.AddWorker[auth.CleanupArgs](workers, auth.NewCleanupWorker(authRepo))
// 	river.AddWorker[patreon.AuthUpdateArgs](workers, patreon.NewSyncWorker(patreonUC))
// 	river.AddWorker[ban.MetaInfoArgs](workers, ban.NewMetaInfoWorker(reports))
// 	river.AddWorker[discord.TokenRefreshArgs](workers, discord.NewTokenRefreshWorker(discordOAuth))

// 	return workers
// }

// func createPeriodicJobs() []*river.PeriodicJob {
// 	jobs := []*river.PeriodicJob{
// 		river.NewPeriodicJob(
// 			river.PeriodicInterval(24*time.Hour),
// 			func() (river.JobArgs, *river.InsertOpts) {
// 				return auth.CleanupArgs{}, nil
// 			},
// 			&river.PeriodicJobOpts{RunOnStart: true}),

// 		river.NewPeriodicJob(
// 			river.PeriodicInterval(time.Hour),
// 			func() (river.JobArgs, *river.InsertOpts) {
// 				return patreon.AuthUpdateArgs{}, nil
// 			},
// 			&river.PeriodicJobOpts{RunOnStart: true}),

// 		river.NewPeriodicJob(
// 			river.PeriodicInterval(24*time.Hour),
// 			func() (river.JobArgs, *river.InsertOpts) {
// 				return report.MetaInfoArgs{}, nil
// 			},
// 			&river.PeriodicJobOpts{RunOnStart: true}),

// 		river.NewPeriodicJob(
// 			river.PeriodicInterval(time.Hour*12),
// 			func() (river.JobArgs, *river.InsertOpts) {
// 				return discord.TokenRefreshArgs{}, nil
// 			},
// 			&river.PeriodicJobOpts{RunOnStart: true}),
// 	}

// 	return jobs
// }

// serveCmd represents the serve command.
func serveCmd() *cobra.Command { //nolint:maintidx
	return &cobra.Command{
		Use:   "serve",
		Short: "Starts the gbans web app",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			slog.Info("Starting gbans...",
				slog.String("version", BuildVersion),
				slog.String("commit", BuildCommit),
				slog.String("date", BuildDate))

			staticConfig, errStatic := config.ReadStaticConfig()
			if errStatic != nil {
				slog.Error("Failed to read static config", log.ErrAttr(errStatic))

				return errStatic
			}

			dbConn := database.New(staticConfig.DatabaseDSN, staticConfig.DatabaseAutoMigrate, staticConfig.DatabaseLogQueries)
			if errConnect := dbConn.Connect(ctx); errConnect != nil {
				slog.Error("Cannot initialize database", log.ErrAttr(errConnect))

				return errConnect
			}

			defer func() {
				if errClose := dbConn.Close(); errClose != nil {
					slog.Error("Failed to close database cleanly", log.ErrAttr(errClose))
				}
			}()

			// Config
			configuration := config.NewConfiguration(staticConfig, config.NewRepository(dbConn))
			if err := configuration.Init(ctx); err != nil {
				slog.Error("Failed to init config", log.ErrAttr(err))

				return err
			}

			if errConfig := configuration.Reload(ctx); errConfig != nil {
				slog.Error("Failed to read config", log.ErrAttr(errConfig))

				return errConfig
			}

			// This is normally set by build time flags, but can be overwritten by the env var.
			if SentryDSN == "" {
				if value, found := os.LookupEnv("SENTRY_DSN"); found && value != "" {
					SentryDSN = value
				}
			}

			conf := configuration.Config()

			if SentryDSN != "" {
				sentryClient, err := log.NewSentryClient(SentryDSN, true, 0.25, BuildVersion, string(conf.General.Mode))
				if err != nil {
					slog.Error("Failed to setup sentry client")
				} else {
					slog.Info("Sentry.io support is enabled.")
					defer sentryClient.Flush(2 * time.Second)
				}
			} else {
				slog.Info("Sentry.io support is disabled. To enable at runtime, set SENTRY_DSN.")
			}

			logCloser := log.MustCreateLogger(ctx, conf.Log.File, conf.Log.Level, SentryDSN != "", BuildVersion)
			defer logCloser()

			eventBroadcaster := fp.NewBroadcaster[logparse.EventType, logparse.ServerEvent]()
			// weaponsMap := fp.NewMutexMap[logparse.Weapon, int]()

			bot, errDiscord := discord.NewDiscord(conf.Discord.AppID, conf.Discord.GuildID, conf.Discord.Token, conf.ExternalURL)
			if errDiscord != nil {
				return errDiscord
			}

			if conf.Discord.Enabled {
				if err := bot.Start(ctx); err != nil {
					slog.Error("Failed to start discord", log.ErrAttr(err))

					return err
				}

				defer bot.Shutdown()
			}

			notifications := notification.NewNotifications(notification.NewRepository(dbConn), bot)

			wordFilters := chat.NewWordFilter(chat.NewWordFilterRepository(dbConn))
			if err := wordFilters.Import(ctx); err != nil {
				slog.Error("Failed to load word filters", log.ErrAttr(err))

				return err
			}

			tfapiClient, errClient := thirdparty.NewTFAPI("https://tf-api.roto.lol", &http.Client{Timeout: time.Second * 15})
			if errClient != nil {
				return errClient
			}

			persons := person.NewPersons(person.NewRepository(conf, dbConn), configuration, tfapiClient)

			networks := network.NewNetworks(eventBroadcaster, network.NewRepository(dbConn, persons), configuration)
			go networks.Start(ctx)

			assetRepo := asset.NewLocalRepository(dbConn, conf.LocalStore.PathRoot)
			if err := assetRepo.Init(ctx); err != nil {
				slog.Error("Failed to init local asset repo", log.ErrAttr(err))

				return err
			}

			assets := asset.NewAssets(assetRepo)
			serversUC := servers.NewServers(servers.NewServersRepository(dbConn))
			demos := servers.NewDemos(asset.BucketDemo, servers.NewDemoRepository(dbConn), assets, configuration)
			reports := ban.NewReports(ban.NewReportRepository(dbConn), configuration, persons, demos, tfapiClient)
			states := servers.NewState(eventBroadcaster, servers.NewStateRepository(servers.NewCollector(serversUC)), configuration, serversUC)
			bans := ban.NewBans(ban.NewBanRepository(dbConn, persons, networks), persons, configuration, reports, states, tfapiClient)
			blocklists := network.NewBlocklists(network.NewBlocklistRepository(dbConn), &bans) // TODO Does THE & work here?

			go func() {
				if err := states.Start(ctx); err != nil {
					slog.Error("Failed to start state tracker", log.ErrAttr(err))
				}
			}()

			discordOAuth := discordoauth.NewDiscordOAuth(discordoauth.NewRepository(dbConn), configuration)

			appeals := ban.NewAppeals(ban.NewAppealRepository(dbConn), bans, persons, configuration)

			chatRepo := chat.NewChatRepository(dbConn, persons, wordFilters, eventBroadcaster)
			go chatRepo.Start(ctx)

			chats := chat.NewChat(configuration, chatRepo, wordFilters, states, bans, persons)
			go chats.Start(ctx)

			forums := forum.NewForums(forum.NewForumRepository(dbConn))
			go forums.Start(ctx)

			metric := metrics.NewMetrics(eventBroadcaster)
			go metric.Start(ctx)

			newsHandler := news.NewNews(news.NewNewsRepository(dbConn))
			patreons := patreon.NewPatreon(patreon.NewRepository(dbConn), configuration)
			gameServers := servers.NewSRCDS(servers.NewSRCDSRepository(dbConn), configuration, serversUC, persons, tfapiClient)
			wikis := wiki.NewWiki(wiki.NewRepository(dbConn))
			authenticator := auth.NewAuthentication(auth.NewRepository(dbConn), configuration, persons, bans, serversUC, SentryDSN)
			anticheats := anticheat.NewAntiCheat(anticheat.NewRepository(dbConn), bans, configuration, persons)

			voteRecorder := votes.NewVotes(votes.NewRepository(dbConn), eventBroadcaster)
			go voteRecorder.Start(ctx)

			contests := contest.NewContests(contest.NewRepository(dbConn))
			speedruns := servers.NewSpeedruns(servers.NewSpeedrunRepository(dbConn, persons))

			if err := firstTimeSetup(ctx, persons, newsHandler, wikis, conf); err != nil {
				slog.Error("Failed to run first time setup", log.ErrAttr(err))

				return err
			}

			if conf.General.Mode == config.ReleaseMode {
				gin.SetMode(gin.ReleaseMode)
			} else {
				gin.SetMode(gin.DebugMode)
			}

			conf.Network.SDREnabled = true
			conf.Network.SDRDNSEnabled = true

			// If we are using Valve SDR network, optionally enable the dynamic DNS update support to automatically
			// update the A record when a change is detected with the new public SDR IP.
			if conf.Network.SDREnabled && conf.Network.SDRDNSEnabled {
				// go dns.MonitorChanges(ctx, conf, stateUsecase, serversUC)
			}

			router, err := CreateRouter(conf, Version())
			if err != nil {
				slog.Error("Could not setup router", log.ErrAttr(err))

				return err
			}

			// Start discord bot service
			if conf.Discord.Enabled {
				discordHandler, errDiscord := discord.NewDiscord(conf.Discord.AppID, conf.Discord.GuildID, conf.Discord.Token, conf.ExternalURL)
				if errDiscord != nil {
					return errDiscord
				}
				discordHandler.Start(ctx)
			}

			// Register all our handlers with router
			anticheat.NewAnticheatHandler(router, authenticator, anticheats)
			ban.NewAppealHandler(router, appeals, authenticator)
			auth.NewAuthHandler(router, authenticator, configuration, persons, tfapiClient)
			ban.NewHandlerSteam(router, bans, configuration, authenticator)
			config.NewConfigHandler(router, configuration, authenticator, BuildVersion)
			discordoauth.NewDiscordOAuthHandler(router, authenticator, configuration, persons, discordOAuth)
			network.NewBlocklistHandler(router, blocklists, networks, authenticator)
			chat.NewChatHandler(router, chats, authenticator)
			contest.NewContestHandler(router, contests, assets, authenticator)
			servers.NewDemoHandler(router, demos, authenticator)
			forum.NewForumHandler(router, forums, authenticator)
			// match.NewMatchHandler(ctx, router, matchUsecase, serversUC, authUsecase, configUsecase)
			asset.NewAssetHandler(router, assets, authenticator)
			metrics.NewMetricsHandler(router)
			network.NewNetworkHandler(router, networks, authenticator)
			news.NewNewsHandler(router, newsHandler, notifications, authenticator)
			notification.NewNotificationHandler(router, notifications, authenticator)
			patreon.NewPatreonHandler(router, patreons, authenticator, configuration)
			person.NewPersonHandler(router, configuration, persons, authenticator)
			ban.NewReportHandler(router, reports, authenticator)
			servers.NewServersHandler(router, serversUC, states, authenticator)
			servers.NewSpeedrunsHandler(router, speedruns, authenticator, configuration, serversUC, SentryDSN)
			servers.NewSRCDSHandler(router, gameServers, serversUC, persons, assets, bans,
				networks, authenticator,
				configuration, states, blocklists, SentryDSN)
			votes.NewVotesHandler(router, voteRecorder, authenticator)
			wiki.NewWikiHandler(router, wikis, authenticator)
			chat.NewWordFilterHandler(router, configuration, wordFilters, chats, authenticator)

			playerqueueRepo := playerqueue.NewPlayerqueueRepository(dbConn, persons)
			// Pre-load some messages into queue message cache
			chatlogs, errChatlogs := playerqueueRepo.Query(ctx, playerqueue.PlayerqueueQueryOpts{Filter: query.Filter{Limit: 100}})
			if errChatlogs != nil {
				slog.Error("Failed to warm playerqueue chatlogs", log.ErrAttr(err))
				chatlogs = []playerqueue.ChatLog{}
			}
			playerqueueUC := playerqueue.NewPlayerqueue(playerqueueRepo, persons, serversUC, states, chatlogs)
			go playerqueueUC.Start(ctx)
			playerqueue.NewPlayerqueueHandler(router, authenticator, configuration, playerqueueUC)

			if conf.Debug.AddRCONLogAddress != "" {
				go states.LogAddressAdd(ctx, conf.Debug.AddRCONLogAddress)
				defer states.LogAddressDel(ctx, conf.Debug.AddRCONLogAddress)
			}

			memberships := ban.NewMemberships(ban.NewBanRepository(dbConn, persons, networks), tfapiClient)
			banExpirations := ban.NewExpirationMonitor(bans, persons, notifications, configuration)

			go func() {
				if errSync := anticheats.SyncDemoIDs(ctx, 100); errSync != nil {
					slog.Error("failed to sync anticheat demos")
				}

				go memberships.Update(ctx)
				go banExpirations.Update(ctx)
				go blocklists.Sync(ctx)
				go demos.Cleanup(ctx)

				membershipsTicker := time.NewTicker(12 * time.Hour)
				expirationsTicker := time.NewTicker(60 * time.Second)
				reportIntoTicker := time.NewTicker(24 * time.Hour)
				blocklistTicker := time.NewTicker(6 * time.Hour)
				demoTicker := time.NewTicker(15 * time.Minute)

				select {
				case <-ctx.Done():
					return
				case <-membershipsTicker.C:
					go memberships.Update(ctx)
				case <-expirationsTicker.C:
					go banExpirations.Update(ctx)
				case <-reportIntoTicker.C:
					go func() {
						if errMeta := reports.MetaStats(ctx); errMeta != nil {
							slog.Error("Failed to generate meta stats", log.ErrAttr(errMeta))
						}
					}()
				case <-blocklistTicker.C:
					go blocklists.Sync(ctx)
				case <-demoTicker.C:
					go demos.Cleanup(ctx)
					if errSync := anticheats.SyncDemoIDs(ctx, 100); errSync != nil {
						slog.Error("failed to sync anticheat demos")
					}
				}
			}()

			httpServer := httphelper.NewServer(conf.Addr(), router)

			demoDownloader := scp.NewDownloader(configuration, dbConn)
			// TODO register handlers
			go demoDownloader.Start(ctx)

			go func() {
				<-ctx.Done()

				slog.Info("Shutting down HTTP service")

				shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
				defer cancel()

				if errShutdown := httpServer.Shutdown(shutdownCtx); errShutdown != nil { //nolint:contextcheck
					slog.Error("Error shutting down http service", log.ErrAttr(errShutdown))
				}
			}()

			errServe := httpServer.ListenAndServe()
			if errServe != nil && !errors.Is(errServe, http.ErrServerClosed) {
				slog.Error("HTTP server returned error", log.ErrAttr(errServe))
			}

			<-ctx.Done()

			slog.Info("Exiting...")

			return nil
		},
	}
}
