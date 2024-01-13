package app

import (
	"context"
	"strings"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"go.uber.org/zap"
)

type OnCompleteFn func(ctx context.Context, m *activeMatchContext) error

type MatchHandler struct {
	uuidMap    fp.MutexMap[int, uuid.UUID]
	triggers   chan matchTrigger
	log        *zap.Logger
	eventChan  chan logparse.ServerEvent
	onComplete OnCompleteFn
}

func NewMatchHandler(log *zap.Logger, eventChan chan logparse.ServerEvent, onComplete OnCompleteFn) *MatchHandler {
	return &MatchHandler{
		log:        log.Named("matchSum"),
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
	Server   store.Server
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
func (mh *MatchHandler) summarizer(ctx context.Context) {
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
					log:            mh.log.Named(trigger.Server.ShortName),
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
							zap.Int("server", matchContext.server.ServerID), zap.Error(errSave))
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
