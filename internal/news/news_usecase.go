package news

import (
	"context"

	"github.com/leighmacdonald/gbans/internal/domain"
)

type newsUsecase struct {
	nr domain.NewsRepository
}

func NewNewsUsecase(nu domain.NewsUsecase) domain.NewsUsecase {
	return &newsUsecase{nr: nu}
}

func (u newsUsecase) GetNewsLatest(ctx context.Context, limit int, includeUnpublished bool) ([]domain.NewsEntry, error) {
	return u.nr.GetNewsLatest(ctx, limit, includeUnpublished)
}

func (u newsUsecase) GetNewsLatestArticle(ctx context.Context, includeUnpublished bool, entry *domain.NewsEntry) error {
	return u.nr.GetNewsLatestArticle(ctx, includeUnpublished, entry)
}

func (u newsUsecase) GetNewsByID(ctx context.Context, newsID int, entry *domain.NewsEntry) error {
	return u.nr.GetNewsByID(ctx, newsID, entry)
}

func (u newsUsecase) SaveNewsArticle(ctx context.Context, entry *domain.NewsEntry) error {
	return u.nr.SaveNewsArticle(ctx, entry)
}

func (u newsUsecase) DropNewsArticle(ctx context.Context, newsID int) error {
	return u.nr.DropNewsArticle(ctx, newsID)
}
