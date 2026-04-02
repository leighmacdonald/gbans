package stats

import (
	"math"
	"slices"
	"sort"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/maps"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/pkg/demoparse"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type TriggerType int

const (
	TriggerStart TriggerType = 1
	TriggerEnd   TriggerType = 2
)

type Trigger struct {
	Type     TriggerType
	UUID     uuid.UUID
	Server   servers.Server
	MapName  string
	DemoName string
}

type QueryOpts struct {
	query.Filter

	SteamID   string     `json:"steam_id"`
	ServerID  int        `json:"server_id"`
	Map       string     `json:"map"`
	TimeStart *time.Time `json:"time_start,omitempty"`
	TimeEnd   *time.Time `json:"time_end,omitempty"`
}

func (mqf QueryOpts) TargetSteamID() (steamid.SteamID, bool) {
	sid := steamid.New(mqf.SteamID)

	return sid, sid.Valid()
}

const MinMedicHealing = 500

type PlayerKillstreak struct {
	MatchKillstreakID int64                `json:"match_killstreak_id"`
	MatchPlayerID     int64                `json:"match_player_id"`
	PlayerClass       logparse.PlayerClass `json:"player_class"`
	Killstreak        int                  `json:"killstreak"`
	// Seconds
	Duration int `json:"duration"`
}

type PlayerClassDetail struct {
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

type Player struct {
	CommonPlayerStats

	MatchPlayerID int64               `json:"match_player_id"`
	Team          logparse.Team       `json:"team"`
	TimeStart     time.Time           `json:"time_start"`
	TimeEnd       time.Time           `json:"time_end"`
	MedicStats    *PlayerMedicStats   `json:"medic_stats"`
	Classes       []PlayerClassDetail `json:"classes"`
	Killstreaks   []PlayerKillstreak  `json:"killstreaks"`
	Weapons       []PlayerWeapon      `json:"weapons"`
}

func (p *Player) ApplySummary(update *demoparse.PlayerSummary) {
	p.Kills += update.Kills
	p.Assists += update.Assists
	p.Deaths += update.Deaths

	p.PostroundKills += update.PostroundKills
	p.PostroundAssists += update.PostroundAssists
	p.PostroundDeaths += update.PostroundDeaths

	p.Damage += update.Damage
	p.DamageTaken += update.Damage

	p.Dominations += update.Dominations
	p.Dominated += update.Dominated
	p.Revenges += update.Revenges
	p.Revenged += update.Revenged

	p.Airshots += update.Airshots
	p.HeadshotKills += update.HeadshotKills
	p.BackstabKills += update.BackstabKills
	p.Headshots += update.Headshots
	p.Backstabs += update.Backstabs
	p.WasHeadshot += update.WasHeadshot
	p.WasBackstabbed += update.WasBackstabbed

	p.MedicStats.PreroundHealing += update.PreroundHealing
	p.MedicStats.PostroundHealing += update.PostroundHealing
	p.MedicStats.Healing += update.Healing
	p.MedicStats.Drops += update.Drops
	p.MedicStats.NearFullChargeDeath += update.NearFullChargeDeath
	p.MedicStats.ChargesVacc += update.ChargesVacc
	p.MedicStats.ChargesKritz += update.ChargesKritz
	p.MedicStats.ChargesQuickfix += update.ChargesQuickfix

	// for weaponName, upd := range update.Weapons {
	// 	cs := PlayerWeaponStats{
	// 		WeaponName: weaponName,
	// 		Kills:      upd.Kills,
	// 		Damage:     upd.Damage,
	// 		Backstabs:  upd.BackstabKills,
	// 		Airshots:   upd.Airshots,
	// 		Headshots:  upd.HeadshotKills,
	// 		//TODO shots/hits
	// 	}

	// 	p.Weapons = append(p.Weapons, cs)
	// }
}

func (p *Player) BiggestKillstreak() *PlayerKillstreak {
	var biggest *PlayerKillstreak

	for _, killstreakVal := range p.Killstreaks {
		killstreak := killstreakVal
		if biggest == nil || killstreak.Killstreak > biggest.Killstreak {
			biggest = &killstreak
		}
	}

	return biggest
}

func (p *Player) KDRatio() float64 {
	if p.Deaths <= 0 {
		return -1
	}

	return math.Ceil((float64(p.Kills)/float64(p.Deaths))*100) / 100
}

func (p *Player) KDARatio() float64 {
	if p.Deaths <= 0 {
		return -1
	}

	return math.Ceil((float64(p.Kills+p.Assists)/float64(p.Deaths))*100) / 100
}

func (p *Player) DamagePerMin() int {
	return slices.Max([]int{int(float64(p.Damage) / p.TimeEnd.Sub(p.TimeStart).Minutes()), 0})
}

type Healer struct {
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

func (h Healer) HealingPerMin(matchDuration time.Duration) int {
	if h.Healing <= 0 {
		return 0
	}

	return int(float64(h.Healing) / matchDuration.Minutes())
}

type PlayerMatchWeapon struct {
	PlayerWeaponID int64 `json:"player_weapon_id"`
	MatchPlayerID  int64 `json:"match_player_id"`
}

type Result struct {
	MatchID    uuid.UUID           `json:"match_id"`
	ServerID   int                 `json:"server_id"`
	Title      string              `json:"title"`
	Map        maps.Map            `json:"map"`
	TeamScores logparse.TeamScores `json:"team_scores"`
	TimeStart  time.Time           `json:"time_start"`
	TimeEnd    time.Time           `json:"time_end"`
	Winner     logparse.Team       `json:"winner"`
	Players    []*Player           `json:"players"`
	Chat       []PersonMessage     `json:"chat"`
}
type PersonMessage struct {
	PersonMessageID   int64           `json:"person_message_id"`
	MatchID           uuid.UUID       `json:"match_id"`
	SteamID           steamid.SteamID `json:"steam_id"`
	AvatarHash        string          `json:"avatar_hash"`
	PersonaName       string          `json:"persona_name"`
	ServerName        string          `json:"server_name"`
	ServerID          int             `json:"server_id"`
	Body              string          `json:"body"`
	Tick              int64           `json:"tick"`
	Team              bool            `json:"team"`
	CreatedOn         time.Time       `json:"created_on"`
	AutoFilterFlagged int64           `json:"auto_filter_flagged"`
}

func (match *Result) TopPlayers() []*Player {
	players := match.Players

	sort.SliceStable(players, func(i, j int) bool {
		return players[i].Kills > players[j].Kills
	})

	return players
}

func (match *Result) TopKillstreaks(count int) []*Player {
	var killStreakPlayers []*Player

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

func (match *Result) Healers() []*Player {
	var healers []*Player

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

type PlayerWeapon struct {
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

type ClassStats struct {
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

func (player ClassStats) KDRatio() float64 {
	if player.Deaths <= 0 {
		return -1
	}

	return math.Ceil((float64(player.Kills)/float64(player.Deaths))*100) / 100
}

func (player ClassStats) KDARatio() float64 {
	if player.Deaths <= 0 {
		return -1
	}

	return math.Ceil((float64(player.Kills+player.Assists)/float64(player.Deaths))*100) / 100
}

func (player ClassStats) DamagePerMin() int {
	return int(float64(player.Damage) / (player.Playtime / 60))
}

type PlayerClassStatsCollection []ClassStats

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
	Healing             int     `json:"healing"`
	Drops               int     `json:"drops"`
	NearFullChargeDeath int     `json:"near_full_charge_death"`
	AvgUberLength       float64 `json:"avg_uber_length"`
	ChargesUber         int     `json:"charges_uber"`
	ChargesKritz        int     `json:"charges_kritz"`
	ChargesVacc         int     `json:"charges_vacc"`
	ChargesQuickfix     int     `json:"charges_quickfix"`
	PreroundHealing     int     `json:"preround_healing"`
	PostroundHealing    int     `json:"postround_healing"`
}

type CommonPlayerStats struct {
	SteamID           steamid.SteamID `json:"steam_id"`
	Name              string          `json:"name"`
	AvatarHash        string          `json:"avatar_hash"` // todo make
	Kills             int             `json:"kills"`
	Assists           int             `json:"assists"`
	Deaths            int             `json:"deaths"`
	Suicides          int             `json:"suicides"`
	Dominations       int             `json:"dominations"`
	Dominated         int             `json:"dominated"`
	Revenges          int             `json:"revenges"`
	Revenged          int             `json:"revenged"`
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
	HeadshotKills     int             `json:"headshot_kills"`
	BackstabKills     int             `json:"backstab_kills"`
	WasHeadshot       int             `json:"was_headshot"`
	WasBackstabbed    int             `json:"was_backstabbed"`
	PostroundKills    int             `json:"postround_kills"`
	PostroundAssists  int             `json:"postround_assists"`
	PostroundDeaths   int             `json:"postround_deaths"`
}

type PlayerStats struct {
	CommonPlayerStats
	PlayerMedicStats

	MatchesTotal int           `json:"matches_total"`
	MatchesWon   int           `json:"matches_won"`
	PlayTime     time.Duration `json:"play_time"`
}

type Summary struct {
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

func (m Summary) Path() string {
	return "/log/" + m.MatchID.String()
}
