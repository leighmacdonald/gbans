/*
Copyright © 2020 Leigh MacDonald <leigh.macdonald@gmail.com>

*/
package cmd

import (
	"fmt"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/service"
	"github.com/spf13/cobra"
	"os"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gbans",
	Short: "",
	Long:  ``,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	if service.BuildVersion == "" {
		service.BuildVersion = "master"
	}
	rootCmd.Version = service.BuildVersion
	cobra.OnInitialize(func() {
		config.Read(cfgFile)
	})
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gbans.yaml)")
}
