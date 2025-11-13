package cmd

import (
	"log/slog"

	"github.com/leighmacdonald/gbans/internal/chat"
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			app, errApp := New()
			if errApp != nil {
				return errApp
			}

			defer func() {
				if errClose := app.Close(ctx); errClose != nil {
					slog.Error("Error closing", slog.String("error", errClose.Error()))
				}
			}()

			if errSetup := app.Init(ctx); errSetup != nil {
				return errSetup
			}

			var query chat.HistoryQueryFilter
			query.DontCalcTotal = true
			query.OrderBy = "created_on"
			query.Desc = false
			query.Limit = 10000
			query.Unrestricted = true

			matches := 0

			admin, errAdmin := app.persons.BySteamID(ctx, steamid.New(app.config.Config().Owner))
			if errAdmin != nil {
				return errAdmin
			}

			for {
				messages, errMessages := app.chat.QueryChatHistory(ctx, admin, query)
				if errMessages != nil {
					slog.Error("Failed to load more messages", slog.String("error", errMessages.Error()))

					break
				}

				for _, message := range messages {
					matched := app.wordFilters.Check(message.Body)
					if len(matched) > 0 {
						if errAdd := app.wordFilters.AddMessageFilterMatch(ctx, message.PersonMessageID, matched[0].FilterID); errAdd != nil {
							slog.Error("Failed to add filter match", slog.String("error", errAdd.Error()))
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

			return nil
		},
	}
}
