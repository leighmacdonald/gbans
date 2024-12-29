package demoparse

import (
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type Demo struct {
	State  GameState `json:"state"`
	Header Header    `json:"header"`
}

type GameState struct {
	Users   map[int]Player        `json:"users"`
	Players map[int]PlayerSummary `json:"players"` //nolint:tagliatelle
	Results Results               `json:"results"` //nolint:tagliatelle
	Rounds  []DemoRoundSummary    `json:"rounds"`
	Chat    []ChatMessage         `json:"chat"`
}

type Header struct {
	DemoType string  `json:"demo_type"`
	Version  int     `json:"version"`
	Protocol int     `json:"protocol"`
	Server   string  `json:"server"`
	Nick     string  `json:"nick"`
	Map      string  `json:"map"`
	Game     string  `json:"game"`
	Duration float64 `json:"duration"`
	Ticks    int     `json:"ticks"`
	Frames   int     `json:"frames"`
	Signon   int     `json:"signon"`
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

type PlayerSummary struct {
	Points             int                        `json:"points"`
	Kills              int                        `json:"kills"`
	Assists            int                        `json:"assists"`
	Deaths             int                        `json:"deaths"`
	BuildingsDestroyed int                        `json:"buildings_destroyed"`
	Captures           int                        `json:"captures"`
	Defenses           int                        `json:"defenses"`
	Dominations        int                        `json:"dominations"`
	Revenges           int                        `json:"revenges"`
	Ubercharges        int                        `json:"ubercharges"`
	Headshots          int                        `json:"headshots"`
	Teleports          int                        `json:"teleports"`
	Healing            int                        `json:"healing"`
	Backstabs          int                        `json:"backstabs"`
	BonusPoints        int                        `json:"bonus_points"`
	Support            int                        `json:"support"`
	DamageDealt        int                        `json:"damage_dealt"`
	DamageTaken        int                        `json:"damage_taken"`
	HealingTaken       int                        `json:"healing_taken"`
	HealthPacks        int                        `json:"health_packs"`
	HealingPacks       int                        `json:"healing_packs"`
	Extinguishes       int                        `json:"extinguishes"`
	BuildingBuilt      int                        `json:"building_built"`
	BuildingDestroyed  int                        `json:"building_destroyed"`
	Airshots           int                        `json:"airshots"`
	Shots              int                        `json:"shots"`
	Hits               int                        `json:"hits"`
	WeaponMap          map[WeaponID]WeaponSummary `json:"weapon_map"`
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
