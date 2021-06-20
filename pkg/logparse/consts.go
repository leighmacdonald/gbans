package logparse

import (
	"fmt"
	"github.com/pkg/errors"
	"strconv"
	"strings"
)

// MsgType defines a known, parsable message type
type MsgType int

//goland:noinspection GoUnnecessarilyExportedIdentifiers
const (
	// UnhandledMsg is used for messages we are ignoring
	UnhandledMsg MsgType = 0
	// UnknownMsg is for any unexpected message formats
	UnknownMsg MsgType = 1

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

	Any MsgType = 10000
)

// Team represents a players team, or spectator state
//goland:noinspection GoUnnecessarilyExportedIdentifiers
type Team int

//goland:noinspection GoUnnecessarilyExportedIdentifiers
const (
	SPEC Team = 0
	RED  Team = 1
	BLU  Team = 2
)

// Pos is a position in 3D space
//goland:noinspection GoUnnecessarilyExportedIdentifiers
type Pos struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// String returns a ST_MakePointM
// Uses ESPG 4326 (WSG-84)
func (p *Pos) Encode() string {
	return fmt.Sprintf(`ST_SetSRID(ST_MakePoint(%f, %f, %f), 4326)`, p.Y, p.X, p.Z)
}

func NewPosFromString(s string, p *Pos) error {
	pcs := strings.Split(s, " ")
	if len(pcs) != 3 {
		return errors.New("Invalid position")
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
	Spectator PlayerClass = 0
	Scout     PlayerClass = 1
	Soldier   PlayerClass = 2
	Pyro      PlayerClass = 3
	Demo      PlayerClass = 4
	Heavy     PlayerClass = 5
	Engineer  PlayerClass = 6
	Medic     PlayerClass = 7
	Sniper    PlayerClass = 8
	Spy       PlayerClass = 9
	Multi     PlayerClass = 10
)

// Medigun holds which medigun a player was using
//goland:noinspection GoUnnecessarilyExportedIdentifiers
type Medigun int

//goland:noinspection GoUnnecessarilyExportedIdentifiers
const (
	Uber Medigun = iota
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
)

func (w Weapon) String() string {
	name, found := weaponNames[w]
	if !found {
		return weaponNames[UnknownWeapon]
	}
	return name
}

func WeaponFromString(s string) Weapon {
	for w, v := range weaponNames {
		if v == s {
			return w
		}
	}
	return UnknownWeapon
}

// weaponNames defines string versions for all all known weapons
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
	DeflectPromode:        "deflect_promode",
	DeflectRocket:         "deflect_rocket",
	Degreaser:             "degreaser",
	DemoKatana:            "demokatana",
	Detonator:             "detonator",
	DiamondBack:           "diamondback",
	DirectHit:             "rocketlauncher_directhit",
	DisciplinaryAction:    "disciplinary_action",
	DragonsFury:           "dragons_fury",
	DragonsFuryBonus:      "dragons_fury_bonus",
	Enforcer:              "enforcer",
	EscapePlan:            "unique_pickaxe_escape",
	EternalReward:         "eternal_reward",
	FamilyBusiness:        "family_business",
	Fists:                 "fists",
	FistsOfSteel:          "steel_fists",
	FlameThrower:          "flamethrower",
	FlareGun:              "flaregun",
	ForceANature:          "force_a_nature",
	FrontierJustice:       "frontier_justice",
	FryingPan:             "fryingpan",
	Gunslinger:            "robot_arm",
	GunslingerCombo:       "robot_arm_combo_kill", // what is this?
	GunslingerTaunt:       "robot_arm_blender_kill",
	HamShank:              "ham_shank",
	HotHand:               "hot_hand",
	Huntsman:              "tf_projectile_arrow",
	IronBomber:            "iron_bomber",
	IronCurtain:           "iron_curtain",
	Jag:                   "wrench_jag",
	Knife:                 "knife",
	Kukri:                 "tribalkukri",
	Kunai:                 "kunai",
	Letranger:             "letranger",
	LibertyLauncher:       "liberty_launcher",
	LockNLoad:             "loch_n_load",
	LongHeatmaker:         "long_heatmaker",
	LooseCannon:           "loose_cannon",
	LooseCannonImpact:     "loose_cannon_impact",
	Machina:               "machina",
	MachinaPen:            "player_penetration",
	MarketGardener:        "market_gardener",
	Maul:                  "the_maul",
	MiniGun:               "minigun",
	MiniSentry:            "obj_minisentry",
	Natascha:              "natascha",
	NecroSmasher:          "necro_smasher",
	Original:              "quake_rl",
	PepPistol:             "pep_pistol",
	Phlog:                 "phlogistinator",
	Pistol:                "pistol",
	PistolScout:           "pistol_scout",
	Powerjack:             "powerjack",
	ProjectilePipe:        "tf_projectile_pipe",
	ProjectilePipeRemote:  "tf_projectile_pipe_remote",
	ProjectileRocket:      "tf_projectile_rocket",
	ProRifle:              "pro_rifle", // heatmaker?
	ProSMG:                "pro_smg",   // carbine?
	ProtoSyringe:          "proto_syringe",
	Quickiebomb:           "quickiebomb_launcher",
	Rainblower:            "rainblower",
	RescueRanger:          "rescue_ranger",
	ReserveShooter:        "reserve_shooter",
	Revolver:              "revolver",
	Sandman:               "sandman",
	Sapper:                "obj_attachment_sapper",
	Scattergun:            "scattergun",
	ScorchShot:            "scorch_shot",
	ScottishResistance:    "sticky_resistance",
	Sentry1:               "obj_sentrygun",
	Sentry2:               "obj_sentrygun2",
	Sentry3:               "obj_sentrygun3",
	SharpDresser:          "sharp_dresser",
	ShootingStar:          "shooting_star",
	ShortStop:             "shortstop",
	ShotgunPrimary:        "shotgun_primary",
	ShotgunPyro:           "shotgun_pyro",
	ShotgunSoldier:        "shotgun_soldier",
	Sledgehammer:          "sledgehammer",
	SMG:                   "smg",
	SniperRifle:           "sniperrifle",
	SodaPopper:            "soda_popper",
	Spycicle:              "spy_cicle",
	SydneySleeper:         "sydney_sleeper",
	SyringeGun:            "syringegun_medic",
	TauntMedic:            "taunt_medic",
	Telefrag:              "telefrag",
	TheClassic:            "the_classic",
	Tomislav:              "tomislav",
	Ubersaw:               "ubersaw",
	WarriorsSpirit:        "warrior_spirit",
	WidowMaker:            "widowmaker",
	World:                 "world",
	Wrangler:              "wrangler_kill",
	WrapAssassin:          "wrap_assassin",
	Wrench:                "wrench",
}

//goland:noinspection GoUnnecessarilyExportedIdentifiers,GoUnusedGlobalVariable
var Weapons = map[PlayerClass][]Weapon{
	Multi: {
		ConscientiousObjector,
		FryingPan,
		HamShank,
		NecroSmasher,
		Pistol,
		ReserveShooter,
		Telefrag,
		World,
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
