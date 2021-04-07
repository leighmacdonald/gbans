package cmd

import (
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/service"
	log "github.com/sirupsen/logrus"
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
		if err := service.Import(importPath); err != nil {
			log.Fatalf("Failed to import: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(importCmd)

	importCmd.Flags().StringVarP(&importPath, "import", "i", "", "Path to data to load")
}
