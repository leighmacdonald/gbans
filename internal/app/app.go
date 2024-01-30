package app

import (
	"github.com/leighmacdonald/gbans/internal/domain"
)

var (
	BuildVersion = "master" //nolint:gochecknoglobals
	BuildCommit  = ""       //nolint:gochecknoglobals
	BuildDate    = ""       //nolint:gochecknoglobals
)

func Version() domain.BuildInfo {
	return domain.BuildInfo{
		BuildVersion: BuildVersion,
		Commit:       BuildCommit,
		Date:         BuildDate,
	}
}

//
//func (app *App) startWorkers(ctx context.Context) {
//	go app.patreon.Start(ctx)
//	go app.banSweeper(ctx)
//	go app.profileUpdater(ctx)
//	go app.warningTracker.Start(ctx)
//	go app.logReader(ctx, app.Config().Debug.WriteUnhandledLogEvents)
//	go app.initLogSrc(ctx)
//	go metrics.logMetricsConsumer(ctx, app.mc, app.eb, app.log)
//	go app.matchSummarizer.Start(ctx)
//	go app.chatLogger.start(ctx)
//	go app.playerConnectionWriter(ctx)
//	go app.steamGroups.Start(ctx)
//	go cleanupTasks(ctx, app.db, app.log)
//	go app.showReportMeta(ctx)
//	go app.notificationSender(ctx)
//	go app.demoCleaner(ctx)
//	go app.state.Start(ctx, func() config.Config {
//		return app.Config()
//	}, func() state.ServerStore {
//		return app.Store()
//	})
//	go app.activityTracker.Start(ctx)
//	go app.steamFriends.Start(ctx)
//}
