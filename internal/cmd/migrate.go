package cmd

import (
	"github.com/leighmacdonald/gbans/internal/service"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var downAll = false

// migrateCmd loads the db schema
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Create or update the database schema",
	Run: func(cmd *cobra.Command, args []string) {
		act := service.MigrateUp
		if downAll {
			act = service.MigrateDn
		}
		if err := service.Migrate(service.MigrationAction(act)); err != nil {
			if err.Error() == "no change" {
				log.Infof("Migration at latest version")
			} else {
				log.Fatalf("Could not migrate schema: %v", err)
			}
		}
		log.Infof("Added server %s with key %s - This key must be added to your servers gbans.cfg under server_key",
			addServer.ServerName, addServer.Password)
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)

	migrateCmd.Flags().BoolVarP(&downAll, "down", "d", false, "Fully reverts all migrations")
}
