package demo

import (
	"context"
	"errors"

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
		LeftJoin("report r on d.demo_id = r.demo_id").
		Where("r.demo_id > 0").
		OrderBy("d.demo_id").
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
		From("demo d").
		LeftJoin("server s ON s.server_id = d.server_id").
		LeftJoin("asset a ON a.asset_id = d.asset_id").
		Where(sq.Eq{"demo_id": demoID}))
	if errRow != nil {
		return r.db.DBErr(errRow)
	}

	var uuidScan *uuid.UUID

	if errQuery := row.Scan(&demoFile.DemoID, &demoFile.ServerID, &demoFile.Title,
		&demoFile.CreatedOn, &demoFile.Downloads, &demoFile.MapName,
		&demoFile.Archive, &demoFile.Stats, &uuidScan, &demoFile.Size, &demoFile.ServerNameShort,
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

func (r *demoRepository) GetDemos(ctx context.Context) ([]domain.DemoFile, error) {
	var demos []domain.DemoFile

	builder := r.db.
		Builder().
		Select("d.demo_id", "d.server_id", "d.title", "d.created_on", "d.downloads",
			"d.map_name", "d.archive", "d.stats", "s.short_name", "s.name", "d.asset_id", "a.size").
		From("demo d").
		LeftJoin("server s ON s.server_id = d.server_id").
		LeftJoin("asset a ON a.asset_id = d.asset_id").
		OrderBy("d.demo_id DESC")

	rows, errQuery := r.db.QueryBuilder(ctx, builder)
	if errQuery != nil {
		if errors.Is(errQuery, domain.ErrNoResult) {
			return demos, nil
		}

		return nil, r.db.DBErr(errQuery)
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
			return nil, r.db.DBErr(errScan)
		}

		if uuidScan != nil {
			demoFile.AssetID = *uuidScan
		}

		demos = append(demos, demoFile)
	}

	if demos == nil {
		return []domain.DemoFile{}, nil
	}

	return demos, nil
}

func (r *demoRepository) SaveDemo(ctx context.Context, demoFile *domain.DemoFile) error {
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
