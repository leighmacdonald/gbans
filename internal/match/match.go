package match

import (
	"context"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	ErrInsufficientPlayers = errors.New("Insufficient Match players")
	ErrIncompleteMatch     = errors.New("Insufficient match data")
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

type OnCompleteFunc func(ctx context.Context, matchContext *Context) error

func NewSummarizer(log *zap.Logger, broadcaster *fp.Broadcaster[logparse.EventType, logparse.ServerEvent],
	matchUUIDMap fp.MutexMap[int, uuid.UUID], completeFunc OnCompleteFunc,
) *Summarizer {
	return &Summarizer{
		log:             log,
		events:          make(chan logparse.ServerEvent),
		broadcaster:     broadcaster,
		matchUUIDMap:    matchUUIDMap,
		onMatchComplete: completeFunc,
	}
}

// Summarizer is the central collection point for summarizing matches live from UDP log events.
type Summarizer struct {
	log             *zap.Logger
	events          chan logparse.ServerEvent
	broadcaster     *fp.Broadcaster[logparse.EventType, logparse.ServerEvent]
	matchUUIDMap    fp.MutexMap[int, uuid.UUID]
	onMatchComplete OnCompleteFunc
}

func (ms *Summarizer) Start(ctx context.Context) {
	log := ms.log.Named("matchSum")

	eventChan := make(chan logparse.ServerEvent)
	if errReg := ms.broadcaster.Consume(eventChan); errReg != nil {
		log.Error("logWriter Tried to register duplicate reader channel", zap.Error(errReg))
	}

	matches := map[int]*Context{}

	for {
		select {
		case evt := <-eventChan:
			matchContext, exists := matches[evt.ServerID]

			if !exists {
				cancelCtx, cancel := context.WithCancel(ctx)
				matchContext = &Context{
					Match:          logparse.NewMatch(evt.ServerID, evt.ServerName),
					cancel:         cancel,
					log:            log.Named(evt.ServerName),
					incomingEvents: make(chan logparse.ServerEvent),
					stopChan:       make(chan bool),
				}

				go matchContext.start(cancelCtx)

				ms.matchUUIDMap.Set(evt.ServerID, matchContext.Match.MatchID)

				matches[evt.ServerID] = matchContext
			}

			matchContext.incomingEvents <- evt

			switch evt.EventType {
			case logparse.WTeamFinalScore:
				matchContext.finalScores++
				if matchContext.finalScores < 2 {
					continue
				}

				fallthrough
			case logparse.LogStop:
				matchContext.stopChan <- true

				if err := ms.onMatchComplete(ctx, matchContext); err != nil {
					switch {
					case errors.Is(err, ErrInsufficientPlayers):
						ms.log.Warn("Insufficient data to save")
					case errors.Is(err, ErrIncompleteMatch):
						ms.log.Warn("Incomplete match, ignoring")
					default:
						ms.log.Error("Failed to save Match results", zap.Error(err))
					}
				}

				delete(matches, evt.ServerID)
			}
		case <-ctx.Done():
			return
		}
	}
}
