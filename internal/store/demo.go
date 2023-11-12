package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/srcdsup/srcdsup"
	"github.com/leighmacdonald/steamid/v3/steamid"
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
	AssetID         uuid.UUID                             `json:"asset_id"`
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

func (db *Store) FlushDemos(ctx context.Context) error {
	query, args, errQuery := db.sb.
		Delete("demo").
		Where(sq.And{
			sq.Eq{"archive": false},
			sq.Lt{"created_on": time.Now().Add(-(time.Hour * 24 * 14))},
		}).ToSql()
	if errQuery != nil {
		return errors.Wrap(errQuery, "Failed to create query")
	}

	return Err(db.Exec(ctx, query, args...))
}

func (db *Store) GetDemoByID(ctx context.Context, demoID int64, demoFile *DemoFile) error {
	query, args, errQueryArgs := db.sb.
		Select("demo_id", "server_id", "title", "raw_data", "created_on", "size", "downloads", "map_name", "archive", "stats").
		From("demo").
		Where(sq.Eq{"demo_id": demoID}).
		ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}

	if errQuery := db.QueryRow(ctx, query, args...).Scan(&demoFile.DemoID, &demoFile.ServerID, &demoFile.Title, &demoFile.Data,
		&demoFile.CreatedOn, &demoFile.Size, &demoFile.Downloads, &demoFile.MapName, &demoFile.Archive, &demoFile.Stats); errQuery != nil {
		return Err(errQuery)
	}

	return nil
}

func (db *Store) GetDemoByName(ctx context.Context, demoName string, demoFile *DemoFile) error {
	query, args, errQueryArgs := db.sb.
		Select("demo_id", "server_id", "title", "raw_data", "created_on", "size", "downloads", "map_name", "archive", "stats").
		From("demo").
		Where(sq.Eq{"title": demoName}).
		ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}

	if errQuery := db.QueryRow(ctx, query, args...).Scan(&demoFile.DemoID, &demoFile.ServerID, &demoFile.Title, &demoFile.Data,
		&demoFile.CreatedOn, &demoFile.Size, &demoFile.Downloads, &demoFile.MapName, &demoFile.Archive, &demoFile.Stats); errQuery != nil {
		return Err(errQuery)
	}

	return nil
}

type DemoFilter struct {
	QueryFilter
	SteamID   StringSID `json:"steam_id"`
	ServerIds []int     `json:"server_ids"`
	MapName   string    `json:"map_name"`
}

func (db *Store) GetDemos(ctx context.Context, opts DemoFilter) ([]DemoFile, int64, error) {
	var (
		demos       []DemoFile
		constraints sq.And
	)

	builder := db.sb.
		Select("d.demo_id", "d.server_id", "d.title", "d.created_on", "d.size", "d.downloads",
			"d.map_name", "d.archive", "d.stats", "s.short_name", "s.name", "d.asset_id").
		From("demo d").
		LeftJoin("server s ON s.server_id = d.server_id")

	if opts.MapName != "" {
		constraints = append(constraints, sq.ILike{"d.map_name": "%" + strings.ToLower(opts.MapName) + "%"})
	}

	if opts.SteamID != "" {
		sid64, errSid := opts.SteamID.SID64(ctx)
		if errSid != nil {
			return nil, 0, consts.ErrInvalidSID
		}

		constraints = append(constraints, sq.Expr("stats ?? ?", sid64.String()))
	}

	orderBy := "demo_id"
	if opts.OrderBy != "" {
		orderBy = opts.OrderBy
	}

	order := "ASC"
	if opts.Desc {
		order = "DESC"
	}

	builder = builder.OrderBy(fmt.Sprintf("d.%s %s", orderBy, order))

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

	var limit uint64

	if opts.Limit <= 0 || opts.Limit > 100 {
		limit = 50
	} else {
		limit = opts.Limit
	}

	query, args, errQueryArgs := builder.Limit(limit).Offset(opts.Offset).Where(constraints).ToSql()
	if errQueryArgs != nil {
		return nil, 0, Err(errQueryArgs)
	}

	rows, errQuery := db.Query(ctx, query, args...)
	if errQuery != nil {
		if errors.Is(errQuery, ErrNoResult) {
			return demos, 0, nil
		}

		return nil, 0, Err(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			demoFile DemoFile
			uuidScan *uuid.UUID // TODO remove this and make column not-null once migrations are complete
		)

		if errScan := rows.Scan(&demoFile.DemoID, &demoFile.ServerID, &demoFile.Title, &demoFile.CreatedOn,
			&demoFile.Size, &demoFile.Downloads, &demoFile.MapName, &demoFile.Archive, &demoFile.Stats,
			&demoFile.ServerNameShort, &demoFile.ServerNameLong, &uuidScan); errScan != nil {
			return nil, 0, Err(errScan)
		}

		if uuidScan != nil {
			demoFile.AssetID = *uuidScan
		}

		demos = append(demos, demoFile)
	}

	if demos == nil {
		return []DemoFile{}, 0, nil
	}

	counter := db.sb.Select("count(d.demo_id)").
		From("demo d").
		Where(constraints)

	countQuery, argsCount, errCountQuery := counter.ToSql()
	if errCountQuery != nil {
		return []DemoFile{}, 0, errors.Wrap(errCountQuery, "Failed to create count query")
	}

	var count int64
	if errCount := db.
		QueryRow(ctx, countQuery, argsCount...).
		Scan(&count); errCount != nil {
		return []DemoFile{}, 0, Err(errCount)
	}

	return demos, count, nil
}

func (db *Store) SaveDemo(ctx context.Context, demoFile *DemoFile) error {
	// Find open reports and if any are returned, mark the demo as archived so that it does not get auto
	// deleted during cleanup.
	// Reports can happen mid-game which is why this is checked when the demo is saved and not during the report where
	// we have no completed demo instance/id yet.
	query, args, queryErr := db.sb.
		Select("count(report_id)").
		From("report").
		Where(sq.Eq{"demo_name": demoFile.Title}).
		ToSql()
	if queryErr != nil {
		return errors.Wrap(queryErr, "Failed to select reports")
	}

	var count int
	if errScan := db.QueryRow(ctx, query, args...).Scan(&count); errScan != nil && !errors.Is(errScan, ErrNoResult) {
		return Err(errScan)
	}

	if count > 0 {
		demoFile.Archive = true
	}

	var err error
	if demoFile.DemoID > 0 {
		err = db.updateDemo(ctx, demoFile)
	} else {
		err = db.insertDemo(ctx, demoFile)
	}

	return Err(err)
}

func (db *Store) insertDemo(ctx context.Context, demoFile *DemoFile) error {
	query, args, errQueryArgs := db.sb.
		Insert(string(tableDemo)).
		Columns("server_id", "title", "raw_data", "created_on", "size", "downloads", "map_name", "archive", "stats", "asset_id").
		Values(demoFile.ServerID, demoFile.Title, demoFile.Data, demoFile.CreatedOn,
			demoFile.Size, demoFile.Downloads, demoFile.MapName, demoFile.Archive, demoFile.Stats, demoFile.AssetID).
		Suffix("RETURNING demo_id").
		ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}

	errQuery := db.QueryRow(ctx, query, args...).Scan(&demoFile.ServerID)
	if errQuery != nil {
		return Err(errQuery)
	}

	db.log.Info("New demo saved", zap.String("name", demoFile.Title))

	return nil
}

func (db *Store) updateDemo(ctx context.Context, demoFile *DemoFile) error {
	query, args, errQueryArgs := db.sb.
		Update(string(tableDemo)).
		Set("title", demoFile.Title).
		Set("size", demoFile.Size).
		Set("downloads", demoFile.Downloads).
		Set("map_name", demoFile.MapName).
		Set("archive", demoFile.Archive).
		Set("stats", demoFile.Stats).
		Set("asset_id", demoFile.AssetID).
		Where(sq.Eq{"demo_id": demoFile.DemoID}).
		ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}

	if errExec := db.Exec(ctx, query, args...); errExec != nil {
		return Err(errExec)
	}

	db.log.Info("Demo updated", zap.String("name", demoFile.Title))

	return nil
}

func (db *Store) DropDemo(ctx context.Context, demoFile *DemoFile) error {
	query, args, errQueryArgs := db.sb.
		Delete(string(tableDemo)).
		Where(sq.Eq{"demo_id": demoFile.DemoID}).
		ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}

	if errExec := db.Exec(ctx, query, args...); errExec != nil {
		return Err(errExec)
	}

	demoFile.DemoID = 0

	db.log.Info("Demo deleted:", zap.String("name", demoFile.Title))

	return nil
}

type Asset struct {
	AssetID  uuid.UUID `json:"asset_id"`
	Bucket   string    `json:"bucket"`
	Path     string    `json:"path"`
	Name     string    `json:"name"`
	MimeType string    `json:"mime_type"`
	Size     int64     `json:"size"`
	OldID    int64     `json:"old_id"` // Pre S3 id
}

func NewAsset(content []byte, bucket string, name string) (Asset, error) {
	detectedMime := mimetype.Detect(content)

	newID, errID := uuid.NewV4()
	if errID != nil {
		return Asset{}, errors.Wrap(errID, "Failed to generate a new asset ID")
	}

	if name == "" {
		name = newID.String()
	}

	asset := Asset{
		AssetID:  newID,
		Bucket:   bucket,
		Path:     "/",
		Name:     name,
		MimeType: detectedMime.String(),
		Size:     int64(len(content)),
	}

	return asset, nil
}

func (db *Store) SaveAsset(ctx context.Context, asset *Asset) error {
	query, args, errQueryArgs := db.sb.Insert("asset").
		Columns("asset_id", "bucket", "path", "name", "mime_type", "size", "old_id").
		Values(asset.AssetID, asset.Bucket, asset.Path, asset.Name, asset.MimeType, asset.Size, asset.OldID).
		ToSql()

	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}

	if err := db.Exec(ctx, query, args...); err != nil {
		return Err(err)
	}

	return nil
}
