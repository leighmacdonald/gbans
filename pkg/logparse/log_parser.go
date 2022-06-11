// Package logparse provides functionality for parsing TF2 console logs into known events and values.
//
// It should be able to parse logs from servers using SupStats2 & MedicStats plugins. These are the same requirements
// as logs.tf, so you should be able to download and parse them without much trouble.
package logparse

import (
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/mitchellh/mapstructure"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type parserType struct {
	Rx   *regexp.Regexp
	Type EventType
}

var (
	rxKVPairs = regexp.MustCompile(`\((?P<key>.+?)\s+"(?P<value>.+?)"\)`)

	// Date stuff
	d = `^L\s(?P<created_on>.+?):\s+`
	// Common player id format eg: "Name<382><STEAM_0:1:22649331><>"
	rxPlayerStr = `"(?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>"`
	rxPlayer    = regexp.MustCompile(rxPlayerStr)
	// Most player events have the same common prefix
	dp       = d + rxPlayerStr + `\s+`
	keyPairs = `\s+(?P<keypairs>.+?)$`
	//rxSkipped      = regexp.MustCompile(`("undefined"$)`)
	rxUnhandled            = regexp.MustCompile(d)
	rxLogStart             = regexp.MustCompile(d + `[Ll]og file started` + keyPairs)
	rxLogStop              = regexp.MustCompile(d + `[Ll]og file closed.$`)
	rxCVAR                 = regexp.MustCompile(d + `server_cvar:\s+"(?P<CVAR>.+?)"\s"(?P<value>.+?)"$`)
	rxRCON                 = regexp.MustCompile(d + `[Rr][Cc][Oo][Nn] from "(?P<ip>.+?)": command "(?P<cmd>.+?)"$`)
	rxConnected            = regexp.MustCompile(dp + `[Cc]onnected, address(\s"(?P<address>.+?)")?$`)
	rxDisconnected         = regexp.MustCompile(dp + `[Dd]isconnected \(reason "(?P<reason>.+?)$`)
	rxValidated            = regexp.MustCompile(dp + `STEAM USERID [vV]alidated$`)
	rxEntered              = regexp.MustCompile(dp + `[Ee]ntered the game$`)
	rxJoinedTeam           = regexp.MustCompile(dp + `joined team "(?P<team>(Red|Blue|Spectator|Unassigned))"$`)
	rxChangeClass          = regexp.MustCompile(dp + `changed role to "(?P<class>.+?)"`)
	rxSpawned              = regexp.MustCompile(dp + `spawned as "(?P<class>\S+)"$`)
	rxSuicide              = regexp.MustCompile(dp + `committed suicide with "(?P<weapon>.+?)"` + keyPairs)
	rxShotFired            = regexp.MustCompile(dp + `triggered "shot_fired"` + keyPairs)
	rxShotHit              = regexp.MustCompile(dp + `triggered "shot_hit"` + keyPairs)
	rxDamage               = regexp.MustCompile(dp + `triggered "[dD]amage" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>"` + keyPairs)
	rxDamageOld            = regexp.MustCompile(dp + `triggered "[dD]amage" \(damage "(?P<damage>\d+)"\)`)
	rxKilled               = regexp.MustCompile(dp + `killed "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>" with "(?P<weapon>.+?)"` + keyPairs)
	rxKilledCustom         = regexp.MustCompile(dp + `killed "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>" with "(?P<weapon>.+?)"\s+(\(customkill "(?P<customkill>.+?)"\))` + keyPairs)
	rxAssist               = regexp.MustCompile(dp + `triggered "kill assist" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>"` + keyPairs)
	rxDomination           = regexp.MustCompile(dp + `triggered "[Dd]omination" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Red|Blue)?)>"`)
	rxRevenge              = regexp.MustCompile(dp + `triggered "[Rr]evenge" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>"\s?(\(assist "(?P<assist>\d+)"\))?`)
	rxPickup               = regexp.MustCompile(dp + `picked up item "(?P<item>\S+)"`)
	rxPickupMedPack        = regexp.MustCompile(dp + `picked up item "(?P<item>\S+)"` + keyPairs)
	rxSay                  = regexp.MustCompile(dp + `say\s+"(?P<msg>.+?)"$`)
	rxSayTeam              = regexp.MustCompile(dp + `say_team\s+"(?P<msg>.+?)"$`)
	rxEmptyUber            = regexp.MustCompile(dp + `triggered "empty_uber"`)
	rxMedicDeath           = regexp.MustCompile(dp + `triggered "medic_death" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue)?)>"` + keyPairs)
	rxJarateAttack         = regexp.MustCompile(dp + `triggered "jarate_attack" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue)?)>" with "(?P<weapon>.+?)"` + keyPairs)
	rxMilkAttack           = regexp.MustCompile(dp + `triggered "milk_attack" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue)?)>" with "(?P<weapon>.+?)"` + keyPairs)
	rxGasAttack            = regexp.MustCompile(dp + `triggered "gas_attack" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue)?)>" with "(?P<weapon>.+?)"` + keyPairs)
	rxMedicDeathEx         = regexp.MustCompile(dp + `triggered "medic_death_ex"` + keyPairs)
	rxLostUberAdv          = regexp.MustCompile(dp + `triggered "lost_uber_advantage"` + keyPairs)
	rxChargeReady          = regexp.MustCompile(dp + `triggered "chargeready"`)
	rxChargeDeployed       = regexp.MustCompile(dp + `triggered "chargedeployed"( \(medigun "(?P<medigun>.+?)"\))?`)
	rxChargeEnded          = regexp.MustCompile(dp + `triggered "chargeended" \(duration "(?P<duration>.+?)"\)`)
	rxHealed               = regexp.MustCompile(dp + `triggered "[hH]ealed" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Unassigned|Red|Blue|Spectator)?)>"` + keyPairs)
	rxExtinguished         = regexp.MustCompile(dp + `triggered "player_extinguished" against "(?P<name2>.+?)<(?P<pid2>\d+)><(?P<sid2>.+?)><(?P<team2>(Red|Blue)?)>" with "(?P<weapon>.+?)"` + keyPairs)
	rxBuiltObject          = regexp.MustCompile(dp + `triggered "player_builtobject"` + keyPairs)
	rxCarryObject          = regexp.MustCompile(dp + `triggered "player_carryobject"` + keyPairs)
	rxDropObject           = regexp.MustCompile(dp + `triggered "player_dropobject"` + keyPairs)
	rxKilledObject         = regexp.MustCompile(dp + `triggered "killedobject"` + keyPairs)
	rxKilledObjectAssisted = regexp.MustCompile(dp + `triggered "killedobject"` + keyPairs)
	rxDetonatedObject      = regexp.MustCompile(dp + `triggered "object_detonated"` + keyPairs)
	rxFirstHealAfterSpawn  = regexp.MustCompile(dp + `triggered "first_heal_after_spawn"` + keyPairs)
	rxWOvertime            = regexp.MustCompile(d + `World triggered "Round_Overtime"`)
	rxWRoundStart          = regexp.MustCompile(d + `World triggered "Round_Start"`)
	rxWMiniRoundStart      = regexp.MustCompile(d + `World triggered "Mini_Round_Start"`)
	rxWRoundSetupEnd       = regexp.MustCompile(d + `World triggered "Round_Setup_End"`)
	rxWRoundSetupBegin     = regexp.MustCompile(d + `World triggered "Round_Setup_Begin"`)
	rxWGameOver            = regexp.MustCompile(d + `World triggered "Game_Over" reason "(?P<reason>.+?)"`)
	rxWRoundLen            = regexp.MustCompile(d + `World triggered "Round_Length"` + keyPairs)
	rxWRoundWin            = regexp.MustCompile(d + `World triggered "Round_Win"` + keyPairs)
	rxWMiniRoundWin        = regexp.MustCompile(d + `World triggered "Mini_Round_Win"` + keyPairs)
	rxWMiniRoundLen        = regexp.MustCompile(d + `World triggered "Mini_Round_Length"` + keyPairs)
	rxWMiniRoundSelected   = regexp.MustCompile(d + `World triggered "Mini_Round_Selected"` + keyPairs)
	rxWTeamFinalScore      = regexp.MustCompile(d + `Team "(?P<team>Red|Blue)" final score "(?P<score>\d+)" with "(?P<players>\d+)" players`)
	rxWTeamScore           = regexp.MustCompile(d + `Team "(?P<team>Red|Blue)" current score "(?P<score>\d+)" with "(?P<players>\d+)" players`)
	rxCaptureBlocked       = regexp.MustCompile(dp + `triggered "captureblocked"` + keyPairs)
	rxPointCaptured        = regexp.MustCompile(d + `Team "(?P<team>.+?)" triggered "pointcaptured"` + keyPairs)
	rxWPaused              = regexp.MustCompile(d + `World triggered "Game_Paused"`)
	rxWResumed             = regexp.MustCompile(d + `World triggered "Game_Unpaused"`)
	rxServerConfigExec     = regexp.MustCompile(d + `Executing dedicated server config file (?P<config>.+?)$`)
	rxLoadingMap           = regexp.MustCompile(d + `Loading map "(?P<map>.+?)"$`)
	rxJunkServerCVAR       = regexp.MustCompile(d + `"(.+?)"\s=\s"(.+?)"$`)
	rxJunkServerCVARStart  = regexp.MustCompile(d + `server cvars start`)
	rxJunkMetaPlugin       = regexp.MustCompile(d + `\[META\]`)
	rxSteamAuth            = regexp.MustCompile(d + `STEAMAUTH: (?P<reason>.+?)$`)

	// Map matching regex to known event types
	rxParsers = []parserType{
		{rxLogStart, LogStart},
		{rxLogStop, LogStop},
		{rxCVAR, CVAR},
		{rxRCON, RCON},
		{rxShotFired, ShotFired},
		{rxShotHit, ShotHit},
		{rxDamage, Damage},
		{rxDamageOld, Damage},
		{rxKilledCustom, KilledCustom}, // Must come before Killed
		{rxKilled, Killed},
		{rxHealed, Healed},
		{rxAssist, KillAssist},
		{rxPickupMedPack, Pickup},
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
		{rxWRoundSetupEnd, WRoundStart},
		{rxWRoundWin, WRoundWin},
		{rxWRoundLen, WRoundLen},
		{rxWGameOver, WGameOver},
		{rxWTeamScore, WTeamScore},
		{rxWTeamFinalScore, WTeamFinalScore},
		{rxWPaused, WPaused},
		{rxWResumed, WResumed},
		{rxLoadingMap, MapLoad},
		{rxServerConfigExec, ServerConfigExec},
		{rxSteamAuth, SteamAuth},
		{rxJarateAttack, JarateAttack},
		{rxMilkAttack, MilkAttack},
		{rxGasAttack, GasAttack},
		{rxWMiniRoundWin, WMiniRoundWin},
		{rxWMiniRoundLen, WMiniRoundLen},
		{rxWRoundSetupBegin, WRoundSetupBegin},
		{rxWMiniRoundSelected, WMiniRoundSelected},
		{rxWMiniRoundStart, WMiniRoundStart},
		{rxJunkServerCVAR, IgnoredMsg},
		{rxJunkServerCVARStart, IgnoredMsg},
		{rxJunkMetaPlugin, IgnoredMsg},
	}
)

func ParsePickupItem(hp string, item *PickupItem) bool {
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

func ParseMedigun(gunStr string, gun *Medigun) bool {
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

func ParsePlayerClass(classStr string, class *PlayerClass) bool {
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

func ParseTeam(teamStr string, team *Team) bool {
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

func reSubMatchMap(r *regexp.Regexp, str string) (map[string]any, bool) {
	match := r.FindStringSubmatch(str)
	subMatchMap := make(map[string]any)
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
	pieces := strings.SplitN(posStr, " ", 3)
	if len(pieces) != 3 {
		return false
	}
	x, errParseX := strconv.ParseFloat(pieces[0], 64)
	if errParseX != nil {
		return false
	}
	y, errParseY := strconv.ParseFloat(pieces[1], 64)
	if errParseY != nil {
		return false
	}
	z, errParseZ := strconv.ParseFloat(pieces[2], 64)
	if errParseZ != nil {
		return false
	}
	pos.X = x
	pos.Y = y
	pos.Z = z
	return true
}

func parseDateTime(dateStr string, t *time.Time) bool {
	parsed, errParseTime := time.Parse("01/02/2006 - 15:04:05", dateStr)
	if errParseTime != nil {
		return false
	}
	*t = parsed
	return true
}

func parseKVs(s string, out map[string]any) bool {
	m := rxKVPairs.FindAllStringSubmatch(s, 10)
	if len(m) == 0 {
		return false
	}
	for mv := range m {
		out[m[mv][1]] = m[mv][2]
	}
	return true
}

func processKV(originalKVMap map[string]any) map[string]any {
	newKVMap := map[string]any{}
	for key, value := range originalKVMap {
		switch key {
		case "created_on":
			var t time.Time
			if parseDateTime(value.(string), &t) {
				newKVMap["created_on"] = t
			}
		case "medigun":
			var medigun Medigun
			if ParseMedigun(value.(string), &medigun) {
				newKVMap["medigun"] = medigun
			}
		case "crit":
			switch value.(string) {
			case "crit":
				newKVMap["crit"] = Crit
			case "mini":
				newKVMap["crit"] = Mini
			default:
				newKVMap["crit"] = NonCrit
			}
		case "reason":
			// Some reasons get output with a newline, so it gets these uneven line endings
			reason := value.(string)
			newKVMap["reason"] = strings.TrimSuffix(reason, `")`)
		case "objectowner":
			ooKV, ok := reSubMatchMap(rxPlayer, "\""+value.(string)+"\"")
			if ok {
				// TODO Make this less static to support >2 targets for events like capping points?
				for key, val := range ooKV {
					newKVMap[key+"2"] = val
				}
			}
		case "address":
			// Split newKVMap client port for easier queries
			pieces := strings.Split(value.(string), ":")
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

// Results hold the  results of parsing a log line
//goland:noinspection GoUnnecessarilyExportedIdentifiers
type Results struct {
	MsgType EventType
	Values  map[string]any
}

// Parse will parse the log line into a known type and values
func Parse(logLine string) Results {
	for _, rx := range rxParsers {
		matchMap, found := reSubMatchMap(rx.Rx, strings.TrimSuffix(strings.TrimSuffix(logLine, "\n"), "\r"))
		if found {
			value, ok := matchMap["keypairs"].(string)
			if ok {
				parseKVs(value, matchMap)
			}
			delete(matchMap, "keypairs")
			delete(matchMap, "")
			return Results{rx.Type, processKV(matchMap)}
		}
	}
	m, found := reSubMatchMap(rxUnhandled, logLine)
	if found {
		return Results{IgnoredMsg, processKV(m)}
	}
	return Results{UnknownMsg, map[string]any{"raw": logLine}}
}

func decodeTeam() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, d any) (any, error) {
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
	return func(f reflect.Type, t reflect.Type, d any) (any, error) {
		if f.Kind() != reflect.String {
			return d, nil
		}
		var playerClass PlayerClass
		if !ParsePlayerClass(d.(string), &playerClass) {
			return d, nil
		}
		return playerClass, nil
	}
}

func decodePos() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, d any) (any, error) {
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
	return func(f reflect.Type, t reflect.Type, d any) (any, error) {
		if f.Kind() != reflect.String {
			return d, nil
		}
		if !strings.HasPrefix(d.(string), "[U") {
			return d, nil
		}
		sid64 := steamid.SID3ToSID64(steamid.SID3(d.(string)))
		if !sid64.Valid() {
			return d, nil
		}
		return sid64, nil
	}
}

//func decodeMedigun() mapstructure.DecodeHookFunc {
//	return func(f reflect.Type, t reflect.Type, d any) (any, error) {
//		if f.Kind() != reflect.String {
//			return d, nil
//		}
//		var m Medigun
//		if !ParseMedigun(d.(string), &m) {
//			return d, nil
//		}
//		return m, nil
//	}
//}

func decodePickupItem() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, d any) (any, error) {
		if f.Kind() != reflect.String {
			return d, nil
		}
		var m PickupItem
		if !ParsePickupItem(d.(string), &m) {
			return d, nil
		}
		return m, nil
	}
}

func decodeWeapon() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, d any) (any, error) {
		if f.Kind() != reflect.String {
			return d, nil
		}
		w := ParseWeapon(d.(string))
		if w != UnknownWeapon {
			return w, nil
		}
		return d, nil
	}
}

func decodeTime() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, d any) (any, error) {
		if f.Kind() != reflect.String {
			return d, nil
		}
		var t0 time.Time
		if parseDateTime(d.(string), &t0) {
			return t0, nil
		}
		return d, nil
	}
}

// Unmarshal will transform a map of values into the struct passed in
// eg: {"sm_nextmap": "pl_frontier_final"} -> CVAREvt
//goland:noinspection GoUnnecessarilyExportedIdentifiers
func Unmarshal(input any, output any) error {
	decoder, errNewDecoder := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			decodeTime(),
			decodeTeam(),
			decodePlayerClass(),
			decodePos(),
			decodeSID3(),
			//decodeMedigun(),
			decodePickupItem(),
			decodeWeapon(),
		),
		Result:           output,
		WeaklyTypedInput: true, // Lets us do str -> int easily
		Squash:           true,
	})
	if errNewDecoder != nil {
		return errNewDecoder
	}
	return decoder.Decode(input)
}
