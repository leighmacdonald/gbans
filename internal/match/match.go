package match

import (
	"context"
	"errors"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"go.uber.org/zap"
)

// Context represents the current Match on any given server instance.
type Context struct {
	Match          logparse.Match
	cancel         context.CancelFunc
	incomingEvents chan logparse.ServerEvent
	log            *zap.Logger
	finalScores    int
	stopChan       chan bool
}

func (am *Context) start(ctx context.Context) {
	am.log.Info("Match started", zap.String("match_id", am.Match.MatchID.String()))

	for {
		select {
		case evt := <-am.incomingEvents:
			if errApply := am.Match.Apply(evt.Results); errApply != nil && !errors.Is(errApply, logparse.ErrIgnored) {
				am.log.Error("Error applying event",
					zap.String("server", evt.ServerName),
					zap.Error(errApply))
			}
		case <-am.stopChan:
			am.log.Info("Match Stopped", zap.String("match_id", am.Match.MatchID.String()))

			return
		case <-ctx.Done():
			am.log.Info("Match Cancelled", zap.String("match_id", am.Match.MatchID.String()))

			return
		}
	}
}
