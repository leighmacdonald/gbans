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
		inMatch:           false,
	}
}

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
	inRound    bool
	useRealDmg bool
}

func (m *Match) Apply(event model.ServerEvent) error {
	switch event.EventType {
	case logparse.MapLoad:
		mn, ok := event.MetaData["map"]
		if ok {
			m.mapName = mn.(string)
		}
		return nil
	case logparse.IgnoredMsg:
		return ErrIgnored
	case logparse.UnknownMsg:
		return ErrUnhandled
	case logparse.WRoundStart:
		m.roundStart()
		return nil
	case logparse.WGameOver:
		m.gameOver()
	case logparse.WMiniRoundStart:
		m.roundStart()
	case logparse.WRoundOvertime:
	case logparse.WRoundLen:
	case logparse.WRoundWin:
		m.roundWin(event.Team)
		return nil
	}
	if !m.inMatch || !m.inRound {
		return nil
	}
	switch event.EventType {
	case logparse.SpawnedAs:
		m.addClass(event.Source.SteamID, event.PlayerClass)
	case logparse.ChangeClass:
		m.addClass(event.Source.SteamID, event.PlayerClass)
	case logparse.ShotFired:
		m.shotFired(event.Source.SteamID)
	case logparse.ShotHit:
		m.shotHit(event.Source.SteamID)
	case logparse.MedicDeath:
		if event.GetValueBool("ubercharge") {
			// TODO record source player stat
			m.drop(event.Target.SteamID, event.Team.Opponent())
		}
	case logparse.EmptyUber:

	case logparse.ChargeDeployed:
		m.medicCharge(event.Source.SteamID, event.MetaData["medigun"].(logparse.Medigun), event.Team)
	case logparse.ChargeEnded:
	case logparse.ChargeReady:
	case logparse.LostUberAdv:
		m.medicLostAdv(event.Source.SteamID, event.GetValueInt("time"))
	case logparse.MedicDeathEx:
		m.medicDeath(event.Source.SteamID, event.GetValueInt("uberpct"))
	case logparse.Domination:
		m.domination(event.Source.SteamID, event.Target.SteamID)
	case logparse.Revenge:
		m.revenge(event.Source.SteamID)
	case logparse.Damage:
		// It's a pub, so why not count over kill dmg
		if m.useRealDmg {
			m.damage(event.Source.SteamID, event.Target.SteamID, event.RealDamage, event.Team)
		} else {
			m.damage(event.Source.SteamID, event.Target.SteamID, event.Damage, event.Team)
		}
	case logparse.Killed:
		m.killed(event.Source.SteamID, event.Target.SteamID, event.Team)
	case logparse.KilledCustom:
		fd, fdF := event.MetaData["customkill"]
		if fdF {
			m.killedCustom(event.Source.SteamID, event.Target.SteamID, fd.(string))
		}
	case logparse.KillAssist:
		m.assist(event.Source.SteamID)
	case logparse.Healed:
		m.healed(event.Source.SteamID, event.Target.SteamID, event.Healing)
	case logparse.Extinguished:
		m.extinguishes(event.Source.SteamID)
	case logparse.BuiltObject:
		m.builtObject(event.Source.SteamID)
	case logparse.KilledObject:
		m.killedObject(event.Source.SteamID)
	case logparse.Pickup:
		m.pickup(event.Source.SteamID, event.Item, event.Healing)
	default:
		log.Tracef("Unhandled apply event")
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
		m.medicSums[sid] = newMatchMedicSum()
	}
	return m.medicSums[sid]
}

func (m *Match) getTeamSum(team logparse.Team) *MatchTeamSum {
	_, ok := m.teamSums[team]
	if !ok {
		m.teamSums[team] = newMatchTeamSum()
	}
	return m.teamSums[team]
}

func (m *Match) roundStart() {
	m.inMatch = true
	m.inRound = true
	for _, p := range m.playerSums {
		if p.timeStart == nil {
			*p.timeStart = config.Now()
		}
	}
}

func (m *Match) roundWin(team logparse.Team) {
	m.inMatch = true
	m.inRound = false
}

func (m *Match) gameOver() {
	m.inMatch = false
	m.inRound = false
}

func (m *Match) addClass(sid steamid.SID64, class logparse.PlayerClass) {
	if class == logparse.Spectator {
		return
	}
	p := m.getPlayerSum(sid)
	if !fp.Contains[logparse.PlayerClass](p.Classes, class) {
		p.Classes = append(p.Classes, class)
		if class == logparse.Medic {
			// Allocate for a new medic
			m.medicSums[sid] = newMatchMedicSum()
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

func (m *Match) assist(sid steamid.SID64) {
	m.getPlayerSum(sid).Assists++
}

func (m *Match) domination(source steamid.SID64, target steamid.SID64) {
	m.getPlayerSum(source).Dominations++
	m.getPlayerSum(target).Dominated++
}

func (m *Match) revenge(source steamid.SID64) {
	m.getPlayerSum(source).Revenges++
}

func (m *Match) builtObject(source steamid.SID64) {
	m.getPlayerSum(source).BuildingBuilt++
}

func (m *Match) killedObject(source steamid.SID64) {
	m.getPlayerSum(source).BuildingDestroyed++
}

func (m *Match) extinguishes(source steamid.SID64) {
	m.getPlayerSum(source).Extinguishes++
}

func (m *Match) damage(source steamid.SID64, target steamid.SID64, damage int64, team logparse.Team) {
	m.getPlayerSum(source).Damage += damage
	m.getPlayerSum(target).DamageTaken += damage
	m.getTeamSum(team).Damage += damage
}

func (m *Match) healed(source steamid.SID64, target steamid.SID64, amount int64) {
	m.getPlayerSum(source).Healing += amount
	m.getPlayerSum(target).HealingTaken += amount
}

func (m *Match) pointCapture(team logparse.Team, sources steamid.Collection) {
	for _, sid := range sources {
		m.getPlayerSum(sid).Captures++
	}
	m.getTeamSum(team).Caps++
}

func (m *Match) midFight(team logparse.Team) {
	m.getTeamSum(team).MidFights++
}

func (m *Match) killed(source steamid.SID64, target steamid.SID64, team logparse.Team) {
	src := m.getPlayerSum(source)
	if m.inRound {
		src.Kills++
		m.getPlayerSum(target).Deaths++
		m.getTeamSum(team).Kills++
	}
}

func (m *Match) pickup(source steamid.SID64, item logparse.PickupItem, healing int64) {
	switch item {
	case logparse.ItemHPSmall:
		fallthrough
	case logparse.ItemHPMedium:
		fallthrough
	case logparse.ItemHPLarge:
		p := m.getPlayerSum(source)
		p.HealthPacks++
		p.Healing += healing
	}
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

func (m *Match) drop(source steamid.SID64, team logparse.Team) {
	m.getMedicSum(source).Drops++
	m.getTeamSum(team).Drops++
}

func (m *Match) medicDeath(source steamid.SID64, uberPct int) {
	if uberPct > 95 && uberPct < 100 {
		m.getMedicSum(source).NearFullChargeDeath++
	}
}

func (m *Match) medicCharge(source steamid.SID64, weapon logparse.Medigun, team logparse.Team) {
	s := m.getMedicSum(source)
	_, found := s.Charges[weapon]
	if !found {
		s.Charges[weapon] = 0
	}
	s.Charges[weapon]++
	m.getTeamSum(team).Charges++
}

func (m *Match) medicLostAdv(source steamid.SID64, timeAdv int) {
	sum := m.getMedicSum(source)
	if timeAdv > 30 {
		// TODO check what is actually the time to trigger
		sum.MajorAdvLost++
	}
	if timeAdv > sum.BiggestAdvLost {
		sum.BiggestAdvLost = timeAdv
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

func newMatchMedicSum() *MatchMedicSum {
	return &MatchMedicSum{
		Charges: map[logparse.Medigun]int{},
	}
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
	Damage    int64
	Charges   int
	Drops     int
	Caps      int
	MidFights int
}

func newMatchTeamSum() *MatchTeamSum {
	return &MatchTeamSum{}
}
