package store

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

func (s Stores) SaveFilter(ctx context.Context, filter *model.Filter) error {
	if filter.FilterID > 0 {
		return s.updateFilter(ctx, filter)
	} else {
		return s.insertFilter(ctx, filter)
	}
}

func (s Stores) insertFilter(ctx context.Context, filter *model.Filter) error {
	const query = `
		INSERT INTO filtered_word (author_id, pattern, is_regex, is_enabled, trigger_count, created_on, updated_on, action, duration, weight) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) 
		RETURNING filter_id`

	if errQuery := s.QueryRow(ctx, query, filter.AuthorID.Int64(), filter.Pattern,
		filter.IsRegex, filter.IsEnabled, filter.TriggerCount, filter.CreatedOn, filter.UpdatedOn, filter.Action, filter.Duration).
		Scan(&filter.FilterID); errQuery != nil {
		return errs.DBErr(errQuery)
	}

	return nil
}

func (s Stores) updateFilter(ctx context.Context, filter *model.Filter) error {
	query := s.
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

	if err := s.ExecUpdateBuilder(ctx, query); err != nil {
		return errs.DBErr(err)
	}

	return nil
}

func (s Stores) DropFilter(ctx context.Context, filter *model.Filter) error {
	query := s.
		Builder().
		Delete("filtered_word").
		Where(sq.Eq{"filter_id": filter.FilterID})
	if errExec := s.ExecDeleteBuilder(ctx, query); errExec != nil {
		return errs.DBErr(errExec)
	}

	return nil
}

func (s Stores) GetFilterByID(ctx context.Context, filterID int64, filter *model.Filter) error {
	query := s.
		Builder().
		Select("filter_id", "author_id", "pattern", "is_regex",
			"is_enabled", "trigger_count", "created_on", "updated_on", "action", "duration", "weight").
		From("filtered_word").
		Where(sq.Eq{"filter_id": filterID})

	row, errQuery := s.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return errs.DBErr(errQuery)
	}

	var authorID int64

	if errScan := row.Scan(&filter.FilterID, &authorID, &filter.Pattern,
		&filter.IsRegex, &filter.IsEnabled, &filter.TriggerCount, &filter.CreatedOn, &filter.UpdatedOn,
		&filter.Action, &filter.Duration, &filter.Weight); errScan != nil {
		return errs.DBErr(errScan)
	}

	filter.AuthorID = steamid.New(authorID)

	filter.Init()

	return nil
}

func (s Stores) GetFilters(ctx context.Context, opts model.FiltersQueryFilter) ([]model.Filter, int64, error) {
	builder := s.
		Builder().
		Select("s.filter_id", "s.author_id", "s.pattern", "s.is_regex",
			"s.is_enabled", "s.trigger_count", "s.created_on", "s.updated_on", "s.action", "s.duration", "s.weight").
		From("filtered_word s")

	builder = opts.QueryFilter.ApplySafeOrder(builder, map[string][]string{
		"s.": {
			"filter_id", "author_id", "pattern", "is_regex", "is_enabled", "trigger_count",
			"created_on", "updated_on", "action", "duration", "weight",
		},
	}, "filter_id")

	builder = opts.QueryFilter.ApplyLimitOffset(builder, model.MaxResultsDefault)

	rows, errExec := s.QueryBuilder(ctx, builder)
	if errExec != nil {
		return nil, 0, errs.DBErr(errExec)
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
			return nil, 0, errs.DBErr(errScan)
		}

		filter.AuthorID = steamid.New(authorID)

		filter.Init()

		filters = append(filters, filter)
	}

	count, errCount := getCount(ctx, s, s.
		Builder().
		Select("count(filter_id)").
		From("filtered_word s"))
	if errCount != nil {
		return nil, 0, errs.DBErr(errCount)
	}

	return filters, count, nil
}

func (s Stores) AddMessageFilterMatch(ctx context.Context, messageID int64, filterID int64) error {
	return errs.DBErr(s.ExecInsertBuilder(ctx, s.
		Builder().
		Insert("person_messages_filter").
		Columns("person_message_id", "filter_id").
		Values(messageID, filterID)))
}
