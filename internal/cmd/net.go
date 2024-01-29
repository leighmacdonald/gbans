package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/leighmacdonald/gbans/internal/blocklist"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/log"
	"github.com/leighmacdonald/gbans/internal/network"
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
			cu := config.NewConfigUsecase(config.NewConfigRepository())
			if errConfig := cu.Read(false); errConfig != nil {
				panic(fmt.Sprintf("Failed to read config: %v", errConfig))
			}

			conf := cu.Config()
			rootLogger := log.MustCreate(conf, nil)
			defer func() {
				_ = rootLogger.Sync()
			}()

			ctx := context.Background()

			connCtx, cancelConn := context.WithTimeout(ctx, time.Second*5)
			defer cancelConn()
			db := database.New(rootLogger, conf.DB.DSN, false, conf.DB.LogQueries)

			rootLogger.Info("Connecting to database")
			if errConnect := db.Connect(connCtx); errConnect != nil {
				rootLogger.Fatal("Failed to connect to database", zap.Error(errConnect))
			}

			defer func() {
				if errClose := db.Close(); errClose != nil {
					rootLogger.Error("Failed to close database cleanly", zap.Error(errClose))
				}
			}()

			nu := network.NewNetworkUsecase(
				rootLogger,
				network.NewNetworkRepository(db),
				blocklist.NewBlocklistUsecase(blocklist.NewBlocklistRepository(db)))

			if errUpdate := ip2location.Update(ctx, conf.IP2Location.CachePath, conf.IP2Location.Token); errUpdate != nil {
				rootLogger.Fatal("Failed to update", zap.Error(errUpdate))
			}

			rootLogger.Info("Reading data")

			blockListData, errRead := ip2location.Read(conf.IP2Location.CachePath)
			if errRead != nil {
				rootLogger.Fatal("Failed to read data", zap.Error(errRead))
			}

			updateCtx, cancelUpdate := context.WithTimeout(ctx, time.Minute*30)
			defer cancelUpdate()

			rootLogger.Info("Starting import")

			if errInsert := nu.InsertBlockListData(updateCtx, rootLogger, blockListData); errInsert != nil {
				rootLogger.Fatal("Failed to import", zap.Error(errInsert))
			}
			rootLogger.Info("Import Complete")

			os.Exit(0)
		},
	}
}
