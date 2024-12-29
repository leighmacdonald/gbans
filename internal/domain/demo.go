package domain

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
)

type DemoUsecase interface {
	ExpiredDemos(ctx context.Context, limit uint64) ([]DemoInfo, error)
	GetDemoByID(ctx context.Context, demoID int64, demoFile *DemoFile) error
	MarkArchived(ctx context.Context, demo *DemoFile) error
	GetDemoByName(ctx context.Context, demoName string, demoFile *DemoFile) error
	GetDemos(ctx context.Context) ([]DemoFile, error)
	CreateFromAsset(ctx context.Context, asset Asset, serverID int) (*DemoFile, error)
	Cleanup(ctx context.Context)
}

type DemoRepository interface {
	ExpiredDemos(ctx context.Context, limit uint64) ([]DemoInfo, error)
	GetDemoByID(ctx context.Context, demoID int64, demoFile *DemoFile) error
	GetDemoByName(ctx context.Context, demoName string, demoFile *DemoFile) error
	GetDemos(ctx context.Context) ([]DemoFile, error)
	SaveDemo(ctx context.Context, demoFile *DemoFile) error
	Delete(ctx context.Context, demoID int64) error
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
	DemoID          int64            `json:"demo_id"`
	ServerID        int              `json:"server_id"`
	ServerNameShort string           `json:"server_name_short"`
	ServerNameLong  string           `json:"server_name_long"`
	Title           string           `json:"title"`
	CreatedOn       time.Time        `json:"created_on"`
	Downloads       int64            `json:"downloads"`
	Size            int64            `json:"size"`
	MapName         string           `json:"map_name"`
	Archive         bool             `json:"archive"` // When true, will not get auto deleted when flushing old demos
	Stats           map[string]gin.H `json:"stats"`
	AssetID         uuid.UUID        `json:"asset_id"`
}

const DemoType = "HL2DEMO"

type DemoInfo struct {
	DemoID  int64
	Title   string
	AssetID uuid.UUID
}
