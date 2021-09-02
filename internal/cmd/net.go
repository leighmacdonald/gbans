package cmd

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/spf13/cobra"
	"log"
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
		if err := ip2location.Update(config.Net.CachePath, config.Net.IP2Location.Token); err != nil {
			log.Fatalf("Failed to update")
		}
		d, err := ip2location.Read(config.Net.CachePath)
		if err != nil {
			log.Fatalf("Failed to read")
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*600)
		defer cancel()
		if err := db.InsertBlockListData(ctx, d); err != nil {
			log.Fatalf("Failed to import")
		}
	},
}

func init() {
	netCmd.AddCommand(netUpdateCmd)
	rootCmd.AddCommand(netCmd)
}
