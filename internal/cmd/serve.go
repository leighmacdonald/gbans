package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/app"
	"github.com/leighmacdonald/gbans/internal/appeal"
	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/auth"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/blocklist"
	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/contest"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/demo"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/forum"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/match"
	"github.com/leighmacdonald/gbans/internal/metrics"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/news"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/patreon"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/report"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/srcds"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/steamgroup"
	"github.com/leighmacdonald/gbans/internal/votes"
	"github.com/leighmacdonald/gbans/internal/wiki"
	"github.com/leighmacdonald/gbans/internal/wordfilter"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/spf13/cobra"
)

// serveCmd represents the serve command.
func serveCmd() *cobra.Command { //nolint:maintidx
	return &cobra.Command{
		Use:   "serve",
		Short: "Starts the gbans service",
		Long:  `Starts the main gbans application`,
		Run: func(cmd *cobra.Command, args []string) {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			configUsecase := config.NewConfigUsecase(config.NewConfigRepository())
			if errConfig := configUsecase.Read(false); errConfig != nil {
				panic(fmt.Sprintf("Failed to read config: %v", errConfig))
			}

			conf := configUsecase.Config()

			var sentryClient *sentry.Client
			var errSentry error

			sentryClient, errSentry = log.NewSentryClient(conf.Log.SentryDSN, conf.Log.SentryTrace, conf.Log.SentrySampleRate, app.BuildVersion)

			logCloser := log.MustCreateLogger(conf.Log.File, conf.Log.Level)
			defer logCloser()

			if errSentry != nil {
				slog.Error("Failed to setup sentry client")
			} else {
				defer sentryClient.Flush(2 * time.Second)
			}

			slog.Info("Starting gbans...",
				slog.String("version", app.BuildVersion),
				slog.String("commit", app.BuildCommit),
				slog.String("date", app.BuildDate))

			dbUsecase := database.New(conf.DB.DSN, conf.DB.AutoMigrate, conf.DB.LogQueries)
			if errConnect := dbUsecase.Connect(ctx); errConnect != nil {
				slog.Error("Cannot initialize database", log.ErrAttr(errConnect))

				return
			}

			defer func() {
				if errClose := dbUsecase.Close(); errClose != nil {
					slog.Error("Failed to close database cleanly")
				}
			}()

			eventBroadcaster := fp.NewBroadcaster[logparse.EventType, logparse.ServerEvent]()
			weaponsMap := fp.NewMutexMap[logparse.Weapon, int]()

			discordRepository, errDR := discord.NewDiscordRepository(conf)
			if errDR != nil {
				slog.Error("Cannot initialize discord", log.ErrAttr(errDR))

				return
			}

			wordFilterUsecase := wordfilter.NewWordFilterUsecase(wordfilter.NewWordFilterRepository(dbUsecase))
			if err := wordFilterUsecase.Import(ctx); err != nil {
				slog.Error("Failed to load word filters", log.ErrAttr(err))

				return
			}

			discordUsecase := discord.NewDiscordUsecase(discordRepository, wordFilterUsecase)

			if err := discordUsecase.Start(); err != nil {
				slog.Error("Failed to start discord", log.ErrAttr(err))

				return
			}

			defer discordUsecase.Shutdown(conf.Discord.GuildID)

			// // Initialize minio client object.
			// minioClient, errMinio := minio.New(conf.S3Store.Endpoint, &minio.Options{
			//	Creds:  credentials.NewStaticV4(conf.S3Store.AccessKey, conf.S3Store.SecretKey, ""),
			//	Secure: conf.S3Store.SSL,
			// })
			// if errMinio != nil {
			//	slog.Error("Cannot initialize minio", log.ErrAttr(errDR))
			//
			//	return
			// }

			personUsecase := person.NewPersonUsecase(person.NewPersonRepository(dbUsecase), configUsecase)

			blocklistUsecase := blocklist.NewBlocklistUsecase(blocklist.NewBlocklistRepository(dbUsecase))

			networkUsecase := network.NewNetworkUsecase(eventBroadcaster, network.NewNetworkRepository(dbUsecase), blocklistUsecase, personUsecase)
			if err := networkUsecase.LoadNetBlocks(ctx); err != nil {
				slog.Error("Failed to load network blocks", log.ErrAttr(err))

				return
			}

			go networkUsecase.Start(ctx)

			assetRepository := asset.NewLocalRepository(dbUsecase, configUsecase)
			if errInitAssets := assetRepository.Init(ctx); errInitAssets != nil {
				slog.Error("Failed to init local asset repo", log.ErrAttr(errInitAssets))

				return
			}

			dbUsecase.
				Builder().
				Select("a.asset_id", "a.bucket", "a.mime_type", "a.size", "a.name", "m.author_id").
				From("asset_temp")

			// assetRepository := asset.NewS3Repository(dbUsecase, minioClient, conf.S3Store.Region)
			// if errInit := assetRepository.Init(ctx); errInit != nil {
			//	slog.Error("Failed to ensure s3 buckets exist", log.ErrAttr(errInit))
			// }

			assetUsecase := asset.NewAssetUsecase(assetRepository)

			serversUsecase := servers.NewServersUsecase(servers.NewServersRepository(dbUsecase))
			demoUsecase := demo.NewDemoUsecase(domain.BucketDemo, demo.NewDemoRepository(dbUsecase), assetUsecase, configUsecase, serversUsecase)
			go demoUsecase.Start(ctx)

			banGroupUsecase := steamgroup.NewBanGroupUsecase(steamgroup.NewSteamGroupRepository(dbUsecase))
			reportUsecase := report.NewReportUsecase(report.NewReportRepository(dbUsecase), discordUsecase, configUsecase, personUsecase, demoUsecase)

			stateUsecase := state.NewStateUsecase(eventBroadcaster,
				state.NewStateRepository(state.NewCollector(serversUsecase)), configUsecase, serversUsecase)

			go func() {
				if err := stateUsecase.Start(ctx); err != nil {
					slog.Error("Failed to start state tracker", log.ErrAttr(err))
				}
			}()

			banRepository := ban.NewBanSteamRepository(dbUsecase, personUsecase, networkUsecase)
			banUsecase := ban.NewBanSteamUsecase(banRepository, personUsecase, configUsecase, discordUsecase, banGroupUsecase, reportUsecase, stateUsecase)

			banASNUsecase := ban.NewBanASNUsecase(ban.NewBanASNRepository(dbUsecase), discordUsecase, networkUsecase)

			banNetUsecase := ban.NewBanNetUsecase(ban.NewBanNetRepository(dbUsecase), personUsecase, configUsecase, discordUsecase, stateUsecase)

			ban.NewBanNetRepository(dbUsecase)

			apu := appeal.NewAppealUsecase(appeal.NewAppealRepository(dbUsecase), banUsecase, personUsecase, discordUsecase, configUsecase)
			matchRepo := match.NewMatchRepository(eventBroadcaster, dbUsecase, personUsecase, serversUsecase, discordUsecase, stateUsecase, weaponsMap)
			go matchRepo.Start(ctx)

			matchUsecase := match.NewMatchUsecase(matchRepo, stateUsecase, serversUsecase, discordUsecase)

			chatRepository := chat.NewChatRepository(dbUsecase, personUsecase, wordFilterUsecase, matchUsecase, eventBroadcaster)
			go chatRepository.Start(ctx)

			chatUsecase := chat.NewChatUsecase(configUsecase, chatRepository, wordFilterUsecase, stateUsecase, banUsecase, personUsecase, discordUsecase)
			go chatUsecase.Start(ctx)

			forumUsecase := forum.NewForumUsecase(forum.NewForumRepository(dbUsecase))

			metricsUsecase := metrics.NewMetricsUsecase(eventBroadcaster)
			go metricsUsecase.Start(ctx)

			go forumUsecase.Start(ctx)

			newsUsecase := news.NewNewsUsecase(news.NewNewsRepository(dbUsecase))
			notificationUsecase := notification.NewNotificationUsecase(notification.NewNotificationRepository(dbUsecase), personUsecase)
			patreonUsecase := patreon.NewPatreonUsecase(patreon.NewPatreonRepository(dbUsecase))
			go patreonUsecase.Start(ctx)

			srcdsUsecase := srcds.NewSrcdsUsecase(configUsecase, serversUsecase, personUsecase, reportUsecase, discordUsecase)

			wikiUsecase := wiki.NewWikiUsecase(wiki.NewWikiRepository(dbUsecase))

			authUsecase := auth.NewAuthUsecase(auth.NewAuthRepository(dbUsecase), configUsecase, personUsecase, banUsecase, serversUsecase)
			go authUsecase.Start(ctx)

			voteUsecase := votes.NewVoteUsecase(votes.NewVoteRepository(dbUsecase), personUsecase, matchUsecase, discordUsecase, eventBroadcaster)
			go voteUsecase.Start(ctx)

			contestUsecase := contest.NewContestUsecase(contest.NewContestRepository(dbUsecase))

			// start workers
			if conf.General.Mode == domain.ReleaseMode {
				gin.SetMode(gin.ReleaseMode)
			} else {
				gin.SetMode(gin.DebugMode)
			}

			go ban.Start(ctx, banUsecase, banNetUsecase, banASNUsecase, personUsecase, discordUsecase, configUsecase)

			router, errRouter := httphelper.CreateRouter(conf, app.Version())
			if errRouter != nil {
				slog.Error("Could not setup router", log.ErrAttr(errRouter))

				return
			}

			discordHandler := discord.NewDiscordHandler(discordUsecase, personUsecase, banUsecase,
				stateUsecase, serversUsecase, configUsecase, networkUsecase, wordFilterUsecase, matchUsecase, banNetUsecase, banASNUsecase)
			discordHandler.Start(ctx)

			appeal.NewAppealHandler(router, apu, banUsecase, configUsecase, personUsecase, discordUsecase, authUsecase)
			auth.NewAuthHandler(router, authUsecase, configUsecase, personUsecase)
			ban.NewBanHandler(router, banUsecase, discordUsecase, personUsecase, configUsecase, authUsecase)
			ban.NewBanNetHandler(router, banNetUsecase, authUsecase)
			ban.NewBanASNHandler(router, banASNUsecase, authUsecase)
			steamgroup.NewSteamgroupHandler(router, banGroupUsecase, authUsecase)
			blocklist.NewBlocklistHandler(router, blocklistUsecase, networkUsecase, authUsecase)
			chat.NewChatHandler(router, chatUsecase, authUsecase)
			contest.NewContestHandler(router, contestUsecase, configUsecase, assetUsecase, authUsecase)
			demo.NewDemoHandler(router, demoUsecase)
			forum.NewForumHandler(router, forumUsecase, authUsecase)
			match.NewMatchHandler(ctx, router, matchUsecase, serversUsecase, authUsecase, configUsecase)
			asset.NewAssetHandler(router, configUsecase, assetUsecase, authUsecase)
			metrics.NewMetricsHandler(router)
			network.NewNetworkHandler(router, networkUsecase, authUsecase)
			news.NewNewsHandler(router, newsUsecase, discordUsecase, authUsecase)
			notification.NewNotificationHandler(router, notificationUsecase, authUsecase)
			patreon.NewPatreonHandler(router, patreonUsecase, authUsecase)
			person.NewPersonHandler(router, configUsecase, personUsecase, authUsecase)
			report.NewReportHandler(router, reportUsecase, configUsecase, discordUsecase, personUsecase, authUsecase, demoUsecase)
			servers.NewServerHandler(router, serversUsecase, stateUsecase, authUsecase, personUsecase)
			srcds.NewSRCDSHandler(router, srcdsUsecase, serversUsecase, personUsecase, assetUsecase,
				reportUsecase, banUsecase, networkUsecase, banGroupUsecase, demoUsecase, authUsecase, banASNUsecase, banNetUsecase,
				configUsecase, discordUsecase, stateUsecase, blocklistUsecase)
			votes.NewVoteHandler(router, voteUsecase, authUsecase)
			wiki.NewWIkiHandler(router, wikiUsecase, authUsecase)
			wordfilter.NewWordFilterHandler(router, configUsecase, wordFilterUsecase, chatUsecase, authUsecase)

			if conf.Debug.AddRCONLogAddress != "" {
				slog.Info("Enabling log forwarding for local host")
				stateUsecase.LogAddressAdd(ctx, conf.Debug.AddRCONLogAddress)
			}

			if conf.SSH.Enabled {
				demoFetcher := demo.NewFetcher(dbUsecase, configUsecase, serversUsecase, assetUsecase, demoUsecase)
				go demoFetcher.Start(ctx)
			}

			httpServer := httphelper.NewHTTPServer(conf.HTTP.TLS, conf.HTTP.Addr(), router)

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
		},
	}
}
