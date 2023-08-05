package cmd

import (
	"context"
	"os"
	"time"

	"github.com/leighmacdonald/gbans/internal/app"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/store"
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
			var conf app.Config
			if errConfig := app.ReadConfig(&conf, false); errConfig != nil {
				panic("Failed to read config")
			}
			conf.Log.Level = "DEBUG"
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

			if errDelete := database.Exec(ctx, "DELETE FROM person_messages_filter"); errDelete != nil {
				rootLogger.Fatal("Failed to delete existing", zap.Error(errDelete))
			}

			bot, errBot := discord.New(rootLogger, conf.Discord.Token,
				conf.Discord.AppID, conf.Discord.UnregisterOnStart, conf.General.ExternalURL)
			if errBot != nil {
				rootLogger.Fatal("Failed to connect to perform initial discord connection")
			}

			application := app.New(&conf, database, bot, rootLogger)
			if errFilters := application.LoadFilters(ctx); errFilters != nil {
				rootLogger.Fatal("Failed to load filters", zap.Error(errFilters))
			}

			var query store.ChatHistoryQueryFilter
			query.DontCalcTotal = true
			query.OrderBy = "created_on"
			query.Desc = false
			query.Limit = 10000
			query.Unrestricted = true

			matches := 0

			for {
				messages, _, errMessages := database.QueryChatHistory(ctx, query)
				if errMessages != nil {
					rootLogger.Error("Failed to load more messages", zap.Error(errMessages))

					break
				}

				for _, message := range messages {
					matched := application.FilterCheck(message.Body)
					if len(matched) > 0 {
						if errAdd := database.AddMessageFilterMatch(ctx, message.PersonMessageID, matched[0].FilterID); errAdd != nil {
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
