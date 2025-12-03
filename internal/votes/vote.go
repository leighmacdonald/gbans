package votes

import (
	"context"
	"log/slog"
	"time"

	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/pkg/broadcaster"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type Query struct {
	query.Filter
	httphelper.SourceIDField
	httphelper.TargetIDField

	ServerID int    `json:"server_id"`
	Name     string `json:"name"`
	Success  int    `json:"success"` // -1 = any, 0 = false, 1 = true
	Code     bool   `json:"code"`
}

type Result struct {
	VoteID           int               `json:"vote_id"`
	SourceID         steamid.SteamID   `json:"source_id"`
	SourceName       string            `json:"source_name"`
	SourceAvatarHash string            `json:"source_avatar_hash"`
	TargetID         steamid.SteamID   `json:"target_id"`
	TargetName       string            `json:"target_name"`
	TargetAvatarHash string            `json:"target_avatar_hash"`
	Name             string            `json:"name"`
	Success          bool              `json:"success"`
	ServerID         int               `json:"server_id"`
	ServerName       string            `json:"server_name"`
	Code             logparse.VoteCode `json:"code"`
	CreatedOn        time.Time         `json:"created_on"`
}

type Votes struct {
	repository   Repository
	broadcaster  *broadcaster.Broadcaster[logparse.EventType, logparse.ServerEvent]
	notif        notification.Notifier
	logChannelID string
	persons      person.Provider
}

func New(repository Repository, broadcaster *broadcaster.Broadcaster[logparse.EventType, logparse.ServerEvent],
	notif notification.Notifier, logChannelID string, persons person.Provider,
) Votes {
	return Votes{
		repository:   repository,
		broadcaster:  broadcaster,
		notif:        notif,
		logChannelID: logChannelID,
		persons:      persons,
	}
}

func (u Votes) Add(ctx context.Context, sourceID steamid.SteamID, targetID steamid.SteamID, name string, success bool, serverID int, code logparse.VoteCode) error {
	return u.repository.AddResult(ctx, Result{
		SourceID:  sourceID,
		TargetID:  targetID,
		Name:      name,
		Success:   success,
		ServerID:  serverID,
		Code:      code,
		CreatedOn: time.Now(),
	})
}

func (u Votes) Query(ctx context.Context, filter Query) ([]Result, int64, error) {
	return u.repository.Query(ctx, filter)
}

// Start will begin ingesting vote events and record them to the database.
func (u Votes) Start(ctx context.Context) {
	type voteState struct {
		name    string
		success bool
		code    logparse.VoteCode
	}

	eventChan := make(chan logparse.ServerEvent)
	if errRegister := u.broadcaster.Consume(eventChan, logparse.VoteSuccess, logparse.VoteFailed, logparse.VoteDetails); errRegister != nil {
		slog.Warn("logWriter Tried to register duplicate reader channel", slog.String("error", errRegister.Error()))

		return
	}

	// Track recent votes and reject duplicates. Sometimes vote results get logged twice
	var recent []Result

	active := map[int]voteState{}

	cleanupTimer := time.NewTicker(time.Second * 5)

	for {
		select {
		case <-ctx.Done():
			return
		case <-cleanupTimer.C:
			// Cleanup timed out results
			var valid []Result

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

				result := Result{
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
					slog.Error("Failed to add vote result", slog.String("error", err.Error()))
				}

				recent = append(recent, result)

				delete(active, evt.ServerID)

				source, errSource := u.persons.GetOrCreatePersonBySteamID(ctx, result.SourceID)
				if errSource != nil {
					slog.Error("Failed to load vote source", slog.String("error", errSource.Error()), slog.String("steam_id", result.SourceID.String()))
				}

				target, errTarget := u.persons.GetOrCreatePersonBySteamID(ctx, result.SourceID)
				if errTarget != nil {
					slog.Error("Failed to load vote target", slog.String("error", errSource.Error()), slog.String("steam_id", result.TargetID.String()))
				}
				u.notif.Send(notification.NewDiscord(u.logChannelID, VoteResultMessage(result, source, target)))
			}
		}
	}
}
