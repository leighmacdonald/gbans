package app

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/chat"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"go.uber.org/zap"
)

func newChatLogger(log *zap.Logger, database database.Stores, broadcaster *fp.Broadcaster[logparse.EventType, logparse.ServerEvent],
	filters *WordFilters, warningTracker *chat.Tracker, matchUUIDMap fp.MutexMap[int, uuid.UUID],
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
	database       database.Stores
	events         chan logparse.ServerEvent
	broadcaster    *fp.Broadcaster[logparse.EventType, logparse.ServerEvent]
	wordFilters    *WordFilters
	warningTracker *chat.Tracker
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
					c.log.Warn("Empty Person message body, skipping")

					continue
				}

				var author domain.Person
				if errPerson := c.database.GetOrCreatePersonBySteamID(ctx, newServerEvent.SID, &author); errPerson != nil {
					c.log.Error("Failed to add chat history, could not get author", zap.Error(errPerson))

					continue
				}

				matchID, _ := c.matchUUIDMap.Get(evt.ServerID)

				msg := domain.PersonMessage{
					SteamID:     newServerEvent.SID,
					PersonaName: strings.ToValidUTF8(newServerEvent.Name, "_"),
					ServerName:  evt.ServerName,
					ServerID:    evt.ServerID,
					Body:        strings.ToValidUTF8(newServerEvent.Msg, "_"),
					Team:        newServerEvent.Team,
					CreatedOn:   newServerEvent.CreatedOn,
					MatchID:     matchID,
				}

				if errChat := c.database.AddChatHistory(ctx, &msg); errChat != nil {
					c.log.Error("Failed to add chat history", zap.Error(errChat))

					continue
				}

				go func(userMsg domain.PersonMessage) {
					if msg.ServerName == "localhost-1" {
						c.log.Debug("Chat message",
							zap.Int64("id", msg.PersonMessageID),
							zap.String("server", evt.ServerName),
							zap.String("name", newServerEvent.Name),
							zap.String("steam_id", newServerEvent.SID.String()),
							zap.Bool("team", msg.Team),
							zap.String("message", msg.Body))
					}

					matchedWord, matchedFilter := c.wordFilters.Match(userMsg.Body)
					if matchedFilter != nil {
						if errSaveMatch := c.database.AddMessageFilterMatch(ctx, userMsg.PersonMessageID, matchedFilter.FilterID); errSaveMatch != nil {
							c.log.Error("Failed to save message findMatch status", zap.Error(errSaveMatch))
						}

						c.warningTracker.WarningChan <- domain.NewUserWarning{
							UserMessage: userMsg,
							UserWarning: domain.UserWarning{
								WarnReason:    domain.Language,
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
