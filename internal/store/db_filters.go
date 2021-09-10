package store

import (
	"context"
	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"regexp"
)

func (db *pgStore) InsertFilter(ctx context.Context, rx string) (*model.Filter, error) {
	r, e := regexp.Compile(rx)
	if e != nil {
		return nil, e
	}
	filter := &model.Filter{
		Word:      r,
		CreatedOn: config.Now(),
	}
	q, a, e := sb.Insert(string(tableFilteredWord)).
		Columns("word", "created_on").
		Values(rx, filter.CreatedOn).
		Suffix("RETURNING word_id").
		ToSql()
	if e != nil {
		return nil, e
	}
	if err := db.c.QueryRow(ctx, q, a...).Scan(&filter.WordID); err != nil {
		return nil, dbErr(err)
	}
	log.Debugf("Created filter: %d", filter.WordID)
	return filter, nil
}

func (db *pgStore) DropFilter(ctx context.Context, filter *model.Filter) error {
	q, a, e := sb.Delete(string(tableFilteredWord)).
		Where(sq.Eq{"word_id": filter.WordID}).
		ToSql()
	if e != nil {
		return dbErr(e)
	}
	if _, err := db.c.Exec(ctx, q, a...); err != nil {
		return dbErr(err)
	}
	log.Debugf("Deleted filter: %d", filter.WordID)
	return nil
}

func (db *pgStore) GetFilterByID(ctx context.Context, wordId int, f *model.Filter) error {
	q, a, e := sb.Select("word_id", "word", "created_on").From(string(tableFilteredWord)).
		Where(sq.Eq{"word_id": wordId}).
		ToSql()
	if e != nil {
		return dbErr(e)
	}
	var w string
	if err := db.c.QueryRow(ctx, q, a...).Scan(&f.WordID, &w, &f.CreatedOn); err != nil {
		return errors.Wrapf(err, "Failed to load filter")
	}
	rx, er := regexp.Compile(w)
	if er != nil {
		return er
	}
	f.Word = rx
	return nil
}

func (db *pgStore) GetFilters(ctx context.Context) ([]*model.Filter, error) {
	q, a, e := sb.Select("word_id", "word", "created_on").From(string(tableFilteredWord)).ToSql()
	if e != nil {
		return nil, dbErr(e)
	}
	rows, err := db.c.Query(ctx, q, a...)
	if err != nil {
		return nil, dbErr(err)
	}
	var filters []*model.Filter
	defer rows.Close()
	for rows.Next() {
		var f model.Filter
		var w string
		if err = rows.Scan(&f.WordID, &w, &f.CreatedOn); err != nil {
			return nil, errors.Wrapf(err, "Failed to load filter")
		}
		rx, er := regexp.Compile(w)
		if er != nil {
			return nil, er
		}
		f.Word = rx
		filters = append(filters, &f)
	}
	return filters, nil
}
