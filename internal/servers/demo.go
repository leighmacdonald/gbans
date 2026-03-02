package servers

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/fs"
	"github.com/leighmacdonald/gbans/internal/network/scp"
	"github.com/leighmacdonald/gbans/pkg/demoparse"
	"github.com/leighmacdonald/gbans/pkg/zstd"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/ricochet2200/go-disk-usage/du"
	"github.com/viant/afs/option"
	"github.com/viant/afs/storage"
)

var (
	ErrDemoLoad       = errors.New("could not load demo file")
	ErrFailedOpenFile = errors.New("failed to open file")
	ErrFailedReadFile = errors.New("failed to read file")
	ErrParse          = errors.New("could not parse demo")
)

type DemoStrategy string

const (
	DemoStrategyPctFree DemoStrategy = "pctfree"
	DemoStrategyCount   DemoStrategy = "count"
)

type DemoConfig struct {
	DemoCleanupEnabled  bool         `json:"demo_cleanup_enabled"`
	DemoCleanupStrategy DemoStrategy `json:"demo_cleanup_strategy"`
	DemoCleanupMinPct   float32      `json:"demo_cleanup_min_pct,string"`
	DemoCleanupMount    string       `json:"demo_cleanup_mount"`
	DemoCountLimit      uint64       `json:"demo_count_limit"`
	DemoParserURL       string       `json:"demo_parser_url"`
}

type DemoFilter struct {
	query.Filter

	SteamID   string `json:"steam_id"`
	ServerIDs []int  `json:"server_ids"` //nolint:tagliatelle
	MapName   string `json:"map_name"`
}

func (f DemoFilter) SourceSteamID() (steamid.SteamID, bool) {
	sid := steamid.New(f.SteamID)

	return sid, sid.Valid()
}

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
	DemoID          int64                     `json:"demo_id"`
	ServerID        int                       `json:"server_id"`
	ServerNameShort string                    `json:"server_name_short"`
	ServerNameLong  string                    `json:"server_name_long"`
	Title           string                    `json:"title"`
	CreatedOn       time.Time                 `json:"created_on"`
	Downloads       int64                     `json:"downloads"`
	Size            int64                     `json:"size"`
	MapName         string                    `json:"map_name"`
	Archive         bool                      `json:"archive"` // When true, will not get auto deleted when flushing old demos
	Stats           map[string]map[string]any `json:"stats"`
	AssetID         uuid.UUID                 `json:"asset_id"`
}

type DemoInfo struct {
	DemoID  int64
	Title   string
	AssetID uuid.UUID
}

type UploadedDemo struct {
	Name     string
	ServerID int
	Content  []byte
}

type Demos struct {
	DemoConfig

	repository  DemoRepository
	asset       asset.Assets
	bucket      asset.Bucket
	cleanupChan chan any
	owner       steamid.SteamID
}

func NewDemos(bucket asset.Bucket, repository DemoRepository, assets asset.Assets, config DemoConfig, owner steamid.SteamID) Demos {
	return Demos{
		DemoConfig:  config,
		bucket:      bucket,
		repository:  repository,
		asset:       assets,
		cleanupChan: make(chan any),
		owner:       owner,
	}
}

func (d Demos) onDemoReceived(ctx context.Context, demo UploadedDemo) error {
	slog.Debug("Got new demo",
		slog.Int("server_id", demo.ServerID),
		slog.String("name", demo.Name))

	// TOOO make these interfaces less clunky for compressed data.
	var compressed bytes.Buffer
	if err := zstd.Compress(bytes.NewReader(demo.Content), bufio.NewWriter(&compressed)); err != nil {
		return err
	}

	demoAsset, errNewAsset := d.asset.Create(ctx, steamid.New(d.owner),
		asset.BucketDemo, demo.Name+zstd.Extension, bytes.NewReader(compressed.Bytes()), false)
	if errNewAsset != nil {
		return errNewAsset
	}

	_, errDemo := d.CreateFromAsset(ctx, &demoAsset, demo.ServerID)
	if errDemo != nil {
		// Cleanup the asset not attached to a valid demo
		if _, errDelete := d.asset.Delete(ctx, demoAsset.AssetID); errDelete != nil {
			return errors.Join(errDelete, errDelete)
		}

		return errDemo
	}

	return nil
}

func (d Demos) DownloadHandler(ctx context.Context, client storage.Storager, server scp.ServerInfo, config scp.Config) error {
	for _, instance := range server.ServerIDs {
		demoDir := server.GamePath(config.DemoPathFmt, instance)
		filelist, errFilelist := client.List(ctx, demoDir, option.NewPage(0, 1))
		if errFilelist != nil {
			slog.Error("remote list dir failed", slog.String("error", errFilelist.Error()),
				slog.String("server", instance.ShortName), slog.String("path", demoDir))

			return nil //nolint:nilerr
		}

		for _, file := range filelist {
			if !strings.HasSuffix(file.Name(), ".dem") {
				continue
			}

			demoPath := path.Join(demoDir, file.Name())

			slog.Debug("Downloading demo", slog.String("name", file.Name()), slog.String("server", instance.ShortName))

			reader, err := client.Open(ctx, demoPath)
			if err != nil {
				return errors.Join(err, ErrFailedOpenFile)
			}

			data, errRead := io.ReadAll(reader)
			if errRead != nil {
				_ = reader.Close()

				return errors.Join(errRead, ErrFailedReadFile)
			}

			_ = reader.Close()

			// need Seeker, but afs does not provide
			demo := UploadedDemo{Name: file.Name(), ServerID: instance.ServerID, Content: data}
			if errDemo := d.onDemoReceived(ctx, demo); errDemo != nil {
				if !errors.Is(errDemo, asset.ErrAssetTooLarge) {
					slog.Error("Failed to create new demo asset", slog.String("error", errDemo.Error()))
				}

				continue
			}

			if errDelete := client.Delete(ctx, demoPath); errDelete != nil {
				slog.Error("Failed to cleanup demo", slog.String("error", errDelete.Error()), slog.String("path", demoPath))

				continue
			}

			slog.Debug("Deleted demo on remote host", slog.String("path", demoPath))
		}
	}

	return nil
}

func (d Demos) oldest(ctx context.Context) (DemoInfo, error) {
	demos, errDemos := d.repository.ExpiredDemos(ctx, 1)
	if errDemos != nil {
		return DemoInfo{}, errDemos
	}

	if len(demos) == 0 {
		return DemoInfo{}, database.ErrNoResult
	}

	return demos[0], nil
}

func (d Demos) MarkArchived(ctx context.Context, demo *DemoFile) error {
	demo.Archive = true

	if err := d.repository.SaveDemo(ctx, demo); err != nil {
		slog.Error("Failed to mark demo as archived", slog.String("error", err.Error()), slog.Int64("demo_id", demo.DemoID))
	}

	slog.Debug("Demo marked as archived", slog.Int64("demo_id", demo.DemoID))

	return nil
}

func diskPercentageUsed(path string) float32 {
	info := du.NewDiskUsage(path)

	return info.Usage() * 100
}

func (d Demos) TruncateBySpace(ctx context.Context, root string, maxAllowedPctUsed float32) (int, int64, error) {
	var (
		count int
		size  int64
	)

	defer func() {
		slog.Debug("Truncate by space completed", slog.Int("count", count), slog.String("total_size", humanize.Bytes(uint64(size)))) //nolint:gosec
	}()

	for {
		usedSpace := diskPercentageUsed(root)

		if usedSpace < maxAllowedPctUsed {
			return count, size, nil
		}

		oldestDemo, errOldest := d.oldest(ctx)
		if errOldest != nil {
			if errors.Is(errOldest, database.ErrNoResult) {
				return count, size, nil
			}

			return count, size, errOldest
		}

		demoSize, err := d.asset.Delete(ctx, oldestDemo.AssetID)
		if err != nil {
			return count, size, err
		}

		size += demoSize
		count++
	}
}

func (d Demos) TruncateByCount(ctx context.Context, maxCount uint64) (int, int64, error) {
	var (
		count int
		size  int64
	)

	expired, errExpired := d.repository.ExpiredDemos(ctx, maxCount)
	if errExpired != nil {
		if errors.Is(errExpired, database.ErrNoResult) {
			return count, size, nil
		}

		return count, size, errExpired
	}

	if len(expired) == 0 {
		return count, size, nil
	}

	for _, demo := range expired {
		// FIXME cascade delete does not work????
		demoSize, errDrop := d.asset.Delete(ctx, demo.AssetID)
		if errDrop != nil && !errors.Is(errDrop, asset.ErrDeleteAssetFile) {
			slog.Error("Failed to remove demo asset", slog.String("error", errDrop.Error()),
				slog.String("bucket", string(d.bucket)), slog.String("name", demo.Title))

			continue
		}

		if err := d.repository.Delete(ctx, demo.DemoID); err != nil {
			slog.Error("Failed to remove demo entry",
				slog.Int64("demo_id", demo.DemoID),
				slog.String("asset_id", demo.AssetID.String()),
				slog.String("error", err.Error()))
		}

		size += demoSize
		count++
	}

	slog.Debug("Truncate by count completed", slog.Int("count", count), slog.String("total_size", humanize.Bytes(uint64(size)))) //nolint:gosec

	return count, size, nil
}

func (d Demos) Cleanup(ctx context.Context) {
	if !d.DemoCleanupEnabled {
		return
	}

	slog.Debug("Starting demo cleanup", slog.String("strategy", string(d.DemoCleanupStrategy)))

	var (
		count int
		err   error
		size  int64
	)

	switch d.DemoCleanupStrategy {
	case DemoStrategyPctFree:
		count, size, err = d.TruncateBySpace(ctx, d.DemoCleanupMount, d.DemoCleanupMinPct)
	case DemoStrategyCount:
		count, size, err = d.TruncateByCount(ctx, d.DemoCountLimit)
	}

	if err != nil {
		slog.Error("Error executing demo cleanup", slog.String("strategy", string(d.DemoCleanupStrategy)))
	}

	slog.Debug("Old demos flushed", slog.Int("count", count), slog.String("size", humanize.Bytes(uint64(size)))) //nolint:gosec

	if errOrphans := d.RemoveOrphans(ctx); errOrphans != nil {
		slog.Error("Failed to execute orphans", slog.String("error", errOrphans.Error()))
	}
}

func (d Demos) ExpiredDemos(ctx context.Context, limit uint64) ([]DemoInfo, error) {
	return d.repository.ExpiredDemos(ctx, limit)
}

func (d Demos) GetDemoByID(ctx context.Context, demoID int64, demoFile *DemoFile) error {
	return d.repository.GetDemoByID(ctx, demoID, demoFile)
}

func (d Demos) GetDemoByName(ctx context.Context, demoName string, demoFile *DemoFile) error {
	return d.repository.GetDemoByName(ctx, demoName, demoFile)
}

func (d Demos) GetDemos(ctx context.Context) ([]DemoFile, error) {
	return d.repository.GetDemos(ctx)
}

func (d Demos) CreateFromAsset(ctx context.Context, asset *asset.Asset, serverID int) (*DemoFile, error) {
	if errGetServer := d.repository.ValidateServer(ctx, serverID); errGetServer != nil {
		return nil, ErrGetServer
	}
	var (
		demo     *demoparse.Demo
		err      error
		filename = asset.Name
		mapName  string
	)

	namePartsAll := strings.Split(filename, "-")

	if strings.Contains(filename, "workshop-") {
		// 20231221-042605-workshop-cp_overgrown_rc8-ugc503939302.dem
		mapName = namePartsAll[3]
	} else {
		// 20231112-063943-koth_harvest_final.dem
		nameParts := strings.Split(namePartsAll[2], ".")
		mapName = nameParts[0]
	}

	demo, err = demoparse.Submit(ctx, d.DemoParserURL, asset.String(), asset)
	if err != nil {
		return nil, err
	}

	// TODO change this data shape as we have not needed this in a long time. Only keys the are used.
	intStats := map[string]map[string]any{}

	for _, key := range demo.SteamIDs() {
		if key.Valid() {
			intStats[key.String()] = map[string]any{}
		}
	}

	timeStr := fmt.Sprintf("%s-%s", namePartsAll[0], namePartsAll[1])
	createdTime, errTime := time.Parse("20060102-150405", timeStr) // 20240511-211121
	if errTime != nil {
		slog.Warn("Failed to parse demo time, using current time", slog.String("time", timeStr))

		createdTime = time.Now()
	}

	newDemo := DemoFile{
		ServerID:  serverID,
		Title:     asset.Name,
		CreatedOn: createdTime,
		MapName:   mapName,
		Stats:     intStats,
		AssetID:   asset.AssetID,
	}

	if errSave := d.repository.SaveDemo(ctx, &newDemo); errSave != nil {
		return nil, errSave
	}

	slog.Debug("Created demo from asset successfully", slog.Int64("demo_id", newDemo.DemoID), slog.String("title", newDemo.Title))

	return &newDemo, nil
}

func (d Demos) RemoveOrphans(ctx context.Context) error {
	demos, errDemos := d.GetDemos(ctx)
	if errDemos != nil {
		return errDemos
	}

	for _, demo := range demos {
		var remove bool
		realAsset, _, errAsset := d.asset.Get(ctx, demo.AssetID)
		if errAsset != nil {
			// If it doesn't exist on disk we want to delete our internal references to it.
			if errors.Is(errAsset, database.ErrNoResult) || errors.Is(errAsset, asset.ErrOpenFile) {
				remove = true
			} else {
				return errAsset
			}
		} else {
			localPath, errPath := d.asset.GenAssetPath(realAsset.HashString())
			if errPath != nil {
				return errPath
			}

			remove = !fs.Exists(localPath)
		}

		if !remove {
			continue
		}

		slog.Debug("Removing orphan demo", slog.Int64("demo_id", demo.DemoID),
			slog.String("title", demo.Title), slog.String("asset_id", demo.AssetID.String()))
		if _, err := d.asset.Delete(ctx, demo.AssetID); err != nil {
			slog.Error("Failed to remove orphan demo asset", slog.String("error", err.Error()))

			continue
		}

		if err := d.repository.Delete(ctx, demo.DemoID); err != nil {
			slog.Error("Failed to remove orphan demo entry", slog.String("error", err.Error()))

			continue
		}

		// TODO delete empty folders
		slog.Warn("Removed orphan demo file", slog.String("filename", demo.Title))
	}

	return nil
}
