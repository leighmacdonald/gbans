package usecase

import (
	"context"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/wiki"
)

type wikiUsecase struct {
	wikiRepo domain.WikiRepository
}

func NewServersUsecase(wr domain.WikiRepository) domain.WikiUsecase {
	return &wikiUsecase{wikiRepo: wr}
}

func (w *wikiUsecase) GetWikiPageBySlug(ctx context.Context, slug string, page *wiki.Page) error {
	return w.wikiRepo.GetWikiPageBySlug(ctx, slug, page)
}

func (w *wikiUsecase) DeleteWikiPageBySlug(ctx context.Context, slug string) error {
	return w.wikiRepo.DeleteWikiPageBySlug(ctx, slug)
}

func (w *wikiUsecase) SaveWikiPage(ctx context.Context, page *wiki.Page) error {
	return w.wikiRepo.SaveWikiPage(ctx, page)
}

func (w *wikiUsecase) SaveMedia(ctx context.Context, media *domain.Media) error {
	return w.wikiRepo.SaveMedia(ctx, media)
}

func (w *wikiUsecase) GetMediaByAssetID(ctx context.Context, uuid uuid.UUID, media *domain.Media) error {
	return w.wikiRepo.GetMediaByAssetID(ctx, uuid, media)
}

func (w *wikiUsecase) GetMediaByName(ctx context.Context, name string, media *domain.Media) error {
	return w.wikiRepo.GetMediaByName(ctx, name, media)
}

func (w *wikiUsecase) GetMediaByID(ctx context.Context, mediaID int, media *domain.Media) error {
	return w.wikiRepo.GetMediaByID(ctx, mediaID, media)
}
