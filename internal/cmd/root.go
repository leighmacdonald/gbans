// Package cmd implements the CLI (Command Line Interface) of the application.
//
// net update - Download and import the latest ip2location databases
// serve - The main application service entry point
// setup - Perform an initial setup task, needs to be ran once.
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

	refreshCommands := refreshCmd()
	refreshCommands.AddCommand(refreshFiltersCmd())

	netCommands := netCmd()
	netCommands.AddCommand(netUpdateCmd())

	root.AddCommand(setupCmd())
	root.AddCommand(netCommands)
	root.AddCommand(serveCmd())
	root.AddCommand(refreshCommands)
	// root.PersistentFlags().StringVar(&cfgFile, "config", "gbans.yml", "config file (default is $HOME/.gbans.yaml)").

	return root
}
