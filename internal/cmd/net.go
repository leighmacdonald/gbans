package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/pkg/fp"
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

			dbConn := database.New(staticConfig.DatabaseDSN, staticConfig.DatabaseAutoMigrate, staticConfig.DatabaseLogQueries)
			if errConnect := dbConn.Connect(ctx); errConnect != nil {
				slog.Error("Cannot initialize database", log.ErrAttr(errConnect))

				return
			}

			defer func() {
				if errClose := dbConn.Close(); errClose != nil {
					slog.Error("Failed to close database cleanly", log.ErrAttr(errClose))
				}
			}()

			configuration := config.NewConfiguration(staticConfig, config.NewRepository(dbConn))

			if err := configuration.Init(ctx); err != nil {
				panic(fmt.Sprintf("Failed to init config: %v", err))
			}

			if errConfig := configuration.Reload(ctx); errConfig != nil {
				panic(fmt.Sprintf("Failed to read config: %v", errConfig))
			}

			conf := configuration.Config()
			logCloser := log.MustCreateLogger(ctx, conf.Log.File, conf.Log.Level, SentryDSN != "", BuildVersion)
			defer logCloser()

			persons := person.NewPersons(person.NewRepository(configuration.Config(), dbConn), configuration, nil)

			eventBroadcaster := fp.NewBroadcaster[logparse.EventType, logparse.ServerEvent]()
			networks := network.NewNetworks(eventBroadcaster, network.NewRepository(dbConn, persons), configuration)

			if err := networks.RefreshLocationData(ctx); err != nil {
				slog.Error("Failed to refresh location data", log.ErrAttr(err))
				os.Exit(1)
			}
		},
	}
}
