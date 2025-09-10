package wordfilter

import (
	"strings"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/domain"
	"golang.org/x/exp/slices"
)

type UserWarning struct {
	WarnReason    ban.Reason `json:"warn_reason"`
	Message       string     `json:"message"`
	Matched       string     `json:"matched"`
	MatchedFilter Filter     `json:"matched_filter"`
	CreatedOn     time.Time  `json:"created_on"`
	Personaname   string     `json:"personaname"`
	Avatar        string     `json:"avatar"`
	ServerName    string     `json:"server_name"`
	ServerID      int        `json:"server_id"`
	SteamID       string     `json:"steam_id"`
	CurrentTotal  int        `json:"current_total"`
}

type NewUserWarning struct {
	UserMessage PersonMessage
	PlayerID    int
	UserWarning
}

type Warnings interface {
	State() map[string][]UserWarning
}

type WordFilters struct {
	*sync.RWMutex
	wordFilters []domain.Filter
}

func NewWordFilters() *WordFilters {
	return &WordFilters{
		RWMutex: &sync.RWMutex{},
	}
}

// Import loads the supplied word list into memory.
func (f *WordFilters) Import(filters []domain.Filter) {
	f.Lock()
	defer f.Unlock()
	f.wordFilters = filters
}

func (f *WordFilters) Add(filter domain.Filter) {
	f.Lock()
	f.wordFilters = append(f.wordFilters, filter)
	f.Unlock()
}

// Match checks to see if the body of text contains a known filtered word
// It will only return the first matched filter found.
func (f *WordFilters) Match(body string) (string, domain.Filter, bool) {
	if body == "" {
		return "", domain.Filter{}, false
	}

	words := strings.Split(strings.ToLower(body), " ")

	f.RLock()
	defer f.RUnlock()

	for _, filter := range f.wordFilters {
		for _, word := range words {
			if filter.IsEnabled && filter.Match(word) {
				return word, filter, true
			}
		}
	}

	return "", domain.Filter{}, false
}

func (f *WordFilters) Remove(filterID int64) {
	f.Lock()
	defer f.Unlock()

	f.wordFilters = slices.DeleteFunc(f.wordFilters, func(filter domain.Filter) bool {
		return filter.FilterID == filterID
	})
}

// Check can be used to check if a phrase will match any filters.
func (f *WordFilters) Check(message string) []domain.Filter {
	if message == "" {
		return nil
	}

	words := strings.Split(strings.ToLower(message), " ")

	f.RLock()
	defer f.RUnlock()

	var found []domain.Filter

	for _, filter := range f.wordFilters {
		for _, word := range words {
			if filter.Match(word) {
				found = append(found, filter)
			}
		}
	}

	return found
}
