package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/spf13/cobra"
)

func refreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "refresh",
		Long:  `refresh`,
	}
}

func refreshFiltersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "filters",
		Short: "refresh filters",
		Long:  `refresh filters`,
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

			if //goland:noinspection ALL
			errDelete := dbConn.Exec(ctx, nil, "DELETE FROM person_messages_filter"); errDelete != nil {
				slog.Error("Failed to delete existing", log.ErrAttr(errDelete))

				return
			}

			eventBroadcaster := fp.NewBroadcaster[logparse.EventType, logparse.ServerEvent]()
			serversCase := servers.NewServers(servers.NewServersRepository(dbConn))
			state := servers.NewState(eventBroadcaster, servers.NewStateRepository(servers.NewCollector(serversCase)), configuration, serversCase)
			wordFilters := chat.NewWordFilter(chat.NewWordFilterRepository(dbConn))
			if errImport := wordFilters.Import(ctx); errImport != nil {
				slog.Error("Failed to load filters")
			}

			tfapiClient, errClient := thirdparty.NewTFAPI("https://tf-api.roto.lol", &http.Client{Timeout: time.Second * 15})
			if errClient != nil {
				slog.Error("Failed to create tfapi client", slog.String("error", errClient.Error()))

				return
			}

			persons := person.NewPersons(person.NewRepository(conf, dbConn), configuration, tfapiClient)
			reports := ban.NewReports(ban.NewReportRepository(dbConn), configuration, persons, servers.Demos{}, tfapiClient)
			networks := network.NewNetworks(eventBroadcaster, network.NewRepository(dbConn, persons), configuration)
			bans := ban.NewBans(ban.NewBanRepository(dbConn, persons, networks), persons, configuration, reports, state, tfapiClient)

			// blocklistUsecase := blocklist.NewBlocklistUsecase(blocklist.NewBlocklistRepository(dbUsecase), banUsecase)

			chatRepo := chat.NewChatRepository(dbConn, persons, wordFilters, eventBroadcaster)
			chats := chat.NewChat(configuration, chatRepo, wordFilters, state, bans, persons)

			var query chat.ChatHistoryQueryFilter
			query.DontCalcTotal = true
			query.OrderBy = "created_on"
			query.Desc = false
			query.Limit = 10000
			query.Unrestricted = true

			matches := 0

			admin, errAdmin := persons.GetPersonBySteamID(ctx, nil, steamid.New(conf.Owner))
			if errAdmin != nil {
				slog.Error("Failed to load admin user", log.ErrAttr(errAdmin))

				return
			}

			for {
				messages, errMessages := chats.QueryChatHistory(ctx, admin, query)
				if errMessages != nil {
					slog.Error("Failed to load more messages", log.ErrAttr(errMessages))

					break
				}

				for _, message := range messages {
					matched := wordFilters.Check(message.Body)
					if len(matched) > 0 {
						if errAdd := wordFilters.AddMessageFilterMatch(ctx, message.PersonMessageID, matched[0].FilterID); errAdd != nil {
							slog.Error("Failed to add filter match", log.ErrAttr(errAdd))
						}

						matches++

						break
					}
				}

				query.Offset += query.Limit

				if query.Offset%(query.Offset*5) == 0 {
					slog.Info("Progress update", slog.Uint64("offset", query.Offset), slog.Int("matches", matches))
				}
			}

			slog.Info("Refresh Complete")

			os.Exit(0)
		},
	}
}
