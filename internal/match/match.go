package match

import (
	"context"
	"errors"
	"log/slog"

	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/logparse"
)

// Context represents the current Match on any given server instance.
type Context struct {
	Match          logparse.Match
	cancel         context.CancelFunc
	incomingEvents chan logparse.ServerEvent
	log            *slog.Logger
	finalScores    int
	stopChan       chan bool
}

func (am *Context) start(ctx context.Context) {
	am.log.Info("Match started", slog.String("match_id", am.Match.MatchID.String()))

	for {
		select {
		case evt := <-am.incomingEvents:
			if errApply := am.Match.Apply(evt.Results); errApply != nil && !errors.Is(errApply, logparse.ErrIgnored) {
				am.log.Error("Error applying event",
					slog.String("server", evt.ServerName),
					log.ErrAttr(errApply))
			}
		case <-am.stopChan:
			am.log.Info("Match Stopped", slog.String("match_id", am.Match.MatchID.String()))

			return
		case <-ctx.Done():
			am.log.Info("Match Cancelled", slog.String("match_id", am.Match.MatchID.String()))

			return
		}
	}
}
