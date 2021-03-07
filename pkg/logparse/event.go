package logparse

import (
	"github.com/leighmacdonald/steamid/v2/steamid"
	"time"
)

type EmptyEvt struct {
	CreatedOn time.Time `json:"created_on"`
}

type UnhandledMsgEvt EmptyEvt
type EnteredEvt EmptyEvt
type WRoundStartEvt EmptyEvt
type WRoundOvertimeEvt EmptyEvt
type WPausedEvt EmptyEvt
type WResumedEvt EmptyEvt

// TargetPlayer maps the common secondary player values name_2
type TargetPlayer struct {
	Name2 string        `json:"name_2"`
	PID2  int           `json:"pid_2"`
	SID2  steamid.SID64 `json:"sid_2"`
	Team2 Team          `json:"team_2"`
}

type LogStartEvt struct {
	File    string `json:"file"`
	Game    string `json:"game"`
	Version string `json:"version"`
	EmptyEvt
}

type LogStopEvt EmptyEvt

type CVAREvt struct {
	CVAR  string `json:"cvar"`
	Value string `json:"value"`
	EmptyEvt
}

type RCONEvt struct {
	Cmd string `json:"cmd"`
	EmptyEvt
}

type JoinedTeamEvt struct {
	Team Team `json:"team"`
	EmptyEvt
}

type ChangeClassEvt struct {
	Class PlayerClass `json:"team"`
	EmptyEvt
}

type SuicideEvt struct {
	Pos Pos `json:"pos"`
	EmptyEvt
}

type MedicDeathEvt struct {
	Healing int `json:"healing"`
	Uber    int `json:"uber"`
	TargetPlayer
	EmptyEvt
}

type MedicDeathExEvt struct {
	UberPct int `json:"uber_pct"`
	EmptyEvt
}

type KilledCustomEvt struct {
	APos       Pos    `json:"a_pos"`
	VPos       Pos    `json:"v_pos"`
	CustomKill string `json:"custom_kill"`
	TargetPlayer
	EmptyEvt
}

type KillAssistEvt struct {
	ASPos Pos `json:"as_pos"`
	APos  Pos `json:"a_pos"`
	VPos  Pos `json:"v_pos"`
	TargetPlayer
	EmptyEvt
}

type PointCapturedEvt struct {
	Team       Team   `json:"team"`
	CP         int    `json:"cp"`
	CPName     string `json:"cp_name"`
	NumCappers int    `json:"num_cappers"`
	// TODO parse to player list
	Body string `json:"body"`
	EmptyEvt
}

type ConnectedEvt struct {
	Address string `json:"address"`
	EmptyEvt
}

type DisconnectedEvt struct {
	Reason string `json:"reason"`
	EmptyEvt
}

type KilledObjectEvt struct {
	Object string `json:"object"`
	Weapon Weapon `json:"weapon"`
	APos   Pos    `json:"a_pos"`
	TargetPlayer
	EmptyEvt
}

type CarryObjectEvt struct {
	Object string `json:"object"`
	Pos    Pos    `json:"a_pos"`
	EmptyEvt
}

type DropObjectEvt CarryObjectEvt
type BuiltObjectEvt CarryObjectEvt

type WRoundWinEvt struct {
	Winner Team `json:"winner"`
	EmptyEvt
}

type WRoundLenEvt struct {
	Length float64 `json:"length"`
	EmptyEvt
}

type WTeamScoreEvt struct {
	Team    Team `json:"team"`
	Score   int  `json:"score"`
	Players int  `json:"players"`
	EmptyEvt
}

type SayEvt struct {
	Msg string `json:"msg"`
	EmptyEvt
}

type SayTeamEvt SayEvt

type DominationEvt struct {
	TargetPlayer
	EmptyEvt
}

type RevengeEvt DominationEvt

type CaptureBlockedEvt struct {
	CP     int    `json:"cp"`
	CPName string `json:"cp_name"`
	Pos    Pos    `json:"pos"`
	EmptyEvt
}

type FirstHealAfterSpawnEvt struct {
	HealTime float32 `json:"time"`
	EmptyEvt
}

type ChargeReadyEvt struct {
	EmptyEvt
}

type ChargeDeployedEvt struct {
	Medigun Medigun `json:"medigun"`
	EmptyEvt
}

type ChargeEndedEvt struct {
	Duration float32 `json:"duration"`
	EmptyEvt
}

type MedicDeathEEvt struct {
	UberPct int `json:"uber_pct"`
	EmptyEvt
}

type LostUberAdvantageEvt struct {
	AdvTime int `json:"advtime"`
	EmptyEvt
}

type EmptyUberEvt struct {
	EmptyEvt
}

type PickupEvt struct {
	Item AmmoPack
	EmptyEvt
}

type ShotFiredEvt struct {
	Weapon Weapon `json:"weapon"`
	EmptyEvt
}

type ShotHitEvt struct {
	Weapon Weapon `json:"weapon"`
	EmptyEvt
}

type DamageEvt struct {
	Damage     int    `json:"damage"`
	RealDamage int    `json:"real_damage"`
	Weapon     Weapon `json:"weapon"`
	Healing    int    `json:"healing,omitempty"` // On ubersaw
	TargetPlayer
	EmptyEvt
}

type WGameOverEvt struct {
	Reason string `json:"reason"`
	EmptyEvt
}

type WTeamFinalScoreEvt struct {
	Score   int `json:"score"`
	Players int `json:"players"`
	EmptyEvt
}
