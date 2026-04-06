package servers

import (
	"context"
	"log/slog"
	"time"

	"github.com/leighmacdonald/gbans/pkg/logparse"
)

func newLogEventRecorder(repo Repository) *LogEventRecorder {
	return &LogEventRecorder{repo: repo, C: make(chan logparse.ServerEvent, 100)}
}

type LogEventRecorder struct {
	repo Repository
	C    chan logparse.ServerEvent
	logs []logparse.ServerEvent
}

func (l *LogEventRecorder) start(ctx context.Context) {
	writeTicker := time.NewTicker(time.Second * 10)
	purgeTicker := time.NewTicker(time.Hour * 24)

	for {
		select {
		case event := <-l.C:
			l.logs = append(l.logs, event)
			slog.Debug(event.Raw)
		case <-writeTicker.C:
			l.flush(ctx)
			l.logs = nil
		case <-purgeTicker.C:
			if err := l.repo.purgeLogs(ctx); err != nil {
				slog.Error("Failed to purge server logs", slog.String("error", err.Error()))
			}
		case <-ctx.Done():
			l.flush(ctx)

			return
		}
	}
}

func (l *LogEventRecorder) send(event logparse.ServerEvent) {
	select {
	case l.C <- event:
	default:
		slog.Debug("Dropped log message", slog.String("reason", "queue_full"))
	}
}

func (l *LogEventRecorder) flush(ctx context.Context) {
	if len(l.logs) == 0 {
		return
	}
	if err := l.repo.InsertLogs(ctx, l.logs); err != nil {
		slog.Error("Failed to flush server logs", slog.String("error", err.Error()))
	}
}
