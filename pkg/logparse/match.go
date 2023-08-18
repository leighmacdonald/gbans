package logparse

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
)

var (
	ErrIgnored     = errors.New("Ignored msg")
	ErrUnhandled   = errors.New("Unhandled msg")
	ErrInvalidType = errors.New("Invalid Type")
)

type MatchPlayerSums map[steamid.SID64]*PlayerStats

func (mps MatchPlayerSums) GetBySteamID(steamID steamid.SID64) (*PlayerStats, error) {
	for _, m := range mps {
		if m.SteamID == steamID {
			return m, nil
		}
	}

	return nil, consts.ErrUnknownID
}

type MatchChat struct {
	SteamID   steamid.SID64
	Name      string
	Message   string
	Team      bool
	CreatedAt time.Time
}

type WeaponStats struct {
	Kills     int
	Damage    int
	Shots     int
	Hits      int
	Airshots  int
	Headshots int
	BackStabs int
}

func NewWeaponStats() *WeaponStats {
	return &WeaponStats{}
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
//   - Track player weapon stats
//   - Track server weapon stats
//   - Track global weapon stats
//   - Track current map to get correct map stats. Tracking the sm_nextmap cvar may partially work for old data.
//     Update sourcemod plugin to send log event with the current map.
//   - Simplify implementation of the maps with generics
//   - Track players taking packs when they are close to 100% hp
type Match struct {
	MatchID    uuid.UUID        `json:"match_id"`
	ServerID   int              `json:"server_id"`
	Title      string           `json:"title"`
	MapName    string           `json:"map_name"`
	TeamScores TeamScores       `json:"team_scores"`
	PlayerSums MatchPlayerSums  `json:"player_sums"`
	Rounds     []*MatchRoundSum `json:"rounds"`
	Chat       []MatchChat      `json:"chat"`
	CreatedOn  *time.Time       `json:"created_on"`
	TimeEnd    *time.Time       `json:"time_end"`
	// inMatch is set to true when we start a round, many stat events are ignored until this is true
	inMatch  bool // We ignore most events until Round_Start event
	inRound  bool
	curRound int
}

func NewMatch(serverID int, serverName string) Match {
	newID, errID := uuid.NewV4()
	if errID != nil {
		panic(errID)
	}

	return Match{
		MatchID:    newID,
		ServerID:   serverID,
		Title:      serverName,
		PlayerSums: MatchPlayerSums{},
		curRound:   -1,
	}
}

func (match *Match) PlayerBySteamID(sid64 steamid.SID64) *PlayerStats {
	if player, found := match.PlayerSums[sid64]; found {
		return player
	}

	return nil
}

func (match *Match) PlayerCount() int {
	return len(match.PlayerSums)
}

func (match *Match) Winner() Team {
	if match.TeamScores.Red > match.TeamScores.Blu {
		return RED
	} else if match.TeamScores.Blu > match.TeamScores.Red {
		return BLU
	}

	if strings.HasPrefix(match.MapName, "pl_") {
		var (
			winner Team
			length time.Duration
		)

		for _, round := range match.Rounds {
			if round.Length > length {
				length = round.Length
				winner = round.RoundWinner
			}
		}

		return winner
	}

	return UNASSIGNED
}

func (match *Match) ChatCount() int {
	return len(match.Chat)
}

func (match *Match) RoundCount() int {
	return len(match.Rounds)
}

func (match *Match) playerSlice() []*PlayerStats {
	var (
		players = make([]*PlayerStats, len(match.PlayerSums))
		index   int
	)

	for _, p := range match.PlayerSums {
		players[index] = p
		index++
	}

	return players
}

func (match *Match) Healers() []*PlayerStats {
	var healers []*PlayerStats

	for _, player := range match.playerSlice() {
		if player.HealingStats != nil {
			healers = append(healers, player)
		}
	}

	sort.SliceStable(healers, func(i, j int) bool {
		return healers[i].HealingStats.Healing > healers[j].HealingStats.Healing
	})

	return healers
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

		match.joinTeam(evt)

		return nil
	case IgnoredMsg:
		return ErrIgnored
	case UnknownMsg:
		return ErrUnhandled
	case MapStarted:
		evt, ok := result.Event.(MapStartedEvt)
		if !ok {
			return ErrInvalidType
		}

		match.MapName = evt.Map

		return nil

	case WRoundStart:
		match.roundStart()

		return nil
	case WGameOver:
		evt, ok := result.Event.(WGameOverEvt)
		if !ok {
			return ErrInvalidType
		}

		match.gameOver(evt)

		return nil
	case WMiniRoundStart:
		// match.roundStart()
	case WRoundOvertime:
		match.overtime()

		return nil
	case WRoundLen:
		evt, ok := result.Event.(WRoundLenEvt)
		if !ok {
			return ErrInvalidType
		}

		match.roundLen(evt)

		return nil
	case WTeamScore:
		evt, ok := result.Event.(WTeamScoreEvt)
		if !ok {
			return ErrInvalidType
		}

		match.roundScore(evt)

		return nil
	case WTeamFinalScore:
		evt, ok := result.Event.(WTeamFinalScoreEvt)
		if !ok {
			return ErrInvalidType
		}

		match.finalScore(evt)

		return nil
	case WIntermissionWinLimit:
		return nil
	case WRoundWin:
		evt, ok := result.Event.(WRoundWinEvt)
		if !ok {
			return ErrInvalidType
		}

		match.roundWin(evt)

		return nil
	case WMiniRoundWin:
		evt, ok := result.Event.(WMiniRoundWinEvt)
		if !ok {
			return ErrInvalidType
		}

		match.miniRoundWin(evt)

		return nil
	case WMiniRoundLen:
		evt, ok := result.Event.(WMiniRoundLenEvt)
		if !ok {
			return ErrInvalidType
		}

		match.miniRoundLen(evt)

		return nil
	case Connected:
		evt, ok := result.Event.(ConnectedEvt)
		if !ok {
			return ErrInvalidType
		}

		match.connected(evt)

		return nil

	case Entered:
		evt, ok := result.Event.(EnteredEvt)
		if !ok {
			return ErrInvalidType
		}

		match.entered(evt)

		return nil

	case Disconnected:
		evt, ok := result.Event.(DisconnectedEvt)
		if !ok {
			return ErrInvalidType
		}

		match.disconnected(evt)

		return nil
	case PointCaptured:
		evt, ok := result.Event.(PointCapturedEvt)
		if !ok {
			return ErrInvalidType
		}

		match.pointCapture(evt)

	case CaptureBlocked:
		evt, ok := result.Event.(CaptureBlockedEvt)
		if !ok {
			return ErrInvalidType
		}

		match.pointCaptureBlocked(evt)

	case SpawnedAs:
		evt, ok := result.Event.(SpawnedAsEvt)
		if !ok {
			return ErrInvalidType
		}

		match.spawnedAs(evt)

		// If we started a match 1/2 way through, create a start time
		if match.CreatedOn == nil {
			match.CreatedOn = &evt.CreatedOn
		}
	case ChangeClass:
		// Spawned as is the better version of this
		_, ok := result.Event.(ChangeClassEvt)
		if !ok {
			return ErrInvalidType
		}
	case ShotFired:
		evt, ok := result.Event.(ShotFiredEvt)
		if !ok {
			return ErrInvalidType
		}

		match.shotFired(evt)
	case ShotHit:
		evt, ok := result.Event.(ShotHitEvt)
		if !ok {
			return ErrInvalidType
		}

		match.shotHit(evt)

	case MedicDeath:
		evt, ok := result.Event.(MedicDeathEvt)
		if !ok {
			return ErrInvalidType
		}

		if evt.Ubercharge {
			// TODO record source player stat
			match.drop(evt)
		}

	case EmptyUber:
		_, _ = result.Event.(EmptyUberEvt)
	case ChargeDeployed:
		evt, ok := result.Event.(ChargeDeployedEvt)
		if !ok {
			return ErrInvalidType
		}

		match.medicCharge(evt)
	case ChargeEnded:
		evt, ok := result.Event.(ChargeEndedEvt)
		if !ok {
			return ErrInvalidType
		}

		match.medicChargeEnded(evt)
	case ChargeReady:
		_, _ = result.Event.(ChargeReadyEvt)

	case LostUberAdv:
		evt, ok := result.Event.(LostUberAdvantageEvt)
		if !ok {
			return ErrInvalidType
		}

		match.medicLostAdv(evt)
	case MedicDeathEx:
		evt, ok := result.Event.(MedicDeathExEvt)
		if !ok {
			return ErrInvalidType
		}

		match.medicDeath(evt)
	case Domination:
		evt, ok := result.Event.(DominationEvt)
		if !ok {
			return ErrInvalidType
		}

		match.domination(evt)
	case Revenge:
		evt, ok := result.Event.(RevengeEvt)
		if !ok {
			return ErrInvalidType
		}

		match.revenge(evt)
	case Damage:
		evt, ok := result.Event.(DamageEvt)
		if !ok {
			return ErrInvalidType
		}

		match.damage(evt)
	case Suicide:
		evt, ok := result.Event.(SuicideEvt)
		if !ok {
			return ErrInvalidType
		}

		match.suicide(evt)

	case Killed:
		evt, ok := result.Event.(KilledEvt)
		if !ok {
			return ErrInvalidType
		}

		match.killed(evt)

	case KilledCustom:
		evt, ok := result.Event.(CustomKilledEvt)
		if !ok {
			return ErrInvalidType
		}

		if errKill := match.killedCustom(evt); errKill != nil {
			return errKill
		}
	case KillAssist:
		evt, ok := result.Event.(KillAssistEvt)
		if !ok {
			return ErrInvalidType
		}

		match.assist(evt)

	case Healed:
		evt, ok := result.Event.(HealedEvt)
		if !ok {
			return ErrInvalidType
		}

		match.healed(evt)

	case Extinguished:
		evt, ok := result.Event.(ExtinguishedEvt)
		if !ok {
			return ErrInvalidType
		}

		match.extinguishes(evt)

	case BuiltObject:
		evt, ok := result.Event.(BuiltObjectEvt)
		if !ok {
			return ErrInvalidType
		}

		match.builtObject(evt)

	case KilledObject:
		evt, ok := result.Event.(KilledObjectEvt)
		if !ok {
			return ErrInvalidType
		}

		match.killedObject(evt)

	case CarryObject:
		evt, ok := result.Event.(CarryObjectEvt)
		if !ok {
			return ErrInvalidType
		}

		match.carriedObject(evt)

	case DetonatedObject:
		evt, ok := result.Event.(DetonatedObjectEvt)
		if !ok {
			return ErrInvalidType
		}

		match.detonatedObject(evt)

	case DropObject:
		evt, ok := result.Event.(DropObjectEvt)
		if !ok {
			return ErrInvalidType
		}

		match.dropObject(evt)

	case Pickup:
		evt, ok := result.Event.(PickupEvt)
		if !ok {
			return ErrInvalidType
		}

		match.pickup(evt)

	case FirstHealAfterSpawn:
		evt, ok := result.Event.(FirstHealAfterSpawnEvt)
		if !ok {
			return ErrInvalidType
		}

		match.firstHealAfterSpawn(evt)
	case JarateAttack:
		return nil
	case MilkAttack:
		return nil
	case GasAttack:
		return nil
	case Validated:
		return nil
	case ServerConfigExec:
		return nil
	case CVAR:
		return nil
	case RCON:
		return nil
	case WRoundSetupBegin:
		return nil
	case WMiniRoundSelected:
		return nil
	case SteamAuth:
		return nil
	default:
		return errors.New(fmt.Sprintf("Unhandled apply event: %d %v", result.EventType, result.Event))
	}

	return nil
}

func (match *Match) getPlayer(evtTime time.Time, sid steamid.SID64) *PlayerStats {
	if playerSum, found := match.PlayerSums[sid]; found {
		return playerSum
	}

	newPs := newMatchPlayerStats(match, sid)
	newPs.TimeStart = &evtTime

	match.PlayerSums[sid] = newPs

	return newPs
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
		Score: TeamScores{},
	})
}

func (match *Match) roundWin(evt WRoundWinEvt) {
	round := match.getRound()
	if round != nil {
		round.RoundWinner = evt.Winner
	}

	match.inMatch = true
	match.inRound = false
}

func (match *Match) gameOver(evt WGameOverEvt) {
	match.inMatch = false
	match.inRound = false
	match.TimeEnd = &evt.CreatedOn

	for _, player := range match.PlayerSums {
		// Players disconnected before game end should already have this set
		if player.TimeEnd == nil {
			player.TimeEnd = &evt.CreatedOn
		}

		player.onClassChangeOrGameEnd(evt.CreatedOn)
	}
}

func (match *Match) overtime() {
	// TODO care about this?
}

func (match *Match) disconnected(evt DisconnectedEvt) {
	player := match.getPlayer(evt.CreatedOn, evt.SID)
	now := evt.CreatedOn
	player.TimeEnd = &now
	player.onClassChangeOrGameEnd(evt.CreatedOn)
}

func (match *Match) connected(evt ConnectedEvt) {
	match.getPlayer(evt.CreatedOn, evt.SID)
}

func (match *Match) entered(evt EnteredEvt) {
	match.getPlayer(evt.CreatedOn, evt.SID)
}

func (match *Match) spawnedAs(evt SpawnedAsEvt) {
	if evt.Class == Spectator {
		return
	}

	playerSum := match.getPlayer(evt.CreatedOn, evt.SID)

	if playerSum.Name == "" && evt.Name != "" {
		playerSum.Name = evt.Name
	}

	if evt.Class != playerSum.currentClass {
		playerSum.onClassChangeOrGameEnd(evt.CreatedOn)
		playerSum.currentClass = evt.Class
		newStats := playerSum.getClassStats()
		newStats.startTime = evt.CreatedOn
	}

	playerSum.currentLifeStart = evt.CreatedOn
}

func (match *Match) shotFired(evt ShotFiredEvt) {
	match.getPlayer(evt.CreatedOn, evt.SID).getWeaponSum(evt.Weapon).Shots++
}

func (match *Match) shotHit(evt ShotHitEvt) {
	match.getPlayer(evt.CreatedOn, evt.SID).getWeaponSum(evt.Weapon).Hits++
}

func (match *Match) assist(evt KillAssistEvt) {
	player := match.getPlayer(evt.CreatedOn, evt.SID)
	player.Assists++

	if classStats := player.getClassStats(); classStats != nil {
		classStats.Assists++
	}
}

func (match *Match) joinTeam(evt JoinedTeamEvt) {
	match.getPlayer(evt.CreatedOn, evt.SID).Team = evt.NewTeam
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

func (match *Match) domination(evt DominationEvt) {
	player := match.getPlayer(evt.CreatedOn, evt.SID)
	if cs := player.getClassStats(); cs != nil {
		cs.Dominations++
	}

	target := player.getTarget(evt.SID2)
	target.Dominations++

	victim := match.getPlayer(evt.CreatedOn, evt.SID2)
	if cs := victim.getClassStats(); cs != nil {
		cs.Dominated++
	}
}

func (match *Match) revenge(evt RevengeEvt) {
	player := match.getPlayer(evt.CreatedOn, evt.SID2)

	target := player.getTarget(evt.SID2)
	target.Revenges++

	if cs := player.getClassStats(); cs != nil {
		cs.Revenges++
	}
}

func (match *Match) builtObject(evt BuiltObjectEvt) {
	match.getPlayer(evt.CreatedOn, evt.SID).BuildingBuilt++
}

func (match *Match) killedObject(evt KilledObjectEvt) {
	if cs := match.getPlayer(evt.CreatedOn, evt.SID).getClassStats(); cs != nil {
		cs.BuildingsDestroyed++
	}
}

func (match *Match) dropObject(evt DropObjectEvt) {
	match.getPlayer(evt.CreatedOn, evt.SID).BuildingDropped++
}

func (match *Match) carriedObject(evt CarryObjectEvt) {
	match.getPlayer(evt.CreatedOn, evt.SID).BuildingCarried++
}

func (match *Match) detonatedObject(evt DetonatedObjectEvt) {
	match.getPlayer(evt.CreatedOn, evt.SID).BuildingDetonated++
}

func (match *Match) extinguishes(evt ExtinguishedEvt) {
	match.getPlayer(evt.CreatedOn, evt.SID).getTarget(evt.SID2).Extinguishes++
}

func (match *Match) damage(evt DamageEvt) {
	player := match.getPlayer(evt.CreatedOn, evt.SID)
	dmg := fp.FirstNonZero(evt.Realdamage, evt.Damage)

	weaponSum := player.getWeaponSum(evt.Weapon)
	weaponSum.Damage += dmg

	if cs := player.getClassStats(); cs != nil {
		cs.Damage += dmg
	}

	if evt.Airshot {
		weaponSum.Airshots++
	}

	if evt.Headshot {
		weaponSum.Headshots++
	}

	target := player.getTarget(evt.SID2)
	target.DamageTaken += dmg

	victim := match.getPlayer(evt.CreatedOn, evt.SID2)
	if cs := victim.getClassStats(); cs != nil {
		cs.DamageTaken += dmg
	}

	if round := match.getRound(); round != nil {
		if evt.Team == RED {
			round.DamageRed += dmg
		} else if evt.Team == BLU {
			round.DamageBlu += dmg
		}
	}
}

func (match *Match) healed(evt HealedEvt) {
	player := match.getPlayer(evt.CreatedOn, evt.SID)

	medicStats := player.getMedicSum()
	medicStats.Healing += evt.Healing

	player.getTarget(evt.SID2).HealingTaken += evt.Healing

	if cs := match.getPlayer(evt.CreatedOn, evt.SID2).getClassStats(); cs != nil {
		cs.HealingTaken += evt.Healing
	}
}

func (match *Match) pointCaptureBlocked(evt CaptureBlockedEvt) {
	player := match.getPlayer(evt.CreatedOn, evt.SID)
	player.CapturesBlocked = append(player.CapturesBlocked, &PointCaptureBlocked{
		CP:       evt.CP,
		CPName:   evt.Cpname,
		Position: evt.Position,
	})

	if cs := player.getClassStats(); cs != nil {
		cs.CapturesBlocked++
	}
}

func (match *Match) pointCapture(evt PointCapturedEvt) {
	for _, evtPlayer := range evt.Players() {
		player := match.getPlayer(evt.CreatedOn, evtPlayer.SID)
		player.Captures = append(player.Captures, &PointCapture{
			SteamID:  evtPlayer.SID,
			CP:       evt.CP,
			CPName:   evt.Cpname,
			Position: evtPlayer.Pos,
		})

		if cs := player.getClassStats(); cs != nil {
			cs.Captures++
		}
	}
}

// func (match *Match) midFight(team logparse.Team) {
//	match.getTeamSum(team).MidFights++
//}

func (match *Match) killed(evt KilledEvt) {
	if match.inRound {
		player := match.getPlayer(evt.CreatedOn, evt.SID)
		player.addKill(evt.CreatedOn, evt.SID2, evt.Weapon, evt.AttackerPosition, evt.VictimPosition)

		if evt.Team == BLU {
			match.getRound().KillsBlu++
		} else if evt.Team == RED {
			match.getRound().KillsRed++
		}
	}
}

func (match *Match) suicide(evt SuicideEvt) {
	match.getPlayer(evt.CreatedOn, evt.SID).Suicides++
}

func (match *Match) firstHealAfterSpawn(evt FirstHealAfterSpawnEvt) {
	player := match.getPlayer(evt.CreatedOn, evt.SID)
	if player.HealingStats != nil {
		player.HealingStats.FirstHealAfterSpawn = append(player.HealingStats.FirstHealAfterSpawn, evt.Time)
	}
}

func (match *Match) pickup(evt PickupEvt) {
	player := match.getPlayer(evt.CreatedOn, evt.SID)

	_, found := player.Pickups[evt.Item]
	if !found {
		player.Pickups[evt.Item] = 0
	}

	player.Pickups[evt.Item]++
	player.HealingPacks += evt.Healing
}

func (match *Match) killedCustom(evt CustomKilledEvt) error {
	player := match.getPlayer(evt.CreatedOn, evt.SID)
	weaponSum := player.getWeaponSum(evt.Weapon)

	switch evt.Customkill {
	case "feign_death":
		// Ignore DR
		return nil
	case "backstab":
		weaponSum.BackStabs++
	case "headshot":
		// This is taken from damage event instead to match logs.tf
		// weaponSum.Headshots++
	default:
		return errors.Errorf("Custom kill type unknown: %s", evt.Customkill)
	}

	player.addKill(evt.CreatedOn, evt.SID2, evt.Weapon, evt.AttackerPosition, evt.VictimPosition)

	return nil
}

func (match *Match) drop(evt MedicDeathEvt) {
	healingSum := match.getPlayer(evt.CreatedOn, evt.SID2).getMedicSum()
	healingSum.Drops = append(healingSum.Drops, evt.SID)
}

func (match *Match) medicDeath(evt MedicDeathExEvt) {
	if evt.Uberpct > 95 && evt.Uberpct < 100 {
		healingSum := match.getPlayer(evt.CreatedOn, evt.SID).getMedicSum()
		healingSum.NearFullChargeDeath++
	}
}

func (match *Match) medicCharge(evt ChargeDeployedEvt) {
	medicSum := match.getPlayer(evt.CreatedOn, evt.SID).getMedicSum()

	_, found := medicSum.Charges[evt.Medigun]
	if !found {
		medicSum.Charges[evt.Medigun] = 0
	}

	medicSum.Charges[evt.Medigun]++

	round := match.getRound()

	if round != nil {
		amount := 1.0
		if evt.Medigun == Vaccinator {
			// Vacc uber worth 25% of regular
			amount = 0.25
		}

		if evt.Team == RED {
			round.UbersRed += amount
		} else if evt.Team == BLU {
			round.UbersBlu += amount
		}
	}
}

func (match *Match) medicChargeEnded(evt ChargeEndedEvt) {
	medicSum := match.getPlayer(evt.CreatedOn, evt.SID).getMedicSum()

	medicSum.UberDurations = append(medicSum.UberDurations, evt.Duration)
}

func (match *Match) medicLostAdv(evt LostUberAdvantageEvt) {
	medicSum := match.getPlayer(evt.CreatedOn, evt.SID).getMedicSum()

	if evt.Time > 30 {
		// TODO check what is actually the time to trigger
		medicSum.MajorAdvLost++
	}

	if evt.Time > medicSum.BiggestAdvLost {
		medicSum.BiggestAdvLost = evt.Time
	}
}

func (match *Match) roundLen(evt WRoundLenEvt) {
	round := match.getRound()
	if round != nil {
		round.Length = time.Duration(evt.Seconds) * time.Second
	}
}

func (match *Match) roundScore(evt WTeamScoreEvt) {
	round := match.getRound()
	if round != nil {
		if evt.Team == RED {
			round.Score.Red = evt.Score
		} else if evt.Team == BLU {
			round.Score.Blu = evt.Score
		}
	}
}

func (match *Match) finalScore(evt WTeamFinalScoreEvt) {
	if evt.Team == RED {
		match.TeamScores.Red = evt.Score
	} else if evt.Team == BLU {
		match.TeamScores.Blu = evt.Score
	}
}

func (match *Match) miniRoundWin(evt WMiniRoundWinEvt) {
	round := match.getRound()
	if round != nil {
		round.RoundWinner = evt.Team
	}
}

func (match *Match) miniRoundLen(evt WMiniRoundLenEvt) {
	round := match.getRound()
	if round != nil {
		round.Length = time.Second * time.Duration(evt.Seconds)
	}
}

type PointCaptureBlocked struct {
	CP       int    `json:"cp"`
	CPName   string `json:"cp_name"`
	Position Pos    `json:"position"`
}

type PointCapture struct {
	SteamID  steamid.SID64 `json:"steam_id"`
	CP       int           `json:"cp"`
	CPName   string        `json:"cp_name"`
	Position Pos           `json:"position"`
}

type KillInfo struct {
	Weapon    Weapon `json:"weapon"`
	SourcePos Pos    `json:"source_pos"`
	TargetPos Pos    `json:"target_pos"`
}

type TargetStats struct {
	SteamID      steamid.SID64 `json:"steam_id"`
	KilledInfo   []KillInfo    `json:"killed_info"`
	Dominations  int           `json:"dominations"`
	DamageTaken  int           `json:"damage_taken"`
	HealingTaken int           `json:"healing_taken"`
	Revenges     int           `json:"revenges"`
	Extinguishes int           `json:"extinguishes"`
}

type PlayerClassStats struct {
	startTime          time.Time
	Kills              int `json:"kills"`
	Assists            int `json:"assists"`
	Deaths             int `json:"deaths"`
	Playtime           int `json:"playtime"`
	Dominations        int `json:"dominations"`
	Dominated          int `json:"dominated"`
	Revenges           int `json:"revenges"`
	Damage             int `json:"damage"`
	DamageTaken        int `json:"damage_taken"`
	HealingTaken       int `json:"healing_taken"`
	Captures           int `json:"captures"`
	CapturesBlocked    int `json:"captures_blocked"`
	BuildingsDestroyed int `json:"buildings_destroyed"`
}

type PlayerKillstreak struct {
	PlayerClass PlayerClass
	Killstreak  int
	Duration    int
}

type PlayerStats struct {
	MatchPlayerID     int64 `json:"match_player_id"`
	match             *Match
	SteamID           steamid.SID64                     `json:"steam_id"`
	Team              Team                              `json:"team"`
	Name              string                            `json:"name"`
	TimeStart         *time.Time                        `json:"time_start"`
	TimeEnd           *time.Time                        `json:"time_end"`
	TargetInfo        map[steamid.SID64]*TargetStats    `json:"target_info"`
	WeaponInfo        map[Weapon]*WeaponStats           `json:"weapon_info"`
	Assists           int                               `json:"assists"`
	Suicides          int                               `json:"suicides"`
	HealingPacks      int                               `json:"healing_packs"` // Healing from packs
	HealingStats      *HealingStats                     `json:"healing_stats"`
	Pickups           map[PickupItem]int                `json:"pickups"`
	Captures          []*PointCapture                   `json:"captures"`
	CapturesBlocked   []*PointCaptureBlocked            `json:"captures_blocked"`
	BuildingBuilt     int                               `json:"building_built"`
	BuildingDetonated int                               `json:"building_detonated"` // self-destruct buildings
	BuildingDestroyed int                               `json:"building_destroyed"` // Opposing team buildings
	BuildingDropped   int                               `json:"building_dropped"`   // Buildings destroyed while carrying
	BuildingCarried   int                               `json:"building_carried"`   // Building pickup count
	Classes           map[PlayerClass]*PlayerClassStats `json:"classes"`
	KillStreaks       []PlayerKillstreak                `json:"kill_streaks"`
	currentKillStreak int
	currentLifeStart  time.Time
	currentClass      PlayerClass
}

func newMatchPlayerStats(match *Match, sid steamid.SID64) *PlayerStats {
	return &PlayerStats{
		match:      match,
		SteamID:    sid,
		Team:       UNASSIGNED,
		TargetInfo: map[steamid.SID64]*TargetStats{},
		WeaponInfo: map[Weapon]*WeaponStats{},
		Pickups:    map[PickupItem]int{},
		Classes:    map[PlayerClass]*PlayerClassStats{},
	}
}

func (player *PlayerStats) DamageTaken() int {
	var total int

	for _, ps := range player.match.PlayerSums {
		if targetStats, found := ps.TargetInfo[player.SteamID]; found {
			total += targetStats.DamageTaken
		}
	}

	return total
}

func (player *PlayerStats) HealingTaken() int {
	var total int

	for _, healer := range player.match.Healers() {
		if targetStats, found := healer.TargetInfo[player.SteamID]; found {
			total += targetStats.HealingTaken
		}
	}

	return total
}

func (player *PlayerStats) Extinguishes() int {
	var total int

	for _, plr := range player.TargetInfo {
		total += plr.Extinguishes
	}

	return total
}

func (player *PlayerStats) getTarget(target steamid.SID64) *TargetStats {
	tSum, found := player.TargetInfo[target]
	if !found {
		tSum = &TargetStats{SteamID: target}
		player.TargetInfo[target] = tSum
	}

	return tSum
}

func (player *PlayerStats) getMedicSum() *HealingStats {
	if player.HealingStats == nil {
		player.HealingStats = newHealingStats(player)
	}

	return player.HealingStats
}

func (player *PlayerStats) KillCount() int {
	var total int
	for _, target := range player.TargetInfo {
		total += len(target.KilledInfo)
	}

	return total
}

func (player *PlayerStats) DominationCount() int {
	var total int
	for _, target := range player.TargetInfo {
		total += target.Dominations
	}

	return total
}

func (player *PlayerStats) DominatedCount() int {
	var total int

	for _, gamePlayer := range player.match.PlayerSums {
		if ti, found := gamePlayer.TargetInfo[player.SteamID]; found {
			total += ti.Dominations
		}
	}

	return total
}

func (player *PlayerStats) RevengeCount() int {
	var total int

	for _, gamePlayer := range player.match.PlayerSums {
		if ti, found := gamePlayer.TargetInfo[player.SteamID]; found {
			total += ti.Revenges
		}
	}

	return total
}

// HealthPacks calculates a total using multipliers the same as logs.tf.
// small = 1, med = 2, full = 4.
func (player *PlayerStats) HealthPacks() int {
	var total int

	for pickup, count := range player.Pickups {
		switch pickup {
		case ItemHPSmall:
			total += count
		case ItemHPMedium:
			total += count * 2
		case ItemHPLarge:
			total += count * 4
		}
	}

	return total
}

func (player *PlayerStats) CaptureCount() int {
	return len(player.Captures)
}

func (player *PlayerStats) CapturesBlockedCount() int {
	return len(player.CapturesBlocked)
}

func (player *PlayerStats) Deaths() int {
	var total int

	for _, target := range player.match.PlayerSums {
		if target.SteamID == player.SteamID {
			continue
		}

		for _, ti := range target.TargetInfo {
			if ti.SteamID == player.SteamID {
				total += len(ti.KilledInfo)

				break
			}
		}
	}

	return total + player.Suicides
}

func (player *PlayerStats) Damage() int {
	var total int
	for _, weaponInfo := range player.WeaponInfo {
		total += weaponInfo.Damage
	}

	return total
}

func (player *PlayerStats) DamagePerMin() int {
	return int(float64(player.Damage()) / player.TimeEnd.Sub(*player.TimeStart).Minutes())
}

func (player *PlayerStats) DamageTakenPerMin() int {
	return int(float64(player.DamageTaken()) / player.TimeEnd.Sub(*player.TimeStart).Minutes())
}

func (player *PlayerStats) BackStabs() int {
	var total int
	for _, weaponInfo := range player.WeaponInfo {
		total += weaponInfo.BackStabs
	}

	return total
}

func (player *PlayerStats) HeadShots() int {
	var total int
	for _, weaponInfo := range player.WeaponInfo {
		total += weaponInfo.Headshots
	}

	return total
}

func (player *PlayerStats) AirShots() int {
	var total int
	for _, weaponInfo := range player.WeaponInfo {
		total += weaponInfo.Airshots
	}

	return total
}

func (player *PlayerStats) resetKillStreak(evtTime time.Time) {
	duration := int(evtTime.Sub(player.currentLifeStart).Seconds())
	if duration > 50000 {
		return
	}

	if player.currentKillStreak >= 3 {
		player.KillStreaks = append(player.KillStreaks, PlayerKillstreak{
			PlayerClass: player.currentClass,
			Killstreak:  player.currentKillStreak,
			Duration:    duration,
		})
	}

	player.currentKillStreak = 0
}

func (player *PlayerStats) addKill(evtTime time.Time, target steamid.SID64, weapon Weapon, sourcePos Pos, targetPos Pos) {
	targetInfo, found := player.TargetInfo[target]
	if !found {
		targetInfo = &TargetStats{
			SteamID: target,
		}

		player.TargetInfo[target] = targetInfo
	}

	ws := player.getWeaponSum(weapon)
	if ws != nil {
		ws.Kills++
	}

	if targetPlayer, ok := player.match.PlayerSums[target]; ok {
		targetPlayer.resetKillStreak(evtTime)

		if classStats := targetPlayer.getClassStats(); classStats != nil {
			classStats.Deaths++
		}
	}

	if classStats := player.getClassStats(); classStats != nil {
		classStats.Kills++
	}

	player.currentKillStreak++

	targetInfo.KilledInfo = append(targetInfo.KilledInfo, KillInfo{
		Weapon:    weapon,
		SourcePos: sourcePos,
		TargetPos: targetPos,
	})
}

func (player *PlayerStats) getWeaponSum(weapon Weapon) *WeaponStats {
	if existing, found := player.WeaponInfo[weapon]; found {
		return existing
	}

	newSum := NewWeaponStats()
	player.WeaponInfo[weapon] = newSum

	return newSum
}

func (player *PlayerStats) AccuracyOverall() float64 {
	var shots, hits int

	for _, info := range player.WeaponInfo {
		if info.Hits > 0 {
			shots += info.Shots
			hits += info.Hits
		}
	}

	if shots == 0 || hits == 0 {
		return 0
	}

	return math.Ceil(float64(hits)/float64(shots)*10000) / 100
}

func (player *PlayerStats) Accuracy(weapon Weapon) float64 {
	if info, found := player.WeaponInfo[weapon]; found {
		return math.Ceil(float64(info.Hits)/float64(info.Shots)*10000) / 100
	}

	return 0
}

func (player *PlayerStats) getClassStats() *PlayerClassStats {
	if player.currentClass == Spectator {
		return nil
	}

	stats, found := player.Classes[player.currentClass]
	if !found {
		stats = &PlayerClassStats{}
		player.Classes[player.currentClass] = stats
	}

	return stats
}

// onClassChangeOrGameEnd updates the players playtime for their current class.
func (player *PlayerStats) onClassChangeOrGameEnd(eventTime time.Time) {
	if player.currentClass == Spectator {
		return
	}

	if cs := player.getClassStats(); cs != nil {
		cs.Playtime += int(eventTime.Sub(cs.startTime).Seconds())
	}
}

type TeamScores struct {
	Red int `json:"red"`
	Blu int `json:"blu"`
}

type MatchRoundSum struct {
	Length      time.Duration `json:"length"`
	Score       TeamScores    `json:"score"`
	KillsBlu    int           `json:"kills_blu"`
	KillsRed    int           `json:"kills_red"`
	UbersBlu    float64       `json:"ubers_blu"`
	UbersRed    float64       `json:"ubers_red"`
	DamageBlu   int           `json:"damage_blu"`
	DamageRed   int           `json:"damage_red"`
	RoundWinner Team          `json:"round_winner,"`
	// MidFight    Team          `json:"mid_fight"`
}

type HealingStats struct {
	MatchMedicID        int64 `json:"match_medic_id"`
	player              *PlayerStats
	FirstHealAfterSpawn []float64           `json:"first_heal_after_spawn"`
	Healing             int                 `json:"healing"`
	Charges             map[MedigunType]int `json:"charges"`
	Drops               []steamid.SID64     `json:"drops"`
	// AvgTimeToBuild      int
	// AvgTimeBeforeUse    int
	NearFullChargeDeath int       `json:"near_full_charge_death"`
	UberDurations       []float32 `json:"uber_durations"`
	// DeathAfterCharge    int
	MajorAdvLost   int `json:"major_adv_lost"`
	BiggestAdvLost int `json:"biggest_adv_lost"`
}

func (ms *HealingStats) ChargesTotal() int {
	var total int

	for _, charges := range ms.Charges {
		total += charges
	}

	return total
}

func (ms *HealingStats) AverageUberLength() float64 {
	var sum float64
	for _, v := range ms.UberDurations {
		sum += float64(v)
	}

	return math.Ceil(sum/float64(len(ms.UberDurations))*10000) / 100
}

func (ms *HealingStats) DropsTotal() int {
	return len(ms.Drops)
}

func (ms *HealingStats) HealingPerMin() int {
	if ms.Healing <= 0 {
		return 0
	}

	return int(float64(ms.Healing) / ms.player.TimeEnd.Sub(*ms.player.TimeStart).Minutes())
}

func newHealingStats(player *PlayerStats) *HealingStats {
	return &HealingStats{
		player: player,
		Charges: map[MedigunType]int{
			Uber:       0,
			Kritzkrieg: 0,
			Vaccinator: 0,
			QuickFix:   0,
		},
	}
}

type HealingStatsMap map[steamid.SID64]*HealingStats

func (mps HealingStatsMap) GetBySteamID(steamID steamid.SID64) (*HealingStats, error) {
	if m, found := mps[steamID]; found {
		return m, nil
	}

	return nil, consts.ErrInvalidSID
}
