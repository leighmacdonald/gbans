package app

import (
	"context"
	"errors"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/fp"
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

type OnCompleteFunc func(ctx context.Context, matchContext *Context) error

func NewSummarizer(logger *zap.Logger, broadcaster *fp.Broadcaster[logparse.EventType, logparse.ServerEvent], matchUUIDMap fp.MutexMap[int, uuid.UUID], onComplete OnCompleteFunc) *Summarizer {
	return &Summarizer{
		log:             logger,
		events:          make(chan logparse.ServerEvent),
		broadcaster:     broadcaster,
		matchUUIDMap:    matchUUIDMap,
		onMatchComplete: onComplete,
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

func onMatchComplete(env *App) OnCompleteFunc {
	return func(ctx context.Context, matchContext *Context) error {
		const minPlayers = 6

		currentState := env.State()
		server, found := currentState.ByServerID(matchContext.Match.ServerID)

		if found && server.Name != "" {
			matchContext.Match.Title = server.Name
		}

		var fullServer domain.Server
		if err := env.Store().GetServer(ctx, server.ServerID, &fullServer); err != nil {
			return errors.Join(err, errLoadServer)
		}

		if !fullServer.EnableStats {
			return nil
		}

		if len(matchContext.Match.PlayerSums) < minPlayers {
			return ErrInsufficientPlayers
		}

		if matchContext.Match.TimeStart == nil || matchContext.Match.MapName == "" {
			return ErrIncompleteMatch
		}

		if errSave := env.Store().MatchSave(ctx, &matchContext.Match, env.WeaponMap()); errSave != nil {
			if errors.Is(errSave, ErrInsufficientPlayers) {
				return ErrInsufficientPlayers
			} else {
				return errors.Join(errSave, errSaveMatch)
			}
		}

		var result domain.MatchResult
		if errResult := env.Store().MatchGetByID(ctx, matchContext.Match.MatchID, &result); errResult != nil {
			return errors.Join(errResult, errLoadMatch)
		}

		conf := env.Config()

		go env.SendPayload(conf.Discord.PublicMatchLogChannelID, discord.MatchMessage(result, ""))

		return nil
	}
}
