package app

import (
	"context"

	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// activeMatchContext represents the current match on any given server instance.
type activeMatchContext struct {
	match          logparse.Match
	cancel         context.CancelFunc
	incomingEvents chan logparse.ServerEvent
	log            *zap.Logger
	finalScores    int
}

func (am *activeMatchContext) start(ctx context.Context) {
	am.log.Info("Match started", zap.String("match_id", am.match.MatchID.String()))

	for {
		select {
		case evt := <-am.incomingEvents:
			if errApply := am.match.Apply(evt.Results); errApply != nil && !errors.Is(errApply, logparse.ErrIgnored) {
				am.log.Error("Error applying event",
					zap.String("server", evt.ServerName),
					zap.Error(errApply))
			}
		case <-ctx.Done():
			am.log.Info("Match Closed", zap.String("match_id", am.match.MatchID.String()))

			return
		}
	}
}
