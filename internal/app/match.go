package app

import (
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"time"
)

// Match and its related Match* structs are designed as a close to 1:1 mirror of the
// logs.tf ui
type Match struct {
	Title             string
	Map               string
	PlayerSums        map[steamid.SID64]MatchPlayerSum
	MedicSums         map[steamid.SID64]MatchMedicSum
	TeamSums          map[logparse.Team]MatchTeamSum
	Rounds            []MatchRoundSum
	ClassKills        MatchPlayerClassSums
	ClassKillsAssists MatchPlayerClassSums
	ClassDeaths       MatchPlayerClassSums

	playerCache playerCache
}

func (m Match) Apply(event model.ServerEvent) error {
	switch event.EventType {

	}
	return nil
}

func NewMatch() Match {
	return Match{
		Title:             "",
		Map:               "",
		PlayerSums:        map[steamid.SID64]MatchPlayerSum{},
		MedicSums:         map[steamid.SID64]MatchMedicSum{},
		TeamSums:          map[logparse.Team]MatchTeamSum{},
		Rounds:            nil,
		ClassKills:        MatchPlayerClassSums{},
		ClassKillsAssists: MatchPlayerClassSums{},
		ClassDeaths:       MatchPlayerClassSums{},
	}
}

type MatchPlayerSum struct {
	Team        logparse.Team
	TimeStart   time.Time
	TimeEnd     time.Time
	Kills       int
	Assists     int
	Deaths      int
	Damage      int
	DamageTaken int
	HealthPacks int
	BackStabs   int
	HeadShots   int
	Airshots    int
	Captures    int
	Classes     []logparse.PlayerClass
}

type TeamScores struct {
	Red int
	Blu int
}

type MatchRoundSum struct {
	Length    time.Duration
	Score     TeamScores
	KillsBlu  int
	KillsRed  int
	UbersBlu  int
	UbersRed  int
	DamageBlu int
	DamageRed int
	MidFight  logparse.Team
}

type MatchMedicSum struct {
	Healing             int
	Charges             map[logparse.Medigun]int
	Drops               int
	AvgTimeToBuild      int
	AvgTimeBeforeUse    int
	NearFullChargeDeath int
	AvgUberLength       float32
	DeathAfterCharge    int
	MajorAdvLost        int
	BiggestAdvLost      int
	HealTargets         MatchPlayerClassSums
}

type MatchClassSums struct {
	Scout    int
	Soldier  int
	Pyro     int
	Demoman  int
	Heavy    int
	Engineer int
	Medic    int
	Sniper   int
	Spy      int
}

func (m MatchClassSums) Sum() int {
	return m.Scout + m.Soldier + m.Pyro +
		m.Demoman + m.Heavy + m.Engineer +
		m.Medic + m.Spy + m.Sniper
}

type MatchPlayerClassSums map[steamid.SID64]MatchClassSums

type MatchTeamSum struct {
	Kills     int
	Damage    int
	Charges   int
	Drops     int
	Caps      int
	MidFights int
}
