package cmd

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/app"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"time"
)

var downAll = false

// migrateCmd loads the db schema
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Create or update the database schema",
	Run: func(cmd *cobra.Command, args []string) {
		rootLogger := app.MustCreateLogger("")
		defer func() {
			if errSync := rootLogger.Sync(); errSync != nil {
				fmt.Printf("Failed to sync log: %v\n", errSync)
			}
		}()
		act := store.MigrateUp
		if downAll {
			act = store.MigrateDn
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*300)
		defer cancel()
		database, errStore := store.New(ctx, rootLogger, config.DB.DSN)
		if errStore != nil {
			rootLogger.Fatal("Failed to initialize database connection", zap.Error(errStore))
		}
		if errMigrate := database.Migrate(store.MigrationAction(act)); errMigrate != nil {
			if errMigrate.Error() == "no change" {
				rootLogger.Info("Migration at latest version")
			} else {
				rootLogger.Fatal("Could not migrate schema", zap.Error(errMigrate))
			}
		} else {
			rootLogger.Info("Migration completed successfully")
		}
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.Flags().BoolVarP(&downAll, "down", "d", false, "Fully reverts all migrations")
}
