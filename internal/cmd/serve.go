package cmd

import (
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
		application, err := app.New()
		if err != nil {
			log.Panicf("Application error: %v", err)
		}
		application.Start()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
