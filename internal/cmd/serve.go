package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/leighmacdonald/gbans/internal/app"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/web"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the gbans service",
	Long:  `Start the main gbans application`,
	Run: func(cmd *cobra.Command, args []string) {
		rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		logFile := ""
		rootLogger := app.MustCreateLogger(logFile)
		defer func() {
			if logFile != "" {
				_ = rootLogger.Sync()
			}
		}()
		if errConnect := store.Init(rootCtx, rootLogger); errConnect != nil {
			rootLogger.Fatal("Cannot initialize database", zap.Error(errConnect))
		}
		defer func() {
			if errClose := store.Close(); errClose != nil {
				rootLogger.Error("Failed to close database cleanly")
			}
		}()

		if errApp := app.Init(rootCtx, rootLogger); errApp != nil {
			rootLogger.Fatal("Failed to init app", zap.Error(errApp))
		}

		errWeb := web.Init(rootLogger)
		if errWeb != nil {
			rootLogger.Fatal("Failed to setup web", zap.Error(errWeb))
		}
		if errDiscord := discord.Start(rootLogger); errDiscord != nil {
			rootLogger.Error("Failed to initialize discord", zap.Error(errDiscord))
		}
		defer discord.Shutdown()
		if errWeb := web.Start(rootCtx); errWeb != nil {
			rootLogger.Error("Web returned error", zap.Error(errWeb))
		}
		<-rootCtx.Done()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
