package demoparse

import (
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type Demo struct {
	Filename        string
	DemoType        string
	Version         int
	Protocol        int
	Server          string
	Nick            string
	Map             string
	Game            string
	Duration        float64
	Ticks           int
	Frames          int
	Signon          int
	PlayerSummaries map[string]PlayerSummary
}

type GameState struct {
	Users   map[int]Player        `json:"users"`
	Players map[int]PlayerSummary `json:"players"` //nolint:tagliatelle
	Results Results               `json:"results"` //nolint:tagliatelle
	Rounds  []DemoRoundSummary    `json:"rounds"`
	Chat    []ChatMessage         `json:"chat"`
}
type HealingSummary struct {
	PreroundHealing     int `json:"preround_healing"`
	Healing             int `json:"healing"`
	PostroundHealing    int `json:"postround_healing"`
	Drops               int `json:"drops"`
	NearFullChargeDeath int `json:"near_full_charge_death"`
	ChargesUber         int `json:"charges_uber"`
	ChargesKritz        int `json:"charges_kritz"`
	ChargesQuickfix     int `json:"charges_quickfix"`
}
type ClassSummary struct {
	Kills            int `json:"kills"`
	Assists          int `json:"assists"`
	Deaths           int `json:"deaths"`
	PostroundKills   int `json:"postround_kills"`
	PostroundAssists int `json:"postround_assists"`
	PostroundDeaths  int `json:"postround_deaths"`
	PreroundHealing  int `json:"preround_healing"`
	Healing          int `json:"healing"`
	PostroundHealing int `json:"postround_healing"`
	Damage           int `json:"damage"`
	DamageTaken      int `json:"damage_taken"`
	Dominations      int `json:"dominations"`
	Dominated        int `json:"dominated"`
	Revenges         int `json:"revenges"`
	Revenged         int `json:"revenged"`
	Airshots         int `json:"airshots"`
	HeadshotKills    int `json:"headshot_kills"`
	BackstabKills    int `json:"backstab_kills"`
	Headshots        int `json:"headshots"`
	Backstabs        int `json:"backstabs"`
	WasHeadshot      int `json:"was_headshot"`
	WasBackstabbed   int `json:"was_backstabbed"`
}

type PlayerSummary struct {
	Name             string          `json:"name"`
	Steamid          steamid.SteamID `json:"steamid"`
	Team             string          `json:"team"`
	TimeStart        int             `json:"time_start"`
	TimeEnd          int             `json:"time_end"`
	Points           int             `json:"points"`
	ConnectionCount  int             `json:"connection_count"`
	BonusPoints      int             `json:"bonus_points"`
	Kills            int             `json:"kills"`
	Assists          int             `json:"assists"`
	Deaths           int             `json:"deaths"`
	PostroundKills   int             `json:"postround_kills"`
	PostroundAssists int             `json:"postround_assists"`
	PostroundDeaths  int             `json:"postround_deaths"`
	PreroundHealing  int             `json:"preround_healing"`
	Healing          HealingSummary  `json:"healing"`
	PostroundHealing int             `json:"postround_healing"`
	Damage           int             `json:"damage"`
	DamageTaken      int             `json:"damage_taken"`
	Dominations      int             `json:"dominations"`
	Dominated        int             `json:"dominated"`
	Revenges         int             `json:"revenges"`
	Revenged         int             `json:"revenged"`
	Airshots         int             `json:"airshots"`
	HeadshotKills    int             `json:"headshot_kills"`
	BackstabKills    int             `json:"backstab_kills"`
	Headshots        int             `json:"headshots"`
	Backstabs        int             `json:"backstabs"`
	WasHeadshot      int             `json:"was_headshot"`
	WasBackstabbed   int             `json:"was_backstabbed"`
	Classes          struct {
		Pyro     ClassSummary `json:"pyro"`
		Heavy    ClassSummary `json:"heavy"`
		Soldier  ClassSummary `json:"soldier"`
		Sniper   ClassSummary `json:"sniper"`
		SpyClass ClassSummary `json:"spy"`
		Scout    ClassSummary `json:"scout"`
		Demoman  ClassSummary `json:"demoman"`
	} `json:"classes"`
	Weapons struct {
	} `json:"weapons"`
	ScoreboardKills   int         `json:"scoreboard_kills"`
	ScoreboardAssists interface{} `json:"scoreboard_assists"`
	Suicides          int         `json:"suicides"`
	ScoreboardDeaths  int         `json:"scoreboard_deaths"`
	Captures          int         `json:"captures"`
	CapturesBlocked   int         `json:"captures_blocked"`
	ScoreboardDamage  int         `json:"scoreboard_damage"`
}

type Player struct {
	Classes map[PlayerClass]int `json:"classes"`
	Name    string              `json:"name"`
	UserID  int                 `json:"userId"`  //nolint:tagliatelle
	SteamID steamid.SteamID     `json:"steamId"` //nolint:tagliatelle
	Team    logparse.Team       `json:"team"`
}

type WeaponSummary struct {
	Kills     int `json:"kills"`
	Damage    int `json:"damage"`
	Shots     int `json:"shots"`
	Hits      int `json:"hits"`
	Backstabs int `json:"backstabs"`
	Headshots int `json:"headshots"`
	Airshots  int `json:"airshots"`
}

type ChatMessage struct {
	SteamID     string `json:"steam_id"`
	PersonaName string `json:"persona_name"`
	Body        string `json:"body"`
	Team        bool   `json:"team"`
}

type Results struct {
	ScoreBlu int `json:"score_blu"`
	BluTime  int `json:"blu_time"`
	ScoreRed int `json:"score_red"`
	RedTime  int `json:"red_time"`
}

type DemoRoundSummary struct{}
