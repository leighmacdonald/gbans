package logparse

import (
	"time"

	"github.com/leighmacdonald/steamid/v3/steamid"
)

// TimeStamp is the base event for all other events. It just contains a timestamp.
type TimeStamp struct {
	CreatedOn time.Time `json:"created_on" mapstructure:"created_on"`
}

type IgnoredMsgEvt struct {
	TimeStamp
	Message string
}

type UnknownMsgEvt IgnoredMsgEvt

type WRoundStartEvt TimeStamp

type WRoundOvertimeEvt TimeStamp

type WPausedEvt TimeStamp

type WResumedEvt TimeStamp

type WRoundSetupBeginEvt TimeStamp

type WMiniRoundSelectedEvt TimeStamp

type WMiniRoundStartEvt TimeStamp

type WMiniRoundWinEvt TimeStamp

type WMiniRoundLenEvt TimeStamp

// SourcePlayer represents the player who initiated the event.
type SourcePlayer struct {
	Name string        `json:"name"`
	PID  int           `json:"pid"`
	SID  steamid.SID64 `json:"sid"`
	Team Team          `json:"team"`
	Bot  bool          `json:"bot"`
}

// TargetPlayer maps the common secondary player values name_2.
type TargetPlayer struct {
	Name2 string        `json:"name2"`
	PID2  int           `json:"pid2"`
	SID2  steamid.SID64 `json:"sid2"`
	Team2 Team          `json:"team2"`
	Bot2  bool          `json:"bot2"`
}

type EnteredEvt struct {
	TimeStamp
	SourcePlayer
}

type LogStartEvt struct {
	TimeStamp
	File    string `json:"file" mapstructure:"file"`
	Game    string `json:"game" mapstructure:"game"`
	Version string `json:"version" mapstructure:"version"`
}

// LogStopEvt is the server shutting down the map and closing the log.
type LogStopEvt TimeStamp

// CVAREvt is emitted on a cvar change.
type CVAREvt struct {
	TimeStamp
	CVAR  string `json:"cvar" mapstructure:"cvar"`
	Value string `json:"value" mapstructure:"value"`
}

// RCONEvt is emitted on a rcon connection executing a command.
type RCONEvt struct {
	TimeStamp
	Cmd string `json:"cmd" mapstructure:"cmd"`
}

type JoinedTeamEvt struct {
	TimeStamp
	SourcePlayer
	Team Team `json:"new_team" mapstructure:"new_team"`
}

type SpawnedAsEvt struct {
	TimeStamp
	SourcePlayer
	PlayerClass PlayerClass `json:"class" mapstructure:"class"`
}

type ChangeClassEvt struct {
	TimeStamp
	SourcePlayer
	Class PlayerClass `json:"class" mapstructure:"class"`
}

type SuicideEvt struct {
	TimeStamp
	SourcePlayer
	Pos    Pos    `json:"attacker_position" mapstructure:"attacker_position"`
	Weapon Weapon `json:"weapon" mapstructure:"weapon"`
}

type JarateAttackEvt struct {
	TimeStamp
	SourcePlayer
	TargetPlayer
	Weapon Weapon `json:"weapon" mapstructure:"weapon"`
	APos   Pos    `json:"attacker_position" mapstructure:"attacker_position"`
	VPos   Pos    `json:"victim_position" mapstructure:"victim_position"`
}

type MilkAttackEvt JarateAttackEvt

type GasAttackEvt JarateAttackEvt

type ExtinguishedEvt JarateAttackEvt

type MedicDeathEvt struct {
	TimeStamp
	SourcePlayer
	TargetPlayer
	Healing int  `json:"healing" mapstructure:"healing"`
	HadUber bool `json:"ubercharge" mapstructure:"ubercharge"`
}

type MedicDeathExEvt struct {
	TimeStamp
	SourcePlayer
	UberPct int `json:"uberpct" mapstructure:"uberpct"`
}

type KilledEvt struct {
	TimeStamp
	SourcePlayer
	TargetPlayer
	APos   Pos    `json:"attacker_position" mapstructure:"attacker_position"`
	VPos   Pos    `json:"victim_position" mapstructure:"victim_position"`
	Weapon Weapon `json:"weapon" mapstructure:"weapon"`
}

type CustomKilledEvt struct {
	TimeStamp
	SourcePlayer
	TargetPlayer
	APos       Pos    `json:"attacker_position" mapstructure:"attacker_position"`
	VPos       Pos    `json:"victim_position" mapstructure:"victim_position"`
	CustomKill string `json:"customkill"  mapstructure:"customkill"`
	Weapon     Weapon `json:"weapon" mapstructure:"weapon"`
}

type KillAssistEvt struct {
	TimeStamp
	SourcePlayer
	TargetPlayer
	ASPos Pos `json:"assister_pos"  mapstructure:"assister_position"`
	APos  Pos `json:"attacker_position" mapstructure:"attacker_position"`
	VPos  Pos `json:"victim_position" mapstructure:"victim_position"`
}

type SourcePlayerPosition struct {
	SourcePlayer
	Pos
}

type PointCapturedEvt struct {
	TimeStamp
	Team       Team   `json:"team" mapstructure:"team"`
	CP         int    `json:"cp" mapstructure:"cp"`
	CPName     string `json:"cpname" mapstructure:"cpname"`
	NumCappers int    `json:"numcappers" mapstructure:"numcappers"`
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
}

func (e *PointCapturedEvt) Players() []SourcePlayerPosition {
	var captors []SourcePlayerPosition
	for i := 0; i < e.NumCappers; i++ {
		var ps string
		var pos Pos
		switch i {
		case 0:
			ps = e.Player1
			pos = e.Position1
		case 1:
			ps = e.Player2
			pos = e.Position2
		case 2:
			ps = e.Player3
			pos = e.Position3
		case 3:
			ps = e.Player4
			pos = e.Position4
		case 4:
			ps = e.Player5
			pos = e.Position5
		default:
			continue
		}
		var src SourcePlayer
		if !ParseSourcePlayer(ps, &src) {
			continue
		}
		captors = append(captors, SourcePlayerPosition{
			SourcePlayer: src,
			Pos:          pos,
		})
	}
	return captors
}

type ConnectedEvt struct {
	TimeStamp
	SourcePlayer
	Address string `json:"address" mapstructure:"address"`
	Port    int    `json:"port" mapstructure:"port"`
}

type DisconnectedEvt struct {
	TimeStamp
	SourcePlayer
	Reason string `json:"reason" mapstructure:"reason"`
}

type KilledObjectEvt struct {
	TimeStamp
	SourcePlayer
	TargetPlayer
	Object string `json:"object" mapstructure:"object"`
	Weapon Weapon `json:"weapon" mapstructure:"weapon"`
	APos   Pos    `json:"attacker_position"  mapstructure:"attacker_position"`
}

type CarryObjectEvt struct {
	TimeStamp
	SourcePlayer
	Object string `json:"object" mapstructure:"object"`
	Pos    Pos    `json:"position"  mapstructure:"position"`
}

type DropObjectEvt CarryObjectEvt

type BuiltObjectEvt CarryObjectEvt

type DetonatedObjectEvt CarryObjectEvt

type WIntermissionWinLimitEvt struct {
	TimeStamp
	Team Team `json:"team" mapstructure:"team"`
}

type WRoundWinEvt struct {
	TimeStamp
	Winner Team `json:"winner" mapstructure:"winner"`
}

type WRoundLenEvt struct {
	TimeStamp
	Length float64 `json:"seconds" mapstructure:"seconds"`
}

type WTeamScoreEvt struct {
	TimeStamp
	Team    Team `json:"team" mapstructure:"team"`
	Score   int  `json:"score" mapstructure:"score"`
	Players int  `json:"players" mapstructure:"players"`
}

type SayEvt struct {
	TimeStamp
	SourcePlayer `json:"source"`
	Msg          string `json:"msg" mapstructure:"msg"`
}

type SayTeamEvt SayEvt

type DominationEvt struct {
	TimeStamp
	SourcePlayer `json:"source"`
	TargetPlayer `json:"target"`
}

type RevengeEvt DominationEvt

type CaptureBlockedEvt struct {
	TimeStamp
	SourcePlayer
	CP     int    `json:"cp" mapstructure:"cp"`
	CPName string `json:"cpname" mapstructure:"cpname"`
	Pos    Pos    `json:"position" mapstructure:"position"`
}

type FirstHealAfterSpawnEvt struct {
	TimeStamp
	SourcePlayer
	HealTime float64 `json:"time" mapstructure:"time"`
}

type ChargeReadyEvt struct {
	TimeStamp
	SourcePlayer
}

type ChargeDeployedEvt struct {
	TimeStamp
	SourcePlayer
	Medigun MedigunType `json:"medigun" mapstructure:"medigun"`
}

type ChargeEndedEvt struct {
	TimeStamp
	SourcePlayer
	Duration float32 `json:"duration" mapstructure:"duration"`
}

type LostUberAdvantageEvt struct {
	TimeStamp
	SourcePlayer
	AdvTime int `json:"time" mapstructure:"time"`
}

type EmptyUberEvt struct {
	TimeStamp
	SourcePlayer
}

type PickupEvt struct {
	TimeStamp
	SourcePlayer
	Item    PickupItem
	Healing int64 `json:"healing" mapstructure:"healing"`
}

type ShotFiredEvt struct {
	TimeStamp
	SourcePlayer
	Weapon Weapon `json:"weapon" mapstructure:"weapon"`
}

type ShotHitEvt struct {
	TimeStamp
	SourcePlayer
	Weapon Weapon `json:"weapon" mapstructure:"weapon"`
}

type DamageEvt struct {
	TimeStamp
	SourcePlayer
	TargetPlayer
	Damage     int64    `json:"damage" mapstructure:"damage"`
	RealDamage int64    `json:"realdamage" mapstructure:"realdamage"`
	Weapon     Weapon   `json:"weapon" mapstructure:"weapon"`
	Healing    int64    `json:"healing,omitempty" mapstructure:"healing"` // On ubersaw
	Crit       CritType `json:"crit" mapstructure:"crit"`
	AirShot    bool     `json:"airshot" mapstructure:"airshot"` // 1/0
}

type HealedEvt struct {
	TimeStamp
	SourcePlayer
	TargetPlayer
	Healing int64 `json:"healing,omitempty" mapstructure:"healing"` // On ubersaw
}

type WGameOverEvt struct {
	TimeStamp
	Reason string `json:"reason" mapstructure:"reason"`
}

type WTeamFinalScoreEvt struct {
	TimeStamp
	Score   int `json:"score" mapstructure:"score"`
	Players int `json:"players" mapstructure:"players"`
}
