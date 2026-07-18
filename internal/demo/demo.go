package demo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/fs"
	"github.com/leighmacdonald/gbans/internal/network/scp"
	"github.com/leighmacdonald/gbans/internal/stats"
	"github.com/leighmacdonald/gbans/pkg/demoparse"
	"github.com/leighmacdonald/gbans/pkg/zstd"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/ricochet2200/go-disk-usage/du"
	"github.com/viant/afs/storage"
)

var (
	ErrDemoLoad       = errors.New("could not load demo file")
	ErrFailedOpenFile = errors.New("failed to open file")
	ErrFailedReadFile = errors.New("failed to read file")
	ErrParse          = errors.New("could not parse demo")
)

type Strategy string

const (
	DemoStrategyPctFree Strategy = "pctfree"
	DemoStrategyCount   Strategy = "count"
)

type Config struct {
	sync.RWMutex

	DemoCleanupEnabled  bool
	DemoCleanupStrategy Strategy
	DemoCleanupMinPct   float32
	DemoCleanupMount    string
	DemoCountLimit      uint64
	DemoParserURL       string
}

type Filter struct {
	query.Filter

	SteamID   string
	ServerIDs []int //nolint:tagliatelle
	MapName   string
}

func (f Filter) SourceSteamID() (steamid.SteamID, bool) {
	sid := steamid.New(f.SteamID)

	return sid, sid.Valid()
}

type PlayerStats struct {
	Score      int
	ScoreTotal int
	Deaths     int
}

type MetaData struct {
	MapName string
	Scores  map[string]PlayerStats
}

type File struct {
	DemoID          int32
	ServerID        int32
	ServerNameShort string
	ServerNameLong  string
	Title           string
	CreatedOn       time.Time
	Downloads       int64
	Size            int64
	MapName         string
	Archive         bool // When true, will not get auto deleted when flushing old demos
	Stats           map[string]map[string]any
	AssetID         uuid.UUID
}

type Info struct {
	DemoID  int32
	Title   string
	AssetID uuid.UUID
}

type UploadedDemo struct {
	Name     string
	ServerID int32
	Content  []byte
}

type Demos struct {
	*Config

	stats       stats.Stats
	repository  Repository
	asset       asset.Assets
	bucket      asset.Bucket
	person      PersonCreator
	chat        *chat.Chat
	cleanupChan chan any
	owner       steamid.SteamID
}

type PersonCreator interface {
	EnsurePerson(ctx context.Context, steamID steamid.SteamID) error
}

func NewDemos(bucket asset.Bucket, repository Repository, assets asset.Assets, stats stats.Stats, chat *chat.Chat, person PersonCreator, config *Config, owner steamid.SteamID) Demos {
	return Demos{
		Config:      config,
		bucket:      bucket,
		repository:  repository,
		asset:       assets,
		stats:       stats,
		chat:        chat,
		cleanupChan: make(chan any),
		owner:       owner,
		person:      person,
	}
}

func (d Demos) createFromAsset(ctx context.Context, asset *asset.Asset, serverID int32, createStats bool) (*File, error) {
	if errGetServer := d.repository.ValidateServer(ctx, serverID); errGetServer != nil {
		return nil, errGetServer
	}
	var (
		parsedDemo *demoparse.Demo
		err        error
		filename   = asset.Name
		mapName    string
	)

	namePartsAll := strings.Split(filename, "-")

	existing, errExisting := d.repository.GetDemoByAssetID(ctx, asset.AssetID)
	if errExisting == nil {
		if err := d.stats.Delete(ctx, existing.DemoID); err != nil {
			return nil, err
		}
		if err := d.chat.DeleteByDemoID(ctx, existing.DemoID); err != nil {
			return nil, err
		}
	}

	if strings.Contains(filename, "workshop-") {
		// 20231221-042605-workshop-cp_overgrown_rc8-ugc503939302.dem
		mapName = namePartsAll[3]
	} else {
		// 20231112-063943-koth_harvest_final.dem
		nameParts := strings.Split(namePartsAll[2], ".")
		mapName = nameParts[0]
	}

	parsedDemo, err = demoparse.Submit(ctx, d.DemoParserURL, asset.String(), asset)
	if err != nil {
		return nil, err
	}

	// TODO change this data shape as we have not needed this in a long time. Only keys the are used.
	intStats := map[string]map[string]any{}

	for _, playerSteamID := range parsedDemo.SteamIDs() {
		if playerSteamID.Valid() {
			intStats[playerSteamID.String()] = map[string]any{}
			if err := d.person.EnsurePerson(ctx, playerSteamID); err != nil {
				slog.Error("Failed to insert player", slog.String("error", err.Error()))

				return nil, err
			}
		}
	}

	timeStr := fmt.Sprintf("%s-%s", namePartsAll[0], namePartsAll[1])
	createdTime, errTime := time.Parse("20060102-150405", timeStr) // 20240511-211121
	if errTime != nil {
		slog.Warn("Failed to parse demo time, using current time", slog.String("time", timeStr))

		createdTime = time.Now()
	}

	newDemo := File{
		ServerID:  serverID,
		Title:     parsedDemo.Server,
		CreatedOn: createdTime,
		MapName:   mapName,
		Stats:     intStats,
		AssetID:   asset.AssetID,
	}

	if errSave := d.repository.SaveDemo(ctx, &newDemo); errSave != nil {
		return nil, errSave
	}

	sort.Slice(parsedDemo.Chat, func(i, j int) bool {
		return parsedDemo.Chat[i].Tick < parsedDemo.Chat[j].Tick
	})

	var matchID *uuid.UUID
	if createStats {
		newMatchID, errStats := d.stats.Import(ctx, serverID, newDemo.DemoID, parsedDemo, createdTime)
		if errStats != nil {
			return nil, errStats
		}
		matchID = newMatchID
		slog.Info("Generated match results", slog.String("match_id", matchID.String()))
	}

	if len(parsedDemo.Chat) > 0 {
		if errChat := d.importChatMessages(ctx, serverID, newDemo.DemoID, parsedDemo, createdTime, matchID); errChat != nil {
			return nil, errChat
		}
	}

	return &newDemo, nil
}

func (d Demos) importChatMessages(ctx context.Context, serverID int32, demoID int32, parsedDemo *demoparse.Demo, startTime time.Time, matchID *uuid.UUID) error {
	for _, msg := range parsedDemo.Chat {
		if msg.User == "BOT" {
			continue
		}

		sid := steamid.New(msg.User)
		if !sid.Valid() {
			slog.Warn("Got invalid steamid from demo chat", slog.String("name", msg.User))

			continue
		}
		userName := parsedDemo.UserName(sid)

		if err := d.chat.AddChatHistory(ctx, &chat.Message{
			ServerID:    serverID,
			DemoID:      &demoID,
			DemoTick:    &msg.Tick,
			Body:        msg.Message,
			PersonaName: userName,
			CreatedOn:   startTime.Add(ticksToDuration(msg.Tick)),
			SteamID:     sid,
			MatchID:     matchID,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (d Demos) onDemoReceived(ctx context.Context, demo UploadedDemo) error {
	slog.Debug("Got new demo",
		slog.Int("server_id", int(demo.ServerID)),
		slog.String("name", demo.Name))

	// TOOO make these interfaces less clunky for compressed data.
	compressed := new(bytes.Buffer)
	reader := bytes.NewReader(demo.Content)
	if err := zstd.Compress(reader, compressed); err != nil {
		return err
	}
	compressedData := compressed.Bytes()

	demoAsset, errNewAsset := d.asset.Create(ctx, d.owner,
		asset.BucketDemo, demo.Name+zstd.Extension, bytes.NewReader(compressedData), false)
	if errNewAsset != nil {
		return errNewAsset
	}

	if _, errDemo := d.createFromAsset(ctx, &demoAsset, demo.ServerID, true); errDemo != nil {
		// Cleanup the asset not attached to a valid demo
		if _, errDelete := d.asset.Delete(ctx, demoAsset.AssetID); errDelete != nil {
			return errors.Join(errDelete, errDelete)
		}

		return errDemo
	}

	return nil
}

func (d Demos) ImportFile(ctx context.Context, serverID int32, demoPath string, createStats bool) (*File, error) {
	demoFile, err := os.Open(demoPath)
	if err != nil {
		return nil, errors.Join(err, ErrDemoLoad)
	}
	defer demoFile.Close()

	demoFileName := filepath.Base(demoFile.Name())
	demoAsset, errAsset := d.asset.Create(ctx, d.owner, asset.BucketDemo, demoFileName, demoFile, false)
	if errAsset != nil {
		return nil, errors.Join(errAsset, ErrDemoLoad)
	}

	demo, errDemo := d.createFromAsset(ctx, &demoAsset, serverID, createStats)
	if errDemo != nil {
		return nil, errors.Join(errDemo, ErrDemoLoad)
	}

	return demo, nil
}

func (d Demos) DownloadHandler(ctx context.Context, client storage.Storager, server scp.ServerInfo, config *scp.Config) error {
	for _, instance := range server.ServerIDs {
		demoDir := server.GamePath(config.DemoPathFmt, instance)
		filelist, errFilelist := client.List(ctx, demoDir)
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

func (d Demos) oldest(ctx context.Context) (Info, error) {
	demos, errDemos := d.repository.ExpiredDemos(ctx, 1)
	if errDemos != nil {
		return Info{}, errDemos
	}

	if len(demos) == 0 {
		return Info{}, database.ErrNoResult
	}

	return demos[0], nil
}

func (d Demos) MarkArchived(ctx context.Context, demo *File) error {
	demo.Archive = true

	if err := d.repository.SaveDemo(ctx, demo); err != nil {
		slog.Error("Failed to mark demo as archived", slog.String("error", err.Error()), slog.Int64("demo_id", int64(demo.DemoID)))
	}

	slog.Debug("Demo marked as archived", slog.Int64("demo_id", int64(demo.DemoID)))

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

		demoSize, err := d.asset.SoftDelete(ctx, oldestDemo.AssetID)
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
		demoSize, errDrop := d.asset.SoftDelete(ctx, demo.AssetID)
		if errDrop != nil && !errors.Is(errDrop, asset.ErrDeleteAssetFile) {
			slog.Error("Failed to soft-delete demo asset", slog.String("error", errDrop.Error()),
				slog.String("bucket", string(d.bucket)), slog.String("name", demo.Title))

			continue
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

func (d Demos) ExpiredDemos(ctx context.Context, limit uint64) ([]Info, error) {
	return d.repository.ExpiredDemos(ctx, limit)
}

func (d Demos) GetDemoByID(ctx context.Context, demoID int32) (*File, error) {
	return d.repository.GetDemoByID(ctx, demoID)
}

func (d Demos) GetDemoByName(ctx context.Context, demoName string) (*File, error) {
	return d.repository.GetDemoByName(ctx, demoName)
}

func (d Demos) GetDemos(ctx context.Context) ([]File, error) {
	return d.repository.GetDemos(ctx)
}

// Were just going to assume the server is relatively consistent, it doesnt matter too much.
const frameDuration = 16600 * time.Microsecond

func ticksToDuration(ticks int32) time.Duration {
	return (frameDuration) * time.Duration(ticks)
}

func (d Demos) RemoveOrphans(ctx context.Context) error {
	demos, errDemos := d.GetDemos(ctx)
	if errDemos != nil {
		return errDemos
	}

	for _, demo := range demos {
		var remove bool
		realAsset, errAsset := d.asset.Get(ctx, demo.AssetID)
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

		slog.Debug("Removing orphan demo", slog.Int64("demo_id", int64(demo.DemoID)),
			slog.String("title", demo.Title), slog.String("asset_id", demo.AssetID.String()))
		if _, err := d.asset.SoftDelete(ctx, demo.AssetID); err != nil {
			slog.Error("Failed to soft-delete orphan demo asset", slog.String("error", err.Error()))

			continue
		}

		// TODO delete empty folders
		slog.Warn("Removed orphan demo file", slog.String("filename", demo.Title))
	}

	return nil
}
