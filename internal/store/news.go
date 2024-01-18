package store

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/pkg/errors"
)

func GetNewsLatest(ctx context.Context, database Store, limit int, includeUnpublished bool) ([]model.NewsEntry, error) {
	builder := database.
		Builder().
		Select("news_id", "title", "body_md", "is_published", "created_on", "updated_on").
		From("news").
		OrderBy("created_on DESC").
		Limit(uint64(limit))

	if !includeUnpublished {
		builder = builder.Where(sq.Eq{"is_published": true})
	}

	rows, errQuery := database.QueryBuilder(ctx, builder)
	if errQuery != nil {
		return nil, DBErr(errQuery)
	}

	defer rows.Close()

	var articles []model.NewsEntry

	for rows.Next() {
		var entry model.NewsEntry
		if errScan := rows.Scan(&entry.NewsID, &entry.Title, &entry.BodyMD, &entry.IsPublished,
			&entry.CreatedOn, &entry.UpdatedOn); errScan != nil {
			return nil, DBErr(errScan)
		}

		articles = append(articles, entry)
	}

	return articles, nil
}

func GetNewsLatestArticle(ctx context.Context, database Store, includeUnpublished bool, entry *model.NewsEntry) error {
	builder := database.
		Builder().
		Select("news_id", "title", "body_md", "is_published", "created_on", "updated_on").
		From("news")
	if !includeUnpublished {
		builder = builder.Where(sq.Eq{"is_published": true})
	}

	query, args, errQueryArgs := builder.OrderBy("created_on DESC").ToSql()
	if errQueryArgs != nil {
		return DBErr(errQueryArgs)
	}

	if errQuery := database.QueryRow(ctx, query, args...).Scan(&entry.NewsID, &entry.Title, &entry.BodyMD, &entry.IsPublished,
		&entry.CreatedOn, &entry.UpdatedOn); errQuery != nil {
		return DBErr(errQuery)
	}

	return nil
}

func GetNewsByID(ctx context.Context, database Store, newsID int, entry *model.NewsEntry) error {
	query, args, errQueryArgs := database.
		Builder().
		Select("news_id", "title", "body_md", "is_published", "created_on", "updated_on").
		From("news").Where(sq.Eq{"news_id": newsID}).ToSql()
	if errQueryArgs != nil {
		return DBErr(errQueryArgs)
	}

	if errQuery := database.QueryRow(ctx, query, args...).Scan(&entry.NewsID, &entry.Title, &entry.BodyMD, &entry.IsPublished,
		&entry.CreatedOn, &entry.UpdatedOn); errQuery != nil {
		return DBErr(errQuery)
	}

	return nil
}

func SaveNewsArticle(ctx context.Context, database Store, entry *model.NewsEntry) error {
	if entry.NewsID > 0 {
		return updateNewsArticle(ctx, database, entry)
	} else {
		return insertNewsArticle(ctx, database, entry)
	}
}

func insertNewsArticle(ctx context.Context, database Store, entry *model.NewsEntry) error {
	query, args, errQueryArgs := database.
		Builder().
		Insert("news").
		Columns("title", "body_md", "is_published", "created_on", "updated_on").
		Values(entry.Title, entry.BodyMD, entry.IsPublished, entry.CreatedOn, entry.UpdatedOn).
		Suffix("RETURNING news_id").
		ToSql()
	if errQueryArgs != nil {
		return errors.Wrapf(errQueryArgs, "Failed to create query")
	}

	errQueryRow := database.QueryRow(ctx, query, args...).Scan(&entry.NewsID)
	if errQueryRow != nil {
		return DBErr(errQueryRow)
	}

	return nil
}

func updateNewsArticle(ctx context.Context, database Store, entry *model.NewsEntry) error {
	return DBErr(database.ExecUpdateBuilder(ctx, database.
		Builder().
		Update("news").
		Set("title", entry.Title).
		Set("body_md", entry.BodyMD).
		Set("is_published", entry.IsPublished).
		Set("updated_on", time.Now()).
		Where(sq.Eq{"news_id": entry.NewsID})))
}

func DropNewsArticle(ctx context.Context, database Store, newsID int) error {
	return DBErr(database.ExecDeleteBuilder(ctx, database.
		Builder().
		Delete("news").
		Where(sq.Eq{"news_id": newsID})))
}
