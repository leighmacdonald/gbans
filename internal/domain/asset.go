package domain

import "context"

type AssetUsecase interface {
	SaveAsset(ctx context.Context, asset *Asset) error
	ExpiredDemos(ctx context.Context, limit uint64) ([]DemoInfo, error)
	GetDemoByID(ctx context.Context, demoID int64, demoFile *DemoFile) error
	GetDemoByName(ctx context.Context, demoName string, demoFile *DemoFile) error
	GetDemos(ctx context.Context, opts DemoFilter) ([]DemoFile, int64, error)
	SaveDemo(ctx context.Context, demoFile *DemoFile) error
	DropDemo(ctx context.Context, demoFile *DemoFile) error
}

type UserUploadedFile struct {
	Content string `json:"content"`
	Name    string `json:"name"`
	Mime    string `json:"mime"`
	Size    int64  `json:"size"`
}
