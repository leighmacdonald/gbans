package cmd

import (
	"github.com/leighmacdonald/gbans/internal/app"
	"github.com/spf13/cobra"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		app.Start()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
