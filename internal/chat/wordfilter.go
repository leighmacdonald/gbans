package chat

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/pkg/datetime"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"golang.org/x/exp/slices"
)

var (
	ErrInvalidRegex   = errors.New("invalid regex format")
	ErrInvalidPattern = errors.New("invalid pattern")

	ErrInvalidWeight = errors.New("invalid weight value")
)

type FiltersQueryFilter struct {
	query.Filter
}

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
	UserMessage Message
	PlayerID    int
	UserWarning
}

type Warnings interface {
	State() map[string][]UserWarning
}

type WordFilters struct {
	*sync.RWMutex
	repository  WordFilterRepository
	wordFilters []Filter
	notif       notification.Notifier
	config      *config.Configuration
}

func NewWordFilters(repository WordFilterRepository, notif notification.Notifier, config *config.Configuration) WordFilters {
	return WordFilters{repository: repository, RWMutex: &sync.RWMutex{}, notif: notif, config: config}
}

func (w *WordFilters) Add(filter Filter) {
	w.Lock()
	w.wordFilters = append(w.wordFilters, filter)
	w.Unlock()
}

// Match checks to see if the body of text contains a known filtered word
// It will only return the first matched filter found.
func (w *WordFilters) Match(body string) (string, Filter, bool) {
	if body == "" {
		return "", Filter{}, false
	}

	words := strings.Split(strings.ToLower(body), " ")

	w.RLock()
	defer w.RUnlock()

	for _, filter := range w.wordFilters {
		for _, word := range words {
			if filter.IsEnabled && filter.Match(word) {
				return word, filter, true
			}
		}
	}

	return "", Filter{}, false
}

func (w *WordFilters) Remove(filterID int64) {
	w.Lock()
	defer w.Unlock()

	w.wordFilters = slices.DeleteFunc(w.wordFilters, func(filter Filter) bool {
		return filter.FilterID == filterID
	})
}

// Check can be used to check if a phrase will match any filters.
func (w *WordFilters) Check(message string) []Filter {
	if message == "" {
		return nil
	}

	words := strings.Split(strings.ToLower(message), " ")

	w.RLock()
	defer w.RUnlock()

	var found []Filter

	for _, filter := range w.wordFilters {
		for _, word := range words {
			if filter.Match(word) {
				found = append(found, filter)
			}
		}
	}

	return found
}

func (w *WordFilters) Import(ctx context.Context) error {
	filters, errFilters := w.repository.GetFilters(ctx)
	if errFilters != nil && !errors.Is(errFilters, database.ErrNoResult) {
		return errFilters
	}

	w.Lock()
	defer w.Unlock()
	w.wordFilters = filters

	return nil
}

func (w *WordFilters) Edit(ctx context.Context, user person.Info, filterID int64, filter Filter) (Filter, error) {
	existingFilter, errGet := w.repository.GetFilterByID(ctx, filterID)
	if errGet != nil {
		return Filter{}, errGet
	}

	existingFilter.AuthorID = user.GetSteamID()
	existingFilter.UpdatedOn = time.Now()
	existingFilter.Pattern = filter.Pattern
	existingFilter.IsRegex = filter.IsRegex
	existingFilter.IsEnabled = filter.IsEnabled
	existingFilter.Action = filter.Action
	existingFilter.Duration = filter.Duration
	existingFilter.Weight = filter.Weight

	if errSave := w.repository.SaveFilter(ctx, &existingFilter); errSave != nil {
		return Filter{}, errSave
	}

	w.Remove(filterID)
	w.Add(existingFilter)

	slog.Info("Filter updated", slog.Int64("filter_id", filterID))

	return existingFilter, nil
}

func (w *WordFilters) Create(ctx context.Context, user person.Info, opts Filter) (Filter, error) {
	if opts.Pattern == "" {
		return Filter{}, ErrInvalidPattern
	}

	_, errDur := datetime.ParseDuration(opts.Duration)
	if errDur != nil {
		return Filter{}, datetime.ErrInvalidDuration
	}

	if opts.IsRegex {
		_, compErr := regexp.Compile(opts.Pattern)
		if compErr != nil {
			return Filter{}, ErrInvalidRegex
		}
	}

	if opts.Weight < 1 {
		return Filter{}, ErrInvalidWeight
	}

	newFilter := Filter{
		AuthorID:  user.GetSteamID(),
		Pattern:   opts.Pattern,
		Action:    opts.Action,
		Duration:  opts.Duration,
		CreatedOn: time.Now(),
		UpdatedOn: time.Now(),
		IsRegex:   opts.IsRegex,
		IsEnabled: opts.IsEnabled,
		Weight:    opts.Weight,
	}

	if errSave := w.repository.SaveFilter(ctx, &newFilter); errSave != nil {
		if errors.Is(errSave, database.ErrDuplicate) {
			return Filter{}, database.ErrDuplicate
		}

		return Filter{}, errors.Join(errSave, database.ErrSaveChanges)
	}

	newFilter.Init()

	w.Add(newFilter)

	w.notif.Send(notification.NewDiscord(w.config.Config().Discord.WordFilterLogChannelID, filterAddMessage(newFilter)))

	slog.Info("Created filter", slog.Int64("filter_id", newFilter.FilterID))

	return newFilter, nil
}

func (w *WordFilters) DropFilter(ctx context.Context, filterID int64) error {
	filter, errGet := w.GetFilterByID(ctx, filterID)
	if errGet != nil {
		return errGet
	}

	if err := w.repository.DropFilter(ctx, filter); err != nil {
		return err
	}

	w.Remove(filterID)

	w.notif.Send(notification.NewDiscord(w.config.Config().Discord.WordFilterLogChannelID, filterDelMessage(filter)))

	slog.Info("Deleted filter", slog.Int64("filter_id", filterID))

	return nil
}

func (w *WordFilters) GetFilterByID(ctx context.Context, filterID int64) (Filter, error) {
	return w.repository.GetFilterByID(ctx, filterID)
}

func (w *WordFilters) GetFilters(ctx context.Context) ([]Filter, error) {
	return w.repository.GetFilters(ctx)
}

func (w *WordFilters) AddMessageFilterMatch(ctx context.Context, messageID int64, filterID int64) error {
	return w.repository.AddMessageFilterMatch(ctx, messageID, filterID)
}
