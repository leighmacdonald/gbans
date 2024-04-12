package demo

import (
	"context"
	"errors"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
)

type demoRepository struct {
	db database.Database
}

func NewDemoRepository(database database.Database) domain.DemoRepository {
	return &demoRepository{db: database}
}

func (r *demoRepository) ExpiredDemos(ctx context.Context, limit uint64) ([]domain.DemoInfo, error) {
	rows, errRow := r.db.QueryBuilder(ctx, r.db.
		Builder().
		Select("d.demo_id", "d.title", "d.asset_id").
		From("demo d").
		Where(sq.NotEq{"d.archive": true}).
		OrderBy("d.created_on desc").
		Offset(limit))
	if errRow != nil {
		return nil, r.db.DBErr(errRow)
	}

	defer rows.Close()

	var demos []domain.DemoInfo

	for rows.Next() {
		var demo domain.DemoInfo
		if err := rows.Scan(&demo.DemoID, &demo.Title, &demo.AssetID); err != nil {
			return nil, r.db.DBErr(err)
		}

		demos = append(demos, demo)
	}

	return demos, nil
}

func (r *demoRepository) GetDemoByID(ctx context.Context, demoID int64, demoFile *domain.DemoFile) error {
	row, errRow := r.db.QueryRowBuilder(ctx, r.db.
		Builder().
		Select("d.demo_id", "d.server_id", "d.title", "d.created_on", "d.downloads",
			"d.map_name", "d.archive", "d.stats", "d.asset_id", "a.size", "s.short_name", "s.name").
		From("demo r").
		LeftJoin("server s ON s.server_id = d.server_id").
		LeftJoin("asset a ON a.asset_id = d.asset_id").
		Where(sq.Eq{"demo_id": demoID}))
	if errRow != nil {
		return r.db.DBErr(errRow)
	}

	var uuidScan *uuid.UUID

	if errQuery := row.Scan(&demoFile.DemoID, &demoFile.ServerID, &demoFile.Title,
		&demoFile.CreatedOn, &demoFile.Downloads, &demoFile.MapName,
		&demoFile.Archive, &demoFile.Stats, uuidScan, &demoFile.Size, &demoFile.ServerNameShort,
		&demoFile.ServerNameLong); errQuery != nil {
		return r.db.DBErr(errQuery)
	}

	if uuidScan != nil {
		demoFile.AssetID = *uuidScan
	}

	return nil
}

func (r *demoRepository) GetDemoByName(ctx context.Context, demoName string, demoFile *domain.DemoFile) error {
	row, errRow := r.db.QueryRowBuilder(ctx, r.db.
		Builder().
		Select("r.demo_id", "r.server_id", "r.title", "r.created_on", "r.downloads",
			"r.map_name", "r.archive", "r.stats", "r.asset_id", "a.size", "s.short_name", "s.name").
		From("demo r").
		LeftJoin("server s ON s.server_id = r.server_id").
		LeftJoin("asset a ON a.asset_id = r.asset_id").
		Where(sq.Eq{"title": demoName}))
	if errRow != nil {
		return r.db.DBErr(errRow)
	}

	var uuidScan *uuid.UUID

	if errQuery := row.Scan(&demoFile.DemoID, &demoFile.ServerID, &demoFile.Title,
		&demoFile.CreatedOn, &demoFile.Downloads, &demoFile.MapName, &demoFile.Archive, &demoFile.Stats,
		&demoFile.Stats, &uuidScan, &demoFile.Size, &demoFile.ServerNameShort,
		&demoFile.ServerNameLong); errQuery != nil {
		return r.db.DBErr(errQuery)
	}

	if uuidScan != nil {
		demoFile.AssetID = *uuidScan
	}

	return nil
}

func (r *demoRepository) GetDemos(ctx context.Context, opts domain.DemoFilter) ([]domain.DemoFile, int64, error) {
	var (
		demos       []domain.DemoFile
		constraints sq.And
	)

	builder := r.db.
		Builder().
		Select("d.demo_id", "d.server_id", "d.title", "d.created_on", "d.downloads",
			"d.map_name", "d.archive", "d.stats", "s.short_name", "s.name", "d.asset_id", "a.size").
		From("demo d").
		LeftJoin("server s ON s.server_id = d.server_id").
		LeftJoin("asset a ON a.asset_id = d.asset_id")

	if opts.MapName != "" {
		constraints = append(constraints, sq.ILike{"d.map_name": "%" + strings.ToLower(opts.MapName) + "%"})
	}

	if sid, ok := opts.SourceSteamID(); ok {
		constraints = append(constraints, sq.Expr("d.stats ?? ?", sid.String()))
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
			constraints = append(constraints, sq.Eq{"d.server_id": opts.ServerIds})
		}
	}

	builder = opts.ApplySafeOrder(builder, map[string][]string{
		"d.": {"demo_id", "server_id", "title", "created_on", "downloads", "map_name"},
		"s.": {"short_name", "name"},
		"a.": {"size"},
	}, "demo_id")

	builder = opts.ApplyLimitOffsetDefault(builder).Where(constraints)

	rows, errQuery := r.db.QueryBuilder(ctx, builder)
	if errQuery != nil {
		if errors.Is(errQuery, domain.ErrNoResult) {
			return demos, 0, nil
		}

		return nil, 0, r.db.DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			demoFile domain.DemoFile
			uuidScan *uuid.UUID // TODO remove this and make column not-null once migrations are complete
		)

		if errScan := rows.Scan(&demoFile.DemoID, &demoFile.ServerID, &demoFile.Title, &demoFile.CreatedOn,
			&demoFile.Downloads, &demoFile.MapName, &demoFile.Archive, &demoFile.Stats,
			&demoFile.ServerNameShort, &demoFile.ServerNameLong, &uuidScan, &demoFile.Size); errScan != nil {
			return nil, 0, r.db.DBErr(errScan)
		}

		if uuidScan != nil {
			demoFile.AssetID = *uuidScan
		}

		demos = append(demos, demoFile)
	}

	if demos == nil {
		return []domain.DemoFile{}, 0, nil
	}

	count, errCount := r.db.GetCount(ctx, r.db.
		Builder().
		Select("count(d.demo_id)").
		From("demo d").
		Where(constraints))
	if errCount != nil {
		return []domain.DemoFile{}, 0, r.db.DBErr(errCount)
	}

	return demos, count, nil
}

func (r *demoRepository) SaveDemo(ctx context.Context, demoFile *domain.DemoFile) error {
	// Find open reports and if any are returned, mark the demo as archived so that it does not get auto
	// deleted during cleanup.
	// Reports can happen mid-game which is why this is checked when the demo is saved and not during the report where
	// we have no completed demo instance/id yet.
	reportRow, reportRowErr := r.db.QueryRowBuilder(ctx, r.db.
		Builder().
		Select("count(report_id)").
		From("report").
		Where(sq.Eq{"demo_name": demoFile.Title}))
	if reportRowErr != nil {
		return errors.Join(reportRowErr, domain.ErrReportCountQuery)
	}

	var count int
	if errScan := reportRow.Scan(&count); errScan != nil && !errors.Is(errScan, domain.ErrNoResult) {
		return r.db.DBErr(errScan)
	}

	if count > 0 {
		demoFile.Archive = true
	}

	var err error
	if demoFile.DemoID > 0 {
		err = r.updateDemo(ctx, demoFile)
	} else {
		err = r.insertDemo(ctx, demoFile)
	}

	return r.db.DBErr(err)
}

func (r *demoRepository) insertDemo(ctx context.Context, demoFile *domain.DemoFile) error {
	query, args, errQueryArgs := r.db.
		Builder().
		Insert("demo").
		Columns("server_id", "title", "created_on", "downloads", "map_name", "archive", "stats", "asset_id").
		Values(demoFile.ServerID, demoFile.Title, demoFile.CreatedOn,
			demoFile.Downloads, demoFile.MapName, demoFile.Archive, demoFile.Stats, demoFile.AssetID).
		Suffix("RETURNING demo_id").
		ToSql()
	if errQueryArgs != nil {
		return r.db.DBErr(errQueryArgs)
	}

	errQuery := r.db.QueryRow(ctx, query, args...).Scan(&demoFile.ServerID)
	if errQuery != nil {
		return r.db.DBErr(errQuery)
	}

	return nil
}

func (r *demoRepository) updateDemo(ctx context.Context, demoFile *domain.DemoFile) error {
	query := r.db.
		Builder().
		Update("demo").
		Set("title", demoFile.Title).
		Set("downloads", demoFile.Downloads).
		Set("map_name", demoFile.MapName).
		Set("archive", demoFile.Archive).
		Set("stats", demoFile.Stats).
		Set("asset_id", demoFile.AssetID).
		Where(sq.Eq{"demo_id": demoFile.DemoID})

	if errExec := r.db.ExecUpdateBuilder(ctx, query); errExec != nil {
		return r.db.DBErr(errExec)
	}

	return nil
}

func (r *demoRepository) DropDemo(ctx context.Context, demoFile *domain.DemoFile) error {
	if errExec := r.db.ExecDeleteBuilder(ctx, r.db.
		Builder().
		Delete("demo").
		Where(sq.Eq{"demo_id": demoFile.DemoID})); errExec != nil {
		return r.db.DBErr(errExec)
	}

	demoFile.DemoID = 0

	return nil
}
