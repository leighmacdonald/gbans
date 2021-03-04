package logparse

import (
	"fmt"
	"github.com/leighmacdonald/gbans/pkg/logparse/msgtype"
	log "github.com/sirupsen/logrus"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type parserType struct {
	Rx   *regexp.Regexp
	Type msgtype.MsgType
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
	rxCVAR         = regexp.MustCompile(rxDate + `server_cvar:\s+"(?P<CVAR>.+?)"\s"(?P<value>.+?)"$`)
	rxRCON         = regexp.MustCompile(rxDate + `RCON from "(?P<ip>.+?)": command "(?P<cmd>.+?)"$`)
	rxConnected    = regexp.MustCompile(dp + `Connected, address(\s"(?P<address>.+?)")?$`)
	rxDisconnected = regexp.MustCompile(dp + `Disconnected \(reason "(?P<reason>.+?)"\)$`)
	rxValidated    = regexp.MustCompile(dp + `STEAM USERID validated$`)
	rxEntered      = regexp.MustCompile(dp + `Entered the game$`)
	rxJoinedTeam   = regexp.MustCompile(dp + `joined team "(?P<team>(Red|Blue|Spectator|Unassigned))"$`)
	rxChangeClass  = regexp.MustCompile(dp + `changed role to "(?P<class>.+?)"`)
	rxSpawned      = regexp.MustCompile(dp + `spawned as "(?P<class>\S+)"`)
	rxSuicide      = regexp.MustCompile(dp + `committed Suicide with "world" \(attacker_position "(?P<pos>.+?)"\)`)
	rxShotFired    = regexp.MustCompile(dp + `triggered "shot_fired" \(weapon "(?P<weapon>\S+)"\)`)
	rxShotHit      = regexp.MustCompile(dp + `triggered "shot_hit" \(weapon "(?P<weapon>\S+)"\)`)
	rxDamage       = regexp.MustCompile(dp + `triggered "Damage" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>"\s(?P<body>.+?)$`)
	//rxDamageRealHeal := regexp.MustCompile(dp + `triggered "Damage" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>" \(Damage "(?P<Damage>\d+)"\) \(realdamage "(?P<realdamage>\d+)"\) \(weapon "(?P<weapon>.+?)"\) \(healing "(?P<healing>\d+)"\)`)
	// rxDamage := regexp.MustCompile(dp + `triggered "Damage" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue)?)>".+?Damage "(?P<Damage>\d+)"\) \(weapon "(?P<weapon>\S+)"\)`)
	// Old format only?
	rxDamageOld            = regexp.MustCompile(dp + `triggered "Damage" \(Damage "(?P<Damage>\d+)"\)`)
	rxKilled               = regexp.MustCompile(dp + `Killed "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>" with "(?P<weapon>.+?)" \(attacker_position "(?P<apos>.+?)"\) \(victim_position "(?P<vpos>.+?)"\)`)
	rxKilledCustom         = regexp.MustCompile(dp + `Killed "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>" with "(?P<weapon>.+?)" \(customkill "(?P<customkill>.+?)"\) \(attacker_position "(?P<apos>.+?)"\) \(victim_position "(?P<vpos>.+?)"\)`)
	rxAssist               = regexp.MustCompile(dp + `triggered "kill assist" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>" \(assister_position "(?P<aspos>.+?)"\) \(attacker_position "(?P<apos>.+?)"\) \(victim_position "(?P<vpos>.+?)"\)`)
	rxDomination           = regexp.MustCompile(dp + `triggered "Domination" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Red|Blue)?)>"`)
	rxRevenge              = regexp.MustCompile(dp + `triggered "Revenge" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>"\s?(\(assist "(?P<assist>\d+)"\))?`)
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
	rxHealed               = regexp.MustCompile(dp + `triggered "Healed" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>" \(healing "(?P<healing>\d+)"\)`)
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
		{rxLogStart, msgtype.LogStart},
		{rxLogStop, msgtype.LogStop},
		{rxCVAR, msgtype.CVAR},
		{rxRCON, msgtype.RCON},
		{rxShotFired, msgtype.ShotFired},
		{rxShotHit, msgtype.ShotHit},
		//{rxDamageRealHeal, Damage},
		{rxDamage, msgtype.Damage},
		{rxDamageOld, msgtype.Damage},
		{rxKilled, msgtype.Killed},
		{rxHealed, msgtype.Healed},
		{rxKilledCustom, msgtype.KilledCustom},
		{rxAssist, msgtype.KillAssist},
		{rxPickup, msgtype.Pickup},
		{rxSpawned, msgtype.SpawnedAs},
		{rxValidated, msgtype.Validated},
		{rxConnected, msgtype.Connected},
		{rxEntered, msgtype.Entered},
		{rxJoinedTeam, msgtype.JoinedTeam},
		{rxChangeClass, msgtype.ChangeClass},
		{rxSuicide, msgtype.Suicide},
		{rxChargeReady, msgtype.ChargeReady},
		{rxChargeDeployed, msgtype.ChargeDeployed},
		{rxChargeEnded, msgtype.ChargeEnded},
		{rxDomination, msgtype.Domination},
		{rxRevenge, msgtype.Revenge},
		{rxSay, msgtype.Say},
		{rxSayTeam, msgtype.SayTeam},
		{rxEmptyUber, msgtype.EmptyUber},
		{rxLostUberAdv, msgtype.LostUberAdv},
		{rxMedicDeath, msgtype.MedicDeath},
		{rxMedicDeathEx, msgtype.MedicDeathEx},
		{rxExtinguished, msgtype.Extinguished},
		{rxBuiltObject, msgtype.BuiltObject},
		{rxCarryObject, msgtype.CarryObject},
		{rxDropObject, msgtype.DropObject},
		{rxKilledObject, msgtype.KilledObject},
		{rxKilledObjectAssisted, msgtype.KilledObject},
		{rxDetonatedObject, msgtype.DetonatedObject},
		{rxFirstHealAfterSpawn, msgtype.FirstHealAfterSpawn},
		{rxPointCaptured, msgtype.PointCaptured},
		{rxCaptureBlocked, msgtype.CaptureBlocked},
		{rxDisconnected, msgtype.Disconnected},
		{rxWOvertime, msgtype.WRoundOvertime},
		{rxWRoundStart, msgtype.WRoundStart},
		{rxWRoundWin, msgtype.WRoundWin},
		{rxWRoundLen, msgtype.WRoundLen},
		{rxWGameOver, msgtype.WGameOver},
		{rxWTeamScore, msgtype.WTeamScore},
		{rxWTeamFinalScore, msgtype.WTeamFinalScore},
		{rxWPaused, msgtype.WPaused},
		{rxWResumed, msgtype.WResumed},
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
func Parse(l string) (Values, msgtype.MsgType) {
	for _, rx := range rxParsers {
		m, found := reSubMatchMap(rx.Rx, l)
		if found {
			return m, rx.Type
		}

	}
	return Values{}, msgtype.UnhandledMsg
}
