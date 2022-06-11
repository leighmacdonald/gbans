package cmd

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/store"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"time"
)

var downAll = false

// migrateCmd loads the db schema
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Create or update the database schema",
	Run: func(cmd *cobra.Command, args []string) {
		act := store.MigrateUp
		if downAll {
			act = store.MigrateDn
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*300)
		defer cancel()
		database, errStore := store.New(ctx, config.DB.DSN)
		if errStore != nil {
			log.Fatalf("Failed to initialize database connection: %v", errStore)
		}
		if errMigrate := database.Migrate(store.MigrationAction(act)); errMigrate != nil {
			if errMigrate.Error() == "no change" {
				log.Infof("Migration at latest version")
			} else {
				log.Fatalf("Could not migrate schema: %v", errMigrate)
			}
		} else {
			log.Infof("Migration completed successfully")
		}
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)

	migrateCmd.Flags().BoolVarP(&downAll, "down", "d", false, "Fully reverts all migrations")
}
