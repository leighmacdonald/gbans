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

// TargetPlayer maps the common secondary player values name_2
//goland:noinspection GoUnnecessarilyExportedIdentifiers
type TargetPlayer struct {
	Name2 string        `json:"name_2"`
	PID2  int           `json:"pid_2"`
	SID2  steamid.SID64 `json:"sid_2"`
	Team2 Team          `json:"team_2"`
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
	Team Team `json:"team"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type ChangeClassEvt struct {
	Class PlayerClass `json:"team"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type SuicideEvt struct {
	Pos Pos `json:"pos"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type MedicDeathEvt struct {
	Healing int `json:"healing"`
	Uber    int `json:"uber"`
	TargetPlayer
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type MedicDeathExEvt struct {
	UberPct int `json:"uber_pct"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type KilledCustomEvt struct {
	APos       Pos    `json:"a_pos"`
	VPos       Pos    `json:"v_pos"`
	CustomKill string `json:"custom_kill"`
	TargetPlayer
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type KillAssistEvt struct {
	ASPos Pos `json:"as_pos"`
	APos  Pos `json:"a_pos"`
	VPos  Pos `json:"v_pos"`
	TargetPlayer
	EmptyEvt
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
	Address string `json:"address"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type DisconnectedEvt struct {
	Reason string `json:"reason"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type KilledObjectEvt struct {
	Object string `json:"object"`
	Weapon Weapon `json:"weapon"`
	APos   Pos    `json:"a_pos"`
	TargetPlayer
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type CarryObjectEvt struct {
	Object string `json:"object"`
	Pos    Pos    `json:"a_pos"`
	EmptyEvt
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
	Msg string `json:"msg"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type SayTeamEvt SayEvt

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type DominationEvt struct {
	TargetPlayer
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type RevengeEvt DominationEvt

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type CaptureBlockedEvt struct {
	CP     int    `json:"cp"`
	CPName string `json:"cp_name"`
	Pos    Pos    `json:"pos"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type FirstHealAfterSpawnEvt struct {
	HealTime float32 `json:"time"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type ChargeReadyEvt struct {
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type ChargeDeployedEvt struct {
	Medigun Medigun `json:"medigun"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type ChargeEndedEvt struct {
	Duration float32 `json:"duration"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type MedicDeathEEvt struct {
	UberPct int `json:"uber_pct"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type LostUberAdvantageEvt struct {
	AdvTime int `json:"advtime"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type EmptyUberEvt struct {
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type PickupEvt struct {
	Item AmmoPack
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type ShotFiredEvt struct {
	Weapon Weapon `json:"weapon"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type ShotHitEvt struct {
	Weapon Weapon `json:"weapon"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type DamageEvt struct {
	Damage     int    `json:"damage"`
	RealDamage int    `json:"real_damage"`
	Weapon     Weapon `json:"weapon"`
	Healing    int    `json:"healing,omitempty"` // On ubersaw
	TargetPlayer
	EmptyEvt
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
