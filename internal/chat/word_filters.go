package chat

import (
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"golang.org/x/exp/slices"
)

var (
	ErrInvalidRegex = errors.New("invalid regex format")
)

type FilterAction int

const (
	FilterActionKick FilterAction = iota
	FilterActionMute
	FilterActionBan
)

func NewFilter(author steamid.SteamID, pattern string, regex bool, action FilterAction, duration string, weight int) (Filter, error) {
	now := time.Now()

	filter := Filter{
		AuthorID:     author,
		Pattern:      pattern,
		IsRegex:      regex,
		IsEnabled:    true,
		Action:       action,
		Duration:     duration,
		Regex:        nil,
		TriggerCount: 0,
		Weight:       weight,
		CreatedOn:    now,
		UpdatedOn:    now,
	}

	if regex {
		compiled, errRegex := regexp.Compile(pattern)
		if errRegex != nil {
			return Filter{}, errors.Join(errRegex, ErrInvalidRegex)
		}

		filter.Regex = compiled
	}

	return filter, nil
}

type Filter struct {
	FilterID     int64           `json:"filter_id"`
	AuthorID     steamid.SteamID `json:"author_id"`
	Pattern      string          `json:"pattern"`
	IsRegex      bool            `json:"is_regex"`
	IsEnabled    bool            `json:"is_enabled"`
	Action       FilterAction    `json:"action"`
	Duration     string          `json:"duration"`
	Regex        *regexp.Regexp  `json:"-"`
	TriggerCount int64           `json:"trigger_count"`
	Weight       int             `json:"weight"`
	CreatedOn    time.Time       `json:"created_on"`
	UpdatedOn    time.Time       `json:"updated_on"`
}

func (f *Filter) Init() {
	if f.IsRegex {
		f.Regex = regexp.MustCompile(f.Pattern)
	}
}

func (f *Filter) Match(value string) bool {
	if f.IsRegex {
		return f.Regex.MatchString(strings.ToLower(value))
	}

	return f.Pattern == strings.ToLower(value)
}

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
	UserMessage chat.PersonMessage
	PlayerID    int
	UserWarning
}

type Warnings interface {
	State() map[string][]UserWarning
}

type WordFilters struct {
	*sync.RWMutex
	wordFilters []Filter
}

func NewWordFilters() *WordFilters {
	return &WordFilters{
		RWMutex: &sync.RWMutex{},
	}
}

// Import loads the supplied word list into memory.
func (f *WordFilters) Import(filters []Filter) {
	f.Lock()
	defer f.Unlock()
	f.wordFilters = filters
}

func (f *WordFilters) Add(filter Filter) {
	f.Lock()
	f.wordFilters = append(f.wordFilters, filter)
	f.Unlock()
}

// Match checks to see if the body of text contains a known filtered word
// It will only return the first matched filter found.
func (f *WordFilters) Match(body string) (string, Filter, bool) {
	if body == "" {
		return "", Filter{}, false
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

	return "", Filter{}, false
}

func (f *WordFilters) Remove(filterID int64) {
	f.Lock()
	defer f.Unlock()

	f.wordFilters = slices.DeleteFunc(f.wordFilters, func(filter Filter) bool {
		return filter.FilterID == filterID
	})
}

// Check can be used to check if a phrase will match any filters.
func (f *WordFilters) Check(message string) []Filter {
	if message == "" {
		return nil
	}

	words := strings.Split(strings.ToLower(message), " ")

	f.RLock()
	defer f.RUnlock()

	var found []Filter

	for _, filter := range f.wordFilters {
		for _, word := range words {
			if filter.Match(word) {
				found = append(found, filter)
			}
		}
	}

	return found
}
