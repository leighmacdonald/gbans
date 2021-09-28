package cmd

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"time"
)

var netCmd = &cobra.Command{
	Use:   "net",
	Short: "Network and client blocking functionality",
	Long:  `Network and client blocking functionality`,
}

var netUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update any enabled block lists",
	Long:  `Update any enabled block lists`,
	Run: func(cmd *cobra.Command, args []string) {
		db, err := store.New(config.DB.DSN)
		if err != nil {
			log.Fatalf("Failed to initialize db connection: %v", err)
		}
		defer db.Close()
		if err := ip2location.Update(config.Net.CachePath, config.Net.IP2Location.Token); err != nil {
			log.Fatalf("Failed to update")
		}
		d, errRead := ip2location.Read(config.Net.CachePath)
		if errRead != nil {
			log.Fatalf("Failed to read: %v", errRead)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*600)
		defer cancel()
		if errIns := db.InsertBlockListData(ctx, d); errIns != nil {
			log.Fatalf("Failed to import: %v", errIns)
		}
	},
}

func init() {
	netCmd.AddCommand(netUpdateCmd)
	rootCmd.AddCommand(netCmd)
}
