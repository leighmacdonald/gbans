package news

import (
	"context"

	"github.com/leighmacdonald/gbans/internal/domain"
)

type newsUsecase struct {
	repository domain.NewsRepository
}

func NewNewsUsecase(repositoryu domain.NewsRepository) domain.NewsUsecase {
	return &newsUsecase{repository: repositoryu}
}

func (u newsUsecase) GetNewsLatest(ctx context.Context, limit int, includeUnpublished bool) ([]domain.NewsEntry, error) {
	return u.repository.GetNewsLatest(ctx, limit, includeUnpublished)
}

func (u newsUsecase) GetNewsLatestArticle(ctx context.Context, includeUnpublished bool, entry *domain.NewsEntry) error {
	return u.repository.GetNewsLatestArticle(ctx, includeUnpublished, entry)
}

func (u newsUsecase) GetNewsByID(ctx context.Context, newsID int, entry *domain.NewsEntry) error {
	return u.repository.GetNewsByID(ctx, newsID, entry)
}

func (u newsUsecase) SaveNewsArticle(ctx context.Context, entry *domain.NewsEntry) error {
	return u.repository.SaveNewsArticle(ctx, entry)
}

func (u newsUsecase) DropNewsArticle(ctx context.Context, newsID int) error {
	return u.repository.DropNewsArticle(ctx, newsID)
}
