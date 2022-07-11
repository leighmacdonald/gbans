package store

import (
	"context"
	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func (database *pgStore) GetNewsLatest(ctx context.Context, limit int, includeUnpublished bool) ([]model.NewsEntry, error) {
	var articles []model.NewsEntry
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
	rows, errQuery := database.conn.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	for rows.Next() {
		var entry model.NewsEntry
		if errScan := rows.Scan(&entry.NewsId, &entry.Title, &entry.BodyMD, &entry.IsPublished,
			&entry.CreatedOn, &entry.UpdatedOn); errScan != nil {
			return nil, Err(errScan)
		}
		articles = append(articles, entry)
	}
	return articles, nil
}

func (database *pgStore) GetNewsLatestArticle(ctx context.Context, includeUnpublished bool, entry *model.NewsEntry) error {
	builder := sb.Select("news_id", "title", "body_md", "is_published", "created_on", "updated_on").
		From("news")
	if !includeUnpublished {
		builder = builder.Where(sq.Eq{"is_published": true})
	}
	query, args, errQueryArgs := builder.OrderBy("created_on DESC").ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	if errQuery := database.conn.QueryRow(ctx, query, args...).Scan(&entry.NewsId, &entry.Title, &entry.BodyMD, &entry.IsPublished,
		&entry.CreatedOn, &entry.UpdatedOn); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func (database *pgStore) GetNewsById(ctx context.Context, newsId int, entry *model.NewsEntry) error {
	query, args, errQueryArgs := sb.Select("news_id", "title", "body_md", "is_published", "created_on", "updated_on").
		From("news").Where(sq.Eq{"news_id": newsId}).ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	if errQuery := database.conn.QueryRow(ctx, query, args...).Scan(&entry.NewsId, &entry.Title, &entry.BodyMD, &entry.IsPublished,
		&entry.CreatedOn, &entry.UpdatedOn); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func (database *pgStore) SaveNewsArticle(ctx context.Context, entry *model.NewsEntry) error {
	if entry.NewsId > 0 {
		return database.updateNewsArticle(ctx, entry)
	} else {
		return database.insertNewsArticle(ctx, entry)
	}
}

func (database *pgStore) insertNewsArticle(ctx context.Context, entry *model.NewsEntry) error {
	query, args, errQueryArgs := sb.Insert("news").
		Columns("title", "body_md", "is_published", "created_on", "updated_on").
		Values(entry.Title, entry.BodyMD, entry.IsPublished, entry.CreatedOn, entry.UpdatedOn).
		Suffix("RETURNING news_id").
		ToSql()
	if errQueryArgs != nil {
		return errQueryArgs
	}
	errQueryRow := database.conn.QueryRow(ctx, query, args...).Scan(&entry.NewsId)
	if errQueryRow != nil {
		return Err(errQueryRow)
	}
	log.Debugf("New article saved: %s", util.SanitizeLog(entry.Title))
	return nil
}

func (database *pgStore) updateNewsArticle(ctx context.Context, entry *model.NewsEntry) error {
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
	if _, errExec := database.conn.Exec(ctx, query, args...); errExec != nil {
		return errors.Wrapf(errExec, "Failed to update article")
	}
	log.Debugf("News article updated: %s", util.SanitizeLog(entry.Title))
	return nil
}

func (database *pgStore) DropNewsArticle(ctx context.Context, newsId int) error {
	query, args, errQueryArgs := sb.Delete("news").Where(sq.Eq{"news_id": newsId}).ToSql()
	if errQueryArgs != nil {
		return errQueryArgs
	}
	if _, errExec := database.conn.Exec(ctx, query, args...); errExec != nil {
		return Err(errExec)
	}
	log.Debugf("News deleted: %d", newsId)
	return nil
}
