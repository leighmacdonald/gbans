package model

import (
	"fmt"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"sort"
	"strings"
	"time"
)

var ErrIgnored = errors.New("Ignored msg")
var ErrUnhandled = errors.New("Unhandled msg")

func NewMatch() Match {
	return Match{
		MatchID:           0,
		ServerId:          0,
		Title:             "Team Fortress 3",
		MapName:           "pl_you_didnt_set_this",
		PlayerSums:        MatchPlayerSums{},
		MedicSums:         []*MatchMedicSum{},
		TeamSums:          []*MatchTeamSum{},
		Rounds:            nil,
		ClassKills:        MatchPlayerClassSums{},
		ClassKillsAssists: MatchPlayerClassSums{},
		ClassDeaths:       MatchPlayerClassSums{},
		inMatch:           false,
		CreatedOn:         config.Now(),
		curRound:          -1,
		inRound:           false,
		useRealDmg:        false,
	}
}

type MatchRoundSums []*MatchRoundSum
type MatchTeamSums []*MatchTeamSum

func (mps MatchTeamSums) GetByTeam(team logparse.Team) (*MatchTeamSum, error) {
	for _, m := range mps {
		if m.Team == team {
			return m, nil
		}
	}
	return nil, consts.ErrInvalidTeam
}

type MatchMedicSums []*MatchMedicSum

func (mps MatchMedicSums) GetBySteamId(steamId steamid.SID64) (*MatchMedicSum, error) {
	for _, m := range mps {
		if m.SteamId == steamId {
			return m, nil
		}
	}
	return nil, consts.ErrInvalidSID
}

type MatchPlayerSums []*MatchPlayerSum

func (mps MatchPlayerSums) GetBySteamId(steamId steamid.SID64) (*MatchPlayerSum, error) {
	for _, m := range mps {
		if m.SteamId == steamId {
			return m, nil
		}
	}

	return nil, consts.ErrUnknownID
}

// Match and its related Match* structs are designed as a close to 1:1 mirror of the
// logs.tf ui
//
// For a simple example of usage, see internal/cmd/stats.go
//
// TODO
// - individual game state cache to track who is on winning team
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
	MatchID           int
	ServerId          int
	Title             string
	MapName           string
	PlayerSums        MatchPlayerSums
	MedicSums         MatchMedicSums
	TeamSums          MatchTeamSums
	Rounds            MatchRoundSums
	ClassKills        MatchPlayerClassSums
	ClassKillsAssists MatchPlayerClassSums
	ClassDeaths       MatchPlayerClassSums
	Chat              []PersonChat
	CreatedOn         time.Time
	Players           People
	// inMatch is set to true when we start a round, many stat events are ignored until this is true
	inMatch    bool // We ignore most events until Round_Start event
	inRound    bool
	useRealDmg bool
	curRound   int
}

type MatchWeaponSum struct {
	Weapon    logparse.Weapon
	MatchId   int
	SteamId   steamid.SID64
	Kills     int
	Deaths    int
	Damage    int64
	Shots     int64
	Hits      int64
	Airshots  int
	Headshots int
	Backstabs int
}

func NewMatchWeaponSum(steamId steamid.SID64, weapon logparse.Weapon) MatchWeaponSum {
	return MatchWeaponSum{SteamId: steamId, Weapon: weapon}
}

type MatchWeaponSums []*MatchWeaponSum

func (match *Match) GetWeaponSum(steamId steamid.SID64, weapon logparse.Weapon) *MatchWeaponSum {
	p := match.getPlayer(steamId)
	for _, existingWeapon := range p.Weapons {
		if existingWeapon.Weapon == weapon {
			return existingWeapon
		}
	}
	newWeapon := NewMatchWeaponSum(steamId, weapon)
	p.Weapons = append(p.Weapons, &newWeapon)
	return &newWeapon
}

type MatchSummary struct {
	MatchID     int       `json:"match_id"`
	ServerId    int       `json:"server_id"`
	MapName     string    `json:"map_name"`
	CreatedOn   time.Time `json:"created_on"`
	PlayerCount int       `json:"player_count"`
	Kills       int       `json:"kills"`
	Assists     int       `json:"assists"`
	Damage      int       `json:"damage"`
	Healing     int       `json:"healing"`
	Airshots    int       `json:"airshots"`
}

type MatchSummaryCollection []*MatchSummary

func (match *Match) playerSlice() []MatchPlayerSum {
	var players []MatchPlayerSum
	for _, p := range match.PlayerSums {
		players = append(players, *p)
	}
	return players
}

func (match *Match) TopPlayers() []MatchPlayerSum {
	players := match.playerSlice()
	sort.SliceStable(players, func(i, j int) bool {
		return players[i].Kills > players[j].Kills
	})
	return players
}

// Apply is used to apply incoming event changes to the current match state
// This is not threadsafe at all
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
	case logparse.Connected:

	}

	if !match.inMatch || !match.inRound {
		return nil
	}
	switch event.EventType {
	case logparse.PointCaptured:
		players := steamid.Collection{}
		count := event.GetValueInt("numcappers")
		for i := 0; i < count; i++ {
			sidVal := event.GetValueString(fmt.Sprintf("player%d", i+1))
			pcs := strings.Split(strings.ReplaceAll(sidVal, "><", " "), " ")
			if len(pcs) != 3 {
				continue
			}
			val := steamid.SID3ToSID64(steamid.SID3(pcs[1]))
			if val.Valid() {
				players = append(players, val)
			} else {
				log.Warnf("Failed to parse player")
			}
		}
		match.pointCapture(event.Team, players)
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
		airShot := false
		val, asF := event.MetaData["airshot"]
		if asF && val == "1" {
			airShot = true
		}
		// It's a pub, so why not count over-kill dmg
		if match.useRealDmg {
			match.damage(event.Source.SteamID, event.Target.SteamID, event.RealDamage, event.Team, airShot)
		} else {
			match.damage(event.Source.SteamID, event.Target.SteamID, event.Damage, event.Team, airShot)
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

func (match *Match) getPlayer(sid steamid.SID64) *MatchPlayerSum {
	if !sid.Valid() {
		log.Fatalf("err")
	}
	m, err := match.PlayerSums.GetBySteamId(sid)
	if err != nil {
		if errors.Is(err, consts.ErrUnknownID) {
			t0 := config.Now()
			ps := &MatchPlayerSum{
				SteamId:   sid,
				TimeStart: &t0,
			}
			if match.inMatch {
				// Account for people who joined after Round_start event
				ps.touch()
			}
			match.PlayerSums = append(match.PlayerSums, ps)
			return ps
		}
	}
	return m
}

func (match *Match) getMedicSum(sid steamid.SID64) *MatchMedicSum {
	m, _ := match.MedicSums.GetBySteamId(sid)
	if m != nil {
		return m
	}
	ms := &MatchMedicSum{
		SteamId: sid,
		Charges: map[logparse.Medigun]int{
			logparse.Uber:       0,
			logparse.Kritzkrieg: 0,
			logparse.Vaccinator: 0,
			logparse.QuickFix:   0,
		},
	}
	match.MedicSums = append(match.MedicSums, ms)
	return ms
}

func (match *Match) getTeamSum(team logparse.Team) *MatchTeamSum {
	m, _ := match.TeamSums.GetByTeam(team)
	if m != nil {
		return m
	}
	ts := newMatchTeamSum(team)
	match.TeamSums = append(match.TeamSums, ts)
	return ts
}

func (match *Match) getRound() *MatchRoundSum {
	return match.Rounds[match.curRound]
}

func (match *Match) roundStart() {
	match.inMatch = true
	match.inRound = true
	match.curRound++
	match.Rounds = append(match.Rounds, &MatchRoundSum{
		Length:    0,
		Score:     TeamScores{},
		KillsBlu:  0,
		KillsRed:  0,
		UbersBlu:  0,
		UbersRed:  0,
		DamageBlu: 0,
		DamageRed: 0,
		MidFight:  0,
	})
	for _, playerSum := range match.PlayerSums {
		if playerSum.TimeStart == nil {
			*playerSum.TimeStart = config.Now()
		}
	}
}

func (match *Match) roundWin(team logparse.Team) {
	match.getRound().RoundWinner = team
	match.inMatch = true
	match.inRound = false
}

func (match *Match) gameOver() {
	match.inMatch = false
	match.inRound = false
}

func (match *Match) addClass(sid steamid.SID64, class logparse.PlayerClass) {
	if class == logparse.Spectator {
		return
	}
	playerSum := match.getPlayer(sid)
	if !fp.Contains[logparse.PlayerClass](playerSum.Classes, class) {
		playerSum.Classes = append(playerSum.Classes, class)
		if class == logparse.Medic {
			// Allocate for a new medic
			match.MedicSums = append(match.MedicSums, newMatchMedicSum(sid))
		}
	}
	if match.inMatch {
		playerSum.touch()
	}
}

func (match *Match) shotFired(sid steamid.SID64) {
	match.getPlayer(sid).Shots++
}

func (match *Match) shotHit(sid steamid.SID64) {
	match.getPlayer(sid).Hits++
}

func (match *Match) assist(sid steamid.SID64) {
	match.getPlayer(sid).Assists++
}

func (match *Match) domination(source steamid.SID64, target steamid.SID64) {
	match.getPlayer(source).Dominations++
	match.getPlayer(target).Dominated++
}

func (match *Match) revenge(source steamid.SID64) {
	match.getPlayer(source).Revenges++
}

func (match *Match) builtObject(source steamid.SID64) {
	match.getPlayer(source).BuildingBuilt++
}

func (match *Match) killedObject(source steamid.SID64) {
	match.getPlayer(source).BuildingDestroyed++
}

func (match *Match) extinguishes(source steamid.SID64) {
	match.getPlayer(source).Extinguishes++
}

func (match *Match) damage(source steamid.SID64, target steamid.SID64, damage int64, team logparse.Team, airshot bool) {
	match.getPlayer(source).Damage += damage
	if airshot {
		match.getPlayer(source).Airshots++
	}
	match.getPlayer(target).DamageTaken += damage
	match.getTeamSum(team).Damage += damage
}

func (match *Match) healed(source steamid.SID64, target steamid.SID64, amount int64) {
	match.getPlayer(source).Healing += amount
	match.getPlayer(target).HealingTaken += amount
	match.getMedicSum(source).Healing += amount
}

func (match *Match) pointCapture(team logparse.Team, sources steamid.Collection) {
	for _, sid := range sources {
		match.getPlayer(sid).Captures++
	}
	match.getTeamSum(team).Caps++
}

//func (match *Match) midFight(team logparse.Team) {
//	match.getTeamSum(team).MidFights++
//}

func (match *Match) killed(source steamid.SID64, target steamid.SID64, team logparse.Team) {
	if match.inRound {
		match.getPlayer(source).Kills++
		match.getPlayer(target).Deaths++
		match.getTeamSum(team).Kills++
		if team == logparse.BLU {
			match.getRound().KillsBlu++
		} else if team == logparse.RED {
			match.getRound().KillsRed++
		}
	}
}

func (match *Match) pickup(source steamid.SID64, item logparse.PickupItem, healing int64) {
	switch item {
	case logparse.ItemHPSmall:
		fallthrough
	case logparse.ItemHPMedium:
		fallthrough
	case logparse.ItemHPLarge:
		p := match.getPlayer(source)
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
		match.getPlayer(source).BackStabs++
	case "headshot":
		match.getPlayer(source).HeadShots++
	default:
		log.Errorf("Custom kill type unknown: %s", custom)
	}
	match.getPlayer(source).Kills++
	match.getPlayer(target).Deaths++
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
	MatchPlayerSumID  int
	SteamId           steamid.SID64
	Team              logparse.Team
	TimeStart         *time.Time
	TimeEnd           *time.Time
	Kills             int
	Assists           int
	Deaths            int
	KDRatio           float32
	KADRatio          float32
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
	Weapons           MatchWeaponSums
}

func (playerSum *MatchPlayerSum) touch() {
	if playerSum.TimeStart == nil {
		t := config.Now()
		playerSum.TimeStart = &t
	}
}

type TeamScores struct {
	Red int
	Blu int
}

type MatchRoundSum struct {
	Length      time.Duration
	Score       TeamScores
	KillsBlu    int
	KillsRed    int
	UbersBlu    int
	UbersRed    int
	DamageBlu   int
	DamageRed   int
	RoundWinner logparse.Team
	MidFight    logparse.Team
}

type MatchMedicSum struct {
	MatchMedicId        int
	MatchId             int
	SteamId             steamid.SID64
	Healing             int64
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

func newMatchMedicSum(steamId steamid.SID64) *MatchMedicSum {
	return &MatchMedicSum{
		SteamId: steamId,
		Charges: map[logparse.Medigun]int{
			logparse.Uber:       0,
			logparse.Kritzkrieg: 0,
			logparse.Vaccinator: 0,
			logparse.QuickFix:   0,
		},
	}
}

type MatchClassSums struct {
	SteamId  steamid.SID64
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

type MatchPlayerClassSums []*MatchClassSums

type MatchTeamSum struct {
	MatchTeamId int
	MatchId     int
	Team        logparse.Team
	Kills       int
	Damage      int64
	Charges     int
	Drops       int
	Caps        int
	MidFights   int
}

func newMatchTeamSum(team logparse.Team) *MatchTeamSum {
	return &MatchTeamSum{
		Team: team,
	}
}
