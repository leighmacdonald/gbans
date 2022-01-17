package store

import (
	"context"
	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func (db *pgStore) GetDemo(ctx context.Context, demoId int64, d *model.DemoFile) error {
	q, a, e := sb.Select("demo_id", "server_id", "title", "raw_data", "created_on", "size", "downloads").
		From("demo").
		Where(sq.Eq{"demo_id": demoId}).ToSql()
	if e != nil {
		return Err(e)
	}
	if err := db.c.QueryRow(ctx, q, a...).Scan(&d.DemoID, &d.ServerID, &d.Title, &d.Data,
		&d.CreatedOn, &d.Size, &d.Downloads); err != nil {
		return Err(err)
	}
	return nil
}

func (db *pgStore) GetDemos(ctx context.Context) ([]model.DemoFile, error) {
	var demos []model.DemoFile
	q, a, e := sb.Select("demo_id", "server_id", "title", "created_on", "size", "downloads").
		From("demo").
		OrderBy("created_on DESC").
		Limit(1000).
		ToSql()
	if e != nil {
		return nil, Err(e)
	}
	rows, err := db.c.Query(ctx, q, a...)
	var rs error
	for rows.Next() {
		var d model.DemoFile
		if rs = rows.Scan(&d.DemoID, &d.ServerID, &d.Title, &d.CreatedOn, &d.Size, &d.Downloads); rs != nil {
			return nil, Err(err)
		}
		demos = append(demos, d)
	}
	return demos, nil
}

func (db *pgStore) SaveDemo(ctx context.Context, d *model.DemoFile) error {
	if d.ServerID > 0 {
		return db.updateDemo(ctx, d)
	} else {
		return db.insertDemo(ctx, d)
	}
}

func (db *pgStore) insertDemo(ctx context.Context, d *model.DemoFile) error {
	q, a, e := sb.Insert(string(tableDemo)).
		Columns("server_id", "title", "raw_data", "created_on", "size", "downloads").
		Values(d.ServerID, d.Title, d.Data, d.CreatedOn, d.Size, d.Downloads).
		Suffix("RETURNING demo_id").
		ToSql()
	if e != nil {
		return e
	}
	err := db.c.QueryRow(ctx, q, a...).Scan(&d.ServerID)
	if err != nil {
		return Err(err)
	}
	log.Debugf("New demo saved: %s", d.Title)
	return nil
}

func (db *pgStore) updateDemo(ctx context.Context, d *model.DemoFile) error {
	q, a, e := sb.Update(string(tableDemo)).
		Set("title", d.Title).
		Set("size", d.Size).
		Set("downloads", d.Downloads).
		Where(sq.Eq{"server_id": d.ServerID}).
		ToSql()
	if e != nil {
		return e
	}
	if _, err := db.c.Exec(ctx, q, a...); err != nil {
		return errors.Wrapf(err, "Failed to update demo")
	}
	log.Debugf("Demo updated: %s", d.Title)
	return nil
}

func (db *pgStore) DropDemo(ctx context.Context, d *model.DemoFile) error {
	q, a, e := sb.Delete(string(tableDemo)).Where(sq.Eq{"demo_id": d.DemoID}).ToSql()
	if e != nil {
		return e
	}
	if _, err := db.c.Exec(ctx, q, a...); err != nil {
		return Err(err)
	}
	d.DemoID = 0
	log.Debugf("Demo deleted: %s", d.Title)
	return nil
}
