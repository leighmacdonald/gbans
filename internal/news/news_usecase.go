package news

import (
	"context"
	"log/slog"

	"github.com/leighmacdonald/gbans/internal/domain"
)

type newsUsecase struct {
	repository domain.NewsRepository
}

func NewNewsUsecase(repository domain.NewsRepository) domain.NewsUsecase {
	return &newsUsecase{repository: repository}
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

func (u newsUsecase) Save(ctx context.Context, entry *domain.NewsEntry) error {
	if entry.Title == "" {
		return domain.ErrTooShort
	}

	if entry.BodyMD == "" {
		return domain.ErrTooShort
	}

	return u.repository.Save(ctx, entry)
}

func (u newsUsecase) DropNewsArticle(ctx context.Context, newsID int) error {
	if err := u.repository.DropNewsArticle(ctx, newsID); err != nil {
		return err
	}

	slog.Info("Deleted news article", slog.Int("news_id", newsID))

	return nil
}
