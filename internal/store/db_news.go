package store

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type NewsEntry struct {
	NewsId      int       `json:"news_id"`
	Title       string    `json:"title"`
	BodyMD      string    `json:"body_md"`
	IsPublished bool      `json:"is_published"`
	CreatedOn   time.Time `json:"created_on,omitempty"`
	UpdatedOn   time.Time `json:"updated_on,omitempty"`
}

func GetNewsLatest(ctx context.Context, limit int, includeUnpublished bool) ([]NewsEntry, error) {
	var articles []NewsEntry
	builder := sb.Select("news_id", "title", "body_md", "is_published", "created_on", "updated_on").
		From("news").
		OrderBy("created_on DESC")
	if !includeUnpublished {
		builder = builder.Where(sq.Eq{"is_published": true})
	}
	query, args, errQueryArgs := builder.Limit(uint64(limit)).ToSql()
	if errQueryArgs != nil {
		return nil, Err(errQueryArgs)
	}
	rows, errQuery := Query(ctx, query, args...)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	for rows.Next() {
		var entry NewsEntry
		if errScan := rows.Scan(&entry.NewsId, &entry.Title, &entry.BodyMD, &entry.IsPublished,
			&entry.CreatedOn, &entry.UpdatedOn); errScan != nil {
			return nil, Err(errScan)
		}
		articles = append(articles, entry)
	}
	return articles, nil
}

func GetNewsLatestArticle(ctx context.Context, includeUnpublished bool, entry *NewsEntry) error {
	builder := sb.Select("news_id", "title", "body_md", "is_published", "created_on", "updated_on").
		From("news")
	if !includeUnpublished {
		builder = builder.Where(sq.Eq{"is_published": true})
	}
	query, args, errQueryArgs := builder.OrderBy("created_on DESC").ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	if errQuery := QueryRow(ctx, query, args...).Scan(&entry.NewsId, &entry.Title, &entry.BodyMD, &entry.IsPublished,
		&entry.CreatedOn, &entry.UpdatedOn); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func GetNewsById(ctx context.Context, newsId int, entry *NewsEntry) error {
	query, args, errQueryArgs := sb.Select("news_id", "title", "body_md", "is_published", "created_on", "updated_on").
		From("news").Where(sq.Eq{"news_id": newsId}).ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	if errQuery := QueryRow(ctx, query, args...).Scan(&entry.NewsId, &entry.Title, &entry.BodyMD, &entry.IsPublished,
		&entry.CreatedOn, &entry.UpdatedOn); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func SaveNewsArticle(ctx context.Context, entry *NewsEntry) error {
	if entry.NewsId > 0 {
		return updateNewsArticle(ctx, entry)
	} else {
		return insertNewsArticle(ctx, entry)
	}
}

func insertNewsArticle(ctx context.Context, entry *NewsEntry) error {
	query, args, errQueryArgs := sb.Insert("news").
		Columns("title", "body_md", "is_published", "created_on", "updated_on").
		Values(entry.Title, entry.BodyMD, entry.IsPublished, entry.CreatedOn, entry.UpdatedOn).
		Suffix("RETURNING news_id").
		ToSql()
	if errQueryArgs != nil {
		return errQueryArgs
	}
	errQueryRow := QueryRow(ctx, query, args...).Scan(&entry.NewsId)
	if errQueryRow != nil {
		return Err(errQueryRow)
	}
	logger.Info("New article saved", zap.String("title", util.SanitizeLog(entry.Title)))
	return nil
}

func updateNewsArticle(ctx context.Context, entry *NewsEntry) error {
	query, args, errQueryArgs := sb.Update("news").
		Set("title", entry.Title).
		Set("body_md", entry.BodyMD).
		Set("is_published", entry.IsPublished).
		Set("updated_on", config.Now()).
		Where(sq.Eq{"news_id": entry.NewsId}).
		ToSql()
	if errQueryArgs != nil {
		return errQueryArgs
	}
	if errExec := Exec(ctx, query, args...); errExec != nil {
		return errors.Wrapf(errExec, "Failed to update article")
	}
	logger.Info("News article updated", zap.String("title", util.SanitizeLog(entry.Title)))
	return nil
}

func DropNewsArticle(ctx context.Context, newsId int) error {
	query, args, errQueryArgs := sb.Delete("news").Where(sq.Eq{"news_id": newsId}).ToSql()
	if errQueryArgs != nil {
		return errQueryArgs
	}
	if errExec := Exec(ctx, query, args...); errExec != nil {
		return Err(errExec)
	}
	logger.Info("News deleted", zap.Int("news_id", newsId))
	return nil
}
