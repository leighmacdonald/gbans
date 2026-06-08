package stats

import (
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type Match struct {
	MatchID       uuid.UUID
	ServerID      int32
	MapID         int32
	DemoID        int32
	StatsBucketID int32
	Hostname      string
	ScoreBlu      uint
	ScoreRed      uint
	StartTime     time.Time
	DurationMs    int64
	CreatedOn     time.Time

	Rounds  []MatchRound
	Players []MatchRoundPlayer
	Weapons []MatchRoundWeaponStats
	Classes []MatchRoundClassStats
}

type MatchRound struct {
	RoundID       uint
	Winner        string
	IsStalemate   bool
	IsSuddenDeath bool
	DurationMs    int64
}

type MatchRoundPlayer struct {
	RoundID             uint
	SteamID             steamid.SteamID
	Team                string
	MVP                 bool
	TickStart           uint
	TickEnd             uint
	Points              uint
	ConnectionCount     uint
	BonusPoints         uint
	Kills               uint
	Assists             uint
	Deaths              uint
	PostroundKills      uint
	PostroundAssists    uint
	PostroundDeaths     uint
	PostroundHealing    uint
	Healing             uint
	PreroundHealing     uint
	Drops               uint
	NearFullChargeDeath uint
	ChargesUber         uint
	ChargesKritz        uint
	ChargesVacc         uint
	ChargesQuickfix     uint
	Damage              uint
	DamageTaken         uint
	Dominations         uint
	Dominated           uint
	Revenges            uint
	Revenged            uint
	Airshots            uint
	Headshots           uint
	HeadshotKills       uint
	Backstabs           uint
	BackstabKills       uint
	WasHeadshot         uint
	WasBackstabbed      uint
	Shots               uint
	Hits                uint
	ScoreboardKills     uint
	ScoreboardAssists   uint
	ScoreboardDeaths    uint
	Suicides            uint
	Captures            uint
	CapturesBlocked     uint
	ScoreboardDamage    uint
	Extinguishes        uint
	Ignites             uint
	ObjectsBuilt        uint
	ObjectsDestroyed    uint
	BuildingsBuilt      uint
	BUildingsDestroyed  uint
}

type MatchBaseStats struct {
	SteamID             steamid.SteamID
	RoundID             uint
	Kills               uint
	Assists             uint
	Deaths              uint
	PostroundKills      uint
	PostroundAssists    uint
	PostroundDeaths     uint
	Damage              uint
	DamageTaken         uint
	Dominations         uint
	Dominated           uint
	Revenges            uint
	Revenged            uint
	Airshots            uint
	HeadshotKills       uint
	BackstabKills       uint
	Headshots           uint
	Backstabs           uint
	WasHeadshot         uint
	WasBackstabbed      uint
	PreroundHeadling    uint
	Healing             uint
	PostroundHealing    uint
	Drops               uint
	NearFullChargeDeath uint
	ChargesUber         uint
	ChargesKritz        uint
	ChargesVacc         uint
	ChargesQuickfix     uint
}

type MatchRoundClassStats struct {
	MatchBaseStats

	Class string
}

type MatchRoundWeaponStats struct {
	MatchBaseStats

	Weapon string
}
