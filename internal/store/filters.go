package store

import (
	"context"
	"regexp"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

type Filter struct {
	FilterID     int64          `json:"filter_id"`
	AuthorID     steamid.SID64  `json:"author_id"`
	Pattern      string         `json:"pattern"`
	IsRegex      bool           `json:"is_regex"`
	IsEnabled    bool           `json:"is_enabled"`
	Regex        *regexp.Regexp `json:"-"`
	TriggerCount int64          `json:"trigger_count"`
	CreatedOn    time.Time      `json:"created_on"`
	UpdatedOn    time.Time      `json:"updated_on"`
}

func (f *Filter) Init() {
	if f.IsRegex {
		f.Regex = regexp.MustCompile(f.Pattern)
	}
}

func (f *Filter) Match(value string) bool {
	if f.IsRegex {
		return f.Regex.MatchString(strings.ToLower(value))
	}

	return f.Pattern == strings.ToLower(value)
}

func (db *Store) SaveFilter(ctx context.Context, filter *Filter) error {
	if filter.FilterID > 0 {
		return db.updateFilter(ctx, filter)
	} else {
		return db.insertFilter(ctx, filter)
	}
}

func (db *Store) insertFilter(ctx context.Context, filter *Filter) error {
	const query = `
		INSERT INTO filtered_word (author_id, pattern, is_regex, is_enabled, trigger_count, created_on, updated_on) 
		VALUES ($1, $2, $3, $4, $5, $6, $7) 
		RETURNING filter_id`

	if errQuery := db.QueryRow(ctx, query, filter.AuthorID.Int64(), filter.Pattern,
		filter.IsRegex, filter.IsEnabled, filter.TriggerCount, filter.CreatedOn, filter.UpdatedOn).
		Scan(&filter.FilterID); errQuery != nil {
		return Err(errQuery)
	}

	db.log.Info("Created filter", zap.Int64("filter_id", filter.FilterID))

	return nil
}

func (db *Store) updateFilter(ctx context.Context, filter *Filter) error {
	query := db.sb.
		Update("filtered_word").
		Set("author_id", filter.AuthorID.Int64()).
		Set("pattern", filter.Pattern).
		Set("is_regex", filter.IsRegex).
		Set("is_enabled", filter.IsEnabled).
		Set("trigger_count", filter.TriggerCount).
		Set("created_on", filter.CreatedOn).
		Set("updated_on", filter.UpdatedOn).
		Where(sq.Eq{"filter_id": filter.FilterID})

	if err := db.ExecUpdateBuilder(ctx, query); err != nil {
		return Err(err)
	}

	db.log.Debug("Updated filter", zap.Int64("filter_id", filter.FilterID))

	return nil
}

func (db *Store) DropFilter(ctx context.Context, filter *Filter) error {
	query := db.sb.
		Delete("filtered_word").
		Where(sq.Eq{"filter_id": filter.FilterID})
	if errExec := db.ExecDeleteBuilder(ctx, query); errExec != nil {
		db.log.Error("Failed to delete filter", zap.Error(errExec))

		return Err(errExec)
	}

	db.log.Info("Deleted filter", zap.Int64("filter_id", filter.FilterID))

	return nil
}

func (db *Store) GetFilterByID(ctx context.Context, filterID int64, filter *Filter) error {
	query := db.sb.
		Select("filter_id", "author_id", "pattern", "is_regex",
			"is_enabled", "trigger_count", "created_on", "updated_on").
		From("filtered_word").
		Where(sq.Eq{"filter_id": filterID})

	row, errQuery := db.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return errQuery
	}

	var authorID int64

	if errScan := row.Scan(&filter.FilterID, &authorID, &filter.Pattern,
		&filter.IsRegex, &filter.IsEnabled, &filter.TriggerCount, &filter.CreatedOn, &filter.UpdatedOn); errScan != nil {
		db.log.Error("Failed to fetch filter", zap.Error(errScan))

		return Err(errScan)
	}

	filter.AuthorID = steamid.New(authorID)

	filter.Init()

	return nil
}

type FiltersQueryFilter struct {
	QueryFilter
}

func (db *Store) GetFilters(ctx context.Context, opts FiltersQueryFilter) ([]Filter, int64, error) {
	builder := db.sb.
		Select("f.filter_id", "f.author_id", "f.pattern", "f.is_regex",
			"f.is_enabled", "f.trigger_count", "f.created_on", "f.updated_on").
		From("filtered_word f")

	builder = opts.QueryFilter.applySafeOrder(builder, map[string][]string{
		"f.": {"filter_id", "author_id", "pattern", "is_regex", "is_enabled", "trigger_count", "created_on", "updated_on"},
	}, "filter_id")

	builder = opts.QueryFilter.applyLimitOffset(builder, maxResultsDefault)

	rows, errExec := db.QueryBuilder(ctx, builder)
	if errExec != nil {
		return nil, 0, Err(errExec)
	}

	defer rows.Close()

	var filters []Filter

	for rows.Next() {
		var (
			filter   Filter
			authorID int64
		)

		if errScan := rows.Scan(&filter.FilterID, &authorID, &filter.Pattern, &filter.IsRegex,
			&filter.IsEnabled, &filter.TriggerCount, &filter.CreatedOn, &filter.UpdatedOn); errScan != nil {
			return nil, 0, Err(errScan)
		}

		filter.AuthorID = steamid.New(authorID)

		filter.Init()

		filters = append(filters, filter)
	}

	count, errCount := db.GetCount(ctx, db.sb.
		Select("count(filter_id)").
		From("filtered_word f"))
	if errCount != nil {
		return nil, 0, errCount
	}

	return filters, count, nil
}

func (db *Store) AddMessageFilterMatch(ctx context.Context, messageID int64, filterID int64) error {
	return db.ExecInsertBuilder(ctx, db.sb.
		Insert("person_messages_filter").
		Columns("person_message_id", "filter_id").
		Values(messageID, filterID))
}
