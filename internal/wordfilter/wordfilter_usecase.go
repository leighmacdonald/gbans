package wordfilter

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/util"
)

type wordfilterUsecase struct {
	filterRepository domain.WordFilterRepository
	wordFilters      *WordFilters
}

func NewWordFilterUsecase(filterRepository domain.WordFilterRepository) domain.WordFilterUsecase {
	return &wordfilterUsecase{filterRepository: filterRepository, wordFilters: NewWordFilters()}
}

func (w *wordfilterUsecase) Import(ctx context.Context) error {
	filters, _, errFilters := w.filterRepository.GetFilters(ctx, domain.FiltersQueryFilter{})
	if errFilters != nil && !errors.Is(errFilters, domain.ErrNoResult) {
		return errFilters
	}

	w.wordFilters.Import(filters)

	return nil
}

func (w *wordfilterUsecase) Check(query string) []domain.Filter {
	return w.wordFilters.Check(query)
}

func (w *wordfilterUsecase) SaveFilter(ctx context.Context, user domain.PersonInfo, filter *domain.Filter) error {
	if filter.Pattern == "" {
		return domain.ErrInvalidPattern
	}

	_, errDur := util.ParseDuration(filter.Duration)
	if errDur != nil {
		return util.ErrInvalidDuration
	}

	if filter.IsRegex {
		_, compErr := regexp.Compile(filter.Pattern)
		if compErr != nil {
			return domain.ErrInvalidRegex
		}
	}

	if filter.Weight < 1 {
		return domain.ErrInvalidWeight
	}

	now := time.Now()

	if filter.FilterID > 0 {
		existingFilter, errGet := w.filterRepository.GetFilterByID(ctx, filter.FilterID)
		if errGet != nil {
			return errGet
		}

		existingFilter.UpdatedOn = now
		existingFilter.Pattern = filter.Pattern
		existingFilter.IsRegex = filter.IsRegex
		existingFilter.IsEnabled = filter.IsEnabled
		existingFilter.Action = filter.Action
		existingFilter.Duration = filter.Duration
		existingFilter.Weight = filter.Weight

		if errSave := w.filterRepository.SaveFilter(ctx, &existingFilter); errSave != nil {
			return errSave
		}

		filter = &existingFilter
	} else {
		newFilter := domain.Filter{
			AuthorID:  user.GetSteamID(),
			Pattern:   filter.Pattern,
			Action:    filter.Action,
			Duration:  filter.Duration,
			CreatedOn: now,
			UpdatedOn: now,
			IsRegex:   filter.IsRegex,
			IsEnabled: filter.IsEnabled,
			Weight:    filter.Weight,
		}

		if errSave := w.filterRepository.SaveFilter(ctx, &newFilter); errSave != nil {
			return errSave
		}

		filter = &newFilter
	}

	if errSave := w.filterRepository.SaveFilter(ctx, filter); errSave != nil {
		if errors.Is(errSave, domain.ErrDuplicate) {
			return domain.ErrDuplicate
		}

		return errors.Join(errSave, domain.ErrSaveChanges)
	}

	filter.Init()

	// TODO
	// app.wordFilters.Add(filter)

	return nil
}

func (w *wordfilterUsecase) DropFilter(ctx context.Context, filter *domain.Filter) error {
	return w.filterRepository.DropFilter(ctx, filter)
}

func (w *wordfilterUsecase) GetFilterByID(ctx context.Context, filterID int64) (domain.Filter, error) {
	return w.filterRepository.GetFilterByID(ctx, filterID)
}

func (w *wordfilterUsecase) GetFilters(ctx context.Context, opts domain.FiltersQueryFilter) ([]domain.Filter, int64, error) {
	return w.filterRepository.GetFilters(ctx, opts)
}

func (w *wordfilterUsecase) AddMessageFilterMatch(ctx context.Context, messageID int64, filterID int64) error {
	return w.filterRepository.AddMessageFilterMatch(ctx, messageID, filterID)
}
