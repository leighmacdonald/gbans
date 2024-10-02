package domain

import (
	"context"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"time"
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

type SpeedrunRound struct {
	RoundID  int              `json:"round_id"`
	Players  []SpeedrunRunner `json:"players"`
	Duration time.Duration    `json:"duration"`
}

type Speedrun struct {
	SpeedrunID  int              `json:"speedrun_id"`
	MapName     string           `json:"map_name"`
	Rounds      []SpeedrunRound  `json:"rounds"`
	Players     []SpeedrunRunner `json:"players"`
	Duration    time.Duration    `json:"duration"`
	Category    string           `json:"category"`
	PlayerCount int              `json:"player_count"`
	BotCount    int              `json:"bot_count"`
	CreatedOn   time.Time        `json:"created_on"`
}

func (sr Speedrun) AsDuration() time.Duration {
	return time.Duration(sr.Duration) * time.Second
}

type SpeedrunRunner struct {
	SteamID  steamid.SteamID `json:"steam_id"`
	Duration time.Duration   `json:"duration"`
}
