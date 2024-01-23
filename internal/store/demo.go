package store

import (
	"context"
	"errors"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/internal/model"
)

var ErrReportCountQuery = errors.New("failed to get reports count for demo")

func (s Stores) ExpiredDemos(ctx context.Context, limit uint64) ([]model.DemoInfo, error) {
	rows, errRow := s.QueryBuilder(ctx, s.
		Builder().
		Select("s.demo_id", "s.title", "s.asset_id").
		From("demo s").
		Where(sq.NotEq{"s.archive": true}).
		OrderBy("s.created_on desc").
		Offset(limit))
	if errRow != nil {
		return nil, errs.DBErr(errRow)
	}

	defer rows.Close()

	var demos []model.DemoInfo

	for rows.Next() {
		var demo model.DemoInfo
		if err := rows.Scan(&demo.DemoID, &demo.Title, &demo.AssetID); err != nil {
			return nil, errs.DBErr(err)
		}

		demos = append(demos, demo)
	}

	return demos, nil
}

func (s Stores) GetDemoByID(ctx context.Context, demoID int64, demoFile *model.DemoFile) error {
	row, errRow := s.QueryRowBuilder(ctx, s.
		Builder().
		Select("s.demo_id", "s.server_id", "s.title", "s.created_on", "s.downloads",
			"s.map_name", "s.archive", "s.stats", "s.asset_id", "a.size", "s.short_name", "s.name").
		From("demo s").
		LeftJoin("server s ON s.server_id = s.server_id").
		LeftJoin("asset a ON a.asset_id = s.asset_id").
		Where(sq.Eq{"demo_id": demoID}))
	if errRow != nil {
		return errs.DBErr(errRow)
	}

	var uuidScan *uuid.UUID

	if errQuery := row.Scan(&demoFile.DemoID, &demoFile.ServerID, &demoFile.Title,
		&demoFile.CreatedOn, &demoFile.Downloads, &demoFile.MapName,
		&demoFile.Archive, &demoFile.Stats, uuidScan, &demoFile.Size, &demoFile.ServerNameShort,
		&demoFile.ServerNameLong); errQuery != nil {
		return errs.DBErr(errQuery)
	}

	if uuidScan != nil {
		demoFile.AssetID = *uuidScan
	}

	return nil
}

func (s Stores) GetDemoByName(ctx context.Context, demoName string, demoFile *model.DemoFile) error {
	row, errRow := s.QueryRowBuilder(ctx, s.
		Builder().
		Select("s.demo_id", "s.server_id", "s.title", "s.created_on", "s.downloads",
			"s.map_name", "s.archive", "s.stats", "s.asset_id", "a.size", "s.short_name", "s.name").
		From("demo s").
		LeftJoin("server s ON s.server_id = s.server_id").
		LeftJoin("asset a ON a.asset_id = s.asset_id").
		Where(sq.Eq{"title": demoName}))
	if errRow != nil {
		return errs.DBErr(errRow)
	}

	var uuidScan *uuid.UUID

	if errQuery := row.Scan(&demoFile.DemoID, &demoFile.ServerID, &demoFile.Title,
		&demoFile.CreatedOn, &demoFile.Downloads, &demoFile.MapName, &demoFile.Archive, &demoFile.Stats,
		&demoFile.Stats, &uuidScan, &demoFile.Size, &demoFile.ServerNameShort,
		&demoFile.ServerNameLong); errQuery != nil {
		return errs.DBErr(errQuery)
	}

	if uuidScan != nil {
		demoFile.AssetID = *uuidScan
	}

	return nil
}

func (s Stores) GetDemos(ctx context.Context, opts model.DemoFilter) ([]model.DemoFile, int64, error) {
	var (
		demos       []model.DemoFile
		constraints sq.And
	)

	builder := s.
		Builder().
		Select("s.demo_id", "s.server_id", "s.title", "s.created_on", "s.downloads",
			"s.map_name", "s.archive", "s.stats", "s.short_name", "s.name", "s.asset_id", "a.size").
		From("demo s").
		LeftJoin("server s ON s.server_id = s.server_id").
		LeftJoin("asset a ON a.asset_id = s.asset_id")

	if opts.MapName != "" {
		constraints = append(constraints, sq.ILike{"s.map_name": "%" + strings.ToLower(opts.MapName) + "%"})
	}

	if opts.SteamID != "" {
		sid64, errSid := opts.SteamID.SID64(ctx)
		if errSid != nil {
			return nil, 0, errs.ErrInvalidSID
		}

		constraints = append(constraints, sq.Expr("s.stats ?? ?", sid64.String()))
	}

	if len(opts.ServerIds) > 0 && opts.ServerIds[0] != 0 {
		anyServer := false

		for _, serverID := range opts.ServerIds {
			if serverID == 0 {
				anyServer = true

				break
			}
		}

		if !anyServer {
			constraints = append(constraints, sq.Eq{"s.server_id": opts.ServerIds})
		}
	}

	builder = opts.ApplySafeOrder(builder, map[string][]string{
		"d.": {"demo_id", "server_id", "title", "created_on", "downloads", "map_name"},
		"s.": {"short_name", "name"},
		"a.": {"size"},
	}, "demo_id")

	builder = opts.ApplyLimitOffsetDefault(builder).Where(constraints)

	rows, errQuery := s.QueryBuilder(ctx, builder)
	if errQuery != nil {
		if errors.Is(errQuery, errs.ErrNoResult) {
			return demos, 0, nil
		}

		return nil, 0, errs.DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			demoFile model.DemoFile
			uuidScan *uuid.UUID // TODO remove this and make column not-null once migrations are complete
		)

		if errScan := rows.Scan(&demoFile.DemoID, &demoFile.ServerID, &demoFile.Title, &demoFile.CreatedOn,
			&demoFile.Downloads, &demoFile.MapName, &demoFile.Archive, &demoFile.Stats,
			&demoFile.ServerNameShort, &demoFile.ServerNameLong, &uuidScan, &demoFile.Size); errScan != nil {
			return nil, 0, errs.DBErr(errScan)
		}

		if uuidScan != nil {
			demoFile.AssetID = *uuidScan
		}

		demos = append(demos, demoFile)
	}

	if demos == nil {
		return []model.DemoFile{}, 0, nil
	}

	count, errCount := getCount(ctx, s, s.
		Builder().
		Select("count(s.demo_id)").
		From("demo s").
		Where(constraints))
	if errCount != nil {
		return []model.DemoFile{}, 0, errs.DBErr(errCount)
	}

	return demos, count, nil
}

func (s Stores) SaveDemo(ctx context.Context, demoFile *model.DemoFile) error {
	// Find open reports and if any are returned, mark the demo as archived so that it does not get auto
	// deleted during cleanup.
	// Reports can happen mid-game which is why this is checked when the demo is saved and not during the report where
	// we have no completed demo instance/id yet.
	reportRow, reportRowErr := s.QueryRowBuilder(ctx, s.
		Builder().
		Select("count(report_id)").
		From("report").
		Where(sq.Eq{"demo_name": demoFile.Title}))
	if reportRowErr != nil {
		return errors.Join(reportRowErr, ErrReportCountQuery)
	}

	var count int
	if errScan := reportRow.Scan(&count); errScan != nil && !errors.Is(errScan, errs.ErrNoResult) {
		return errs.DBErr(errScan)
	}

	if count > 0 {
		demoFile.Archive = true
	}

	var err error
	if demoFile.DemoID > 0 {
		err = s.updateDemo(ctx, demoFile)
	} else {
		err = s.insertDemo(ctx, demoFile)
	}

	return errs.DBErr(err)
}

func (s Stores) insertDemo(ctx context.Context, demoFile *model.DemoFile) error {
	query, args, errQueryArgs := s.
		Builder().
		Insert("demo").
		Columns("server_id", "title", "created_on", "downloads", "map_name", "archive", "stats", "asset_id").
		Values(demoFile.ServerID, demoFile.Title, demoFile.CreatedOn,
			demoFile.Downloads, demoFile.MapName, demoFile.Archive, demoFile.Stats, demoFile.AssetID).
		Suffix("RETURNING demo_id").
		ToSql()
	if errQueryArgs != nil {
		return errs.DBErr(errQueryArgs)
	}

	errQuery := s.QueryRow(ctx, query, args...).Scan(&demoFile.ServerID)
	if errQuery != nil {
		return errs.DBErr(errQuery)
	}

	return nil
}

func (s Stores) updateDemo(ctx context.Context, demoFile *model.DemoFile) error {
	query := s.
		Builder().
		Update("demo").
		Set("title", demoFile.Title).
		Set("downloads", demoFile.Downloads).
		Set("map_name", demoFile.MapName).
		Set("archive", demoFile.Archive).
		Set("stats", demoFile.Stats).
		Set("asset_id", demoFile.AssetID).
		Where(sq.Eq{"demo_id": demoFile.DemoID})

	if errExec := s.ExecUpdateBuilder(ctx, query); errExec != nil {
		return errs.DBErr(errExec)
	}

	return nil
}

func (s Stores) DropDemo(ctx context.Context, demoFile *model.DemoFile) error {
	if errExec := s.ExecDeleteBuilder(ctx, s.
		Builder().
		Delete("demo").
		Where(sq.Eq{"demo_id": demoFile.DemoID})); errExec != nil {
		return errs.DBErr(errExec)
	}

	demoFile.DemoID = 0

	return nil
}

func (s Stores) SaveAsset(ctx context.Context, asset *model.Asset) error {
	return errs.DBErr(s.ExecInsertBuilder(ctx, s.
		Builder().
		Insert("asset").
		Columns("asset_id", "bucket", "path", "name", "mime_type", "size", "old_id").
		Values(asset.AssetID, asset.Bucket, asset.Path, asset.Name, asset.MimeType, asset.Size, asset.OldID)))
}
