package cmd

import (
	"context"
	"os"
	"time"

	"github.com/leighmacdonald/gbans/internal/app"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func netCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "net",
		Short: "Network and client blocking functionality",
		Long:  `Network and client blocking functionality`,
	}
}

func netUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Updates ip2location dataset",
		Long:  `Updates ip2location dataset`,
		Run: func(cmd *cobra.Command, args []string) {
			var conf config.Config
			if errConfig := config.Read(&conf); errConfig != nil {
				panic("Failed to read config")
			}
			rootLogger := app.MustCreateLogger(&conf)
			defer func() {
				_ = rootLogger.Sync()
			}()

			ctx := context.Background()

			connCtx, cancelConn := context.WithTimeout(ctx, time.Second*5)
			defer cancelConn()
			database := store.New(rootLogger, conf.DB.DSN, false, conf.DB.LogQueries)

			rootLogger.Info("Connecting to database")
			if errConnect := database.Connect(connCtx); errConnect != nil {
				rootLogger.Fatal("Failed to connect to database", zap.Error(errConnect))
			}
			defer func() {
				if errClose := database.Close(); errClose != nil {
					rootLogger.Error("Failed to close database cleanly", zap.Error(errClose))
				}
			}()

			if errUpdate := ip2location.Update(ctx, conf.NetBans.CachePath, conf.NetBans.IP2Location.Token); errUpdate != nil {
				rootLogger.Fatal("Failed to update", zap.Error(errUpdate))
			}
			rootLogger.Info("Reading data")
			blockListData, errRead := ip2location.Read(conf.NetBans.CachePath)
			if errRead != nil {
				rootLogger.Fatal("Failed to read data", zap.Error(errRead))
			}
			updateCtx, cancelUpdate := context.WithTimeout(ctx, time.Minute*30)
			defer cancelUpdate()
			rootLogger.Info("Starting import")
			if errInsert := database.InsertBlockListData(updateCtx, blockListData); errInsert != nil {
				rootLogger.Fatal("Failed to import", zap.Error(errInsert))
			}
			rootLogger.Info("Import Complete")

			os.Exit(0)
		},
	}
}
