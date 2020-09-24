/*
Copyright © 2020 Leigh MacDonald <leigh.macdonald@gmail.com>

*/
package cmd

import (
	"github.com/leighmacdonald/gbans/config"
	"github.com/leighmacdonald/gbans/service"
	"github.com/spf13/cobra"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		service.Start(config.DB.Path, config.HTTP.Addr())
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
