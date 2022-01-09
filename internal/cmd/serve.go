package cmd

import (
	"github.com/leighmacdonald/gbans/internal/app"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the gbans service",
	Long:  `Start the main gbans application`,
	Run: func(cmd *cobra.Command, args []string) {
		if errApp := app.Start(); errApp != nil {
			log.Errorf("Application error: %v", errApp)
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
