package app

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/event"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	log "github.com/sirupsen/logrus"
	"sync"
)

var (
	wordFilters   []model.Filter
	wordFiltersMu *sync.RWMutex
)

func init() {
	wordFiltersMu = &sync.RWMutex{}
}

// importFilteredWords loads the supplied word list into memory
func importFilteredWords(filters []model.Filter) {
	wordFiltersMu.Lock()
	defer wordFiltersMu.Unlock()
	wordFilters = filters
}

func filterWorker(ctx context.Context, database store.Store, botSendMessageChan chan discordPayload) {
	eventChan := make(chan model.ServerEvent)
	if errRegister := event.Consume(eventChan, []logparse.EventType{logparse.Say, logparse.SayTeam}); errRegister != nil {
		log.Fatalf("Failed to register event reader: %v", errRegister)
	}
	for {
		select {
		case serverEvent := <-eventChan:
			msg, found := serverEvent.MetaData["msg"].(string)
			if !found {
				continue
			}
			matched, _ := ContainsFilteredWord(msg)
			if matched {
				warningChan <- newUserWarning{
					SteamId: serverEvent.Source.SteamID,
					userWarning: userWarning{
						WarnReason: model.Language,
						CreatedOn:  config.Now(),
					},
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
