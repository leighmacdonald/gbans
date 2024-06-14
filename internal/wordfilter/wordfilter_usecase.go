package wordfilter

import (
	"context"
	"errors"
	"github.com/leighmacdonald/gbans/internal/discord"
	"log/slog"
	"regexp"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/datetime"
)

type wordFilterUsecase struct {
	repository  domain.WordFilterRepository
	wordFilters *WordFilters
	discord     domain.DiscordUsecase
}

func NewWordFilterUsecase(repository domain.WordFilterRepository, discord domain.DiscordUsecase) domain.WordFilterUsecase {
	return &wordFilterUsecase{repository: repository, wordFilters: NewWordFilters(), discord: discord}
}

func (w *wordFilterUsecase) Import(ctx context.Context) error {
	filters, errFilters := w.repository.GetFilters(ctx)
	if errFilters != nil && !errors.Is(errFilters, domain.ErrNoResult) {
		return errFilters
	}

	w.wordFilters.Import(filters)

	return nil
}

func (w *wordFilterUsecase) Check(query string) []domain.Filter {
	return w.wordFilters.Check(query)
}

func (w *wordFilterUsecase) Edit(ctx context.Context, user domain.PersonInfo, filterID int64, filter domain.Filter) (domain.Filter, error) {
	existingFilter, errGet := w.repository.GetFilterByID(ctx, filterID)
	if errGet != nil {
		return domain.Filter{}, errGet
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
		return domain.Filter{}, errSave
	}

	slog.Info("Edited filter", slog.Int64("filter_id", filterID))

	return existingFilter, nil
}

func (w *wordFilterUsecase) Create(ctx context.Context, user domain.PersonInfo, opts domain.Filter) (domain.Filter, error) {
	if opts.Pattern == "" {
		return domain.Filter{}, domain.ErrInvalidPattern
	}

	_, errDur := datetime.ParseDuration(opts.Duration)
	if errDur != nil {
		return domain.Filter{}, datetime.ErrInvalidDuration
	}

	if opts.IsRegex {
		_, compErr := regexp.Compile(opts.Pattern)
		if compErr != nil {
			return domain.Filter{}, domain.ErrInvalidRegex
		}
	}

	if opts.Weight < 1 {
		return domain.Filter{}, domain.ErrInvalidWeight
	}

	now := time.Now()

	newFilter := domain.Filter{
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
		if errors.Is(errSave, domain.ErrDuplicate) {
			return domain.Filter{}, domain.ErrDuplicate
		}

		return domain.Filter{}, errors.Join(errSave, domain.ErrSaveChanges)
	}

	newFilter.Init()

	w.wordFilters.Add(&newFilter)

	slog.Info("Created filter", slog.Int64("filter_id", newFilter.FilterID))

	w.discord.SendPayload(domain.ChannelWordFilterLog, discord.FilterAddMessage(newFilter))

	return newFilter, nil
}

func (w *wordFilterUsecase) DropFilter(ctx context.Context, filter domain.Filter) error {
	if err := w.repository.DropFilter(ctx, filter); err != nil {
		return err
	}

	slog.Info("Deleted filter", slog.Int64("id", filter.FilterID))

	w.discord.SendPayload(domain.ChannelWordFilterLog, discord.FilterDelMessage(filter))

	return nil
}

func (w *wordFilterUsecase) GetFilterByID(ctx context.Context, filterID int64) (domain.Filter, error) {
	return w.repository.GetFilterByID(ctx, filterID)
}

func (w *wordFilterUsecase) GetFilters(ctx context.Context) ([]domain.Filter, error) {
	return w.repository.GetFilters(ctx)
}

func (w *wordFilterUsecase) AddMessageFilterMatch(ctx context.Context, messageID int64, filterID int64) error {
	return w.repository.AddMessageFilterMatch(ctx, messageID, filterID)
}
