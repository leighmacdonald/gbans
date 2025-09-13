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
	"github.com/leighmacdonald/gbans/internal/app"
	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/auth"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/contest"
	"github.com/leighmacdonald/gbans/internal/database"
	discordoauth "github.com/leighmacdonald/gbans/internal/discord_oauth"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/forum"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/match"
	"github.com/leighmacdonald/gbans/internal/metrics"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/network/dns"
	"github.com/leighmacdonald/gbans/internal/news"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/patreon"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/playerqueue"
	"github.com/leighmacdonald/gbans/internal/queue"
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

func firstTimeSetup(ctx context.Context, persons person.PersonUsecase, newsUC news.NewsUsecase,
	wikiUC wiki.WikiUsecase, conf config.Config,
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

	newsEntry := news.NewsEntry{
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

	_, errSave := wikiUC.SaveWikiPage(ctx, newOwner, page.Slug, page.BodyMD, page.PermissionLevel)
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
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			slog.Info("Starting gbans...",
				slog.String("version", app.BuildVersion),
				slog.String("commit", app.BuildCommit),
				slog.String("date", app.BuildDate))

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

			if err := queue.Init(ctx, dbConn.Pool()); err != nil {
				slog.Error("Failed to initialize queue", log.ErrAttr(err))

				return err
			}

			// Config
			configUsecase := config.NewConfigUsecase(staticConfig, config.NewConfigRepository(dbConn))
			if err := configUsecase.Init(ctx); err != nil {
				slog.Error("Failed to init config", log.ErrAttr(err))

				return err
			}

			if errConfig := configUsecase.Reload(ctx); errConfig != nil {
				slog.Error("Failed to read config", log.ErrAttr(errConfig))

				return errConfig
			}

			conf := configUsecase.Config()

			// This is normally set by build time flags, but can be overwritten by the env var.
			if app.SentryDSN == "" {
				if value, found := os.LookupEnv("SENTRY_DSN"); found && value != "" {
					app.SentryDSN = value
				}
			}

			if app.SentryDSN != "" {
				sentryClient, err := log.NewSentryClient(app.SentryDSN, true, 0.25, app.BuildVersion, string(conf.General.Mode))
				if err != nil {
					slog.Error("Failed to setup sentry client")
				} else {
					slog.Info("Sentry.io support is enabled.")
					defer sentryClient.Flush(2 * time.Second)
				}
			} else {
				slog.Info("Sentry.io support is disabled. To enable at runtime, set SENTRY_DSN.")
			}

			logCloser := log.MustCreateLogger(ctx, conf.Log.File, conf.Log.Level, app.SentryDSN != "")
			defer logCloser()

			eventBroadcaster := fp.NewBroadcaster[logparse.EventType, logparse.ServerEvent]()
			weaponsMap := fp.NewMutexMap[logparse.Weapon, int]()

			discordUsecase, errDiscord := discord.NewDiscord(conf.Discord.AppID, conf.Discord.GuildID, conf.Discord.Token, conf.ExternalURL)
			if errDiscord != nil {
				return errDiscord
			}

			if conf.Discord.Enabled {
				if err := discordUsecase.Start(); err != nil {
					slog.Error("Failed to start discord", log.ErrAttr(err))

					return err
				}

				defer discordUsecase.Shutdown()
			}

			notificationUsecase := notification.NewNotificationUsecase(notification.NewNotificationRepository(dbConn), discordUsecase)

			wordFilterUsecase := chat.NewWordFilterUsecase(chat.NewWordFilterRepository(dbConn))
			if err := wordFilterUsecase.Import(ctx); err != nil {
				slog.Error("Failed to load word filters", log.ErrAttr(err))

				return err
			}

			tfapiClient, errClient := thirdparty.NewTFAPI("https://tf-api.roto.lol", &http.Client{Timeout: time.Second * 15})
			if errClient != nil {
				return errClient
			}

			personUsecase := person.NewPersonUsecase(person.NewPersonRepository(conf, dbConn), configUsecase, tfapiClient)

			networkUsecase := network.NewNetworkUsecase(eventBroadcaster, network.NewNetworkRepository(dbConn), configUsecase)
			go networkUsecase.Start(ctx)

			assetRepository := asset.NewLocalRepository(dbConn, conf.LocalStore.PathRoot)
			if err := assetRepository.Init(ctx); err != nil {
				slog.Error("Failed to init local asset repo", log.ErrAttr(err))

				return err
			}

			assets := asset.NewAssetUsecase(assetRepository)
			serversUC := servers.NewServersUsecase(servers.NewServersRepository(dbConn))
			demos := servers.NewDemoUsecase(asset.BucketDemo, servers.NewDemoRepository(dbConn), assets, configUsecase)

			reportUsecase := ban.NewReportUsecase(ban.NewReportRepository(dbConn), configUsecase, personUsecase, demos, tfapiClient)

			stateUsecase := servers.NewStateUsecase(eventBroadcaster,
				servers.NewStateRepository(servers.NewCollector(serversUC)), configUsecase, serversUC)

			banRepo := ban.NewBanRepository(dbConn, personUsecase, networkUsecase)
			banUsecase := ban.NewBanUsecase(banRepo, personUsecase, configUsecase, reportUsecase, stateUsecase, tfapiClient)
			blocklistUsecase := network.NewBlocklistUsecase(network.NewBlocklistRepository(dbConn), &banUsecase) // TODO Does THE & work here?

			go func() {
				if err := stateUsecase.Start(ctx); err != nil {
					slog.Error("Failed to start state tracker", log.ErrAttr(err))
				}
			}()

			discordOAuthUsecase := discordoauth.NewDiscordOAuthUsecase(discordoauth.NewDiscordOAuthRepository(dbConn), configUsecase)

			appeals := ban.NewAppealUsecase(ban.NewAppealRepository(dbConn), banUsecase, personUsecase, configUsecase)

			matchRepo := match.NewMatchRepository(eventBroadcaster, dbConn, personUsecase, serversUC, stateUsecase, weaponsMap)
			go matchRepo.Start(ctx)

			matchUsecase := match.NewMatchUsecase(matchRepo, stateUsecase, serversUC, notificationUsecase)

			if errWeapons := matchUsecase.LoadWeapons(ctx, weaponsMap); errWeapons != nil {
				slog.Error("Failed to import weapons", log.ErrAttr(errWeapons))
			}

			chatRepository := chat.NewChatRepository(dbConn, personUsecase, wordFilterUsecase, matchUsecase, eventBroadcaster)
			go chatRepository.Start(ctx)

			chatUsecase := chat.NewChatUsecase(configUsecase, chatRepository, wordFilterUsecase, stateUsecase, banUsecase, personUsecase)
			go chatUsecase.Start(ctx)

			forumUsecase := forum.NewForumUsecase(forum.NewForumRepository(dbConn))
			go forumUsecase.Start(ctx)

			metricsUsecase := metrics.NewMetricsUsecase(eventBroadcaster)
			go metricsUsecase.Start(ctx)

			newsUsecase := news.NewNewsUsecase(news.NewNewsRepository(dbConn))
			patreonUsecase := patreon.NewPatreonUsecase(patreon.NewPatreonRepository(dbConn), configUsecase)
			srcdsUsecase := servers.NewSrcdsUsecase(servers.NewRepository(dbConn), configUsecase, serversUC, personUsecase, tfapiClient)
			wikiUsecase := wiki.NewWikiUsecase(wiki.NewWikiRepository(dbConn))
			authRepo := auth.NewAuthRepository(dbConn)
			authUsecase := auth.NewAuthUsecase(authRepo, configUsecase, personUsecase, banUsecase, serversUC)
			anticheatUsecase := anticheat.NewAntiCheatUsecase(anticheat.NewAntiCheatRepository(dbConn), personUsecase, banUsecase, configUsecase)

			voteUsecase := votes.NewVoteUsecase(votes.NewVoteRepository(dbConn), personUsecase, matchUsecase, configUsecase, eventBroadcaster)
			go voteUsecase.Start(ctx)

			contestUsecase := contest.NewContestUsecase(contest.NewContestRepository(dbConn))

			speedruns := servers.NewSpeedrunUsecase(servers.NewSpeedrunRepository(dbConn, personUsecase))

			if err := firstTimeSetup(ctx, personUsecase, newsUsecase, wikiUsecase, conf); err != nil {
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
				go dns.MonitorChanges(ctx, conf, stateUsecase, serversUC)
			}

			router, err := CreateRouter(conf, app.Version())
			if err != nil {
				slog.Error("Could not setup router", log.ErrAttr(err))

				return err
			}

			// Start discord bot service
			if conf.Discord.Enabled {
				discordHandler := discord.NewDiscord(discordUsecase, personUsecase, banUsecase,
					stateUsecase, serversUC, configUsecase, networkUsecase, wordFilterUsecase, matchUsecase, anticheatUsecase, tfapiClient)
				discordHandler.Start(ctx)
			}

			// Register all our handlers with router
			anticheat.NewHandler(router, authUsecase, anticheatUsecase)
			ban.NewAppealHandler(router, appeals, authUsecase)
			auth.NewHandler(router, authUsecase, configUsecase, personUsecase, tfapiClient)
			ban.NewHandlerSteam(router, banUsecase, configUsecase, authUsecase)
			config.NewHandler(router, configUsecase, authUsecase, app.Version())
			discord.NewHandler(router, authUsecase, configUsecase, personUsecase, discordOAuthUsecase)
			blocklist.NewHandler(router, blocklistUsecase, networkUsecase, authUsecase)
			chat.NewHandler(router, chatUsecase, authUsecase)
			contest.NewHandler(router, contestUsecase, assets, authUsecase)
			servers.NewDemoHandler(router, demos, authUsecase)
			forum.NewHandler(router, forumUsecase, authUsecase)
			match.NewHandler(ctx, router, matchUsecase, serversUC, authUsecase, configUsecase)
			asset.NewHandler(router, configUsecase, assets, authUsecase)
			metrics.NewHandler(router)
			network.NewHandler(router, networkUsecase, authUsecase)
			news.NewHandler(router, newsUsecase, notificationUsecase, authUsecase)
			notification.NewHandler(router, notificationUsecase, authUsecase)
			patreon.NewHandler(router, patreonUsecase, authUsecase, configUsecase)
			person.NewHandler(router, configUsecase, personUsecase, authUsecase)
			ban.NewReportHandler(router, reportUsecase, authUsecase, notificationUsecase)
			servers.NewHandler(router, serversUC, stateUsecase, authUsecase)
			servers.NewHandler(router, speedruns, authUsecase, configUsecase)
			servers.NewHandlerSRCDS(router, srcdsUsecase, serversUC, personUsecase, assets,
				reportUsecase, banUsecase, networkUsecase, authUsecase,
				configUsecase, notificationUsecase, stateUsecase, blocklistUsecase)
			votes.NewHandler(router, voteUsecase, authUsecase)
			wiki.NewHandler(router, wikiUsecase, authUsecase)
			chat.NewWordFilterHandler(router, configUsecase, wordFilterUsecase, chatUsecase, authUsecase)

			playerqueueRepo := playerqueue.NewPlayerqueueRepository(dbConn, personUsecase)
			// Pre-load some messages into queue message cache
			chatlogs, errChatlogs := playerqueueRepo.Query(ctx, playerqueue.PlayerqueueQueryOpts{QueryFilter: domain.QueryFilter{Limit: 100}})
			if errChatlogs != nil {
				slog.Error("Failed to warm playerqueue chatlogs", log.ErrAttr(err))
				chatlogs = []playerqueue.ChatLog{}
			}
			playerqueueUC := playerqueue.NewPlayerqueueUsecase(playerqueueRepo, personUsecase, serversUC, stateUsecase, chatlogs, notificationUsecase)
			go playerqueueUC.Start(ctx)
			playerqueue.NewPlayerqueueHandler(router, authUsecase, configUsecase, playerqueueUC)

			if conf.Debug.AddRCONLogAddress != "" {
				go stateUsecase.LogAddressAdd(ctx, conf.Debug.AddRCONLogAddress)
				defer stateUsecase.LogAddressDel(ctx, conf.Debug.AddRCONLogAddress)
			}

			memberships := ban.NewMemberships(banRepo, tfapiClient)
			banExpirations := ban.NewExpirationMonitor(banUsecase, personUsecase, notificationUsecase, configUsecase)

			go func() {
				if errSync := anticheatUsecase.SyncDemoIDs(ctx, 100); errSync != nil {
					slog.Error("failed to sync anticheat demos")
				}

				go memberships.Update(ctx)
				go banExpirations.Update(ctx)
				go blocklistUsecase.Sync(ctx)
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
						if errMeta := reportUsecase.GenerateMetaStats(ctx); errMeta != nil {
							slog.Error("Failed to generate meta stats", log.ErrAttr(errMeta))
						}
					}()
				case <-blocklistTicker.C:
					go blocklistUsecase.Sync(ctx)
				case <-demoTicker.C:
					go demos.Cleanup(ctx)
					if errSync := anticheatUsecase.SyncDemoIDs(ctx, 100); errSync != nil {
						slog.Error("failed to sync anticheat demos")
					}
				}
			}()

			// River Queue
			workers := createQueueWorkers(
				personUsecase,
				notificationUsecase,
				discordUsecase,
				authRepo,
				patreonUsecase,
				reportUsecase,
				discordOAuthUsecase)

			queueClient, errClient := queue.New(dbConn.Pool(), workers, createPeriodicJobs())
			if errClient != nil {
				slog.Error("Failed to setup job queue", log.ErrAttr(errClient))

				return errClient
			}

			if errClientStart := queueClient.Start(ctx); errClientStart != nil {
				slog.Error("Failed to start job client", log.ErrAttr(errClientStart))

				return errors.Join(errClientStart, queue.ErrStartQueue)
			}

			notificationUsecase.SetQueueClient(queueClient)

			httpServer := httphelper.NewServer(conf.Addr(), router)

			demoDownloader := demo.NewDownloader(configUsecase, dbConn, serversUC, assets, demos, anticheatUsecase)
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
