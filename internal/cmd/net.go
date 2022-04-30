package cmd

import (
	"context"
	"time"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var netCmd = &cobra.Command{
	Use:   "net",
	Short: "Network and client blocking functionality",
	Long:  `Network and client blocking functionality`,
}

var netUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Updates ip2location dataset",
	Long:  `Updates ip2location dataset`,
	Run: func(cmd *cobra.Command, args []string) {
		connCtx, cancelConn := context.WithTimeout(context.Background(), time.Second*5)
		defer cancelConn()
		database, errStore := store.New(connCtx, config.DB.DSN)
		if errStore != nil {
			log.Fatalf("Failed to initialize database connection: %v", errStore)
		}
		defer func() {
			if errClose := database.Close(); errClose != nil {
				log.Errorf("Failed to close database cleanly: %v", errClose)
			}
		}()
		if errUpdate := ip2location.Update(config.Net.CachePath, config.Net.IP2Location.Token); errUpdate != nil {
			log.Fatalf("Failed to update")
		}
		blockListData, errRead := ip2location.Read(config.Net.CachePath)
		if errRead != nil {
			log.Fatalf("Failed to read: %v", errRead)
		}
		updateCtx, cancelUpdate := context.WithTimeout(context.Background(), time.Minute*30)
		defer cancelUpdate()
		if errInsert := database.InsertBlockListData(updateCtx, blockListData); errInsert != nil {
			log.Fatalf("Failed to import: %v", errInsert)
		}
	},
}

func init() {
	netCmd.AddCommand(netUpdateCmd)
	rootCmd.AddCommand(netCmd)
}
