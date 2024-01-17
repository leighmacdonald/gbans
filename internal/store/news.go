package store

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (db *Store) GetNewsLatest(ctx context.Context, limit int, includeUnpublished bool) ([]model.NewsEntry, error) {
	builder := db.sb.Select("news_id", "title", "body_md", "is_published", "created_on", "updated_on").
		From("news").
		OrderBy("created_on DESC").
		Limit(uint64(limit))

	if !includeUnpublished {
		builder = builder.Where(sq.Eq{"is_published": true})
	}

	rows, errQuery := db.QueryBuilder(ctx, builder)
	if errQuery != nil {
		return nil, Err(errQuery)
	}

	defer rows.Close()

	var articles []model.NewsEntry

	for rows.Next() {
		var entry model.NewsEntry
		if errScan := rows.Scan(&entry.NewsID, &entry.Title, &entry.BodyMD, &entry.IsPublished,
			&entry.CreatedOn, &entry.UpdatedOn); errScan != nil {
			return nil, Err(errScan)
		}

		articles = append(articles, entry)
	}

	return articles, nil
}

func (db *Store) GetNewsLatestArticle(ctx context.Context, includeUnpublished bool, entry *model.NewsEntry) error {
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

func (db *Store) GetNewsByID(ctx context.Context, newsID int, entry *model.NewsEntry) error {
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

func (db *Store) SaveNewsArticle(ctx context.Context, entry *model.NewsEntry) error {
	if entry.NewsID > 0 {
		return db.updateNewsArticle(ctx, entry)
	} else {
		return db.insertNewsArticle(ctx, entry)
	}
}

func (db *Store) insertNewsArticle(ctx context.Context, entry *model.NewsEntry) error {
	query, args, errQueryArgs := db.sb.Insert("news").
		Columns("title", "body_md", "is_published", "created_on", "updated_on").
		Values(entry.Title, entry.BodyMD, entry.IsPublished, entry.CreatedOn, entry.UpdatedOn).
		Suffix("RETURNING news_id").
		ToSql()
	if errQueryArgs != nil {
		return errors.Wrapf(errQueryArgs, "Failed to create query")
	}

	errQueryRow := db.QueryRow(ctx, query, args...).Scan(&entry.NewsID)
	if errQueryRow != nil {
		return Err(errQueryRow)
	}

	db.log.Info("New article saved", zap.String("title", util.SanitizeLog(entry.Title)))

	return nil
}

func (db *Store) updateNewsArticle(ctx context.Context, entry *model.NewsEntry) error {
	if errExec := db.ExecUpdateBuilder(ctx, db.sb.
		Update("news").
		Set("title", entry.Title).
		Set("body_md", entry.BodyMD).
		Set("is_published", entry.IsPublished).
		Set("updated_on", time.Now()).
		Where(sq.Eq{"news_id": entry.NewsID})); errExec != nil {
		return errors.Wrapf(errExec, "Failed to update article")
	}

	db.log.Info("News article updated", zap.String("title", util.SanitizeLog(entry.Title)))

	return nil
}

func (db *Store) DropNewsArticle(ctx context.Context, newsID int) error {
	if errExec := db.ExecDeleteBuilder(ctx, db.sb.
		Delete("news").
		Where(sq.Eq{"news_id": newsID})); errExec != nil {
		return Err(errExec)
	}

	db.log.Info("News deleted", zap.Int("news_id", newsID))

	return nil
}
