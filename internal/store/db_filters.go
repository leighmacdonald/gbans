package store

import (
	"context"
	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/model"
	log "github.com/sirupsen/logrus"
)

func (database *pgStore) SaveFilter(ctx context.Context, filter *model.Filter) error {
	if filter.FilterID > 0 {
		return database.updateFilter(ctx, filter)
	} else {
		return database.insertFilter(ctx, filter)
	}
}

// todo squirrel version, it expects sql.db though...
func (database *pgStore) insertFilter(ctx context.Context, filter *model.Filter) error {
	const query = `
		INSERT INTO filtered_word (author_id, pattern, is_regex, is_enabled, trigger_count, created_on, updated_on) 
		VALUES ($1, $2, $3, $4, $5, $6) 
		RETURNING filter_id`
	if errQuery := database.QueryRow(ctx, query, filter.AuthorId, filter.Pattern,
		filter.IsRegex, filter.IsEnabled, filter.TriggerCount, filter.CreatedOn, filter.UpdatedOn).
		Scan(&filter.FilterID); errQuery != nil {
		return Err(errQuery)
	}
	log.Debugf("Created filter: %d", filter.FilterID)
	return nil
}

func (database *pgStore) updateFilter(ctx context.Context, filter *model.Filter) error {
	query, args, errQuery := sb.Update("filtered_word").
		Set("author_id", filter.AuthorId).
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
	if err := database.Exec(ctx, query, args...); err != nil {
		return Err(err)
	}
	log.Debugf("Created filter: %d", filter.FilterID)
	return nil
}
func (database *pgStore) DropFilter(ctx context.Context, filter *model.Filter) error {
	const query = `DELETE FROM filtered_word WHERE filter_id = $1`
	if errExec := database.Exec(ctx, query, filter.FilterID); errExec != nil {
		return Err(errExec)
	}
	log.WithFields(log.Fields{"filter_id": filter.FilterID}).Infof("Deleted filter")
	return nil
}

func (database *pgStore) GetFilterByID(ctx context.Context, wordId int64, f *model.Filter) error {
	const query = `
		SELECT filter_id, author_id, pattern, is_regex, is_enabled, trigger_count, created_on, updated_on 
		FROM filtered_word 
		WHERE filter_id = $1`
	if errQuery := database.QueryRow(ctx, query, wordId).Scan(&f.FilterID, &f.AuthorId, &f.Pattern,
		&f.IsRegex, &f.IsEnabled, &f.TriggerCount, &f.CreatedOn, &f.UpdatedOn); errQuery != nil {
		return Err(errQuery)
	}
	f.Init()
	return nil
}

func (database *pgStore) GetFilters(ctx context.Context) ([]model.Filter, error) {
	const query = `
		SELECT filter_id, author_id, pattern, is_regex, is_enabled, trigger_count, created_on, updated_on
		FROM filtered_word`
	rows, errQuery := database.Query(ctx, query)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	var filters []model.Filter
	defer rows.Close()
	for rows.Next() {
		var filter model.Filter
		if errQuery = rows.Scan(&filter.FilterID, &filter.AuthorId, &filter.Pattern, &filter.IsRegex,
			&filter.IsEnabled, &filter.TriggerCount, &filter.CreatedOn, &filter.UpdatedOn); errQuery != nil {
			return nil, Err(errQuery)
		}
		filter.Init()
		filters = append(filters, filter)
	}
	return filters, nil
}
