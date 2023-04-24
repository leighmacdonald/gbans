package cmd

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/app"
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
		gbans := app.New(rootCtx, rootLogger)
		if errApp := gbans.Start(); errApp != nil {
			rootLogger.Error("Application error", zap.Error(errApp))
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
