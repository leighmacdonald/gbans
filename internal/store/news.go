package store

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/internal/model"
)

func (s Stores) GetNewsLatest(ctx context.Context, limit int, includeUnpublished bool) ([]model.NewsEntry, error) {
	builder := s.
		Builder().
		Select("news_id", "title", "body_md", "is_published", "created_on", "updated_on").
		From("news").
		OrderBy("created_on DESC").
		Limit(uint64(limit))

	if !includeUnpublished {
		builder = builder.Where(sq.Eq{"is_published": true})
	}

	rows, errQuery := s.QueryBuilder(ctx, builder)
	if errQuery != nil {
		return nil, errs.DBErr(errQuery)
	}

	defer rows.Close()

	var articles []model.NewsEntry

	for rows.Next() {
		var entry model.NewsEntry
		if errScan := rows.Scan(&entry.NewsID, &entry.Title, &entry.BodyMD, &entry.IsPublished,
			&entry.CreatedOn, &entry.UpdatedOn); errScan != nil {
			return nil, errs.DBErr(errScan)
		}

		articles = append(articles, entry)
	}

	return articles, nil
}

func (s Stores) GetNewsLatestArticle(ctx context.Context, includeUnpublished bool, entry *model.NewsEntry) error {
	builder := s.
		Builder().
		Select("news_id", "title", "body_md", "is_published", "created_on", "updated_on").
		From("news")
	if !includeUnpublished {
		builder = builder.Where(sq.Eq{"is_published": true})
	}

	query, args, errQueryArgs := builder.OrderBy("created_on DESC").ToSql()
	if errQueryArgs != nil {
		return errs.DBErr(errQueryArgs)
	}

	if errQuery := s.QueryRow(ctx, query, args...).Scan(&entry.NewsID, &entry.Title, &entry.BodyMD, &entry.IsPublished,
		&entry.CreatedOn, &entry.UpdatedOn); errQuery != nil {
		return errs.DBErr(errQuery)
	}

	return nil
}

func (s Stores) GetNewsByID(ctx context.Context, newsID int, entry *model.NewsEntry) error {
	query, args, errQueryArgs := s.
		Builder().
		Select("news_id", "title", "body_md", "is_published", "created_on", "updated_on").
		From("news").Where(sq.Eq{"news_id": newsID}).ToSql()
	if errQueryArgs != nil {
		return errs.DBErr(errQueryArgs)
	}

	if errQuery := s.QueryRow(ctx, query, args...).Scan(&entry.NewsID, &entry.Title, &entry.BodyMD, &entry.IsPublished,
		&entry.CreatedOn, &entry.UpdatedOn); errQuery != nil {
		return errs.DBErr(errQuery)
	}

	return nil
}

func (s Stores) SaveNewsArticle(ctx context.Context, entry *model.NewsEntry) error {
	if entry.NewsID > 0 {
		return s.updateNewsArticle(ctx, entry)
	} else {
		return s.insertNewsArticle(ctx, entry)
	}
}

func (s Stores) insertNewsArticle(ctx context.Context, entry *model.NewsEntry) error {
	query, args, errQueryArgs := s.
		Builder().
		Insert("news").
		Columns("title", "body_md", "is_published", "created_on", "updated_on").
		Values(entry.Title, entry.BodyMD, entry.IsPublished, entry.CreatedOn, entry.UpdatedOn).
		Suffix("RETURNING news_id").
		ToSql()
	if errQueryArgs != nil {
		return errors.Join(errQueryArgs, errors.New("Failed to create query"))
	}

	errQueryRow := s.QueryRow(ctx, query, args...).Scan(&entry.NewsID)
	if errQueryRow != nil {
		return errs.DBErr(errQueryRow)
	}

	return nil
}

func (s Stores) updateNewsArticle(ctx context.Context, entry *model.NewsEntry) error {
	return errs.DBErr(s.ExecUpdateBuilder(ctx, s.
		Builder().
		Update("news").
		Set("title", entry.Title).
		Set("body_md", entry.BodyMD).
		Set("is_published", entry.IsPublished).
		Set("updated_on", time.Now()).
		Where(sq.Eq{"news_id": entry.NewsID})))
}

func (s Stores) DropNewsArticle(ctx context.Context, newsID int) error {
	return errs.DBErr(s.ExecDeleteBuilder(ctx, s.
		Builder().
		Delete("news").
		Where(sq.Eq{"news_id": newsID})))
}
