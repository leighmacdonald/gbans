package app

import (
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"time"
)

var ErrIgnored = errors.New("Ignored msg")
var ErrUnhandled = errors.New("Unhandled msg")

// Match and its related Match* structs are designed as a close to 1:1 mirror of the
// logs.tf ui
//
// For a simple example of usage, see internal/cmd/stats.go
//
// TODO
// - Use medic death event to calc medic healing count
// - Use healed event for tracking healing received
// - individual game state cache to track who is on winning team
// - Filter out certain pre-game events likes kills/damage
// - Track current player session
// - Track player playtime per class
// - Track server playtime per class
// - Track global playtime per class
// - Track player midfights won
// - Track player biggest killstreaks (min 18 players in server)
// - Track server biggest killstreaks (min 18 players in server)
// - Track global biggest killstreaks (min 18 players in server)
// - Track player classes killed
// - Track player classes killedBy
// - Track server classes killed
// - Track server classes killedBy
// - Track global classes killed
// - Track global classes killedBy
// - Calculate player points
// - Calculate server points
// - Calculate global points
// - Track player weapon stats
// - Track server weapon stats
// - Track global weapon stats
// - calc HealsTaken (live round time only)
// - calc Heals/min (live round time only)
// - calc Dmg/min (live round time only)
// - calc DmgTaken/min (live round time only)
// - Count headshots
// - Count airshots
// - Count headshots
// - Track current map to get correct map stats. Tracking the sm_nextmap cvar may partially work for old data.
//   Update sourcemod plugin to send log event with the current map.
// - Simplify implementation of the maps with generics
// - Track players taking packs when they are close to 100% hp
type Match struct {
	MatchID           int64
	title             string
	mapName           string
	playerSums        map[steamid.SID64]*MatchPlayerSum
	medicSums         map[steamid.SID64]*MatchMedicSum
	teamSums          map[logparse.Team]*MatchTeamSum
	rounds            []*MatchRoundSum
	classKills        MatchPlayerClassSums
	classKillsAssists MatchPlayerClassSums
	classDeaths       MatchPlayerClassSums

	// inMatch is set to true when we start a round, many stat events are ignored until this is true
	inMatch    bool // We ignore most events until Round_Start event
	state      *playerCache
	useRealDmg bool
}

func (m *Match) Apply(event model.ServerEvent) error {
	switch event.EventType {
	case logparse.MapLoad:
		mn, ok := event.MetaData["map"]
		if ok {
			m.mapName = mn.(string)
		}
	case logparse.IgnoredMsg:
		return ErrIgnored
	case logparse.UnknownMsg:
		return ErrUnhandled
	case logparse.WRoundStart:
		m.inMatch = true
		return nil
	case logparse.WGameOver:
		fallthrough
	case logparse.WRoundWin:
		m.inMatch = false
		return nil
	}

	if m.inMatch {
		switch event.EventType {
		case logparse.SpawnedAs:
			m.state.setClass(event.Source.SteamID, event.PlayerClass)
			m.addClass(event.Source.SteamID, event.PlayerClass)
		case logparse.ChangeClass:
			m.state.setClass(event.Source.SteamID, event.PlayerClass)
		case logparse.ShotFired:
			m.shotFired(event.Source.SteamID)
		case logparse.ShotHit:
			m.shotHit(event.Source.SteamID)
		case logparse.MedicDeath:
			if event.GetValueBool("ubercharge") {
				// TODO record source player stat
				m.drop(event.Target.SteamID)
			}
		case logparse.MedicDeathEx:
			m.medicDeath(event.Source.SteamID, event.GetValueInt("uberpct"))
		case logparse.Domination:
			m.domination(event.Source.SteamID, event.Target.SteamID)
		case logparse.Revenge:
			m.revenge(event.Source.SteamID)
		case logparse.Damage:
			// It's a pub, so why not count over kill dmg
			if m.useRealDmg {
				m.damage(event.Source.SteamID, event.Target.SteamID, event.RealDamage)
			} else {
				m.damage(event.Source.SteamID, event.Target.SteamID, event.Damage)
			}
		case logparse.Killed:
			m.killed(event.Source.SteamID, event.Target.SteamID)
		case logparse.KilledCustom:
			fd, fdF := event.MetaData["customkill"]
			if fdF {
				m.killedCustom(event.Source.SteamID, event.Target.SteamID, fd.(string))
			}
		case logparse.KillAssist:
			m.getPlayerSum(event.Source.SteamID).Assists++
		case logparse.Healed:
			m.getPlayerSum(event.Source.SteamID).Healing += event.Healing
			m.getPlayerSum(event.Target.SteamID).HealingTaken += event.Healing
		case logparse.Extinguished:
			m.getPlayerSum(event.Source.SteamID).Extinguishes++
		case logparse.BuiltObject:
			m.getPlayerSum(event.Source.SteamID).BuildingBuilt++
		case logparse.KilledObject:
			m.getPlayerSum(event.Source.SteamID).BuildingDestroyed++
		case logparse.Pickup:
			switch event.Item {
			case logparse.ItemHPSmall:
				fallthrough
			case logparse.ItemHPMedium:
				fallthrough
			case logparse.ItemHPLarge:
				p := m.getPlayerSum(event.Source.SteamID)
				p.HealthPacks++
				p.Healing += event.Healing
			}
		default:
			log.Tracef("Unhandled apply event")
		}
	}
	return nil
}

func (m *Match) getPlayerSum(sid steamid.SID64) *MatchPlayerSum {
	_, ok := m.playerSums[sid]
	if !ok {
		m.playerSums[sid] = &MatchPlayerSum{}
		if m.inMatch {
			// Account for people who joined after Round_start event
			m.playerSums[sid].touch()
		}
	}
	return m.playerSums[sid]
}

func (m *Match) getMedicSum(sid steamid.SID64) *MatchMedicSum {
	_, ok := m.medicSums[sid]
	if !ok {
		m.medicSums[sid] = &MatchMedicSum{}
	}
	return m.medicSums[sid]
}

func (m *Match) addClass(sid steamid.SID64, class logparse.PlayerClass) {
	p := m.getPlayerSum(sid)
	if !fp.Contains[logparse.PlayerClass](p.Classes, class) {
		p.Classes = append(p.Classes, class)
		if class == logparse.Medic {
			// Allocate for a new medic
			m.medicSums[sid] = &MatchMedicSum{}
		}
	}
	if m.inMatch {
		p.touch()
	}
}

func (m *Match) shotFired(sid steamid.SID64) {
	m.getPlayerSum(sid).Shots++
}

func (m *Match) shotHit(sid steamid.SID64) {
	m.getPlayerSum(sid).Hits++
}

func (m *Match) domination(source steamid.SID64, target steamid.SID64) {
	m.getPlayerSum(source).Dominations++
	m.getPlayerSum(target).Dominated++
}

func (m *Match) revenge(source steamid.SID64) {
	m.getPlayerSum(source).Revenges++
}

func (m *Match) damage(source steamid.SID64, target steamid.SID64, damage int64) {
	m.getPlayerSum(source).Damage += damage
	m.getPlayerSum(target).DamageTaken += damage
}

func (m *Match) killed(source steamid.SID64, target steamid.SID64) {
	m.getPlayerSum(source).Kills++
	m.getPlayerSum(target).Deaths++
}

func (m *Match) killedCustom(source steamid.SID64, target steamid.SID64, custom string) {
	switch custom {
	case "feign_death":
		// Ignore DR
		return
	case "backstab":
		m.getPlayerSum(source).BackStabs++
	case "headshot":
		m.getPlayerSum(source).HeadShots++
	case "airshot":
		m.getPlayerSum(source).Airshots++
	}
	m.getPlayerSum(source).Kills++
	m.getPlayerSum(target).Deaths++
}

func (m *Match) drop(source steamid.SID64) {
	m.getMedicSum(source).Drops++
}

func (m *Match) medicDeath(source steamid.SID64, uberPct int) {
	if uberPct > 95 && uberPct < 100 {
		m.getMedicSum(source).NearFullChargeDeath++
	}
}

func NewMatch() Match {
	return Match{
		MatchID:           0,
		title:             "Qixalite Booking: RED vs BLU",
		mapName:           "koth_cascade_rc2",
		playerSums:        map[steamid.SID64]*MatchPlayerSum{},
		medicSums:         map[steamid.SID64]*MatchMedicSum{},
		teamSums:          map[logparse.Team]*MatchTeamSum{},
		rounds:            nil,
		classKills:        MatchPlayerClassSums{},
		classKillsAssists: MatchPlayerClassSums{},
		classDeaths:       MatchPlayerClassSums{},
		state:             newPlayerCache(),
		inMatch:           false,
	}
}

type MatchPlayerSum struct {
	Team              logparse.Team
	TimeStart         time.Time
	TimeEnd           time.Time
	Kills             int
	Assists           int
	Deaths            int
	Dominations       int
	Dominated         int
	Revenges          int
	Damage            int64
	DamageTaken       int64
	Healing           int64
	HealingTaken      int64
	HealthPacks       int
	BackStabs         int
	HeadShots         int
	Airshots          int
	Captures          int
	Shots             int
	Hits              int
	Extinguishes      int
	BuildingBuilt     int
	BuildingDestroyed int
	Classes           []logparse.PlayerClass
	timeStart         *time.Time
	timeEnd           *time.Time
}

func (p *MatchPlayerSum) touch() {
	if p.timeStart == nil {
		t := config.Now()
		p.timeStart = &t
	}
}

type TeamScores struct {
	Red int
	Blu int
}

type MatchRoundSum struct {
	Length    time.Duration
	Score     TeamScores
	KillsBlu  int
	KillsRed  int
	UbersBlu  int
	UbersRed  int
	DamageBlu int
	DamageRed int
	MidFight  logparse.Team
}

type MatchMedicSum struct {
	Healing             int
	Charges             map[logparse.Medigun]int
	Drops               int
	AvgTimeToBuild      int
	AvgTimeBeforeUse    int
	NearFullChargeDeath int
	AvgUberLength       float32
	DeathAfterCharge    int
	MajorAdvLost        int
	BiggestAdvLost      int
	HealTargets         MatchPlayerClassSums
}

type MatchClassSums struct {
	Scout    int
	Soldier  int
	Pyro     int
	Demoman  int
	Heavy    int
	Engineer int
	Medic    int
	Sniper   int
	Spy      int
}

func (m *MatchClassSums) Sum() int {
	return m.Scout + m.Soldier + m.Pyro +
		m.Demoman + m.Heavy + m.Engineer +
		m.Medic + m.Spy + m.Sniper
}

type MatchPlayerClassSums map[steamid.SID64]*MatchClassSums

type MatchTeamSum struct {
	Kills     int
	Damage    int
	Charges   int
	Drops     int
	Caps      int
	MidFights int
}
