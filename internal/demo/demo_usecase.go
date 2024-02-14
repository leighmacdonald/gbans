package demo

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/demoparser"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

type demoUsecase struct {
	repository     domain.DemoRepository
	assetUsecase   domain.AssetUsecase
	configUsecase  domain.ConfigUsecase
	serversUsecase domain.ServersUsecase
	bucket         string
	log            *zap.Logger
}

func NewDemoUsecase(log *zap.Logger, bucket string, demoRepository domain.DemoRepository, assetUsecase domain.AssetUsecase,
	configUsecase domain.ConfigUsecase, serversUsecase domain.ServersUsecase,
) domain.DemoUsecase {
	return &demoUsecase{
		log:            log.Named("demo"),
		bucket:         bucket,
		repository:     demoRepository,
		assetUsecase:   assetUsecase,
		configUsecase:  configUsecase,
		serversUsecase: serversUsecase,
	}
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
			conf := d.configUsecase.Config()

			if !conf.General.DemoCleanupEnabled {
				continue
			}

			log.Debug("Starting demo cleanup")

			expired, errExpired := d.repository.ExpiredDemos(ctx, conf.General.DemoCountLimit)
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
	return d.repository.ExpiredDemos(ctx, limit)
}

func (d demoUsecase) GetDemoByID(ctx context.Context, demoID int64, demoFile *domain.DemoFile) error {
	return d.repository.GetDemoByID(ctx, demoID, demoFile)
}

func (d demoUsecase) GetDemoByName(ctx context.Context, demoName string, demoFile *domain.DemoFile) error {
	return d.repository.GetDemoByName(ctx, demoName, demoFile)
}

func (d demoUsecase) GetDemos(ctx context.Context, opts domain.DemoFilter) ([]domain.DemoFile, int64, error) {
	return d.repository.GetDemos(ctx, opts)
}

func (d demoUsecase) Create(ctx context.Context, name string, content io.Reader, demoName string, serverID int) (*domain.DemoFile, error) {
	_, errGetServer := d.serversUsecase.GetServer(ctx, serverID)
	if errGetServer != nil {
		return nil, domain.ErrGetServer
	}

	demoContent, errRead := io.ReadAll(content)
	if errRead != nil {
		return nil, errRead
	}

	dir, errDir := os.MkdirTemp("", "gbans-demo")
	if errDir != nil {
		return nil, errDir
	}

	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			d.log.Error("Failed to cleanup temp demo path", zap.Error(err))
		}
	}()

	namePartsAll := strings.Split(demoName, "-")

	var mapName string

	if strings.Contains(demoName, "workshop-") {
		// 20231221-042605-workshop-cp_overgrown_rc8-ugc503939302.dem
		mapName = namePartsAll[3]
	} else {
		// 20231112-063943-koth_harvest_final.dem
		nameParts := strings.Split(namePartsAll[2], ".")
		mapName = nameParts[0]
	}

	tempPath := filepath.Join(dir, demoName)

	localFile, errLocalFile := os.Create(tempPath)
	if errLocalFile != nil {
		return nil, errLocalFile
	}

	if _, err := localFile.Write(demoContent); err != nil {
		return nil, err
	}

	_ = localFile.Close()

	var demoInfo demoparser.DemoInfo
	if errParse := demoparser.Parse(ctx, tempPath, &demoInfo); errParse != nil {
		return nil, errParse
	}

	intStats := map[steamid.SID64]gin.H{}

	for _, steamID := range demoInfo.SteamIDs() {
		intStats[steamID] = gin.H{}
	}

	fileContents, errReadContent := io.ReadAll(content)
	if errReadContent != nil {
		return nil, errors.Join(errReadContent, domain.ErrReadContent)
	}

	asset, errAsset := domain.NewAsset(fileContents, d.bucket, name)
	if errAsset != nil {
		return nil, errAsset
	}

	if errPut := d.assetUsecase.SaveAsset(ctx, d.bucket, &asset, fileContents); errPut != nil {
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

	if errSave := d.repository.SaveDemo(ctx, &newDemo); errSave != nil {
		return nil, errSave
	}

	return &newDemo, nil
}

func (d demoUsecase) DropDemo(ctx context.Context, demoFile *domain.DemoFile) error {
	conf := d.configUsecase.Config()

	asset, _, errAsset := d.assetUsecase.GetAsset(ctx, demoFile.AssetID)
	if errAsset != nil {
		return errAsset
	}

	if errRemove := d.assetUsecase.DropAsset(ctx, &asset); errRemove != nil {
		d.log.Error("Failed to remove demo asset from S3",
			zap.Error(errRemove), zap.String("bucket", conf.S3.BucketDemo), zap.String("name", demoFile.Title))

		return errRemove
	}

	if err := d.repository.DropDemo(ctx, demoFile); err != nil {
		return err
	}

	d.log.Info("Demo expired and removed",
		zap.String("bucket", conf.S3.BucketDemo), zap.String("name", demoFile.Title))

	return nil
}
