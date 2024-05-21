package demo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/demoparser"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/ricochet2200/go-disk-usage/du"
)

type demoUsecase struct {
	repository     domain.DemoRepository
	assetUsecase   domain.AssetUsecase
	configUsecase  domain.ConfigUsecase
	serversUsecase domain.ServersUsecase
	bucket         domain.Bucket
}

func NewDemoUsecase(bucket domain.Bucket, demoRepository domain.DemoRepository, assetUsecase domain.AssetUsecase,
	configUsecase domain.ConfigUsecase, serversUsecase domain.ServersUsecase,
) domain.DemoUsecase {
	return &demoUsecase{
		bucket:         bucket,
		repository:     demoRepository,
		assetUsecase:   assetUsecase,
		configUsecase:  configUsecase,
		serversUsecase: serversUsecase,
	}
}

func (d demoUsecase) OldestDemo(ctx context.Context) (domain.DemoInfo, error) {
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

	return d.repository.SaveDemo(ctx, demo)
}

func diskPercentageUsed(path string) float32 {
	info := du.NewDiskUsage(path)

	return info.Usage() * 100
}

func (d demoUsecase) truncateBySpace(ctx context.Context, root string, maxAllowedPctUsed float32) {
	for {
		usedSpace := diskPercentageUsed(root)

		if usedSpace < maxAllowedPctUsed {
			return
		}

		oldestDemo, errOldest := d.OldestDemo(ctx)
		if errOldest != nil {
			if errors.Is(errOldest, domain.ErrNoResult) {
				return
			}

			slog.Error("Failed to fetch oldest demo", log.ErrAttr(errOldest))

			return
		}

		if err := d.assetUsecase.Delete(ctx, oldestDemo.AssetID); err != nil {
			slog.Error("Failed to fetch oldest demo", log.ErrAttr(errOldest))

			return
		}

		slog.Debug("Pruned demo", slog.String("demo", oldestDemo.Title), slog.Float64("free_pct", float64(usedSpace)))
	}
}

func (d demoUsecase) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	triggerChan := make(chan any)

	go func() {
		triggerChan <- true
	}()

	for {
		select {
		case <-ticker.C:
			triggerChan <- true
		case <-triggerChan:
			conf := d.configUsecase.Config()

			if !conf.General.DemoCleanupEnabled {
				continue
			}

			if conf.General.DemoCleanupStrategy == domain.DemoStrategyPctFree {
				d.truncateBySpace(ctx, conf.General.DemoCleanupMount, conf.General.DemoCleanupMinPct)

				return
			}

			slog.Debug("Starting demo cleanup")

			expired, errExpired := d.repository.ExpiredDemos(ctx, conf.General.DemoCountLimit)
			if errExpired != nil {
				if errors.Is(errExpired, domain.ErrNoResult) {
					continue
				}

				slog.Error("Failed to fetch expired demos", log.ErrAttr(errExpired))
			}

			if len(expired) == 0 {
				continue
			}

			count := 0

			for _, demo := range expired {
				// Dropping asset will cascade to demo
				if errDrop := d.assetUsecase.Delete(ctx, demo.AssetID); errDrop != nil {
					slog.Error("Failed to remove demo asset", log.ErrAttr(errDrop),
						slog.String("bucket", string(d.bucket)), slog.String("name", demo.Title))

					continue
				}

				count++
			}

			slog.Info("Old demos flushed", slog.Int("count", count))
		case <-ctx.Done():
			slog.Debug("demoCleaner shutting down")

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
	_, errGetServer := d.serversUsecase.GetServer(ctx, serverID)
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

	var demoInfo demoparser.DemoInfo
	if errParse := demoparser.Parse(ctx, asset.LocalPath, &demoInfo); errParse != nil {
		return nil, errParse
	}

	intStats := map[string]gin.H{}

	for _, steamID := range demoInfo.SteamIDs() {
		intStats[steamID.String()] = gin.H{}
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

	return &newDemo, nil
}
