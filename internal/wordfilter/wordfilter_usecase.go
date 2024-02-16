package wordfilter

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/util"
)

type wordFilterUsecase struct {
	filterRepository domain.WordFilterRepository
	wordFilters      *WordFilters
}

func NewWordFilterUsecase(filterRepository domain.WordFilterRepository) domain.WordFilterUsecase {
	return &wordFilterUsecase{filterRepository: filterRepository, wordFilters: NewWordFilters()}
}

func (w *wordFilterUsecase) Import(ctx context.Context) error {
	filters, _, errFilters := w.filterRepository.GetFilters(ctx, domain.FiltersQueryFilter{})
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
	existingFilter, errGet := w.filterRepository.GetFilterByID(ctx, filterID)
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

	if errSave := w.filterRepository.SaveFilter(ctx, &existingFilter); errSave != nil {
		return domain.Filter{}, errSave
	}

	return existingFilter, nil
}

func (w *wordFilterUsecase) Create(ctx context.Context, user domain.PersonInfo, opts domain.Filter) (domain.Filter, error) {
	if opts.Pattern == "" {
		return domain.Filter{}, domain.ErrInvalidPattern
	}

	_, errDur := util.ParseDuration(opts.Duration)
	if errDur != nil {
		return domain.Filter{}, util.ErrInvalidDuration
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

	if errSave := w.filterRepository.SaveFilter(ctx, &newFilter); errSave != nil {
		if errors.Is(errSave, domain.ErrDuplicate) {
			return domain.Filter{}, domain.ErrDuplicate
		}

		return domain.Filter{}, errors.Join(errSave, domain.ErrSaveChanges)
	}

	newFilter.Init()

	w.wordFilters.Add(&newFilter)

	return newFilter, nil
}

func (w *wordFilterUsecase) DropFilter(ctx context.Context, filter domain.Filter) error {
	return w.filterRepository.DropFilter(ctx, filter)
}

func (w *wordFilterUsecase) GetFilterByID(ctx context.Context, filterID int64) (domain.Filter, error) {
	return w.filterRepository.GetFilterByID(ctx, filterID)
}

func (w *wordFilterUsecase) GetFilters(ctx context.Context, opts domain.FiltersQueryFilter) ([]domain.Filter, int64, error) {
	return w.filterRepository.GetFilters(ctx, opts)
}

func (w *wordFilterUsecase) AddMessageFilterMatch(ctx context.Context, messageID int64, filterID int64) error {
	return w.filterRepository.AddMessageFilterMatch(ctx, messageID, filterID)
}
