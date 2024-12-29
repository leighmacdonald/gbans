package demoparse

// Team represents a players team, or spectator state.
type Team int

const (
	UNASSIGNED Team = iota
	SPEC
	RED
	BLU
)

type PlayerClass int

//goland:noinspection GoUnnecessarilyExportedIdentifiers
const (
	Spectator PlayerClass = iota
	Scout
	Sniper
	Soldier
	Demoman
	Medic
	Heavy
	Pyro
	Spy
	Engineer
	Multi
)

type RoundState int

const (
	Init RoundState = iota
	Pregame
	StartGame
	PreRound
	RoundRunning
	TeamWin
	Restart
	Stalemate
	GameOver
	Bonus
	BetweenRounds
)

type WeaponID int

const (
	WeaponNone WeaponID = iota
	WeaponBat
	WeaponBatWood
	WeaponBottle
	WeaponFireaxe
	WeaponClub
	WeaponCrowbar
	WeaponKnife
	WeaponFists
	WeaponShovel
	WeaponWrench
	WeaponBonesaw
	WeaponShotgunPrimary
	WeaponShotgunSoldier
	WeaponShotgunHwg
	WeaponShotgunPyro
	WeaponScattergun
	WeaponSniperrifle
	WeaponMinigun
	WeaponSmg
	WeaponSyringegunMedic
	WeaponTranq
	WeaponRocketlauncher
	WeaponGrenadelauncher
	WeaponPipebomblauncher
	WeaponFlamethrower
	WeaponGrenadeNormal
	WeaponGrenadeConcussion
	WeaponGrenadeNail
	WeaponGrenadeMirv
	WeaponGrenadeMirvDemoman
	WeaponGrenadeNapalm
	WeaponGrenadeGas
	WeaponGrenadeEmp
	WeaponGrenadeCaltrop
	WeaponGrenadePipebomb
	WeaponGrenadeSmokeBomb
	WeaponGrenadeHeal
	WeaponGrenadeStunball
	WeaponGrenadeJar
	WeaponGrenadeJarMilk
	WeaponPistol
	WeaponPistolScout
	WeaponRevolver
	WeaponNailgun
	WeaponPda
	WeaponPdaEngineerBuild
	WeaponPdaEngineerDestroy
	WeaponPdaSpy
	WeaponBuilder
	WeaponMedigun
	WeaponGrenadeMirvbomb
	WeaponFlamethrowerRocket
	WeaponGrenadeDemoman
	WeaponSentryBullet
	WeaponSentryRocket
	WeaponDispenser
	WeaponInvis
	WeaponFlaregun
	WeaponLunchbox
	WeaponJar
	WeaponCompoundBow
	WeaponBuffItem
	WeaponPumpkinBomb
	WeaponSword
	WeaponRocketlauncherDirecthit
	WeaponLifeline
	WeaponLaserPointer
	WeaponDispenserGun
	WeaponSentryRevenge
	WeaponJarMilk
	WeaponHandgunScoutPrimary
	WeaponBatFish
	WeaponCrossbow
	WeaponStickbomb
	WeaponHandgunScoutSecondary
	WeaponSodaPopper
	WeaponSniperrifleDecap
	WeaponRaygun
	WeaponParticleCannon
	WeaponMechanicalArm
	WeaponDrgPomson
	WeaponBatGiftwrap
	WeaponGrenadeOrnamentBall
	WeaponFlaregunRevenge
	WeaponPepBrawlerBlaster
	WeaponCleaver
	WeaponGrenadeCleaver
	WeaponStickyBallLauncher
	WeaponGrenadeStickyBall
	WeaponShotgunBuildingRescue
	WeaponCannon
	WeaponThrowable
	WeaponGrenadeThrowable
	WeaponPdaSpyBuild
	WeaponGrenadeWaterballoon
	WeaponHarvesterSaw
	WeaponSpellbook
	WeaponSpellbookProjectile
	WeaponSniperrifleClassic
	WeaponParachute
	WeaponGrapplinghook
	WeaponPasstimeGun
	WeaponSniperrifleRevolver
	WeaponChargedSmg
)
