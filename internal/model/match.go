package model

import (
	"github.com/leighmacdonald/gbans/internal/config"
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
		Title:             "Team Fortress 3",
		MapName:           "pl_you_didnt_set_this",
		PlayerSums:        map[steamid.SID64]*MatchPlayerSum{},
		MedicSums:         map[steamid.SID64]*MatchMedicSum{},
		TeamSums:          map[logparse.Team]*MatchTeamSum{},
		Rounds:            nil,
		ClassKills:        MatchPlayerClassSums{},
		ClassKillsAssists: MatchPlayerClassSums{},
		ClassDeaths:       MatchPlayerClassSums{},
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
	Title             string
	MapName           string
	PlayerSums        map[steamid.SID64]*MatchPlayerSum
	MedicSums         map[steamid.SID64]*MatchMedicSum
	TeamSums          map[logparse.Team]*MatchTeamSum
	Rounds            []*MatchRoundSum
	ClassKills        MatchPlayerClassSums
	ClassKillsAssists MatchPlayerClassSums
	ClassDeaths       MatchPlayerClassSums

	// inMatch is set to true when we start a round, many stat events are ignored until this is true
	inMatch    bool // We ignore most events until Round_Start event
	inRound    bool
	useRealDmg bool
}

func (match *Match) Apply(event ServerEvent) error {
	switch event.EventType {
	case logparse.MapLoad:
		mapName, ok := event.MetaData["map"]
		if ok {
			match.MapName = mapName.(string)
		}
		return nil
	case logparse.IgnoredMsg:
		return ErrIgnored
	case logparse.UnknownMsg:
		return ErrUnhandled
	case logparse.WRoundStart:
		match.roundStart()
		return nil
	case logparse.WGameOver:
		match.gameOver()
	case logparse.WMiniRoundStart:
		match.roundStart()
	case logparse.WRoundOvertime:
	case logparse.WRoundLen:
	case logparse.WRoundWin:
		match.roundWin(event.Team)
		return nil
	}
	if !match.inMatch || !match.inRound {
		return nil
	}
	switch event.EventType {
	case logparse.SpawnedAs:
		match.addClass(event.Source.SteamID, event.PlayerClass)
	case logparse.ChangeClass:
		match.addClass(event.Source.SteamID, event.PlayerClass)
	case logparse.ShotFired:
		match.shotFired(event.Source.SteamID)
	case logparse.ShotHit:
		match.shotHit(event.Source.SteamID)
	case logparse.MedicDeath:
		if event.GetValueBool("ubercharge") {
			// TODO record source player stat
			match.drop(event.Target.SteamID, event.Team.Opponent())
		}
	case logparse.EmptyUber:

	case logparse.ChargeDeployed:
		match.medicCharge(event.Source.SteamID, event.MetaData["medigun"].(logparse.Medigun), event.Team)
	case logparse.ChargeEnded:
	case logparse.ChargeReady:
	case logparse.LostUberAdv:
		match.medicLostAdv(event.Source.SteamID, event.GetValueInt("time"))
	case logparse.MedicDeathEx:
		match.medicDeath(event.Source.SteamID, event.GetValueInt("uberpct"))
	case logparse.Domination:
		match.domination(event.Source.SteamID, event.Target.SteamID)
	case logparse.Revenge:
		match.revenge(event.Source.SteamID)
	case logparse.Damage:
		// It's a pub, so why not count over kill dmg
		if match.useRealDmg {
			match.damage(event.Source.SteamID, event.Target.SteamID, event.RealDamage, event.Team)
		} else {
			match.damage(event.Source.SteamID, event.Target.SteamID, event.Damage, event.Team)
		}
	case logparse.Killed:
		match.killed(event.Source.SteamID, event.Target.SteamID, event.Team)
	case logparse.KilledCustom:
		fd, fdF := event.MetaData["customkill"]
		if fdF {
			match.killedCustom(event.Source.SteamID, event.Target.SteamID, fd.(string))
		}
	case logparse.KillAssist:
		match.assist(event.Source.SteamID)
	case logparse.Healed:
		match.healed(event.Source.SteamID, event.Target.SteamID, event.Healing)
	case logparse.Extinguished:
		match.extinguishes(event.Source.SteamID)
	case logparse.BuiltObject:
		match.builtObject(event.Source.SteamID)
	case logparse.KilledObject:
		match.killedObject(event.Source.SteamID)
	case logparse.Pickup:
		match.pickup(event.Source.SteamID, event.Item, event.Healing)
	default:
		log.Tracef("Unhandled apply event")
	}

	return nil
}

func (match *Match) getPlayerSum(sid steamid.SID64) *MatchPlayerSum {
	_, ok := match.PlayerSums[sid]
	if !ok {
		match.PlayerSums[sid] = &MatchPlayerSum{}
		if match.inMatch {
			// Account for people who joined after Round_start event
			match.PlayerSums[sid].touch()
		}
	}
	return match.PlayerSums[sid]
}

func (match *Match) getMedicSum(sid steamid.SID64) *MatchMedicSum {
	_, ok := match.MedicSums[sid]
	if !ok {
		match.MedicSums[sid] = newMatchMedicSum()
	}
	return match.MedicSums[sid]
}

func (match *Match) getTeamSum(team logparse.Team) *MatchTeamSum {
	_, ok := match.TeamSums[team]
	if !ok {
		match.TeamSums[team] = newMatchTeamSum()
	}
	return match.TeamSums[team]
}

func (match *Match) roundStart() {
	match.inMatch = true
	match.inRound = true
	for _, playerSum := range match.PlayerSums {
		if playerSum.timeStart == nil {
			*playerSum.timeStart = config.Now()
		}
	}
}

func (match *Match) roundWin(team logparse.Team) {
	match.inMatch = true
	match.inRound = false
	//match.rounds[0].Score.
}

func (match *Match) gameOver() {
	match.inMatch = false
	match.inRound = false
}

func (match *Match) addClass(sid steamid.SID64, class logparse.PlayerClass) {
	if class == logparse.Spectator {
		return
	}
	playerSum := match.getPlayerSum(sid)
	if !fp.Contains[logparse.PlayerClass](playerSum.Classes, class) {
		playerSum.Classes = append(playerSum.Classes, class)
		if class == logparse.Medic {
			// Allocate for a new medic
			match.MedicSums[sid] = newMatchMedicSum()
		}
	}
	if match.inMatch {
		playerSum.touch()
	}
}

func (match *Match) shotFired(sid steamid.SID64) {
	match.getPlayerSum(sid).Shots++
}

func (match *Match) shotHit(sid steamid.SID64) {
	match.getPlayerSum(sid).Hits++
}

func (match *Match) assist(sid steamid.SID64) {
	match.getPlayerSum(sid).Assists++
}

func (match *Match) domination(source steamid.SID64, target steamid.SID64) {
	match.getPlayerSum(source).Dominations++
	match.getPlayerSum(target).Dominated++
}

func (match *Match) revenge(source steamid.SID64) {
	match.getPlayerSum(source).Revenges++
}

func (match *Match) builtObject(source steamid.SID64) {
	match.getPlayerSum(source).BuildingBuilt++
}

func (match *Match) killedObject(source steamid.SID64) {
	match.getPlayerSum(source).BuildingDestroyed++
}

func (match *Match) extinguishes(source steamid.SID64) {
	match.getPlayerSum(source).Extinguishes++
}

func (match *Match) damage(source steamid.SID64, target steamid.SID64, damage int64, team logparse.Team) {
	match.getPlayerSum(source).Damage += damage
	match.getPlayerSum(target).DamageTaken += damage
	match.getTeamSum(team).Damage += damage
}

func (match *Match) healed(source steamid.SID64, target steamid.SID64, amount int64) {
	match.getPlayerSum(source).Healing += amount
	match.getPlayerSum(target).HealingTaken += amount
}

//func (match *Match) pointCapture(team logparse.Team, sources steamid.Collection) {
//	for _, sid := range sources {
//		match.getPlayerSum(sid).Captures++
//	}
//	match.getTeamSum(team).Caps++
//}

//func (match *Match) midFight(team logparse.Team) {
//	match.getTeamSum(team).MidFights++
//}

func (match *Match) killed(source steamid.SID64, target steamid.SID64, team logparse.Team) {
	if match.inRound {
		match.getPlayerSum(source).Kills++
		match.getPlayerSum(target).Deaths++
		match.getTeamSum(team).Kills++
	}
}

func (match *Match) pickup(source steamid.SID64, item logparse.PickupItem, healing int64) {
	switch item {
	case logparse.ItemHPSmall:
		fallthrough
	case logparse.ItemHPMedium:
		fallthrough
	case logparse.ItemHPLarge:
		p := match.getPlayerSum(source)
		p.HealthPacks++
		p.Healing += healing
	}
}

func (match *Match) killedCustom(source steamid.SID64, target steamid.SID64, custom string) {
	switch custom {
	case "feign_death":
		// Ignore DR
		return
	case "backstab":
		match.getPlayerSum(source).BackStabs++
	case "headshot":
		match.getPlayerSum(source).HeadShots++
	case "airshot":
		match.getPlayerSum(source).Airshots++
	}
	match.getPlayerSum(source).Kills++
	match.getPlayerSum(target).Deaths++
}

func (match *Match) drop(source steamid.SID64, team logparse.Team) {
	match.getMedicSum(source).Drops++
	match.getTeamSum(team).Drops++
}

func (match *Match) medicDeath(source steamid.SID64, uberPct int) {
	if uberPct > 95 && uberPct < 100 {
		match.getMedicSum(source).NearFullChargeDeath++
	}
}

func (match *Match) medicCharge(source steamid.SID64, weapon logparse.Medigun, team logparse.Team) {
	medicSum := match.getMedicSum(source)
	_, found := medicSum.Charges[weapon]
	if !found {
		medicSum.Charges[weapon] = 0
	}
	medicSum.Charges[weapon]++
	match.getTeamSum(team).Charges++
}

func (match *Match) medicLostAdv(source steamid.SID64, timeAdv int) {
	medicSum := match.getMedicSum(source)
	if timeAdv > 30 {
		// TODO check what is actually the time to trigger
		medicSum.MajorAdvLost++
	}
	if timeAdv > medicSum.BiggestAdvLost {
		medicSum.BiggestAdvLost = timeAdv
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
	//timeEnd           *time.Time
}

func (playerSum *MatchPlayerSum) touch() {
	if playerSum.timeStart == nil {
		t := config.Now()
		playerSum.timeStart = &t
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

func (classSum *MatchClassSums) Sum() int {
	return classSum.Scout + classSum.Soldier + classSum.Pyro +
		classSum.Demoman + classSum.Heavy + classSum.Engineer +
		classSum.Medic + classSum.Spy + classSum.Sniper
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
