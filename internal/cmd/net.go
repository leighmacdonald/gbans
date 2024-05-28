package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/spf13/cobra"
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
		Run: func(_ *cobra.Command, _ []string) {
			ctx := context.Background()

			staticConfig, errStatic := config.ReadStaticConfig()
			if errStatic != nil {
				panic(fmt.Sprintf("Failed to read static config: %v", errStatic))
			}

			dbUsecase := database.New(staticConfig.DatabaseDSN, staticConfig.DatabaseAutoMigrate, staticConfig.DatabaseLogQueries)
			if errConnect := dbUsecase.Connect(ctx); errConnect != nil {
				slog.Error("Cannot initialize database", log.ErrAttr(errConnect))

				return
			}

			defer func() {
				if errClose := dbUsecase.Close(); errClose != nil {
					slog.Error("Failed to close database cleanly", log.ErrAttr(errClose))
				}
			}()

			configUsecase := config.NewConfigUsecase(staticConfig, config.NewConfigRepository(dbUsecase))

			if err := configUsecase.Init(ctx); err != nil {
				panic(fmt.Sprintf("Failed to init config: %v", err))
			}

			if errConfig := configUsecase.Reload(ctx); errConfig != nil {
				panic(fmt.Sprintf("Failed to read config: %v", errConfig))
			}

			conf := configUsecase.Config()
			logCloser := log.MustCreateLogger(conf.Log.File, conf.Log.Level)
			defer logCloser()

			eventBroadcaster := fp.NewBroadcaster[logparse.EventType, logparse.ServerEvent]()

			personUsecase := person.NewPersonUsecase(person.NewPersonRepository(dbUsecase), configUsecase)

			if errUpdate := ip2location.Update(ctx, conf.GeoLocation.CachePath, conf.GeoLocation.Token); errUpdate != nil {
				slog.Error("Failed to update", log.ErrAttr(errUpdate))

				return
			}

			slog.Info("Reading data")

			blockListData, errRead := ip2location.Read(conf.GeoLocation.CachePath)
			if errRead != nil {
				slog.Error("Failed to read data", log.ErrAttr(errRead))

				return
			}

			updateCtx, cancelUpdate := context.WithTimeout(ctx, time.Minute*30)
			defer cancelUpdate()

			slog.Info("Starting import")

			networkUsecase := network.NewNetworkUsecase(
				eventBroadcaster,
				network.NewNetworkRepository(dbUsecase),
				personUsecase)

			if errInsert := networkUsecase.InsertIP2LocationData(updateCtx, blockListData); errInsert != nil {
				slog.Error("Failed to import", log.ErrAttr(errInsert))

				return
			}

			slog.Info("Import Complete")

			os.Exit(0)
		},
	}
}
