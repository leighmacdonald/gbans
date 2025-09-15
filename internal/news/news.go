package news

import (
	"context"
	"log/slog"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
)

type Article struct {
	NewsID      int       `json:"news_id"`
	Title       string    `json:"title"`
	BodyMD      string    `json:"body_md"`
	IsPublished bool      `json:"is_published"`
	CreatedOn   time.Time `json:"created_on,omitzero"`
	UpdatedOn   time.Time `json:"updated_on,omitzero"`
}

type News struct {
	repository NewsRepository
}

func NewNews(repository NewsRepository) News {
	return News{repository: repository}
}

func (u News) GetNewsLatest(ctx context.Context, limit int, includeUnpublished bool) ([]Article, error) {
	return u.repository.GetNewsLatest(ctx, limit, includeUnpublished)
}

func (u News) GetNewsLatestArticle(ctx context.Context, includeUnpublished bool, entry *Article) error {
	return u.repository.GetNewsLatestArticle(ctx, includeUnpublished, entry)
}

func (u News) GetNewsByID(ctx context.Context, newsID int, entry *Article) error {
	return u.repository.GetNewsByID(ctx, newsID, entry)
}

func (u News) Save(ctx context.Context, entry *Article) error {
	if entry.Title == "" {
		return domain.ErrTooShort
	}

	if entry.BodyMD == "" {
		return domain.ErrTooShort
	}

	return u.repository.Save(ctx, entry)
}

func (u News) DropNewsArticle(ctx context.Context, newsID int) error {
	if err := u.repository.DropNewsArticle(ctx, newsID); err != nil {
		return err
	}

	slog.Info("Deleted news article", slog.Int("news_id", newsID))

	return nil
}
