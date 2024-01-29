package wordfilter

import (
	"context"
	"errors"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
)

type wordfilterUsecase struct {
	wfr         domain.WordFilterRepository
	discord     domain.DiscordUsecase
	wordFilters *WordFilters
}

func NewWordFilterUsecase(wfr domain.WordFilterRepository, duc domain.DiscordUsecase) domain.WordFilterUsecase {
	return &wordfilterUsecase{wfr: wfr, discord: duc, wordFilters: NewWordFilters()}
}

func (w *wordfilterUsecase) Import(ctx context.Context) error {
	filters, _, errFilters := w.wfr.GetFilters(ctx, domain.FiltersQueryFilter{})
	if errFilters != nil && !errors.Is(errFilters, domain.ErrNoResult) {
		return errFilters
	}
	w.wordFilters.Import(filters)

	return nil
}

// FilterAdd creates a new chat filter using a regex pattern.
func (w *wordfilterUsecase) FilterAdd(ctx context.Context, filter *domain.Filter) error {
	if errSave := w.wfr.SaveFilter(ctx, filter); errSave != nil {
		if errors.Is(errSave, domain.ErrDuplicate) {
			return domain.ErrDuplicate
		}

		// env.Log().Error("Error saving filter word", zap.Error(errSave))

		return errors.Join(errSave, domain.ErrSaveChanges)
	}

	filter.Init()

	// TODO
	// app.wordFilters.Add(filter)

	w.discord.SendPayload(domain.ChannelModLog, discord.FilterAddMessage(*filter))

	return nil
}

func (w *wordfilterUsecase) Check(query string) []domain.Filter {
	return w.wordFilters.Check(query)
}

func (w *wordfilterUsecase) SaveFilter(ctx context.Context, filter *domain.Filter) error {
	return w.wfr.SaveFilter(ctx, filter)
}

func (w *wordfilterUsecase) DropFilter(ctx context.Context, filter *domain.Filter) error {
	return w.wfr.DropFilter(ctx, filter)
}

func (w *wordfilterUsecase) GetFilterByID(ctx context.Context, filterID int64, filter *domain.Filter) error {
	return w.wfr.GetFilterByID(ctx, filterID, filter)
}

func (w *wordfilterUsecase) GetFilters(ctx context.Context, opts domain.FiltersQueryFilter) ([]domain.Filter, int64, error) {
	return w.wfr.GetFilters(ctx, opts)
}

func (w *wordfilterUsecase) AddMessageFilterMatch(ctx context.Context, messageID int64, filterID int64) error {
	return w.wfr.AddMessageFilterMatch(ctx, messageID, filterID)
}
