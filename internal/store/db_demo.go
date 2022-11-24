package store

import (
	"context"
	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/model"
	log "github.com/sirupsen/logrus"
)

func (database *pgStore) GetDemo(ctx context.Context, demoId int64, demoFile *model.DemoFile) error {
	query, args, errQueryArgs := sb.
		Select("demo_id", "server_id", "title", "raw_data", "created_on", "size", "downloads", "map_name", "archive", "stats").
		From("demo").
		Where(sq.Eq{"demo_id": demoId}).ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	if errQuery := database.conn.QueryRow(ctx, query, args...).Scan(&demoFile.DemoID, &demoFile.ServerID, &demoFile.Title, &demoFile.Data,
		&demoFile.CreatedOn, &demoFile.Size, &demoFile.Downloads, &demoFile.MapName, &demoFile.Archive, &demoFile.Stats); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func (database *pgStore) GetDemos(ctx context.Context) ([]model.DemoFile, error) {
	var demos []model.DemoFile
	query, args, errQueryArgs := sb.
		Select("demo_id", "server_id", "title", "created_on", "size", "downloads", "map_name", "archive", "stats").
		From("demo").
		OrderBy("created_on DESC").
		Limit(1000).
		ToSql()
	if errQueryArgs != nil {
		return nil, Err(errQueryArgs)
	}
	rows, errQuery := database.conn.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	for rows.Next() {
		var demoFile model.DemoFile
		if errScan := rows.Scan(&demoFile.DemoID, &demoFile.ServerID, &demoFile.Title, &demoFile.CreatedOn,
			&demoFile.Size, &demoFile.Downloads, &demoFile.MapName, &demoFile.Archive, &demoFile.Stats); errScan != nil {
			return nil, Err(errQuery)
		}
		demos = append(demos, demoFile)
	}
	return demos, nil
}

func (database *pgStore) SaveDemo(ctx context.Context, demoFile *model.DemoFile) error {
	if demoFile.DemoID > 0 {
		return database.updateDemo(ctx, demoFile)
	} else {
		return database.insertDemo(ctx, demoFile)
	}
}

func (database *pgStore) insertDemo(ctx context.Context, demoFile *model.DemoFile) error {
	query, args, errQueryArgs := sb.
		Insert(string(tableDemo)).
		Columns("server_id", "title", "raw_data", "created_on", "size", "downloads", "map_name", "archive", "stats").
		Values(demoFile.ServerID, demoFile.Title, demoFile.Data, demoFile.CreatedOn,
			demoFile.Size, demoFile.Downloads, demoFile.MapName, demoFile.Archive, demoFile.Stats).
		Suffix("RETURNING demo_id").
		ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	errQuery := database.conn.QueryRow(ctx, query, args...).Scan(&demoFile.ServerID)
	if errQuery != nil {
		return Err(errQuery)
	}
	log.Debugf("New demo saved: %s", demoFile.Title)
	return nil
}

func (database *pgStore) updateDemo(ctx context.Context, demoFile *model.DemoFile) error {
	query, args, errQueryArgs := sb.
		Update(string(tableDemo)).
		Set("title", demoFile.Title).
		Set("size", demoFile.Size).
		Set("downloads", demoFile.Downloads).
		Set("map_name", demoFile.MapName).
		Set("archive", demoFile.Archive).
		Set("stats", demoFile.Stats).
		Where(sq.Eq{"server_id": demoFile.ServerID}).
		ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	if _, errExec := database.conn.Exec(ctx, query, args...); errExec != nil {
		return Err(errExec)
	}
	log.Debugf("Demo updated: %s", demoFile.Title)
	return nil
}

func (database *pgStore) DropDemo(ctx context.Context, demoFile *model.DemoFile) error {
	query, args, errQueryArgs := sb.
		Delete(string(tableDemo)).Where(sq.Eq{"demo_id": demoFile.DemoID}).ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	if _, errExec := database.conn.Exec(ctx, query, args...); errExec != nil {
		return Err(errExec)
	}
	demoFile.DemoID = 0
	log.Debugf("Demo deleted: %s", demoFile.Title)
	return nil
}
