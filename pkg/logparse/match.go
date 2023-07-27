package logparse

import (
	"fmt"
	"sort"
	"time"

	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	ErrIgnored     = errors.New("Ignored msg")
	ErrUnhandled   = errors.New("Unhandled msg")
	ErrInvalidType = errors.New("Invalid Type")
)

func NewMatch(logger *zap.Logger, serverID int, serverName string) Match {
	return Match{
		logger:            logger.Named(fmt.Sprintf("match-%d", serverID)),
		ServerID:          serverID,
		Title:             serverName,
		PlayerSums:        MatchPlayerSums{},
		MedicSums:         []*MatchMedicSum{},
		TeamSums:          []*MatchTeamSum{},
		ClassKills:        MatchPlayerClassSums{},
		ClassKillsAssists: MatchPlayerClassSums{},
		ClassDeaths:       MatchPlayerClassSums{},
		CreatedOn:         time.Now(),
		curRound:          -1,
	}
}

type (
	MatchRoundSums []*MatchRoundSum
	MatchTeamSums  []*MatchTeamSum
)

func (mps MatchTeamSums) GetByTeam(team Team) (*MatchTeamSum, error) {
	for _, m := range mps {
		if m.Team == team {
			return m, nil
		}
	}

	return nil, consts.ErrInvalidTeam
}

type MatchMedicSums []*MatchMedicSum

func (mps MatchMedicSums) GetBySteamID(steamID steamid.SID64) (*MatchMedicSum, error) {
	for _, m := range mps {
		if m.SteamID == steamID {
			return m, nil
		}
	}

	return nil, consts.ErrInvalidSID
}

type MatchPlayerSums []*MatchPlayerSum

func (mps MatchPlayerSums) GetBySteamID(steamID steamid.SID64) (*MatchPlayerSum, error) {
	for _, m := range mps {
		if m.SteamID == steamID {
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
	ServerID          int
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

	// inMatch is set to true when we start a round, many stat events are ignored until this is true
	inMatch    bool // We ignore most events until Round_Start event
	inRound    bool
	useRealDmg bool
	curRound   int
}

type MatchChat struct {
	SteamID   steamid.SID64
	Name      string
	Message   string
	Team      bool
	CreatedAt time.Time
}

type MatchWeaponSum struct {
	Weapon    Weapon
	MatchID   int
	SteamID   steamid.SID64
	Kills     int
	Deaths    int
	Damage    int64
	Shots     int64
	Hits      int64
	Airshots  int
	Headshots int
	BackStabs int
}

func NewMatchWeaponSum(steamID steamid.SID64, weapon Weapon) MatchWeaponSum {
	return MatchWeaponSum{SteamID: steamID, Weapon: weapon}
}

type MatchWeaponSums []*MatchWeaponSum

func (match *Match) GetWeaponSum(steamID steamid.SID64, weapon Weapon) *MatchWeaponSum {
	player := match.getPlayer(steamID)
	for _, existingWeapon := range player.Weapons {
		if existingWeapon.Weapon == weapon {
			return existingWeapon
		}
	}

	newWeapon := NewMatchWeaponSum(steamID, weapon)

	player.Weapons = append(player.Weapons, &newWeapon)

	return &newWeapon
}

type MatchSummary struct {
	MatchID     int       `json:"match_id"`
	ServerID    int       `json:"server_id"`
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
	players := make([]MatchPlayerSum, len(match.PlayerSums))
	for index, p := range match.PlayerSums {
		players[index] = *p
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
// This is not threadsafe at all.
func (match *Match) Apply(result *Results) error { //nolint:maintidx
	// This first switch is used for events that can happen at any point in time during a game without
	// having effects on things like player stats.
	switch result.EventType {
	case MapLoad:
		return nil
	case SayTeam:
		fallthrough
	case Say:
		evt, ok := result.Event.(SayEvt)
		if !ok {
			return ErrInvalidType
		}

		match.addChat(evt.SID, evt.Name, evt.Msg, evt.Team, evt.CreatedOn)

		return nil
	case JoinedTeam:
		evt, ok := result.Event.(JoinedTeamEvt)
		if !ok {
			return ErrInvalidType
		}

		match.joinTeam(evt.SID, evt.NewTeam)

		return nil
	case IgnoredMsg:
		return ErrIgnored
	case UnknownMsg:
		return ErrUnhandled
	case WRoundStart:
		match.roundStart()

		return nil
	case WGameOver:
		match.gameOver()
	case WMiniRoundStart:
		match.roundStart()
	case WRoundOvertime:
		match.overtime()

		return nil
	case WRoundLen:
	case WRoundWin:
		evt, ok := result.Event.(WRoundWinEvt)
		if !ok {
			return ErrInvalidType
		}

		match.roundWin(evt.Winner)

		return nil

	case Connected:
		evt, ok := result.Event.(ConnectedEvt)
		if !ok {
			return ErrInvalidType
		}

		match.connected(evt.SID)

		return nil

	case Entered:
		evt, ok := result.Event.(EnteredEvt)
		if !ok {
			return ErrInvalidType
		}

		match.entered(evt.SID)

		return nil

	case Disconnected:
		evt, ok := result.Event.(DisconnectedEvt)
		if !ok {
			return ErrInvalidType
		}

		match.disconnected(evt.SID)

		return nil
	}

	if !match.inMatch || !match.inRound {
		return nil
	}
	// These remaining events deal with handling the actual player stats during live rounds.
	switch result.EventType {
	case PointCaptured:
		evt, ok := result.Event.(PointCapturedEvt)
		if !ok {
			return ErrInvalidType
		}

		match.pointCapture(evt.Team, evt.CP, evt.Cpname, evt.Players())

	case CaptureBlocked:
		evt, ok := result.Event.(CaptureBlockedEvt)
		if !ok {
			return ErrInvalidType
		}

		match.pointCaptureBlocked(evt.CP, evt.Cpname, SourcePlayerPosition{
			SourcePlayer: evt.SourcePlayer,
			Pos:          evt.Position,
		})

	case SpawnedAs:
		evt, ok := result.Event.(SpawnedAsEvt)
		if !ok {
			return ErrInvalidType
		}

		match.addClass(evt.SID, evt.Class)
	case ChangeClass:
		evt, ok := result.Event.(ChangeClassEvt)
		if !ok {
			return ErrInvalidType
		}

		match.addClass(evt.SID, evt.Class)

	case ShotFired:
		evt, ok := result.Event.(ShotFiredEvt)
		if !ok {
			return ErrInvalidType
		}

		match.shotFired(evt.SID)

	case ShotHit:
		evt, ok := result.Event.(ShotHitEvt)
		if !ok {
			return ErrInvalidType
		}

		match.shotHit(evt.SID)

	case MedicDeath:
		evt, ok := result.Event.(MedicDeathEvt)
		if !ok {
			return ErrInvalidType
		}

		if evt.Ubercharge {
			// TODO record source player stat
			match.drop(evt.SID2, evt.Team)
		}

	case EmptyUber:
		_, _ = result.Event.(EmptyUberEvt)
	case ChargeDeployed:
		evt, ok := result.Event.(ChargeDeployedEvt)
		if !ok {
			return ErrInvalidType
		}

		match.medicCharge(evt.SID, evt.Medigun, evt.Team)
	case ChargeEnded:
		_, _ = result.Event.(ChargeEndedEvt)

	case ChargeReady:
		_, _ = result.Event.(ChargeReadyEvt)

	case LostUberAdv:
		evt, ok := result.Event.(LostUberAdvantageEvt)
		if !ok {
			return ErrInvalidType
		}

		match.medicLostAdv(evt.SID, evt.Time)
	case MedicDeathEx:
		evt, ok := result.Event.(MedicDeathExEvt)
		if !ok {
			return ErrInvalidType
		}

		match.medicDeath(evt.SID, evt.Uberpct)

	case Domination:
		evt, ok := result.Event.(DominationEvt)
		if !ok {
			return ErrInvalidType
		}

		match.domination(evt.SID, evt.SID2)

	case Revenge:
		evt, ok := result.Event.(RevengeEvt)
		if !ok {
			return ErrInvalidType
		}

		match.revenge(evt.SID)

	case Damage:
		evt, ok := result.Event.(DamageEvt)
		if !ok {
			return ErrInvalidType
		}

		if match.useRealDmg {
			match.damage(evt.SID, evt.SID2, evt.Realdamage, evt.Team, evt.Airshot)
		} else {
			match.damage(evt.SID, evt.SID2, evt.Damage, evt.Team, evt.Airshot)
		}

	case Suicide:
		evt, ok := result.Event.(SuicideEvt)
		if !ok {
			return ErrInvalidType
		}

		match.suicide(evt.SID, evt.Weapon)

	case Killed:
		evt, ok := result.Event.(KilledEvt)
		if !ok {
			return ErrInvalidType
		}

		match.killed(evt.SID, evt.SID2, evt.Team)

	case KilledCustom:
		evt, ok := result.Event.(CustomKilledEvt)
		if !ok {
			return ErrInvalidType
		}

		match.killedCustom(evt.SID, evt.SID2, evt.Customkill)

	case KillAssist:
		evt, ok := result.Event.(KillAssistEvt)
		if !ok {
			return ErrInvalidType
		}

		match.assist(evt.SID)

	case Healed:
		evt, ok := result.Event.(HealedEvt)
		if !ok {
			return ErrInvalidType
		}

		match.healed(evt.SID, evt.SID2, evt.Healing)

	case Extinguished:
		evt, ok := result.Event.(ExtinguishedEvt)
		if !ok {
			return ErrInvalidType
		}

		match.extinguishes(evt.SID)

	case BuiltObject:
		evt, ok := result.Event.(BuiltObjectEvt)
		if !ok {
			return ErrInvalidType
		}

		match.builtObject(evt.SID, evt.Object)

	case KilledObject:
		evt, ok := result.Event.(KilledObjectEvt)
		if !ok {
			return ErrInvalidType
		}

		match.killedObject(evt.SID, evt.Object)

	case CarryObject:
		evt, ok := result.Event.(CarryObjectEvt)
		if !ok {
			return ErrInvalidType
		}

		match.carriedObject(evt.SID, evt.Object)

	case DetonatedObject:
		evt, ok := result.Event.(DetonatedObjectEvt)
		if !ok {
			return ErrInvalidType
		}

		match.detonatedObject(evt.SID, evt.Object)

	case DropObject:
		evt, ok := result.Event.(DropObjectEvt)
		if !ok {
			return ErrInvalidType
		}

		match.dropObject(evt.SID, evt.Object)

	case Pickup:
		evt, ok := result.Event.(PickupEvt)
		if !ok {
			return ErrInvalidType
		}

		match.pickup(evt.SID, evt.Item, evt.Healing)

	case FirstHealAfterSpawn:
		evt, ok := result.Event.(FirstHealAfterSpawnEvt)
		if !ok {
			return ErrInvalidType
		}

		match.firstHealAfterSpawn(evt.SID, evt.Time)

	default:
		return errors.New("Unhandled apply event")
	}

	return nil
}

func (match *Match) getPlayer(sid steamid.SID64) *MatchPlayerSum {
	playerSum, err := match.PlayerSums.GetBySteamID(sid)
	if err != nil {
		if errors.Is(err, consts.ErrUnknownID) {
			t0 := time.Now()
			newPs := &MatchPlayerSum{
				SteamID:   sid,
				TimeStart: &t0,
			}

			if match.inMatch {
				// Account for people who joined after Round_start event
				newPs.touch()
			}

			match.PlayerSums = append(match.PlayerSums, newPs)

			return newPs
		}
	}

	return playerSum
}

func (match *Match) getMedicSum(sid steamid.SID64) *MatchMedicSum {
	m, _ := match.MedicSums.GetBySteamID(sid)
	if m != nil {
		return m
	}

	medicSum := &MatchMedicSum{
		SteamID: sid,
		Charges: map[MedigunType]int{
			Uber:       0,
			Kritzkrieg: 0,
			Vaccinator: 0,
			QuickFix:   0,
		},
	}

	match.MedicSums = append(match.MedicSums, medicSum)

	return medicSum
}

func (match *Match) getTeamSum(team Team) *MatchTeamSum {
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
			*playerSum.TimeStart = time.Now()
		}
	}
}

func (match *Match) roundWin(team Team) {
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

func (match *Match) addClass(sid steamid.SID64, class PlayerClass) {
	if class == Spectator {
		return
	}

	playerSum := match.getPlayer(sid)

	if !fp.Contains[PlayerClass](playerSum.Classes, class) {
		playerSum.Classes = append(playerSum.Classes, class)

		if class == Medic {
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

func (match *Match) joinTeam(_ steamid.SID64, _ Team) {
	// TODO join a team
}

func (match *Match) addChat(sid steamid.SID64, name string, message string, team bool, created time.Time) {
	match.Chat = append(match.Chat, MatchChat{
		SteamID:   sid,
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

func (match *Match) builtObject(source steamid.SID64, _ string) {
	match.getPlayer(source).BuildingBuilt++
}

func (match *Match) killedObject(source steamid.SID64, _ string) {
	match.getPlayer(source).BuildingDestroyed++
}

func (match *Match) dropObject(source steamid.SID64, _ string) {
	match.getPlayer(source).BuildingDropped++
}

func (match *Match) carriedObject(source steamid.SID64, _ string) {
	match.getPlayer(source).BuildingCarried++
}

func (match *Match) detonatedObject(source steamid.SID64, _ string) {
	match.getPlayer(source).BuildingDetonated++
}

func (match *Match) extinguishes(source steamid.SID64) {
	match.getPlayer(source).Extinguishes++
}

func (match *Match) damage(source steamid.SID64, target steamid.SID64, damage int64, team Team, airShot bool) {
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
}

func (match *Match) pointCaptureBlocked(cp int, cpName string, pp SourcePlayerPosition) {
	player := match.getPlayer(pp.SID)
	player.CapturesBlocked = append(player.CapturesBlocked, PointCaptureBlocked{
		CP:       cp,
		CPName:   cpName,
		Position: pp.Pos,
	})
}

func (match *Match) pointCapture(team Team, cp int, cpName string, players []SourcePlayerPosition) {
	for _, p := range players {
		match.getPlayer(p.SID).Captures = append(match.getPlayer(p.SID).Captures, PointCapture{
			SteamID:  p.SID,
			CP:       cp,
			CPName:   cpName,
			Position: p.Pos,
		})
	}
	match.getTeamSum(team).Caps++
}

// func (match *Match) midFight(team logparse.Team) {
//	match.getTeamSum(team).MidFights++
//}

func (match *Match) killed(source steamid.SID64, target steamid.SID64, team Team) {
	if match.inRound {
		match.getPlayer(source).Kills++
		match.getPlayer(target).Deaths++
		match.getTeamSum(team).Kills++

		if team == BLU {
			match.getRound().KillsBlu++
		} else if team == RED {
			match.getRound().KillsRed++
		}
	}
}

func (match *Match) suicide(source steamid.SID64, _ Weapon) {
	match.getPlayer(source).Suicides++
}

func (match *Match) firstHealAfterSpawn(source steamid.SID64, timeUntil float64) {
	match.getMedicSum(source).FirstHealAfterSpawn = append(match.getMedicSum(source).FirstHealAfterSpawn, timeUntil)
}

func (match *Match) pickup(source steamid.SID64, item PickupItem, healing int64) {
	switch item { //nolint:exhaustive
	case ItemHPSmall:
		fallthrough
	case ItemHPMedium:
		fallthrough
	case ItemHPLarge:
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

func (match *Match) drop(source steamid.SID64, team Team) {
	match.getMedicSum(source).Drops++
	match.getTeamSum(team).Drops++
}

func (match *Match) medicDeath(source steamid.SID64, uberPct int) {
	if uberPct > 95 && uberPct < 100 {
		match.getMedicSum(source).NearFullChargeDeath++
	}
}

func (match *Match) medicCharge(source steamid.SID64, weapon MedigunType, team Team) {
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
	Position Pos
}

type PointCapture struct {
	SteamID  steamid.SID64
	CP       int
	CPName   string
	Position Pos
}

type MatchPlayerSum struct {
	MatchPlayerSumID  int
	SteamID           steamid.SID64
	Team              Team
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
	Classes           []PlayerClass
	Weapons           MatchWeaponSums
}

func (playerSum *MatchPlayerSum) touch() {
	if playerSum.TimeStart == nil {
		t := time.Now()
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
	RoundWinner Team
	MidFight    Team
}

type MatchMedicSum struct {
	MatchMedicID        int
	MatchID             int
	FirstHealAfterSpawn []float64
	SteamID             steamid.SID64
	Healing             int64
	Charges             map[MedigunType]int
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

func newMatchMedicSum(steamID steamid.SID64) *MatchMedicSum {
	return &MatchMedicSum{
		SteamID: steamID,
		Charges: map[MedigunType]int{
			Uber:       0,
			Kritzkrieg: 0,
			Vaccinator: 0,
			QuickFix:   0,
		},
	}
}

type MatchClassSums struct {
	SteamID  steamid.SID64
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
	MatchTeamID int
	MatchID     int
	Team        Team
	Kills       int
	Damage      int64
	Charges     int
	Drops       int
	Caps        int
	MidFights   int
}

func newMatchTeamSum(team Team) *MatchTeamSum {
	return &MatchTeamSum{
		Team: team,
	}
}
