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
	"github.com/leighmacdonald/gbans/internal/http_helper"
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
func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Starts the gbans service",
		Long:  `Starts the main gbans application`,
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			rootCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
			defer stop()

			cu := config.NewConfigUsecase(config.NewConfigRepository())
			if errConfig := cu.Read(false); errConfig != nil {
				panic(fmt.Sprintf("Failed to read config: %v", errConfig))
			}

			conf := cu.Config()

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

			db := database.New(rootLogger, conf.DB.DSN, conf.DB.AutoMigrate, conf.DB.LogQueries)
			if errConnect := db.Connect(rootCtx); errConnect != nil {
				rootLogger.Fatal("Cannot initialize database", zap.Error(errConnect))
			}

			defer func() {
				if errClose := db.Close(); errClose != nil {
					rootLogger.Error("Failed to close database cleanly")
				}
			}()

			eventBroadcaster := fp.NewBroadcaster[logparse.EventType, logparse.ServerEvent]()
			wm := fp.NewMutexMap[logparse.Weapon, int]()

			dr, errDR := discord.NewDiscordRepository(rootLogger, conf)
			if errDR != nil {
				rootLogger.Fatal("Cannot initialize discord", zap.Error(errDR))
			}
			du := discord.NewDiscordUsecase(dr)
			// du.Start()
			defer du.Shutdown(conf.Discord.GuildID)

			// Initialize minio client object.
			minioClient, errMinio := minio.New(conf.S3.Endpoint, &minio.Options{
				Creds:  credentials.NewStaticV4(conf.S3.AccessKey, conf.S3.SecretKey, ""),
				Secure: conf.S3.SSL,
			})
			if errMinio != nil {
				rootLogger.Fatal("Cannot initialize minio", zap.Error(errDR))
			}

			pu := person.NewPersonUsecase(rootLogger, person.NewPersonRepository(db))

			blu := blocklist.NewBlocklistUsecase(blocklist.NewBlocklistRepository(db))
			nu := network.NewNetworkUsecase(rootLogger, eventBroadcaster, network.NewNetworkRepository(db), blu, pu)
			nu.LoadNetBlocks(ctx)
			au := asset.NewAssetUsecase(asset.NewS3Repository(rootLogger, db, minioClient, conf.S3.Region))
			meu := media.NewMediaUsecase(conf.S3.BucketMedia, media.NewMediaRepository(db), au)
			deu := demo.NewDemoUsecase(rootLogger, conf.S3.BucketDemo, demo.NewDemoRepository(db), au, cu)
			go deu.Cleaner(ctx)

			sgu := steamgroup.NewBanGroupUsecase(rootLogger, steamgroup.NewSteamGroupRepository(db))
			ru := report.NewReportUsecase(rootLogger, report.NewReportRepository(db), du, cu)
			sv := servers.NewServersUsecase(servers.NewServersRepository(db))
			st := state.NewStateUsecase(rootLogger, eventBroadcaster, state.NewStateRepository(state.NewCollector(rootLogger, sv)), cu, sv)
			br := ban.NewBanRepository(db, pu, nu)
			bu := ban.NewBanUsecase(rootLogger, br, pu, cu, du, sgu, ru, st)

			apu := appeal.NewAppealUsecase(appeal.NewAppealRepository(db), bu, pu, du, cu)

			wfu := wordfilter.NewWordFilterUsecase(wordfilter.NewWordFilterRepository(db), du)
			wfu.Import(ctx)

			cr := chat.NewChatRepository(db, rootLogger, pu, wfu, eventBroadcaster)
			chu := chat.NewChatUsecase(rootLogger, cu, cr, wfu, st, bu, pu, du, st)
			fu := forum.NewForumUsecase(forum.NewForumRepository(db))
			mu := match.NewMatchUsecase(rootLogger, eventBroadcaster, match.NewMatchRepository(db, pu), st, sv, du, wm)
			neu := news.NewNewsUsecase(news.NewNewsRepository(db))
			nou := notification.NewNotificationUsecase(rootLogger, notification.NewNotificationRepository(db), pu)
			pat := patreon.NewPatreonUsecase(patreon.NewPatreonRepository(db))

			srcdsu := srcds.NewSrcdsUsecase(rootLogger, cu, sv, pu, ru, du)

			wu := wiki.NewWikiUsecase(wiki.NewWikiRepository(db, meu))

			athu := auth.NewAuthUsecase(rootLogger, auth.NewAuthRepository(db), cu, pu, bu, sv)

			cnu := contest.NewContestUsecase(contest.NewContestRepository(db))

			// start workers

			router, errRouter := http_helper.CreateRouter(rootLogger, conf, app.Version())
			if errRouter != nil {
				rootLogger.Fatal("Could not setup router", zap.Error(errRouter))
			}

			appeal.NewAppealHandler(rootLogger, router, apu, bu, cu, pu, du)
			auth.NewAuthHandler(rootLogger, router, athu, cu, pu)
			ban.NewBanHandler(rootLogger, router, bu, du, pu, cu)
			blocklist.NewBlocklistHandler(rootLogger, router, blu, nu)
			chat.NewChatHandler(rootLogger, router, chu)
			contest.NewContestHandler(rootLogger, router, cnu, cu, meu)
			demo.NewDemoHandler(rootLogger, router, deu)
			forum.NewForumHandler(rootLogger, router, fu)
			match.NewMatchHandler(ctx, rootLogger, router, mu)
			media.NewMediaHandler(rootLogger, router, meu, cu, au)
			metrics.NewMetricsHandler(rootLogger, router)
			network.NewNetworkHandler(rootLogger, router, nu)
			news.NewNewsHandler(rootLogger, router, neu, du)
			notification.NewNotificationHandler(rootLogger, router, nou)
			patreon.NewPatreonHandler(rootLogger, router, pat)
			report.NewReportHandler(rootLogger, router, ru, cu, du, pu)
			servers.NewServerHandler(rootLogger, router, sv, st, pu)
			srcds.NewSRCDSHandler(rootLogger, router, srcdsu, sv, pu, au, ru, au, bu, nu, sgu, deu)
			wiki.NewWIkiHandler(rootLogger, router, wu)
			wordfilter.NewWordFilterHandler(rootLogger, router, cu, wfu, chu)

			defer du.Shutdown(conf.Discord.GuildID)

			srv := http_helper.NewHTTPServer(conf.HTTP.TLS, conf.HTTP.Addr(), router)
			if conf.General.Mode == domain.ReleaseMode {
				gin.SetMode(gin.ReleaseMode)
			} else {
				gin.SetMode(gin.DebugMode)
			}

			go func() {
				<-ctx.Done()

				shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)

				defer cancel()

				if errShutdown := srv.Shutdown(shutdownCtx); errShutdown != nil { //nolint:contextcheck
					rootLogger.Error("Error shutting down http service", zap.Error(errShutdown))
				}
			}()

			errServe := srv.ListenAndServe()
			if errServe != nil && !errors.Is(errServe, http.ErrServerClosed) {
				rootLogger.Error("HTTP server returned error", zap.Error(errServe))
			}

			<-rootCtx.Done()
		},
	}
}
