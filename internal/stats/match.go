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

const MinMedicHealing = 500

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

	SteamID   string
	ServerID  int
	Map       string
	TimeStart *time.Time
	TimeEnd   *time.Time
}

func (mqf QueryOpts) TargetSteamID() (steamid.SteamID, bool) {
	sid := steamid.New(mqf.SteamID)

	return sid, sid.Valid()
}

type Match struct {
	MatchID    uuid.UUID
	ServerID   int32
	Title      string
	Map        maps.Map
	TeamScores logparse.TeamScores
	TimeStart  time.Time
	Duration   time.Duration
	Winner     logparse.Team
	Round      demoparse.DemoRoundSummary
	Players    []*Player
	Chat       []PersonMessage
}

type PlayerKillstreak struct {
	MatchKillstreakID int64
	MatchPlayerID     int64
	PlayerClass       logparse.PlayerClass
	Killstreak        int
	// Seconds
	Duration int
}

type PlayerClassDetail struct {
	MatchPlayerClassID int
	MatchPlayerID      int64
	PlayerClass        logparse.PlayerClass
	Kills              int
	Assists            int
	Deaths             int
	Playtime           int
	Dominations        int
	Dominated          int
	Revenges           int
	Damage             int
	DamageTaken        int
	HealingTaken       int
	Captures           int
	CapturesBlocked    int
	BuildingDestroyed  int
}

type Player struct {
	CommonPlayerStats

	MatchPlayerID int64
	Team          logparse.Team
	TimeStart     time.Time
	TimeEnd       time.Time
	MedicStats    *PlayerMedicStats
	Classes       []PlayerClassDetail
	Killstreaks   []PlayerKillstreak
	Weapons       []PlayerWeapon
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
	MatchMedicID        int64
	MatchPlayerID       int64
	Healing             int
	ChargesUber         int
	ChargesKritz        int
	ChargesVacc         int
	ChargesQuickfix     int
	Drops               int
	NearFullChargeDeath int
	AvgUberLength       float32
	MajorAdvLost        int
	BiggestAdvLost      int
}

func (h Healer) HealingPerMin(matchDuration time.Duration) int {
	if h.Healing <= 0 {
		return 0
	}

	return int(float64(h.Healing) / matchDuration.Minutes())
}

type PlayerMatchWeapon struct {
	PlayerWeaponID int64
	MatchPlayerID  int64
}

type PersonMessage struct {
	PersonMessageID   int64
	MatchID           uuid.UUID
	SteamID           steamid.SteamID
	AvatarHash        string
	PersonaName       string
	ServerName        string
	ServerID          int32
	Body              string
	Tick              int32
	Team              bool
	CreatedOn         time.Time
	AutoFilterFlagged int64
}

func (match *Match) TopPlayers() []*Player {
	players := match.Players

	sort.SliceStable(players, func(i, j int) bool {
		return players[i].Kills > players[j].Kills
	})

	return players
}

func (match *Match) TopKillstreaks(count int) []*Player {
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

func (match *Match) Healers() []*Player {
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

	Kills     int
	Damage    int
	Shots     int
	Hits      int
	Accuracy  float64
	Backstabs int
	Headshots int
	Airshots  int
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
	Class     logparse.PlayerClass
	ClassName string
	Kills     int
	Duration  int
	CreatedOn time.Time
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
	PreroundHealing     int
	PostroundHealing    int
}

type CommonPlayerStats struct {
	SteamID           steamid.SteamID
	Name              string
	AvatarHash        string
	Kills             int
	Assists           int
	Deaths            int
	Suicides          int
	Dominations       int
	Dominated         int
	Revenges          int
	Revenged          int
	Damage            int
	DamageTaken       int
	HealingTaken      int
	HealthPacks       int
	HealingPacks      int
	Captures          int
	CapturesBlocked   int
	Extinguishes      int
	BuildingBuilt     int
	BuildingDestroyed int
	Backstabs         int
	Airshots          int
	Headshots         int
	Shots             int
	Hits              int
	HeadshotKills     int
	BackstabKills     int
	WasHeadshot       int
	WasBackstabbed    int
	PostroundKills    int
	PostroundAssists  int
	PostroundDeaths   int
}

type PlayerStats struct {
	CommonPlayerStats
	PlayerMedicStats

	MatchesTotal int
	MatchesWon   int
	PlayTime     time.Duration
}

type Summary struct {
	MatchID   uuid.UUID
	ServerID  int
	IsWinner  bool
	ShortName string
	Title     string
	MapName   string
	ScoreBlu  int
	ScoreRed  int
	TimeStart time.Time
	TimeEnd   time.Time
}

func (m Summary) Path() string {
	return "/log/" + m.MatchID.String()
}
