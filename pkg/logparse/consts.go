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

	WRoundOvertime        EventType = 100
	WRoundStart           EventType = 101
	WRoundWin             EventType = 102
	WRoundLen             EventType = 103
	WTeamScore            EventType = 104
	WTeamFinalScore       EventType = 105
	WGameOver             EventType = 106
	WPaused               EventType = 107
	WResumed              EventType = 108
	WRoundSetupEnd        EventType = 109
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

// TODO String()
//
//goland:noinspection GoUnnecessarilyExportedIdentifiers
type Weapon int

//goland:noinspection GoUnnecessarilyExportedIdentifiers
const (
	AiFlamethrower Weapon = iota + 1
	Airstrike
	Ambassador
	Amputator
	ApocoFists
	ApSap
	Atomizer
	AwperHand
	Axtinguisher
	BabyFaceBlaster
	BackScatter
	BackScratcher
	Backburner
	Bat
	BatOuttaHell
	BatSaber
	BatSpell
	BazaarBargain
	BeggarsBazooka
	BigEarner
	BigKill
	BlackRose
	BlackBox
	BleedKill
	Blutsauger
	Bonesaw
	BostonBasher
	Bottle
	BoxingGloveSpell
	BuffBanner
	BrassBeast
	BreadBite
	BuildingCarriedDestroyed
	Bushwacka
	Caber
	CaberExplosion
	CandyCane
	CharginTarge
	ClaidheamhMor
	Club
	ConscientiousObjector
	CowMangler
	Crocodile
	Crossbow
	CrossbowBolt
	CrossingGuard
	DeflectArrow
	DeflectFlare
	DeflectFlareDetonator
	DeflectGrenade
	DeflectHunstmanBurning
	DeflectLooseCannon
	DeflectRescueRanger
	DeflectRocket
	DeflectRocketMangler
	DeflectSticky
	Degreaser
	DemoKatana
	Detonator
	Diamondback
	DirectHit
	DisciplinaryAction
	Dispenser
	DragonsFury
	DragonsFuryBonus
	Enforcer
	EntBonesaw
	EntBuilder // ?
	EntFrontierKill
	EntManmelter // Fire suck extinguish
	EntPickaxe
	EntSniperRifle
	Equalizer
	EscapePlan
	EternalReward
	EurekaEffect
	EvictionNotice
	Eyelander
	FamilyBusiness
	FanOWar
	FireAxe
	Fists
	FistsOfSteel
	FlameThrower
	Flare
	FlareGun
	FlyingGuillotine
	ForceANature
	FortifiedCompound
	FreedomStaff
	FrontierJustice
	FryingPan
	GoldenFryingPan
	GRU
	GasPasser
	GigerCounter
	GoldenWrench
	Gunslinger
	GunslingerCombo
	GunslingerKill
	HHHHeadtaker
	HamShank
	HolidayPunch
	HolyMackerel
	HotHand
	Huntsman
	IronBomber
	IronCurtain
	Jag
	JarBased
	Jarate
	JetpackStomp
	KGB
	Knife
	Kukri
	Kunai
	Letranger
	LibertyLauncher
	LightningOrbSpell
	LockNLoad
	Lollichop
	LongHeatmaker
	LooseCannon
	LooseCannonExplosion
	LooseCannonImpact
	Lugermorph
	Lunchbox
	Machina
	MachinaPen
	MadMilk
	Manmelter
	Mantreads
	MarketGardener
	Maul
	Medigun
	MeteorShowerSpell
	Minigun
	MiniSentry
	Natascha
	NecroSmasher
	Needle
	NeonAnnihilator
	NessiesNineIron
	Original
	OverdoseSyringe
	PDAEngineer
	PainTrain
	PanicAttack
	PersianPersuader
	Phlog
	PipebombLauncher //
	PistolEngy
	PistolScout
	Player // Finish off player
	Pomson
	PostalPummeler
	Powerjack
	PrettyBoysPocketPistol
	Prinny
	ProRifle
	ProSMG
	ProjectileArrow
	ProjectileArrowFire
	ProjectileDragonsFury
	ProjectileGrenade
	ProjectileJarGas
	ProjectileRocket
	ProjectileShortCircuit
	ProjectileSticky
	ProjectileWrapAssassin
	PumpkinBomb
	Quickiebomb
	Rainblower
	RedTapeRecorder
	RescueRanger
	ReserveShooter
	Revolver
	RighteousBison
	RocketLauncher
	SMG
	Sandman
	SandmanBall
	Sapper
	Saxxy
	Scattergun
	ScorchShot
	ScotsmansSkullcutter
	ScottishHandshake
	ScottishResistance
	Sentry1
	Sentry2
	Sentry3
	SentryRocket
	Shahanshah
	Shark
	SharpDresser
	SharpenedVolcanoFragment
	ShootingStar
	ShortCircuit
	Shortstop
	ShotgunEngy
	ShotgunHeavy
	ShotgunPyro
	ShotgunSoldier
	Shovel
	SkeletonSpawnSpell
	Sledgehammer
	SnackAttack
	SniperRifle
	SodaPopper
	SolemnVow
	SouthernComfort
	SouthernHospitality
	SplendidScreen
	Spycicle
	SuicideWeapon
	SunOnAStick
	SydneySleeper
	SyringeGun
	TFFlameThrower
	TFMedigun
	TauntArmageddon
	TauntDemoman
	TauntEngineer
	TauntGuitarKill
	TauntGunslinger
	TauntHeavy
	TauntMedic
	TauntPyro
	TauntScout   // Sandman
	TauntSniper  // huntsman
	TauntSoldier // Equalizer
	TauntSoldierLumbricus
	TauntSpy // knife skewer
	Telefrag
	TheCAPPER
	TheClassic
	TheWinger
	ThirdDegree
	ThreeRuneBlade
	TideTurner
	Tomislav
	TribalmansShiv
	Ubersaw
	UnarmedCombat
	UnknownWeapon
	VitaSaw
	WangaPrick
	WarriorsSpirit
	Widowmaker
	World
	Wrangler
	WrapAssassin
	Wrench
)

func (w Weapon) String() string { //nolint:maintidx
	switch w {
	case AiFlamethrower:
		return "Nostromo Napalmer"
	case Airstrike:
		return "Air Strike"
	case Ambassador:
		return "Ambassador"
	case Amputator:
		return "Amputator"
	case ApocoFists:
		return "Apoco Fists"
	case ApSap:
		return "Ap-Sap"
	case Atomizer:
		return "Atomizer"
	case AwperHand:
		return "Awper and"
	case Axtinguisher:
		return "Axtinguisher"
	case BabyFaceBlaster:
		return "Baby Face's Blaster"
	case BackScatter:
		return "Back Scatter"
	case BackScratcher:
		return "Back Scratcher"
	case Backburner:
		return "Backburner"
	case Bat:
		return "Bat"
	case BatOuttaHell:
		return "Bat Outta Hell"
	case BatSaber:
		return "Bat Saber"
	case BatSpell:
		return "Bat Spell"
	case BazaarBargain:
		return "Bazaar Bargain"
	case BeggarsBazooka:
		return "Beggars Bazooka"
	case BigEarner:
		return "Big Earner"
	case BigKill:
		return "Big Kill"
	case BlackRose:
		return "Black Rose"
	case BlackBox:
		return "Black Box"
	case BleedKill:
		return "Bleed"
	case Blutsauger:
		return "Blutsauger"
	case Bonesaw:
		return "Bonesaw"
	case BostonBasher:
		return "Boston Basher"
	case Bottle:
		return "Bottle"
	case BoxingGloveSpell:
		return "Boxing Glove Spell"
	case BuffBanner:
		return "Buff Banner"
	case BrassBeast:
		return "Brass Beast"
	case BreadBite:
		return "Bread Bite"
	case BuildingCarriedDestroyed:
		return "Building Destroyed (Carried)"
	case Bushwacka:
		return "Bushwacka"
	case Caber:
		return "Caber"
	case CaberExplosion:
		return "Caber Explosion"
	case CandyCane:
		return "Candy Cane"
	case CharginTarge:
		return "Chargin' Targe"
	case ClaidheamhMor:
		return "Claidheamh Mòr"
	case Club:
		return "Club"
	case ConscientiousObjector:
		return "Conscientious Objector"
	case CowMangler:
		return "Cow Mangler 5000"
	case Crossbow:
		return "Crusader's Crossbow"
	case CrossbowBolt:
		return "Crusader's Crossbow Bolt"
	case CrossingGuard:
		return "Crossing Guard"
	case DeflectArrow:
		return "Deflect Arrow"
	case DeflectFlare:
		return "Deflect Flare"
	case DeflectFlareDetonator:
		return "Deflect Detonator"
	case DeflectGrenade:
		return "Deflect Grenade"
	case DeflectHunstmanBurning:
		return "Deflect Huntsman (Burning)"
	case DeflectLooseCannon:
		return "Deflect Loose Cannon"
	case DeflectRocket:
		return "Deflect Rocket"
	case DeflectRocketMangler:
		return "Deflect Cow Mangler 5000"
	case DeflectSticky:
		return "Deflect Sticky"
	case Degreaser:
		return "Degreaser"
	case DemoKatana:
		return "Half-Zatoichi"
	case Detonator:
		return "Detonator"
	case Diamondback:
		return "Diamondback"
	case DirectHit:
		return "Direct Hit"
	case DisciplinaryAction:
		return "Disciplinary Action"
	case Dispenser:
		return "Dispenser"
	case DragonsFury:
		return "Dragons Fury"
	case DragonsFuryBonus:
		return "Dragons Fury Bonus"
	case Enforcer:
		return "Enforcer"
	case EntBonesaw:
		return "Bonesaw (Ent)"
	case EntBuilder:
		return "Builder (Ent)"
	case EntFrontierKill:
		return "Frontier Kill (Ent)"
	case EntManmelter:
		return "Manmelter (Ent)"
	case EntPickaxe:
		return "Pickaxe (Ent)"
	case EntSniperRifle:
		return "Sniper Rifle (Ent)"
	case Equalizer:
		return "Equalizer"
	case EscapePlan:
		return "Escape Plan"
	case EternalReward:
		return "Your Eternal Reward"
	case EurekaEffect:
		return "Eureka Effect"
	case EvictionNotice:
		return "Eviction Notice"
	case Eyelander:
		return "Eyelander"
	case FamilyBusiness:
		return "Family Business"
	case FanOWar:
		return "Fan O'War"
	case FireAxe:
		return "Fire Axe"
	case Fists:
		return "Fists"
	case FistsOfSteel:
		return "Fists of Steel"
	case FlameThrower:
		return "Flame Thrower"
	case Flare:
		return "Flare"
	case FlareGun:
		return "Flare Gun"
	case FlyingGuillotine:
		return "Flying Guillotine"
	case ForceANature:
		return "Force-A-Nature"
	case FortifiedCompound:
		return "Fortified Compound"
	case FreedomStaff:
		return "Freedom Staff"
	case FrontierJustice:
		return "Frontier Justice"
	case FryingPan:
		return "Frying Pan"
	case GoldenFryingPan:
		return "Golden Frying Pan"
	case GRU:
		return "Gloves of Running Urgently"
	case GasPasser:
		return "Gas Passer"
	case GigerCounter:
		return "Giger Counter"
	case GoldenWrench:
		return "Golden Wrench"
	case Gunslinger:
		return "Gunslinger"
	case GunslingerCombo:
		return "Gunslinger Combo"
	case GunslingerKill:
		return "Gunslinger Kill"
	case HHHHeadtaker:
		return "Horseless Headless Horsemann's Headtaker"
	case HamShank:
		return "Ham Shank"
	case HolidayPunch:
		return "Holiday Punch"
	case HolyMackerel:
		return "Holy Mackerel"
	case HotHand:
		return "Hot Hand"
	case Huntsman:
		return "Huntsman"
	case IronBomber:
		return "Iron Bomber"
	case IronCurtain:
		return "Iron Curtain"
	case Jag:
		return "Jag"
	case JarBased:
		return "Jar Based"
	case Jarate:
		return "Jarate"
	case JetpackStomp:
		return "Jetpack Stomp"
	case KGB:
		return "Killing Gloves of Boxing"
	case Knife:
		return "Knife"
	case Kukri:
		return "Kukri"
	case Kunai:
		return "Kunai"
	case Letranger:
		return "L'Etranger"
	case LibertyLauncher:
		return "Liberty Launcher"
	case LightningOrbSpell:
		return "Lightning Orb Spell"
	case LockNLoad:
		return "Loch-n-Load"
	case Lollichop:
		return "Lollichop"
	case LongHeatmaker:
		return "Hitman's Heatmaker (Ent)"
	case LooseCannon:
		return "Loose Cannon"
	case LooseCannonExplosion:
		return "Loose Cannon Explosion"
	case LooseCannonImpact:
		return "Loose Cannon Impact"
	case Lugermorph:
		return "Lugermorph"
	case Machina:
		return "Machina"
	case MachinaPen:
		return "Machina Penetration"
	case MadMilk:
		return "Mad Milk"
	case Manmelter:
		return "Manmelter"
	case Mantreads:
		return "Mantreads"
	case MarketGardener:
		return "Market Gardener"
	case Maul:
		return "Maul"
	case Medigun:
		return "Medigun"
	case MeteorShowerSpell:
		return "Meteor Shower Spell"
	case Minigun:
		return "Minigun"
	case MiniSentry:
		return "Sentry (mini)"
	case Natascha:
		return "Natascha"
	case NecroSmasher:
		return "Necro Smasher"
	case Needle:
		return "Needle Gun"
	case NeonAnnihilator:
		return "Neon Annihilator"
	case NessiesNineIron:
		return "Nessie's Nine Iron"
	case Original:
		return "Original"
	case OverdoseSyringe:
		return "Overdose"
	case PDAEngineer:
		return "PDA"
	case PainTrain:
		return "Pain Train"
	case PanicAttack:
		return "Panic Attack"
	case PersianPersuader:
		return "Persian Persuader"
	case Phlog:
		return "Phlogistinator"
	case PipebombLauncher:
		return "Grenade Launcher"
	case PistolEngy:
		return "Pistol (Engy)"
	case PistolScout:
		return "Pistol (Scout)"
	case Player: // Finish off player
		return "Finished Off"
	case Pomson:
		return "Pomson 6000"
	case PostalPummeler:
		return "Postal Pummeler"
	case Powerjack:
		return "Powerjack"
	case PrettyBoysPocketPistol:
		return "Pretty Boy's Pocket Pistol"
	case Prinny:
		return "Prinny Machete"
	case ProRifle:
		return "Hitman's Heatmaker"
	case ProSMG:
		return "Cleaner's Carbine"
	case ProjectileArrow:
		return "Arrow"
	case ProjectileArrowFire:
		return "Arrow (Burning)"
	case ProjectileDragonsFury:
		return "Dragons Fury Ball"
	case ProjectileGrenade:
		return "Grenade"
	case ProjectileJarGas:
		return "Gas Passer"
	case ProjectileRocket:
		return "Rocket Launcher"
	case ProjectileShortCircuit:
		return "Short Circuit"
	case ProjectileSticky:
		return "Stickybomb"
	case PumpkinBomb:
		return "Pumpkin Bomb"
	case Quickiebomb:
		return "Quickiebomb Launcher"
	case Rainblower:
		return "Rainblower"
	case RedTapeRecorder:
		return "Red-Tape Recorder"
	case RescueRanger:
		return "Rescue Ranger"
	case ReserveShooter:
		return "Reserve Shooter"
	case Revolver:
		return "Revolver"
	case RighteousBison:
		return "Righteous Bison"
	case RocketLauncher:
		return "Rocket Launcher (Ent)"
	case SMG:
		return "SMG"
	case Sandman:
		return "Sandman"
	case SandmanBall:
		return "Sandman Ball"
	case Sapper:
		return "Sapper"
	case Saxxy:
		return "Saxxy"
	case Scattergun:
		return "Scattergun"
	case ScorchShot:
		return "Scorch Shot"
	case ScotsmansSkullcutter:
		return "Scotsman's Skullcutter"
	case ScottishHandshake:
		return "Scottish Handshake"
	case ScottishResistance:
		return "Scottish Resistance"
	case Sentry1:
		return "Sentry (Level 1)"
	case Sentry2:
		return "Sentry (Level 2)"
	case Sentry3:
		return "Sentry (Level 3)"
	case SentryRocket:
		return "Sentry (Rocket)"
	case Shahanshah:
		return "Shahanshah"
	case Shark:
		return "Shark"
	case SharpDresser:
		return "Sharp Dresser"
	case SharpenedVolcanoFragment:
		return "Sharpened Volcano Fragment"
	case ShootingStar:
		return "Shooting Star"
	case ShortCircuit:
		return "Short Circuit"
	case Shortstop:
		return "Shortstop"
	case ShotgunEngy:
		return "Shotgun (Engy)"
	case ShotgunHeavy:
		return "Shotgun (Heavy)"
	case ShotgunPyro:
		return "Shotgun (Pyro)"
	case ShotgunSoldier:
		return "Shotgun (Soldier)"
	case Shovel:
		return "Shovel"
	case SkeletonSpawnSpell:
		return "Skeleton Spawn Spell"
	case Sledgehammer:
		return "Homewrecker"
	case SnackAttack:
		return "Snack Attack"
	case SniperRifle:
		return "Sniper Rifle"
	case SodaPopper:
		return "Soda Popper"
	case SolemnVow:
		return "Solemn Vow"
	case SouthernComfort:
		return "Southern Comfort"
	case SouthernHospitality:
		return "Southern Hospitality"
	case SplendidScreen:
		return "Splendid Screen"
	case Spycicle:
		return "Spy-cicle"
	case SuicideWeapon:
		return "Suicide"
	case SunOnAStick:
		return "Sun-on-a-Stick"
	case SydneySleeper:
		return "Sydney Sleeper"
	case SyringeGun:
		return "Syringe Gun"
	case TFFlameThrower:
		return "Flame Thrower (Ent"
	case TFMedigun:
		return "Medigun (Ent)"
	case TauntDemoman:
		return "Taunt (Demoman)"
	case TauntEngineer:
		return "Taunt (Engineer)"
	case TauntGuitarKill:
		return "Taunt (Guitar)"
	case TauntGunslinger:
		return "Taunt (Gunslinger)"
	case TauntHeavy:
		return "Taunt (Heavy)"
	case TauntMedic:
		return "Taunt (Medic)"
	case TauntPyro:
		return "Taunt (Pyro)"
	case TauntScout: // Sandman
		return "Taunt (Scout)"
	case TauntSniper: // huntsman
		return "Taunt (Sniper)"
	case TauntSoldier: // Equalizer
		return "Taunt (Soldier)"
	case TauntSoldierLumbricus:
		return "Taunt (Soldier Lumbricus)"
	case TauntSpy: // knife skewer
		return "Taunt (Spy)"
	case Telefrag:
		return "Telefrag"
	case TheCAPPER:
		return "C.A.P.P.E.R"
	case TheClassic:
		return "Classic"
	case TheWinger:
		return "Winger"
	case ThirdDegree:
		return "Third Degree"
	case ThreeRuneBlade:
		return "Three-Rune Blade"
	case TideTurner:
		return "Tide Turner"
	case Tomislav:
		return "Tomislav"
	case TribalmansShiv:
		return "Tribalman's Shiv"
	case Ubersaw:
		return "Ubersaw"
	case UnarmedCombat:
		return "Unarmed Combat"
	case VitaSaw:
		return "Vita-Saw"
	case WangaPrick:
		return "Wanga Prick"
	case WarriorsSpirit:
		return "Warrior's Spirit"
	case Widowmaker:
		return "Widowmaker"
	case World:
		return "World"
	case Wrangler:
		return "Wrangler"
	case WrapAssassin:
		return "Wrap Assassin"
	case Wrench:
		return "Wrench"
	default:
		return "Unknown"
	}
}
