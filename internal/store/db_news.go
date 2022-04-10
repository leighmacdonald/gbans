package store

import (
	"context"
	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func (db *pgStore) GetNewsLatest(ctx context.Context, limit int, includeUnpublished bool) ([]model.NewsEntry, error) {
	var articles []model.NewsEntry
	builder := sb.Select("news_id", "title", "body_md", "is_published", "created_on", "updated_on").
		From("news").
		OrderBy("created_on DESC")
	if !includeUnpublished {
		builder = builder.Where(sq.Eq{"is_published": true})
	}
	q, a, e := builder.Limit(uint64(limit)).ToSql()
	if e != nil {
		return nil, Err(e)
	}
	rows, err := db.c.Query(ctx, q, a...)
	var rs error
	for rows.Next() {
		var entry model.NewsEntry
		if rs = rows.Scan(&entry.NewsId, &entry.Title, &entry.BodyMD, &entry.IsPublished,
			&entry.CreatedOn, &entry.UpdatedOn); rs != nil {
			return nil, Err(err)
		}
		articles = append(articles, entry)
	}
	return articles, nil
}

func (db *pgStore) GetNewsArticle(ctx context.Context, includeUnpublished bool, entry *model.NewsEntry) error {
	builder := sb.Select("news_id", "title", "body_md", "is_published", "created_on", "updated_on").
		From("news")
	if !includeUnpublished {
		builder = builder.Where(sq.Eq{"is_published": true})
	}
	q, a, e := builder.OrderBy("created_on DESC").ToSql()
	if e != nil {
		return Err(e)
	}
	if err := db.c.QueryRow(ctx, q, a...).Scan(&entry.NewsId, &entry.Title, &entry.BodyMD, &entry.IsPublished,
		&entry.CreatedOn, &entry.UpdatedOn); err != nil {
		return Err(err)
	}
	return nil
}

func (db *pgStore) SaveNewsArticle(ctx context.Context, entry *model.NewsEntry) error {
	if entry.NewsId > 0 {
		return db.updateNewsArticle(ctx, entry)
	} else {
		return db.insertNewsArticle(ctx, entry)
	}
}

func (db *pgStore) insertNewsArticle(ctx context.Context, entry *model.NewsEntry) error {
	q, a, e := sb.Insert(string(tableDemo)).
		Columns("title", "body_md", "is_published", "created_on", "updated_on").
		Values(entry.Title, entry.BodyMD, entry.IsPublished, entry.CreatedOn, entry.UpdatedOn).
		Suffix("RETURNING news_id").
		ToSql()
	if e != nil {
		return e
	}
	err := db.c.QueryRow(ctx, q, a...).Scan(&entry.NewsId)
	if err != nil {
		return Err(err)
	}
	log.Debugf("New article saved: %s", entry.Title)
	return nil
}

func (db *pgStore) updateNewsArticle(ctx context.Context, entry *model.NewsEntry) error {
	q, a, e := sb.Update(string(tableDemo)).
		Set("title", entry.Title).
		Set("body_md", entry.BodyMD).
		Set("is_published", entry.IsPublished).
		Set("updated_on", config.Now()).
		Where(sq.Eq{"news_id": entry.NewsId}).
		ToSql()
	if e != nil {
		return e
	}
	if _, err := db.c.Exec(ctx, q, a...); err != nil {
		return errors.Wrapf(err, "Failed to update article")
	}
	log.Debugf("News article updated: %s", entry.Title)
	return nil
}

func (db *pgStore) DropNewsArticle(ctx context.Context, newsId int) error {
	q, a, e := sb.Delete("news").Where(sq.Eq{"news_id": newsId}).ToSql()
	if e != nil {
		return e
	}
	if _, err := db.c.Exec(ctx, q, a...); err != nil {
		return Err(err)
	}
	log.Debugf("News deleted: %d", newsId)
	return nil
}
