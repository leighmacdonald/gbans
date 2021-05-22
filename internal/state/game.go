package state

import (
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"time"
)

type WeaponStats struct {
	Shots int
	Hits int
}

type Player struct {
	WeaponCurrent logparse.Weapon
	Kills []logparse.KilledCustomEvt
	Assists []logparse.KillAssistEvt
	Damage int
	Healing int
	ShotFired map[logparse.Weapon]WeaponStats
	Classes []logparse.PlayerClass
	IsConnected bool
	ConnectedAt time.Time
	DisconnectedAt time.Time
}

func newPlayer() *Player {
	return &Player{
		ShotFired: map[logparse.Weapon]WeaponStats{},
	}
}

type Game struct {
	ServerID string
	Name string
	Map string
	ScoreRed int
	ScoreBlu int
	MapLoadedAt time.Time
	MapEndedAt time.Time
	RoundStartedAt time.Time
	RoundEndedAt time.Time
	Players []*Player
}

func newGame() *Game {
	return &Game{
		MapLoadedAt: config.Now(),
	}
}