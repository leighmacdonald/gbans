/*
Copyright © 2020 Leigh MacDonald <leigh.macdonald@gmail.com>

*/
package cmd

import (
	"github.com/leighmacdonald/gbans/service"
	"github.com/spf13/cobra"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		service.Start("./db.sqlite", "0.0.0.0:6969")
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serveCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	serveCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
