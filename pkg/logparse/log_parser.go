// Package logparse provides functionality for parsing TF2 console logs into known events and values.
//
// It should be able to parse logs from servers using SupStats2 & MedicStats plugins. These are the same requirements
// as logs.tf, so you should be able to download and parse them without much trouble.
package logparse

import (
	"fmt"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type parserType struct {
	Rx   *regexp.Regexp
	Type MsgType
}

var (
	rxKVPairs = regexp.MustCompile(`\((?P<key>.+?)\s+"(?P<value>.+?)"\)`)

	// Date stuff
	d = `^L\s(?P<date>.+?)\s+-\s+(?P<time>.+?):\s+`
	// Common player id format eg: "Name<382><STEAM_0:1:22649331><>"
	rxPlayerStr = `"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>([Uu]nassigned|[Rr]ed|[Bb]lue|[Ss]pectator|[Uu]nknown))?>"`
	//rxPlayer    = regexp.MustCompile(`(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator)?)>`)
	// Most player events have the same common prefix
	dp = d + rxPlayerStr + `\s+`
	kv = `\s+\((?P<kv>.+)\)$`
	//rxSkipped      = regexp.MustCompile(`("undefined"$)`)
	rxPlayer       = regexp.MustCompile(rxPlayerStr)
	rxUnhandled    = regexp.MustCompile(d)
	rxLogStart     = regexp.MustCompile(d + `Log file started` + kv)
	rxLogStop      = regexp.MustCompile(d + `Log file closed.$`)
	rxCVAR         = regexp.MustCompile(d + `server_cvar:\s+"(?P<CVAR>.+?)"\s"(?P<value>.+?)"$`)
	rxRCON         = regexp.MustCompile(d + `rcon from "(?P<ip>.+?)": command "(?P<cmd>.+?)"$`)
	rxConnected    = regexp.MustCompile(dp + `connected, address(\s"(?P<address>.+?)")?$`)
	rxDisconnected = regexp.MustCompile(dp + `disconnected \(reason "(?P<reason>.+?)"\)$`)
	rxValidated    = regexp.MustCompile(dp + `STEAM USERID [vV]alidated$`)
	rxEntered      = regexp.MustCompile(dp + `entered the game$`)
	rxJoinedTeam   = regexp.MustCompile(dp + `joined team "(?P<team>(Red|Blue|Spectator|Unassigned))"$`)
	rxChangeClass  = regexp.MustCompile(dp + `changed role to "(?P<class>.+?)"`)
	rxSpawned      = regexp.MustCompile(dp + `spawned as "(?P<class>\S+)"`)
	rxSuicide      = regexp.MustCompile(dp + `committed suicide with "world"` + kv)
	rxShotFired    = regexp.MustCompile(dp + `triggered "shot_fired"` + kv)
	rxShotHit      = regexp.MustCompile(dp + `triggered "shot_hit"` + kv)
	rxDamage       = regexp.MustCompile(dp + `triggered "[dD]amage" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>"` + kv)
	//rxDamageRealHeal := regexp.MustCompile(dp + `triggered "Damage" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>" \(Damage "(?P<Damage>\d+)"\) \(realdamage "(?P<realdamage>\d+)"\) \(weapon "(?P<weapon>.+?)"\) \(healing "(?P<healing>\d+)"\)`)
	// rxDamage := regexp.MustCompile(dp + `triggered "Damage" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue)?)>".+?Damage "(?P<Damage>\d+)"\) \(weapon "(?P<weapon>\S+)"\)`)
	// Old format only?
	//rxDamageOld            = regexp.MustCompile(dp + `triggered "damage" \(damage "(?P<damage>\d+)"\)`)
	rxKilled = regexp.MustCompile(dp + `killed "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>" with "(?P<weapon>.+?)"` + kv)
	//rxKilledCustom         = regexp.MustCompile(dp + `killed "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>" with "(?P<weapon>.+?)" \(customkill "(?P<customkill>.+?)"\) \(attacker_position "(?P<apos>.+?)"\) \(victim_position "(?P<vpos>.+?)"\)`)
	rxAssist               = regexp.MustCompile(dp + `triggered "kill assist" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>"` + kv)
	rxDomination           = regexp.MustCompile(dp + `triggered "domination" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Red|Blue)?)>"`)
	rxRevenge              = regexp.MustCompile(dp + `triggered "revenge" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>"\s?(\(assist "(?P<assist>\d+)"\))?`)
	rxPickup               = regexp.MustCompile(dp + `picked up item "(?P<item>\S+)"`)
	rxSay                  = regexp.MustCompile(dp + `say\s+"(?P<msg>.+?)"$`)
	rxSayTeam              = regexp.MustCompile(dp + `say_team\s+"(?P<msg>.+?)"$`)
	rxEmptyUber            = regexp.MustCompile(dp + `triggered "empty_uber"`)
	rxMedicDeath           = regexp.MustCompile(dp + `triggered "medic_death" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue)?)>" \(healing "(?P<healing>\d+)"\) \(ubercharge "(?P<uber>\d+)"\)`)
	rxMedicDeathEx         = regexp.MustCompile(dp + `triggered "medic_death_ex" \(uberpct "(?P<uberpct>\d+)"\)`)
	rxLostUberAdv          = regexp.MustCompile(dp + `triggered "lost_uber_advantage" \(time "(?P<advtime>\d+)"\)`)
	rxChargeReady          = regexp.MustCompile(dp + `triggered "chargeready"`)
	rxChargeDeployed       = regexp.MustCompile(dp + `triggered "chargedeployed"` + kv)
	rxChargeEnded          = regexp.MustCompile(dp + `triggered "chargeended"` + kv)
	rxHealed               = regexp.MustCompile(dp + `triggered "healed" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>" \(healing "(?P<healing>\d+)"\)`)
	rxExtinguished         = regexp.MustCompile(dp + `triggered "player_extinguished" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Red|Blue)?)>" with "(?P<weapon>.+?)" \(attacker_position "(?P<apos>.+?)"\) \(victim_position "(?P<vpos>.+?)"\)`)
	rxBuiltObject          = regexp.MustCompile(dp + `triggered "player_builtobject"` + kv)
	rxCarryObject          = regexp.MustCompile(dp + `triggered "player_carryobject"` + kv)
	rxDropObject           = regexp.MustCompile(dp + `triggered "player_dropobject"` + kv)
	rxKilledObject         = regexp.MustCompile(dp + `triggered "killedobject"` + kv)
	rxKilledObjectAssisted = regexp.MustCompile(dp + `triggered "killedobject"` + kv)
	rxDetonatedObject      = regexp.MustCompile(dp + `triggered "object_detonated"` + kv)
	rxFirstHealAfterSpawn  = regexp.MustCompile(dp + `triggered "first_heal_after_spawn"` + kv)
	rxMilkAttack           = regexp.MustCompile(dp + `triggered "milk_attack" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>([Uu]nassigned|[Rr]ed|[Bb]lue|[Ss]pectator|[Uu]nknown))?>"\s+with\s+"(?P<weapon>.+?)"` + kv)
	rxWOvertime            = regexp.MustCompile(d + `World triggered "Round_Overtime"`)
	rxWRoundStart          = regexp.MustCompile(d + `World triggered "Round_Start"`)
	rxWGameOver            = regexp.MustCompile(d + `World triggered "Game_Over" reason "(?P<reason>.+?)"`)
	rxWRoundLen            = regexp.MustCompile(d + `World triggered "Round_Length" \(seconds "(?P<length>.+?)"\)`)
	rxWRoundWin            = regexp.MustCompile(d + `World triggered "Round_Win" \(winner "(?P<winner>.+?)"\)`)
	rxWTeamFinalScore      = regexp.MustCompile(d + `Team "(?P<team>Red|Blue)" final score "(?P<score>\d+)" with "(?P<players>\d+)" players`)
	rxWTeamScore           = regexp.MustCompile(d + `Team "(?P<team>Red|Blue)" current score "(?P<score>\d+)" with "(?P<players>\d+)" players`)
	rxCaptureBlocked       = regexp.MustCompile(dp + `triggered "captureblocked"` + kv)
	rxPointCaptured        = regexp.MustCompile(d + `Team "(?P<team>.+?)" triggered "pointcaptured" \(cp "(?P<cp>\d+)"\) \(cpname "(?P<cpname>.+?)"\) \(numcappers "(?P<numcappers>\d+)"\)(\s+(?P<body>.+?))\s?$`)
	rxWPaused              = regexp.MustCompile(d + `World triggered "Game_Paused"`)
	rxWResumed             = regexp.MustCompile(d + `World triggered "Game_Unpaused"`)

	rxParsers = []parserType{
		{rxMilkAttack, MilkAttack},
		{rxLogStart, LogStart},
		{rxLogStop, LogStop},
		{rxCVAR, CVAR},
		{rxRCON, RCON},
		{rxShotFired, ShotFired},
		{rxShotHit, ShotHit},
		//{rxDamageRealHeal, Damage},
		{rxDamage, Damage},
		//{rxDamageOld, Damage},
		{rxKilled, Killed},
		{rxHealed, Healed},
		{rxAssist, KillAssist},
		{rxPickup, Pickup},
		{rxSpawned, SpawnedAs},
		{rxValidated, Validated},
		{rxConnected, Connected},
		{rxEntered, Entered},
		{rxJoinedTeam, JoinedTeam},
		{rxChangeClass, ChangeClass},
		{rxSuicide, Suicide},
		{rxChargeReady, ChargeReady},
		{rxChargeDeployed, ChargeDeployed},
		{rxChargeEnded, ChargeEnded},
		{rxDomination, Domination},
		{rxRevenge, Revenge},
		{rxSay, Say},
		{rxSayTeam, SayTeam},
		{rxEmptyUber, EmptyUber},
		{rxLostUberAdv, LostUberAdv},
		{rxMedicDeath, MedicDeath},
		{rxMedicDeathEx, MedicDeathEx},
		{rxExtinguished, Extinguished},
		{rxBuiltObject, BuiltObject},
		{rxCarryObject, CarryObject},
		{rxDropObject, DropObject},
		{rxKilledObject, KilledObject},
		{rxKilledObjectAssisted, KilledObject},
		{rxDetonatedObject, DetonatedObject},
		{rxFirstHealAfterSpawn, FirstHealAfterSpawn},
		{rxPointCaptured, PointCaptured},
		{rxCaptureBlocked, CaptureBlocked},
		{rxDisconnected, Disconnected},
		{rxWOvertime, WRoundOvertime},
		{rxWRoundStart, WRoundStart},
		{rxWRoundWin, WRoundWin},
		{rxWRoundLen, WRoundLen},
		{rxWGameOver, WGameOver},
		{rxWTeamScore, WTeamScore},
		{rxWTeamFinalScore, WTeamFinalScore},
		{rxWPaused, WPaused},
		{rxWResumed, WResumed},
	}
)

func kvToMap(s string, kv map[string]string) {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "(", ""), ")", "")
	v := strings.Split(s, " ")
	curKey := ""
	isPos := false
	isPlayerFmt := false
	var posPieces []string
	keyValue := true
	for _, value := range v {
		if keyValue && !isPos {
			curKey = value
			if strings.Contains(curKey, "position") {
				isPos = true
				posPieces = nil
			}
			if curKey == "objectowner" {
				isPlayerFmt = true
			}
			keyValue = false
		} else {
			if isPlayerFmt {
				m, ok := reSubMatchMap(rxPlayer, value)
				if ok {
					for k, val := range m {
						kv[k+"2"] = val
					}
				}
				isPlayerFmt = false
				keyValue = true
			} else if isPos {
				posPieces = append(posPieces, strings.Trim(value, "\""))
				if len(posPieces) == 3 {
					kv[curKey] = strings.Join(posPieces, " ")
					curKey = ""
					isPos = false
					posPieces = nil
					curKey = ""
					keyValue = true
				}
			} else {
				kv[curKey] = strings.Trim(value, "\"")
				curKey = ""
				keyValue = true
			}
		}
	}
	delete(kv, "kv")
}

func parseHealthPack(hp string, v *HealthPack) bool {
	switch hp {
	case "medkit_small":
		*v = HPSmall
	case "medkit_medium":
		*v = HPMedium
	case "medkit_large":
		*v = HPLarge
	default:
		return false
	}
	return true
}

func parseAmmoPack(hp string, pack *AmmoPack) bool {
	switch hp {
	case "ammopack_small":
		fallthrough
	case "tf_ammo_pack":
		*pack = AmmoSmall
	case "ammopack_medium":
		*pack = AmmoMedium
	case "ammopack_large":
		*pack = AmmoLarge
	default:
		return false
	}
	return true
}

func parseMedigun(gunStr string, gun *Medigun) bool {
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
//func playerClassStr(cls PlayerClass) string {
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
	case "undefined":
		fallthrough
	case "spectator":
		fallthrough
	case "spec":
		*class = Spectator
	default:
		return false
	}
	return true
}

func ParseTeam(teamStr string, team *Team) bool {
	switch strings.ToLower(teamStr) {
	case "red":
		*team = RED
	case "blue":
		fallthrough
	case "blu":
		*team = BLU
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

func parsePos(posStr string, pos *Pos) bool {
	p := strings.SplitN(posStr, " ", 3)
	if len(p) != 3 {
		return false
	}
	x, err1 := strconv.ParseInt(p[0], 10, 64)
	if err1 != nil {
		return false
	}
	y, err2 := strconv.ParseInt(p[1], 10, 64)
	if err2 != nil {
		return false
	}
	z, err3 := strconv.ParseInt(p[2], 10, 64)
	if err3 != nil {
		return false
	}
	pos.X = x
	pos.Y = y
	pos.Z = z
	return true
}

func parseDateTime(dateStr, timeStr string) time.Time {
	fDateStr := fmt.Sprintf("%s %s", dateStr, timeStr)
	t, err := time.Parse("01/02/2006 15:04:05", fDateStr)
	if err != nil {
		log.WithError(err).Errorf("Failed to parse date: %s", fDateStr)
		return time.Now()
	}
	return t
}

func parseKVs(s string, out map[string]string) bool {
	m := rxKVPairs.FindAllStringSubmatch(s, 10)
	if len(m) == 0 {
		return false
	}
	for mv := range m {
		out[m[mv][1]] = m[mv][2]
	}
	return true
}

// Results hold the  results of parsing a log line
//goland:noinspection GoUnnecessarilyExportedIdentifiers
type Results struct {
	MsgType MsgType
	Values  map[string]string
}

// Parse will parse the log line into a known type and values
func Parse(l string) Results {
	for _, rx := range rxParsers {
		m, found := reSubMatchMap(rx.Rx, l)
		if found {
			_, keyExists := m["kv"]
			if keyExists {
				kvToMap(m["kv"], m)
				delete(m, "kv")
			}
			return Results{rx.Type, m}
		}
	}
	m, found := reSubMatchMap(rxUnhandled, l)
	if found {
		return Results{UnhandledMsg, m}
	}
	return Results{UnknownMsg, map[string]string{"raw": l}}
}

func decodeTeam() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, d interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return d, nil
		}
		var team Team
		if !ParseTeam(d.(string), &team) {
			return d, nil
		}
		return team, nil
	}
}

func decodePlayerClass() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, d interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return d, nil
		}
		var cls PlayerClass
		if !parsePlayerClass(d.(string), &cls) {
			return d, nil
		}
		return cls, nil
	}
}

func decodePos() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, d interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return d, nil
		}
		var pos Pos
		if !parsePos(d.(string), &pos) {
			return d, nil
		}
		return pos, nil
	}
}

func decodeSID3() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, d interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return d, nil
		}
		if !strings.HasPrefix(d.(string), "[U") {
			return d, nil
		}
		sid := steamid.SID3ToSID64(steamid.SID3(d.(string)))
		if !sid.Valid() {
			return d, nil
		}
		return sid, nil
	}
}

func decodeMedigun() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, d interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return d, nil
		}
		var m Medigun
		if !parseMedigun(d.(string), &m) {
			return d, nil
		}
		return m, nil
	}
}

func decodeAmmoPack() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, d interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return d, nil
		}
		var m AmmoPack
		if !parseAmmoPack(d.(string), &m) {
			return d, nil
		}
		return m, nil
	}
}

func decodeHealthPack() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, d interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return d, nil
		}
		var m HealthPack
		if !parseHealthPack(d.(string), &m) {
			return d, nil
		}
		return m, nil
	}
}

// Unmarshal will transform a map of values into the struct passed in
// eg: {"sm_nextmap": "pl_frontier_final"} -> CVAREvt
//goland:noinspection GoUnnecessarilyExportedIdentifiers
func Unmarshal(input interface{}, output interface{}) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			decodeTeam(),
			decodePlayerClass(),
			decodePos(),
			decodeSID3(),
			decodeMedigun(),
			decodeAmmoPack(),
			decodeHealthPack(),
		),
		Result:           output,
		WeaklyTypedInput: true, // Lets us do str -> int easily
		Squash:           true,
	})
	if err != nil {
		return err
	}
	return decoder.Decode(input)
}
