package cmd

import (
	"errors"
	"log/slog"

	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/spf13/cobra"
)

func refreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Data maintenance tasks",
	}
}

func netUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "location",
		Short: "Updates ip2location dataset",
		Long:  `Updates ip2location dataset`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			app, errApp := New()
			if errApp != nil {
				return errApp
			}

			defer func() {
				if errClose := app.Shutdown(ctx); errClose != nil {
					slog.Error("Error closing", slog.String("error", errClose.Error()))
				}
			}()

			if errSetup := app.Init(ctx); errSetup != nil {
				return errSetup
			}

			if err := app.networks.RefreshLocationData(ctx); err != nil {
				return err
			}

			return nil
		},
	}
}

var errRefresh = errors.New("refresh error")

// refreshSteamIDs handles trying to fix some potential legacy data validation errors.
func refreshSteamIDs() *cobra.Command {
	return &cobra.Command{
		Use:   "steamid",
		Short: "Fix some potential steamid data problems",
		Long:  "Fix some potential steamid data problems. Updates invalid steamids to the steam_id configured as the owner",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			app, errApp := New()
			if errApp != nil {
				return errApp
			}

			defer func() {
				if errClose := app.Shutdown(ctx); errClose != nil {
					slog.Error("Error closing", slog.String("error", errClose.Error()))
				}
			}()

			if errSetup := app.Init(ctx); errSetup != nil {
				return errSetup
			}

			rows, err := app.database.Query(ctx, "SELECT steam_id from person")
			if err != nil {
				return err
			}
			defer rows.Close()

			owner := steamid.New(app.staticConfig.Owner)

			for rows.Next() {
				var sid64 int64
				if err := rows.Scan(&sid64); err != nil {
					return errors.Join(err, errRefresh)
				}

				sid := steamid.New(sid64)
				if !sid.Valid() {
					slog.Error("Found bad sid! Assigning to owner instead", slog.String("steam_id", sid.String()), slog.String("owner", owner.String()))
					if errExec := app.database.Exec(ctx, "UPDATE vote_result SET source_id = $1 WHERE source_id = $2", owner.Int64(), sid64); errExec != nil {
						return errors.Join(errExec, errRefresh)
					}
					if errExec := app.database.Exec(ctx, "UPDATE vote_result SET target_id = $1 WHERE target_id = $2", owner.Int64(), sid64); errExec != nil {
						return errors.Join(errExec, errRefresh)
					}
					if errExec := app.database.Exec(ctx, "UPDATE ban SET source_id = $1 WHERE source_id = $2", owner.Int64(), sid64); errExec != nil {
						return errors.Join(errExec, errRefresh)
					}
					if errExec := app.database.Exec(ctx, "UPDATE ban SET target_id = $1 WHERE target_id = $2", owner.Int64(), sid64); errExec != nil {
						return errors.Join(errExec, errRefresh)
					}
					if errExec := app.database.Exec(ctx, "UPDATE asset SET author_id = $1 WHERE author_id = $2", owner.Int64(), sid64); errExec != nil {
						return errors.Join(errExec, errRefresh)
					}
					if errExec := app.database.Exec(ctx, "DELETE FROM person WHERE steam_id = $1", sid64); errExec != nil {
						return errors.Join(errExec, errRefresh)
					}
				}
			}

			return nil
		},
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
				if errClose := app.Shutdown(ctx); errClose != nil {
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
				messages, _, errMessages := app.chat.QueryChatHistory(ctx, admin.PermissionLevel, query)
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
