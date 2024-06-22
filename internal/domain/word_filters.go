package domain

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

type WordFilterRepository interface {
	SaveFilter(ctx context.Context, filter *Filter) error
	DropFilter(ctx context.Context, filter Filter) error
	GetFilterByID(ctx context.Context, filterID int64) (Filter, error)
	GetFilters(ctx context.Context) ([]Filter, error)
	AddMessageFilterMatch(ctx context.Context, messageID int64, filterID int64) error
}

type WordFilterUsecase interface {
	Edit(ctx context.Context, user PersonInfo, filterID int64, filter Filter) (Filter, error)
	Create(ctx context.Context, user PersonInfo, filter Filter) (Filter, error)
	DropFilter(ctx context.Context, filterID int64) error
	GetFilterByID(ctx context.Context, filterID int64) (Filter, error)
	GetFilters(ctx context.Context) ([]Filter, error)
	Check(query string) []Filter
	Import(ctx context.Context) error
	AddMessageFilterMatch(ctx context.Context, messageID int64, filterID int64) error
}

type RequestQuery struct {
	Query string `json:"query" url:"query"`
}

type FilterAction int

const (
	Kick FilterAction = iota
	Mute
	Ban
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
