package wiki

import (
	"context"

	"github.com/leighmacdonald/gbans/internal/domain"
)

type wikiUsecase struct {
	wikiRepo domain.WikiRepository
}

func NewWikiUsecase(wr domain.WikiRepository) domain.WikiUsecase {
	return &wikiUsecase{wikiRepo: wr}
}

func (w *wikiUsecase) GetWikiPageBySlug(ctx context.Context, slug string, page *domain.Page) error {
	return w.wikiRepo.GetWikiPageBySlug(ctx, slug, page)
}

func (w *wikiUsecase) DeleteWikiPageBySlug(ctx context.Context, slug string) error {
	return w.wikiRepo.DeleteWikiPageBySlug(ctx, slug)
}

func (w *wikiUsecase) SaveWikiPage(ctx context.Context, page *domain.Page) error {
	return w.wikiRepo.SaveWikiPage(ctx, page)
}
