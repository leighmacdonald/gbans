package app

import (
	"context"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"go.uber.org/zap"
)

func newChatLogger(log *zap.Logger, database store.Store, broadcaster *fp.Broadcaster[logparse.EventType, logparse.ServerEvent],
	filters *wordFilters, warningTracker *WarningTracker, matchUUIDMap fp.MutexMap[int, uuid.UUID],
) *chatLogger {
	return &chatLogger{
		log:            log.Named("chatRecorder"),
		events:         make(chan logparse.ServerEvent),
		broadcaster:    broadcaster,
		database:       database,
		wordFilters:    filters,
		warningTracker: warningTracker,
		matchUUIDMap:   matchUUIDMap,
	}
}

type chatLogger struct {
	log            *zap.Logger
	database       store.Store
	events         chan logparse.ServerEvent
	broadcaster    *fp.Broadcaster[logparse.EventType, logparse.ServerEvent]
	wordFilters    *wordFilters
	warningTracker *WarningTracker
	matchUUIDMap   fp.MutexMap[int, uuid.UUID]
}

func (c *chatLogger) start(ctx context.Context) {
	if errRegister := c.broadcaster.Consume(c.events, logparse.Say, logparse.SayTeam); errRegister != nil {
		c.log.Warn("logWriter Tried to register duplicate reader channel", zap.Error(errRegister))

		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-c.events:
			switch evt.EventType {
			case logparse.Say:
				fallthrough
			case logparse.SayTeam:
				newServerEvent, ok := evt.Event.(logparse.SayEvt)
				if !ok {
					continue
				}

				if newServerEvent.Msg == "" {
					c.log.Warn("Empty person message body, skipping")

					continue
				}

				var author model.Person
				if errPerson := store.GetPersonBySteamID(ctx, c.database, newServerEvent.SID, &author); errPerson != nil {
					c.log.Error("Failed to add chat history, could not get author", zap.Error(errPerson))

					continue
				}

				matchID, _ := c.matchUUIDMap.Get(evt.ServerID)

				msg := model.PersonMessage{
					SteamID:     newServerEvent.SID,
					PersonaName: strings.ToValidUTF8(newServerEvent.Name, "_"),
					ServerName:  evt.ServerName,
					ServerID:    evt.ServerID,
					Body:        strings.ToValidUTF8(newServerEvent.Msg, "_"),
					Team:        newServerEvent.Team,
					CreatedOn:   newServerEvent.CreatedOn,
					MatchID:     matchID,
				}

				if errChat := store.AddChatHistory(ctx, c.database, &msg); errChat != nil {
					c.log.Error("Failed to add chat history", zap.Error(errChat))

					continue
				}

				go func(userMsg model.PersonMessage) {
					if msg.ServerName == "localhost-1" {
						c.log.Debug("Chat message",
							zap.Int64("id", msg.PersonMessageID),
							zap.String("server", evt.ServerName),
							zap.String("name", newServerEvent.Name),
							zap.String("steam_id", newServerEvent.SID.String()),
							zap.Bool("team", msg.Team),
							zap.String("message", msg.Body))
					}

					matchedWord, matchedFilter := c.wordFilters.findMatch(userMsg.Body)
					if matchedFilter != nil {
						if errSaveMatch := store.AddMessageFilterMatch(ctx, c.database, userMsg.PersonMessageID, matchedFilter.FilterID); errSaveMatch != nil {
							c.log.Error("Failed to save message findMatch status", zap.Error(errSaveMatch))
						}

						c.warningTracker.warningChan <- newUserWarning{
							userMessage: userMsg,
							userWarning: userWarning{
								WarnReason:    model.Language,
								Message:       userMsg.Body,
								Matched:       matchedWord,
								MatchedFilter: matchedFilter,
								CreatedOn:     time.Now(),
								Personaname:   userMsg.PersonaName,
								Avatar:        userMsg.AvatarHash,
								ServerName:    userMsg.ServerName,
								ServerID:      userMsg.ServerID,
								SteamID:       userMsg.SteamID.String(),
							},
						}
					}
				}(msg)
			}
		}
	}
}
