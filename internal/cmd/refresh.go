package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/blocklist"
	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/log"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/report"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/steamgroup"
	"github.com/leighmacdonald/gbans/internal/wordfilter"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
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
		Run: func(cmd *cobra.Command, args []string) {
			configUsecase := config.NewConfigUsecase(config.NewConfigRepository())
			if errConfig := configUsecase.Read(false); errConfig != nil {
				panic(fmt.Sprintf("Failed to read config: %v", errConfig))
			}

			conf := configUsecase.Config()
			rootLogger := log.MustCreate(conf, nil)
			defer func() {
				_ = rootLogger.Sync()
			}()

			defer func() {
				_ = rootLogger.Sync()
			}()

			ctx := context.Background()

			connCtx, cancelConn := context.WithTimeout(ctx, time.Second*5)
			defer cancelConn()
			dbUsecase := database.New(rootLogger, conf.DB.DSN, false, conf.DB.LogQueries)

			rootLogger.Info("Connecting to database")
			if errConnect := dbUsecase.Connect(connCtx); errConnect != nil {
				rootLogger.Fatal("Failed to connect to database", zap.Error(errConnect))
			}
			defer func() {
				if errClose := dbUsecase.Close(); errClose != nil {
					rootLogger.Error("Failed to close database cleanly", zap.Error(errClose))
				}
			}()

			if //goland:noinspection ALL
			errDelete := dbUsecase.Exec(ctx, "DELETE FROM person_messages_filter"); errDelete != nil {
				rootLogger.Fatal("Failed to delete existing", zap.Error(errDelete))
			}

			eventBroadcaster := fp.NewBroadcaster[logparse.EventType, logparse.ServerEvent]()

			serversUsecase := servers.NewServersUsecase(servers.NewServersRepository(dbUsecase))

			stateRepository := state.NewStateRepository(state.NewCollector(rootLogger, serversUsecase))
			stateUsecase := state.NewStateUsecase(rootLogger, eventBroadcaster, stateRepository, configUsecase, serversUsecase)

			discordRepository, _ := discord.NewDiscordRepository(rootLogger, conf)

			discordUsecase := discord.NewDiscordUsecase(discordRepository)

			reportUsecase := report.NewReportUsecase(rootLogger, report.NewReportRepository(dbUsecase), discordUsecase, configUsecase)

			personUsecase := person.NewPersonUsecase(rootLogger, person.NewPersonRepository(dbUsecase))

			wordFilterUsecase := wordfilter.NewWordFilterUsecase(wordfilter.NewWordFilterRepository(dbUsecase), discordUsecase)
			if errImport := wordFilterUsecase.Import(ctx); errImport != nil {
				rootLogger.Fatal("Failed to load filters")
			}

			blocklistUsecase := blocklist.NewBlocklistUsecase(blocklist.NewBlocklistRepository(dbUsecase))
			networkUsecase := network.NewNetworkUsecase(rootLogger, eventBroadcaster, network.NewNetworkRepository(dbUsecase), blocklistUsecase, personUsecase)
			banRepository := ban.NewBanSteamRepository(dbUsecase, personUsecase, networkUsecase)
			banGroupUsecase := steamgroup.NewBanGroupUsecase(rootLogger, steamgroup.NewSteamGroupRepository(dbUsecase))
			banUsecase := ban.NewBanSteamUsecase(rootLogger, banRepository, personUsecase, configUsecase, discordUsecase, banGroupUsecase, reportUsecase, stateUsecase)
			chatRepository := chat.NewChatRepository(dbUsecase, rootLogger, personUsecase, wordFilterUsecase, eventBroadcaster)
			chatUsecase := chat.NewChatUsecase(rootLogger, configUsecase, chatRepository, wordFilterUsecase, stateUsecase, banUsecase,
				personUsecase, discordUsecase)

			var query domain.ChatHistoryQueryFilter
			query.DontCalcTotal = true
			query.OrderBy = "created_on"
			query.Desc = false
			query.Limit = 10000
			query.Unrestricted = true

			matches := 0

			for {
				messages, _, errMessages := chatUsecase.QueryChatHistory(ctx, query)
				if errMessages != nil {
					rootLogger.Error("Failed to load more messages", zap.Error(errMessages))

					break
				}

				for _, message := range messages {
					matched := wordFilterUsecase.Check(message.Body)
					if len(matched) > 0 {
						if errAdd := wordFilterUsecase.AddMessageFilterMatch(ctx, message.PersonMessageID, matched[0].FilterID); errAdd != nil {
							rootLogger.Error("Failed to add filter match", zap.Error(errAdd))
						}

						matches++

						break
					}
				}

				query.Offset += query.Limit

				if query.Offset%(query.Offset*5) == 0 {
					rootLogger.Info("Progress update", zap.Uint64("offset", query.Offset), zap.Int("matches", matches))
				}
			}

			rootLogger.Info("Refresh Complete")

			os.Exit(0)
		},
	}
}
