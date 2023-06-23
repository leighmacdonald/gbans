package cmd

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/app"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/web"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
)

// serveCmd represents the serve command.
func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Starts the gbans service",
		Long:  `Start the main gbans application`,
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			rootCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
			defer stop()

			var conf config.Config
			if errConfig := config.Read(&conf); errConfig != nil {
				panic("Failed to read config")
			}

			rootLogger := app.MustCreateLogger(&conf)
			defer func() {
				if conf.Log.File != "" {
					_ = rootLogger.Sync()
				}
			}()
			if errConnect := store.Init(rootCtx, rootLogger, conf.DB.DSN, conf.DB.AutoMigrate); errConnect != nil {
				rootLogger.Fatal("Cannot initialize database", zap.Error(errConnect))
			}
			defer func() {
				if errClose := store.Close(); errClose != nil {
					rootLogger.Error("Failed to close database cleanly")
				}
			}()

			if errApp := app.Init(rootCtx, rootLogger, &conf); errApp != nil {
				rootLogger.Fatal("Failed to init app", zap.Error(errApp))
			}

			if errWeb := web.Init(rootLogger, &conf); errWeb != nil {
				rootLogger.Fatal("Failed to setup web", zap.Error(errWeb))
			}
			if errDiscord := discord.Start(rootLogger, &conf); errDiscord != nil {
				rootLogger.Error("Failed to initialize discord", zap.Error(errDiscord))
			}
			defer discord.Shutdown(conf.Discord.GuildID)
			if errWebStart := web.Start(rootCtx); errWebStart != nil {
				rootLogger.Error("Web returned error", zap.Error(errWebStart))
			}
			<-rootCtx.Done()
		},
	}
}
