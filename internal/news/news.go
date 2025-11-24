package news

import (
	"context"
	"log/slog"
	"time"

	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/notification"
)

type Article struct {
	NewsID      int       `json:"news_id"`
	Title       string    `json:"title"`
	BodyMD      string    `json:"body_md"`
	IsPublished bool      `json:"is_published"`
	CreatedOn   time.Time `json:"created_on"`
	UpdatedOn   time.Time `json:"updated_on"`
}

type News struct {
	repository    Repository
	notifications notification.Notifier
	logChannelID  string
}

func NewNews(repository Repository, notifications notification.Notifier, logChannelID string) News {
	return News{repository: repository, notifications: notifications, logChannelID: logChannelID}
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
		return httphelper.ErrTooShort
	}

	if entry.BodyMD == "" {
		return httphelper.ErrTooShort
	}

	isNew := entry.NewsID > 0
	if err := u.repository.Save(ctx, entry); err != nil {
		return err
	}

	if entry.IsPublished {
		if isNew {
			go u.notifications.Send(notification.NewDiscord(u.logChannelID,
				NewNewsMessage(entry.BodyMD, entry.Title)))
		} else {
			go u.notifications.Send(notification.NewDiscord(u.logChannelID,
				EditNewsMessages(entry.BodyMD, entry.Title)))
		}
	}

	return nil
}

func (u News) DropNewsArticle(ctx context.Context, newsID int) error {
	if err := u.repository.DropNewsArticle(ctx, newsID); err != nil {
		return err
	}

	slog.Info("Deleted news article", slog.Int("news_id", newsID))

	return nil
}
