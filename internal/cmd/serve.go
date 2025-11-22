package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"
)

// func createQueueWorkers(people person.PersonUsecase, notifications notification.NotificationPayload,
// 	discordUC *discord.Discord, authRepo auth.AuthRepository,
// 	patreonUC patreon.PatreonCredential, reports ban.ReportUsecase, discordOAuth discordoauth.DiscordOAuthUsecase,
// ) *river.Workers {
// 	workers := river.NewWorkers()

// 	river.AddWorker[notification.SenderArgs](workers, notification.NewSenderWorker(people, notifications, discordUC))
// 	river.AddWorker[auth.CleanupArgs](workers, auth.NewCleanupWorker(authRepo))
// 	river.AddWorker[patreon.AuthUpdateArgs](workers, patreon.NewSyncWorker(patreonUC))
// 	river.AddWorker[ban.MetaInfoArgs](workers, ban.NewMetaInfoWorker(reports))
// 	river.AddWorker[discord.TokenRefreshArgs](workers, discord.NewTokenRefreshWorker(discordOAuth))

// 	return workers
// }

// serveCmd represents the serve command.
func serveCmd() *cobra.Command { //nolint:maintidx
	return &cobra.Command{
		Use:   "serve",
		Short: "Starts the gbans web app",
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

			go app.StartBackground(ctx)

			if errServe := app.Serve(ctx); errServe != nil {
				return errServe
			}

			return nil
		},
	}
}
