package store

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/srcdsup/srcdsup"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type DemoFile struct {
	DemoID          int64                                 `json:"demo_id"`
	ServerID        int                                   `json:"server_id"`
	ServerNameShort string                                `json:"server_name_short"`
	ServerNameLong  string                                `json:"server_name_long"`
	Title           string                                `json:"title"`
	Data            []byte                                `json:"-"` // Dont send mega data to frontend by accident
	CreatedOn       time.Time                             `json:"created_on"`
	Size            int64                                 `json:"size"`
	Downloads       int64                                 `json:"downloads"`
	MapName         string                                `json:"map_name"`
	Archive         bool                                  `json:"archive"` // When true, will not get auto deleted when flushing old demos
	Stats           map[steamid.SID64]srcdsup.PlayerStats `json:"stats"`
}

// func NewDemoFile(serverId int64, title string, rawData []byte) (DemoFile, error) {
//	size := int64(len(rawData))
//	if size == 0 {
//		return DemoFile{}, errors.New("Empty demo")
//	}
//	return DemoFile{
//		ServerID:  serverId,
//		Title:     title,
//		Data:      rawData,
//		CreatedOn: config.Now(),
//		Size:      size,
//		Downloads: 0,
//	}, nil
//}

func FlushDemos(ctx context.Context) error {
	query, args, errQuery := sb.
		Delete("demo").
		Where(sq.And{
			sq.Eq{"archive": false},
			sq.Lt{"created_on": config.Now().Add(-(time.Hour * 24 * 14))},
		}).ToSql()
	if errQuery != nil {
		return errQuery
	}
	return Err(Exec(ctx, query, args...))
}

func GetDemoByID(ctx context.Context, demoID int64, demoFile *DemoFile) error {
	query, args, errQueryArgs := sb.
		Select("demo_id", "server_id", "title", "raw_data", "created_on", "size", "downloads", "map_name", "archive", "stats").
		From("demo").
		Where(sq.Eq{"demo_id": demoID}).
		ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	if errQuery := QueryRow(ctx, query, args...).Scan(&demoFile.DemoID, &demoFile.ServerID, &demoFile.Title, &demoFile.Data,
		&demoFile.CreatedOn, &demoFile.Size, &demoFile.Downloads, &demoFile.MapName, &demoFile.Archive, &demoFile.Stats); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func GetDemoByName(ctx context.Context, demoName string, demoFile *DemoFile) error {
	query, args, errQueryArgs := sb.
		Select("demo_id", "server_id", "title", "raw_data", "created_on", "size", "downloads", "map_name", "archive", "stats").
		From("demo").
		Where(sq.Eq{"title": demoName}).
		ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	if errQuery := QueryRow(ctx, query, args...).Scan(&demoFile.DemoID, &demoFile.ServerID, &demoFile.Title, &demoFile.Data,
		&demoFile.CreatedOn, &demoFile.Size, &demoFile.Downloads, &demoFile.MapName, &demoFile.Archive, &demoFile.Stats); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

type GetDemosOptions struct {
	SteamID   string `json:"steamId"`
	ServerIds []int  `json:"serverIds"`
	MapName   string `json:"mapName"`
}

func GetDemos(ctx context.Context, opts GetDemosOptions) ([]DemoFile, error) {
	var demos []DemoFile
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
	if opts.SteamID != "" {
		sid64, errSid := steamid.SID64FromString(opts.SteamID)
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
	rows, errQuery := Query(ctx, query, args...)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	for rows.Next() {
		var demoFile DemoFile
		if errScan := rows.Scan(&demoFile.DemoID, &demoFile.ServerID, &demoFile.Title, &demoFile.CreatedOn,
			&demoFile.Size, &demoFile.Downloads, &demoFile.MapName, &demoFile.Archive, &demoFile.Stats,
			&demoFile.ServerNameShort, &demoFile.ServerNameLong); errScan != nil {
			return nil, Err(errQuery)
		}
		demos = append(demos, demoFile)
	}
	return demos, nil
}

func SaveDemo(ctx context.Context, demoFile *DemoFile) error {
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
	if errScan := QueryRow(ctx, query, args...).Scan(&count); errScan != nil && !errors.Is(errScan, ErrNoResult) {
		return Err(errScan)
	}
	if count > 0 {
		demoFile.Archive = true
	}
	var err error
	if demoFile.DemoID > 0 {
		err = updateDemo(ctx, demoFile)
	} else {
		err = insertDemo(ctx, demoFile)
	}
	return Err(err)
}

func insertDemo(ctx context.Context, demoFile *DemoFile) error {
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
	errQuery := QueryRow(ctx, query, args...).Scan(&demoFile.ServerID)
	if errQuery != nil {
		return Err(errQuery)
	}
	logger.Info("New demo saved", zap.String("name", demoFile.Title))
	return nil
}

func updateDemo(ctx context.Context, demoFile *DemoFile) error {
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
	if errExec := Exec(ctx, query, args...); errExec != nil {
		return Err(errExec)
	}
	logger.Info("Demo updated", zap.String("name", demoFile.Title))
	return nil
}

func DropDemo(ctx context.Context, demoFile *DemoFile) error {
	query, args, errQueryArgs := sb.
		Delete(string(tableDemo)).Where(sq.Eq{"demo_id": demoFile.DemoID}).ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}
	if errExec := Exec(ctx, query, args...); errExec != nil {
		return Err(errExec)
	}
	demoFile.DemoID = 0
	logger.Info("Demo deleted:", zap.String("name", demoFile.Title))
	return nil
}
