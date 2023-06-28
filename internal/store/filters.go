package store

import (
	"context"
	"regexp"
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
		return f.Regex.MatchString(value)
	}
	return f.Pattern == value
}

func (db *Store) SaveFilter(ctx context.Context, filter *Filter) error {
	if filter.FilterID > 0 {
		return db.updateFilter(ctx, filter)
	} else {
		return db.insertFilter(ctx, filter)
	}
}

// todo squirrel version, it expects sql.db though...
func (db *Store) insertFilter(ctx context.Context, filter *Filter) error {
	const query = `
		INSERT INTO filtered_word (author_id, pattern, is_regex, is_enabled, trigger_count, created_on, updated_on) 
		VALUES ($1, $2, $3, $4, $5, $6, $7) 
		RETURNING filter_id`
	if errQuery := db.QueryRow(ctx, query, filter.AuthorID, filter.Pattern,
		filter.IsRegex, filter.IsEnabled, filter.TriggerCount, filter.CreatedOn, filter.UpdatedOn).
		Scan(&filter.FilterID); errQuery != nil {
		return Err(errQuery)
	}
	db.log.Info("Created filter", zap.Int64("filter_id", filter.FilterID))
	return nil
}

func (db *Store) updateFilter(ctx context.Context, filter *Filter) error {
	query, args, errQuery := db.sb.Update("filtered_word").
		Set("author_id", filter.AuthorID).
		Set("pattern", filter.Pattern).
		Set("is_regex", filter.IsRegex).
		Set("is_enabled", filter.IsEnabled).
		Set("trigger_count", filter.TriggerCount).
		Set("created_on", filter.CreatedOn).
		Set("updated_on", filter.UpdatedOn).
		Where(sq.Eq{"filter_id": filter.FilterID}).ToSql()
	if errQuery != nil {
		return Err(errQuery)
	}
	if err := db.Exec(ctx, query, args...); err != nil {
		return Err(err)
	}
	db.log.Debug("Updated filter", zap.Int64("filter_id", filter.FilterID))
	return nil
}

func (db *Store) DropFilter(ctx context.Context, filter *Filter) error {
	const query = `DELETE FROM filtered_word WHERE filter_id = $1`
	if errExec := db.Exec(ctx, query, filter.FilterID); errExec != nil {
		return Err(errExec)
	}
	db.log.Info("Deleted filter", zap.Int64("filter_id", filter.FilterID))
	return nil
}

func (db *Store) GetFilterByID(ctx context.Context, wordID int64, f *Filter) error {
	const query = `
		SELECT filter_id, author_id, pattern, is_regex, is_enabled, trigger_count, created_on, updated_on 
		FROM filtered_word 
		WHERE filter_id = $1`
	if errQuery := db.QueryRow(ctx, query, wordID).Scan(&f.FilterID, &f.AuthorID, &f.Pattern,
		&f.IsRegex, &f.IsEnabled, &f.TriggerCount, &f.CreatedOn, &f.UpdatedOn); errQuery != nil {
		return Err(errQuery)
	}
	f.Init()
	return nil
}

func (db *Store) GetFilters(ctx context.Context) ([]Filter, error) {
	const query = `
		SELECT filter_id, author_id, pattern, is_regex, is_enabled, trigger_count, created_on, updated_on
		FROM filtered_word`
	rows, errQuery := db.Query(ctx, query)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	var filters []Filter
	defer rows.Close()
	for rows.Next() {
		var filter Filter
		if errQuery = rows.Scan(&filter.FilterID, &filter.AuthorID, &filter.Pattern, &filter.IsRegex,
			&filter.IsEnabled, &filter.TriggerCount, &filter.CreatedOn, &filter.UpdatedOn); errQuery != nil {
			return nil, Err(errQuery)
		}
		filter.Init()
		filters = append(filters, filter)
	}
	return filters, nil
}
