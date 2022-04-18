package cmd

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "stats",
	Long:  "stats",
}

var statsRebuildCmd = &cobra.Command{
	Use:   "rebuild",
	Short: "Rebuild all stats tables",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		database, errStore := store.New(config.DB.DSN)
		if errStore != nil {
			log.Fatalf("Failed to initialize database connection: %v", errStore)
		}
		defer func() {
			if errClose := database.Close(); errClose != nil {
				log.Errorf("Failed to close database cleanly: %v", errClose)
			}
		}()
		var (
			inputChan        = make(chan model.ServerEvent)
			offset    uint64 = 0
			limit     uint64 = 100_000
		)
		var errRead error
		for errRead == nil {
			rows, errFetch := database.GetReplayLogs(ctx, offset*limit, limit)
			if errFetch != nil {
				log.Errorf("Error fetching replat logs: %v", errFetch)
				break
			}
			for range rows {
				// do stuff
			}
			offset++
		}
		if errRead != nil {
			log.Errorf("Failed to read event: %v", errRead)
		}
		close(inputChan)
	},
}

func init() {
	statsCmd.AddCommand(statsRebuildCmd)
	rootCmd.AddCommand(statsCmd)
}
