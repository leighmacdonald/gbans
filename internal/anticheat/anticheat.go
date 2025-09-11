package anticheat

import (
	"context"
	"io"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type AnticheatEntry struct {
	logparse.StacEntry
	Personaname string `json:"personaname"`
	AvatarHash  string `json:"avatar_hash"`
	Triggered   int    `json:"triggered"`
}

type AntiCheatRepository interface {
	DetectionsBySteamID(ctx context.Context, steamID steamid.SteamID) ([]logparse.StacEntry, error)
	DetectionsByType(ctx context.Context, detectionType logparse.Detection) ([]logparse.StacEntry, error)
	SaveEntries(ctx context.Context, entries []logparse.StacEntry) error
	SyncDemoIDs(ctx context.Context, limit uint64) error
	Query(ctx context.Context, query AnticheatQuery) ([]AnticheatEntry, error)
}

type AntiCheatUsecase interface {
	DetectionsBySteamID(ctx context.Context, steamID steamid.SteamID) ([]logparse.StacEntry, error)
	DetectionsByType(ctx context.Context, detectionType logparse.Detection) ([]logparse.StacEntry, error)
	Import(ctx context.Context, fileName string, reader io.ReadCloser, serverID int) ([]logparse.StacEntry, error)
	SyncDemoIDs(ctx context.Context, limit uint64) error
	Query(ctx context.Context, query AnticheatQuery) ([]AnticheatEntry, error)
	Handle(ctx context.Context, entries []logparse.StacEntry) error
}

type AnticheatQuery struct {
	domain.QueryFilter
	Name      string             `json:"name" schema:"name"`
	SteamID   string             `json:"steam_id" schema:"steam_id"`
	ServerID  int                `json:"server_id" schema:"server_id"`
	Summary   string             `json:"summary" schema:"summary"`
	Detection logparse.Detection `json:"detection" schema:"detection"`
}
