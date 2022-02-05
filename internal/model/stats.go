package model

import (
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"time"
)

// Match and its related Match* structs are designed as a close to 1:1 mirror of the
// logs.tf ui
type Match struct {
	Title             string
	Map               string
	PlayerSums        []MatchPlayerSum
	MedicSums         []MatchMedicSum
	TeamSums          map[logparse.Team]MatchTeamSum
	Rounds            []MatchRoundSum
	ClassKills        MatchPlayerClassSums
	ClassKillsAssists MatchPlayerClassSums
	ClassDeaths       MatchPlayerClassSums
}

type MatchPlayerSum struct {
	Team         logparse.Team
	Name         string
	TimeStart    time.Time
	TimeEnd      time.Time
	Kills        int
	Assists      int
	Deaths       int
	Damage       int
	DamagePerMin int
	DamageTaken  int
	HealthPacks  int
	BackStabs    int
	HeadShots    int
	Airshots     int
	Captures     int
	Classes      []logparse.PlayerClass
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
	Charges             map[logparse.Weapon]int
	Drops               int
	AvgTimeToBuild      int
	AvgTimeBeforeUse    int
	NearFullChargeDeath int
	AvgUberLength       float32
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
	Spy      int
	Sniper   int
}

func (m MatchClassSums) Sum() int {
	return m.Scout + m.Soldier + m.Pyro +
		m.Demoman + m.Heavy + m.Engineer +
		m.Medic + m.Spy + m.Sniper
}

type MatchPlayerClassSums map[steamid.SID64]MatchClassSums

type MatchTeamSum struct {
	Kills        int
	Damage       int
	Charges      int
	Drops        int
	Caps         int
	MidFightsRed map[logparse.Team]int
}
