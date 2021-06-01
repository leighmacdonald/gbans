package logparse

import (
	"github.com/leighmacdonald/steamid/v2/steamid"
	"time"
)

// EmptyEvt is the base event for all other events. It just contains a timestamp.
//goland:noinspection GoUnnecessarilyExportedIdentifiers
type EmptyEvt struct {
	CreatedOn time.Time `json:"created_on"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type UnhandledMsgEvt EmptyEvt

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type EnteredEvt EmptyEvt

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type WRoundStartEvt EmptyEvt

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type WRoundOvertimeEvt EmptyEvt

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type WPausedEvt EmptyEvt

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type WResumedEvt EmptyEvt

type SourcePlayer struct {
	Name string        `json:"name"`
	PID  int           `json:"pid"`
	SID  steamid.SID64 `json:"sid"`
	Team Team          `json:"team"`
}

// TargetPlayer maps the common secondary player values name_2
//goland:noinspection GoUnnecessarilyExportedIdentifiers
type TargetPlayer struct {
	Name2 string        `json:"name2"`
	PID2  int           `json:"pid2"`
	SID2  steamid.SID64 `json:"sid2"`
	Team2 Team          `json:"team2"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type LogStartEvt struct {
	File    string `json:"file"`
	Game    string `json:"game"`
	Version string `json:"version"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type LogStopEvt EmptyEvt

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type CVAREvt struct {
	CVAR  string `json:"cvar"`
	Value string `json:"value"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type RCONEvt struct {
	Cmd string `json:"cmd"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type JoinedTeamEvt struct {
	EmptyEvt
	SourcePlayer
	Team Team `json:"team"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type ChangeClassEvt struct {
	EmptyEvt
	Class PlayerClass `json:"class"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type SuicideEvt struct {
	EmptyEvt
	SourcePlayer
	Pos Pos `json:"pos"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type MedicDeathEvt struct {
	EmptyEvt
	SourcePlayer
	TargetPlayer
	Healing int `json:"healing"`
	Uber    int `json:"uber"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type MedicDeathExEvt struct {
	UberPct int `json:"uber_pct"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type KilledEvt struct {
	EmptyEvt
	SourcePlayer
	TargetPlayer
	APos       Pos    `json:"attacker_position" mapstructure:"attacker_position"`
	VPos       Pos    `json:"victim_position" mapstructure:"victim_position"`
	Weapon     Weapon `json:"weapon" mapstructure:"weapon"`
	CustomKill string `json:"custom_kill"  mapstructure:"custom_kill"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type KillAssistEvt struct {
	EmptyEvt
	SourcePlayer
	TargetPlayer
	ASPos Pos `json:"as_pos"`
	APos  Pos `json:"a_pos"`
	VPos  Pos `json:"v_pos"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type PointCapturedEvt struct {
	Team       Team   `json:"team"`
	CP         int    `json:"cp"`
	CPName     string `json:"cp_name"`
	NumCappers int    `json:"num_cappers"`
	// TODO parse to player list
	Body string `json:"body"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type ConnectedEvt struct {
	EmptyEvt
	SourcePlayer
	Address string `json:"address"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type DisconnectedEvt struct {
	EmptyEvt
	SourcePlayer
	Reason string `json:"reason"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type KilledObjectEvt struct {
	EmptyEvt
	SourcePlayer
	TargetPlayer
	Object string `json:"object"`
	Weapon Weapon `json:"weapon"`
	APos   Pos    `json:"a_pos"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type CarryObjectEvt struct {
	EmptyEvt
	SourcePlayer
	Object string `json:"object"`
	Pos    Pos    `json:"a_pos"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type DropObjectEvt CarryObjectEvt

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type BuiltObjectEvt CarryObjectEvt

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type WRoundWinEvt struct {
	Winner Team `json:"winner"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type WRoundLenEvt struct {
	Length float64 `json:"length"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type WTeamScoreEvt struct {
	Team    Team `json:"team"`
	Score   int  `json:"score"`
	Players int  `json:"players"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type SayEvt struct {
	EmptyEvt
	SourcePlayer `json:"source"`
	Msg          string `json:"msg"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type SayTeamEvt SayEvt

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type DominationEvt struct {
	EmptyEvt
	SourcePlayer `json:"source"`
	TargetPlayer `json:"target"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type RevengeEvt DominationEvt

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type CaptureBlockedEvt struct {
	EmptyEvt
	SourcePlayer
	CP     int    `json:"cp"`
	CPName string `json:"cp_name"`
	Pos    Pos    `json:"pos"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type FirstHealAfterSpawnEvt struct {
	EmptyEvt
	SourcePlayer
	HealTime float32 `json:"time"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type ChargeReadyEvt struct {
	EmptyEvt
	SourcePlayer
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type ChargeDeployedEvt struct {
	EmptyEvt
	SourcePlayer
	Medigun Medigun `json:"medigun"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type ChargeEndedEvt struct {
	EmptyEvt
	SourcePlayer
	Duration float32 `json:"duration"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type LostUberAdvantageEvt struct {
	EmptyEvt
	SourcePlayer
	AdvTime int `json:"advtime"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type EmptyUberEvt struct {
	EmptyEvt
	SourcePlayer
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type PickupEvt struct {
	EmptyEvt
	SourcePlayer
	Item AmmoPack
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type ShotFiredEvt struct {
	EmptyEvt
	SourcePlayer
	Weapon Weapon `json:"weapon"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type ShotHitEvt struct {
	EmptyEvt
	SourcePlayer
	Weapon Weapon `json:"weapon"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type DamageEvt struct {
	EmptyEvt
	SourcePlayer
	TargetPlayer
	Damage     int    `json:"damage"`
	RealDamage int    `json:"real_damage"`
	Weapon     Weapon `json:"weapon"`
	Healing    int    `json:"healing,omitempty"` // On ubersaw
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type HealedEvt struct {
	EmptyEvt
	SourcePlayer
	TargetPlayer
	Healing int `json:"healing,omitempty"` // On ubersaw
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type WGameOverEvt struct {
	Reason string `json:"reason"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type WTeamFinalScoreEvt struct {
	Score   int `json:"score"`
	Players int `json:"players"`
	EmptyEvt
}
