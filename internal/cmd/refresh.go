package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gofrs/uuid/v5"
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
			cu := config.NewConfigUsecase(config.NewConfigRepository())
			if errConfig := cu.Read(false); errConfig != nil {
				panic(fmt.Sprintf("Failed to read config: %v", errConfig))
			}

			conf := cu.Config()
			conf.Log.Level = "DEBUG"
			rootLogger := log.MustCreate(&conf, nil)
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

			if errDelete := db.Exec(ctx, "DELETE FROM person_messages_filter"); errDelete != nil {
				rootLogger.Fatal("Failed to delete existing", zap.Error(errDelete))
			}

			eventBroadcaster := fp.NewBroadcaster[logparse.EventType, logparse.ServerEvent]()
			matchUUIDMap := fp.NewMutexMap[int, uuid.UUID]()

			sv := servers.NewServersUsecase(servers.NewServersRepository(db))
			st := state.NewStateUsecase(rootLogger, state.NewStateRepository(state.NewCollector(rootLogger, sv)))
			ru := report.NewReportUsecase(report.NewReportRepository(db))

			dr, _ := discord.NewDiscordRepository(rootLogger, conf)

			du := discord.NewDiscordUsecase(dr)

			pu := person.NewPersonUsecase(person.NewPersonRepository(db))
			wfu := wordfilter.NewWordFilterUsecase(wordfilter.NewWordFilterRepository(db), du)
			wfu.Import(ctx)
			blu := blocklist.NewBlocklistUsecase(blocklist.NewBlocklistRepository(db))
			nu := network.NewNetworkUsecase(rootLogger, network.NewNetworkRepository(db), blu)
			br := ban.NewBanRepository(db, pu, nu)
			sgu := steamgroup.NewBanGroupUsecase(rootLogger, steamgroup.NewSteamGroupRepository(db))
			bu := ban.NewBanUsecase(rootLogger, br, pu, cu, du, sgu, ru, st)
			cr := chat.NewChatRepository(db, rootLogger, pu, wfu, eventBroadcaster, matchUUIDMap)
			chu := chat.NewChatUsecase(rootLogger, cu, cr, wfu, st, bu, pu, du, st)

			var query domain.ChatHistoryQueryFilter
			query.DontCalcTotal = true
			query.OrderBy = "created_on"
			query.Desc = false
			query.Limit = 10000
			query.Unrestricted = true

			matches := 0

			for {
				messages, _, errMessages := chu.QueryChatHistory(ctx, query)
				if errMessages != nil {
					rootLogger.Error("Failed to load more messages", zap.Error(errMessages))

					break
				}

				for _, message := range messages {
					matched := wfu.Check(message.Body)
					if len(matched) > 0 {
						if errAdd := wfu.AddMessageFilterMatch(ctx, message.PersonMessageID, matched[0].FilterID); errAdd != nil {
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
