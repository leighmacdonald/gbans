package domain

import (
	"context"
	"time"
)

type NewsRepository interface {
	GetNewsLatest(ctx context.Context, limit int, includeUnpublished bool) ([]NewsEntry, error)
	GetNewsLatestArticle(ctx context.Context, includeUnpublished bool, entry *NewsEntry) error
	GetNewsByID(ctx context.Context, newsID int, entry *NewsEntry) error
	SaveNewsArticle(ctx context.Context, entry *NewsEntry) error
	DropNewsArticle(ctx context.Context, newsID int) error
}

type NewsUsecase interface {
	GetNewsLatest(ctx context.Context, limit int, includeUnpublished bool) ([]NewsEntry, error)
	GetNewsLatestArticle(ctx context.Context, includeUnpublished bool, entry *NewsEntry) error
	GetNewsByID(ctx context.Context, newsID int, entry *NewsEntry) error
	SaveNewsArticle(ctx context.Context, entry *NewsEntry) error
	DropNewsArticle(ctx context.Context, newsID int) error
}

type NewsEntry struct {
	NewsID      int       `json:"news_id"`
	Title       string    `json:"title"`
	BodyMD      string    `json:"body_md"`
	IsPublished bool      `json:"is_published"`
	CreatedOn   time.Time `json:"created_on,omitempty"`
	UpdatedOn   time.Time `json:"updated_on,omitempty"`
}
