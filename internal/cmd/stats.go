package cmd

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/app"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "stats",
}

var statsRebuildCmd = &cobra.Command{
	Use:   "rebuild",
	Short: "Rebuild all stats tables",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		db, err := store.New(config.DB.DSN)
		if err != nil {
			log.Fatalf("Failed to initialize db connection: %v", err)
		}
		defer db.Close()
		var (
			inputChan        = make(chan model.ServerEvent)
			offset    uint64 = 0
			limit     uint64 = 100_000
			acc              = app.NewStatTrak()
		)
		var errRead error
		for errRead == nil {
			rows, errFetch := db.GetReplayLogs(ctx, offset*limit, limit)
			if errFetch != nil {
				log.Errorf("Error fetching replat logs: %v", errFetch)
				break
			}
			for _, row := range rows {
				if errRead = acc.Read(row); errRead != nil {
					break
				}
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
