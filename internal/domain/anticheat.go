package domain

import (
	"context"
	"io"

	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type AntiCheatRepository interface {
	DetectionsBySteamID(ctx context.Context, steamID steamid.SteamID) ([]logparse.StacEntry, error)
	DetectionsByType(ctx context.Context, detectionType logparse.Detection) ([]logparse.StacEntry, error)
	SaveEntries(ctx context.Context, entries []logparse.StacEntry) error
	SyncDemoIDs(ctx context.Context, limit uint64) error
}

type AntiCheatUsecase interface {
	DetectionsBySteamID(ctx context.Context, steamID steamid.SteamID) ([]logparse.StacEntry, error)
	DetectionsByType(ctx context.Context, detectionType logparse.Detection) ([]logparse.StacEntry, error)
	Import(ctx context.Context, fileName string, reader io.ReadCloser, serverID int) error
	SyncDemoIDs(ctx context.Context, limit uint64) error
}
