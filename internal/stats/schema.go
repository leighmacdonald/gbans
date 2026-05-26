package stats

import (
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type MapUseDetail struct {
	Map      string
	Playtime int64
	Percent  float64
}

type Weapon struct {
	WeaponID int
	Key      logparse.Weapon
	Name     string
}

type RankedResult struct {
	Rank int
}

type WeaponsOverallResult struct {
	Weapon
	RankedResult

	Kills        int64
	KillsPct     float64
	Damage       int64
	DamagePct    float64
	Headshots    int64
	HeadshotsPct float64
	Airshots     int64
	AirshotsPct  float64
	Backstabs    int64
	BackstabsPct float64
	Shots        int64
	ShotsPct     float64
	Hits         int64
	HitsPct      float64
}

type PlayerWeaponResult struct {
	Rank               int
	SteamID            steamid.SteamID
	Personaname        string
	AvatarHash         string
	KA                 int64
	Kills              int64
	Assists            int64
	Deaths             int64
	KD                 float64
	KAD                float64
	DPM                float64
	Shots              int64
	Hits               int64
	Accuracy           float64
	Airshots           int64
	Backstabs          int64
	Headshots          int64
	Playtime           int64
	Dominations        int64
	Dominated          int64
	Revenges           int64
	Damage             int64
	DamageTaken        int64
	Captures           int64
	CapturesBlocked    int64
	BuildingsDestroyed int64
}

type HealingOverallResult struct {
	RankedResult

	SteamID             steamid.SteamID
	Personaname         string
	AvatarHash          string
	Healing             int
	Drops               int
	NearFullChargeDeath int
	ChargesUber         int
	ChargesKritz        int
	ChargesVacc         int
	ChargesQuickfix     int
	AvgUberLength       float32
	MajorAdvLost        int
	BiggestAdvLost      int
	Extinguishes        int64
	HealthPacks         int64
	Assists             int64
	Deaths              int64
	HPM                 float64
	KA                  float64
	KAD                 float64
	Playtime            int64
	Dominations         int64
	Dominated           int64
	Revenges            int64
	DamageTaken         int64
	DTM                 float64
	Wins                int64
	Matches             int64
	WinRate             float64
}

type PlayerClass struct {
	PlayerClassID int
	ClassName     string
	ClassKey      string
}

type PlayerClassOverallResult struct {
	PlayerClass

	Kills              int64
	KA                 int64
	Assists            int64
	Deaths             int64
	KD                 float64
	KAD                float64
	DPM                float64
	Playtime           int64
	Dominations        int64
	Dominated          int64
	Revenges           int64
	Damage             int64
	DamageTaken        int64
	HealingTaken       int64
	Captures           int64
	CapturesBlocked    int64
	BuildingsDestroyed int64
}

type PlayerOverallResult struct {
	Healing             int64
	Drops               int64
	NearFullChargeDeath int64
	AvgUberLen          float64
	BiggestAdvLost      float64
	MajorAdvLost        float64
	ChargesUber         int64
	ChargesKritz        int64
	ChargesVacc         int64
	ChargesQuickfix     int64
	Buildings           int64
	Extinguishes        int64
	HealthPacks         int64
	KA                  int64
	Kills               int64
	Assists             int64
	Deaths              int64
	KD                  float64
	KAD                 float64
	DPM                 float64
	Shots               int64
	Hits                int64
	Accuracy            float64
	Airshots            int64
	Backstabs           int64
	Headshots           int64
	Playtime            int64
	Dominations         int64
	Dominated           int64
	Revenges            int64
	Damage              int64
	DamageTaken         int64
	Captures            int64
	CapturesBlocked     int64
	BuildingsDestroyed  int64
	HealingTaken        int64
	Wins                int64
	Matches             int64
	WinRate             float64
}
