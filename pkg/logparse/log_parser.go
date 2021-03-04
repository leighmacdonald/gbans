package logparse

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type MsgType int

const (
	UnhandledMsg MsgType = 0

	// Live player actions
	Say                 MsgType = 10
	SayTeam             MsgType = 11
	Killed              MsgType = 12
	KillAssist          MsgType = 13
	suicide             MsgType = 14
	shotFired           MsgType = 15
	shotHit             MsgType = 16
	damage              MsgType = 17
	domination          MsgType = 18
	revenge             MsgType = 19
	pickup              MsgType = 20
	emptyUber           MsgType = 21
	medicDeath          MsgType = 22
	medicDeathEx        MsgType = 23
	lostUberAdv         MsgType = 24
	chargeReady         MsgType = 25
	chargeDeployed      MsgType = 26
	chargeEnded         MsgType = 27
	healed              MsgType = 28
	extinguished        MsgType = 29
	builtObject         MsgType = 30
	carryObject         MsgType = 31
	killedObject        MsgType = 32
	detonatedObject     MsgType = 33
	dropObject          MsgType = 34
	firstHealAfterSpawn MsgType = 35
	captureBlocked      MsgType = 36
	killedCustom        MsgType = 37
	pointCaptured       MsgType = 48
	joinedTeam          MsgType = 49
	changeClass         MsgType = 50
	spawnedAs           MsgType = 51

	// World events not attached to specific players
	wRoundOvertime  MsgType = 100
	wRoundStart     MsgType = 101
	wRoundWin       MsgType = 102
	wRoundLen       MsgType = 103
	wTeamScore      MsgType = 104
	wTeamFinalScore MsgType = 105
	wGameOver       MsgType = 106
	wPaused         MsgType = 107
	wResumed        MsgType = 108

	// Metadata
	logStart     MsgType = 1000
	logStop      MsgType = 1001
	cvar         MsgType = 1002
	rcon         MsgType = 1003
	connected    MsgType = 1004
	disconnected MsgType = 1005
	validated    MsgType = 1006
	entered      MsgType = 1007
)

type parserType struct {
	Rx   *regexp.Regexp
	Type MsgType
}

var (
	// Date stuff
	rxDate = `^L\s(?P<date>.+?)\s+-\s+(?P<time>.+?):\s+`
	// Common player id format eg: "funk. Bubi<382><STEAM_0:1:22649331><>"
	rxPlayerStr = `"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator))?>"`
	//rxPlayer    = regexp.MustCompile(`(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator)?)>`)
	// Most player events have the same common prefix
	dp = rxDate + rxPlayerStr + `\s+`

	//rxSkipped      = regexp.MustCompile(`("undefined"$)`)
	rxLogStart     = regexp.MustCompile(rxDate + `Log file started \(file "(?P<file>.+?)"\) \(game "(?P<game>.+?)"\) \(version "(?P<version>.+?)"\)$`)
	rxLogStop      = regexp.MustCompile(rxDate + `Log file closed.$`)
	rxCVAR         = regexp.MustCompile(rxDate + `server_cvar:\s+"(?P<cvar>.+?)"\s"(?P<value>.+?)"$`)
	rxRCON         = regexp.MustCompile(rxDate + `rcon from "(?P<ip>.+?)": command "(?P<cmd>.+?)"$`)
	rxConnected    = regexp.MustCompile(dp + `connected, address(\s"(?P<address>.+?)")?$`)
	rxDisconnected = regexp.MustCompile(dp + `disconnected \(reason "(?P<reason>.+?)"\)$`)
	rxValidated    = regexp.MustCompile(dp + `STEAM USERID validated$`)
	rxEntered      = regexp.MustCompile(dp + `entered the game$`)
	rxJoinedTeam   = regexp.MustCompile(dp + `joined team "(?P<team>(Red|Blue|Spectator|Unassigned))"$`)
	rxChangeClass  = regexp.MustCompile(dp + `changed role to "(?P<class>.+?)"`)
	rxSpawned      = regexp.MustCompile(dp + `spawned as "(?P<class>\S+)"`)
	rxSuicide      = regexp.MustCompile(dp + `committed suicide with "world" \(attacker_position "(?P<pos>.+?)"\)`)
	rxShotFired    = regexp.MustCompile(dp + `triggered "shot_fired" \(weapon "(?P<weapon>\S+)"\)`)
	rxShotHit      = regexp.MustCompile(dp + `triggered "shot_hit" \(weapon "(?P<weapon>\S+)"\)`)
	rxDamage       = regexp.MustCompile(dp + `triggered "damage" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>"\s(?P<body>.+?)$`)
	//rxDamageRealHeal := regexp.MustCompile(dp + `triggered "damage" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>" \(damage "(?P<damage>\d+)"\) \(realdamage "(?P<realdamage>\d+)"\) \(weapon "(?P<weapon>.+?)"\) \(healing "(?P<healing>\d+)"\)`)
	// rxDamage := regexp.MustCompile(dp + `triggered "damage" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue)?)>".+?damage "(?P<damage>\d+)"\) \(weapon "(?P<weapon>\S+)"\)`)
	// Old format only?
	rxDamageOld            = regexp.MustCompile(dp + `triggered "damage" \(damage "(?P<damage>\d+)"\)`)
	rxKilled               = regexp.MustCompile(dp + `Killed "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>" with "(?P<weapon>.+?)" \(attacker_position "(?P<apos>.+?)"\) \(victim_position "(?P<vpos>.+?)"\)`)
	rxKilledCustom         = regexp.MustCompile(dp + `Killed "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>" with "(?P<weapon>.+?)" \(customkill "(?P<customkill>.+?)"\) \(attacker_position "(?P<apos>.+?)"\) \(victim_position "(?P<vpos>.+?)"\)`)
	rxAssist               = regexp.MustCompile(dp + `triggered "kill assist" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>" \(assister_position "(?P<aspos>.+?)"\) \(attacker_position "(?P<apos>.+?)"\) \(victim_position "(?P<vpos>.+?)"\)`)
	rxDomination           = regexp.MustCompile(dp + `triggered "domination" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Red|Blue)?)>"`)
	rxRevenge              = regexp.MustCompile(dp + `triggered "revenge" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>"\s?(\(assist "(?P<assist>\d+)"\))?`)
	rxPickup               = regexp.MustCompile(dp + `picked up item "(?P<item>\S+)"`)
	rxSay                  = regexp.MustCompile(dp + `Say\s+"(?P<msg>.+?)"$`)
	rxSayTeam              = regexp.MustCompile(dp + `say_team\s+"(?P<msg>.+?)"$`)
	rxEmptyUber            = regexp.MustCompile(dp + `triggered "empty_uber"`)
	rxMedicDeath           = regexp.MustCompile(dp + `triggered "medic_death" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue)?)>" \(healing "(?P<healing>\d+)"\) \(ubercharge "(?P<uber>\d+)"\)`)
	rxMedicDeathEx         = regexp.MustCompile(dp + `triggered "medic_death_ex" \(uberpct "(?P<pct>\d+)"\)`)
	rxLostUberAdv          = regexp.MustCompile(dp + `triggered "lost_uber_advantage" \(time "(?P<time>\d+)"\)`)
	rxChargeReady          = regexp.MustCompile(dp + `triggered "chargeready"`)
	rxChargeDeployed       = regexp.MustCompile(dp + `triggered "chargedeployed"( \(medigun "(?P<medigun>.+?)"\))?`)
	rxChargeEnded          = regexp.MustCompile(dp + `triggered "chargeended" \(duration "(?P<duration>.+?)"\)`)
	rxHealed               = regexp.MustCompile(dp + `triggered "healed" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>" \(healing "(?P<healing>\d+)"\)`)
	rxExtinguished         = regexp.MustCompile(dp + `triggered "player_extinguished" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Red|Blue)?)>" with "(?P<weapon>.+?)" \(attacker_position "(?P<apos>.+?)"\) \(victim_position "(?P<vpos>.+?)"\)`)
	rxBuiltObject          = regexp.MustCompile(dp + `triggered "player_builtobject" \(object "(?P<object>.+?)"\) \(position "(?P<pos>.+?)"\)`)
	rxCarryObject          = regexp.MustCompile(dp + `triggered "player_carryobject" \(object "(?P<object>.+?)"\) \(position "(?P<pos>.+?)"\)`)
	rxDropObject           = regexp.MustCompile(dp + `triggered "player_dropobject" \(object "(?P<object>.+?)"\) \(position "(?P<pos>.+?)"\)`)
	rxKilledObject         = regexp.MustCompile(dp + `triggered "killedobject" \(object "(?P<object>.+?)"\) \(weapon "(?P<weapon>.+?)"\) \(objectowner "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>"\) \(attacker_position "(?P<apos>.+?)"\)`)
	rxKilledObjectAssisted = regexp.MustCompile(dp + `triggered "killedobject" \(object "(?P<object>.+?)"\) \(objectowner "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>"\)\s+\(assist "1"\) \(assister_position "(?P<aspos>.+?)"\) \(attacker_position "(?P<apos>.+?)"\)`)
	rxDetonatedObject      = regexp.MustCompile(dp + `triggered "object_detonated" \(object "(?P<object>.+?)"\) \(position "(?P<Position>.+?)"\)`)
	rxFirstHealAfterSpawn  = regexp.MustCompile(dp + `triggered "first_heal_after_spawn" \(time "(?P<healtime>.+?)"\)`)
	rxWOvertime            = regexp.MustCompile(rxDate + `World triggered "Round_Overtime"`)
	rxWRoundStart          = regexp.MustCompile(rxDate + `World triggered "Round_Start"`)
	rxWGameOver            = regexp.MustCompile(rxDate + `World triggered "Game_Over" reason "(?P<reason>.+?)"`)
	rxWRoundLen            = regexp.MustCompile(rxDate + `World triggered "Round_Length" \(seconds "(?P<length>.+?)"\)`)
	rxWRoundWin            = regexp.MustCompile(rxDate + `World triggered "Round_Win" \(winner "(?P<winner>.+?)"\)`)
	rxWTeamFinalScore      = regexp.MustCompile(rxDate + `Team "(?P<team>Red|Blue)" final score "(?P<score>\d+)" with "(?P<players>\d+)" players`)
	rxWTeamScore           = regexp.MustCompile(rxDate + `Team "(?P<team>Red|Blue)" current score "(?P<score>\d+)" with "(?P<players>\d+)" players`)
	rxCaptureBlocked       = regexp.MustCompile(dp + `triggered "captureblocked" \(cp "(?P<cp>\d+)"\) \(cpname "(?P<cpname>.+?)"\) \(position "(?P<pos>.+?)"\)`)
	rxPointCaptured        = regexp.MustCompile(rxDate + `Team "(?P<team>.+?)" triggered "pointcaptured" \(cp "(?P<cp>\d+)"\) \(cpname "(?P<cpname>.+?)"\) \(numcappers "(?P<numcappers>\d+)"\)(\s+(?P<body>.+?))\s?$`)
	rxWPaused              = regexp.MustCompile(rxDate + `World triggered "Game_Paused"`)
	rxWResumed             = regexp.MustCompile(rxDate + `World triggered "Game_Unpaused"`)

	rxParsers = []parserType{
		{rxLogStart, logStart},
		{rxLogStop, logStop},
		{rxCVAR, cvar},
		{rxRCON, rcon},
		{rxShotFired, shotFired},
		{rxShotHit, shotHit},
		//{rxDamageRealHeal, damage},
		{rxDamage, damage},
		{rxDamageOld, damage},
		{rxKilled, Killed},
		{rxHealed, healed},
		{rxKilledCustom, killedCustom},
		{rxAssist, KillAssist},
		{rxPickup, pickup},
		{rxSpawned, spawnedAs},
		{rxValidated, validated},
		{rxConnected, connected},
		{rxEntered, entered},
		{rxJoinedTeam, joinedTeam},
		{rxChangeClass, changeClass},
		{rxSuicide, suicide},
		{rxChargeReady, chargeReady},
		{rxChargeDeployed, chargeDeployed},
		{rxChargeEnded, chargeEnded},
		{rxDomination, domination},
		{rxRevenge, revenge},
		{rxSay, Say},
		{rxSayTeam, SayTeam},
		{rxEmptyUber, emptyUber},
		{rxLostUberAdv, lostUberAdv},
		{rxMedicDeath, medicDeath},
		{rxMedicDeathEx, medicDeathEx},
		{rxExtinguished, extinguished},
		{rxBuiltObject, builtObject},
		{rxCarryObject, carryObject},
		{rxDropObject, dropObject},
		{rxKilledObject, killedObject},
		{rxKilledObjectAssisted, killedObject},
		{rxDetonatedObject, detonatedObject},
		{rxFirstHealAfterSpawn, firstHealAfterSpawn},
		{rxPointCaptured, pointCaptured},
		{rxCaptureBlocked, captureBlocked},
		{rxDisconnected, disconnected},
		{rxWOvertime, wRoundOvertime},
		{rxWRoundStart, wRoundStart},
		{rxWRoundWin, wRoundWin},
		{rxWRoundLen, wRoundLen},
		{rxWGameOver, wGameOver},
		{rxWTeamScore, wTeamScore},
		{rxWTeamFinalScore, wTeamFinalScore},
		{rxWPaused, wPaused},
		{rxWResumed, wResumed},
	}
)

type PlayerClass int

const (
	spectator PlayerClass = iota
	scout
	soldier
	pyro
	demo
	heavy
	engineer
	medic
	sniper
	spy
)

type Medigun int

const (
	uber Medigun = iota
	kritzkrieg
	vaccinator
	quickFix
)

type HealthPack int

const (
	hpSmall HealthPack = iota
	hpMedium
	hpLarge
)

func parseHealthPack(hp string) HealthPack {
	switch hp {
	case "medkit_small":
		return hpSmall
	case "medkit_medium":
		return hpMedium
	case "medkit_full":
		return hpLarge
	default:
		return hpMedium
	}
}

type AmmoPack int

const (
	ammoSmall AmmoPack = iota
	ammoMedium
	ammoLarge
)

func parseAmmoPack(hp string) AmmoPack {
	switch hp {
	case "tf_ammo_pack":
		return ammoSmall
	case "ammopack_medium":
		return ammoMedium
	default:
		return ammoLarge
	}
}
func parseMedigun(gunStr string) Medigun {
	switch strings.ToLower(gunStr) {
	case "medigun":
		return uber
	case "kritzkrieg":
		return kritzkrieg
	case "vaccinator":
		return vaccinator
	default:
		return quickFix
	}
}

func playerClassStr(cls PlayerClass) string {
	switch cls {
	case scout:
		return "Scout"
	case soldier:
		return "Soldier"
	case pyro:
		return "Pyro"
	case heavy:
		return "Heavy"
	case engineer:
		return "Engineer"
	case medic:
		return "Medic"
	case sniper:
		return "Sniper"
	case spy:
		return "Spy"
	default:
		return "Spectator"
	}
}

type Position struct {
	X int64
	Y int64
	Z int64
}

func parsePlayerClass(classStr string) PlayerClass {
	switch strings.ToLower(classStr) {
	case "scout":
		return scout
	case "soldier":
		return soldier
	case "pyro":
		return pyro
	case "demoman":
		return demo
	case "heavyweapons":
		return heavy
	case "engineer":
		return engineer
	case "medic":
		return medic
	case "sniper":
		return sniper
	case "spy":
		return spy
	default:
		return spectator
	}
}

type Team int

const (
	SPEC Team = 0
	RED  Team = 1
	BLU  Team = 2
)

func parseTeam(team string) Team {
	t := SPEC
	if team == "Red" {
		t = RED
	} else if team == "Blue" {
		t = BLU
	} else {
		t = SPEC
	}
	return t
}

func reSubMatchMap(r *regexp.Regexp, str string) (map[string]string, bool) {
	match := r.FindStringSubmatch(str)
	subMatchMap := make(map[string]string)
	if match == nil {
		return nil, false
	}
	for i, name := range r.SubexpNames() {
		if i != 0 {
			subMatchMap[name] = match[i]
		}
	}
	return subMatchMap, true
}

func parsePos(pos string) Position {
	p := strings.SplitN(pos, " ", 3)
	x, err := strconv.ParseInt(p[0], 10, 64)
	if err != nil {
		log.Warnf("Failed to parse x pos: %s", p[0])
		x = 0
	}
	y, err := strconv.ParseInt(p[1], 10, 64)
	if err != nil {
		log.Warnf("Failed to parse y pos: %s", p[1])
		y = 0
	}
	z, err := strconv.ParseInt(p[2], 10, 64)
	if err != nil {
		log.Warnf("Failed to parse z pos: %s", p[2])
		z = 0
	}
	return Position{x, y, z}
}

func parseDateTime(dateStr, timeStr string) time.Time {
	fDateStr := fmt.Sprintf("%s %s", dateStr, timeStr)
	t, err := time.Parse("02/01/2006 15:04:05", fDateStr)
	if err != nil {
		log.WithError(err).Errorf("Failed to parse date: %s", fDateStr)
		return time.Now()
	}
	return t
}

type Values map[string]string

// Parse will parse the log line into a known type and values
func Parse(l string) (Values, MsgType) {
	for _, rx := range rxParsers {
		m, found := reSubMatchMap(rx.Rx, l)
		if found {
			return m, rx.Type
		}

	}
	return Values{}, UnhandledMsg
}
