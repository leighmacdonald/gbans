package match

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/logparse"
)

// Context represents the current Match on any given server instance.
type Context struct {
	Match          logparse.Match
	cancel         context.CancelFunc
	incomingEvents chan logparse.ServerEvent
	finalScores    int
	stopChan       chan bool
}

func (am *Context) start(ctx context.Context) {
	slog.Info("Match started", slog.String("match_id", am.Match.MatchID.String()))

	for {
		select {
		case evt := <-am.incomingEvents:
			if errApply := am.Match.Apply(evt.Results); errApply != nil && !errors.Is(errApply, logparse.ErrIgnored) {
				slog.Error("Error applying event",
					slog.String("server", evt.ServerName),
					log.ErrAttr(errApply))
			}
		case <-am.stopChan:
			slog.Info("Match Stopped", slog.String("match_id", am.Match.MatchID.String()))

			return
		case <-ctx.Done():
			slog.Info("Match Cancelled", slog.String("match_id", am.Match.MatchID.String()))

			return
		}
	}
}

// activeMatchContext represents the current match on any given server instance.
type activeMatchContext struct {
	match          logparse.Match
	cancel         context.CancelFunc
	incomingEvents chan logparse.ServerEvent
	log            *slog.Logger
	finalScores    int
	server         domain.Server
}

func (am *activeMatchContext) start(ctx context.Context) {
	am.log.Info("Match started", slog.String("match_id", am.match.MatchID.String()))

	for {
		select {
		case evt := <-am.incomingEvents:
			if errApply := am.match.Apply(evt.Results); errApply != nil && !errors.Is(errApply, logparse.ErrIgnored) {
				am.log.Error("Error applying event",
					slog.String("server", evt.ServerName),
					log.ErrAttr(errApply))
			}
		case <-ctx.Done():
			am.log.Info("Match Closed", slog.String("match_id", am.match.MatchID.String()))

			return
		}
	}
}

type OnCompleteFn func(ctx context.Context, m *activeMatchContext) error

type Summarizer struct {
	uuidMap    fp.MutexMap[int, uuid.UUID]
	triggers   chan matchTrigger
	log        *slog.Logger
	eventChan  chan logparse.ServerEvent
	onComplete OnCompleteFn
}

func NewMatchSummarizer(eventChan chan logparse.ServerEvent, onComplete OnCompleteFn) *Summarizer {
	return &Summarizer{
		log:        slog.With("matchSum"),
		uuidMap:    fp.NewMutexMap[int, uuid.UUID](),
		triggers:   make(chan matchTrigger),
		eventChan:  eventChan,
		onComplete: onComplete,
	}
}

type matchTriggerType int

const (
	matchTriggerStart matchTriggerType = 1
	matchTriggerEnd   matchTriggerType = 2
)

type matchTrigger struct {
	Type     matchTriggerType
	UUID     uuid.UUID
	Server   domain.Server
	MapName  string
	DemoName string
}

func parseMapName(name string) string {
	if strings.HasPrefix(name, "workshop/") {
		parts := strings.Split(strings.TrimPrefix(name, "workshop/"), ".ugc")
		name = parts[0]
	}

	return name
}

// summarizer is the central collection point for summarizing matches live from UDP log events.
func (mh *Summarizer) summarizer(ctx context.Context) {
	matches := map[int]*activeMatchContext{}

	for {
		select {
		case trigger := <-mh.triggers:
			switch trigger.Type {
			case matchTriggerStart:
				cancelCtx, cancel := context.WithCancel(ctx)
				match := logparse.NewMatch(trigger.Server.ServerID, trigger.Server.Name)
				match.MapName = parseMapName(trigger.MapName)
				match.DemoName = trigger.DemoName

				matchContext := &activeMatchContext{
					match:          match,
					cancel:         cancel,
					log:            mh.log.With(slog.String("server", trigger.Server.ShortName)),
					incomingEvents: make(chan logparse.ServerEvent),
					server:         trigger.Server,
				}

				go matchContext.start(cancelCtx)

				mh.uuidMap.Set(trigger.Server.ServerID, trigger.UUID)

				matches[trigger.Server.ServerID] = matchContext
			case matchTriggerEnd:
				matchContext, exists := matches[trigger.Server.ServerID]
				if !exists {
					return
				}

				// Stop the incoming event handler
				matchContext.cancel()

				if matchContext.server.EnableStats {
					if errSave := mh.onComplete(ctx, matchContext); errSave != nil {
						mh.log.Error("Failed to save match data",
							slog.Int("server", matchContext.server.ServerID), log.ErrAttr(errSave))
					}
				}

				delete(matches, trigger.Server.ServerID)
			}
		case evt := <-mh.eventChan:
			matchContext, exists := matches[evt.ServerID]
			if !exists {
				// Discord any events w/o an existing match
				continue
			}

			matchContext.incomingEvents <- evt

			if evt.EventType == logparse.WTeamFinalScore {
				matchContext.finalScores++
				if matchContext.finalScores < 2 {
					continue
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
