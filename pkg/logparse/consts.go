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

	// WRoundSetupEnd        EventType = 109.

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
	MapStarted       EventType = 1011

	VoteDetails EventType = 1100
	VoteSuccess EventType = 1101
	VoteFailed  EventType = 1102
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
	Sniper
	Soldier
	Demo
	Medic
	Heavy
	Pyro
	Spy
	Engineer
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

type Weapon string

const (
	AiFlamethrower           Weapon = "ai_flamethrower"
	Airstrike                Weapon = "airstrike"
	Ambassador               Weapon = "ambassador"
	Amputator                Weapon = "amputator"
	ApocoFists               Weapon = "apocofists"
	ApSap                    Weapon = "psapper"
	Atomizer                 Weapon = "atomizer"
	AwperHand                Weapon = "awper_hand"
	Axtinguisher             Weapon = "axtinguisher"
	BabyFaceBlaster          Weapon = "pep_brawlerblaster"
	BackScatter              Weapon = "back_scatter"
	BackScratcher            Weapon = "back_scratcher"
	Backburner               Weapon = "backburner"
	Bat                      Weapon = "bat"
	BatOuttaHell             Weapon = "skullbat"
	BatSaber                 Weapon = "batsaber"
	BatSpell                 Weapon = "tf_projectile_spellbats"
	BazaarBargain            Weapon = "bazaar_bargain"
	BeggarsBazooka           Weapon = "dumpster_device"
	BigEarner                Weapon = "big_earner"
	BigKill                  Weapon = "samrevolver"
	BlackRose                Weapon = "black_rose"
	BlackBox                 Weapon = "blackbox"
	BleedKill                Weapon = "bleed_kill"
	Blutsauger               Weapon = "blutsauger"
	Bonesaw                  Weapon = "bonesaw"
	BostonBasher             Weapon = "boston_basher"
	Bottle                   Weapon = "bottle"
	BoxingGloveSpell         Weapon = "tf_projectile_spellkartorb"
	BuffBanner               Weapon = "buff_item"
	BrassBeast               Weapon = "brass_beast"
	BreadBite                Weapon = "bread_bite"
	BuildingCarriedDestroyed Weapon = "building_carried_destroyed"
	Bushwacka                Weapon = "bushwacka"
	Caber                    Weapon = "ullapool_caber"
	CaberExplosion           Weapon = "ullapool_caber_explosion" //nolint:gosec
	CandyCane                Weapon = "candy_cane"
	CharginTarge             Weapon = "demoshield"
	ClaidheamhMor            Weapon = "claidheamohmor"
	Club                     Weapon = "club"
	ConscientiousObjector    Weapon = "nonnonviolent_protest"
	CowMangler               Weapon = "cow_mangler"
	Crocodile                Weapon = "crocodile"
	Crossbow                 Weapon = "crusaders_crossbow"
	CrossbowBolt             Weapon = "tf_projectile_healing_bolt"
	CrossingGuard            Weapon = "crossing_guard"
	DeflectArrow             Weapon = "deflect_arrow"
	DeflectFlare             Weapon = "deflect_flare"
	DeflectFlareDetonator    Weapon = "deflect_flare_detonator"
	DeflectGrenade           Weapon = "deflect_promode"
	DeflectHunstmanBurning   Weapon = "deflect_huntsman_flyingburn"
	DeflectLooseCannon       Weapon = "loose_cannon_reflect"
	DeflectRescueRanger      Weapon = "deflect_rocket"
	DeflectRocket            Weapon = "rescue_ranger_reflect"
	DeflectRocketMangler     Weapon = "tf_projectile_energy_ball"
	DeflectSticky            Weapon = "deflect_sticky"
	Degreaser                Weapon = "degreaser"
	DemoKatana               Weapon = "demokatana"
	Detonator                Weapon = "detonator"
	Diamondback              Weapon = "diamondback"
	DirectHit                Weapon = "rocketlauncher_directhit"
	DisciplinaryAction       Weapon = "disciplinary_action"
	Dispenser                Weapon = "dispenser"
	DragonsFury              Weapon = "dragons_fury"
	DragonsFuryBonus         Weapon = "dragons_fury_bonus"
	Enforcer                 Weapon = "enforcer"
	EntBonesaw               Weapon = "tf_weapon_bonesaw"
	EntBuilder               Weapon = "builder" // ?
	EntFrontierKill          Weapon = "frontier_kill"
	EntManmelter             Weapon = "tf_weapon_flaregun_revenge" // Fire suck extinguish
	EntPickaxe               Weapon = "pickaxe"
	EntSniperRifle           Weapon = "tf_weapon_sniperrifle"
	Equalizer                Weapon = "unique_pickaxe"
	EscapePlan               Weapon = "unique_pickaxe_escape"
	EternalReward            Weapon = "eternal_reward"
	EurekaEffect             Weapon = "eureka_effect"
	EvictionNotice           Weapon = "eviction_notice"
	Eyelander                Weapon = "sword"
	FamilyBusiness           Weapon = "family_business"
	FanOWar                  Weapon = "warfan"
	FireAxe                  Weapon = "fireaxe"
	Fists                    Weapon = "fists"
	FistsOfSteel             Weapon = "steel_fists"
	FlameThrower             Weapon = "flamethrower"
	Flare                    Weapon = "tf_projectile_flare"
	FlareGun                 Weapon = "flaregun"
	FlyingGuillotine         Weapon = "guillotine"
	ForceANature             Weapon = "force_a_nature"
	FortifiedCompound        Weapon = "compound_bow"
	FreedomStaff             Weapon = "freedom_staff"
	FrontierJustice          Weapon = "frontier_justice"
	FryingPan                Weapon = "fryingpan"
	GasBlast                 Weapon = "gas_blast"
	GoldenFryingPan          Weapon = "golden_fryingpan"
	GRU                      Weapon = "gloves_running_urgently"
	GasPasser                Weapon = "tf_weapon_jar_gas" //nolint:gosec
	GigerCounter             Weapon = "giger_counter"
	GoldenWrench             Weapon = "wrench_golden"
	GrenadeLauncher          Weapon = "grenadelauncher"
	Gunslinger               Weapon = "robot_arm"
	GunslingerCombo          Weapon = "robot_arm_combo_kill"
	GunslingerKill           Weapon = "robot_arm_kill"
	HHHHeadtaker             Weapon = "headtaker"
	HamShank                 Weapon = "ham_shank"
	HolidayPunch             Weapon = "holiday_punch"
	HolyMackerel             Weapon = "holymackerel"
	HotHand                  Weapon = "hot_hand"
	Huntsman                 Weapon = "huntsman"
	IronBomber               Weapon = "iron_bomber"
	IronCurtain              Weapon = "iron_curtain"
	Jag                      Weapon = "wrench_jag"
	JarBased                 Weapon = "tf_weapon_jar"
	Jarate                   Weapon = "tf_projectile_jar"
	JetpackStomp             Weapon = "rocketpack_stomp"
	KGB                      Weapon = "gloves"
	Knife                    Weapon = "knife"
	Kukri                    Weapon = "kukri"
	Kunai                    Weapon = "kunai"
	Letranger                Weapon = "letranger"
	LibertyLauncher          Weapon = "liberty_launcher"
	LightningOrbSpell        Weapon = "tf_projectile_lightningorb"
	LockNLoad                Weapon = "loch_n_load"
	Lollichop                Weapon = "lollichop"
	LongHeatmaker            Weapon = "long_heatmaker"
	LooseCannon              Weapon = "loose_cannon"
	LooseCannonExplosion     Weapon = "loose_cannon_explosion"
	LooseCannonImpact        Weapon = "loose_cannon_impact"
	Lugermorph               Weapon = "lunchbox"
	Lunchbox                 Weapon = "maxgun"
	Machina                  Weapon = "machina"
	MachinaPen               Weapon = "player_penetration"
	MadMilk                  Weapon = "tf_projectile_jar_milk"
	Manmelter                Weapon = "manmelter"
	Mantreads                Weapon = "mantreads"
	MarketGardener           Weapon = "market_gardener"
	Maul                     Weapon = "the_maul"
	Medigun                  Weapon = "medigun"
	MeteorShowerSpell        Weapon = "tf_projectile_spellmeteorshower"
	Minigun                  Weapon = "minigun"
	MiniSentry               Weapon = "obj_minisentry"
	Natascha                 Weapon = "natascha"
	NecroSmasher             Weapon = "necro_smasher"
	Needle                   Weapon = "tf_projectile_syringe"
	NeonAnnihilator          Weapon = "annihilator"
	NessiesNineIron          Weapon = "nessieclub"
	Original                 Weapon = "quake_rl"
	OverdoseSyringe          Weapon = "proto_syringe"
	PDAEngineer              Weapon = "pda_engineer"
	PainTrain                Weapon = "paintrain"
	PanicAttack              Weapon = "panic_attack"
	PersianPersuader         Weapon = "persian_persuader"
	Phlog                    Weapon = "phlogistinator"
	PipebombLauncher         Weapon = "pipebomblauncher" //
	PistolEngy               Weapon = "pistol"
	PistolScout              Weapon = "pistol_scout"
	Player                   Weapon = "player" // Finish off player
	Pomson                   Weapon = "pomson"
	PostalPummeler           Weapon = "mailbox"
	Powerjack                Weapon = "powerjack"
	PrettyBoysPocketPistol   Weapon = "pep_pistol"
	Prinny                   Weapon = "prinny_machete"
	ProRifle                 Weapon = "pro_rifle"
	ProSMG                   Weapon = "pro_smg"
	ProjectileArrow          Weapon = "tf_projectile_arrow"
	ProjectileArrowFire      Weapon = "tf_projectile_arrow_fire"
	ProjectileDragonsFury    Weapon = "tf_projectile_balloffire"
	ProjectileGrenade        Weapon = "tf_projectile_pipe"
	ProjectileJarGas         Weapon = "jar_gas"
	ProjectileRocket         Weapon = "tf_projectile_rocket"
	ProjectileShortCircuit   Weapon = "tf_projectile_mechanicalarmorb"
	ProjectileSticky         Weapon = "tf_projectile_pipe_remote"
	ProjectileWrapAssassin   Weapon = "tf_projectile_ball_ornament"
	PumpkinBomb              Weapon = "tf_pumpkin_bomb"
	Quickiebomb              Weapon = "quickiebomb_launcher"
	Rainblower               Weapon = "rainblower"
	RedTapeRecorder          Weapon = "recorder"
	RescueRanger             Weapon = "rescue_ranger"
	ReserveShooter           Weapon = "reserve_shooter"
	Revolver                 Weapon = "revolver"
	RighteousBison           Weapon = "righteous_bison"
	RocketLauncher           Weapon = "rocketlauncher"
	SMG                      Weapon = "smg"
	Sandman                  Weapon = "sandman"
	SandmanBall              Weapon = "ball"
	Sapper                   Weapon = "obj_attachment_sapper"
	Saxxy                    Weapon = "saxxy"
	Scattergun               Weapon = "scattergun"
	ScorchShot               Weapon = "scorch_shot"
	ScotsmansSkullcutter     Weapon = "battleaxe"
	ScottishHandshake        Weapon = "scotland_shard"
	ScottishResistance       Weapon = "sticky_resistance"
	Sentry1                  Weapon = "obj_sentrygun"
	Sentry2                  Weapon = "obj_sentrygun2"
	Sentry3                  Weapon = "obj_sentrygun3"
	SentryRocket             Weapon = "tf_projectile_sentryrocket"
	Shahanshah               Weapon = "shahanshah"
	Shark                    Weapon = "shark"
	SharpDresser             Weapon = "sharp_dresser"
	SharpenedVolcanoFragment Weapon = "lava_axe"
	ShootingStar             Weapon = "shooting_star"
	ShortCircuit             Weapon = "short_circuit"
	Shortstop                Weapon = "shortstop"
	ShotgunEngy              Weapon = "shotgun_primary"
	ShotgunHeavy             Weapon = "shotgun_hwg"
	ShotgunPyro              Weapon = "shotgun_pyro"
	ShotgunSoldier           Weapon = "shotgun_soldier"
	Shovel                   Weapon = "shovel"
	SkeletonSpawnSpell       Weapon = "tf_projectile_spellspawnzombie"
	Sledgehammer             Weapon = "sledgehammer"
	SnackAttack              Weapon = "snack_attack"
	SniperRifle              Weapon = "sniperrifle"
	SodaPopper               Weapon = "soda_popper"
	SolemnVow                Weapon = "solemn_vow"
	SouthernComfort          Weapon = "southern_comfort_kill"
	SouthernHospitality      Weapon = "southern_hospitality"
	SplendidScreen           Weapon = "splendid_screen"
	Spycicle                 Weapon = "spy_cicle"
	SuicideWeapon            Weapon = "suicide"
	SunOnAStick              Weapon = "lava_bat"
	SydneySleeper            Weapon = "sydney_sleeper"
	SyringeGun               Weapon = "syringegun_medic"
	TFFlameThrower           Weapon = "tf_weapon_flamethrower"
	TFMedigun                Weapon = "tf_weapon_medigun"
	TauntArmageddon          Weapon = "armageddon"
	TauntDemoman             Weapon = "taunt_demoman"
	TauntEngineer            Weapon = "taunt_engineer"
	TauntGuitarKill          Weapon = "taunt_guitar_kill"
	TauntGunslinger          Weapon = "robot_arm_blender_kill"
	TauntHeavy               Weapon = "taunt_heavy"
	TauntMedic               Weapon = "taunt_medic"
	TauntPyro                Weapon = "taunt_pyro"
	TauntScout               Weapon = "taunt_scout"   // Sandman
	TauntSniper              Weapon = "taunt_sniper"  // huntsman
	TauntSoldier             Weapon = "taunt_soldier" // Equalizer
	TauntSoldierLumbricus    Weapon = "taunt_soldier_lumbricus"
	TauntSpy                 Weapon = "taunt_spy" // knife skewer
	Telefrag                 Weapon = "telefrag"
	TheCAPPER                Weapon = "the_capper"
	TheClassic               Weapon = "the_classic"
	TheWinger                Weapon = "the_winger"
	ThirdDegree              Weapon = "thirddegree"
	ThreeRuneBlade           Weapon = "scout_sword"
	TideTurner               Weapon = "tide_turner"
	Tomislav                 Weapon = "tomislav"
	TribalmansShiv           Weapon = "tribalkukri"
	Ubersaw                  Weapon = "ubersaw"
	UnarmedCombat            Weapon = "unarmed_combat"
	UnknownWeapon            Weapon = "unknown"
	VitaSaw                  Weapon = "battleneedle"
	WangaPrick               Weapon = "voodoo_pin"
	WarriorsSpirit           Weapon = "warrior_spirit"
	Widowmaker               Weapon = "widowmaker"
	World                    Weapon = "world"
	Wrangler                 Weapon = "wrangler_kill"
	WrapAssassin             Weapon = "wrap_assassin"
	Wrench                   Weapon = "wrench"
)
