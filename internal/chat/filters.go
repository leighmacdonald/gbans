package chat

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
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
