package model

import (
	"fmt"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"sort"
	"time"
)

var ErrIgnored = errors.New("Ignored msg")
var ErrUnhandled = errors.New("Unhandled msg")

func NewMatch(logger *zap.Logger, serverId int, serverName string) Match {
	return Match{
		logger:            logger.Named(fmt.Sprintf("match-%d", serverId)),
		ServerId:          serverId,
		Title:             serverName,
		PlayerSums:        MatchPlayerSums{},
		MedicSums:         []*MatchMedicSum{},
		TeamSums:          []*MatchTeamSum{},
		ClassKills:        MatchPlayerClassSums{},
		ClassKillsAssists: MatchPlayerClassSums{},
		ClassDeaths:       MatchPlayerClassSums{},
		CreatedOn:         config.Now(),
		curRound:          -1,
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
//   - individual game state cache to track who is on winning team
//   - Track current player session
//   - Track player playtime per class
//   - Track server playtime per class
//   - Track global playtime per class
//   - Track player midfights won
//   - Track player biggest killstreaks (min 18 players in server)
//   - Track server biggest killstreaks (min 18 players in server)
//   - Track global biggest killstreaks (min 18 players in server)
//   - Track player classes killed
//   - Track player classes killedBy
//   - Track server classes killed
//   - Track server classes killedBy
//   - Track global classes killed
//   - Track global classes killedBy
//   - Calculate player points
//   - Calculate server points
//   - Calculate global points
//   - Track player weapon stats
//   - Track server weapon stats
//   - Track global weapon stats
//   - calc HealsTaken (live round time only)
//   - calc Heals/min (live round time only)
//   - calc Dmg/min (live round time only)
//   - calc DmgTaken/min (live round time only)
//   - Count headshots
//   - Count airshots
//   - Count headshots
//   - Track current map to get correct map stats. Tracking the sm_nextmap cvar may partially work for old data.
//     Update sourcemod plugin to send log event with the current map.
//   - Simplify implementation of the maps with generics
//   - Track players taking packs when they are close to 100% hp
type Match struct {
	logger            *zap.Logger
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
	Chat              []MatchChat
	CreatedOn         time.Time
	Players           People
	// inMatch is set to true when we start a round, many stat events are ignored until this is true
	inMatch    bool // We ignore most events until Round_Start event
	inRound    bool
	useRealDmg bool
	curRound   int
}

type MatchChat struct {
	SteamId   steamid.SID64
	Name      string
	Message   string
	Team      bool
	CreatedAt time.Time
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
	BackStabs int
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
func (match *Match) Apply(result *logparse.Results) error {
	// This first switch is used for events that can happen at any point in time during a game without
	// having effects on things like player stats.
	switch result.EventType {
	case logparse.MapLoad:
		return nil
	case logparse.Say:
		evt := result.Event.(logparse.SayEvt)
		match.addChat(evt.SID, evt.Name, evt.Msg, false, evt.CreatedOn)
		return nil
	case logparse.SayTeam:
		evt := result.Event.(logparse.SayTeamEvt)
		match.addChat(evt.SID, evt.Name, evt.Msg, true, evt.CreatedOn)
		return nil
	case logparse.JoinedTeam:
		evt := result.Event.(logparse.JoinedTeamEvt)
		match.joinTeam(evt.SID, evt.Team)
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
		match.overtime()
		return nil
	case logparse.WRoundLen:
	case logparse.WRoundWin:
		match.roundWin(result.Event.(logparse.WRoundWinEvt).Winner)
		return nil

	case logparse.Connected:
		evt := result.Event.(logparse.ConnectedEvt)
		match.connected(evt.SID)
		return nil

	case logparse.Entered:
		evt := result.Event.(logparse.EnteredEvt)
		match.entered(evt.SID)
		return nil

	case logparse.Disconnected:
		evt := result.Event.(logparse.DisconnectedEvt)
		match.disconnected(evt.SID)
		return nil
	}

	if !match.inMatch || !match.inRound {
		return nil
	}
	// These remaining events deal with handling the actual player stats during live rounds.
	switch result.EventType {
	case logparse.PointCaptured:
		evt := result.Event.(logparse.PointCapturedEvt)
		match.pointCapture(evt.Team, evt.CP, evt.CPName, evt.Players())

	case logparse.CaptureBlocked:
		evt := result.Event.(logparse.CaptureBlockedEvt)
		match.pointCaptureBlocked(evt.CP, evt.CPName, logparse.SourcePlayerPosition{
			SourcePlayer: evt.SourcePlayer,
			Pos:          evt.Pos,
		})

	case logparse.SpawnedAs:
		evt := result.Event.(logparse.SpawnedAsEvt)
		match.addClass(evt.SID, evt.PlayerClass)
	case logparse.ChangeClass:
		evt := result.Event.(logparse.ChangeClassEvt)
		match.addClass(evt.SID, evt.Class)

	case logparse.ShotFired:
		match.shotFired(result.Event.(logparse.ShotFiredEvt).SID)

	case logparse.ShotHit:
		match.shotHit(result.Event.(logparse.ShotHitEvt).SID)

	case logparse.MedicDeath:
		evt := result.Event.(logparse.MedicDeathEvt)
		if evt.HadUber {
			// TODO record source player stat
			match.drop(evt.SID2, evt.Team)
		}

	case logparse.EmptyUber:
		_ = result.Event.(logparse.EmptyUberEvt)

	case logparse.ChargeDeployed:
		evt := result.Event.(logparse.ChargeDeployedEvt)
		match.medicCharge(evt.SID, evt.Medigun, evt.Team)

	case logparse.ChargeEnded:
		_ = result.Event.(logparse.ChargeEndedEvt)

	case logparse.ChargeReady:
		_ = result.Event.(logparse.ChargeReadyEvt)

	case logparse.LostUberAdv:
		evt := result.Event.(logparse.LostUberAdvantageEvt)
		match.medicLostAdv(evt.SID, evt.AdvTime)

	case logparse.MedicDeathEx:
		evt := result.Event.(logparse.MedicDeathExEvt)
		match.medicDeath(evt.SID, evt.UberPct)

	case logparse.Domination:
		evt := result.Event.(logparse.DominationEvt)
		match.domination(evt.SID, evt.SID2)

	case logparse.Revenge:
		evt := result.Event.(logparse.RevengeEvt)
		match.revenge(evt.SID)

	case logparse.Damage:
		evt := result.Event.(logparse.DamageEvt)
		if match.useRealDmg {
			match.damage(evt.SID, evt.SID2, evt.RealDamage, evt.Team, evt.AirShot)
		} else {
			match.damage(evt.SID, evt.SID2, evt.Damage, evt.Team, evt.AirShot)
		}

	case logparse.Suicide:
		evt := result.Event.(logparse.SuicideEvt)
		match.suicide(evt.SID, evt.Weapon)

	case logparse.Killed:
		evt := result.Event.(logparse.KilledEvt)
		match.killed(evt.SID, evt.SID2, evt.Team)

	case logparse.KilledCustom:
		evt := result.Event.(logparse.CustomKilledEvt)
		match.killedCustom(evt.SID, evt.SID2, evt.CustomKill)

	case logparse.KillAssist:
		evt := result.Event.(logparse.KillAssistEvt)
		match.assist(evt.SID)

	case logparse.Healed:
		evt := result.Event.(logparse.HealedEvt)
		match.healed(evt.SID, evt.SID2, evt.Healing)

	case logparse.Extinguished:
		evt := result.Event.(logparse.ExtinguishedEvt)
		match.extinguishes(evt.SID)

	case logparse.BuiltObject:
		evt := result.Event.(logparse.BuiltObjectEvt)
		match.builtObject(evt.SID, evt.Object)

	case logparse.KilledObject:
		evt := result.Event.(logparse.KilledObjectEvt)
		match.killedObject(evt.SID, evt.Object)

	case logparse.CarryObject:
		evt := result.Event.(logparse.CarryObjectEvt)
		match.carriedObject(evt.SID, evt.Object)

	case logparse.DetonatedObject:
		evt := result.Event.(logparse.DetonatedObjectEvt)
		match.detonatedObject(evt.SID, evt.Object)

	case logparse.DropObject:
		evt := result.Event.(logparse.DropObjectEvt)
		match.dropObject(evt.SID, evt.Object)

	case logparse.Pickup:
		evt := result.Event.(logparse.PickupEvt)
		match.pickup(evt.SID, evt.Item, evt.Healing)

	case logparse.FirstHealAfterSpawn:
		evt := result.Event.(logparse.FirstHealAfterSpawnEvt)
		match.firstHealAfterSpawn(evt.SID, evt.HealTime)

	default:
		return errors.New("Unhandled apply event")
	}

	return nil
}

func (match *Match) getPlayer(sid steamid.SID64) *MatchPlayerSum {
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
		Charges: map[logparse.MedigunType]int{
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
	if match.curRound == -1 {
		return nil
	}
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
	round := match.getRound()
	if round != nil {
		round.RoundWinner = team
	}
	match.inMatch = true
	match.inRound = false
}

func (match *Match) gameOver() {
	match.inMatch = false
	match.inRound = false
}

func (match *Match) overtime() {
	// TODO care about this?
}

func (match *Match) disconnected(_ steamid.SID64) {
	// TODO care about this?
}

func (match *Match) connected(_ steamid.SID64) {
	// TODO care about this?
}

func (match *Match) entered(_ steamid.SID64) {
	// TODO care about this?
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

func (match *Match) joinTeam(sid steamid.SID64, team logparse.Team) {
	// TODO join a team
}

func (match *Match) addChat(sid steamid.SID64, name string, message string, team bool, created time.Time) {
	match.Chat = append(match.Chat, MatchChat{
		SteamId:   sid,
		Name:      name,
		Message:   message,
		Team:      team,
		CreatedAt: created,
	})
}

func (match *Match) domination(source steamid.SID64, target steamid.SID64) {
	match.getPlayer(source).Dominations++
	match.getPlayer(target).Dominated++
}

func (match *Match) revenge(source steamid.SID64) {
	match.getPlayer(source).Revenges++
}

func (match *Match) builtObject(source steamid.SID64, object string) {
	match.getPlayer(source).BuildingBuilt++
}

func (match *Match) killedObject(source steamid.SID64, object string) {
	match.getPlayer(source).BuildingDestroyed++
}

func (match *Match) dropObject(source steamid.SID64, object string) {
	match.getPlayer(source).BuildingDropped++
}

func (match *Match) carriedObject(source steamid.SID64, object string) {
	match.getPlayer(source).BuildingCarried++
}

func (match *Match) detonatedObject(source steamid.SID64, object string) {
	match.getPlayer(source).BuildingDetonated++
}

func (match *Match) extinguishes(source steamid.SID64) {
	match.getPlayer(source).Extinguishes++
}

func (match *Match) damage(source steamid.SID64, target steamid.SID64, damage int64, team logparse.Team, airShot bool) {
	match.getPlayer(source).Damage += damage
	if airShot {
		match.getPlayer(source).AirShots++
	}
	match.getPlayer(target).DamageTaken += damage
	match.getTeamSum(team).Damage += damage
}

func (match *Match) healed(source steamid.SID64, target steamid.SID64, amount int64) {
	match.getPlayer(source).Healing += amount
	match.getPlayer(target).HealingTaken += amount
	//match.getMedicSum(source).Healing += amount
}

func (match *Match) pointCaptureBlocked(cp int, cpname string, pp logparse.SourcePlayerPosition) {
	player := match.getPlayer(pp.SID)
	player.CapturesBlocked = append(player.CapturesBlocked, PointCaptureBlocked{
		CP:       cp,
		CPName:   cpname,
		Position: pp.Pos,
	})
}

func (match *Match) pointCapture(team logparse.Team, cp int, cpname string, players []logparse.SourcePlayerPosition) {
	for _, p := range players {
		match.getPlayer(p.SID).Captures = append(match.getPlayer(p.SID).Captures, PointCapture{
			SteamId:  p.SID,
			CP:       cp,
			CPName:   cpname,
			Position: p.Pos,
		})
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

func (match *Match) suicide(source steamid.SID64, weapon logparse.Weapon) {
	match.getPlayer(source).Suicides++
}

func (match *Match) firstHealAfterSpawn(source steamid.SID64, timeUntil float64) {
	match.getMedicSum(source).FirstHealAfterSpawn = append(match.getMedicSum(source).FirstHealAfterSpawn, timeUntil)
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
		match.logger.Error("Custom kill type unknown", zap.String("type", custom))
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

func (match *Match) medicCharge(source steamid.SID64, weapon logparse.MedigunType, team logparse.Team) {
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

type PointCaptureBlocked struct {
	CP       int
	CPName   string
	Position logparse.Pos
}

type PointCapture struct {
	SteamId  steamid.SID64
	CP       int
	CPName   string
	Position logparse.Pos
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
	Suicides          int
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
	AirShots          int
	Captures          []PointCapture
	CapturesBlocked   []PointCaptureBlocked
	Shots             int
	Hits              int
	Extinguishes      int
	BuildingBuilt     int
	BuildingDetonated int // self-destruct buildings
	BuildingDestroyed int // Opposing team buildings
	BuildingDropped   int // Buildings destroyed while carrying
	BuildingCarried   int // Building pickup count
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
	FirstHealAfterSpawn []float64
	SteamId             steamid.SID64
	Healing             int64
	Charges             map[logparse.MedigunType]int
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
		Charges: map[logparse.MedigunType]int{
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
