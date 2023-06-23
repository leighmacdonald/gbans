// Package cmd implements the CLI (Command Line Interface) of the application.
//
// ban asn - Ban based on ASN
// ban cidr - Ban an IP or network with CIDR notation
// ban steam - Ban a player via steamid or vanity name
// import - Imports bans from a folder in json format
// migrate - Initiate a database migration manually
// net update - Download and import the latest ip2location databases
// seed - Pre seed the database with data, used for development mostly
// serve - The main application service entry point
// server create
// server delete
// server list
// server update
// unban asn - Unban a ASN
// unban cidr - Unban a CIDR network or IP
// unban steam - Unban a steam profile
package cmd

import (
	"log"
	"os"

	"github.com/leighmacdonald/gbans/internal/app"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/spf13/cobra"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "gbans",
	Short: "",
	Long:  ``,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if errExecute := rootCmd.Execute(); errExecute != nil {
		os.Exit(1)
	}
}

func init() {
	if app.BuildVersion == "" {
		app.BuildVersion = "master"
	}
	rootCmd.Version = app.BuildVersion
	cobra.OnInitialize(func() {
		if _, errRead := config.Read(); errRead != nil {
			log.Printf("Failed to read config: %v", errRead)
		}
	})
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "gbans.yml", "config file (default is $HOME/.gbans.yaml)")
}
