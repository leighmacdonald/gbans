package cmd

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/store"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"time"
)

var importPath = ""

// importCmd loads the db schema
var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Create or update the database schema",
	Run: func(cmd *cobra.Command, args []string) {
		// TODO add user confirmation on recreate
		database, errStore := store.New(config.DB.DSN)
		if errStore != nil {
			log.Fatalf("Failed to connect to database: %v", errStore)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
		defer cancel()
		if errImport := database.Import(ctx, importPath); errImport != nil {
			log.Fatalf("Failed to import: %v", errImport)
		}
	},
}

func init() {
	rootCmd.AddCommand(importCmd)

	importCmd.Flags().StringVarP(&importPath, "import", "i", "", "Path to data to load")
}
