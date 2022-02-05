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

type WRoundSetupBeginEvt EmptyEvt

type WMiniRoundSelectedEvt EmptyEvt

type WMiniRoundStartEvt EmptyEvt

type WMiniRoundWinEvt EmptyEvt

type WMiniRoundLenEvt EmptyEvt

// SourcePlayer represents the player who initiated the event
type SourcePlayer struct {
	Name string        `json:"name"`
	PID  int           `json:"pid"`
	SID  steamid.SID64 `json:"sid"`
	Team Team          `json:"team"`
}

// TargetPlayer maps the common secondary player values name_2
type TargetPlayer struct {
	Name2 string        `json:"name2"`
	PID2  int           `json:"pid2"`
	SID2  steamid.SID64 `json:"sid2"`
	Team2 Team          `json:"team2"`
}

type LogStartEvt struct {
	File    string `json:"file"`
	Game    string `json:"game"`
	Version string `json:"version"`
	EmptyEvt
}

// LogStopEvt is the server shutting down the map and closing the log
type LogStopEvt EmptyEvt

// CVAREvt is emitted on a cvar change
type CVAREvt struct {
	CVAR  string `json:"cvar"`
	Value string `json:"value"`
	EmptyEvt
}

// RCONEvt is emitted on a rcon connection executing a command
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
	Pos    Pos    `json:"attacker_position" mapstructure:"attacker_position"`
	Weapon Weapon `json:"weapon"`
}

type JarateAttackEvt struct {
	EmptyEvt
	SourcePlayer
	TargetPlayer
	Weapon Weapon `json:"weapon" mapstructure:"weapon"`
	APos   Pos    `json:"attacker_position" mapstructure:"attacker_position"`
	VPos   Pos    `json:"victim_position" mapstructure:"victim_position"`
}

type MilkAttackEvt JarateAttackEvt

type GasAttackEvt JarateAttackEvt

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
	CustomKill string `json:"custom_kill"  mapstructure:"customkill"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type KillAssistEvt struct {
	EmptyEvt
	SourcePlayer
	TargetPlayer
	ASPos Pos `json:"assister_pos"  mapstructure:"assister_position"`
	APos  Pos `json:"attacker_position" mapstructure:"attacker_position"`
	VPos  Pos `json:"victim_position" mapstructure:"victim_position"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type PointCapturedEvt struct {
	Team       Team   `json:"team"`
	CP         int    `json:"cp"`
	CPName     string `json:"cp_name"`
	NumCappers int    `json:"num_cappers"`
	Player1    string `json:"player1" mapstructure:"player1"`
	Position1  Pos    `json:"position1"  mapstructure:"position1"`
	Player2    string `json:"player2" mapstructure:"player2"`
	Position2  Pos    `json:"position2"  mapstructure:"position2"`
	Player3    string `json:"player3" mapstructure:"player3"`
	Position3  Pos    `json:"position3"  mapstructure:"position3"`
	Player4    string `json:"player4" mapstructure:"player4"`
	Position4  Pos    `json:"position4"  mapstructure:"position4"`
	Player5    string `json:"player5" mapstructure:"player5"`
	Position5  Pos    `json:"position5"  mapstructure:"position5"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type ConnectedEvt struct {
	EmptyEvt
	SourcePlayer
	Address string `json:"address"`
	Port    int    `json:"port"`
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
	APos   Pos    `json:"attacker_position"  mapstructure:"attacker_position"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type CarryObjectEvt struct {
	EmptyEvt
	SourcePlayer
	Object string `json:"object"`
	Pos    Pos    `json:"position"  mapstructure:"position"`
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
	Length float64 `json:"seconds" mapstructure:"seconds"`
	EmptyEvt
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type WTeamScoreEvt struct {
	Team    Team `json:"team" mapstructure:"team"`
	Score   int  `json:"score" mapstructure:"score"`
	Players int  `json:"players" mapstructure:"players"`
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
	CP     int    `json:"cp" mapstructure:"cp"`
	CPName string `json:"cpname" mapstructure:"cpname"`
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
	Medigun Medigun `json:"medigun" mapstructure:"medigun"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type ChargeEndedEvt struct {
	EmptyEvt
	SourcePlayer
	Duration float32 `json:"duration" mapstructure:"duration"`
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type LostUberAdvantageEvt struct {
	EmptyEvt
	SourcePlayer
	AdvTime int `json:"time" mapstructure:"time"`
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
	Item    PickupItem
	Healing int `json:"healing" mapstructure:"healing"`
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
