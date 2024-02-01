package cmd

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/leighmacdonald/gbans/internal/log"
	"github.com/leighmacdonald/gbans/internal/match"
	"github.com/leighmacdonald/gbans/internal/media"
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
	"github.com/leighmacdonald/gbans/internal/wiki"
	"github.com/leighmacdonald/gbans/internal/wordfilter"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// serveCmd represents the serve command.
func serveCmd() *cobra.Command { //nolint:maintidx
	return &cobra.Command{
		Use:   "serve",
		Short: "Starts the gbans service",
		Long:  `Starts the main gbans application`,
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			rootCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
			defer stop()

			configUsecase := config.NewConfigUsecase(config.NewConfigRepository())
			if errConfig := configUsecase.Read(false); errConfig != nil {
				panic(fmt.Sprintf("Failed to read config: %v", errConfig))
			}

			conf := configUsecase.Config()

			var sentryClient *sentry.Client
			var errSentry error

			sentryClient, errSentry = log.NewSentryClient(conf.Log.SentryDSN, conf.Log.SentryTrace, conf.Log.SentrySampleRate, app.BuildVersion)

			rootLogger := log.MustCreate(conf, sentryClient)
			defer func() {
				if conf.Log.File != "" {
					_ = rootLogger.Sync()
				}
			}()

			if errSentry != nil {
				rootLogger.Error("Failed to setup sentry client")
			} else {
				defer sentryClient.Flush(2 * time.Second)
			}

			rootLogger.Info("Starting gbans...",
				zap.String("version", app.BuildVersion),
				zap.String("commit", app.BuildCommit),
				zap.String("date", app.BuildDate))

			dbUsecase := database.New(rootLogger, conf.DB.DSN, conf.DB.AutoMigrate, conf.DB.LogQueries)
			if errConnect := dbUsecase.Connect(rootCtx); errConnect != nil {
				rootLogger.Fatal("Cannot initialize database", zap.Error(errConnect))
			}

			defer func() {
				if errClose := dbUsecase.Close(); errClose != nil {
					rootLogger.Error("Failed to close database cleanly")
				}
			}()

			eventBroadcaster := fp.NewBroadcaster[logparse.EventType, logparse.ServerEvent]()
			weaponsMap := fp.NewMutexMap[logparse.Weapon, int]()

			dr, errDR := discord.NewDiscordRepository(rootLogger, conf)
			if errDR != nil {
				rootLogger.Fatal("Cannot initialize discord", zap.Error(errDR))
			}

			discordUsecase := discord.NewDiscordUsecase(dr)

			if err := discordUsecase.Start(); err != nil {
				rootLogger.Fatal("Failed to start discord", zap.Error(err))
			}

			defer discordUsecase.Shutdown(conf.Discord.GuildID)

			// Initialize minio client object.
			minioClient, errMinio := minio.New(conf.S3.Endpoint, &minio.Options{
				Creds:  credentials.NewStaticV4(conf.S3.AccessKey, conf.S3.SecretKey, ""),
				Secure: conf.S3.SSL,
			})
			if errMinio != nil {
				rootLogger.Fatal("Cannot initialize minio", zap.Error(errDR))
			}

			personUsecase := person.NewPersonUsecase(rootLogger, person.NewPersonRepository(dbUsecase))

			blocklistUsecase := blocklist.NewBlocklistUsecase(blocklist.NewBlocklistRepository(dbUsecase))

			networkUsecase := network.NewNetworkUsecase(rootLogger, eventBroadcaster, network.NewNetworkRepository(dbUsecase), blocklistUsecase, personUsecase)
			if err := networkUsecase.LoadNetBlocks(ctx); err != nil {
				rootLogger.Fatal("Failed to load network blocks", zap.Error(err))
			}

			assetUsecase := asset.NewAssetUsecase(asset.NewS3Repository(rootLogger, dbUsecase, minioClient, conf.S3.Region))
			mediaUsecase := media.NewMediaUsecase(conf.S3.BucketMedia, media.NewMediaRepository(dbUsecase), assetUsecase)
			demoUsecase := demo.NewDemoUsecase(rootLogger, conf.S3.BucketDemo, demo.NewDemoRepository(dbUsecase), assetUsecase, configUsecase)
			go demoUsecase.Start(ctx)

			banGroupUsecase := steamgroup.NewBanGroupUsecase(rootLogger, steamgroup.NewSteamGroupRepository(dbUsecase))
			reportUsecase := report.NewReportUsecase(rootLogger, report.NewReportRepository(dbUsecase), discordUsecase, configUsecase)
			serversUsecase := servers.NewServersUsecase(servers.NewServersRepository(dbUsecase))

			stateUsecase := state.NewStateUsecase(rootLogger, eventBroadcaster, state.NewStateRepository(state.NewCollector(rootLogger, serversUsecase)), configUsecase, serversUsecase)
			go stateUsecase.Start(ctx)

			banRepository := ban.NewBanSteamRepository(dbUsecase, personUsecase, networkUsecase)
			banUsecase := ban.NewBanSteamUsecase(rootLogger, banRepository, personUsecase, configUsecase, discordUsecase, banGroupUsecase, reportUsecase, stateUsecase)

			banASNUsecase := ban.NewBanASNUsecase(ban.NewBanASNRepository(dbUsecase), discordUsecase, networkUsecase)

			banNetUsecase := ban.NewBanNetUsecase(rootLogger, ban.NewBanNetRepository(dbUsecase), personUsecase, configUsecase, discordUsecase, stateUsecase)

			ban.NewBanNetRepository(dbUsecase)

			apu := appeal.NewAppealUsecase(appeal.NewAppealRepository(dbUsecase), banUsecase, personUsecase, discordUsecase, configUsecase)

			wordFilterUsecase := wordfilter.NewWordFilterUsecase(wordfilter.NewWordFilterRepository(dbUsecase), discordUsecase)
			if err := wordFilterUsecase.Import(ctx); err != nil {
				rootLogger.Fatal("Failed to load word filters", zap.Error(err))
			}

			chatRepository := chat.NewChatRepository(dbUsecase, rootLogger, personUsecase, wordFilterUsecase, eventBroadcaster)

			chatUsecase := chat.NewChatUsecase(rootLogger, configUsecase, chatRepository, wordFilterUsecase, stateUsecase, banUsecase, personUsecase, discordUsecase)
			go chatUsecase.Start(ctx)

			forumUsecase := forum.NewForumUsecase(forum.NewForumRepository(dbUsecase))

			metricsUsecase := metrics.NewMetricsUsecase(rootLogger, eventBroadcaster)
			go metricsUsecase.Start(ctx)

			go forumUsecase.Start(ctx)
			matchUsecase := match.NewMatchUsecase(rootLogger, eventBroadcaster, match.NewMatchRepository(dbUsecase, personUsecase), stateUsecase, serversUsecase, discordUsecase, weaponsMap)
			go matchUsecase.Start(ctx)
			newsUsecase := news.NewNewsUsecase(news.NewNewsRepository(dbUsecase))
			notificationUsecase := notification.NewNotificationUsecase(rootLogger, notification.NewNotificationRepository(dbUsecase), personUsecase)
			patreonUsecase := patreon.NewPatreonUsecase(rootLogger, patreon.NewPatreonRepository(dbUsecase))
			go patreonUsecase.Start(ctx)

			srcdsUsecase := srcds.NewSrcdsUsecase(rootLogger, configUsecase, serversUsecase, personUsecase, reportUsecase, discordUsecase)

			wikiUsecase := wiki.NewWikiUsecase(wiki.NewWikiRepository(dbUsecase, mediaUsecase))

			authUsecase := auth.NewAuthUsecase(rootLogger, auth.NewAuthRepository(dbUsecase), configUsecase, personUsecase, banUsecase, serversUsecase)
			go authUsecase.Start(ctx)

			contestUsecase := contest.NewContestUsecase(contest.NewContestRepository(dbUsecase))

			// start workers
			if conf.General.Mode == domain.ReleaseMode {
				gin.SetMode(gin.ReleaseMode)
			} else {
				gin.SetMode(gin.DebugMode)
			}

			go ban.Start(ctx, rootLogger, banUsecase, banNetUsecase, banASNUsecase, personUsecase, discordUsecase, configUsecase)

			router, errRouter := httphelper.CreateRouter(rootLogger, conf, app.Version())
			if errRouter != nil {
				rootLogger.Fatal("Could not setup router", zap.Error(errRouter))
			}

			discordHandler := discord.NewDiscordHandler(rootLogger, discordUsecase, personUsecase, banUsecase,
				stateUsecase, serversUsecase, configUsecase, networkUsecase, wordFilterUsecase, matchUsecase, banNetUsecase, banASNUsecase)
			discordHandler.Start()

			appeal.NewAppealHandler(rootLogger, router, apu, banUsecase, configUsecase, personUsecase, discordUsecase, authUsecase)
			auth.NewAuthHandler(rootLogger, router, authUsecase, configUsecase, personUsecase)
			ban.NewBanHandler(rootLogger, router, banUsecase, discordUsecase, personUsecase, configUsecase, authUsecase)
			ban.NewBanNetHandler(rootLogger, router, banNetUsecase, authUsecase)
			ban.NewBanASNHandler(rootLogger, router, banASNUsecase, authUsecase)
			blocklist.NewBlocklistHandler(rootLogger, router, blocklistUsecase, networkUsecase, authUsecase)
			chat.NewChatHandler(rootLogger, router, chatUsecase, authUsecase)
			contest.NewContestHandler(rootLogger, router, contestUsecase, configUsecase, mediaUsecase, authUsecase)
			demo.NewDemoHandler(rootLogger, router, demoUsecase)
			forum.NewForumHandler(rootLogger, router, forumUsecase, authUsecase)
			match.NewMatchHandler(ctx, rootLogger, router, matchUsecase, authUsecase)
			media.NewMediaHandler(rootLogger, router, mediaUsecase, configUsecase, assetUsecase, authUsecase)
			metrics.NewMetricsHandler(rootLogger, router)
			network.NewNetworkHandler(rootLogger, router, networkUsecase, authUsecase)
			news.NewNewsHandler(rootLogger, router, newsUsecase, discordUsecase, authUsecase)
			notification.NewNotificationHandler(rootLogger, router, notificationUsecase, authUsecase)
			patreon.NewPatreonHandler(rootLogger, router, patreonUsecase, authUsecase)
			person.NewPersonHandler(rootLogger, router, configUsecase, personUsecase, authUsecase)
			report.NewReportHandler(rootLogger, router, reportUsecase, configUsecase, discordUsecase, personUsecase, authUsecase)
			servers.NewServerHandler(rootLogger, router, serversUsecase, stateUsecase, authUsecase)
			srcds.NewSRCDSHandler(rootLogger, router, srcdsUsecase, serversUsecase, personUsecase, assetUsecase,
				reportUsecase, banUsecase, networkUsecase, banGroupUsecase, demoUsecase, authUsecase, banASNUsecase, banNetUsecase,
				configUsecase, discordUsecase, stateUsecase)
			wiki.NewWIkiHandler(rootLogger, router, wikiUsecase, authUsecase)
			wordfilter.NewWordFilterHandler(rootLogger, router, configUsecase, wordFilterUsecase, chatUsecase, authUsecase)

			defer discordUsecase.Shutdown(conf.Discord.GuildID)

			httpServer := httphelper.NewHTTPServer(conf.HTTP.TLS, conf.HTTP.Addr(), router)

			go func() {
				<-ctx.Done()

				shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)

				defer cancel()

				if errShutdown := httpServer.Shutdown(shutdownCtx); errShutdown != nil { //nolint:contextcheck
					rootLogger.Error("Error shutting down http service", zap.Error(errShutdown))
				}
			}()

			errServe := httpServer.ListenAndServe()
			if errServe != nil && !errors.Is(errServe, http.ErrServerClosed) {
				rootLogger.Error("HTTP server returned error", zap.Error(errServe))
			}

			<-rootCtx.Done()
		},
	}
}
