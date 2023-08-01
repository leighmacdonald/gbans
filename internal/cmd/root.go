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
// unban asn - Unban an ASN
// unban cidr - Unban a CIDR network or IP
// unban steam - Unban a steam profile
package cmd

import (
	"os"

	"github.com/leighmacdonald/gbans/internal/app"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands.
func rootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gbans",
		Short: "",
		Long:  ``,
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cmd := setupRootCmd()
	if errExecute := cmd.Execute(); errExecute != nil {
		os.Exit(1)
	}
}

func setupRootCmd() *cobra.Command {
	if app.BuildVersion == "" {
		app.BuildVersion = "master"
	}

	root := rootCmd()

	root.Version = app.BuildVersion

	importCommands := importCmd()
	importCommands.AddCommand(importConnectionsCmd())
	importCommands.AddCommand(importMessagesCmd())

	netCommands := netCmd()
	netCommands.AddCommand(netUpdateCmd())

	root.AddCommand(netCommands)
	root.AddCommand(importCommands)
	root.AddCommand(serveCmd())
	// root.PersistentFlags().StringVar(&cfgFile, "config", "gbans.yml", "config file (default is $HOME/.gbans.yaml)").

	return root
}
