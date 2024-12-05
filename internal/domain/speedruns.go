package domain

import (
	"context"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

type SpeedrunRepository interface {
	Query(ctx context.Context, query SpeedrunQuery) ([]Speedrun, error)
	Save(ctx context.Context, details *Speedrun) error
}

type SpeedrunUsecase interface {
	Save(ctx context.Context, details Speedrun) (Speedrun, error)
	Query(ctx context.Context, query SpeedrunQuery) ([]Speedrun, error)
}

type SpeedrunInterval int

const (
	Daily   SpeedrunInterval = 86400
	Weekly                   = Daily * 7
	Monthly                  = Weekly * 7
	Yearly                   = Monthly * 12
	AllTime                  = -1
)

type SpeedrunQuery struct {
	Map      string           `json:"map"`
	Interval SpeedrunInterval `json:"interval"`
	Count    int              `json:"count"`
}

type SpeedrunPointCaptures struct {
	RoundID  int                   `json:"round_id"`
	Players  []SpeedrunParticipant `json:"players"`
	Duration time.Duration         `json:"duration"`
}

type Speedrun struct {
	SpeedrunID    int                     `json:"speedrun_id"`
	MapName       string                  `json:"map_name"`
	PointCaptures []SpeedrunPointCaptures `json:"point_captures"`
	Players       []SpeedrunParticipant   `json:"players"`
	Duration      int                     `json:"duration"`
	PlayerCount   int                     `json:"player_count"`
	HostAddr      string                  `json:"host_addr"`
	BotCount      int                     `json:"bot_count"`
	CreatedOn     time.Time               `json:"created_on"`
	Category      string                  `json:"category"`
}

func (sr Speedrun) AsDuration() time.Duration {
	return time.Duration(sr.Duration) * time.Millisecond
}

type SpeedrunParticipant struct {
	SteamID  steamid.SteamID `json:"steam_id"`
	Duration time.Duration   `json:"duration"`
}
