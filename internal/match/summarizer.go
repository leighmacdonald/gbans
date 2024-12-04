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

// activeMatchContext controls and represents the broader match context including extra metadata.
type activeMatchContext struct {
	match       logparse.Match
	cancel      context.CancelFunc
	finalScores int
	server      domain.Server
}

type OnCompleteFn func(ctx context.Context, m *activeMatchContext) error

type Summarizer struct {
	uuidMap    fp.MutexMap[int, uuid.UUID]
	triggers   chan domain.MatchTrigger
	log        *slog.Logger
	eventChan  chan logparse.ServerEvent
	onComplete OnCompleteFn
}

func newMatchSummarizer(eventChan chan logparse.ServerEvent, onComplete OnCompleteFn) *Summarizer {
	return &Summarizer{
		uuidMap:    fp.NewMutexMap[int, uuid.UUID](),
		triggers:   make(chan domain.MatchTrigger),
		eventChan:  eventChan,
		onComplete: onComplete,
	}
}

func parseMapName(name string) string {
	if strings.HasPrefix(name, "workshop/") {
		parts := strings.Split(strings.TrimPrefix(name, "workshop/"), ".ugc")
		name = parts[0]
	}

	return name
}

func (mh *Summarizer) Start(ctx context.Context) {
	matches := map[int]*activeMatchContext{}

	for {
		select {
		case trigger := <-mh.triggers:
			switch trigger.Type {
			case domain.MatchTriggerStart:
				match := logparse.NewMatch(trigger.Server.ServerID, trigger.Server.Name)
				match.MapName = parseMapName(trigger.MapName)
				match.DemoName = trigger.DemoName

				matchContext := &activeMatchContext{
					match:  match,
					server: trigger.Server,
				}

				mh.uuidMap.Set(trigger.Server.ServerID, trigger.UUID)

				matches[trigger.Server.ServerID] = matchContext
			case domain.MatchTriggerEnd:
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

			if errApply := matchContext.match.Apply(evt.Results); errApply != nil && !errors.Is(errApply, logparse.ErrIgnored) {
				slog.Error("Error applying event",
					slog.String("server", evt.ServerName),
					log.ErrAttr(errApply))
			}

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
