package store

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"regexp"
)

func (database *pgStore) SaveFilter(ctx context.Context, filter *model.Filter) error {
	_, errInsert := database.insertFilter(ctx, filter)
	return errInsert
}

func (database *pgStore) insertFilter(ctx context.Context, filter *model.Filter) (*model.Filter, error) {
	const query = `INSERT INTO filtered_word (word, created_on) VALUES ($1, $2) RETURNING word_id`
	if errQuery := database.QueryRow(ctx, query, filter.Pattern.String(), filter.CreatedOn).Scan(&filter.WordID); errQuery != nil {
		return nil, Err(errQuery)
	}
	log.Debugf("Created filter: %d", filter.WordID)
	return filter, nil
}

func (database *pgStore) DropFilter(ctx context.Context, filter *model.Filter) error {
	const query = `DELETE FROM filtered_word WHERE word_id = $1`
	if errExec := database.Exec(ctx, query, filter.WordID); errExec != nil {
		return Err(errExec)
	}
	log.Debugf("Deleted filter: %d", filter.WordID)
	return nil
}

func (database *pgStore) GetFilterByID(ctx context.Context, wordId int, f *model.Filter) error {
	const query = `SELECT word_id, word, created_on from filtered_word WHERE word_id = $1`
	var word string
	if errQuery := database.QueryRow(ctx, query, wordId).Scan(&f.WordID, &word, &f.CreatedOn); errQuery != nil {
		return errors.Wrapf(errQuery, "Failed to load filter")
	}
	rx, errCompile := regexp.Compile(word)
	if errCompile != nil {
		return errCompile
	}
	f.Pattern = rx
	return nil
}

func (database *pgStore) GetFilters(ctx context.Context) ([]model.Filter, error) {
	rows, errQuery := database.Query(ctx, `SELECT word_id, word, created_on from filtered_word`)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	var filters []model.Filter
	defer rows.Close()
	for rows.Next() {
		var filter model.Filter
		var pattern string
		if errQuery = rows.Scan(&filter.WordID, &pattern, &filter.CreatedOn); errQuery != nil {
			return nil, errors.Wrapf(errQuery, "Failed to load filter")
		}
		rx, errCompile := regexp.Compile(pattern)
		if errCompile != nil {
			return nil, errCompile
		}
		filter.Pattern = rx
		filters = append(filters, filter)
	}
	return filters, nil
}
