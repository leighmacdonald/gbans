package demo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/demoparser"
	"github.com/leighmacdonald/gbans/pkg/fs"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/ricochet2200/go-disk-usage/du"
)

type demoUsecase struct {
	repository  domain.DemoRepository
	asset       domain.AssetUsecase
	config      domain.ConfigUsecase
	servers     domain.ServersUsecase
	bucket      domain.Bucket
	cleanupChan chan any
}

func NewDemoUsecase(bucket domain.Bucket, repository domain.DemoRepository, assets domain.AssetUsecase,
	config domain.ConfigUsecase, servers domain.ServersUsecase,
) domain.DemoUsecase {
	return &demoUsecase{
		bucket:      bucket,
		repository:  repository,
		asset:       assets,
		config:      config,
		servers:     servers,
		cleanupChan: make(chan any),
	}
}

func (d demoUsecase) oldest(ctx context.Context) (domain.DemoInfo, error) {
	demos, errDemos := d.repository.ExpiredDemos(ctx, 1)
	if errDemos != nil {
		return domain.DemoInfo{}, errDemos
	}

	if len(demos) == 0 {
		return domain.DemoInfo{}, domain.ErrNoResult
	}

	return demos[0], nil
}

func (d demoUsecase) MarkArchived(ctx context.Context, demo *domain.DemoFile) error {
	demo.Archive = true

	if err := d.repository.SaveDemo(ctx, demo); err != nil {
		slog.Error("Failed to mark demo as archived", log.ErrAttr(err), slog.Int64("demo_id", demo.DemoID))
	}

	slog.Debug("Demo marked as archived", slog.Int64("demo_id", demo.DemoID))

	return nil
}

func diskPercentageUsed(path string) float32 {
	info := du.NewDiskUsage(path)

	return info.Usage() * 100
}

func (d demoUsecase) TruncateBySpace(ctx context.Context, root string, maxAllowedPctUsed float32) (int, int64, error) {
	var (
		count int
		size  int64
	)

	defer func() {
		slog.Debug("Truncate by space completed", slog.Int("count", count), slog.String("total_size", humanize.Bytes(uint64(size))))
	}()

	for {
		usedSpace := diskPercentageUsed(root)

		if usedSpace < maxAllowedPctUsed {
			return count, size, nil
		}

		oldestDemo, errOldest := d.oldest(ctx)
		if errOldest != nil {
			if errors.Is(errOldest, domain.ErrNoResult) {
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

func (d demoUsecase) TruncateByCount(ctx context.Context, maxCount uint64) (int, int64, error) {
	var (
		count int
		size  int64
	)

	expired, errExpired := d.repository.ExpiredDemos(ctx, maxCount)
	if errExpired != nil {
		if errors.Is(errExpired, domain.ErrNoResult) {
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
		if errDrop != nil {
			slog.Error("Failed to remove demo asset", log.ErrAttr(errDrop),
				slog.String("bucket", string(d.bucket)), slog.String("name", demo.Title))

			continue
		}

		if err := d.repository.Delete(ctx, demo.DemoID); err != nil {
			slog.Error("Failed to remove demo entry",
				slog.Int64("demo_id", demo.DemoID),
				slog.String("asset_id", demo.AssetID.String()),
				log.ErrAttr(err))
		}

		size += demoSize
		count++
	}

	slog.Debug("Truncate by count completed", slog.Int("count", count), slog.String("total_size", humanize.Bytes(uint64(size))))

	return count, size, nil
}

func (d demoUsecase) Cleanup(ctx context.Context) {
	conf := d.config.Config()

	if !conf.Demo.DemoCleanupEnabled {
		return
	}

	slog.Debug("Starting demo cleanup", slog.String("strategy", string(conf.Demo.DemoCleanupStrategy)))

	var (
		count int
		err   error
		size  int64
	)

	switch conf.Demo.DemoCleanupStrategy {
	case domain.DemoStrategyPctFree:
		count, size, err = d.TruncateBySpace(ctx, conf.Demo.DemoCleanupMount, conf.Demo.DemoCleanupMinPct)
	case domain.DemoStrategyCount:
		count, size, err = d.TruncateByCount(ctx, conf.Demo.DemoCountLimit)
	}

	if err != nil {
		slog.Error("Error executing demo cleanup", slog.String("strategy", string(conf.Demo.DemoCleanupStrategy)))
	}

	slog.Debug("Old demos flushed", slog.Int("count", count), slog.String("size", humanize.Bytes(uint64(size))))
}

func (d demoUsecase) TriggerCleanup() {
	d.cleanupChan <- true
}

func (d demoUsecase) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	tickerOrphans := time.NewTicker(time.Hour * 24)

	d.Cleanup(ctx)

	if err := d.RemoveOrphans(ctx); err != nil {
		slog.Error("Failed to execute orphans", log.ErrAttr(err))
	}

	for {
		select {
		case <-ticker.C:
			d.cleanupChan <- true
		case <-d.cleanupChan:
			d.Cleanup(ctx)
		case <-tickerOrphans.C:
			if err := d.RemoveOrphans(ctx); err != nil {
				slog.Error("Failed to execute orphans", log.ErrAttr(err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (d demoUsecase) ExpiredDemos(ctx context.Context, limit uint64) ([]domain.DemoInfo, error) {
	return d.repository.ExpiredDemos(ctx, limit)
}

func (d demoUsecase) GetDemoByID(ctx context.Context, demoID int64, demoFile *domain.DemoFile) error {
	return d.repository.GetDemoByID(ctx, demoID, demoFile)
}

func (d demoUsecase) GetDemoByName(ctx context.Context, demoName string, demoFile *domain.DemoFile) error {
	return d.repository.GetDemoByName(ctx, demoName, demoFile)
}

func (d demoUsecase) GetDemos(ctx context.Context) ([]domain.DemoFile, error) {
	return d.repository.GetDemos(ctx)
}

func (d demoUsecase) CreateFromAsset(ctx context.Context, asset domain.Asset, serverID int) (*domain.DemoFile, error) {
	_, errGetServer := d.servers.Server(ctx, serverID)
	if errGetServer != nil {
		return nil, domain.ErrGetServer
	}

	namePartsAll := strings.Split(asset.Name, "-")

	var mapName string

	if strings.Contains(asset.Name, "workshop-") {
		// 20231221-042605-workshop-cp_overgrown_rc8-ugc503939302.dem
		mapName = namePartsAll[3]
	} else {
		// 20231112-063943-koth_harvest_final.dem
		nameParts := strings.Split(namePartsAll[2], ".")
		mapName = nameParts[0]
	}

	intStats := map[string]gin.H{}

	// temp thing until proper demo parsing is implemented
	if d.config.Config().General.Mode != domain.TestMode {
		var demoInfo demoparser.DemoInfo
		if errParse := demoparser.Parse(ctx, asset.LocalPath, &demoInfo); errParse != nil {
			return nil, errParse
		}

		for _, steamID := range demoInfo.SteamIDs() {
			intStats[steamID.String()] = gin.H{}
		}
	} else {
		intStats[d.config.Config().Owner] = gin.H{}
	}

	timeStr := fmt.Sprintf("%s-%s", namePartsAll[0], namePartsAll[1])

	createdTime, errTime := time.Parse("20060102-150405", timeStr) // 20240511-211121
	if errTime != nil {
		slog.Warn("Failed to parse demo time, using current time", slog.String("time", timeStr))

		createdTime = time.Now()
	}

	newDemo := domain.DemoFile{
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

func (d demoUsecase) RemoveOrphans(ctx context.Context) error {
	demos, errDemos := d.GetDemos(ctx)
	if errDemos != nil {
		return errDemos
	}

	for _, demo := range demos {
		var remove bool
		asset, _, errAsset := d.asset.Get(ctx, demo.AssetID)
		if errAsset != nil {
			if errors.Is(errAsset, domain.ErrNoResult) {
				remove = true
			} else {
				return errAsset
			}
		} else {
			localPath, errPath := d.asset.GenAssetPath(asset.HashString())
			if errPath != nil {
				return errPath
			}

			remove = !fs.Exists(localPath)
		}

		if !remove {
			continue
		}

		if _, err := d.asset.Delete(ctx, asset.AssetID); err != nil {
			slog.Error("Failed to remove orphan demo asset", log.ErrAttr(err))
		}

		if err := d.repository.Delete(ctx, demo.DemoID); err != nil {
			slog.Error("Failed to remove orphan demo entry", log.ErrAttr(err))
		}

		slog.Info("Removed orphan demo file", slog.String("filename", demo.Title))
	}

	return nil
}
