package logparse

// EventType defines a known, parsable message type.
type EventType int

const (
	// IgnoredMsg is used for messages we are ignoring.
	IgnoredMsg EventType = 0
	// UnknownMsg is for any unexpected message formats.
	UnknownMsg EventType = 1

	// Live player actions.

	Say                 EventType = 10
	SayTeam             EventType = 11
	Killed              EventType = 12
	KillAssist          EventType = 13
	Suicide             EventType = 14
	ShotFired           EventType = 15
	ShotHit             EventType = 16
	Damage              EventType = 17
	Domination          EventType = 18
	Revenge             EventType = 19
	Pickup              EventType = 20
	EmptyUber           EventType = 21
	MedicDeath          EventType = 22
	MedicDeathEx        EventType = 23
	LostUberAdv         EventType = 24
	ChargeReady         EventType = 25
	ChargeDeployed      EventType = 26
	ChargeEnded         EventType = 27
	Healed              EventType = 28
	Extinguished        EventType = 29
	BuiltObject         EventType = 30
	CarryObject         EventType = 31
	KilledObject        EventType = 32
	DetonatedObject     EventType = 33
	DropObject          EventType = 34
	FirstHealAfterSpawn EventType = 35
	CaptureBlocked      EventType = 36
	PointCaptured       EventType = 48
	JoinedTeam          EventType = 49
	ChangeClass         EventType = 50
	SpawnedAs           EventType = 51
	JarateAttack        EventType = 52
	MilkAttack          EventType = 53
	GasAttack           EventType = 54
	KilledCustom                  = 55

	// World events not attached to specific players.

	WRoundOvertime  EventType = 100
	WRoundStart     EventType = 101
	WRoundWin       EventType = 102
	WRoundLen       EventType = 103
	WTeamScore      EventType = 104
	WTeamFinalScore EventType = 105
	WGameOver       EventType = 106
	WPaused         EventType = 107
	WResumed        EventType = 108
	// WRoundSetupEnd     EventType = 109.
	WMiniRoundWin         EventType = 110 // World triggered "Mini_Round_Win" (winner "Blue") (round "round_a")
	WMiniRoundLen         EventType = 111 // World triggered "Mini_Round_Length" (seconds "820.00")
	WMiniRoundSelected    EventType = 112 // World triggered "Mini_Round_Selected" (round "Round_A")
	WMiniRoundStart       EventType = 113 // World triggered "Mini_Round_Start"
	WRoundSetupBegin      EventType = 114 // World triggered "Round_Setup_Begin"
	WIntermissionWinLimit EventType = 115 // Team "RED" triggered "Intermission_Win_Limit"

	// Metadata.

	LogStart         EventType = 1000
	LogStop          EventType = 1001
	CVAR             EventType = 1002
	RCON             EventType = 1003
	Connected        EventType = 1004
	Disconnected     EventType = 1005
	Validated        EventType = 1006
	Entered          EventType = 1007
	MapLoad          EventType = 1008
	ServerConfigExec EventType = 1009
	SteamAuth        EventType = 1010

	Any EventType = 10000
)

type CritType int

const (
	NonCrit CritType = iota
	Mini
	Crit
)

// Team represents a players team, or spectator state.
type Team int

const (
	UNASSIGNED Team = iota
	SPEC
	RED
	BLU
)

func (t Team) String() string {
	switch t {
	case UNASSIGNED:
		return "UNASSIGNED"
	case RED:
		return "RED"
	case BLU:
		return "BLU"
	case SPEC:
		return "SPEC"
	default:
		return ""
	}
}

func (t Team) Opponent() Team {
	switch t { //nolint:exhaustive
	case RED:
		return BLU
	case BLU:
		return RED
	default:
		return SPEC
	}
}

// PickupItem is used for
//
//goland:noinspection GoUnnecessarilyExportedIdentifiers
type PickupItem int

//goland:noinspection GoUnnecessarilyExportedIdentifiers
const (
	ItemHPSmall PickupItem = iota
	ItemHPMedium
	ItemHPLarge
	ItemAmmoSmall
	ItemAmmoMedium
	ItemAmmoLarge
)

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type PlayerClass int

//goland:noinspection GoUnnecessarilyExportedIdentifiers
const (
	Spectator PlayerClass = iota
	Scout
	Soldier
	Pyro
	Demo
	Heavy
	Engineer
	Medic
	Sniper
	Spy
	Multi
)

func (pc PlayerClass) String() string {
	return map[PlayerClass]string{
		Spectator: "spectator",
		Scout:     "scout",
		Soldier:   "soldier",
		Pyro:      "pyro",
		Demo:      "demo",
		Heavy:     "heavy",
		Engineer:  "engineer",
		Medic:     "medic",
		Sniper:    "sniper",
		Spy:       "spy",
		Multi:     "multi",
	}[pc]
}

// MedigunType holds which medigun a player was using
//
//goland:noinspection GoUnnecessarilyExportedIdentifiers
type MedigunType int

//goland:noinspection GoUnnecessarilyExportedIdentifiers
const (
	Uber MedigunType = iota
	Kritzkrieg
	Vaccinator
	QuickFix
)

//goland:noinspection GoUnnecessarilyExportedIdentifiers
type Weapon int

//goland:noinspection GoUnnecessarilyExportedIdentifiers
const (
	UnknownWeapon Weapon = iota
	AiFlamethrower
	Airstrike
	Ambassador
	Amputator
	Atomizer
	AwperHand
	Backburner
	BackScratcher
	Bat
	BazaarBargain
	BigEarner
	Blackbox
	BlackRose
	Blutsauger
	Bonesaw
	Bottle
	BrassBeast
	Bushwacka
	Caber
	Club
	ConscientiousObjector
	CowMangler
	Crossbow
	DeflectPromode
	DeflectRocket
	Degreaser
	DemoKatana
	Detonator
	DiamondBack
	DirectHit
	DisciplinaryAction
	DragonsFury
	DragonsFuryBonus
	Enforcer
	EscapePlan
	EternalReward
	FamilyBusiness
	Fists
	FistsOfSteel
	FlameThrower
	FlareGun
	ForceANature
	FrontierJustice
	FryingPan
	Gunslinger
	GunslingerCombo
	GunslingerTaunt
	HamShank
	HotHand
	Huntsman
	IronBomber
	IronCurtain
	Jag
	Knife
	Kukri
	Kunai
	Letranger
	LibertyLauncher
	LockNLoad
	LongHeatmaker
	LooseCannon
	LooseCannonImpact
	Lugermorph
	Machina
	MachinaPen
	MarketGardener
	Maul
	MiniGun
	MiniSentry
	Natascha
	NecroSmasher
	Original
	PepPistol
	Phlog
	Pistol
	PistolScout
	Powerjack
	ProjectilePipe
	ProjectilePipeRemote
	ProjectileRocket
	ProRifle
	ProSMG
	ProtoSyringe
	Quickiebomb
	Rainblower
	RescueRanger
	ReserveShooter
	Revolver
	Sandman
	Scattergun
	ScorchShot
	ScottishResistance
	Sentry1
	Sentry2
	Sentry3
	SharpDresser
	ShootingStar
	ShortStop
	ShotgunPrimary
	ShotgunPyro
	ShotgunSoldier
	Sledgehammer
	SMG
	SniperRifle
	SodaPopper
	Spycicle
	SydneySleeper
	SyringeGun
	TauntMedic
	Telefrag
	TheClassic
	Tomislav
	Ubersaw
	WarriorsSpirit
	WidowMaker
	World
	Wrangler
	WrapAssassin
	Wrench
	Sapper
	JarBased
	PanicAttack
	PDAEngineer
	TFMedigun
	TFFlameThrower
	Dispenser
	TideTurner
	BabyFaceBlaster
	ClaidheamhMor
	BatOuttaHell
	FortifiedCompound
	TheWinger
	ShortCircuit
	SplendidScreen
	NessiesNineIron
	BuildingCarriedDestroyed
	SouthernHospitality
	BeggarsBazooka
	PersianPersuader
)

// var backStabWeapons = []Weapon{
//	BigEarner,
//	EternalReward,
//	BlackRose,
//	Knife,
//	Kunai,
//	Spycicle,
//	SharpDresser,
//	SniperRifle,
//	Machina,
//	Ambassador,
//	DiamondBack,
//	MarketGardener,
//	// kgb
//	Backburner,
//	AwperHand,
//	ProRifle,
//	ShootingStar,
//	TheClassic,
//}

// func IsCritWeapon(weapon Weapon) bool {
//	return fp.Contains[Weapon](backStabWeapons, weapon)
//}
