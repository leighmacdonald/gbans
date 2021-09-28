package cmd

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/app"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		application, err := app.New(ctx)
		if err != nil {
			log.Panicf("Application error: %v", err)
		}
		defer application.Close()
		application.Start()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
