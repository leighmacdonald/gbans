package cmd

import (
	"github.com/leighmacdonald/gbans/config"
	"github.com/leighmacdonald/gbans/internal/service"
	"github.com/spf13/cobra"
)

var importPath = ""

// importCmd loads the db schema
var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Create or update the database schema",
	Run: func(cmd *cobra.Command, args []string) {
		// TODO add user confirmation on recreate
		service.Init(config.DB.DSN)
		service.Import(importPath)
	},
}

func init() {
	rootCmd.AddCommand(importCmd)

	importCmd.Flags().StringVarP(&importPath, "import", "i", "", "Path to data to load")
}
