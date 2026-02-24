package demoparse

import (
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type Demo struct {
	Filename string         `json:"filename"`
	DemoType string         `json:"demo_type"`
	Version  int            `json:"version"`
	Protocol int            `json:"protocol"`
	Server   string         `json:"server"`
	Nick     string         `json:"nick"`
	Map      string         `json:"map"`
	Game     string         `json:"game"`
	Duration float64        `json:"duration"`
	Ticks    int            `json:"ticks"`
	Frames   int            `json:"frames"`
	Signon   int            `json:"signon"`
	Rounds   []RoundSummary `json:"rounds"`
	Chat     []ChatMessage  `json:"chat"`
}

func (d Demo) SteamIDs() steamid.Collection {
	var col steamid.Collection

	for _, round := range d.Rounds {
		for _, player := range round.Players {
			sid := steamid.New(player.SteamID)
			if !col.Contains(sid) {
				col = append(col, sid)
			}
		}
	}

	return col
}

type GameState struct {
	Users   map[int]Player        `json:"users"`
	Players map[int]PlayerSummary `json:"players"` //nolint:tagliatelle
	Results Results               `json:"results"` //nolint:tagliatelle
	Rounds  []DemoRoundSummary    `json:"rounds"`
	Chat    []ChatMessage         `json:"chat"`
}

type HealingSummary struct{}

type Stats struct {
	Kills               int `json:"kills"`
	Assists             int `json:"assists"`
	Deaths              int `json:"deaths"`
	PostroundKills      int `json:"postround_kills"`
	PostroundAssists    int `json:"postround_assists"`
	PostroundDeaths     int `json:"postround_deaths"`
	Damage              int `json:"damage"`
	DamageTaken         int `json:"damage_taken"`
	Dominations         int `json:"dominations"`
	Dominated           int `json:"dominated"`
	Revenges            int `json:"revenges"`
	Revenged            int `json:"revenged"`
	Airshots            int `json:"airshots"`
	HeadshotKills       int `json:"headshot_kills"`
	BackstabKills       int `json:"backstab_kills"`
	Headshots           int `json:"headshots"`
	Backstabs           int `json:"backstabs"`
	WasHeadshot         int `json:"was_headshot"`
	PreroundHealing     int `json:"preround_healing"`
	Healing             int `json:"healing"`
	PostroundHealing    int `json:"postround_healing"`
	Drops               int `json:"drops"`
	NearFullChargeDeath int `json:"near_full_charge_death"`
	ChargesUber         int `json:"charges_uber"`
	ChargesKritz        int `json:"charges_kritz"`
	ChargesQuickfix     int `json:"charges_quickfix"`
	WasBackstabbed      int `json:"was_backstabbed"`
}

type Classes struct {
	Pyro     Stats `json:"pyro"`
	Heavy    Stats `json:"heavy"`
	Soldier  Stats `json:"soldier"`
	Sniper   Stats `json:"sniper"`
	Spy      Stats `json:"spy"`
	Scout    Stats `json:"scout"`
	Demoman  Stats `json:"demoman"`
	Engineer Stats `json:"engineer"`
	Medic    Stats `json:"medic"`
}

type PlayerSummary struct {
	Name string `json:"name"`
	// Not a steamid.SteamID, since this can be BOT
	SteamID         string `json:"steamid"` //nolint:tagliatelle
	Team            string `json:"team"`
	TickStart       int    `json:"tick_start"`
	TickEnd         int    `json:"tick_end"`
	Points          int    `json:"points"`
	ConnectionCount int    `json:"connection_count"`
	BonusPoints     int    `json:"bonus_points"`

	Kills            int `json:"kills"`
	Assists          int `json:"assists"`
	Deaths           int `json:"deaths"`
	PostroundKills   int `json:"postround_kills"`
	PostroundAssists int `json:"postround_assists"`

	PreroundHealing     int `json:"preround_healing"`
	Healing             int `json:"healing"`
	PostroundHealing    int `json:"postround_healing"`
	Drops               int `json:"drops"`
	NearFullChargeDeath int `json:"near_full_charge_death"`
	ChargesUber         int `json:"charges_uber"`
	ChargesKritz        int `json:"charges_kritz"`
	ChargesQuickfix     int `json:"charges_quickfix"`
	Damage              int `json:"damage"`
	DamageTaken         int `json:"damage_taken"`
	Dominations         int `json:"dominations"`
	Dominated           int `json:"dominated"`
	Revenges            int `json:"revenges"`
	Revenged            int `json:"revenged"`
	Airshots            int `json:"airshots"`
	HeadshotKills       int `json:"headshot_kills"`
	BackstabKills       int `json:"backstab_kills"`
	Headshots           int `json:"headshots"`
	Backstabs           int `json:"backstabs"`
	WasHeadshot         int `json:"was_headshot"`
	WasBackstabbed      int `json:"was_backstabbed"`
	Shots               int `json:"shots"`
	Hits                int `json:"hits"`
	ObjectBuilt         int `json:"object_built"`
	ObjectDestroyed     int `json:"object_destroyed"`

	Classes Classes          `json:"classes"`
	Weapons map[string]Stats `json:"weapons"`

	ScoreboardKills   int `json:"scoreboard_kills"`
	ScoreboardAssists int `json:"scoreboard_assists"`
	Suicides          int `json:"suicides"`
	ScoreboardDeaths  int `json:"scoreboard_deaths"`
	PostroundDeaths   int `json:"postround_deaths"`

	Captures        int `json:"captures"`
	CapturesBlocked int `json:"captures_blocked"`

	ScoreboardDamage int `json:"scoreboard_damage"`

	IsFakePlayer bool `json:"is_fake_player"`
	IsHlTv       bool `json:"is_hl_tv"`
	IsReplay     bool `json:"is_replay"`

	// TODO
	// HealingTaken     int `json:"healing_taken"`
	// HealthPacksCount int `json:"health_packs_count"`
	// HealingFromPacks int `json:"health_from_packs"`

	Extinguishes int `json:"extinguishes"`
	Ignites      int `json:"ignites"`

	BuildingBuilt     int `json:"building_built"`
	BuildingDestroyed int `json:"building_destroyed"`
}

type RoundSummary struct {
	Winner        string  `json:"winner"`
	IsStalemate   bool    `json:"is_stalemate"`
	IsSuddenDeath bool    `json:"is_sudden_death"`
	Time          float64 `json:"time"` // seconds

	Duration float64         `json:"duration"`
	Mvps     []string        `json:"mvps"`
	Players  []PlayerSummary `json:"players"`

	Winners []string `json:"winners"`
	Losers  []string `json:"losers"`
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

	PreroundHealing     int `json:"preround_healing"`
	Healing             int `json:"healing"`
	PostroundHealing    int `json:"postround_healing"`
	Drops               int `json:"drops"`
	NearFullChargeDeath int `json:"near_full_charge_death"`
	ChargesUber         int `json:"charges_uber"`
	ChargesKritz        int `json:"charges_kritz"`
	ChargesQuickfix     int `json:"charges_quickfix"`
}

type ChatMessage struct {
	User    string `json:"user"`
	Tick    int64  `json:"tick"`
	Message string `json:"message"`
}

type Results struct {
	ScoreBlu int `json:"score_blu"`
	BluTime  int `json:"blu_time"`
	ScoreRed int `json:"score_red"`
	RedTime  int `json:"red_time"`
}

type DemoRoundSummary struct{}
