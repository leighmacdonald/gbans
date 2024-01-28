package usecase

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"time"
)

type demoUsecase struct {
	dr     domain.DemoRepository
	au     domain.AssetUsecase
	bucket string
}

func NewDemoUsecase(bucket string, dr domain.DemoRepository, au domain.AssetUsecase) domain.DemoUsecase {
	return &demoUsecase{bucket: bucket, dr: dr, au: au}
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
	//TODO implement me
	panic("implement me")
}
