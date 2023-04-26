package logparse

import (
	"fmt"
	"github.com/pkg/errors"
	"strconv"
	"strings"
)

// EventType defines a known, parsable message type
type EventType int

const (
	// IgnoredMsg is used for messages we are ignoring
	IgnoredMsg EventType = 0
	// UnknownMsg is for any unexpected message formats
	UnknownMsg EventType = 1

	// Live player actions

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

	// World events not attached to specific players

	WRoundOvertime  EventType = 100
	WRoundStart     EventType = 101
	WRoundWin       EventType = 102
	WRoundLen       EventType = 103
	WTeamScore      EventType = 104
	WTeamFinalScore EventType = 105
	WGameOver       EventType = 106
	WPaused         EventType = 107
	WResumed        EventType = 108
	// WRoundSetupEnd     EventType = 109
	WMiniRoundWin      EventType = 110 // World triggered "Mini_Round_Win" (winner "Blue") (round "round_a")
	WMiniRoundLen      EventType = 111 // World triggered "Mini_Round_Length" (seconds "820.00")
	WMiniRoundSelected EventType = 112 // World triggered "Mini_Round_Selected" (round "Round_A")
	WMiniRoundStart    EventType = 113 // World triggered "Mini_Round_Start"
	WRoundSetupBegin   EventType = 114 // World triggered "Round_Setup_Begin"

	// Metadata

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

// Team represents a players team, or spectator state
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
	default:
		return "SPEC"
	}
}

func (t Team) Opponent() Team {
	switch t {
	case RED:
		return BLU
	case BLU:
		return RED
	default:
		return SPEC
	}
}

// Pos is a position in 3D space
type Pos struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// Encode returns a ST_MakePointM
// Uses ESPG 4326 (WSG-84)
func (p *Pos) Encode() string {
	return fmt.Sprintf(`ST_SetSRID(ST_MakePoint(%f, %f, %f), 4326)`, p.Y, p.X, p.Z)
}

// ParsePOS parses a players 3d position
func ParsePOS(s string, p *Pos) error {
	pcs := strings.Split(s, " ")
	if len(pcs) != 3 {
		return errors.Errorf("Invalid position: %s", s)
	}
	xv, ex := strconv.ParseFloat(pcs[0], 64)
	if ex != nil {
		return ex
	}
	yv, ey := strconv.ParseFloat(pcs[1], 64)
	if ey != nil {
		return ey
	}
	zv, ez := strconv.ParseFloat(pcs[2], 64)
	if ez != nil {
		return ez
	}
	p.X = xv
	p.Y = yv
	p.Z = zv
	return nil
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
)

func (w Weapon) String() string {
	name, found := weaponNames[w]
	if !found {
		return weaponNames[UnknownWeapon]
	}
	return name
}

func ParseWeapon(s string) Weapon {
	for w, v := range weaponNames {
		if v == s {
			return w
		}
	}
	return UnknownWeapon
}

// weaponNames defines string versions for all known weapons
var weaponNames = map[Weapon]string{
	UnknownWeapon:         "unknown",
	AiFlamethrower:        "ai_flamethrower",
	Airstrike:             "airstrike",
	Ambassador:            "ambassador",
	Amputator:             "amputator",
	Atomizer:              "atomizer",
	AwperHand:             "awper_hand",
	Backburner:            "backburner",
	BackScratcher:         "back_scratcher",
	Bat:                   "bat",
	BazaarBargain:         "bazaar_bargain",
	BigEarner:             "big_earner",
	Blackbox:              "blackbox",
	BlackRose:             "black_rose",
	Blutsauger:            "blutsauger",
	Bonesaw:               "bonesaw",
	Bottle:                "bottle",
	BrassBeast:            "brass_beast",
	Bushwacka:             "bushwacka",
	Caber:                 "ullapool_caber",
	Club:                  "club",
	ConscientiousObjector: "nonnonviolent_protest",
	CowMangler:            "cow_mangler",
	Crossbow:              "crusaders_crossbow",
	// TODO add remaining deflects
	DeflectPromode:       "deflect_promode",
	DeflectRocket:        "deflect_rocket",
	Degreaser:            "degreaser",
	DemoKatana:           "demokatana",
	Detonator:            "detonator",
	DiamondBack:          "diamondback",
	DirectHit:            "direct_hit",
	DisciplinaryAction:   "disciplinary_action",
	DragonsFury:          "dragons_fury",
	DragonsFuryBonus:     "dragons_fury_bonus",
	Enforcer:             "enforcer",
	EscapePlan:           "unique_pickaxe_escape",
	EternalReward:        "eternal_reward",
	FamilyBusiness:       "family_business",
	Fists:                "fists",
	FistsOfSteel:         "steel_fists",
	FlameThrower:         "flamethrower",
	FlareGun:             "flaregun",
	ForceANature:         "force_a_nature",
	FrontierJustice:      "frontier_justice",
	FryingPan:            "fryingpan",
	Gunslinger:           "robot_arm",
	GunslingerCombo:      "robot_arm_combo_kill",
	GunslingerTaunt:      "robot_arm_blender_kill",
	HamShank:             "ham_shank",
	HotHand:              "hot_hand",
	Huntsman:             "tf_projectile_arrow",
	IronBomber:           "iron_bomber",
	IronCurtain:          "iron_curtain",
	Jag:                  "wrench_jag",
	JarBased:             "tf_weapon_jar",
	Knife:                "knife",
	Kukri:                "tribalkukri",
	Kunai:                "kunai",
	Letranger:            "letranger",
	LibertyLauncher:      "liberty_launcher",
	LockNLoad:            "loch_n_load",
	LongHeatmaker:        "long_heatmaker",
	LooseCannon:          "loose_cannon",
	LooseCannonImpact:    "loose_cannon_impact",
	Lugermorph:           "maxgun",
	Machina:              "machina",
	MachinaPen:           "player_penetration",
	MarketGardener:       "market_gardener",
	Maul:                 "the_maul",
	MiniGun:              "minigun",
	MiniSentry:           "obj_minisentry",
	Natascha:             "natascha",
	NecroSmasher:         "necro_smasher",
	Original:             "quake_rl",
	PanicAttack:          "panic_attack",
	PDAEngineer:          "pda_engineer",
	PepPistol:            "pep_pistol",
	Phlog:                "phlogistinator",
	Pistol:               "pistol",
	PistolScout:          "pistol_scout",
	Powerjack:            "powerjack",
	ProjectilePipe:       "tf_projectile_pipe",
	ProjectilePipeRemote: "tf_projectile_pipe_remote",
	ProjectileRocket:     "tf_projectile_rocket",
	ProRifle:             "pro_rifle", // heatmaker?
	ProSMG:               "pro_smg",   // carbine?
	ProtoSyringe:         "proto_syringe",
	Quickiebomb:          "quickiebomb_launcher",
	Rainblower:           "rainblower",
	RescueRanger:         "rescue_ranger",
	ReserveShooter:       "reserve_shooter",
	Revolver:             "revolver",
	Sandman:              "sandman",
	Sapper:               "obj_attachment_sapper",
	Scattergun:           "scattergun",
	ScorchShot:           "scorch_shot",
	ScottishResistance:   "sticky_resistance",
	Sentry1:              "obj_sentrygun",
	Sentry2:              "obj_sentrygun2",
	Sentry3:              "obj_sentrygun3",
	SharpDresser:         "sharp_dresser",
	ShootingStar:         "shooting_star",
	ShortStop:            "shortstop",
	ShotgunPrimary:       "shotgun_primary",
	ShotgunPyro:          "shotgun_pyro",
	ShotgunSoldier:       "shotgun_soldier",
	Sledgehammer:         "sledgehammer",
	SMG:                  "smg",
	SniperRifle:          "sniperrifle",
	SodaPopper:           "soda_popper",
	Spycicle:             "spy_cicle",
	SydneySleeper:        "sydney_sleeper",
	SyringeGun:           "syringegun_medic",
	TauntMedic:           "taunt_medic",
	Telefrag:             "telefrag",
	TheClassic:           "the_classic",
	Tomislav:             "tomislav",
	Ubersaw:              "ubersaw",
	WarriorsSpirit:       "warrior_spirit",
	WidowMaker:           "widowmaker",
	World:                "world",
	Wrangler:             "wrangler_kill",
	WrapAssassin:         "wrap_assassin",
	Wrench:               "wrench",
	// Special weapons
	TFMedigun:      "tf_weapon_medigun", // When used to extinguish
	TFFlameThrower: "tf_weapon_flamethrower",
	Dispenser:      "dispenser",
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers,GoUnusedGlobalVariable
var Weapons = map[PlayerClass][]Weapon{
	Multi: {
		ConscientiousObjector,
		FryingPan,
		HamShank,
		Lugermorph,
		NecroSmasher,
		Pistol,
		ReserveShooter,
		Telefrag,
		World,
		JarBased, // reflect?
	},
	Scout: {
		Atomizer,
		Bat,
		ForceANature,
		PepPistol,
		PistolScout,
		Sandman,
		Scattergun,
		ShortStop,
		SodaPopper,
		WrapAssassin,
	},
	Soldier: {
		Airstrike,
		Blackbox,
		CowMangler,
		DirectHit,
		DisciplinaryAction,
		EscapePlan,
		LibertyLauncher,
		MarketGardener,
		Original,
		ProjectileRocket,
		ShotgunSoldier,
	},
	Pyro: {
		AiFlamethrower,
		Backburner,
		BackScratcher,
		DeflectPromode,
		DeflectRocket,
		Degreaser,
		Detonator,
		DragonsFury,
		DragonsFuryBonus,
		FlameThrower,
		FlareGun,
		HotHand,
		Maul,
		Phlog,
		Powerjack,
		Rainblower,
		ScorchShot,
		ShotgunPyro,
		Sledgehammer,
	},
	Demo: {
		Bottle,
		Caber,
		DemoKatana,
		IronBomber,
		LockNLoad,
		LooseCannon,
		LooseCannonImpact,
		ProjectilePipe,
		ProjectilePipeRemote,
		Quickiebomb,
		ScottishResistance,
	},
	Heavy: {
		BrassBeast,
		FamilyBusiness,
		Fists,
		FistsOfSteel,
		IronCurtain,
		LongHeatmaker,
		MiniGun,
		Natascha,
		Tomislav,
		WarriorsSpirit,
	},
	Engineer: {
		FrontierJustice,
		Gunslinger,
		GunslingerCombo,
		GunslingerTaunt,
		Jag,
		MiniSentry,
		RescueRanger,
		Sentry1,
		Sentry2,
		Sentry3,
		ShotgunPrimary,
		WidowMaker,
		Wrangler,
		Wrench,
	},
	Medic: {
		Amputator,
		Blutsauger,
		Bonesaw,
		Crossbow,
		ProtoSyringe,
		SyringeGun,
		TauntMedic,
		Ubersaw,
	},
	Sniper: {
		AwperHand,
		BazaarBargain,
		Bushwacka,
		Club,
		Huntsman,
		Kukri,
		Machina,
		MachinaPen,
		ProRifle,
		ProSMG,
		ShootingStar,
		SMG,
		SniperRifle,
		SydneySleeper,
		TheClassic,
	},
	Spy: {
		Ambassador,
		BigEarner,
		BlackRose,
		DiamondBack,
		Enforcer,
		EternalReward,
		Knife,
		Kunai,
		Letranger,
		Revolver,
		Sapper,
		SharpDresser,
		Spycicle,
	},
}

//var backStabWeapons = []Weapon{
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

//func IsCritWeapon(weapon Weapon) bool {
//	return fp.Contains[Weapon](backStabWeapons, weapon)
//}
