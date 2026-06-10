package stats

import (
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type Bucket struct {
	BucketID   int32
	BucketName string
}

type Match struct {
	MatchID       uuid.UUID
	ServerID      int32
	MapID         int32
	DemoID        int32
	StatsBucketID int32
	Hostname      string
	ScoreBlu      uint32
	ScoreRed      uint32
	StartTime     time.Time
	DurationMs    uint64
	CreatedOn     time.Time

	Rounds  []MatchRound
	Players []MatchRoundPlayer
	Weapons []MatchRoundWeaponStats
	Classes []MatchRoundClassStats
}

type MatchRound struct {
	RoundID       uint32
	Winner        string
	IsStalemate   bool
	IsSuddenDeath bool
	DurationMs    uint64
}

type MatchRoundPlayer struct {
	RoundID             uint32
	SteamID             steamid.SteamID
	Team                string
	MVP                 bool
	TickStart           uint64
	TickEnd             uint64
	Points              uint64
	ConnectionCount     uint64
	BonusPoints         uint64
	Kills               uint64
	Assists             uint64
	Deaths              uint64
	PostroundKills      uint64
	PostroundAssists    uint64
	PostroundDeaths     uint64
	PostroundHealing    uint64
	Healing             uint64
	PreroundHealing     uint64
	Drops               uint64
	NearFullChargeDeath uint64
	ChargesUber         uint64
	ChargesKritz        uint64
	ChargesVacc         uint64
	ChargesQuickfix     uint64
	Damage              uint64
	DamageTaken         uint64
	Dominations         uint64
	Dominated           uint64
	Revenges            uint64
	Revenged            uint64
	Airshots            uint64
	Headshots           uint64
	HeadshotKills       uint64
	Backstabs           uint64
	BackstabKills       uint64
	WasHeadshot         uint64
	WasBackstabbed      uint64
	Shots               uint64
	Hits                uint64
	ScoreboardKills     uint64
	ScoreboardAssists   uint64
	ScoreboardDeaths    uint64
	Suicides            uint64
	Captures            uint64
	CapturesBlocked     uint64
	ScoreboardDamage    uint64
	Extinguishes        uint64
	Ignites             uint64
	ObjectsBuilt        uint64
	ObjectsDestroyed    uint64
	BuildingsBuilt      uint64
	BUildingsDestroyed  uint64
}

type MatchBaseStats struct {
	SteamID             steamid.SteamID
	RoundID             uint32
	Kills               uint64
	Assists             uint64
	Deaths              uint64
	PostroundKills      uint64
	PostroundAssists    uint64
	PostroundDeaths     uint64
	Damage              uint64
	DamageTaken         uint64
	Dominations         uint64
	Dominated           uint64
	Revenges            uint64
	Revenged            uint64
	Airshots            uint64
	HeadshotKills       uint64
	BackstabKills       uint64
	Headshots           uint64
	Backstabs           uint64
	WasHeadshot         uint64
	WasBackstabbed      uint64
	PreroundHealing     uint64
	Healing             uint64
	PostroundHealing    uint64
	Drops               uint64
	NearFullChargeDeath uint64
	ChargesUber         uint64
	ChargesKritz        uint64
	ChargesVacc         uint64
	ChargesQuickfix     uint64
}

type MatchRoundClassStats struct {
	MatchBaseStats

	Class string
}

type MatchRoundWeaponStats struct {
	MatchBaseStats

	Weapon string
}
