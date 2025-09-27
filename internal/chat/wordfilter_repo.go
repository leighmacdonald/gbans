package chat

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type WordFilterRepository struct {
	db database.Database
}

func NewWordFilterRepository(database database.Database) WordFilterRepository {
	return WordFilterRepository{db: database}
}

func (r *WordFilterRepository) SaveFilter(ctx context.Context, filter *Filter) error {
	if filter.FilterID > 0 {
		return r.updateFilter(ctx, filter)
	}

	return r.insertFilter(ctx, filter)
}

func (r *WordFilterRepository) insertFilter(ctx context.Context, filter *Filter) error {
	const query = `
		INSERT INTO filtered_word (author_id, pattern, is_regex, is_enabled, trigger_count, created_on, updated_on, action, duration, weight)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING filter_id`

	if errQuery := r.db.QueryRow(ctx, nil, query, filter.AuthorID.Int64(), filter.Pattern,
		filter.IsRegex, filter.IsEnabled, filter.TriggerCount, filter.CreatedOn, filter.UpdatedOn, filter.Action, filter.Duration, filter.Weight).
		Scan(&filter.FilterID); errQuery != nil {
		return database.DBErr(errQuery)
	}

	return nil
}

func (r *WordFilterRepository) updateFilter(ctx context.Context, filter *Filter) error {
	query := r.db.
		Builder().
		Update("filtered_word").
		Set("author_id", filter.AuthorID.Int64()).
		Set("pattern", filter.Pattern).
		Set("is_regex", filter.IsRegex).
		Set("is_enabled", filter.IsEnabled).
		Set("trigger_count", filter.TriggerCount).
		Set("action", filter.Action).
		Set("duration", filter.Duration).
		Set("weight", filter.Weight).
		Set("created_on", filter.CreatedOn).
		Set("updated_on", filter.UpdatedOn).
		Where(sq.Eq{"filter_id": filter.FilterID})

	if err := r.db.ExecUpdateBuilder(ctx, nil, query); err != nil {
		return database.DBErr(err)
	}

	return nil
}

func (r *WordFilterRepository) DropFilter(ctx context.Context, filter Filter) error {
	query := r.db.
		Builder().
		Delete("filtered_word").
		Where(sq.Eq{"filter_id": filter.FilterID})
	if errExec := r.db.ExecDeleteBuilder(ctx, nil, query); errExec != nil {
		return database.DBErr(errExec)
	}

	return nil
}

func (r *WordFilterRepository) GetFilterByID(ctx context.Context, filterID int64) (Filter, error) {
	var filter Filter

	query := r.db.
		Builder().
		Select("filter_id", "author_id", "pattern", "is_regex",
			"is_enabled", "trigger_count", "created_on", "updated_on", "action", "duration", "weight").
		From("filtered_word").
		Where(sq.Eq{"filter_id": filterID})

	row, errQuery := r.db.QueryRowBuilder(ctx, nil, query)
	if errQuery != nil {
		return filter, database.DBErr(errQuery)
	}

	var authorID int64

	if errScan := row.Scan(&filter.FilterID, &authorID, &filter.Pattern,
		&filter.IsRegex, &filter.IsEnabled, &filter.TriggerCount, &filter.CreatedOn, &filter.UpdatedOn,
		&filter.Action, &filter.Duration, &filter.Weight); errScan != nil {
		return filter, database.DBErr(errScan)
	}

	filter.AuthorID = steamid.New(authorID)

	filter.Init()

	return filter, nil
}

func (r *WordFilterRepository) GetFilters(ctx context.Context) ([]Filter, error) {
	builder := r.db.
		Builder().
		Select("r.filter_id", "r.author_id", "r.pattern", "r.is_regex",
			"r.is_enabled", "r.trigger_count", "r.created_on", "r.updated_on", "r.action", "r.duration", "r.weight").
		From("filtered_word r")

	rows, errExec := r.db.QueryBuilder(ctx, nil, builder)
	if errExec != nil {
		return nil, database.DBErr(errExec)
	}

	defer rows.Close()

	var filters []Filter

	for rows.Next() {
		var (
			filter   Filter
			authorID int64
		)

		if errScan := rows.Scan(&filter.FilterID, &authorID, &filter.Pattern, &filter.IsRegex,
			&filter.IsEnabled, &filter.TriggerCount, &filter.CreatedOn, &filter.UpdatedOn, &filter.Action,
			&filter.Duration, &filter.Weight); errScan != nil {
			return nil, database.DBErr(errScan)
		}

		filter.AuthorID = steamid.New(authorID)

		filter.Init()

		filters = append(filters, filter)
	}

	return filters, nil
}

func (r *WordFilterRepository) AddMessageFilterMatch(ctx context.Context, messageID int64, filterID int64) error {
	return database.DBErr(r.db.ExecInsertBuilder(ctx, nil, r.db.Builder().
		Insert("person_messages_filter").
		Columns("person_message_id", "filter_id").
		Values(messageID, filterID)))
}
