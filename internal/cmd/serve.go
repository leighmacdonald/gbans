package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/leighmacdonald/gbans/internal/app"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/store"
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

			database := store.New(rootLogger, conf.DB.DSN, conf.DB.AutoMigrate)
			if errConnect := database.Connect(rootCtx); errConnect != nil {
				rootLogger.Fatal("Cannot initialize database", zap.Error(errConnect))
			}

			defer func() {
				if errClose := database.Close(); errClose != nil {
					rootLogger.Error("Failed to close database cleanly")
				}
			}()

			bot, errBot := discord.New(rootLogger, &conf)
			if errBot != nil {
				rootLogger.Fatal("Failed to connect to perform initial discord connection")
			}

			application := app.New(&conf, database, bot, rootLogger)

			if errInit := application.Init(rootCtx); errInit != nil {
				rootLogger.Fatal("Failed to init app", zap.Error(errInit))
			}

			if errDiscord := bot.Start(); errDiscord != nil {
				rootLogger.Error("Failed to start discord", zap.Error(errDiscord))
			}

			defer bot.Shutdown(conf.Discord.GuildID)
			if errWebStart := application.StartHTTP(rootCtx); errWebStart != nil {
				rootLogger.Error("Web returned error", zap.Error(errWebStart))
			}
			<-rootCtx.Done()
		},
	}
}
