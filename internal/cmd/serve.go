package cmd

import (
	"context"
	"fmt"
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
		rootCtx := context.Background()
		rootLogger := app.MustCreateLogger("")
		defer func() {
			if errSync := rootLogger.Sync(); errSync != nil {
				fmt.Printf("Failed to sync log: %v\n", errSync)
			}
		}()
		if errConnect := store.Init(rootCtx, rootLogger, ""); errConnect != nil {
			rootLogger.Fatal("Cannot initialize database", zap.Error(errConnect))
		}
		defer func() {
			if errClose := store.Close(); errClose != nil {
				rootLogger.Error("Failed to close database cleanly")
			}
		}()

		app.Init(rootCtx, rootLogger)

		errWeb := web.Setup(rootLogger)
		if errWeb != nil {
			rootLogger.Fatal("Failed to setup web", zap.Error(errWeb))
		}
		web.Start(rootCtx)

		if errDiscord := discord.Start(rootCtx, rootLogger); errDiscord != nil {
			rootLogger.Error("Failed to initialize discord", zap.Error(errDiscord))
		}

	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
