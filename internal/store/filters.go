package store

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

func SaveFilter(ctx context.Context, database Store, filter *model.Filter) error {
	if filter.FilterID > 0 {
		return updateFilter(ctx, database, filter)
	} else {
		return insertFilter(ctx, database, filter)
	}
}

func insertFilter(ctx context.Context, database Store, filter *model.Filter) error {
	const query = `
		INSERT INTO filtered_word (author_id, pattern, is_regex, is_enabled, trigger_count, created_on, updated_on, action, duration, weight) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) 
		RETURNING filter_id`

	if errQuery := database.QueryRow(ctx, query, filter.AuthorID.Int64(), filter.Pattern,
		filter.IsRegex, filter.IsEnabled, filter.TriggerCount, filter.CreatedOn, filter.UpdatedOn, filter.Action, filter.Duration).
		Scan(&filter.FilterID); errQuery != nil {
		return DBErr(errQuery)
	}

	return nil
}

func updateFilter(ctx context.Context, database Store, filter *model.Filter) error {
	query := database.
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

	if err := database.ExecUpdateBuilder(ctx, query); err != nil {
		return DBErr(err)
	}

	return nil
}

func DropFilter(ctx context.Context, database Store, filter *model.Filter) error {
	query := database.
		Builder().
		Delete("filtered_word").
		Where(sq.Eq{"filter_id": filter.FilterID})
	if errExec := database.ExecDeleteBuilder(ctx, query); errExec != nil {
		return DBErr(errExec)
	}

	return nil
}

func GetFilterByID(ctx context.Context, database Store, filterID int64, filter *model.Filter) error {
	query := database.
		Builder().
		Select("filter_id", "author_id", "pattern", "is_regex",
			"is_enabled", "trigger_count", "created_on", "updated_on", "action", "duration", "weight").
		From("filtered_word").
		Where(sq.Eq{"filter_id": filterID})

	row, errQuery := database.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return DBErr(errQuery)
	}

	var authorID int64

	if errScan := row.Scan(&filter.FilterID, &authorID, &filter.Pattern,
		&filter.IsRegex, &filter.IsEnabled, &filter.TriggerCount, &filter.CreatedOn, &filter.UpdatedOn,
		&filter.Action, &filter.Duration, &filter.Weight); errScan != nil {
		return DBErr(errScan)
	}

	filter.AuthorID = steamid.New(authorID)

	filter.Init()

	return nil
}

type FiltersQueryFilter struct {
	QueryFilter
}

func GetFilters(ctx context.Context, database Store, opts FiltersQueryFilter) ([]model.Filter, int64, error) {
	builder := database.
		Builder().
		Select("f.filter_id", "f.author_id", "f.pattern", "f.is_regex",
			"f.is_enabled", "f.trigger_count", "f.created_on", "f.updated_on", "f.action", "f.duration", "f.weight").
		From("filtered_word f")

	builder = opts.QueryFilter.applySafeOrder(builder, map[string][]string{
		"f.": {
			"filter_id", "author_id", "pattern", "is_regex", "is_enabled", "trigger_count",
			"created_on", "updated_on", "action", "duration", "weight",
		},
	}, "filter_id")

	builder = opts.QueryFilter.applyLimitOffset(builder, maxResultsDefault)

	rows, errExec := database.QueryBuilder(ctx, builder)
	if errExec != nil {
		return nil, 0, DBErr(errExec)
	}

	defer rows.Close()

	var filters []model.Filter

	for rows.Next() {
		var (
			filter   model.Filter
			authorID int64
		)

		if errScan := rows.Scan(&filter.FilterID, &authorID, &filter.Pattern, &filter.IsRegex,
			&filter.IsEnabled, &filter.TriggerCount, &filter.CreatedOn, &filter.UpdatedOn, &filter.Action,
			&filter.Duration, &filter.Weight); errScan != nil {
			return nil, 0, DBErr(errScan)
		}

		filter.AuthorID = steamid.New(authorID)

		filter.Init()

		filters = append(filters, filter)
	}

	count, errCount := getCount(ctx, database, database.
		Builder().
		Select("count(filter_id)").
		From("filtered_word f"))
	if errCount != nil {
		return nil, 0, DBErr(errCount)
	}

	return filters, count, nil
}

func AddMessageFilterMatch(ctx context.Context, database Store, messageID int64, filterID int64) error {
	return DBErr(database.ExecInsertBuilder(ctx, database.
		Builder().
		Insert("person_messages_filter").
		Columns("person_message_id", "filter_id").
		Values(messageID, filterID)))
}
