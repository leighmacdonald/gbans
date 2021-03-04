// Package msgtype declares all the known log message formats we can parse
// or at least are aware of and ignoring.
//
// TODO Move this to a tf2 specific package
package msgtype

type MsgType int

const (
	UnhandledMsg MsgType = 0

	// Live player actions
	Say                 MsgType = 10
	SayTeam             MsgType = 11
	Killed              MsgType = 12
	KillAssist          MsgType = 13
	Suicide             MsgType = 14
	ShotFired           MsgType = 15
	ShotHit             MsgType = 16
	Damage              MsgType = 17
	Domination          MsgType = 18
	Revenge             MsgType = 19
	Pickup              MsgType = 20
	EmptyUber           MsgType = 21
	MedicDeath          MsgType = 22
	MedicDeathEx        MsgType = 23
	LostUberAdv         MsgType = 24
	ChargeReady         MsgType = 25
	ChargeDeployed      MsgType = 26
	ChargeEnded         MsgType = 27
	Healed              MsgType = 28
	Extinguished        MsgType = 29
	BuiltObject         MsgType = 30
	CarryObject         MsgType = 31
	KilledObject        MsgType = 32
	DetonatedObject     MsgType = 33
	DropObject          MsgType = 34
	FirstHealAfterSpawn MsgType = 35
	CaptureBlocked      MsgType = 36
	KilledCustom        MsgType = 37
	PointCaptured       MsgType = 48
	JoinedTeam          MsgType = 49
	ChangeClass         MsgType = 50
	SpawnedAs           MsgType = 51

	// World events not attached to specific players
	WRoundOvertime  MsgType = 100
	WRoundStart     MsgType = 101
	WRoundWin       MsgType = 102
	WRoundLen       MsgType = 103
	WTeamScore      MsgType = 104
	WTeamFinalScore MsgType = 105
	WGameOver       MsgType = 106
	WPaused         MsgType = 107
	WResumed        MsgType = 108

	// Metadata
	LogStart     MsgType = 1000
	LogStop      MsgType = 1001
	CVAR         MsgType = 1002
	RCON         MsgType = 1003
	Connected    MsgType = 1004
	Disconnected MsgType = 1005
	Validated    MsgType = 1006
	Entered      MsgType = 1007
)
