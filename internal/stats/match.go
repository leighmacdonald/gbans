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
	startTime     time.Time
	durationMs    int64
	createdOn     time.Time

	Rounds  []MatchRound
	Players []MatchPlayer
}

type MatchRound struct {
	RoundID       uint
	Winner        string
	IsStalemate   bool
	IsSuddenDeath bool
	DurationMs    int64
}

type MatchPlayer struct {
	SteamID             steamid.SteamID
	Team                string
	MVP                 bool
	TickStart           uint
	TickEnd             uint
	Points              uint
	ConnectionCount     uint
	BonusPoints         uint
	Kills               uint
	Deaths              uint
	PostroundKills      uint
	PostroundAssists    uint
	PostroundDeaths     uint
	Healing             uint
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
	ObjectsBUilt        uint
	ObjectsDestroyed    uint
	BuildingsBuilt      uint
	BUildingsDestroyed  uint
}

type MatchBaseStats struct {
	SteamID             steamid.SteamID
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

type MatchClassStats struct {
	MatchBaseStats

	Class string
}

type MatchWeaponStats struct {
	MatchBaseStats

	Weapon string
}
