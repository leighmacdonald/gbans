package demo

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/demoparser"
	"github.com/leighmacdonald/gbans/pkg/log"
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
				if errDrop := d.DropDemo(ctx, &domain.DemoFile{DemoID: demo.DemoID, Title: demo.Title}); errDrop != nil {
					slog.Error("Failed to remove demo", log.ErrAttr(errDrop),
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

	newDemo := domain.DemoFile{
		ServerID:  serverID,
		Title:     asset.Name,
		CreatedOn: time.Now(),
		MapName:   mapName,
		Stats:     intStats,
		AssetID:   asset.AssetID,
	}

	if errSave := d.repository.SaveDemo(ctx, &newDemo); errSave != nil {
		return nil, errSave
	}

	return &newDemo, nil
}

func (d demoUsecase) DropDemo(ctx context.Context, demoFile *domain.DemoFile) error {
	asset, _, errAsset := d.assetUsecase.Get(ctx, demoFile.AssetID)
	if errAsset == nil {
		// TODO assets should exist, but can be missing
		if errRemove := d.assetUsecase.Delete(ctx, demoFile.AssetID); errRemove != nil {
			slog.Warn("Failed to remove demo asset from S3Store",
				log.ErrAttr(errRemove), slog.String("bucket", string(domain.BucketDemo)), slog.String("name", demoFile.Title))
		}

		if err := d.repository.DropDemo(ctx, demoFile); err != nil {
			return err
		}
	}

	slog.Debug("Demo expired and removed",
		slog.String("bucket", string(asset.Bucket)), slog.String("name", asset.Name))

	return nil
}
