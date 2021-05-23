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
	Class PlayerClass `json:"team"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type SuicideEvt struct {
	EmptyEvt
	SourcePlayer
	AttackerPosition Pos `mapstructure:"attacker_position" json:"attacker_position"`
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
	APos       Pos    `mapstructure:"attacker_position" json:"attacker_position"`
	VPos       Pos    `mapstructure:"victim_position" json:"victim_position"`
	CustomKill string `json:"custom_kill"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type KillAssistEvt struct {
	EmptyEvt
	SourcePlayer
	TargetPlayer
	AssisterPosition Pos `mapstructure:"assister_position" json:"assister_position"`
	AttackerPosition Pos `mapstructure:"attacker_position" json:"attacker_position"`
	VictimPosition   Pos `mapstructure:"victim_position" json:"victim_position"`
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
	APos   Pos    `mapstructure:"attacker_position" json:"attacker_position"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type CarryObjectEvt struct {
	EmptyEvt
	SourcePlayer
	Object           string `json:"object"`
	AttackerPosition Pos    `mapstructure:"position" json:"position"`
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
	SourcePlayer
	Msg string `json:"msg"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type SayTeamEvt SayEvt

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type DominationEvt struct {
	EmptyEvt
	SourcePlayer
	TargetPlayer
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type RevengeEvt DominationEvt

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type CaptureBlockedEvt struct {
	EmptyEvt
	SourcePlayer
	CP     int    `json:"cp"`
	CPName string `json:"cp_name"`
	Pos    Pos    `json:"position" mapstructure:"position"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type FirstHealAfterSpawnEvt struct {
	EmptyEvt
	SourcePlayer
	HealTime float32 `json:"time" mapstructure:"time"`
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
type MedicDeathEEvt struct {
	EmptyEvt
	SourcePlayer
	UberPct int `json:"uber_pct"`
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

// MilkAttackEvt
// L 05/21/2021 - 20:39:34: "Five<636><[U:1:66374745]><Blue>" triggered "milk_attack" against "Silexos<635><[U:1:307374149]><Red>" with "tf_weapon_jar" (attacker_position "-353 -445 52") (victim_position "99 -126 7")
type MilkAttackEvt struct {
	EmptyEvt
	SourcePlayer
	TargetPlayer
	APos Pos `mapstructure:"attacker_position" json:"attacker_position"`
	VPos Pos `mapstructure:"victim_position" json:"victim_position"`
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
