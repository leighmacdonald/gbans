package store

import (
	"context"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type DemoPlayerStats struct {
	Score      int `json:"score"`
	ScoreTotal int `json:"score_total"`
	Deaths     int `json:"deaths"`
}

type DemoMetaData struct {
	MapName string                     `json:"map_name"`
	Scores  map[string]DemoPlayerStats `json:"scores"`
}

type DemoFile struct {
	DemoID          int64                   `json:"demo_id"`
	ServerID        int                     `json:"server_id"`
	ServerNameShort string                  `json:"server_name_short"`
	ServerNameLong  string                  `json:"server_name_long"`
	Title           string                  `json:"title"`
	CreatedOn       time.Time               `json:"created_on"`
	Downloads       int64                   `json:"downloads"`
	Size            int64                   `json:"size"`
	MapName         string                  `json:"map_name"`
	Archive         bool                    `json:"archive"` // When true, will not get auto deleted when flushing old demos
	Stats           map[steamid.SID64]gin.H `json:"stats"`
	AssetID         uuid.UUID               `json:"asset_id"`
}

func (db *Store) FlushDemos(ctx context.Context) error {
	return db.ExecDeleteBuilder(ctx, db.sb.
		Delete("demo").
		Where(sq.And{
			sq.Eq{"archive": false},
			sq.Lt{"created_on": time.Now().Add(-(time.Hour * 24 * 14))},
		}))
}

func (db *Store) GetDemoByID(ctx context.Context, demoID int64, demoFile *DemoFile) error {
	row, errRow := db.QueryRowBuilder(ctx, db.sb.
		Select("d.demo_id", "d.server_id", "d.title", "d.created_on", "d.downloads",
			"d.map_name", "d.archive", "d.stats", "d.asset_id", "a.size", "s.short_name", "s.name").
		From("demo d").
		LeftJoin("server s ON s.server_id = d.server_id").
		LeftJoin("asset a ON a.asset_id = d.asset_id").
		Where(sq.Eq{"demo_id": demoID}))
	if errRow != nil {
		return errRow
	}

	var uuidScan *uuid.UUID

	if errQuery := row.Scan(&demoFile.DemoID, &demoFile.ServerID, &demoFile.Title,
		&demoFile.CreatedOn, &demoFile.Downloads, &demoFile.MapName,
		&demoFile.Archive, &demoFile.Stats, uuidScan, &demoFile.Size, &demoFile.ServerNameShort,
		&demoFile.ServerNameLong); errQuery != nil {
		return Err(errQuery)
	}

	if uuidScan != nil {
		demoFile.AssetID = *uuidScan
	}

	return nil
}

func (db *Store) GetDemoByName(ctx context.Context, demoName string, demoFile *DemoFile) error {
	row, errRow := db.QueryRowBuilder(ctx, db.sb.
		Select("d.demo_id", "d.server_id", "d.title", "d.created_on", "d.downloads",
			"d.map_name", "d.archive", "d.stats", "d.asset_id", "a.size", "s.short_name", "s.name").
		From("demo d").
		LeftJoin("server s ON s.server_id = d.server_id").
		LeftJoin("asset a ON a.asset_id = d.asset_id").
		Where(sq.Eq{"title": demoName}))
	if errRow != nil {
		return errRow
	}

	var uuidScan *uuid.UUID

	if errQuery := row.Scan(&demoFile.DemoID, &demoFile.ServerID, &demoFile.Title,
		&demoFile.CreatedOn, &demoFile.Downloads, &demoFile.MapName, &demoFile.Archive, &demoFile.Stats,
		&demoFile.Stats, &uuidScan, &demoFile.Size, &demoFile.ServerNameShort,
		&demoFile.ServerNameLong); errQuery != nil {
		return Err(errQuery)
	}

	if uuidScan != nil {
		demoFile.AssetID = *uuidScan
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
		Select("d.demo_id", "d.server_id", "d.title", "d.created_on", "d.downloads",
			"d.map_name", "d.archive", "d.stats", "s.short_name", "s.name", "d.asset_id", "a.size").
		From("demo d").
		LeftJoin("server s ON s.server_id = d.server_id").
		LeftJoin("asset a ON a.asset_id = d.asset_id")

	if opts.MapName != "" {
		constraints = append(constraints, sq.ILike{"d.map_name": "%" + strings.ToLower(opts.MapName) + "%"})
	}

	if opts.SteamID != "" {
		sid64, errSid := opts.SteamID.SID64(ctx)
		if errSid != nil {
			return nil, 0, consts.ErrInvalidSID
		}

		constraints = append(constraints, sq.Expr("d.stats ?? ?", sid64.String()))
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

	builder = opts.QueryFilter.applySafeOrder(builder, map[string][]string{
		"d.": {"demo_id", "server_id", "title", "created_on", "downloads", "map_name"},
		"s.": {"short_name", "name"},
		"a.": {"size"},
	}, "demo_id")

	builder = opts.QueryFilter.applyLimitOffsetDefault(builder).Where(constraints)

	rows, errQuery := db.QueryBuilder(ctx, builder)
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
			&demoFile.Downloads, &demoFile.MapName, &demoFile.Archive, &demoFile.Stats,
			&demoFile.ServerNameShort, &demoFile.ServerNameLong, &uuidScan, &demoFile.Size); errScan != nil {
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

	count, errCount := db.GetCount(ctx, db.sb.Select("count(d.demo_id)").
		From("demo d").
		Where(constraints))
	if errCount != nil {
		return []DemoFile{}, 0, Err(errCount)
	}

	return demos, count, nil
}

func (db *Store) SaveDemo(ctx context.Context, demoFile *DemoFile) error {
	// Find open reports and if any are returned, mark the demo as archived so that it does not get auto
	// deleted during cleanup.
	// Reports can happen mid-game which is why this is checked when the demo is saved and not during the report where
	// we have no completed demo instance/id yet.
	reportRow, reportRowErr := db.QueryRowBuilder(ctx, db.sb.
		Select("count(report_id)").
		From("report").
		Where(sq.Eq{"demo_name": demoFile.Title}))
	if reportRowErr != nil {
		return errors.Wrap(reportRowErr, "Failed to select reports")
	}

	var count int
	if errScan := reportRow.Scan(&count); errScan != nil && !errors.Is(errScan, ErrNoResult) {
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
		Insert("demo").
		Columns("server_id", "title", "created_on", "downloads", "map_name", "archive", "stats", "asset_id").
		Values(demoFile.ServerID, demoFile.Title, demoFile.CreatedOn,
			demoFile.Downloads, demoFile.MapName, demoFile.Archive, demoFile.Stats, demoFile.AssetID).
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
	query := db.sb.
		Update("demo").
		Set("title", demoFile.Title).
		Set("downloads", demoFile.Downloads).
		Set("map_name", demoFile.MapName).
		Set("archive", demoFile.Archive).
		Set("stats", demoFile.Stats).
		Set("asset_id", demoFile.AssetID).
		Where(sq.Eq{"demo_id": demoFile.DemoID})

	if errExec := db.ExecUpdateBuilder(ctx, query); errExec != nil {
		return errExec
	}

	db.log.Info("Demo updated", zap.String("name", demoFile.Title))

	return nil
}

func (db *Store) DropDemo(ctx context.Context, demoFile *DemoFile) error {
	if errExec := db.ExecDeleteBuilder(ctx, db.sb.
		Delete("demo").
		Where(sq.Eq{"demo_id": demoFile.DemoID})); errExec != nil {
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
	return db.ExecInsertBuilder(ctx, db.sb.
		Insert("asset").
		Columns("asset_id", "bucket", "path", "name", "mime_type", "size", "old_id").
		Values(asset.AssetID, asset.Bucket, asset.Path, asset.Name, asset.MimeType, asset.Size, asset.OldID))
}
