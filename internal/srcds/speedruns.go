package srcds

import (
	"context"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

type SpeedrunRepository interface {
	Save(ctx context.Context, details *Speedrun) error
	ByID(ctx context.Context, speedrunID int) (Speedrun, error)
	ByMap(ctx context.Context, name string) ([]SpeedrunMapOverview, error)
	Recent(ctx context.Context, limit int) ([]SpeedrunMapOverview, error)
	TopNOverall(ctx context.Context, count int) (map[string][]Speedrun, error)
}

type SpeedrunUsecase interface {
	Save(ctx context.Context, details Speedrun) (Speedrun, error)
	ByID(ctx context.Context, speedrunID int) (Speedrun, error)
	ByMap(ctx context.Context, name string) ([]SpeedrunMapOverview, error)
	Recent(ctx context.Context, limit int) ([]SpeedrunMapOverview, error)
	TopNOverall(ctx context.Context, count int) (map[string][]Speedrun, error)
}

type SpeedrunInterval int

const (
	Daily   SpeedrunInterval = 86400
	Weekly                   = Daily * 7
	Monthly                  = Weekly * 7
	Yearly                   = Monthly * 12
	AllTime                  = -1
)

type SpeedrunCategory string

const (
	Mode24v40 SpeedrunCategory = "24_40"
)

type MapDetail struct {
	MapID     int       `json:"map_id"`
	MapName   string    `json:"map_name"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
}

type SpeedrunQuery struct {
	Map      string           `json:"map"`
	Interval SpeedrunInterval `json:"interval"`
	Count    int              `json:"count"`
}

type SpeedrunPointCaptures struct {
	SpeedrunID int                   `json:"speedrun_id"`
	RoundID    int                   `json:"round_id"`
	Players    []SpeedrunParticipant `json:"players"`
	Duration   time.Duration         `json:"duration"`
	PointName  string                `json:"point_name"`
}

type Speedrun struct {
	SpeedrunID    int                     `json:"speedrun_id"`
	ServerID      int                     `json:"server_id"`
	Rank          int                     `json:"rank,omitempty"`
	InitialRank   int                     `json:"initial_rank,omitempty"`
	MapDetail     MapDetail               `json:"map_detail"`
	PointCaptures []SpeedrunPointCaptures `json:"point_captures"`
	Players       []SpeedrunParticipant   `json:"players"`
	Duration      time.Duration           `json:"duration"`
	PlayerCount   int                     `json:"player_count"`
	BotCount      int                     `json:"bot_count"`
	CreatedOn     time.Time               `json:"created_on"`
	Category      SpeedrunCategory        `json:"category"`
	TotalPlayers  int                     `json:"total_players"`
}

type SpeedrunParticipant struct {
	RoundID     int             `json:"round_id"`
	SteamID     steamid.SteamID `json:"steam_id"`
	Duration    time.Duration   `json:"duration"`
	AvatarHash  string          `json:"avatar_hash"`
	PersonaName string          `json:"persona_name"`
}

type SpeedrunMapOverview struct {
	SpeedrunID   int              `json:"speedrun_id"`
	ServerID     int              `json:"server_id"`
	Rank         int              `json:"rank"`
	InitialRank  int              `json:"initial_rank"`
	MapDetail    MapDetail        `json:"map_detail"`
	Duration     time.Duration    `json:"duration"`
	PlayerCount  int              `json:"player_count"`
	BotCount     int              `json:"bot_count"`
	CreatedOn    time.Time        `json:"created_on"`
	Category     SpeedrunCategory `json:"category"`
	TotalPlayers int              `json:"total_players"`
}
