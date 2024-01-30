package demo

import (
	"context"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

type demoUsecase struct {
	dr     domain.DemoRepository
	au     domain.AssetUsecase
	cu     domain.ConfigUsecase
	bucket string
	log    *zap.Logger
}

func NewDemoUsecase(log *zap.Logger, bucket string, dr domain.DemoRepository, au domain.AssetUsecase, cu domain.ConfigUsecase) domain.DemoUsecase {
	return &demoUsecase{log: log.Named("demo"), bucket: bucket, dr: dr, au: au, cu: cu}
}

func (d demoUsecase) Start(ctx context.Context) {
	log := d.log.Named("demoCleaner")
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
			conf := d.cu.Config()

			if !conf.General.DemoCleanupEnabled {
				continue
			}

			log.Debug("Starting demo cleanup")

			expired, errExpired := d.dr.ExpiredDemos(ctx, conf.General.DemoCountLimit)
			if errExpired != nil {
				if errors.Is(errExpired, domain.ErrNoResult) {
					continue
				}

				log.Error("Failed to fetch expired demos", zap.Error(errExpired))
			}

			if len(expired) == 0 {
				continue
			}

			count := 0

			for _, demo := range expired {
				if errDrop := d.DropDemo(ctx, &domain.DemoFile{DemoID: demo.DemoID, Title: demo.Title}); errDrop != nil {
					log.Error("Failed to remove demo", zap.Error(errDrop),
						zap.String("bucket", conf.S3.BucketDemo), zap.String("name", demo.Title))

					continue
				}

				count++
			}

			log.Info("Old demos flushed", zap.Int("count", count))
		case <-ctx.Done():
			log.Debug("demoCleaner shutting down")

			return
		}
	}
}

func (d demoUsecase) ExpiredDemos(ctx context.Context, limit uint64) ([]domain.DemoInfo, error) {
	return d.dr.ExpiredDemos(ctx, limit)
}

func (d demoUsecase) GetDemoByID(ctx context.Context, demoID int64, demoFile *domain.DemoFile) error {
	return d.GetDemoByID(ctx, demoID, demoFile)
}

func (d demoUsecase) GetDemoByName(ctx context.Context, demoName string, demoFile *domain.DemoFile) error {
	return d.dr.GetDemoByName(ctx, demoName, demoFile)
}

func (d demoUsecase) GetDemos(ctx context.Context, opts domain.DemoFilter) ([]domain.DemoFile, int64, error) {
	return d.dr.GetDemos(ctx, opts)
}

func (d demoUsecase) Create(ctx context.Context, name string, content []byte, mapName string, intStats map[steamid.SID64]gin.H, serverID int) (*domain.DemoFile, error) {
	asset, errAsset := domain.NewAsset(content, d.bucket, name)
	if errAsset != nil {
		return nil, errAsset
	}

	if errPut := d.au.SaveAsset(ctx, d.bucket, &asset, content); errPut != nil {
		return nil, errPut
	}

	newDemo := domain.DemoFile{
		ServerID:  serverID,
		Title:     asset.Name,
		CreatedOn: time.Now(),
		MapName:   mapName,
		Stats:     intStats,
		AssetID:   asset.AssetID,
	}

	if errSave := d.dr.SaveDemo(ctx, &newDemo); errSave != nil {
		return nil, errSave
	}

	return &newDemo, nil
}

func (d demoUsecase) DropDemo(ctx context.Context, demoFile *domain.DemoFile) error {
	conf := d.cu.Config()

	asset, errAsset := d.au.GetAsset(ctx, demoFile.AssetID)
	if errAsset != nil {
		return errAsset
	}

	if errRemove := d.au.DropAsset(ctx, asset); errRemove != nil {
		d.log.Error("Failed to remove demo asset from S3",
			zap.Error(errRemove), zap.String("bucket", conf.S3.BucketDemo), zap.String("name", demoFile.Title))

		return errRemove
	}

	if err := d.dr.DropDemo(ctx, demoFile); err != nil {
		return err
	}

	d.log.Info("Demo expired and removed",
		zap.String("bucket", conf.S3.BucketDemo), zap.String("name", demoFile.Title))

	return nil
}
