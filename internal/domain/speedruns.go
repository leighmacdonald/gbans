package domain

import (
	"github.com/leighmacdonald/steamid/v4/steamid"
	"time"
)

type SpeedrunRepository interface {
	Query(query SpeedrunQuery) ([]SpeedrunDetails, error)
	Save(details SpeedrunDetails) error
}

type SpeedrunUsecase interface {
	RoundFinish(details SpeedrunDetails) error
	Query(query SpeedrunQuery) ([]SpeedrunDetails, error)
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
	Players []SpeedrunPlayer `json:"players"`
}

type SpeedrunDetails struct {
	Rounds  []SpeedrunRound  `json:"rounds"`
	Players []SpeedrunPlayer `json:"players"`
	Time    int              `json:"time"`
}

func (sr SpeedrunDetails) Duration() time.Duration {
	return time.Duration(sr.Time) * time.Second
}

type SpeedrunPlayer struct {
	SteamID steamid.SteamID
	Time    int
}

type SpeedrunResult struct {
	MapName string
	Players UserProfile
}
