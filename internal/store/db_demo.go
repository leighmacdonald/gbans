package store

import (
	"context"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func (database *pgStore) GetDemoById(ctx context.Context, demoId int64, demoFile *model.DemoFile) error {
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

func (database *pgStore) GetDemoByName(ctx context.Context, demoName string, demoFile *model.DemoFile) error {
	query, args, errQueryArgs := sb.
		Select("demo_id", "server_id", "title", "raw_data", "created_on", "size", "downloads", "map_name", "archive", "stats").
		From("demo").
		Where(sq.Eq{"title": demoName}).ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	if errQuery := database.conn.QueryRow(ctx, query, args...).Scan(&demoFile.DemoID, &demoFile.ServerID, &demoFile.Title, &demoFile.Data,
		&demoFile.CreatedOn, &demoFile.Size, &demoFile.Downloads, &demoFile.MapName, &demoFile.Archive, &demoFile.Stats); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

type GetDemosOptions struct {
	SteamId   string `json:"steamId"`
	ServerIds []int  `json:"serverIds"`
	MapName   string `json:"mapName"`
}

func (database *pgStore) GetDemos(ctx context.Context, opts GetDemosOptions) ([]model.DemoFile, error) {
	var demos []model.DemoFile
	qb := sb.
		Select("d.demo_id", "d.server_id", "d.title", "d.created_on", "d.size", "d.downloads",
			"d.map_name", "d.archive", "d.stats", "s.short_name", "s.name").
		From("demo d").
		LeftJoin("server s ON s.server_id = d.server_id").
		OrderBy("created_on DESC").
		Limit(1000)
	if opts.MapName != "" {
		qb = qb.Where(sq.Eq{"map_name": opts.MapName})
	}
	if opts.SteamId != "" {
		sid64, errSid := steamid.SID64FromString(opts.SteamId)
		if errSid != nil {
			return nil, consts.ErrInvalidSID
		}
		// FIXME Can this be done with normal parameters + sb?
		qb = qb.Where(fmt.Sprintf("stats @?? '$ ?? (exists (@.\"%d\"))'", sid64.Int64()))
	}
	if len(opts.ServerIds) > 0 {
		// 0 = all
		if opts.ServerIds[0] != 0 {
			qb = qb.Where(sq.Eq{"d.server_id": opts.ServerIds})
		}
	}
	query, args, errQueryArgs := qb.ToSql()
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
			&demoFile.Size, &demoFile.Downloads, &demoFile.MapName, &demoFile.Archive, &demoFile.Stats,
			&demoFile.ServerNameShort, &demoFile.ServerNameLong); errScan != nil {
			return nil, Err(errQuery)
		}
		demos = append(demos, demoFile)
	}
	return demos, nil
}

func (database *pgStore) SaveDemo(ctx context.Context, demoFile *model.DemoFile) error {
	// Find open reports and if any are returned, mark the demo as archived so that it does not get auto
	// deleted during cleanup.
	// Reports can happen mid-game which is why this is checked when the demo is saved and not during the report where
	// we have no completed demo instance/id yet.
	query, args, queryErr := sb.
		Select("count(report_id)").
		From("report").
		Where(sq.Eq{"demo_name": demoFile.Title}).
		ToSql()
	if queryErr != nil {
		return errors.Wrap(queryErr, "Failed to select reports")
	}
	var count int
	if errScan := database.QueryRow(ctx, query, args...).Scan(&count); errScan != nil && !errors.Is(errScan, ErrNoResult) {
		return Err(errScan)
	}
	if count > 0 {
		demoFile.Archive = true
	}
	var err error
	if demoFile.DemoID > 0 {
		err = database.updateDemo(ctx, demoFile)
	} else {
		err = database.insertDemo(ctx, demoFile)
	}
	return Err(err)
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
		Where(sq.Eq{"demo_id": demoFile.DemoID}).
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
