package domain

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"golang.org/x/exp/slices"
)

type MatchRepository interface {
	Matches(ctx context.Context, opts MatchesQueryOpts) ([]MatchSummary, int64, error)
	MatchGetByID(ctx context.Context, matchID uuid.UUID, match *MatchResult) error
	MatchSave(ctx context.Context, match *logparse.Match, weaponMap fp.MutexMap[logparse.Weapon, int]) error
	StatsPlayerClass(ctx context.Context, sid64 steamid.SteamID) (PlayerClassStatsCollection, error)
	StatsPlayerWeapons(ctx context.Context, sid64 steamid.SteamID) ([]PlayerWeaponStats, error)
	StatsPlayerKillstreaks(ctx context.Context, sid64 steamid.SteamID) ([]PlayerKillstreakStats, error)
	StatsPlayerMedic(ctx context.Context, sid64 steamid.SteamID) ([]PlayerMedicStats, error)
	PlayerStats(ctx context.Context, steamID steamid.SteamID, stats *PlayerStats) error
	WeaponsOverall(ctx context.Context) ([]WeaponsOverallResult, error)
	GetMapUsageStats(ctx context.Context) ([]MapUseDetail, error)
	Weapons(ctx context.Context) ([]Weapon, error)
	SaveWeapon(ctx context.Context, weapon *Weapon) error
	GetWeaponByKey(ctx context.Context, key logparse.Weapon, weapon *Weapon) error
	GetWeaponByID(ctx context.Context, weaponID int, weapon *Weapon) error
	LoadWeapons(ctx context.Context, weaponMap fp.MutexMap[logparse.Weapon, int]) error
	WeaponsOverallTopPlayers(ctx context.Context, weaponID int) ([]PlayerWeaponResult, error)
	WeaponsOverallByPlayer(ctx context.Context, steamID steamid.SteamID) ([]WeaponsOverallResult, error)
	PlayersOverallByKills(ctx context.Context, count int) ([]PlayerWeaponResult, error)
	HealersOverallByHealing(ctx context.Context, count int) ([]HealingOverallResult, error)
	PlayerOverallClassStats(ctx context.Context, steamID steamid.SteamID) ([]PlayerClassOverallResult, error)
	PlayerOverallStats(ctx context.Context, steamID steamid.SteamID, por *PlayerOverallResult) error
	GetMatchIDFromServerID(serverID int) (uuid.UUID, bool)
}
type MatchUsecase interface {
	CreateFromDemo(ctx context.Context, serverID int, details DemoDetails) (MatchSummary, error)
	GetMatchIDFromServerID(serverID int) (uuid.UUID, bool)
	Matches(ctx context.Context, opts MatchesQueryOpts) ([]MatchSummary, int64, error)
	MatchGetByID(ctx context.Context, matchID uuid.UUID, match *MatchResult) error
	MatchSave(ctx context.Context, match *logparse.Match, weaponMap fp.MutexMap[logparse.Weapon, int]) error
	StatsPlayerClass(ctx context.Context, sid64 steamid.SteamID) (PlayerClassStatsCollection, error)
	StatsPlayerWeapons(ctx context.Context, sid64 steamid.SteamID) ([]PlayerWeaponStats, error)
	StatsPlayerKillstreaks(ctx context.Context, sid64 steamid.SteamID) ([]PlayerKillstreakStats, error)
	StatsPlayerMedic(ctx context.Context, sid64 steamid.SteamID) ([]PlayerMedicStats, error)
	PlayerStats(ctx context.Context, steamID steamid.SteamID, stats *PlayerStats) error
	WeaponsOverall(ctx context.Context) ([]WeaponsOverallResult, error)
	GetMapUsageStats(ctx context.Context) ([]MapUseDetail, error)
	Weapons(ctx context.Context) ([]Weapon, error)
	SaveWeapon(ctx context.Context, weapon *Weapon) error
	GetWeaponByKey(ctx context.Context, key logparse.Weapon, weapon *Weapon) error
	GetWeaponByID(ctx context.Context, weaponID int, weapon *Weapon) error
	LoadWeapons(ctx context.Context, weaponMap fp.MutexMap[logparse.Weapon, int]) error
	WeaponsOverallTopPlayers(ctx context.Context, weaponID int) ([]PlayerWeaponResult, error)
	WeaponsOverallByPlayer(ctx context.Context, steamID steamid.SteamID) ([]WeaponsOverallResult, error)
	PlayersOverallByKills(ctx context.Context, count int) ([]PlayerWeaponResult, error)
	HealersOverallByHealing(ctx context.Context, count int) ([]HealingOverallResult, error)
	PlayerOverallClassStats(ctx context.Context, steamID steamid.SteamID) ([]PlayerClassOverallResult, error)
	PlayerOverallStats(ctx context.Context, steamID steamid.SteamID, por *PlayerOverallResult) error
}

const MinMedicHealing = 500

type MatchPlayerKillstreak struct {
	MatchKillstreakID int64                `json:"match_killstreak_id"`
	MatchPlayerID     int64                `json:"match_player_id"`
	PlayerClass       logparse.PlayerClass `json:"player_class"`
	Killstreak        int                  `json:"killstreak"`
	// Seconds
	Duration int `json:"duration"`
}

type MatchPlayerClass struct {
	MatchPlayerClassID int                  `json:"match_player_class_id"`
	MatchPlayerID      int64                `json:"match_player_id"`
	PlayerClass        logparse.PlayerClass `json:"player_class"`
	Kills              int                  `json:"kills"`
	Assists            int                  `json:"assists"`
	Deaths             int                  `json:"deaths"`
	Playtime           int                  `json:"playtime"`
	Dominations        int                  `json:"dominations"`
	Dominated          int                  `json:"dominated"`
	Revenges           int                  `json:"revenges"`
	Damage             int                  `json:"damage"`
	DamageTaken        int                  `json:"damage_taken"`
	HealingTaken       int                  `json:"healing_taken"`
	Captures           int                  `json:"captures"`
	CapturesBlocked    int                  `json:"captures_blocked"`
	BuildingDestroyed  int                  `json:"building_destroyed"`
}

type MatchPlayer struct {
	MatchPlayerID int64 `json:"match_player_id"`
	CommonPlayerStats
	Team      logparse.Team `json:"team"`
	TimeStart time.Time     `json:"time_start"`
	TimeEnd   time.Time     `json:"time_end"`

	MedicStats  *MatchHealer            `json:"medic_stats"`
	Classes     []MatchPlayerClass      `json:"classes"`
	Killstreaks []MatchPlayerKillstreak `json:"killstreaks"`
	Weapons     []MatchPlayerWeapon     `json:"weapons"`
}

func (player MatchPlayer) BiggestKillstreak() *MatchPlayerKillstreak {
	var biggest *MatchPlayerKillstreak

	for _, killstreakVal := range player.Killstreaks {
		killstreak := killstreakVal
		if biggest == nil || killstreak.Killstreak > biggest.Killstreak {
			biggest = &killstreak
		}
	}

	return biggest
}

func (player MatchPlayer) KDRatio() float64 {
	if player.Deaths <= 0 {
		return -1
	}

	return math.Ceil((float64(player.Kills)/float64(player.Deaths))*100) / 100
}

func (player MatchPlayer) KDARatio() float64 {
	if player.Deaths <= 0 {
		return -1
	}

	return math.Ceil((float64(player.Kills+player.Assists)/float64(player.Deaths))*100) / 100
}

func (player MatchPlayer) DamagePerMin() int {
	return slices.Max([]int{int(float64(player.Damage) / player.TimeEnd.Sub(player.TimeStart).Minutes()), 0})
}

type MatchHealer struct {
	MatchMedicID        int64   `json:"match_medic_id"`
	MatchPlayerID       int64   `json:"match_player_id"`
	Healing             int     `json:"healing"`
	ChargesUber         int     `json:"charges_uber"`
	ChargesKritz        int     `json:"charges_kritz"`
	ChargesVacc         int     `json:"charges_vacc"`
	ChargesQuickfix     int     `json:"charges_quickfix"`
	Drops               int     `json:"drops"`
	NearFullChargeDeath int     `json:"near_full_charge_death"`
	AvgUberLength       float32 `json:"avg_uber_length"`
	MajorAdvLost        int     `json:"major_adv_lost"`
	BiggestAdvLost      int     `json:"biggest_adv_lost"`
}

func (h MatchHealer) HealingPerMin(matchDuration time.Duration) int {
	if h.Healing <= 0 {
		return 0
	}

	return int(float64(h.Healing) / matchDuration.Minutes())
}

type MatchWeapon struct {
	PlayerWeaponID int64 `json:"player_weapon_id"`
	MatchPlayerID  int64 `json:"match_player_id"`
}

type MatchResult struct {
	MatchID    uuid.UUID           `json:"match_id"`
	ServerID   int                 `json:"server_id"`
	Title      string              `json:"title"`
	MapName    string              `json:"map_name"`
	TeamScores logparse.TeamScores `json:"team_scores"`
	TimeStart  time.Time           `json:"time_start"`
	TimeEnd    time.Time           `json:"time_end"`
	Winner     logparse.Team       `json:"winner"`
	Players    []*MatchPlayer      `json:"players"`
	Chat       PersonMessages      `json:"chat"`
}

func (match *MatchResult) TopPlayers() []*MatchPlayer {
	players := match.Players

	sort.SliceStable(players, func(i, j int) bool {
		return players[i].Kills > players[j].Kills
	})

	return players
}

func (match *MatchResult) TopKillstreaks(count int) []*MatchPlayer {
	var killStreakPlayers []*MatchPlayer

	for _, player := range match.Players {
		if killStreak := player.BiggestKillstreak(); killStreak != nil {
			killStreakPlayers = append(killStreakPlayers, player)
		}
	}

	sort.SliceStable(killStreakPlayers, func(i, j int) bool {
		return killStreakPlayers[i].BiggestKillstreak().Killstreak > killStreakPlayers[j].BiggestKillstreak().Killstreak
	})

	if len(killStreakPlayers) > count {
		return killStreakPlayers[0:count]
	}

	return killStreakPlayers
}

func (match *MatchResult) Healers() []*MatchPlayer {
	var healers []*MatchPlayer

	for _, player := range match.Players {
		if player.MedicStats != nil {
			healers = append(healers, player)
		}
	}

	sort.SliceStable(healers, func(i, j int) bool {
		return healers[i].MedicStats.Healing > healers[j].MedicStats.Healing
	})

	return healers
}

type MatchPlayerWeapon struct {
	Weapon
	Kills     int     `json:"kills"`
	Damage    int     `json:"damage"`
	Shots     int     `json:"shots"`
	Hits      int     `json:"hits"`
	Accuracy  float64 `json:"accuracy"`
	Backstabs int     `json:"backstabs"`
	Headshots int     `json:"headshots"`
	Airshots  int     `json:"airshots"`
}

type PlayerClassStats struct {
	Class              logparse.PlayerClass
	ClassName          string
	Kills              int
	Assists            int
	Deaths             int
	Damage             int
	Dominations        int
	Dominated          int
	Revenges           int
	DamageTaken        int
	HealingTaken       int
	HealthPacks        int
	Captures           int
	CapturesBlocked    int
	Extinguishes       int
	BuildingsBuilt     int
	BuildingsDestroyed int
	Playtime           float64 // seconds
}

func (player PlayerClassStats) KDRatio() float64 {
	if player.Deaths <= 0 {
		return -1
	}

	return math.Ceil((float64(player.Kills)/float64(player.Deaths))*100) / 100
}

func (player PlayerClassStats) KDARatio() float64 {
	if player.Deaths <= 0 {
		return -1
	}

	return math.Ceil((float64(player.Kills+player.Assists)/float64(player.Deaths))*100) / 100
}

func (player PlayerClassStats) DamagePerMin() int {
	return int(float64(player.Damage) / (player.Playtime / 60))
}

type PlayerClassStatsCollection []PlayerClassStats

func (ps PlayerClassStatsCollection) Kills() int {
	var total int
	for _, class := range ps {
		total += class.Kills
	}

	return total
}

func (ps PlayerClassStatsCollection) Assists() int {
	var total int
	for _, class := range ps {
		total += class.Assists
	}

	return total
}

func (ps PlayerClassStatsCollection) Deaths() int {
	var total int
	for _, class := range ps {
		total += class.Deaths
	}

	return total
}

func (ps PlayerClassStatsCollection) Damage() int {
	var total int
	for _, class := range ps {
		total += class.Damage
	}

	return total
}

func (ps PlayerClassStatsCollection) DamageTaken() int {
	var total int
	for _, class := range ps {
		total += class.DamageTaken
	}

	return total
}

func (ps PlayerClassStatsCollection) Captures() int {
	var total int
	for _, class := range ps {
		total += class.Captures
	}

	return total
}

func (ps PlayerClassStatsCollection) Dominations() int {
	var total int
	for _, class := range ps {
		total += class.Dominations
	}

	return total
}

func (ps PlayerClassStatsCollection) Dominated() int {
	var total int
	for _, class := range ps {
		total += class.Dominated
	}

	return total
}

func (ps PlayerClassStatsCollection) Playtime() float64 {
	var total float64
	for _, class := range ps {
		total += class.Playtime
	}

	return total
}

func (ps PlayerClassStatsCollection) DamagePerMin() int {
	return int(float64(ps.Damage()) / (ps.Playtime() / 60))
}

func (ps PlayerClassStatsCollection) KDRatio() float64 {
	if ps.Deaths() <= 0 {
		return -1
	}

	return math.Ceil((float64(ps.Kills())/float64(ps.Deaths()))*100) / 100
}

func (ps PlayerClassStatsCollection) KDARatio() float64 {
	if ps.Deaths() <= 0 {
		return -1
	}

	return math.Ceil((float64(ps.Kills()+ps.Assists())/float64(ps.Deaths()))*100) / 100
}

type PlayerWeaponStats struct {
	Weapon     logparse.Weapon
	WeaponName string
	Kills      int
	Damage     int
	Shots      int
	Hits       int
	Backstabs  int
	Headshots  int
	Airshots   int
}

func (ws PlayerWeaponStats) Accuracy() float64 {
	if ws.Shots == 0 {
		return 0
	}

	return math.Ceil(float64(ws.Hits)/float64(ws.Shots)*10000) / 100
}

type PlayerKillstreakStats struct {
	Class     logparse.PlayerClass `json:"class"`
	ClassName string               `json:"class_name"`
	Kills     int                  `json:"kills"`
	Duration  int                  `json:"duration"`
	CreatedOn time.Time            `json:"created_on"`
}

type PlayerMedicStats struct {
	Healing             int
	Drops               int
	NearFullChargeDeath int
	AvgUberLength       float64
	ChargesUber         int
	ChargesKritz        int
	ChargesVacc         int
	ChargesQuickfix     int
}

type CommonPlayerStats struct {
	SteamID           steamid.SteamID `json:"steam_id"`
	Name              string          `json:"name"`
	AvatarHash        string          `json:"avatar_hash"` //todo make
	Kills             int             `json:"kills"`
	Assists           int             `json:"assists"`
	Deaths            int             `json:"deaths"`
	Suicides          int             `json:"suicides"`
	Dominations       int             `json:"dominations"`
	Dominated         int             `json:"dominated"`
	Revenges          int             `json:"revenges"`
	Damage            int             `json:"damage"`
	DamageTaken       int             `json:"damage_taken"`
	HealingTaken      int             `json:"healing_taken"`
	HealthPacks       int             `json:"health_packs"`
	HealingPacks      int             `json:"healing_packs"` // Healing from packs
	Captures          int             `json:"captures"`
	CapturesBlocked   int             `json:"captures_blocked"`
	Extinguishes      int             `json:"extinguishes"`
	BuildingBuilt     int             `json:"building_built"`
	BuildingDestroyed int             `json:"building_destroyed"` // Opposing team buildings
	Backstabs         int             `json:"backstabs"`
	Airshots          int             `json:"airshots"`
	Headshots         int             `json:"headshots"`
	Shots             int             `json:"shots"`
	Hits              int             `json:"hits"`
}
type PlayerStats struct {
	CommonPlayerStats
	PlayerMedicStats
	MatchesTotal int           `json:"matches_total"`
	MatchesWon   int           `json:"matches_won"`
	PlayTime     time.Duration `json:"play_time"`
}

type MatchSummary struct {
	MatchID   uuid.UUID `json:"match_id"`
	ServerID  int       `json:"server_id"`
	IsWinner  bool      `json:"is_winner"`
	ShortName string    `json:"short_name"`
	Title     string    `json:"title"`
	MapName   string    `json:"map_name"`
	ScoreBlu  int       `json:"score_blu"`
	ScoreRed  int       `json:"score_red"`
	TimeStart time.Time `json:"time_start"`
	TimeEnd   time.Time `json:"time_end"`
}

func (m MatchSummary) Path() string {
	return "/log/" + m.MatchID.String()
}
