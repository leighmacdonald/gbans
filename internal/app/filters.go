package app

import (
	"github.com/leighmacdonald/gbans/internal/event"
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

// IsFilteredWord checks to see if the body of text contains a known filtered word
func IsFilteredWord(body string) (bool, *model.Filter) {
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

func (g *gbans) filterWorker() {
	c := make(chan model.ServerEvent)
	if err := event.RegisterConsumer(c, []logparse.MsgType{logparse.Say, logparse.SayTeam}); err != nil {
		log.Fatalf("Failed to register event reader: %v", err)
	}
	for {
		select {
		case evt := <-c:
			wordFiltersMu.RLock()
			var matched *model.Filter
			for _, f := range wordFilters {
				if f.Match(evt.Extra) {
					matched = f
					break
				}
			}
			wordFiltersMu.RUnlock()
			if matched != nil {
				g.addWarning(evt.Source.SteamID, warnLanguage)
			}
		case <-g.ctx.Done():
			return
		}
	}
}
