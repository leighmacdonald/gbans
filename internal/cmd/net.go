package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/leighmacdonald/gbans/internal/app"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
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
		rootLogger := app.MustCreateLogger("")
		defer func() {
			if errSync := rootLogger.Sync(); errSync != nil {
				fmt.Printf("Failed to sync log: %v\n", errSync)
			}
		}()
		connCtx, cancelConn := context.WithTimeout(context.Background(), time.Second*5)
		defer cancelConn()
		errStore := store.Init(connCtx, rootLogger)
		if errStore != nil {
			rootLogger.Fatal("Failed to initialize database connection", zap.Error(errStore))
		}
		defer func() {
			if errClose := store.Close(); errClose != nil {
				rootLogger.Error("Failed to close database cleanly", zap.Error(errClose))
			}
		}()
		ctx := context.Background()
		if errUpdate := ip2location.Update(ctx, config.Net.CachePath, config.Net.IP2Location.Token); errUpdate != nil {
			rootLogger.Fatal("Failed to update", zap.Error(errUpdate))
		}
		blockListData, errRead := ip2location.Read(config.Net.CachePath)
		if errRead != nil {
			rootLogger.Fatal("Failed to read data", zap.Error(errRead))
		}
		updateCtx, cancelUpdate := context.WithTimeout(context.Background(), time.Minute*30)
		defer cancelUpdate()
		if errInsert := store.InsertBlockListData(updateCtx, blockListData); errInsert != nil {
			rootLogger.Fatal("Failed to import", zap.Error(errInsert))
		}
	},
}

func init() {
	netCmd.AddCommand(netUpdateCmd)
	rootCmd.AddCommand(netCmd)
}
