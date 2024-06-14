package votes

import (
	"context"
	"log/slog"
	"time"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/logparse"
)

type voteUsecase struct {
	repository domain.VoteRepository
	persons    domain.PersonUsecase
	matches    domain.MatchUsecase
	discord    domain.DiscordUsecase
	config     domain.ConfigUsecase

	broadcaster *fp.Broadcaster[logparse.EventType, logparse.ServerEvent]
}

func NewVoteUsecase(repository domain.VoteRepository, persons domain.PersonUsecase, matched domain.MatchUsecase,
	discord domain.DiscordUsecase, config domain.ConfigUsecase, broadcaster *fp.Broadcaster[logparse.EventType, logparse.ServerEvent],
) domain.VoteUsecase {
	return &voteUsecase{
		repository:  repository,
		persons:     persons,
		matches:     matched,
		discord:     discord,
		config:      config,
		broadcaster: broadcaster,
	}
}

func (u voteUsecase) Query(ctx context.Context, filter domain.VoteQueryFilter) ([]domain.VoteResult, int64, error) {
	return u.repository.Query(ctx, filter)
}

// Start will begin ingesting vote events and record them to the database.
func (u voteUsecase) Start(ctx context.Context) {
	type voteState struct {
		name    string
		success bool
		code    logparse.VoteCode
	}

	eventChan := make(chan logparse.ServerEvent)
	if errRegister := u.broadcaster.Consume(eventChan, logparse.VoteSuccess, logparse.VoteFailed, logparse.VoteDetails); errRegister != nil {
		slog.Warn("logWriter Tried to register duplicate reader channel", log.ErrAttr(errRegister))

		return
	}

	// Track recent votes and reject duplicates. Sometimes vote results get logged twice
	var recent []domain.VoteResult

	active := map[int]voteState{}

	cleanupTimer := time.NewTicker(time.Second * 5)

	for {
		select {
		case <-ctx.Done():
			return
		case <-cleanupTimer.C:
			// Cleanup timed out results
			var valid []domain.VoteResult

			for _, result := range recent {
				if time.Since(result.CreatedOn) > time.Second*20 {
					continue
				}

				valid = append(valid, result)
			}

			recent = valid
		case evt := <-eventChan:
			switch evt.EventType {
			case logparse.VoteSuccess:
				successEvt, ok := evt.Event.(logparse.VoteSuccessEvt)
				if !ok {
					continue
				}

				active[evt.ServerID] = voteState{
					name:    successEvt.Name,
					success: true,
				}
			case logparse.VoteFailed:
				failEvt, ok := evt.Event.(logparse.VoteFailEvt)
				if !ok {
					continue
				}

				active[evt.ServerID] = voteState{
					name:    failEvt.Name,
					success: false,
					code:    failEvt.Code,
				}
			case logparse.VoteDetails:
				serverEvent, validEvent := evt.Event.(logparse.VoteKickDetailsEvt)
				if !validEvent {
					delete(active, evt.ServerID)

					continue
				}

				// matchID, _ := u.matchUsecase.GetMatchIDFromServerID(evt.ServerID)

				currentState, validState := active[evt.ServerID]
				if !validState {
					// Sometimes this event doesn't fire? Add defaults
					currentState = voteState{
						name:    serverEvent.Name,
						success: false,
						code:    0,
					}
				}

				result := domain.VoteResult{
					ServerID:  evt.ServerID,
					SourceID:  serverEvent.SID,
					TargetID:  serverEvent.SID2,
					Success:   currentState.success,
					Name:      currentState.name,
					Code:      currentState.code,
					CreatedOn: time.Now(),
				}

				// Vote results sometimes get sent twice
				skip := false

				for _, existing := range recent {
					if result.ServerID == existing.ServerID &&
						result.SourceID == existing.SourceID &&
						result.TargetID == existing.TargetID &&
						result.Success == existing.Success {
						skip = true

						break
					}
				}

				if skip {
					delete(active, evt.ServerID)
					slog.Warn("Skipped duplicate result")

					continue
				}

				if err := u.repository.AddResult(ctx, result); err != nil {
					slog.Error("Failed to add vote result", log.ErrAttr(err))
				}

				recent = append(recent, result)

				delete(active, evt.ServerID)

				source, errSource := u.persons.GetOrCreatePersonBySteamID(ctx, result.SourceID)
				if errSource != nil {
					slog.Error("Failed to load vote source", log.ErrAttr(errSource), slog.String("steam_id", result.SourceID.String()))
				}

				target, errTarget := u.persons.GetOrCreatePersonBySteamID(ctx, result.SourceID)
				if errTarget != nil {
					slog.Error("Failed to load vote target", log.ErrAttr(errSource), slog.String("steam_id", result.TargetID.String()))
				}

				u.discord.SendPayload(domain.ChannelModVoteLog, discord.VoteResultMessage(u.config.Config(), result, source, target))

			}
		}
	}
}
