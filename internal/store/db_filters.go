package store

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func (database *pgStore) SaveFilter(ctx context.Context, filter *model.Filter) error {
	if filter.WordID > 0 {
		return database.updateFilter(ctx, filter)
	} else {
		return database.insertFilter(ctx, filter)
	}
}

func (database *pgStore) insertFilter(ctx context.Context, filter *model.Filter) error {
	const query = `
		INSERT INTO filtered_word (word, filter_name, created_on, discord_created_on, discord_id) 
		VALUES ($1, $2, $3, $4, $5) 
		RETURNING word_id`
	if errQuery := database.QueryRow(ctx, query, filter.Patterns.String(), filter.FilterName,
		filter.CreatedOn, filter.DiscordCreatedOn, filter.DiscordId).Scan(&filter.WordID); errQuery != nil {
		return Err(errQuery)
	}
	log.Debugf("Created filter: %d", filter.WordID)
	return nil
}

func (database *pgStore) updateFilter(ctx context.Context, filter *model.Filter) error {
	const query = `
		UPDATE filtered_word SET word = $2, created_on = $3, discord_id = $4, discord_created_on = $5, filter_name = $6
    	WHERE word_id = $1`
	if errQuery := database.Exec(ctx, query, filter.WordID, filter.Patterns.String(),
		filter.CreatedOn, filter.DiscordId, filter.DiscordCreatedOn, filter.FilterName); errQuery != nil {
		return Err(errQuery)
	}
	log.Debugf("Created filter: %d", filter.WordID)
	return nil
}
func (database *pgStore) DropFilter(ctx context.Context, filter *model.Filter) error {
	const query = `DELETE FROM filtered_word WHERE word_id = $1`
	if errExec := database.Exec(ctx, query, filter.WordID); errExec != nil {
		return Err(errExec)
	}
	log.Debugf("Deleted filter: %d", filter.WordID)
	return nil
}

func (database *pgStore) GetFilterByID(ctx context.Context, wordId int64, f *model.Filter) error {
	const query = `SELECT word_id, word, created_on,discord_id, discord_created_on, filter_name 
		FROM filtered_word 
		WHERE word_id = $1`
	var word string
	if errQuery := database.QueryRow(ctx, query, wordId).Scan(&f.WordID, &word, &f.CreatedOn,
		&f.DiscordId, &f.DiscordCreatedOn, &f.FilterName); errQuery != nil {
		return errors.Wrapf(errQuery, "Failed to load filter")
	}
	f.Patterns = model.WordFiltersFromString(word)
	return nil
}

func (database *pgStore) GetFilterByName(ctx context.Context, filterName string, f *model.Filter) error {
	const query = `
		SELECT word_id, word, created_on,discord_id, discord_created_on, filter_name 
		FROM filtered_word 
		WHERE filter_name = $1`
	var word string
	if errQuery := database.QueryRow(ctx, query, filterName).Scan(&f.WordID, &word, &f.CreatedOn,
		&f.DiscordId, &f.DiscordCreatedOn, &f.FilterName); errQuery != nil {
		return errors.Wrapf(errQuery, "Failed to load filter")
	}
	f.Patterns = model.WordFiltersFromString(word)
	return nil
}

func (database *pgStore) GetFilters(ctx context.Context) ([]model.Filter, error) {
	const query = `
		SELECT word_id, word, created_on, discord_id, discord_created_on, filter_name
		FROM filtered_word`
	rows, errQuery := database.Query(ctx, query)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	var filters []model.Filter
	defer rows.Close()
	for rows.Next() {
		var filter model.Filter
		if errQuery = rows.Scan(&filter.WordID, &filter.PatternsString, &filter.CreatedOn, &filter.DiscordId,
			&filter.DiscordCreatedOn, &filter.FilterName); errQuery != nil {
			return nil, errors.Wrapf(errQuery, "Failed to load filter")
		}
		filter.Patterns = model.WordFiltersFromString(filter.PatternsString)
		filters = append(filters, filter)
	}
	return filters, nil
}
