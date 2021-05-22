package logparse

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
	X int64
	Y int64
	Z int64
}

// AmmoPack is used for
//goland:noinspection GoUnnecessarilyExportedIdentifiers
type AmmoPack int

//goland:noinspection GoUnnecessarilyExportedIdentifiers
const (
	AmmoSmall AmmoPack = iota
	AmmoMedium
	AmmoLarge
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

// HealthPack contains the 3 types of health packs
//goland:noinspection GoUnnecessarilyExportedIdentifiers
type HealthPack int

//goland:noinspection GoUnnecessarilyExportedIdentifiers
const (
	HPSmall HealthPack = iota
	HPMedium
	HPLarge
)

// Weapon defines all known weapons
//goland:noinspection GoUnnecessarilyExportedIdentifiers
type Weapon string

//goland:noinspection GoUnnecessarilyExportedIdentifiers
const (
	AiFlamethrower        Weapon = "ai_flamethrower"
	Airstrike             Weapon = "airstrike"
	Ambassador            Weapon = "ambassador"
	Amputator             Weapon = "amputator"
	Atomizer              Weapon = "atomizer"
	AwperHand             Weapon = "awper_hand"
	Backburner            Weapon = "backburner"
	BackScratcher         Weapon = "back_scratcher"
	Bat                   Weapon = "bat"
	BazaarBargain         Weapon = "bazaar_bargain"
	BigEarner             Weapon = "big_earner"
	Blackbox              Weapon = "blackbox"
	BlackRose             Weapon = "black_rose"
	Blutsauger            Weapon = "blutsauger"
	Bonesaw               Weapon = "bonesaw"
	Bottle                Weapon = "bottle"
	BrassBeast            Weapon = "brass_beast"
	Bushwacka             Weapon = "bushwacka"
	Caber                 Weapon = "ullapool_caber"
	Club                  Weapon = "club"
	ConscientiousObjector Weapon = "nonnonviolent_protest"
	CowMangler            Weapon = "cow_mangler"
	Crossbow              Weapon = "crusaders_crossbow"
	DeflectPromode        Weapon = "deflect_promode"
	DeflectRocket         Weapon = "deflect_rocket"
	Degreaser             Weapon = "degreaser"
	DemoKatana            Weapon = "demokatana"
	Detonator             Weapon = "detonator"
	DiamondBack           Weapon = "diamondback"
	DirectHit             Weapon = "rocketlauncher_directhit"
	DisciplinaryAction    Weapon = "disciplinary_action"
	DragonsFury           Weapon = "dragons_fury"
	DragonsFuryBonus      Weapon = "dragons_fury_bonus"
	Enforcer              Weapon = "enforcer"
	EscapePlan            Weapon = "unique_pickaxe_escape"
	EternalReward         Weapon = "eternal_reward"
	FamilyBusiness        Weapon = "family_business"
	Fists                 Weapon = "fists"
	FistsOfSteel          Weapon = "steel_fists"
	FlameThrower          Weapon = "flamethrower"
	FlareGun              Weapon = "flaregun"
	ForceANature          Weapon = "force_a_nature"
	FrontierJustice       Weapon = "frontier_justice"
	FryingPan             Weapon = "fryingpan"
	Gunslinger            Weapon = "robot_arm"
	GunslingerCombo       Weapon = "robot_arm_combo_kill" // what is this?
	GunslingerTaunt       Weapon = "robot_arm_blender_kill"
	HamShank              Weapon = "ham_shank"
	HotHand               Weapon = "hot_hand"
	Huntsman              Weapon = "tf_projectile_arrow"
	IronBomber            Weapon = "iron_bomber"
	IronCurtain           Weapon = "iron_curtain"
	Jag                   Weapon = "wrench_jag"
	Knife                 Weapon = "knife"
	Kukri                 Weapon = "tribalkukri"
	Kunai                 Weapon = "kunai"
	Letranger             Weapon = "letranger"
	LibertyLauncher       Weapon = "liberty_launcher"
	LockNLoad             Weapon = "loch_n_load"
	LongHeatmaker         Weapon = "long_heatmaker"
	LooseCannon           Weapon = "loose_cannon"
	LooseCannonImpact     Weapon = "loose_cannon_impact"
	Machina               Weapon = "machina"
	MachinaPen            Weapon = "player_penetration"
	MarketGardener        Weapon = "market_gardener"
	Maul                  Weapon = "the_maul"
	MiniGun               Weapon = "minigun"
	MiniSentry            Weapon = "obj_minisentry"
	Natascha              Weapon = "natascha"
	NecroSmasher          Weapon = "necro_smasher"
	Original              Weapon = "quake_rl"
	PepPistol             Weapon = "pep_pistol"
	Phlog                 Weapon = "phlogistinator"
	Pistol                Weapon = "pistol"
	PistolScout           Weapon = "pistol_scout"
	Powerjack             Weapon = "powerjack"
	ProjectilePipe        Weapon = "tf_projectile_pipe"
	ProjectilePipeRemote  Weapon = "tf_projectile_pipe_remote"
	ProjectileRocket      Weapon = "tf_projectile_rocket"
	ProRifle              Weapon = "pro_rifle" // heatmaker?
	ProSMG                Weapon = "pro_smg"   // carbine?
	ProtoSyringe          Weapon = "proto_syringe"
	Quickiebomb           Weapon = "quickiebomb_launcher"
	Rainblower            Weapon = "rainblower"
	RescueRanger          Weapon = "rescue_ranger"
	ReserveShooter        Weapon = "reserve_shooter"
	Revolver              Weapon = "revolver"
	Sandman               Weapon = "sandman"
	Scattergun            Weapon = "scattergun"
	ScorchShot            Weapon = "scorch_shot"
	ScottishResistance    Weapon = "sticky_resistance"
	Sentry1               Weapon = "obj_sentrygun"
	Sentry2               Weapon = "obj_sentrygun2"
	Sentry3               Weapon = "obj_sentrygun3"
	SharpDresser          Weapon = "sharp_dresser"
	ShootingStar          Weapon = "shooting_star"
	ShortStop             Weapon = "shortstop"
	ShotgunPrimary        Weapon = "shotgun_primary"
	ShotgunPyro           Weapon = "shotgun_pyro"
	ShotgunSoldier        Weapon = "shotgun_soldier"
	Sledgehammer          Weapon = "sledgehammer"
	SMG                   Weapon = "smg"
	SniperRifle           Weapon = "sniperrifle"
	SodaPopper            Weapon = "soda_popper"
	Spycicle              Weapon = "spy_cicle"
	SydneySleeper         Weapon = "sydney_sleeper"
	SyringeGun            Weapon = "syringegun_medic"
	TauntMedic            Weapon = "taunt_medic"
	Telefrag              Weapon = "telefrag"
	TheClassic            Weapon = "the_classic"
	Tomislav              Weapon = "tomislav"
	Ubersaw               Weapon = "ubersaw"
	WarriorsSpirit        Weapon = "warrior_spirit"
	WidowMaker            Weapon = "widowmaker"
	World                 Weapon = "world"
	Wrangler              Weapon = "wrangler_kill"
	WrapAssassin          Weapon = "wrap_assassin"
	Wrench                Weapon = "wrench"
)

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
		SharpDresser,
		Spycicle,
	},
}
