package news

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
)

type newsRepository struct {
	db database.Database
}

func NewNewsRepository(database database.Database) NewsRepository {
	return &newsRepository{db: database}
}

func (r newsRepository) GetNewsLatest(ctx context.Context, limit int, includeUnpublished bool) ([]NewsEntry, error) {
	builder := r.db.
		Builder().
		Select("news_id", "title", "body_md", "is_published", "created_on", "updated_on").
		From("news").
		OrderBy("created_on DESC").
		Limit(uint64(limit)) //nolint:gosec

	if !includeUnpublished {
		builder = builder.Where(sq.Eq{"is_published": true})
	}

	rows, errQuery := r.db.QueryBuilder(ctx, nil, builder)
	if errQuery != nil {
		return nil, database.DBErr(errQuery)
	}

	defer rows.Close()

	//goland:noinspection GoPreferNilSlice
	articles := []NewsEntry{}

	for rows.Next() {
		var entry NewsEntry
		if errScan := rows.Scan(&entry.NewsID, &entry.Title, &entry.BodyMD, &entry.IsPublished,
			&entry.CreatedOn, &entry.UpdatedOn); errScan != nil {
			return nil, database.DBErr(errScan)
		}

		articles = append(articles, entry)
	}

	return articles, nil
}

func (r newsRepository) GetNewsLatestArticle(ctx context.Context, includeUnpublished bool, entry *NewsEntry) error {
	builder := r.db.
		Builder().
		Select("news_id", "title", "body_md", "is_published", "created_on", "updated_on").
		From("news")
	if !includeUnpublished {
		builder = builder.Where(sq.Eq{"is_published": true})
	}

	query, args, errQueryArgs := builder.OrderBy("created_on DESC").ToSql()
	if errQueryArgs != nil {
		return database.DBErr(errQueryArgs)
	}

	if errQuery := r.db.QueryRow(ctx, nil, query, args...).Scan(&entry.NewsID, &entry.Title, &entry.BodyMD, &entry.IsPublished,
		&entry.CreatedOn, &entry.UpdatedOn); errQuery != nil {
		return database.DBErr(errQuery)
	}

	return nil
}

func (r newsRepository) GetNewsByID(ctx context.Context, newsID int, entry *NewsEntry) error {
	query, args, errQueryArgs := r.db.
		Builder().
		Select("news_id", "title", "body_md", "is_published", "created_on", "updated_on").
		From("news").Where(sq.Eq{"news_id": newsID}).ToSql()
	if errQueryArgs != nil {
		return database.DBErr(errQueryArgs)
	}

	if errQuery := r.db.QueryRow(ctx, nil, query, args...).Scan(&entry.NewsID, &entry.Title, &entry.BodyMD, &entry.IsPublished,
		&entry.CreatedOn, &entry.UpdatedOn); errQuery != nil {
		return database.DBErr(errQuery)
	}

	return nil
}

func (r newsRepository) Save(ctx context.Context, entry *NewsEntry) error {
	if entry.NewsID > 0 {
		return r.updateNewsArticle(ctx, entry)
	}

	return r.insertNewsArticle(ctx, entry)
}

func (r newsRepository) insertNewsArticle(ctx context.Context, entry *NewsEntry) error {
	query, args, errQueryArgs := r.db.
		Builder().
		Insert("news").
		Columns("title", "body_md", "is_published", "created_on", "updated_on").
		Values(entry.Title, entry.BodyMD, entry.IsPublished, entry.CreatedOn, entry.UpdatedOn).
		Suffix("RETURNING news_id").
		ToSql()
	if errQueryArgs != nil {
		return errors.Join(errQueryArgs, database.ErrCreateQuery)
	}

	errQueryRow := r.db.QueryRow(ctx, nil, query, args...).Scan(&entry.NewsID)
	if errQueryRow != nil {
		return database.DBErr(errQueryRow)
	}

	return nil
}

func (r newsRepository) updateNewsArticle(ctx context.Context, entry *NewsEntry) error {
	return database.DBErr(r.db.ExecUpdateBuilder(ctx, nil, r.db.
		Builder().
		Update("news").
		Set("title", entry.Title).
		Set("body_md", entry.BodyMD).
		Set("is_published", entry.IsPublished).
		Set("updated_on", time.Now()).
		Where(sq.Eq{"news_id": entry.NewsID})))
}

func (r newsRepository) DropNewsArticle(ctx context.Context, newsID int) error {
	return database.DBErr(r.db.ExecDeleteBuilder(ctx, nil, r.db.
		Builder().
		Delete("news").
		Where(sq.Eq{"news_id": newsID})))
}
