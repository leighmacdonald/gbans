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

// func createPeriodicJobs() []*river.PeriodicJob {
// 	jobs := []*river.PeriodicJob{
// 		river.NewPeriodicJob(
// 			river.PeriodicInterval(24*time.Hour),
// 			func() (river.JobArgs, *river.InsertOpts) {
// 				return auth.CleanupArgs{}, nil
// 			},
// 			&river.PeriodicJobOpts{RunOnStart: true}),

// 		river.NewPeriodicJob(
// 			river.PeriodicInterval(time.Hour),
// 			func() (river.JobArgs, *river.InsertOpts) {
// 				return patreon.AuthUpdateArgs{}, nil
// 			},
// 			&river.PeriodicJobOpts{RunOnStart: true}),

// 		river.NewPeriodicJob(
// 			river.PeriodicInterval(24*time.Hour),
// 			func() (river.JobArgs, *river.InsertOpts) {
// 				return report.MetaInfoArgs{}, nil
// 			},
// 			&river.PeriodicJobOpts{RunOnStart: true}),

// 		river.NewPeriodicJob(
// 			river.PeriodicInterval(time.Hour*12),
// 			func() (river.JobArgs, *river.InsertOpts) {
// 				return discord.TokenRefreshArgs{}, nil
// 			},
// 			&river.PeriodicJobOpts{RunOnStart: true}),
// 	}

// 	return jobs
// }

// serveCmd represents the serve command.
func serveCmd() *cobra.Command { //nolint:maintidx
	return &cobra.Command{
		Use:   "serve",
		Short: "Starts the gbans web app",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			app, errApp := NewGBans()
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

			app.StartBackground(ctx)

			// _, _, err := slur.Calc(ctx, &chat.MessageProvider{Db: app.database})
			// if err != nil {
			// 	return err
			// }

			if errServe := app.Serve(ctx); errServe != nil {
				return errServe
			}

			return nil
		},
	}
}
