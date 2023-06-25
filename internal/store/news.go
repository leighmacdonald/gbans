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
	NewsID      int       `json:"news_id"`
	Title       string    `json:"title"`
	BodyMD      string    `json:"body_md"`
	IsPublished bool      `json:"is_published"`
	CreatedOn   time.Time `json:"created_on,omitempty"`
	UpdatedOn   time.Time `json:"updated_on,omitempty"`
}

func (db *Store) GetNewsLatest(ctx context.Context, limit int, includeUnpublished bool) ([]NewsEntry, error) {
	var articles []NewsEntry
	builder := db.sb.Select("news_id", "title", "body_md", "is_published", "created_on", "updated_on").
		From("news").
		OrderBy("created_on DESC")
	if !includeUnpublished {
		builder = builder.Where(sq.Eq{"is_published": true})
	}
	query, args, errQueryArgs := builder.Limit(uint64(limit)).ToSql()
	if errQueryArgs != nil {
		return nil, Err(errQueryArgs)
	}
	rows, errQuery := db.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	for rows.Next() {
		var entry NewsEntry
		if errScan := rows.Scan(&entry.NewsID, &entry.Title, &entry.BodyMD, &entry.IsPublished,
			&entry.CreatedOn, &entry.UpdatedOn); errScan != nil {
			return nil, Err(errScan)
		}
		articles = append(articles, entry)
	}
	return articles, nil
}

func (db *Store) GetNewsLatestArticle(ctx context.Context, includeUnpublished bool, entry *NewsEntry) error {
	builder := db.sb.Select("news_id", "title", "body_md", "is_published", "created_on", "updated_on").
		From("news")
	if !includeUnpublished {
		builder = builder.Where(sq.Eq{"is_published": true})
	}
	query, args, errQueryArgs := builder.OrderBy("created_on DESC").ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	if errQuery := db.QueryRow(ctx, query, args...).Scan(&entry.NewsID, &entry.Title, &entry.BodyMD, &entry.IsPublished,
		&entry.CreatedOn, &entry.UpdatedOn); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func (db *Store) GetNewsByID(ctx context.Context, newsID int, entry *NewsEntry) error {
	query, args, errQueryArgs := db.sb.Select("news_id", "title", "body_md", "is_published", "created_on", "updated_on").
		From("news").Where(sq.Eq{"news_id": newsID}).ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	if errQuery := db.QueryRow(ctx, query, args...).Scan(&entry.NewsID, &entry.Title, &entry.BodyMD, &entry.IsPublished,
		&entry.CreatedOn, &entry.UpdatedOn); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func (db *Store) SaveNewsArticle(ctx context.Context, entry *NewsEntry) error {
	if entry.NewsID > 0 {
		return db.updateNewsArticle(ctx, entry)
	} else {
		return db.insertNewsArticle(ctx, entry)
	}
}

func (db *Store) insertNewsArticle(ctx context.Context, entry *NewsEntry) error {
	query, args, errQueryArgs := db.sb.Insert("news").
		Columns("title", "body_md", "is_published", "created_on", "updated_on").
		Values(entry.Title, entry.BodyMD, entry.IsPublished, entry.CreatedOn, entry.UpdatedOn).
		Suffix("RETURNING news_id").
		ToSql()
	if errQueryArgs != nil {
		return errQueryArgs
	}
	errQueryRow := db.QueryRow(ctx, query, args...).Scan(&entry.NewsID)
	if errQueryRow != nil {
		return Err(errQueryRow)
	}
	db.log.Info("New article saved", zap.String("title", util.SanitizeLog(entry.Title)))
	return nil
}

func (db *Store) updateNewsArticle(ctx context.Context, entry *NewsEntry) error {
	query, args, errQueryArgs := db.sb.Update("news").
		Set("title", entry.Title).
		Set("body_md", entry.BodyMD).
		Set("is_published", entry.IsPublished).
		Set("updated_on", config.Now()).
		Where(sq.Eq{"news_id": entry.NewsID}).
		ToSql()
	if errQueryArgs != nil {
		return errQueryArgs
	}
	if errExec := db.exec(ctx, query, args...); errExec != nil {
		return errors.Wrapf(errExec, "Failed to update article")
	}
	db.log.Info("News article updated", zap.String("title", util.SanitizeLog(entry.Title)))
	return nil
}

func (db *Store) DropNewsArticle(ctx context.Context, newsID int) error {
	query, args, errQueryArgs := db.sb.Delete("news").Where(sq.Eq{"news_id": newsID}).ToSql()
	if errQueryArgs != nil {
		return errQueryArgs
	}
	if errExec := db.exec(ctx, query, args...); errExec != nil {
		return Err(errExec)
	}
	db.log.Info("News deleted", zap.Int("news_id", newsID))
	return nil
}