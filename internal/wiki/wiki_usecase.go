package wiki

import (
	"context"

	"github.com/leighmacdonald/gbans/internal/domain"
)

type wikiUsecase struct {
	wikiRepo domain.WikiRepository
}

func NewWikiUsecase(repository domain.WikiRepository) domain.WikiUsecase {
	return &wikiUsecase{wikiRepo: repository}
}

func (w *wikiUsecase) GetWikiPageBySlug(ctx context.Context, slug string) (domain.WikiPage, error) {
	return w.wikiRepo.GetWikiPageBySlug(ctx, slug)
}

func (w *wikiUsecase) DeleteWikiPageBySlug(ctx context.Context, slug string) error {
	return w.wikiRepo.DeleteWikiPageBySlug(ctx, slug)
}

func (w *wikiUsecase) SaveWikiPage(ctx context.Context, page *domain.WikiPage) error {
	return w.wikiRepo.SaveWikiPage(ctx, page)
}
