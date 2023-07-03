// Package logparse provides functionality for parsing TF2 console logs into known events and values.
//
// It should be able to parse logs from servers using SupStats2 & MedicStats plugins. These are the same requirements
// as logs.tf, so you should be able to download and parse them without much trouble.
package logparse

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

type parserType struct {
	Rx   *regexp.Regexp
	Type EventType
}

type LogParser struct {
	rxKVPairs *regexp.Regexp
	// Common player id format eg: "Name<382><STEAM_0:1:22649331><>".
	rxPlayer    *regexp.Regexp
	rxUnhandled *regexp.Regexp
	rxParsers   []parserType
	weapons     *WeaponParser
}

type WeaponParser struct {
	weapons     map[PlayerClass][]Weapon
	weaponNames map[Weapon]string
}

func (w *WeaponParser) Name(weapon Weapon) string {
	name, found := w.weaponNames[weapon]
	if !found {
		return w.weaponNames[UnknownWeapon]
	}

	return name
}

func (w *WeaponParser) Parse(s string) Weapon {
	for weaponName, v := range w.weaponNames {
		if v == s {
			return weaponName
		}
	}

	return UnknownWeapon
}

func NewWeaponParser() *WeaponParser {
	return &WeaponParser{
		weaponNames: map[Weapon]string{
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
		},
		weapons: map[PlayerClass][]Weapon{
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
		},
	}
}

func New() *LogParser {
	return &LogParser{
		rxKVPairs: regexp.MustCompile(`\((?P<key>.+?)\s+"(?P<value>.+?)"\)`),
		// Common player id format eg: "Name<382><STEAM_0:1:22649331><>".
		rxUnhandled: regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+`),
		rxPlayer:    regexp.MustCompile(`"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"`),
		weapons:     NewWeaponParser(),
		// Map matching regex to known event types.
		//nolint:lll
		rxParsers: []parserType{
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+[Ll]og file started\s+(?P<keypairs>.+?)$`), LogStart},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+[Ll]og file closed.$`), LogStop},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+server_cvar:\s+"(?P<CVAR>.+?)"\s"(?P<value>.+?)"$`), CVAR},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+[Rr][Cc][Oo][Nn] from "(?P<ip>.+?)": command "(?P<cmd>.+?)"$`), RCON},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "shot_fired"\s+(?P<keypairs>.+?)$`), ShotFired},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "shot_hit"\s+(?P<keypairs>.+?)$`), ShotHit},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "[dD]amage" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>"\s+(?P<keypairs>.+?)$`), Damage},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "[dD]amage" \(damage "(?P<damage>\d+)"\)`), Damage},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+killed "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>" with "(?P<weapon>.+?)"\s+(\(customkill "(?P<customkill>.+?)"\))\s+(?P<keypairs>.+?)$`), KilledCustom}, // Must come before Killed
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+killed "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>" with "(?P<weapon>.+?)"\s+(?P<keypairs>.+?)$`), Killed},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "[hH]ealed" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>"\s+(?P<keypairs>.+?)$`), Healed},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "kill assist" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>"\s+(?P<keypairs>.+?)$`), KillAssist},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+picked up item "(?P<item>\S+)"\s+(?P<keypairs>.+?)$`), Pickup},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+picked up item "(?P<item>\S+)"`), Pickup},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+spawned as "(?P<class>\S+)"$`), SpawnedAs},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+STEAM USERID [vV]alidated$`), Validated},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+[Cc]onnected, address(\s"(?P<address>.+?)")?$`), Connected},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+[Ee]ntered the game$`), Entered},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+joined team "(?P<new_team>(Red|Blue|Spectator|Unassigned))"$`), JoinedTeam},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+changed role to "(?P<class>.+?)"`), ChangeClass},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+committed suicide with "(?P<weapon>.+?)"\s+(?P<keypairs>.+?)$`), Suicide},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "chargeready"`), ChargeReady},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "chargedeployed"( \(medigun "(?P<medigun>.+?)"\))?`), ChargeDeployed},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "chargeended" \(duration "(?P<duration>.+?)"\)`), ChargeEnded},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "[Dd]omination" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Red|Blue)?)>"`), Domination},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "[Rr]evenge" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>"\s?(\(assist "(?P<assist>\d+)"\))?`), Revenge},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+say\s+"(?P<msg>.+?)"$`), Say},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+say_team\s+"(?P<msg>.+?)"$`), SayTeam},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "empty_uber"`), EmptyUber},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "lost_uber_advantage"\s+(?P<keypairs>.+?)$`), LostUberAdv},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "medic_death" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue)?)>"\s+(?P<keypairs>.+?)$`), MedicDeath},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "medic_death_ex"\s+(?P<keypairs>.+?)$`), MedicDeathEx},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "player_extinguished" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Red|Blue)?)>" with "(?P<weapon>.+?)"\s+(?P<keypairs>.+?)$`), Extinguished},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "player_builtobject"\s+(?P<keypairs>.+?)$`), BuiltObject},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "player_carryobject"\s+(?P<keypairs>.+?)$`), CarryObject},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "player_dropobject"\s+(?P<keypairs>.+?)$`), DropObject},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "killedobject"\s+(?P<keypairs>.+?)$`), KilledObject},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "killedobject"\s+(?P<keypairs>.+?)$`), KilledObject},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "object_detonated"\s+(?P<keypairs>.+?)$`), DetonatedObject},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "first_heal_after_spawn"\s+(?P<keypairs>.+?)$`), FirstHealAfterSpawn},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+Team "(?P<team>.+?)" triggered "pointcaptured"\s+(?P<keypairs>.+?)$`), PointCaptured},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "captureblocked"\s+(?P<keypairs>.+?)$`), CaptureBlocked},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+[Dd]isconnected \(reason "(?P<reason>.+?)$`), Disconnected},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+World triggered "Round_Overtime"`), WRoundOvertime},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+World triggered "Round_Start"`), WRoundStart},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+World triggered "Round_Setup_End"`), WRoundStart},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+World triggered "Round_Win"\s+(?P<keypairs>.+?)$`), WRoundWin},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+World triggered "Round_Length"\s+(?P<keypairs>.+?)$`), WRoundLen},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+World triggered "Game_Over" reason "(?P<reason>.+?)"`), WGameOver},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+Team "(?P<team>Red|Blue)" current score "(?P<score>\d+)" with "(?P<players>\d+)" players`), WTeamScore},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+Team "(?P<team>Red|Blue)" final score "(?P<score>\d+)" with "(?P<players>\d+)" players`), WTeamFinalScore},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+World triggered "Game_Paused"`), WPaused},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+World triggered "Game_Unpaused"`), WResumed},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+Loading map "(?P<map>.+?)"$`), MapLoad},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+Executing dedicated server config file (?P<config>.+?)$`), ServerConfigExec},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+STEAMAUTH: (?P<reason>.+?)$`), SteamAuth},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "jarate_attack" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue)?)>" with "(?P<weapon>.+?)"\s+(?P<keypairs>.+?)$`), JarateAttack},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "milk_attack" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue)?)>" with "(?P<weapon>.+?)"\s+(?P<keypairs>.+?)$`), MilkAttack},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"\s+triggered "gas_attack" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue)?)>" with "(?P<weapon>.+?)"\s+(?P<keypairs>.+?)$`), GasAttack},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+World triggered "Mini_Round_Win"\s+(?P<keypairs>.+?)$`), WMiniRoundWin},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+World triggered "Mini_Round_Length"\s+(?P<keypairs>.+?)$`), WMiniRoundLen},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+World triggered "Round_Setup_Begin"`), WRoundSetupBegin},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+World triggered "Mini_Round_Selected"\s+(?P<keypairs>.+?)$`), WMiniRoundSelected},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+World triggered "Mini_Round_Start"`), WMiniRoundStart},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+"(.+?)"\s=\s"(.+?)"$`), IgnoredMsg},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+server cvars start`), IgnoredMsg},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+\[META]`), IgnoredMsg},
			{regexp.MustCompile(`^L\s(?P<created_on>.+?):\s+Team\s"(?P<team>RED|BLUE)"\striggered\s"Intermission_Win_Limit"$`), WIntermissionWinLimit},
		},
	}
}

func parsePickupItem(hp string, item *PickupItem) bool {
	switch hp {
	case "ammopack_small":
		fallthrough
	case "tf_ammo_pack":
		*item = ItemAmmoSmall
	case "ammopack_medium":
		*item = ItemAmmoMedium
	case "ammopack_large":
		*item = ItemAmmoLarge
	case "medkit_small":
		*item = ItemHPSmall
	case "medkit_medium":
		*item = ItemHPMedium
	case "medkit_large":
		*item = ItemHPLarge
	default:
		return false
	}

	return true
}

func parseMedigun(gunStr string, gun *MedigunType) bool {
	switch strings.ToLower(gunStr) {
	case "medigun":
		*gun = Uber
	case "kritzkrieg":
		*gun = Kritzkrieg
	case "vaccinator":
		*gun = Vaccinator
	case "quickfix":
		*gun = QuickFix
	default:
		return false
	}

	return true
}

//
// func playerClassStr(cls Class) string {
//	switch cls {
//	case Scout:
//		return "Scout"
//	case Soldier:
//		return "Soldier"
//	case Demo:
//		return "Demo"
//	case Pyro:
//		return "Pyro"
//	case Heavy:
//		return "Heavy"
//	case Engineer:
//		return "Engineer"
//	case Medic:
//		return "Medic"
//	case Sniper:
//		return "Sniper"
//	case Spy:
//		return "Spy"
//	default:
//		return "Spectator"
//	}
//}

func parsePlayerClass(classStr string, class *PlayerClass) bool {
	switch strings.ToLower(classStr) {
	case "scout":
		*class = Scout
	case "soldier":
		*class = Soldier
	case "pyro":
		*class = Pyro
	case "demoman":
		*class = Demo
	case "heavyweapons":
		*class = Heavy
	case "engineer":
		*class = Engineer
	case "medic":
		*class = Medic
	case "sniper":
		*class = Sniper
	case "spy":
		*class = Spy
	case "spectator":
		fallthrough
	case "undefined":
		fallthrough
	case "spec":
		*class = Spectator
	default:
		return false
	}

	return true
}

func parseTeam(teamStr string, team *Team) bool {
	switch strings.ToLower(teamStr) {
	case "red":
		*team = RED
	case "blue":
		fallthrough
	case "blu":
		*team = BLU
	case "unknown":
		fallthrough
	case "unassigned":
		fallthrough
	case "spectator":
		fallthrough
	case "spec":
		*team = SPEC
	default:
		return false
	}

	return true
}

func reSubMatchMap(regex *regexp.Regexp, str string) (map[string]any, bool) {
	match := regex.FindStringSubmatch(str)
	if match == nil {
		return nil, false
	}

	subMatchMap := make(map[string]any)

	for i, name := range regex.SubexpNames() {
		if i != 0 {
			subMatchMap[name] = match[i]
		}
	}

	return subMatchMap, true
}

func ParsePos(posStr string, pos *Pos) bool {
	pieces := strings.SplitN(posStr, " ", 3)
	if len(pieces) != 3 {
		return false
	}

	posX, errParseX := strconv.ParseFloat(pieces[0], 64)
	if errParseX != nil {
		return false
	}

	posY, errParseY := strconv.ParseFloat(pieces[1], 64)
	if errParseY != nil {
		return false
	}

	posZ, errParseZ := strconv.ParseFloat(pieces[2], 64)
	if errParseZ != nil {
		return false
	}

	pos.X = posX
	pos.Y = posY
	pos.Z = posZ

	return true
}

func ParseSourcePlayer(srcStr string, player *SourcePlayer) bool {
	rxPlayer := regexp.MustCompile(`"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"`)

	ooKV, ok := reSubMatchMap(rxPlayer, "\""+srcStr+"\"")
	if !ok {
		return false
	}

	nameVal, nameOk := ooKV["name"].(string)
	if !nameOk {
		return false
	}

	player.Name = nameVal

	pidVal, pidOk := ooKV["pid"].(string)
	if !pidOk {
		return false
	}

	pid, errPid := strconv.ParseInt(pidVal, 10, 32)
	if errPid != nil {
		return false
	}

	player.PID = int(pid)

	var team Team

	teamVal, teamOk := ooKV["team"].(string)
	if !teamOk {
		return false
	}

	if !parseTeam(teamVal, &team) {
		return false
	}

	player.Team = team

	sidStr, sidOk := ooKV["sid"].(string)
	if !sidOk {
		return false
	}

	player.SID = steamid.SID3ToSID64(steamid.SID3(sidStr))

	return true
}

func ParseDateTime(dateStr string, outTime *time.Time) bool {
	parsed, errParseTime := time.Parse("01/02/2006 - 15:04:05", dateStr)
	if errParseTime != nil {
		return false
	}

	*outTime = parsed

	return true
}

func (p *LogParser) ParseKVs(stringVal string, out map[string]any) bool {
	matches := p.rxKVPairs.FindAllStringSubmatch(stringVal, 10)
	if len(matches) == 0 {
		return false
	}

	for match := range matches {
		out[matches[match][1]] = matches[match][2]
	}

	return true
}

func (p *LogParser) processKV(originalKVMap map[string]any) map[string]any {
	newKVMap := map[string]any{}

	for key, origValue := range originalKVMap {
		value, castOk := origValue.(string)
		if !castOk {
			continue
		}

		switch key {
		case "created_on":
			var t time.Time
			if ParseDateTime(value, &t) {
				newKVMap["created_on"] = t
			}
		case "medigun":
			var medigun MedigunType
			if parseMedigun(value, &medigun) {
				newKVMap["medigun"] = medigun
			}
		case "crit":
			switch value {
			case "crit":
				newKVMap["crit"] = Crit
			case "mini":
				newKVMap["crit"] = Mini
			default:
				newKVMap["crit"] = NonCrit
			}
		case "reason":
			// Some reasons get output with a newline, so it gets these uneven line endings
			reason := value
			newKVMap["reason"] = strings.TrimSuffix(reason, `")`)
		case "objectowner":
			ooKV, matchOk := reSubMatchMap(p.rxPlayer, "\""+value+"\"")
			if matchOk {
				// TODO Make this less static to support >2 targets for events like capping points?
				for keyVal, val := range ooKV {
					newKVMap[keyVal+"2"] = val
				}
			}
		case "address":
			// Split newKVMap client port for easier queries
			pieces := strings.Split(value, ":")
			if len(pieces) != 2 {
				newKVMap[key] = value

				continue
			}

			newKVMap["address"] = pieces[0]
			newKVMap["port"] = pieces[1]
		default:
			newKVMap[key] = value
		}
	}

	return newKVMap
}

// Results hold the  results of parsing a log line.
type Results struct {
	EventType EventType
	Event     any
}

// Parse will parse the log line into a known type and values.
//
//nolint:gocognit,funlen,maintidx
func (p *LogParser) Parse(logLine string) (*Results, error) {
	for _, parser := range p.rxParsers {
		matchMap, found := reSubMatchMap(parser.Rx, strings.TrimSuffix(strings.TrimSuffix(logLine, "\n"), "\r"))
		if found {
			value, ok := matchMap["keypairs"].(string)
			if ok {
				p.ParseKVs(value, matchMap)
			}

			// Temporary values
			delete(matchMap, "keypairs")
			delete(matchMap, "")

			var (
				errUnmarshal error
				event        any
				values       = p.processKV(matchMap)
			)

			switch parser.Type {
			case CaptureBlocked:
				var t CaptureBlockedEvt
				if errUnmarshal = p.unmarshal(values, &t); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = t
			case LogStart:
				var t LogStartEvt
				if errUnmarshal = p.unmarshal(values, &t); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = t
			case CVAR:
				var t CVAREvt
				if errUnmarshal = p.unmarshal(values, &t); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = t
			case RCON:
				var t RCONEvt
				if errUnmarshal = p.unmarshal(values, &t); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = t
			case Entered:
				var t EnteredEvt
				if errUnmarshal = p.unmarshal(values, &t); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = t
			case JoinedTeam:
				var t JoinedTeamEvt
				if errUnmarshal = p.unmarshal(values, &t); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = t
			case ChangeClass:
				var t ChangeClassEvt
				if errUnmarshal = p.unmarshal(values, &t); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = t
			case SpawnedAs:
				var t SpawnedAsEvt
				if errUnmarshal = p.unmarshal(values, &t); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = t
			case Suicide:
				var t SuicideEvt
				if errUnmarshal = p.unmarshal(values, &t); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = t
			case WRoundStart:
				var t WRoundStartEvt
				if errUnmarshal = p.unmarshal(values, &t); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = t
			case MedicDeath:
				var t MedicDeathEvt
				if errUnmarshal = p.unmarshal(values, &t); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = t
			case Killed:
				var t KilledEvt
				if errUnmarshal = p.unmarshal(values, &t); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = t
			case KilledCustom:
				var t CustomKilledEvt
				if errUnmarshal = p.unmarshal(values, &t); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = t
			case KillAssist:
				var t KillAssistEvt
				if errUnmarshal = p.unmarshal(values, &t); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = t
			case Healed:
				var t HealedEvt
				if errUnmarshal = p.unmarshal(values, &t); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = t
			case Extinguished:
				var t ExtinguishedEvt
				if errUnmarshal = p.unmarshal(values, &t); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = t
			case PointCaptured:
				var parsedEvent PointCapturedEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case Connected:
				var parsedEvent ConnectedEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case KilledObject:
				var parsedEvent KilledObjectEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case CarryObject:
				var parsedEvent CarryObjectEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case DetonatedObject:
				var parsedEvent DetonatedObjectEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case DropObject:
				var parsedEvent DropObjectEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case BuiltObject:
				var parsedEvent BuiltObjectEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case WRoundWin:
				var parsedEvent WRoundWinEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case WRoundLen:
				var parsedEvent WRoundLenEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case WTeamScore:
				var parsedEvent WTeamScoreEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case Say:
				var parsedEvent SayEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case SayTeam:
				var parsedEvent SayTeamEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case Domination:
				var parsedEvent DominationEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case Disconnected:
				var parsedEvent DisconnectedEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case Revenge:
				var parsedEvent RevengeEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case WRoundOvertime:
				var parsedEvent WRoundOvertimeEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case WGameOver:
				var parsedEvent WGameOverEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case WTeamFinalScore:
				var parsedEvent WTeamFinalScoreEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case LogStop:
				var parsedEvent LogStopEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case WPaused:
				var parsedEvent WPausedEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case WResumed:
				var parsedEvent WResumedEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case WIntermissionWinLimit:
				var parsedEvent WIntermissionWinLimitEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case FirstHealAfterSpawn:
				var parsedEvent FirstHealAfterSpawnEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case ChargeReady:
				var parsedEvent ChargeReadyEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case ChargeDeployed:
				var parsedEvent ChargeDeployedEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case ChargeEnded:
				var parsedEvent ChargeEndedEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case MedicDeathEx:
				var parsedEvent MedicDeathExEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case LostUberAdv:
				var parsedEvent LostUberAdvantageEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case EmptyUber:
				var parsedEvent EmptyUberEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case Pickup:
				var parsedEvent PickupEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case ShotFired:
				var parsedEvent ShotFiredEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case ShotHit:
				var parsedEvent ShotHitEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case Damage:
				var parsedEvent DamageEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case JarateAttack:
				var parsedEvent JarateAttackEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case WMiniRoundWin:
				var parsedEvent WMiniRoundWinEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case WMiniRoundLen:
				var parsedEvent WMiniRoundLenEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case WRoundSetupBegin:
				var parsedEvent WRoundSetupBeginEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case WMiniRoundSelected:
				var parsedEvent WMiniRoundSelectedEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case WMiniRoundStart:
				var parsedEvent WMiniRoundStartEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case MilkAttack:
				var parsedEvent MilkAttackEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			case GasAttack:
				var parsedEvent GasAttackEvt
				if errUnmarshal = p.unmarshal(values, &parsedEvent); errUnmarshal != nil {
					return nil, errUnmarshal
				}

				event = parsedEvent
			}

			return &Results{parser.Type, event}, nil
		}
	}

	matchMap, found := reSubMatchMap(p.rxUnhandled, logLine)
	if found {
		var parsedEvent IgnoredMsgEvt
		if errUnmarshal := p.unmarshal(matchMap, &parsedEvent); errUnmarshal != nil {
			return nil, errUnmarshal
		}

		parsedEvent.Message = logLine

		return &Results{IgnoredMsg, parsedEvent}, nil
	}

	var parsedEvent UnknownMsgEvt
	if errUnmarshal := p.unmarshal(matchMap, &parsedEvent); errUnmarshal != nil {
		return nil, errUnmarshal
	}

	parsedEvent.Message = logLine

	return &Results{UnknownMsg, parsedEvent}, nil
}

func (p *LogParser) decodeTeam() func(reflect.Type, reflect.Type, any) (any, error) {
	return func(fromType reflect.Type, toType reflect.Type, value any) (any, error) {
		if fromType.Kind() != reflect.String {
			return value, nil
		}

		var team Team

		teamVal, ok := value.(string)
		if !ok {
			return value, nil
		}

		if !parseTeam(teamVal, &team) {
			return value, nil
		}

		return team, nil
	}
}

func (p *LogParser) decodePlayerClass() func(reflect.Type, reflect.Type, any) (any, error) {
	return func(fromType reflect.Type, toType reflect.Type, value any) (any, error) {
		if fromType.Kind() != reflect.String {
			return value, nil
		}

		var playerClass PlayerClass

		pcVal, ok := value.(string)
		if !ok {
			return value, nil
		}

		if !parsePlayerClass(pcVal, &playerClass) {
			return value, nil
		}

		return playerClass, nil
	}
}

func (p *LogParser) decodePos() func(reflect.Type, reflect.Type, any) (any, error) {
	return func(fromType reflect.Type, toType reflect.Type, value any) (any, error) {
		if fromType.Kind() != reflect.String {
			return value, nil
		}

		var pos Pos

		posVal, ok := value.(string)
		if !ok {
			return value, nil
		}

		if !ParsePos(posVal, &pos) {
			return value, nil
		}

		return pos, nil
	}
}

// BotSid Special internal SID used to track bots internally.
const BotSid = 807

func (p *LogParser) decodeSID3() func(reflect.Type, reflect.Type, any) (any, error) {
	return func(fromType reflect.Type, toType reflect.Type, value any) (any, error) {
		if fromType.Kind() != reflect.String {
			return value, nil
		}

		sidVal, ok := value.(string)
		if !ok {
			return value, nil
		}

		if sidVal == "BOT" {
			return BotSid, nil
		}

		if !strings.HasPrefix(sidVal, "[U") {
			return value, nil
		}

		sid64 := steamid.SID3ToSID64(steamid.SID3(sidVal))
		if !sid64.Valid() {
			return value, nil
		}

		return sid64, nil
	}
}

// func decodeMedigun() mapstructure.DecodeHookFunc {
//	return func(f reflect.Type, t reflect.Type, d any) (any, error) {
//		if f.Kind() != reflect.String {
//			return d, nil
//		}
//		var m Medigun
//		if !parseMedigun(d.(string), &m) {
//			return d, nil
//		}
//		return m, nil
//	}
//}

func (p *LogParser) decodePickupItem() func(reflect.Type, reflect.Type, any) (any, error) {
	return func(fromType reflect.Type, toType reflect.Type, value any) (any, error) {
		if fromType.Kind() != reflect.String {
			return value, nil
		}

		var item PickupItem

		itemVal, ok := value.(string)
		if !ok {
			return value, nil
		}

		if !parsePickupItem(itemVal, &item) {
			return value, nil
		}

		return item, nil
	}
}

func (p *LogParser) decodeWeapon() func(reflect.Type, reflect.Type, any) (any, error) {
	return func(fromType reflect.Type, toType reflect.Type, value any) (any, error) {
		if fromType.Kind() != reflect.String {
			return value, nil
		}

		weaponString, ok := value.(string)
		if !ok {
			return value, nil
		}

		weapon := p.weapons.Parse(weaponString)
		if weapon != UnknownWeapon {
			return weapon, nil
		}

		return value, nil
	}
}

func (p *LogParser) decodeTime() func(reflect.Type, reflect.Type, any) (any, error) {
	return func(fromType reflect.Type, toType reflect.Type, value any) (any, error) {
		if fromType.Kind() != reflect.String {
			return value, nil
		}

		var timeValue time.Time

		dateVal, ok := value.(string)
		if !ok {
			return value, nil
		}

		if ParseDateTime(dateVal, &timeValue) {
			return timeValue, nil
		}

		return value, nil
	}
}

// unmarshal will transform a map of values into the struct passed in
// eg: {"sm_nextmap": "pl_frontier_final"} -> CVAREvt
func (p *LogParser) unmarshal(input any, output any) error {
	decoder, errNewDecoder := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			p.decodeTime(),
			p.decodeTeam(),
			p.decodePlayerClass(),
			p.decodePos(),
			p.decodeSID3(),

			p.decodePickupItem(),
			p.decodeWeapon(),
		),
		Result:           output,
		WeaklyTypedInput: true, // Lets us do str -> int easily
		Squash:           true,
	})
	if errNewDecoder != nil {
		return errors.Wrap(errNewDecoder, "Failed to create decoder")
	}

	if errDecode := decoder.Decode(input); errDecode != nil {
		return errors.Wrap(errDecode, "Failed to decode unmarshal input")
	}

	return nil
}

// Pos is a position in 3D space.
type Pos struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// Encode returns an ST_MakePointM
// Uses ESPG 4326 (WSG-84).
func (p *Pos) Encode() string {
	return fmt.Sprintf(`ST_SetSRID(ST_MakePoint(%f, %f, %f), 4326)`, p.Y, p.X, p.Z)
}
