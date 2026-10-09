package demoparse

import (
	"sort"
	"strings"

	demostatsv1 "github.com/leighmacdonald/gbans/internal/demostats/v1"
)

// DemoFromProto converts a v0.3.3+ ConnectRPC ParseDemoResponse into the
// domain Demo model used by the stats and demo packages.
//
// The protobuf schema nests per-player totals under PlayerSummary.stats and
// models classes as a list, while the domain model keeps the historical flat
// layout for backwards compatibility with existing callers and fixtures.
func DemoFromProto(resp *demostatsv1.ParseDemoResponse, fallbackName string) *Demo {
	demo := &Demo{}

	if resp == nil || resp.GetDemo() == nil {
		demo.Filename = fallbackName

		return demo
	}

	out := resp.GetDemo()
	header := out.GetHeader()
	summary := out.GetSummary()

	demo.Filename = out.GetFilename()
	if demo.Filename == "" {
		demo.Filename = fallbackName
	}

	demo.DemoType = DemoType(header.GetDemoType())
	demo.Version = int(header.GetVersion())
	demo.Protocol = int(header.GetProtocol())
	demo.Server = header.GetServer()
	demo.Nick = header.GetNick()
	demo.Map = header.GetMap()
	demo.Game = header.GetGame()
	demo.Duration = float64(header.GetDuration())
	demo.Ticks = int(header.GetTicks())
	demo.Frames = int(header.GetFrames())
	demo.Signon = int(header.GetSignon())

	if summary != nil {
		demo.Rounds = make([]RoundSummary, len(summary.GetRounds()))
		for i, round := range summary.GetRounds() {
			demo.Rounds[i] = roundFromProto(round)
		}

		demo.Chat = make([]ChatMessage, len(summary.GetChat()))
		for i, msg := range summary.GetChat() {
			demo.Chat[i] = ChatMessage{
				User:         msg.GetUser(),
				Tick:         int32(msg.GetTick()), //nolint:gosec
				Message:      msg.GetMessage(),
				IsDead:       msg.GetIsDead(),
				IsTeam:       msg.GetIsTeam(),
				IsSpec:       msg.GetIsSpec(),
				IsNameChange: msg.GetIsNameChange(),
			}
		}

		demo.Votes = make([]VoteSummary, len(summary.GetVotes()))
		for i, vote := range summary.GetVotes() {
			demo.Votes[i] = voteFromProto(vote)
		}

		demo.SourceModVotes = make([]SourceModVote, len(summary.GetSourcemodVotes()))
		for i, vote := range summary.GetSourcemodVotes() {
			demo.SourceModVotes[i] = sourceModVoteFromProto(vote)
		}

		demo.Events = make([]MatchEvent, len(summary.GetEvents()))
		for i, event := range summary.GetEvents() {
			demo.Events[i] = eventFromProto(event)
		}
	}

	return demo
}

func roundFromProto(round *demostatsv1.RoundSummary) RoundSummary {
	out := RoundSummary{
		Winner:        teamToString(round.GetWinner()),
		IsStalemate:   round.GetIsStalemate(),
		IsSuddenDeath: round.GetIsSuddenDeath(),
		Time:          float64(round.GetTime()),
		Mvps:          round.GetMvps(),
		Winners:       round.GetWinners(),
		Losers:        round.GetLosers(),
	}

	out.Players = make([]PlayerSummary, len(round.GetPlayers()))
	for i, player := range round.GetPlayers() {
		out.Players[i] = playerFromProto(player)
	}

	return out
}

func playerFromProto(player *demostatsv1.PlayerSummary) PlayerSummary {
	out := PlayerSummary{
		Name:            player.GetName(),
		SteamID:         player.GetSteamid(),
		TickStart:       int(player.GetTickStart()),
		TickEnd:         int(player.GetTickEnd()),
		Points:          int(player.GetPoints()),
		ConnectionCount: int(player.GetConnectionCount()),
		BonusPoints:     int(player.GetBonusPoints()),

		ScoreboardKills:   int(player.GetScoreboardKills()),
		ScoreboardAssists: int(player.GetScoreboardAssists()),
		Suicides:          int(player.GetSuicides()),
		ScoreboardDeaths:  int(player.GetScoreboardDeaths()),
		PostroundDeaths:   int(player.GetPostroundDeaths()),
		Captures:          int(player.GetCaptures()),
		CapturesBlocked:   int(player.GetCapturesBlocked()),
		ScoreboardDamage:  int(player.GetScoreboardDamage()),
		IsFakePlayer:      player.GetIsFakePlayer(),
		IsHlTv:            player.GetIsHlTv(),
		IsReplay:          player.GetIsReplay(),
	}

	applyStats(&out, player.GetStats())

	out.Classes = make(map[string]Stats, len(player.GetClasses()))
	for _, class := range player.GetClasses() {
		out.Classes[classToString(class.GetClass())] = statsFromProto(class.GetStats())
	}

	out.Weapons = make(map[string]Stats, len(player.GetWeapons()))
	for name, weaponStats := range player.GetWeapons() {
		out.Weapons[name] = statsFromProto(weaponStats)
	}

	if healTargets := player.GetHealTargets(); len(healTargets) > 0 {
		out.HealTargets = make(map[string]float64, len(healTargets))
		for steamID, seconds := range healTargets {
			out.HealTargets[steamID] = float64(seconds)
		}
	}

	return out
}

// applyStats flattens a nested proto Stats message into the player totals.
func applyStats(out *PlayerSummary, stats *demostatsv1.Stats) {
	out.Kills = int(stats.GetKills())
	out.Assists = int(stats.GetAssists())
	out.Deaths = int(stats.GetDeaths())
	out.PostroundKills = int(stats.GetPostroundKills())
	out.PostroundAssists = int(stats.GetPostroundAssists())
	out.PreroundHealing = int(stats.GetPreroundHealing())
	out.Healing = int(stats.GetHealing())
	out.PostroundHealing = int(stats.GetPostroundHealing())
	out.Drops = int(stats.GetDrops())
	out.NearFullChargeDeath = int(stats.GetNearFullChargeDeath())
	out.ChargesUber = int(stats.GetChargesUber())
	out.ChargesKritz = int(stats.GetChargesKritz())
	out.ChargesQuickfix = int(stats.GetChargesQuickfix())
	out.Damage = int(stats.GetDamage())
	out.DamageTaken = int(stats.GetDamageTaken())
	out.Dominations = int(stats.GetDominations())
	out.Dominated = int(stats.GetDominated())
	out.Revenges = int(stats.GetRevenges())
	out.Revenged = int(stats.GetRevenged())
	out.Airshots = int(stats.GetAirshots())
	out.HeadshotKills = int(stats.GetHeadshotKills())
	out.BackstabKills = int(stats.GetBackstabKills())
	out.Headshots = int(stats.GetHeadshots())
	out.Backstabs = int(stats.GetBackstabs())
	out.Captures = int(stats.GetCaptures())
	out.CapturesBlocked = int(stats.GetCapturesBlocked())
	out.WasHeadshot = int(stats.GetWasHeadshot())
	out.WasBackstabbed = int(stats.GetWasBackstabbed())
	out.Shots = int(stats.GetShots())
	out.Hits = int(stats.GetHits())
	out.ObjectBuilt = int(stats.GetObjectBuilt())
	out.ObjectDestroyed = int(stats.GetObjectDestroyed())
	out.Heals = int(stats.GetHeals())
	out.Healed = int(stats.GetHealed())
	out.CrossbowHeals = int(stats.GetCrossbowHeals())
	out.CrossbowHealing = int(stats.GetCrossbowHealing())
	out.HealOnHit = int(stats.GetHealOnHit())
	out.Extinguishes = int(stats.GetExtinguishes())
	out.BuildingHealing = int(stats.GetBuildingHealing())
	out.DroppedUbers = int(stats.GetDroppedUbers())
	out.Reflects = int(stats.GetReflects())
	out.Defenses = int(stats.GetDefenses())
	out.DirectHits = int(stats.GetDirectHits())
	out.Teleports = int(stats.GetTeleports())
	out.PushDistance = int(stats.GetPushDistance())
	out.AmmoPacks = int(stats.GetAmmoPacks())
	out.HealthPacks = int(stats.GetHealthPacks())
	out.HealthPackHealing = int(stats.GetHealthPackHealing())
	out.EnvironmentalDeaths = int(stats.GetEnvironmentalDeaths())
	out.EnvironmentalKills = int(stats.GetEnvironmentalKills())
	out.ObjectPlaced = int(stats.GetObjectPlaced())
	out.ObjectUpgraded = int(stats.GetObjectUpgraded())
	out.ObjectCarried = int(stats.GetObjectCarried())
	out.ObjectDropped = int(stats.GetObjectDropped())
	out.ObjectRemoved = int(stats.GetObjectRemoved())
	out.ObjectDetonated = int(stats.GetObjectDetonated())
}

func statsFromProto(stats *demostatsv1.Stats) Stats {
	var out Stats

	flatten := PlayerSummary{}
	applyStats(&flatten, stats)

	out.Kills = flatten.Kills
	out.Assists = flatten.Assists
	out.Deaths = flatten.Deaths
	out.PostroundKills = flatten.PostroundKills
	out.PostroundAssists = flatten.PostroundAssists
	out.PostroundDeaths = int(stats.GetPostroundDeaths())
	out.Damage = flatten.Damage
	out.DamageTaken = flatten.DamageTaken
	out.Dominations = flatten.Dominations
	out.Dominated = flatten.Dominated
	out.Revenges = flatten.Revenges
	out.Revenged = flatten.Revenged
	out.Airshots = flatten.Airshots
	out.HeadshotKills = flatten.HeadshotKills
	out.BackstabKills = flatten.BackstabKills
	out.Headshots = flatten.Headshots
	out.Backstabs = flatten.Backstabs
	out.WasHeadshot = flatten.WasHeadshot
	out.PreroundHealing = flatten.PreroundHealing
	out.Healing = flatten.Healing
	out.PostroundHealing = flatten.PostroundHealing
	out.Drops = flatten.Drops
	out.NearFullChargeDeath = flatten.NearFullChargeDeath
	out.ChargesUber = flatten.ChargesUber
	out.ChargesKritz = flatten.ChargesKritz
	out.ChargesQuickfix = flatten.ChargesQuickfix
	out.WasBackstabbed = flatten.WasBackstabbed
	out.Captures = flatten.Captures
	out.CapturesBlocked = flatten.CapturesBlocked
	out.Shots = flatten.Shots
	out.Hits = flatten.Hits
	out.ObjectBuilt = flatten.ObjectBuilt
	out.ObjectDestroyed = flatten.ObjectDestroyed
	out.Heals = flatten.Heals
	out.Healed = flatten.Healed
	out.CrossbowHeals = flatten.CrossbowHeals
	out.CrossbowHealing = flatten.CrossbowHealing
	out.HealOnHit = flatten.HealOnHit
	out.Extinguishes = flatten.Extinguishes
	out.BuildingHealing = flatten.BuildingHealing
	out.DroppedUbers = flatten.DroppedUbers
	out.Reflects = flatten.Reflects
	out.Defenses = flatten.Defenses
	out.DirectHits = flatten.DirectHits
	out.Teleports = flatten.Teleports
	out.PushDistance = flatten.PushDistance
	out.EnvironmentalDeaths = int(stats.GetEnvironmentalDeaths())
	out.EnvironmentalKills = int(stats.GetEnvironmentalKills())
	out.ObjectPlaced = int(stats.GetObjectPlaced())
	out.ObjectUpgraded = int(stats.GetObjectUpgraded())
	out.ObjectCarried = int(stats.GetObjectCarried())
	out.ObjectDropped = int(stats.GetObjectDropped())
	out.ObjectRemoved = int(stats.GetObjectRemoved())
	out.ObjectDetonated = int(stats.GetObjectDetonated())
	out.AmmoPacks = flatten.AmmoPacks
	out.HealthPacks = flatten.HealthPacks
	out.HealthPackHealing = flatten.HealthPackHealing

	return out
}

func teamToString(team demostatsv1.Team) string {
	switch team {
	case demostatsv1.Team_TEAM_RED:
		return "red"
	case demostatsv1.Team_TEAM_BLUE:
		return "blue"
	case demostatsv1.Team_TEAM_SPECTATOR:
		return "spec"
	default:
		return ""
	}
}

func classToString(class demostatsv1.Class) string {
	switch class {
	case demostatsv1.Class_CLASS_SCOUT:
		return "scout"
	case demostatsv1.Class_CLASS_SNIPER:
		return "sniper"
	case demostatsv1.Class_CLASS_SOLDIER:
		return "soldier"
	case demostatsv1.Class_CLASS_DEMOMAN:
		return "demoman"
	case demostatsv1.Class_CLASS_MEDIC:
		return "medic"
	case demostatsv1.Class_CLASS_HEAVY:
		return "heavy"
	case demostatsv1.Class_CLASS_PYRO:
		return "pyro"
	case demostatsv1.Class_CLASS_SPY:
		return "spy"
	case demostatsv1.Class_CLASS_ENGINEER:
		return "engineer"
	default:
		return "other"
	}
}

func voteFromProto(vote *demostatsv1.VoteSummary) VoteSummary {
	out := VoteSummary{
		VoteIdx:         int(vote.GetVoteidx()),
		TickStart:       int(vote.GetTickStart()),
		TickEnd:         int(vote.GetTickEnd()),
		Issue:           vote.GetIssue(),
		Param1:          vote.GetParam1(),
		Team:            int(vote.GetTeam()),
		InitiatorEntity: int(vote.GetInitiatorEntity()),
		Initiator:       vote.GetInitiator(),
		InitiatorName:   vote.GetInitiatorName(),
		Options:         vote.GetOptions(),
		Counts:          make([]int, len(vote.GetCounts())),
		PotentialVotes:  int(vote.GetPotentialVotes()),
		Passed:          vote.GetPassed(),
		ResultDetails:   vote.GetResultDetails(),
		ResultParam1:    vote.GetResultParam1(),
	}
	for i, count := range vote.GetCounts() {
		out.Counts[i] = int(count)
	}

	out.Ballots = make([]VoteBallot, len(vote.GetBallots()))
	for i, ballot := range vote.GetBallots() {
		out.Ballots[i] = VoteBallot{
			Tick:        int(ballot.GetTick()),
			VoterEntity: int(ballot.GetVoterEntity()),
			Voter:       ballot.GetVoter(),
			VoterName:   ballot.GetVoterName(),
			Option:      int(ballot.GetOption()),
			OptionName:  ballot.GetOptionName(),
		}
	}

	return out
}

func sourceModVoteFromProto(vote *demostatsv1.SourceModVote) SourceModVote {
	out := SourceModVote{
		Kind:           vote.GetKind(),
		TickStart:      int(vote.GetTickStart()),
		TickEnd:        int(vote.GetTickEnd()),
		TotalVotes:     int(vote.GetTotalVotes()),
		PotentialVotes: int(vote.GetPotentialVotes()),
		Result:         vote.GetResult(),
		Passed:         vote.GetPassed(),
	}

	out.Initiators = make([]SmVoteInitiator, len(vote.GetInitiators()))
	for i, initiator := range vote.GetInitiators() {
		out.Initiators[i] = SmVoteInitiator{
			Name:     initiator.GetName(),
			SteamID:  initiator.GetSteamid(),
			Tick:     int(initiator.GetTick()),
			Current:  int(initiator.GetCurrent()),
			Required: int(initiator.GetRequired()),
		}
	}

	out.Nominations = make([]SmNomination, len(vote.GetNominations()))
	for i, nomination := range vote.GetNominations() {
		out.Nominations[i] = SmNomination{
			Name:    nomination.GetName(),
			SteamID: nomination.GetSteamid(),
			Map:     nomination.GetMap(),
			Tick:    int(nomination.GetTick()),
		}
	}

	out.Options = make([]SmVoteOption, len(vote.GetOptions()))
	for i, option := range vote.GetOptions() {
		out.Options[i] = SmVoteOption{
			Name:  option.GetName(),
			Votes: int(option.GetVotes()),
		}
	}

	return out
}

func positionFromProto(pos *demostatsv1.Position) *Position {
	if pos == nil {
		return nil
	}

	return &Position{X: float64(pos.GetX()), Y: float64(pos.GetY()), Z: float64(pos.GetZ())}
}

func eyeAnglesFromProto(angles *demostatsv1.EyeAngles) *EyeAngles {
	if angles == nil {
		return nil
	}

	return &EyeAngles{Pitch: float64(angles.GetPitch()), Yaw: float64(angles.GetYaw())}
}

func positionToProto(pos *Position) *demostatsv1.Position {
	if pos == nil {
		return nil
	}

	return &demostatsv1.Position{
		X: float32(pos.X), //nolint:gosec
		Y: float32(pos.Y), //nolint:gosec
		Z: float32(pos.Z), //nolint:gosec
	}
}

func eyeAnglesToProto(angles *EyeAngles) *demostatsv1.EyeAngles {
	if angles == nil {
		return nil
	}

	return &demostatsv1.EyeAngles{
		Pitch: float32(angles.Pitch), //nolint:gosec
		Yaw:   float32(angles.Yaw),   //nolint:gosec
	}
}

func buildingToString(building demostatsv1.BuildingType) string {
	switch building {
	case demostatsv1.BuildingType_BUILDING_SENTRY:
		return "sentry"
	case demostatsv1.BuildingType_BUILDING_DISPENSER:
		return "dispenser"
	case demostatsv1.BuildingType_BUILDING_TELEPORTER:
		return "teleporter"
	case demostatsv1.BuildingType_BUILDING_SAPPER:
		return "sapper"
	default:
		return "unknown"
	}
}

func stringToBuilding(building string) demostatsv1.BuildingType {
	switch strings.ToLower(building) {
	case "sentry":
		return demostatsv1.BuildingType_BUILDING_SENTRY
	case "dispenser":
		return demostatsv1.BuildingType_BUILDING_DISPENSER
	case "teleporter":
		return demostatsv1.BuildingType_BUILDING_TELEPORTER
	case "sapper":
		return demostatsv1.BuildingType_BUILDING_SAPPER
	default:
		return demostatsv1.BuildingType_BUILDING_UNKNOWN
	}
}

func eventWinnerToString(winner demostatsv1.Team) string {
	switch winner {
	case demostatsv1.Team_TEAM_RED:
		return "red"
	case demostatsv1.Team_TEAM_BLUE:
		return "blu"
	default:
		return ""
	}
}

func stringToEventWinner(winner string) *demostatsv1.Team {
	switch strings.ToLower(winner) {
	case "red":
		out := demostatsv1.Team_TEAM_RED

		return &out
	case "blu", "blue":
		out := demostatsv1.Team_TEAM_BLUE

		return &out
	default:
		return nil
	}
}

func killFromProto(kill *demostatsv1.KillEvent) KillEvent {
	return KillEvent{
		Tick:         int(kill.GetTick()),
		Killer:       kill.GetKiller(),
		Victim:       kill.GetVictim(),
		Weapon:       kill.GetWeapon(),
		KillerPos:    positionFromProto(kill.GetKillerPos()),
		VictimPos:    positionFromProto(kill.GetVictimPos()),
		KillerAngles: eyeAnglesFromProto(kill.GetKillerAngles()),
		VictimAngles: eyeAnglesFromProto(kill.GetVictimAngles()),
		IsFirstBlood: kill.GetIsFirstBlood(),
		IsDomination: kill.GetIsDomination(),
		IsRevenge:    kill.GetIsRevenge(),
	}
}

// eventFromProto converts a v0.3.3 GameEvent oneof into the domain MatchEvent
// model. Unknown or absent kinds produce a zero MatchEvent carrying only the
// tick so the feed stays append-only and observable.
func eventFromProto(event *demostatsv1.GameEvent) MatchEvent {
	out := MatchEvent{Tick: int(event.GetTick())}

	switch kind := event.GetKind().(type) {
	case *demostatsv1.GameEvent_Kill:
		out.Type = MatchEventKill
		kill := killFromProto(kind.Kill)
		out.Kill = &kill
	case *demostatsv1.GameEvent_CaptureStarted:
		out.Type = MatchEventCaptureStarted
		out.CaptureStarted = &CaptureStartedEvent{
			Cp: int(kind.CaptureStarted.GetCp()), CpName: kind.CaptureStarted.GetCpName(),
			Team: int(kind.CaptureStarted.GetTeam()), CapTeam: int(kind.CaptureStarted.GetCapTeam()),
			Cappers: kind.CaptureStarted.GetCappers(), CapTime: float64(kind.CaptureStarted.GetCapTime()),
		}
	case *demostatsv1.GameEvent_Capture:
		out.Type = MatchEventCapture
		out.Capture = &CaptureEvent{
			Cp: int(kind.Capture.GetCp()), CpName: kind.Capture.GetCpName(),
			Team: int(kind.Capture.GetTeam()), CapTeam: int(kind.Capture.GetCapTeam()),
			Cappers: kind.Capture.GetCappers(),
		}
	case *demostatsv1.GameEvent_CaptureBlocked:
		out.Type = MatchEventCaptureBlocked
		out.CaptureBlocked = &CaptureBlockedEvent{
			Cp: int(kind.CaptureBlocked.GetCp()), CpName: kind.CaptureBlocked.GetCpName(),
			Blocker: kind.CaptureBlocked.GetBlocker(), Victim: kind.CaptureBlocked.GetVictim(),
		}
	case *demostatsv1.GameEvent_CaptureBroken:
		out.Type = MatchEventCaptureBroken
		out.CaptureBroken = &CaptureBrokenEvent{
			Cp: int(kind.CaptureBroken.GetCp()), CpName: kind.CaptureBroken.GetCpName(),
			TimeRemaining: float64(kind.CaptureBroken.GetTimeRemaining()),
		}
	case *demostatsv1.GameEvent_BuildingBuilt:
		out.Type = MatchEventBuildingBuilt
		out.BuildingBuilt = &BuildingBuiltEvent{
			Owner: kind.BuildingBuilt.GetOwner(), Building: buildingToString(kind.BuildingBuilt.GetBuilding()),
			Level: int(kind.BuildingBuilt.GetLevel()), IsMini: kind.BuildingBuilt.GetIsMini(),
			Pos: *positionFromProto(kind.BuildingBuilt.GetPos()),
		}
	case *demostatsv1.GameEvent_BuildingDestroyed:
		out.Type = MatchEventBuildingDestroyed
		out.BuildingDestroyed = &BuildingDestroyedEvent{
			Owner: kind.BuildingDestroyed.GetOwner(), Attacker: kind.BuildingDestroyed.GetAttacker(),
			Assister: kind.BuildingDestroyed.GetAssister(), Weapon: kind.BuildingDestroyed.GetWeapon(),
			Building: buildingToString(kind.BuildingDestroyed.GetBuilding()),
			Pos:      positionFromProto(kind.BuildingDestroyed.GetPos()),
		}
	case *demostatsv1.GameEvent_BuildingUpgraded:
		out.Type = MatchEventBuildingUpgraded
		out.BuildingUpgraded = buildingLifecycleFromProto(kind.BuildingUpgraded)
	case *demostatsv1.GameEvent_BuildingCarried:
		out.Type = MatchEventBuildingCarried
		out.BuildingCarried = buildingLifecycleFromProto(kind.BuildingCarried)
	case *demostatsv1.GameEvent_BuildingDropped:
		out.Type = MatchEventBuildingDropped
		out.BuildingDropped = buildingLifecycleFromProto(kind.BuildingDropped)
	case *demostatsv1.GameEvent_BuildingRemoved:
		out.Type = MatchEventBuildingRemoved
		out.BuildingRemoved = buildingLifecycleFromProto(kind.BuildingRemoved)
	case *demostatsv1.GameEvent_BuildingDetonated:
		out.Type = MatchEventBuildingDetonated
		out.BuildingDetonated = buildingLifecycleFromProto(kind.BuildingDetonated)
	case *demostatsv1.GameEvent_SapperPlaced:
		out.Type = MatchEventSapperPlaced
		out.SapperPlaced = &SapperPlacedEvent{
			Spy: kind.SapperPlaced.GetSpy(), Owner: kind.SapperPlaced.GetOwner(),
			Building:    buildingToString(kind.SapperPlaced.GetBuilding()),
			SapperIndex: int(kind.SapperPlaced.GetSapperIndex()),
		}
	case *demostatsv1.GameEvent_RoundStarted:
		out.Type = MatchEventRoundStarted
		out.RoundStarted = &RoundStartedEvent{FullReset: kind.RoundStarted.GetFullReset()}
	case *demostatsv1.GameEvent_RoundWon:
		out.Type = MatchEventRoundWon
		out.RoundWon = &RoundWonEvent{
			Winner:      eventWinnerToString(kind.RoundWon.GetWinner()),
			IsStalemate: kind.RoundWon.GetIsStalemate(), WinReason: int(kind.RoundWon.GetWinReason()),
			RoundTime: float64(kind.RoundWon.GetRoundTime()), WasSuddenDeath: kind.RoundWon.GetWasSuddenDeath(),
		}
	case *demostatsv1.GameEvent_Stalemate:
		out.Type = MatchEventStalemate
		out.Stalemate = &StalemateEvent{Reason: int(kind.Stalemate.GetReason())}
	case *demostatsv1.GameEvent_GameOver:
		out.Type = MatchEventGameOver
		out.GameOver = &GameOverEvent{Reason: kind.GameOver.GetReason()}
	case *demostatsv1.GameEvent_SuddenDeathBegin:
		out.Type = MatchEventSuddenDeathBegin
	case *demostatsv1.GameEvent_SuddenDeathEnd:
		out.Type = MatchEventSuddenDeathEnd
	case *demostatsv1.GameEvent_OvertimeBegin:
		out.Type = MatchEventOvertimeBegin
	case *demostatsv1.GameEvent_OvertimeEnd:
		out.Type = MatchEventOvertimeEnd
	case *demostatsv1.GameEvent_SetupFinished:
		out.Type = MatchEventSetupFinished
	case *demostatsv1.GameEvent_UberDropped:
		out.Type = MatchEventUberDropped
		out.UberDropped = &UberDroppedEvent{
			Medic: kind.UberDropped.GetMedic(), Attacker: kind.UberDropped.GetAttacker(),
			Healing: int(kind.UberDropped.GetHealing()),
		}
	case *demostatsv1.GameEvent_UberDeployed:
		out.Type = MatchEventUberDeployed
		out.UberDeployed = &UberDeployedEvent{
			Medic: kind.UberDeployed.GetMedic(), Target: kind.UberDeployed.GetTarget(),
		}
	case *demostatsv1.GameEvent_FlagEvent:
		out.Type = MatchEventFlagEvent
		out.FlagEvent = &FlagEvent{
			Player: kind.FlagEvent.GetPlayer(), Carrier: kind.FlagEvent.GetCarrier(),
			EventType: int(kind.FlagEvent.GetEventType()), Team: int(kind.FlagEvent.GetTeam()),
			Home: kind.FlagEvent.GetHome(),
		}
	case *demostatsv1.GameEvent_FlagCaptured:
		out.Type = MatchEventFlagCaptured
		out.FlagCaptured = &FlagCapturedEvent{
			CappingTeam: int(kind.FlagCaptured.GetCappingTeam()), Score: int(kind.FlagCaptured.GetScore()),
		}
	case *demostatsv1.GameEvent_KillstreakEnded:
		out.Type = MatchEventKillstreakEnded
		out.KillstreakEnded = &KillstreakEndedEvent{
			Player: kind.KillstreakEnded.GetPlayer(), Streak: int(kind.KillstreakEnded.GetStreak()),
			Killer: kind.KillstreakEnded.GetKiller(),
		}
	}

	return out
}

func buildingLifecycleFromProto(lifecycle *demostatsv1.BuildingLifecycle) *BuildingLifecycleEvent {
	return &BuildingLifecycleEvent{
		Player: lifecycle.GetPlayer(), Building: buildingToString(lifecycle.GetBuilding()),
		Index: int(lifecycle.GetIndex()),
	}
}

// eventToProto converts a domain MatchEvent back into a v0.3.3 GameEvent. It
// is primarily used to serve fixture data from ConnectRPC test servers.
func eventToProto(event MatchEvent) *demostatsv1.GameEvent {
	out := &demostatsv1.GameEvent{Tick: uint32(event.Tick)} //nolint:gosec

	switch event.Type {
	case MatchEventKill:
		if event.Kill == nil {
			break
		}
		kill := event.Kill
		out.Kind = &demostatsv1.GameEvent_Kill{Kill: &demostatsv1.KillEvent{
			Tick: uint32(event.Tick), Killer: strPtr(kill.Killer), Victim: kill.Victim, Weapon: kill.Weapon, //nolint:gosec
			KillerPos: positionToProto(kill.KillerPos), VictimPos: positionToProto(kill.VictimPos),
			KillerAngles: eyeAnglesToProto(kill.KillerAngles), VictimAngles: eyeAnglesToProto(kill.VictimAngles),
			IsFirstBlood: kill.IsFirstBlood, IsDomination: kill.IsDomination, IsRevenge: kill.IsRevenge,
		}}
	case MatchEventCaptureStarted:
		if event.CaptureStarted == nil {
			break
		}
		cap := event.CaptureStarted
		out.Kind = &demostatsv1.GameEvent_CaptureStarted{CaptureStarted: &demostatsv1.PointCaptureStart{
			Tick: uint32(event.Tick), Cp: uint32(cap.Cp), CpName: cap.CpName, Team: uint32(cap.Team), //nolint:gosec
			CapTeam: uint32(cap.CapTeam), Cappers: cap.Cappers, CapTime: float32(cap.CapTime), //nolint:gosec
		}}
	case MatchEventCapture:
		if event.Capture == nil {
			break
		}
		cap := event.Capture
		out.Kind = &demostatsv1.GameEvent_Capture{Capture: &demostatsv1.PointCapture{
			Tick: uint32(event.Tick), Cp: uint32(cap.Cp), CpName: cap.CpName, Team: uint32(cap.Team), //nolint:gosec
			CapTeam: uint32(cap.CapTeam), Cappers: cap.Cappers, //nolint:gosec
		}}
	case MatchEventCaptureBlocked:
		if event.CaptureBlocked == nil {
			break
		}
		blocked := event.CaptureBlocked
		out.Kind = &demostatsv1.GameEvent_CaptureBlocked{CaptureBlocked: &demostatsv1.CaptureBlocked{
			Tick: uint32(event.Tick), Cp: uint32(blocked.Cp), CpName: blocked.CpName, //nolint:gosec
			Blocker: strPtr(blocked.Blocker), Victim: strPtr(blocked.Victim),
		}}
	case MatchEventCaptureBroken:
		if event.CaptureBroken == nil {
			break
		}
		broken := event.CaptureBroken
		out.Kind = &demostatsv1.GameEvent_CaptureBroken{CaptureBroken: &demostatsv1.CaptureBroken{
			Tick: uint32(event.Tick), Cp: uint32(broken.Cp), CpName: broken.CpName, //nolint:gosec
			TimeRemaining: float32(broken.TimeRemaining), //nolint:gosec
		}}
	case MatchEventBuildingBuilt:
		if event.BuildingBuilt == nil {
			break
		}
		built := event.BuildingBuilt
		out.Kind = &demostatsv1.GameEvent_BuildingBuilt{BuildingBuilt: &demostatsv1.BuildingBuilt{
			Tick: uint32(event.Tick), Owner: strPtr(built.Owner), Building: buildingToEnum(built.Building), //nolint:gosec
			Level: uint32(built.Level), IsMini: built.IsMini, Pos: positionToProto(&built.Pos), //nolint:gosec
		}}
	case MatchEventBuildingDestroyed:
		if event.BuildingDestroyed == nil {
			break
		}
		destroyed := event.BuildingDestroyed
		out.Kind = &demostatsv1.GameEvent_BuildingDestroyed{BuildingDestroyed: &demostatsv1.BuildingDestroyed{
			Tick: uint32(event.Tick), Owner: strPtr(destroyed.Owner), Attacker: strPtr(destroyed.Attacker), //nolint:gosec
			Assister: strPtr(destroyed.Assister), Weapon: destroyed.Weapon,
			Building: buildingToEnum(destroyed.Building), Pos: positionToProto(destroyed.Pos),
		}}
	case MatchEventBuildingUpgraded:
		out.Kind = &demostatsv1.GameEvent_BuildingUpgraded{BuildingUpgraded: buildingLifecycleToProto(event.Tick, event.BuildingUpgraded)}
	case MatchEventBuildingCarried:
		out.Kind = &demostatsv1.GameEvent_BuildingCarried{BuildingCarried: buildingLifecycleToProto(event.Tick, event.BuildingCarried)}
	case MatchEventBuildingDropped:
		out.Kind = &demostatsv1.GameEvent_BuildingDropped{BuildingDropped: buildingLifecycleToProto(event.Tick, event.BuildingDropped)}
	case MatchEventBuildingRemoved:
		out.Kind = &demostatsv1.GameEvent_BuildingRemoved{BuildingRemoved: buildingLifecycleToProto(event.Tick, event.BuildingRemoved)}
	case MatchEventBuildingDetonated:
		out.Kind = &demostatsv1.GameEvent_BuildingDetonated{BuildingDetonated: buildingLifecycleToProto(event.Tick, event.BuildingDetonated)}
	case MatchEventSapperPlaced:
		if event.SapperPlaced == nil {
			break
		}
		sapper := event.SapperPlaced
		out.Kind = &demostatsv1.GameEvent_SapperPlaced{SapperPlaced: &demostatsv1.SapperPlaced{
			Tick: uint32(event.Tick), Spy: strPtr(sapper.Spy), Owner: strPtr(sapper.Owner), //nolint:gosec
			Building: buildingToEnum(sapper.Building), SapperIndex: uint32(sapper.SapperIndex), //nolint:gosec
		}}
	case MatchEventRoundStarted:
		if event.RoundStarted == nil {
			break
		}
		out.Kind = &demostatsv1.GameEvent_RoundStarted{RoundStarted: &demostatsv1.RoundStarted{
			Tick: uint32(event.Tick), FullReset: event.RoundStarted.FullReset, //nolint:gosec
		}}
	case MatchEventRoundWon:
		if event.RoundWon == nil {
			break
		}
		won := event.RoundWon
		out.Kind = &demostatsv1.GameEvent_RoundWon{RoundWon: &demostatsv1.RoundWon{
			Tick: uint32(event.Tick), Winner: stringToEventWinner(won.Winner), IsStalemate: won.IsStalemate, //nolint:gosec
			WinReason: uint32(won.WinReason), RoundTime: float32(won.RoundTime), WasSuddenDeath: won.WasSuddenDeath, //nolint:gosec
		}}
	case MatchEventStalemate:
		if event.Stalemate == nil {
			break
		}
		out.Kind = &demostatsv1.GameEvent_Stalemate{Stalemate: &demostatsv1.Stalemate{
			Tick: uint32(event.Tick), Reason: uint32(event.Stalemate.Reason), //nolint:gosec
		}}
	case MatchEventGameOver:
		if event.GameOver == nil {
			break
		}
		out.Kind = &demostatsv1.GameEvent_GameOver{GameOver: &demostatsv1.GameOver{
			Tick: uint32(event.Tick), Reason: event.GameOver.Reason, //nolint:gosec
		}}
	case MatchEventSuddenDeathBegin:
		out.Kind = &demostatsv1.GameEvent_SuddenDeathBegin{SuddenDeathBegin: &demostatsv1.TickMarker{Tick: uint32(event.Tick)}} //nolint:gosec
	case MatchEventSuddenDeathEnd:
		out.Kind = &demostatsv1.GameEvent_SuddenDeathEnd{SuddenDeathEnd: &demostatsv1.TickMarker{Tick: uint32(event.Tick)}} //nolint:gosec
	case MatchEventOvertimeBegin:
		out.Kind = &demostatsv1.GameEvent_OvertimeBegin{OvertimeBegin: &demostatsv1.TickMarker{Tick: uint32(event.Tick)}} //nolint:gosec
	case MatchEventOvertimeEnd:
		out.Kind = &demostatsv1.GameEvent_OvertimeEnd{OvertimeEnd: &demostatsv1.TickMarker{Tick: uint32(event.Tick)}} //nolint:gosec
	case MatchEventSetupFinished:
		out.Kind = &demostatsv1.GameEvent_SetupFinished{SetupFinished: &demostatsv1.TickMarker{Tick: uint32(event.Tick)}} //nolint:gosec
	case MatchEventUberDropped:
		if event.UberDropped == nil {
			break
		}
		dropped := event.UberDropped
		out.Kind = &demostatsv1.GameEvent_UberDropped{UberDropped: &demostatsv1.UberDropped{
			Tick: uint32(event.Tick), Medic: strPtr(dropped.Medic), Attacker: strPtr(dropped.Attacker), //nolint:gosec
			Healing: uint32(dropped.Healing), //nolint:gosec
		}}
	case MatchEventUberDeployed:
		if event.UberDeployed == nil {
			break
		}
		deployed := event.UberDeployed
		out.Kind = &demostatsv1.GameEvent_UberDeployed{UberDeployed: &demostatsv1.UberDeployed{
			Tick: uint32(event.Tick), Medic: strPtr(deployed.Medic), Target: strPtr(deployed.Target), //nolint:gosec
		}}
	case MatchEventFlagEvent:
		if event.FlagEvent == nil {
			break
		}
		flag := event.FlagEvent
		out.Kind = &demostatsv1.GameEvent_FlagEvent{FlagEvent: &demostatsv1.FlagEvent{
			Tick: uint32(event.Tick), Player: strPtr(flag.Player), Carrier: strPtr(flag.Carrier), //nolint:gosec
			EventType: uint32(flag.EventType), Team: uint32(flag.Team), Home: flag.Home, //nolint:gosec
		}}
	case MatchEventFlagCaptured:
		if event.FlagCaptured == nil {
			break
		}
		captured := event.FlagCaptured
		out.Kind = &demostatsv1.GameEvent_FlagCaptured{FlagCaptured: &demostatsv1.FlagCaptured{
			Tick: uint32(event.Tick), CappingTeam: uint32(captured.CappingTeam), Score: uint32(captured.Score), //nolint:gosec
		}}
	case MatchEventKillstreakEnded:
		if event.KillstreakEnded == nil {
			break
		}
		streak := event.KillstreakEnded
		out.Kind = &demostatsv1.GameEvent_KillstreakEnded{KillstreakEnded: &demostatsv1.KillstreakEnded{
			Tick: uint32(event.Tick), Player: streak.Player, Streak: uint32(streak.Streak), Killer: strPtr(streak.Killer), //nolint:gosec
		}}
	}

	return out
}

func buildingToEnum(building string) demostatsv1.BuildingType {
	return stringToBuilding(building)
}

// strPtr preserves proto optional presence: empty steamids stay absent rather
// than becoming present-but-empty.
func strPtr(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}

func buildingLifecycleToProto(tick int, lifecycle *BuildingLifecycleEvent) *demostatsv1.BuildingLifecycle {
	out := &demostatsv1.BuildingLifecycle{Tick: uint32(tick)} //nolint:gosec
	if lifecycle == nil {
		return out
	}

	out.Player = strPtr(lifecycle.Player)
	out.Building = buildingToEnum(lifecycle.Building)
	out.Index = uint32(lifecycle.Index) //nolint:gosec

	return out
}

// ProtoFromDemo converts a domain Demo back into a ParseDemoResponse. It is
// primarily used to serve fixture data from ConnectRPC test servers.
func ProtoFromDemo(demo *Demo) *demostatsv1.ParseDemoResponse {
	out := &demostatsv1.ParseDemoResponse{
		Demo: &demostatsv1.DemoOutput{
			Filename: demo.Filename,
			Header: &demostatsv1.Header{
				DemoType: string(demo.DemoType),
				Version:  uint32(demo.Version),  //nolint:gosec
				Protocol: uint32(demo.Protocol), //nolint:gosec
				Server:   demo.Server,
				Nick:     demo.Nick,
				Map:      demo.Map,
				Game:     demo.Game,
				Duration: float32(demo.Duration),
				Ticks:    uint32(demo.Ticks),  //nolint:gosec
				Frames:   uint32(demo.Frames), //nolint:gosec
				Signon:   uint32(demo.Signon), //nolint:gosec
			},
			Summary: &demostatsv1.DemoSummary{},
		},
	}

	summary := out.GetDemo().GetSummary()

	summary.Rounds = make([]*demostatsv1.RoundSummary, len(demo.Rounds))
	for i, round := range demo.Rounds {
		summary.Rounds[i] = roundToProto(round)
	}

	summary.Chat = make([]*demostatsv1.ChatMessage, len(demo.Chat))
	for i, msg := range demo.Chat {
		summary.Chat[i] = &demostatsv1.ChatMessage{
			Tick:         uint32(msg.Tick), //nolint:gosec
			User:         msg.User,
			Message:      msg.Message,
			IsDead:       msg.IsDead,
			IsTeam:       msg.IsTeam,
			IsSpec:       msg.IsSpec,
			IsNameChange: msg.IsNameChange,
		}
	}

	summary.Events = make([]*demostatsv1.GameEvent, len(demo.Events))
	for i, event := range demo.Events {
		summary.Events[i] = eventToProto(event)
	}

	return out
}

func roundToProto(round RoundSummary) *demostatsv1.RoundSummary {
	out := &demostatsv1.RoundSummary{
		Winner:        stringToTeam(round.Winner),
		IsStalemate:   round.IsStalemate,
		IsSuddenDeath: round.IsSuddenDeath,
		Time:          float32(round.Time),
		Mvps:          round.Mvps,
		Winners:       round.Winners,
		Losers:        round.Losers,
	}

	out.Players = make([]*demostatsv1.PlayerSummary, len(round.Players))
	for i, player := range round.Players {
		out.Players[i] = playerToProto(player)
	}

	return out
}

func playerToProto(player PlayerSummary) *demostatsv1.PlayerSummary {
	out := &demostatsv1.PlayerSummary{
		Name:            player.Name,
		Steamid:         player.SteamID,
		TickStart:       u32Ptr(player.TickStart),
		TickEnd:         u32Ptr(player.TickEnd),
		Points:          u32Ptr(player.Points),
		ConnectionCount: uint32(player.ConnectionCount), //nolint:gosec
		BonusPoints:     u32Ptr(player.BonusPoints),
		Stats:           statsToProto(playerStats(player)),
		ScoreboardKills: u32Ptr(player.ScoreboardKills),
		ScoreboardAssists: func() *uint32 {
			if player.ScoreboardAssists == 0 {
				return nil
			}
			v := uint32(player.ScoreboardAssists) //nolint:gosec

			return &v
		}(),
		Suicides:         uint32(player.Suicides), //nolint:gosec
		ScoreboardDeaths: u32Ptr(player.ScoreboardDeaths),
		PostroundDeaths:  uint32(player.PostroundDeaths), //nolint:gosec
		Captures:         uint32(player.Captures),        //nolint:gosec
		CapturesBlocked:  uint32(player.CapturesBlocked), //nolint:gosec
		ScoreboardDamage: u32Ptr(player.ScoreboardDamage),
		IsFakePlayer:     player.IsFakePlayer,
		IsHlTv:           player.IsHlTv,
		IsReplay:         player.IsReplay,
	}

	classes := make([]*demostatsv1.ClassStats, 0, len(player.Classes))
	for name, stats := range player.Classes {
		classes = append(classes, &demostatsv1.ClassStats{
			Class: stringToClass(name),
			Stats: statsToProto(stats),
		})
	}

	sort.Slice(classes, func(i, j int) bool { return classes[i].GetClass() < classes[j].GetClass() })
	out.Classes = classes

	out.Weapons = make(map[string]*demostatsv1.Stats, len(player.Weapons))
	for name, stats := range player.Weapons {
		out.Weapons[name] = statsToProto(stats)
	}

	if len(player.HealTargets) > 0 {
		out.HealTargets = make(map[string]float32, len(player.HealTargets))
		for steamID, seconds := range player.HealTargets {
			out.HealTargets[steamID] = float32(seconds)
		}
	}

	return out
}

// playerStats collects the flattened player totals back into a Stats value.
func playerStats(player PlayerSummary) Stats {
	return Stats{
		Kills:               player.Kills,
		Assists:             player.Assists,
		Deaths:              player.Deaths,
		PostroundKills:      player.PostroundKills,
		PostroundAssists:    player.PostroundAssists,
		PostroundDeaths:     player.PostroundDeaths,
		Damage:              player.Damage,
		DamageTaken:         player.DamageTaken,
		Dominations:         player.Dominations,
		Dominated:           player.Dominated,
		Revenges:            player.Revenges,
		Revenged:            player.Revenged,
		Airshots:            player.Airshots,
		HeadshotKills:       player.HeadshotKills,
		BackstabKills:       player.BackstabKills,
		Headshots:           player.Headshots,
		Backstabs:           player.Backstabs,
		WasHeadshot:         player.WasHeadshot,
		PreroundHealing:     player.PreroundHealing,
		Healing:             player.Healing,
		PostroundHealing:    player.PostroundHealing,
		Drops:               player.Drops,
		NearFullChargeDeath: player.NearFullChargeDeath,
		ChargesUber:         player.ChargesUber,
		ChargesKritz:        player.ChargesKritz,
		ChargesVacc:         player.ChargesVacc,
		ChargesQuickfix:     player.ChargesQuickfix,
		WasBackstabbed:      player.WasBackstabbed,
		Captures:            player.Captures,
		CapturesBlocked:     player.CapturesBlocked,
		Shots:               player.Shots,
		Hits:                player.Hits,
		ObjectBuilt:         player.ObjectBuilt,
		ObjectDestroyed:     player.ObjectDestroyed,
		Heals:               player.Heals,
		Healed:              player.Healed,
		CrossbowHeals:       player.CrossbowHeals,
		CrossbowHealing:     player.CrossbowHealing,
		HealOnHit:           player.HealOnHit,
		Extinguishes:        player.Extinguishes,
		BuildingHealing:     player.BuildingHealing,
		DroppedUbers:        player.DroppedUbers,
		Reflects:            player.Reflects,
		Defenses:            player.Defenses,
		DirectHits:          player.DirectHits,
		Teleports:           player.Teleports,
		PushDistance:        player.PushDistance,
		AmmoPacks:           player.AmmoPacks,
		HealthPacks:         player.HealthPacks,
		HealthPackHealing:   player.HealthPackHealing,
		EnvironmentalDeaths: player.EnvironmentalDeaths,
		EnvironmentalKills:  player.EnvironmentalKills,
		ObjectPlaced:        player.ObjectPlaced,
		ObjectUpgraded:      player.ObjectUpgraded,
		ObjectCarried:       player.ObjectCarried,
		ObjectDropped:       player.ObjectDropped,
		ObjectRemoved:       player.ObjectRemoved,
		ObjectDetonated:     player.ObjectDetonated,
	}
}

func statsToProto(stats Stats) *demostatsv1.Stats {
	return &demostatsv1.Stats{
		Kills:               uint32(stats.Kills),               //nolint:gosec
		Assists:             uint32(stats.Assists),             //nolint:gosec
		Deaths:              uint32(stats.Deaths),              //nolint:gosec
		PostroundKills:      uint32(stats.PostroundKills),      //nolint:gosec
		PostroundAssists:    uint32(stats.PostroundAssists),    //nolint:gosec
		PostroundDeaths:     uint32(stats.PostroundDeaths),     //nolint:gosec
		PreroundHealing:     uint32(stats.PreroundHealing),     //nolint:gosec
		Healing:             uint32(stats.Healing),             //nolint:gosec
		PostroundHealing:    uint32(stats.PostroundHealing),    //nolint:gosec
		Drops:               uint32(stats.Drops),               //nolint:gosec
		NearFullChargeDeath: uint32(stats.NearFullChargeDeath), //nolint:gosec
		ChargesUber:         uint32(stats.ChargesUber),         //nolint:gosec
		ChargesKritz:        uint32(stats.ChargesKritz),        //nolint:gosec
		ChargesQuickfix:     uint32(stats.ChargesQuickfix),     //nolint:gosec
		Damage:              uint32(stats.Damage),              //nolint:gosec
		DamageTaken:         uint32(stats.DamageTaken),         //nolint:gosec
		Dominations:         uint32(stats.Dominations),         //nolint:gosec
		Dominated:           uint32(stats.Dominated),           //nolint:gosec
		Revenges:            uint32(stats.Revenges),            //nolint:gosec
		Revenged:            uint32(stats.Revenged),            //nolint:gosec
		Airshots:            uint32(stats.Airshots),            //nolint:gosec
		HeadshotKills:       uint32(stats.HeadshotKills),       //nolint:gosec
		BackstabKills:       uint32(stats.BackstabKills),       //nolint:gosec
		Headshots:           uint32(stats.Headshots),           //nolint:gosec
		Backstabs:           uint32(stats.Backstabs),           //nolint:gosec
		Captures:            uint32(stats.Captures),            //nolint:gosec
		CapturesBlocked:     uint32(stats.CapturesBlocked),     //nolint:gosec
		WasHeadshot:         uint32(stats.WasHeadshot),         //nolint:gosec
		WasBackstabbed:      uint32(stats.WasBackstabbed),      //nolint:gosec
		Shots:               uint32(stats.Shots),               //nolint:gosec
		Hits:                uint32(stats.Hits),                //nolint:gosec
		ObjectBuilt:         uint32(stats.ObjectBuilt),         //nolint:gosec
		ObjectDestroyed:     uint32(stats.ObjectDestroyed),     //nolint:gosec
		Heals:               uint32(stats.Heals),               //nolint:gosec
		Healed:              uint32(stats.Healed),              //nolint:gosec
		CrossbowHeals:       uint32(stats.CrossbowHeals),       //nolint:gosec
		CrossbowHealing:     uint32(stats.CrossbowHealing),     //nolint:gosec
		HealOnHit:           uint32(stats.HealOnHit),           //nolint:gosec
		Extinguishes:        uint32(stats.Extinguishes),        //nolint:gosec
		BuildingHealing:     uint32(stats.BuildingHealing),     //nolint:gosec
		DroppedUbers:        uint32(stats.DroppedUbers),        //nolint:gosec
		Reflects:            uint32(stats.Reflects),            //nolint:gosec
		Defenses:            uint32(stats.Defenses),            //nolint:gosec
		DirectHits:          uint32(stats.DirectHits),          //nolint:gosec
		Teleports:           uint32(stats.Teleports),           //nolint:gosec
		PushDistance:        uint32(stats.PushDistance),        //nolint:gosec
		EnvironmentalDeaths: uint32(stats.EnvironmentalDeaths), //nolint:gosec
		EnvironmentalKills:  uint32(stats.EnvironmentalKills),  //nolint:gosec
		ObjectPlaced:        uint32(stats.ObjectPlaced),        //nolint:gosec
		ObjectUpgraded:      uint32(stats.ObjectUpgraded),      //nolint:gosec
		ObjectCarried:       uint32(stats.ObjectCarried),       //nolint:gosec
		ObjectDropped:       uint32(stats.ObjectDropped),       //nolint:gosec
		ObjectRemoved:       uint32(stats.ObjectRemoved),       //nolint:gosec
		ObjectDetonated:     uint32(stats.ObjectDetonated),     //nolint:gosec
		AmmoPacks:           uint32(stats.AmmoPacks),           //nolint:gosec
		HealthPacks:         uint32(stats.HealthPacks),         //nolint:gosec
		HealthPackHealing:   uint32(stats.HealthPackHealing),   //nolint:gosec
	}
}

func stringToTeam(team string) *demostatsv1.Team {
	var out demostatsv1.Team

	switch strings.ToLower(team) {
	case "red":
		out = demostatsv1.Team_TEAM_RED
	case "blu", "blue":
		out = demostatsv1.Team_TEAM_BLUE
	case "spec", "spectator":
		out = demostatsv1.Team_TEAM_SPECTATOR
	default:
		return nil
	}

	return &out
}

func stringToClass(class string) demostatsv1.Class {
	switch strings.ToLower(class) {
	case "scout":
		return demostatsv1.Class_CLASS_SCOUT
	case "sniper":
		return demostatsv1.Class_CLASS_SNIPER
	case "soldier":
		return demostatsv1.Class_CLASS_SOLDIER
	case "demoman", "demo":
		return demostatsv1.Class_CLASS_DEMOMAN
	case "medic":
		return demostatsv1.Class_CLASS_MEDIC
	case "heavy", "heavyweapons":
		return demostatsv1.Class_CLASS_HEAVY
	case "pyro":
		return demostatsv1.Class_CLASS_PYRO
	case "spy":
		return demostatsv1.Class_CLASS_SPY
	case "engineer", "engy":
		return demostatsv1.Class_CLASS_ENGINEER
	default:
		return demostatsv1.Class_CLASS_OTHER
	}
}

func u32Ptr(value int) *uint32 {
	if value == 0 {
		return nil
	}

	out := uint32(value) //nolint:gosec

	return &out
}
