package stats

import (
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type Bucket struct {
	BucketID   int32
	BucketName string
	IsEnabled  bool
}

type PlayerMatchHistory struct {
	MatchID         uuid.UUID
	ServerID        int32
	ServerName      string
	ServerNameShort string
	MapID           int32
	MapName         string
	DemoID          int32
	BucketID        int32
	BucketName      string
	Hostname        string
	ScoreBlu        uint32
	ScoreRed        uint32
	DurationMs      uint64
	IsWinnder       bool
	CreatedOn       time.Time
}

// Match is the top level container for all of the matches data.
type Match struct {
	MatchID         uuid.UUID
	AssetID         uuid.UUID
	ServerID        int32
	ServerName      string
	ServerNameShort string
	MapID           int32
	MapName         string
	DemoID          int32
	StatsBucketID   int32
	StatsBucketName string
	Hostname        string
	ScoreBlu        uint32
	ScoreRed        uint32
	StartTime       time.Time
	DurationMs      uint64
	CreatedOn       time.Time

	// Data sums are split into rounds for a bit more fine grained info.
	// They are stored in a enturely flat structure here just for ease of processing
	Rounds   []MatchRound
	Players  []MatchOverallStatsRound
	Variants []MatchVariantStatsRound
	ChatLogs []MatchChatLog
}

type MatchChatLog struct {
	PersonMessageID int64
	SteamID         steamid.SteamID
	Name            string
	Body            string
	DemoTick        int32
}

type MatchRound struct {
	RoundID       uint32
	Winner        string
	IsStalemate   bool
	IsSuddenDeath bool
	DurationMs    uint64
}

// Used in the match queries.
type MatchOverallStatsRound struct {
	OverallStats

	Team    string
	RoundID uint32
}

type MatchVariantStatsRound struct {
	VariantStats

	RoundID uint32
}
