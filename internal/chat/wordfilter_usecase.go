package chat

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"time"

	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/datetime"
)

type wordFilterUsecase struct {
	repository    WordFilterRepository
	wordFilters   *WordFilters
	notifications domain.NotificationUsecase
}

func NewWordFilterUsecase(repository WordFilterRepository, notifications domain.NotificationUsecase) WordFilterUsecase {
	return &wordFilterUsecase{repository: repository, wordFilters: NewWordFilters(), notifications: notifications}
}

func (w *wordFilterUsecase) Import(ctx context.Context) error {
	filters, errFilters := w.repository.GetFilters(ctx)
	if errFilters != nil && !errors.Is(errFilters, database.ErrNoResult) {
		return errFilters
	}

	w.wordFilters.Import(filters)

	return nil
}

func (w *wordFilterUsecase) Check(query string) []Filter {
	return w.wordFilters.Check(query)
}

func (w *wordFilterUsecase) Edit(ctx context.Context, user domain.PersonInfo, filterID int64, filter Filter) (Filter, error) {
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

	w.wordFilters.Remove(filterID)
	w.wordFilters.Add(existingFilter)

	slog.Info("Filter updated", slog.Int64("filter_id", filterID))

	return existingFilter, nil
}

func (w *wordFilterUsecase) Create(ctx context.Context, user domain.PersonInfo, opts Filter) (Filter, error) {
	if opts.Pattern == "" {
		return Filter{}, domain.ErrInvalidPattern
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
		return Filter{}, domain.ErrInvalidWeight
	}

	now := time.Now()

	newFilter := Filter{
		AuthorID:  user.GetSteamID(),
		Pattern:   opts.Pattern,
		Action:    opts.Action,
		Duration:  opts.Duration,
		CreatedOn: now,
		UpdatedOn: now,
		IsRegex:   opts.IsRegex,
		IsEnabled: opts.IsEnabled,
		Weight:    opts.Weight,
	}

	if errSave := w.repository.SaveFilter(ctx, &newFilter); errSave != nil {
		if errors.Is(errSave, database.ErrDuplicate) {
			return Filter{}, database.ErrDuplicate
		}

		return Filter{}, errors.Join(errSave, domain.ErrSaveChanges)
	}

	newFilter.Init()

	w.wordFilters.Add(newFilter)

	w.notifications.Enqueue(ctx, domain.NewDiscordNotification(discord.ChannelWordFilterLog, discord.FilterAddMessage(newFilter)))

	slog.Info("Created filter", slog.Int64("filter_id", newFilter.FilterID))

	return newFilter, nil
}

func (w *wordFilterUsecase) DropFilter(ctx context.Context, filterID int64) error {
	filter, errGet := w.GetFilterByID(ctx, filterID)
	if errGet != nil {
		return errGet
	}

	if err := w.repository.DropFilter(ctx, filter); err != nil {
		return err
	}

	w.wordFilters.Remove(filterID)

	w.notifications.Enqueue(ctx, domain.NewDiscordNotification(discord.ChannelWordFilterLog, discord.FilterDelMessage(filter)))

	slog.Info("Deleted filter", slog.Int64("filter_id", filterID))

	return nil
}

func (w *wordFilterUsecase) GetFilterByID(ctx context.Context, filterID int64) (Filter, error) {
	return w.repository.GetFilterByID(ctx, filterID)
}

func (w *wordFilterUsecase) GetFilters(ctx context.Context) ([]Filter, error) {
	return w.repository.GetFilters(ctx)
}

func (w *wordFilterUsecase) AddMessageFilterMatch(ctx context.Context, messageID int64, filterID int64) error {
	return w.repository.AddMessageFilterMatch(ctx, messageID, filterID)
}
