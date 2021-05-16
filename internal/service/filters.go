package service

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	log "github.com/sirupsen/logrus"
	"strings"
	"sync"
)

var (
	wordFilters   []*model.Filter
	wordFiltersMu *sync.RWMutex
)

func init() {
	wordFiltersMu = &sync.RWMutex{}
}

// importFilteredWords loads the supplied word list into memory
func importFilteredWords(filters []*model.Filter) {
	wordFiltersMu.Lock()
	defer wordFiltersMu.Unlock()
	wordFilters = filters
}

// isFilteredWord checks to see if the body of text contains a known filtered word
func isFilteredWord(body string) (bool, *model.Filter) {
	if body == "" {
		return false, nil
	}
	wordFiltersMu.RLock()
	defer wordFiltersMu.RUnlock()
	ls := strings.ToLower(body)
	for _, filter := range wordFilters {
		if filter.Match(ls) {
			return true, filter
		}
	}
	return false, nil
}

func filterWorker(ctx context.Context) {
	c := make(chan logEvent)
	if err := registerLogEventReader(c, []logparse.MsgType{logparse.Say, logparse.SayTeam}); err != nil {
		log.Fatalf("Failed to register event reader: %v", err)
	}
	for {
		select {
		case evt := <-c:
			var m logparse.SayTeamEvt
			if err := evt.Decode(&m); err != nil {
				log.Errorf("Failed to decode event")
			}
			wordFiltersMu.RLock()
			var matched *model.Filter
			for _, f := range wordFilters {
				if f.Match(m.Msg) {
					matched = f
					break
				}
			}
			wordFiltersMu.RUnlock()
			if matched != nil {
				addWarning(evt.Player1.SteamID, warnLanguage)
			}
		case <-ctx.Done():
			return
		}
	}
}
