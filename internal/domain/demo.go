package domain

import (
	"context"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
)

type DemoUsecase interface {
	ExpiredDemos(ctx context.Context, limit uint64) ([]DemoInfo, error)
	GetDemoByID(ctx context.Context, demoID int64, demoFile *DemoFile) error
	MarkArchived(ctx context.Context, demo *DemoFile) error
	GetDemoByName(ctx context.Context, demoName string, demoFile *DemoFile) error
	GetDemos(ctx context.Context) ([]DemoFile, error)
	CreateFromAsset(ctx context.Context, asset Asset, serverID int) (*DemoFile, error)
	Cleanup(ctx context.Context)
	SendAndParseDemo(ctx context.Context, path string) (*DemoDetails, error)
}

type DemoRepository interface {
	ExpiredDemos(ctx context.Context, limit uint64) ([]DemoInfo, error)
	GetDemoByID(ctx context.Context, demoID int64, demoFile *DemoFile) error
	GetDemoByName(ctx context.Context, demoName string, demoFile *DemoFile) error
	GetDemos(ctx context.Context) ([]DemoFile, error)
	SaveDemo(ctx context.Context, demoFile *DemoFile) error
	Delete(ctx context.Context, demoID int64) error
}

type DemoPlayerStats struct {
	Score      int `json:"score"`
	ScoreTotal int `json:"score_total"`
	Deaths     int `json:"deaths"`
}

type DemoMetaData struct {
	MapName string                     `json:"map_name"`
	Scores  map[string]DemoPlayerStats `json:"scores"`
}

type DemoFile struct {
	DemoID          int64            `json:"demo_id"`
	ServerID        int              `json:"server_id"`
	ServerNameShort string           `json:"server_name_short"`
	ServerNameLong  string           `json:"server_name_long"`
	Title           string           `json:"title"`
	CreatedOn       time.Time        `json:"created_on"`
	Downloads       int64            `json:"downloads"`
	Size            int64            `json:"size"`
	MapName         string           `json:"map_name"`
	Archive         bool             `json:"archive"` // When true, will not get auto deleted when flushing old demos
	Stats           map[string]gin.H `json:"stats"`
	AssetID         uuid.UUID        `json:"asset_id"`
}

const DemoType = "HL2DEMO"

type DemoInfo struct {
	DemoID  int64
	Title   string
	AssetID uuid.UUID
}

type DemoPlayer struct {
	Classes map[logparse.PlayerClass]int `json:"classes"`
	Name    string                       `json:"name"`
	UserID  int                          `json:"userId"`  //nolint:tagliatelle
	SteamID steamid.SteamID              `json:"steamId"` //nolint:tagliatelle
	Team    logparse.Team                `json:"team"`
}

type DemoHeader struct {
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

type DemoWeaponDetail struct {
	Kills     int `json:"kills"`
	Damage    int `json:"damage"`
	Shots     int `json:"shots"`
	Hits      int `json:"hits"`
	Backstabs int `json:"backstabs,`
	Headshots int `json:"headshots"`
	Airshots  int `json:"airshots"`
}

type DemoPlayerSummaries struct {
	Points             int                                  `json:"points"`
	Kills              int                                  `json:"kills"`
	Assists            int                                  `json:"assists"`
	Deaths             int                                  `json:"deaths"`
	BuildingsDestroyed int                                  `json:"buildings_destroyed"`
	Captures           int                                  `json:"captures"`
	Defenses           int                                  `json:"defenses"`
	Dominations        int                                  `json:"dominations"`
	Revenges           int                                  `json:"revenges"`
	Ubercharges        int                                  `json:"ubercharges"`
	Headshots          int                                  `json:"headshots"`
	Teleports          int                                  `json:"teleports"`
	Healing            int                                  `json:"healing"`
	Backstabs          int                                  `json:"backstabs"`
	BonusPoints        int                                  `json:"bonus_points"`
	Support            int                                  `json:"support"`
	DamgageDealt       int                                  `json:"damgage_dealt"`
	WeaponMap          map[logparse.Weapon]DemoWeaponDetail `json:"weapon_map"`
}

type DemoChatMessage struct {
}

type DemoMatchSummary struct {
	ScoreBlu int               `json:"score_blu"`
	ScoreRed int               `json:"score_red"`
	Chat     []DemoChatMessage `json:"chat"`
}

type DemoRoundSummary struct {
}

type DemoState struct {
	DemoPlayerSummaries map[int]DemoPlayerSummaries `json:"player_summaries"` //nolint:tagliatelle
	Users               map[int]DemoPlayer          `json:"users"`
	DemoMatchSummary    DemoMatchSummary            `json:"match_summary"` //nolint:tagliatelle
	DemoRoundSummary    DemoRoundSummary            `json:"round_summary"`
}

type DemoDetails struct {
	State  DemoState  `json:"state"`
	Header DemoHeader `json:"header"`
}
