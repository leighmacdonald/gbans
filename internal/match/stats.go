package match

import (
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type MapUseDetail struct {
	Map      string  `json:"map"`
	Playtime int64   `json:"playtime"`
	Percent  float64 `json:"percent"`
}

type Weapon struct {
	WeaponID int             `json:"weapon_id"`
	Key      logparse.Weapon `json:"key"`
	Name     string          `json:"name"`
}

type RankedResult struct {
	Rank int `json:"rank"`
}

type WeaponsOverallResult struct {
	Weapon
	RankedResult

	Kills        int64   `json:"kills"`
	KillsPct     float64 `json:"kills_pct"`
	Damage       int64   `json:"damage"`
	DamagePct    float64 `json:"damage_pct"`
	Headshots    int64   `json:"headshots"`
	HeadshotsPct float64 `json:"headshots_pct"`
	Airshots     int64   `json:"airshots"`
	AirshotsPct  float64 `json:"airshots_pct"`
	Backstabs    int64   `json:"backstabs"`
	BackstabsPct float64 `json:"backstabs_pct"`
	Shots        int64   `json:"shots"`
	ShotsPct     float64 `json:"shots_pct"`
	Hits         int64   `json:"hits"`
	HitsPct      float64 `json:"hits_pct"`
}

type PlayerWeaponResult struct {
	Rank               int             `json:"rank"`
	SteamID            steamid.SteamID `json:"steam_id"`
	Personaname        string          `json:"personaname"`
	AvatarHash         string          `json:"avatar_hash"`
	KA                 int64           `json:"ka"`
	Kills              int64           `json:"kills"`
	Assists            int64           `json:"assists"`
	Deaths             int64           `json:"deaths"`
	KD                 float64         `json:"kd"`
	KAD                float64         `json:"kad"`
	DPM                float64         `json:"dpm"`
	Shots              int64           `json:"shots"`
	Hits               int64           `json:"hits"`
	Accuracy           float64         `json:"accuracy"`
	Airshots           int64           `json:"airshots"`
	Backstabs          int64           `json:"backstabs"`
	Headshots          int64           `json:"headshots"`
	Playtime           int64           `json:"playtime"`
	Dominations        int64           `json:"dominations"`
	Dominated          int64           `json:"dominated"`
	Revenges           int64           `json:"revenges"`
	Damage             int64           `json:"damage"`
	DamageTaken        int64           `json:"damage_taken"`
	Captures           int64           `json:"captures"`
	CapturesBlocked    int64           `json:"captures_blocked"`
	BuildingsDestroyed int64           `json:"buildings_destroyed"`
}

type HealingOverallResult struct {
	RankedResult

	SteamID             steamid.SteamID `json:"steam_id"`
	Personaname         string          `json:"personaname"`
	AvatarHash          string          `json:"avatar_hash"`
	Healing             int             `json:"healing"`
	Drops               int             `json:"drops"`
	NearFullChargeDeath int             `json:"near_full_charge_death"`
	ChargesUber         int             `json:"charges_uber"`
	ChargesKritz        int             `json:"charges_kritz"`
	ChargesVacc         int             `json:"charges_vacc"`
	ChargesQuickfix     int             `json:"charges_quickfix"`
	AvgUberLength       float32         `json:"avg_uber_length"`
	MajorAdvLost        int             `json:"major_adv_lost"`
	BiggestAdvLost      int             `json:"biggest_adv_lost"`
	Extinguishes        int64           `json:"extinguishes"`
	HealthPacks         int64           `json:"health_packs"`
	Assists             int64           `json:"assists"`
	Deaths              int64           `json:"deaths"`
	HPM                 float64         `json:"hpm"`
	KA                  float64         `json:"ka"`
	KAD                 float64         `json:"kad"`
	Playtime            int64           `json:"playtime"`
	Dominations         int64           `json:"dominations"`
	Dominated           int64           `json:"dominated"`
	Revenges            int64           `json:"revenges"`
	DamageTaken         int64           `json:"damage_taken"`
	DTM                 float64         `json:"dtm"`
	Wins                int64           `json:"wins"`
	Matches             int64           `json:"matches"`
	WinRate             float64         `json:"win_rate"`
}

type PlayerClass struct {
	PlayerClassID int    `json:"player_class_id"`
	ClassName     string `json:"class_name"`
	ClassKey      string `json:"class_key"`
}

type PlayerClassOverallResult struct {
	PlayerClass

	Kills              int64   `json:"kills"`
	KA                 int64   `json:"ka"`
	Assists            int64   `json:"assists"`
	Deaths             int64   `json:"deaths"`
	KD                 float64 `json:"kd"`
	KAD                float64 `json:"kad"`
	DPM                float64 `json:"dpm"`
	Playtime           int64   `json:"playtime"`
	Dominations        int64   `json:"dominations"`
	Dominated          int64   `json:"dominated"`
	Revenges           int64   `json:"revenges"`
	Damage             int64   `json:"damage"`
	DamageTaken        int64   `json:"damage_taken"`
	HealingTaken       int64   `json:"healing_taken"`
	Captures           int64   `json:"captures"`
	CapturesBlocked    int64   `json:"captures_blocked"`
	BuildingsDestroyed int64   `json:"buildings_destroyed"`
}

type PlayerOverallResult struct {
	Healing             int64   `json:"healing"`
	Drops               int64   `json:"drops"`
	NearFullChargeDeath int64   `json:"near_full_charge_death"`
	AvgUberLen          float64 `json:"avg_uber_len"`
	BiggestAdvLost      float64 `json:"biggest_adv_lost"`
	MajorAdvLost        float64 `json:"major_adv_lost"`
	ChargesUber         int64   `json:"charges_uber"`
	ChargesKritz        int64   `json:"charges_kritz"`
	ChargesVacc         int64   `json:"charges_vacc"`
	ChargesQuickfix     int64   `json:"charges_quickfix"`
	Buildings           int64   `json:"buildings"`
	Extinguishes        int64   `json:"extinguishes"`
	HealthPacks         int64   `json:"health_packs"`
	KA                  int64   `json:"ka"`
	Kills               int64   `json:"kills"`
	Assists             int64   `json:"assists"`
	Deaths              int64   `json:"deaths"`
	KD                  float64 `json:"kd"`
	KAD                 float64 `json:"kad"`
	DPM                 float64 `json:"dpm"`
	Shots               int64   `json:"shots"`
	Hits                int64   `json:"hits"`
	Accuracy            float64 `json:"accuracy"`
	Airshots            int64   `json:"airshots"`
	Backstabs           int64   `json:"backstabs"`
	Headshots           int64   `json:"headshots"`
	Playtime            int64   `json:"playtime"`
	Dominations         int64   `json:"dominations"`
	Dominated           int64   `json:"dominated"`
	Revenges            int64   `json:"revenges"`
	Damage              int64   `json:"damage"`
	DamageTaken         int64   `json:"damage_taken"`
	Captures            int64   `json:"captures"`
	CapturesBlocked     int64   `json:"captures_blocked"`
	BuildingsDestroyed  int64   `json:"buildings_destroyed"`
	HealingTaken        int64   `json:"healing_taken"`
	Wins                int64   `json:"wins"`
	Matches             int64   `json:"matches"`
	WinRate             float64 `json:"win_rate"`
}
