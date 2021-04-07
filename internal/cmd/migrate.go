package cmd

import (
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/service"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var recreateSchema = false

// migrateCmd loads the db schema
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Create or update the database schema",
	Run: func(cmd *cobra.Command, args []string) {
		// TODO add user confirmation on recreate
		service.Init(config.DB.DSN)
		if err := service.Migrate(recreateSchema); err != nil {
			log.Fatalf("Could not create server: %v", err)
		}
		log.Infof("Added server %s with key %s - This key must be added to your servers gbans.cfg under server_key",
			addServer.ServerName, addServer.Password)
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)

	migrateCmd.Flags().BoolVarP(&recreateSchema, "recreate", "r", false, "Recreate the database, WARN: this wipes *ALL* data")
}
