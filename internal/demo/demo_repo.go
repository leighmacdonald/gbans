package demo

import (
	"context"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/database"
)

var ErrServerValidate = errors.New("failed to validate server")

type Repository struct {
	database.Database
}

func NewRepository(database database.Database) Repository {
	return Repository{Database: database}
}

func (r Repository) ValidateServer(ctx context.Context, serverID int32) error {
	if serverID == 0 {
		return ErrServerValidate
	}

	var serverIDScan int32
	if errQuery := r.
		QueryRow(ctx, `SELECT server_id FROM server WHERE server_id = $1`, serverID).
		Scan(&serverIDScan); errQuery != nil {
		return errors.Join(errQuery, ErrServerValidate)
	}

	return nil
}

func (r Repository) ExpiredDemos(ctx context.Context, limit uint64) ([]Info, error) {
	rows, errRow := r.QueryBuilder(ctx, r.Builder().
		Select("d.demo_id", "d.title", "d.asset_id").
		From("demo d").
		Where(sq.Eq{"d.archive": false}).
		OrderBy("d.demo_id DESC").
		Offset(limit))
	if errRow != nil {
		return nil, database.Err(errRow)
	}

	defer rows.Close()

	var demos []Info

	for rows.Next() {
		var demo Info
		if err := rows.Scan(&demo.DemoID, &demo.Title, &demo.AssetID); err != nil {
			return nil, database.Err(err)
		}

		demos = append(demos, demo)
	}

	return demos, nil
}

func (r Repository) GetDemoByColumn(ctx context.Context, key string, value any) (*File, error) {
	var demoFile File
	row, errRow := r.Database.QueryRowBuilder(ctx, r.Builder().
		Select("d.demo_id", "d.server_id", "d.title", "d.created_on", "d.downloads",
			"d.map_name", "d.archive", "d.stats", "d.asset_id", "a.size", "s.short_name", "s.name").
		From("demo d").
		LeftJoin("server s ON s.server_id = d.server_id").
		LeftJoin("asset a ON a.asset_id = d.asset_id").
		Where(sq.Eq{key: value}))
	if errRow != nil {
		return nil, database.Err(errRow)
	}

	var uuidScan *uuid.UUID

	if errQuery := row.Scan(&demoFile.DemoID, &demoFile.ServerID, &demoFile.Title,
		&demoFile.CreatedOn, &demoFile.Downloads, &demoFile.MapName,
		&demoFile.Archive, &demoFile.Stats, &uuidScan, &demoFile.Size, &demoFile.ServerNameShort,
		&demoFile.ServerNameLong); errQuery != nil {
		return nil, database.Err(errQuery)
	}

	if uuidScan != nil {
		demoFile.AssetID = *uuidScan
	}

	return &demoFile, nil
}

func (r Repository) GetDemoByAssetID(ctx context.Context, assetID uuid.UUID) (*File, error) {
	return r.GetDemoByColumn(ctx, "a.asset_id", assetID)
}

func (r Repository) GetDemoByID(ctx context.Context, demoID int32) (*File, error) {
	return r.GetDemoByColumn(ctx, "d.demo_id", demoID)
}

func (r Repository) GetDemoByName(ctx context.Context, demoName string) (*File, error) {
	return r.GetDemoByColumn(ctx, "d.title", demoName)
}

func (r Repository) GetDemos(ctx context.Context) ([]File, error) {
	var demos []File

	builder := r.Builder().
		Select("d.demo_id", "d.server_id", "d.title", "d.created_on", "d.downloads",
			"d.map_name", "d.archive", "d.stats", "s.short_name", "s.name", "d.asset_id", "a.size").
		From("demo d").
		LeftJoin("server s ON s.server_id = d.server_id").
		LeftJoin("asset a ON a.asset_id = d.asset_id").
		OrderBy("d.demo_id DESC")

	rows, errQuery := r.QueryBuilder(ctx, builder)
	if errQuery != nil {
		if errors.Is(errQuery, database.ErrNoResult) {
			return demos, nil
		}

		return nil, database.Err(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			demoFile File
			uuidScan *uuid.UUID // TODO remove this and make column not-null once migrations are complete
		)

		if errScan := rows.Scan(&demoFile.DemoID, &demoFile.ServerID, &demoFile.Title, &demoFile.CreatedOn,
			&demoFile.Downloads, &demoFile.MapName, &demoFile.Archive, &demoFile.Stats,
			&demoFile.ServerNameShort, &demoFile.ServerNameLong, &uuidScan, &demoFile.Size); errScan != nil {
			return nil, database.Err(errScan)
		}

		if uuidScan != nil {
			demoFile.AssetID = *uuidScan
		}

		demos = append(demos, demoFile)
	}

	if demos == nil {
		return []File{}, nil
	}

	return demos, nil
}

func (r Repository) SaveDemo(ctx context.Context, demoFile *File) error {
	var err error
	if demoFile.DemoID > 0 {
		err = r.updateDemo(ctx, demoFile)
	} else {
		err = r.insertDemo(ctx, demoFile)
	}

	return database.Err(err)
}

func (r Repository) insertDemo(ctx context.Context, demoFile *File) error {
	query, args, errQueryArgs := r.Builder().
		Insert("demo").
		Columns("server_id", "title", "created_on", "downloads", "map_name", "archive", "stats", "asset_id").
		Values(demoFile.ServerID, demoFile.Title, demoFile.CreatedOn,
			demoFile.Downloads, demoFile.MapName, demoFile.Archive, demoFile.Stats, demoFile.AssetID).
		Suffix("RETURNING demo_id").
		ToSql()
	if errQueryArgs != nil {
		return database.Err(errQueryArgs)
	}

	errQuery := r.QueryRow(ctx, query, args...).Scan(&demoFile.DemoID)
	if errQuery != nil {
		return database.Err(errQuery)
	}

	return nil
}

func (r Repository) updateDemo(ctx context.Context, demoFile *File) error {
	query := r.Builder().
		Update("demo").
		Set("title", demoFile.Title).
		Set("downloads", demoFile.Downloads).
		Set("map_name", demoFile.MapName).
		Set("archive", demoFile.Archive).
		Set("stats", demoFile.Stats).
		Set("asset_id", demoFile.AssetID).
		Where(sq.Eq{"demo_id": demoFile.DemoID})

	if errExec := r.ExecUpdateBuilder(ctx, query); errExec != nil {
		return database.Err(errExec)
	}

	return nil
}

func (r Repository) Delete(ctx context.Context, demoID int32) error {
	const query = `DELETE FROM demo WHERE demo_id = $1`
	if err := r.Exec(ctx, query, demoID); err != nil {
		return database.Err(err)
	}

	return nil
}
