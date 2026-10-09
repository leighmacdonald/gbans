//nolint:tagliatelle
package demoparse

import (
	"strings"

	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type Demo struct {
	Filename string         `json:"filename"`
	DemoType DemoType       `json:"demo_type"`
	Version  int            `json:"version"`
	Protocol int            `json:"protocol"`
	Server   string         `json:"server"`
	Nick     string         `json:"nick"`
	Map      string         `json:"map"`
	Game     string         `json:"game"`
	Duration float64        `json:"duration"`
	Ticks    int            `json:"ticks"`
	Frames   int            `json:"frames"`
	Signon   int            `json:"signon"`
	Rounds   []RoundSummary `json:"rounds"`
	Chat     []ChatMessage  `json:"chat"`

	Votes          []VoteSummary   `json:"votes"`
	SourceModVotes []SourceModVote `json:"sourcemod_votes"`
	Events         []MatchEvent    `json:"events"`
}

func (d Demo) UserName(steamID steamid.SteamID) string {
	for _, round := range d.Rounds {
		for _, player := range round.Players {
			if steamID.Equal(steamid.New(player.SteamID)) {
				return player.Name
			}
		}
	}

	return steamID.String()
}

func (d Demo) UserSteamID(user string) steamid.SteamID {
	for _, round := range d.Rounds {
		for _, player := range round.Players {
			if strings.EqualFold(player.Name, user) {
				return steamid.New(player.SteamID)
			}
		}
	}

	return steamid.New(0)
}

func (d Demo) Winner() logparse.Team {
	scores := d.Scores()
	if scores.Red > scores.Blu {
		return logparse.RED
	} else if scores.Blu > scores.Red {
		return logparse.BLU
	}

	return logparse.UNASSIGNED
}

func (d Demo) Scores() logparse.TeamScores {
	var scores logparse.TeamScores
	for _, round := range d.Rounds {
		switch strings.ToLower(round.Winner) {
		case "red":
			scores.Red++
		case "blu", "blue":
			scores.Blu++
		}
	}

	return scores
}

func (d Demo) SteamIDs() steamid.Collection {
	var col steamid.Collection

	for _, round := range d.Rounds {
		for _, player := range round.Players {
			sid := steamid.New(player.SteamID)
			if !col.Contains(sid) {
				col = append(col, sid)
			}
		}
	}

	return col
}

type GameState struct {
	Users   map[int]Player        `json:"users"`
	Players map[int]PlayerSummary `json:"players"` //nolint:tagliatelle
	Results Results               `json:"results"` //nolint:tagliatelle
	Rounds  []DemoRoundSummary    `json:"rounds"`
	Chat    []ChatMessage         `json:"chat"`
}

type Stats struct {
	Kills               int `json:"kills"`
	Assists             int `json:"assists"`
	Deaths              int `json:"deaths"`
	PostroundKills      int `json:"postround_kills"`
	PostroundAssists    int `json:"postround_assists"`
	PostroundDeaths     int `json:"postround_deaths"`
	Damage              int `json:"damage"`
	DamageTaken         int `json:"damage_taken"`
	Dominations         int `json:"dominations"`
	Dominated           int `json:"dominated"`
	Revenges            int `json:"revenges"`
	Revenged            int `json:"revenged"`
	Airshots            int `json:"airshots"`
	HeadshotKills       int `json:"headshot_kills"`
	BackstabKills       int `json:"backstab_kills"`
	Headshots           int `json:"headshots"`
	Backstabs           int `json:"backstabs"`
	WasHeadshot         int `json:"was_headshot"`
	PreroundHealing     int `json:"preround_healing"`
	Healing             int `json:"healing"`
	PostroundHealing    int `json:"postround_healing"`
	Drops               int `json:"drops"`
	NearFullChargeDeath int `json:"near_full_charge_death"`
	ChargesUber         int `json:"charges_uber"`
	ChargesKritz        int `json:"charges_kritz"`
	ChargesVacc         int `json:"charges_vacc"`
	ChargesQuickfix     int `json:"charges_quickfix"`
	WasBackstabbed      int `json:"was_backstabbed"`
	Captures            int `json:"captures"`
	CapturesBlocked     int `json:"captures_blocked"`
	Shots               int `json:"shots"`
	Hits                int `json:"hits"`
	ObjectBuilt         int `json:"object_built"`
	ObjectDestroyed     int `json:"object_destroyed"`
	Heals               int `json:"heals"`
	Healed              int `json:"healed"`
	CrossbowHeals       int `json:"crossbow_heals"`
	CrossbowHealing     int `json:"crossbow_healing"`
	HealOnHit           int `json:"heal_on_hit"`
	Extinguishes        int `json:"extinguishes"`
	BuildingHealing     int `json:"building_healing"`
	DroppedUbers        int `json:"dropped_ubers"`
	Reflects            int `json:"reflects"`
	Defenses            int `json:"defenses"`
	DirectHits          int `json:"direct_hits"`
	Teleports           int `json:"teleports"`
	PushDistance        int `json:"push_distance"`
	EnvironmentalDeaths int `json:"environmental_deaths"`
	EnvironmentalKills  int `json:"environmental_kills"`
	ObjectPlaced        int `json:"object_placed"`
	ObjectUpgraded      int `json:"object_upgraded"`
	ObjectCarried       int `json:"object_carried"`
	ObjectDropped       int `json:"object_dropped"`
	ObjectRemoved       int `json:"object_removed"`
	ObjectDetonated     int `json:"object_detonated"`
	AmmoPacks           int `json:"ammo_packs"`
	HealthPacks         int `json:"health_packs"`
	HealthPackHealing   int `json:"health_pack_healing"`
}

type PlayerSummary struct {
	Name string `json:"name"`
	// Not a steamid.SteamID, since this can be BOT
	SteamID         string `json:"steamid"` //nolint:tagliatelle
	Team            string `json:"team"`
	TickStart       int    `json:"tick_start"`
	TickEnd         int    `json:"tick_end"`
	Points          int    `json:"points"`
	ConnectionCount int    `json:"connection_count"`
	BonusPoints     int    `json:"bonus_points"`

	Kills            int `json:"kills"`
	Assists          int `json:"assists"`
	Deaths           int `json:"deaths"`
	PostroundKills   int `json:"postround_kills"`
	PostroundAssists int `json:"postround_assists"`

	PreroundHealing     int `json:"preround_healing"`
	Healing             int `json:"healing"`
	PostroundHealing    int `json:"postround_healing"`
	Drops               int `json:"drops"`
	NearFullChargeDeath int `json:"near_full_charge_death"`
	ChargesUber         int `json:"charges_uber"`
	ChargesKritz        int `json:"charges_kritz"`
	ChargesVacc         int `json:"charges_vacc"`
	ChargesQuickfix     int `json:"charges_quickfix"`
	Damage              int `json:"damage"`
	DamageTaken         int `json:"damage_taken"`
	Dominations         int `json:"dominations"`
	Dominated           int `json:"dominated"`
	Revenges            int `json:"revenges"`
	Revenged            int `json:"revenged"`
	Airshots            int `json:"airshots"`
	HeadshotKills       int `json:"headshot_kills"`
	BackstabKills       int `json:"backstab_kills"`
	Headshots           int `json:"headshots"`
	Backstabs           int `json:"backstabs"`
	WasHeadshot         int `json:"was_headshot"`
	WasBackstabbed      int `json:"was_backstabbed"`
	Shots               int `json:"shots"`
	Hits                int `json:"hits"`
	ObjectBuilt         int `json:"object_built"`
	ObjectDestroyed     int `json:"object_destroyed"`

	Heals             int `json:"heals"`
	Healed            int `json:"healed"`
	CrossbowHeals     int `json:"crossbow_heals"`
	CrossbowHealing   int `json:"crossbow_healing"`
	HealOnHit         int `json:"heal_on_hit"`
	BuildingHealing   int `json:"building_healing"`
	DroppedUbers      int `json:"dropped_ubers"`
	Reflects          int `json:"reflects"`
	Defenses          int `json:"defenses"`
	DirectHits        int `json:"direct_hits"`
	Teleports         int `json:"teleports"`
	PushDistance      int `json:"push_distance"`
	AmmoPacks         int `json:"ammo_packs"`
	HealthPacks       int `json:"health_packs"`
	HealthPackHealing int `json:"health_pack_healing"`

	EnvironmentalDeaths int `json:"environmental_deaths"`
	EnvironmentalKills  int `json:"environmental_kills"`
	ObjectPlaced        int `json:"object_placed"`
	ObjectUpgraded      int `json:"object_upgraded"`
	ObjectCarried       int `json:"object_carried"`
	ObjectDropped       int `json:"object_dropped"`
	ObjectRemoved       int `json:"object_removed"`
	ObjectDetonated     int `json:"object_detonated"`

	Classes map[string]Stats `json:"classes"`
	Weapons map[string]Stats `json:"weapons"`

	HealTargets map[string]float64 `json:"heal_targets"`

	ScoreboardKills   int `json:"scoreboard_kills"`
	ScoreboardAssists int `json:"scoreboard_assists"`
	ScoreboardHealing int `json:"scoreboard_healing"`
	Suicides          int `json:"suicides"`
	ScoreboardDeaths  int `json:"scoreboard_deaths"`
	PostroundDeaths   int `json:"postround_deaths"`

	Captures        int `json:"captures"`
	CapturesBlocked int `json:"captures_blocked"`

	ScoreboardDamage int `json:"scoreboard_damage"`

	IsFakePlayer bool `json:"is_fake_player"`
	IsHlTv       bool `json:"is_hl_tv"`
	IsReplay     bool `json:"is_replay"`

	// TODO
	// HealingTaken     int `json:"healing_taken"`
	// HealthPacksCount int `json:"health_packs_count"`
	// HealingFromPacks int `json:"health_from_packs"`

	Extinguishes int `json:"extinguishes"`
	Ignites      int `json:"ignites"`

	BuildingBuilt     int `json:"building_built"`
	BuildingDestroyed int `json:"building_destroyed"`
}

type RoundSummary struct {
	Winner        string  `json:"winner"`
	IsStalemate   bool    `json:"is_stalemate"`
	IsSuddenDeath bool    `json:"is_sudden_death"`
	Time          float64 `json:"time"` // seconds

	Duration float64         `json:"duration"`
	Mvps     []string        `json:"mvps"`
	Players  []PlayerSummary `json:"players"`

	Winners []string `json:"winners"`
	Losers  []string `json:"losers"`
}

type Player struct {
	Classes map[PlayerClass]int `json:"classes"`
	Name    string              `json:"name"`
	UserID  int                 `json:"userId"`  //nolint:tagliatelle
	SteamID steamid.SteamID     `json:"steamId"` //nolint:tagliatelle
	Team    logparse.Team       `json:"team"`
}

type ChatMessage struct {
	User         string `json:"user"`
	Tick         int32  `json:"tick"`
	Message      string `json:"message"`
	IsDead       bool   `json:"is_dead"`
	IsTeam       bool   `json:"is_team"`
	IsSpec       bool   `json:"is_spec"`
	IsNameChange bool   `json:"is_name_change"`
}

type Results struct {
	ScoreBlu int `json:"score_blu"`
	BluTime  int `json:"blu_time"`
	ScoreRed int `json:"score_red"`
	RedTime  int `json:"red_time"`
}

type DemoRoundSummary struct{}

type VoteBallot struct {
	Tick        int    `json:"tick"`
	VoterEntity int    `json:"voter_entity"`
	Voter       string `json:"voter"`
	VoterName   string `json:"voter_name"`
	Option      int    `json:"option"`
	OptionName  string `json:"option_name"`
}

type VoteSummary struct {
	VoteIdx         int          `json:"voteidx"`
	TickStart       int          `json:"tick_start"`
	TickEnd         int          `json:"tick_end"`
	Issue           string       `json:"issue"`
	Param1          string       `json:"param1"`
	Team            int          `json:"team"`
	InitiatorEntity int          `json:"initiator_entity"`
	Initiator       string       `json:"initiator"`
	InitiatorName   string       `json:"initiator_name"`
	Options         []string     `json:"options"`
	Ballots         []VoteBallot `json:"ballots"`
	Counts          []int        `json:"counts"`
	PotentialVotes  int          `json:"potential_votes"`
	Passed          bool         `json:"passed"`
	ResultDetails   string       `json:"result_details"`
	ResultParam1    string       `json:"result_param1"`
}

type SmVoteInitiator struct {
	Name     string `json:"name"`
	SteamID  string `json:"steamid"`
	Tick     int    `json:"tick"`
	Current  int    `json:"current"`
	Required int    `json:"required"`
}

type SmNomination struct {
	Name    string `json:"name"`
	SteamID string `json:"steamid"`
	Map     string `json:"map"`
	Tick    int    `json:"tick"`
}

type SmVoteOption struct {
	Name  string `json:"name"`
	Votes int    `json:"votes"`
}

type SourceModVote struct {
	Kind           string            `json:"kind"`
	TickStart      int               `json:"tick_start"`
	TickEnd        int               `json:"tick_end"`
	Initiators     []SmVoteInitiator `json:"initiators"`
	Nominations    []SmNomination    `json:"nominations"`
	TotalVotes     int               `json:"total_votes"`
	PotentialVotes int               `json:"potential_votes"`
	Options        []SmVoteOption    `json:"options"`
	Result         string            `json:"result"`
	Passed         bool              `json:"passed"`
}

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type EyeAngles struct {
	Pitch float64 `json:"pitch"`
	Yaw   float64 `json:"yaw"`
}

type KillEvent struct {
	Tick         int        `json:"tick"`
	Killer       string     `json:"killer"`
	Victim       string     `json:"victim"`
	Weapon       string     `json:"weapon"`
	KillerPos    *Position  `json:"killer_pos"`
	VictimPos    *Position  `json:"victim_pos"`
	KillerAngles *EyeAngles `json:"killer_angles"`
	VictimAngles *EyeAngles `json:"victim_angles"`
	IsFirstBlood bool       `json:"is_first_blood"`
	IsDomination bool       `json:"is_domination"`
	IsRevenge    bool       `json:"is_revenge"`
}

// MatchEventType identifies the variant of a MatchEvent. Values mirror the
// snake_case variant names exported by tf2_demostats.
type MatchEventType string

const (
	MatchEventKill              MatchEventType = "kill"
	MatchEventCaptureStarted    MatchEventType = "capture_started"
	MatchEventCapture           MatchEventType = "capture"
	MatchEventCaptureBlocked    MatchEventType = "capture_blocked"
	MatchEventCaptureBroken     MatchEventType = "capture_broken"
	MatchEventBuildingBuilt     MatchEventType = "building_built"
	MatchEventBuildingDestroyed MatchEventType = "building_destroyed"
	MatchEventBuildingUpgraded  MatchEventType = "building_upgraded"
	MatchEventBuildingCarried   MatchEventType = "building_carried"
	MatchEventBuildingDropped   MatchEventType = "building_dropped"
	MatchEventBuildingRemoved   MatchEventType = "building_removed"
	MatchEventBuildingDetonated MatchEventType = "building_detonated"
	MatchEventSapperPlaced      MatchEventType = "sapper_placed"
	MatchEventRoundStarted      MatchEventType = "round_started"
	MatchEventRoundWon          MatchEventType = "round_won"
	MatchEventStalemate         MatchEventType = "stalemate"
	MatchEventGameOver          MatchEventType = "game_over"
	MatchEventSuddenDeathBegin  MatchEventType = "sudden_death_begin"
	MatchEventSuddenDeathEnd    MatchEventType = "sudden_death_end"
	MatchEventOvertimeBegin     MatchEventType = "overtime_begin"
	MatchEventOvertimeEnd       MatchEventType = "overtime_end"
	MatchEventSetupFinished     MatchEventType = "setup_finished"
	MatchEventUberDropped       MatchEventType = "uber_dropped"
	MatchEventUberDeployed      MatchEventType = "uber_deployed"
	MatchEventFlagEvent         MatchEventType = "flag_event"
	MatchEventFlagCaptured      MatchEventType = "flag_captured"
	MatchEventKillstreakEnded   MatchEventType = "killstreak_ended"
)

// MatchEvent is a single noteworthy match moment exported by tf2_demostats
// v0.3.3+. Exactly one variant payload is set; tick-marker variants carry no
// payload beyond Type.
type MatchEvent struct {
	Tick int            `json:"tick"`
	Type MatchEventType `json:"type"`

	Kill              *KillEvent              `json:"kill,omitempty"`
	CaptureStarted    *CaptureStartedEvent    `json:"capture_started,omitempty"`
	Capture           *CaptureEvent           `json:"capture,omitempty"`
	CaptureBlocked    *CaptureBlockedEvent    `json:"capture_blocked,omitempty"`
	CaptureBroken     *CaptureBrokenEvent     `json:"capture_broken,omitempty"`
	BuildingBuilt     *BuildingBuiltEvent     `json:"building_built,omitempty"`
	BuildingDestroyed *BuildingDestroyedEvent `json:"building_destroyed,omitempty"`
	BuildingUpgraded  *BuildingLifecycleEvent `json:"building_upgraded,omitempty"`
	BuildingCarried   *BuildingLifecycleEvent `json:"building_carried,omitempty"`
	BuildingDropped   *BuildingLifecycleEvent `json:"building_dropped,omitempty"`
	BuildingRemoved   *BuildingLifecycleEvent `json:"building_removed,omitempty"`
	BuildingDetonated *BuildingLifecycleEvent `json:"building_detonated,omitempty"`
	SapperPlaced      *SapperPlacedEvent      `json:"sapper_placed,omitempty"`
	RoundStarted      *RoundStartedEvent      `json:"round_started,omitempty"`
	RoundWon          *RoundWonEvent          `json:"round_won,omitempty"`
	Stalemate         *StalemateEvent         `json:"stalemate,omitempty"`
	GameOver          *GameOverEvent          `json:"game_over,omitempty"`
	UberDropped       *UberDroppedEvent       `json:"uber_dropped,omitempty"`
	UberDeployed      *UberDeployedEvent      `json:"uber_deployed,omitempty"`
	FlagEvent         *FlagEvent              `json:"flag_event,omitempty"`
	FlagCaptured      *FlagCapturedEvent      `json:"flag_captured,omitempty"`
	KillstreakEnded   *KillstreakEndedEvent   `json:"killstreak_ended,omitempty"`
}

// SteamIDs returns the raw participant identifiers referenced by the event,
// including multi-party roles such as capture cappers.
func (e MatchEvent) SteamIDs() []string {
	var out []string

	add := func(raw ...string) {
		for _, id := range raw {
			if id == "" {
				continue
			}
			out = append(out, id)
		}
	}

	switch e.Type {
	case MatchEventKill:
		if e.Kill != nil {
			add(e.Kill.Killer, e.Kill.Victim)
		}
	case MatchEventCaptureStarted:
		if e.CaptureStarted != nil {
			add(e.CaptureStarted.Cappers...)
		}
	case MatchEventCapture:
		if e.Capture != nil {
			add(e.Capture.Cappers...)
		}
	case MatchEventCaptureBlocked:
		if e.CaptureBlocked != nil {
			add(e.CaptureBlocked.Blocker, e.CaptureBlocked.Victim)
		}
	case MatchEventBuildingBuilt:
		if e.BuildingBuilt != nil {
			add(e.BuildingBuilt.Owner)
		}
	case MatchEventBuildingDestroyed:
		if e.BuildingDestroyed != nil {
			add(e.BuildingDestroyed.Owner, e.BuildingDestroyed.Attacker, e.BuildingDestroyed.Assister)
		}
	case MatchEventBuildingUpgraded:
		if e.BuildingUpgraded != nil {
			add(e.BuildingUpgraded.Player)
		}
	case MatchEventBuildingCarried:
		if e.BuildingCarried != nil {
			add(e.BuildingCarried.Player)
		}
	case MatchEventBuildingDropped:
		if e.BuildingDropped != nil {
			add(e.BuildingDropped.Player)
		}
	case MatchEventBuildingRemoved:
		if e.BuildingRemoved != nil {
			add(e.BuildingRemoved.Player)
		}
	case MatchEventBuildingDetonated:
		if e.BuildingDetonated != nil {
			add(e.BuildingDetonated.Player)
		}
	case MatchEventSapperPlaced:
		if e.SapperPlaced != nil {
			add(e.SapperPlaced.Spy, e.SapperPlaced.Owner)
		}
	case MatchEventUberDropped:
		if e.UberDropped != nil {
			add(e.UberDropped.Medic, e.UberDropped.Attacker)
		}
	case MatchEventUberDeployed:
		if e.UberDeployed != nil {
			add(e.UberDeployed.Medic, e.UberDeployed.Target)
		}
	case MatchEventFlagEvent:
		if e.FlagEvent != nil {
			add(e.FlagEvent.Player, e.FlagEvent.Carrier)
		}
	case MatchEventKillstreakEnded:
		if e.KillstreakEnded != nil {
			add(e.KillstreakEnded.Player, e.KillstreakEnded.Killer)
		}
	}

	return out
}

type CaptureStartedEvent struct {
	Cp      int      `json:"cp"`
	CpName  string   `json:"cp_name"`
	Team    int      `json:"team"`
	CapTeam int      `json:"cap_team"`
	Cappers []string `json:"cappers"`
	CapTime float64  `json:"cap_time"`
}

type CaptureEvent struct {
	Cp      int      `json:"cp"`
	CpName  string   `json:"cp_name"`
	Team    int      `json:"team"`
	CapTeam int      `json:"cap_team"`
	Cappers []string `json:"cappers"`
}

type CaptureBlockedEvent struct {
	Cp      int    `json:"cp"`
	CpName  string `json:"cp_name"`
	Blocker string `json:"blocker"`
	Victim  string `json:"victim"`
}

type CaptureBrokenEvent struct {
	Cp            int     `json:"cp"`
	CpName        string  `json:"cp_name"`
	TimeRemaining float64 `json:"time_remaining"`
}

type BuildingBuiltEvent struct {
	Owner    string   `json:"owner"`
	Building string   `json:"building"`
	Level    int      `json:"level"`
	IsMini   bool     `json:"is_mini"`
	Pos      Position `json:"pos"`
}

type BuildingDestroyedEvent struct {
	Owner    string    `json:"owner"`
	Attacker string    `json:"attacker"`
	Assister string    `json:"assister"`
	Weapon   string    `json:"weapon"`
	Building string    `json:"building"`
	Pos      *Position `json:"pos"`
}

type BuildingLifecycleEvent struct {
	Player   string `json:"player"`
	Building string `json:"building"`
	Index    int    `json:"index"`
}

type SapperPlacedEvent struct {
	Spy         string `json:"spy"`
	Owner       string `json:"owner"`
	Building    string `json:"building"`
	SapperIndex int    `json:"sapper_index"`
}

type RoundStartedEvent struct {
	FullReset bool `json:"full_reset"`
}

type RoundWonEvent struct {
	Winner         string  `json:"winner"`
	IsStalemate    bool    `json:"is_stalemate"`
	WinReason      int     `json:"win_reason"`
	RoundTime      float64 `json:"round_time"`
	WasSuddenDeath bool    `json:"was_sudden_death"`
}

type StalemateEvent struct {
	Reason int `json:"reason"`
}

type GameOverEvent struct {
	Reason string `json:"reason"`
}

type UberDroppedEvent struct {
	Medic    string `json:"medic"`
	Attacker string `json:"attacker"`
	Healing  int    `json:"healing"`
}

type UberDeployedEvent struct {
	Medic  string `json:"medic"`
	Target string `json:"target"`
}

type FlagEvent struct {
	Player    string `json:"player"`
	Carrier   string `json:"carrier"`
	EventType int    `json:"event_type"`
	Team      int    `json:"team"`
	Home      bool   `json:"home"`
}

type FlagCapturedEvent struct {
	CappingTeam int `json:"capping_team"`
	Score       int `json:"score"`
}

type KillstreakEndedEvent struct {
	Player string `json:"player"`
	Streak int    `json:"streak"`
	Killer string `json:"killer"`
}
