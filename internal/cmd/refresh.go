package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/report"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/wordfilter"
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

			if //goland:noinspection ALL
			errDelete := dbUsecase.Exec(ctx, "DELETE FROM person_messages_filter"); errDelete != nil {
				slog.Error("Failed to delete existing", log.ErrAttr(errDelete))

				return
			}

			eventBroadcaster := fp.NewBroadcaster[logparse.EventType, logparse.ServerEvent]()

			serversUsecase := servers.NewServersUsecase(servers.NewServersRepository(dbUsecase))

			stateUsecase := state.NewStateUsecase(eventBroadcaster,
				state.NewStateRepository(state.NewCollector(serversUsecase)), configUsecase, serversUsecase)

			wordFilterUsecase := wordfilter.NewWordFilterUsecase(wordfilter.NewWordFilterRepository(dbUsecase))
			if errImport := wordFilterUsecase.Import(ctx); errImport != nil {
				slog.Error("Failed to load filters")
			}

			discordRepository, _ := discord.NewDiscordRepository(conf)

			discordUsecase := discord.NewDiscordUsecase(discordRepository, wordFilterUsecase)

			personUsecase := person.NewPersonUsecase(person.NewPersonRepository(dbUsecase), configUsecase)
			reportUsecase := report.NewReportUsecase(report.NewReportRepository(dbUsecase), discordUsecase, configUsecase, personUsecase, nil)
			// banGroupUsecase := steamgroup.NewBanGroupUsecase(steamgroup.NewSteamGroupRepository(dbUsecase), personUsecase)
			networkUsecase := network.NewNetworkUsecase(eventBroadcaster, network.NewNetworkRepository(dbUsecase), personUsecase)
			banUsecase := ban.NewBanSteamUsecase(ban.NewBanSteamRepository(dbUsecase, personUsecase, networkUsecase), personUsecase, configUsecase, discordUsecase, reportUsecase, stateUsecase)

			// blocklistUsecase := blocklist.NewBlocklistUsecase(blocklist.NewBlocklistRepository(dbUsecase), banUsecase)

			chatRepository := chat.NewChatRepository(dbUsecase, personUsecase, wordFilterUsecase, nil, eventBroadcaster)
			chatUsecase := chat.NewChatUsecase(configUsecase, chatRepository, wordFilterUsecase, stateUsecase, banUsecase,
				personUsecase, discordUsecase)

			var query domain.ChatHistoryQueryFilter
			query.DontCalcTotal = true
			query.OrderBy = "created_on"
			query.Desc = false
			query.Limit = 10000
			query.Unrestricted = true

			matches := 0

			admin, errAdmin := personUsecase.GetPersonBySteamID(ctx, steamid.New(conf.Owner))
			if errAdmin != nil {
				slog.Error("Failed to load admin user", log.ErrAttr(errAdmin))

				return
			}

			for {
				messages, errMessages := chatUsecase.QueryChatHistory(ctx, admin, query)
				if errMessages != nil {
					slog.Error("Failed to load more messages", log.ErrAttr(errMessages))

					break
				}

				for _, message := range messages {
					matched := wordFilterUsecase.Check(message.Body)
					if len(matched) > 0 {
						if errAdd := wordFilterUsecase.AddMessageFilterMatch(ctx, message.PersonMessageID, matched[0].FilterID); errAdd != nil {
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
