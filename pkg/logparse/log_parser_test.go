package logparse

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	logEntries := strings.Split(exampleLog, "\n")
	tests := []struct {
		T MsgType // Expected type
		E Values  // Expected values
	}{
		{UnhandledMsg, Values{}},
		{LogStart, Values{"file": "logs/L0221034.log", "game": "/home/tf2server/serverfiles/tf", "version": "6300758"}},
		{CVAR, Values{"CVAR": "sm_nextmap", "value": "pl_frontier_final"}},
		{RCON, Values{"cmd": "status"}},
		{Entered, nil},
		{JoinedTeam, Values{"team": "Red"}},
		{ChangeClass, Values{"class": "scout"}},
		{Suicide, Values{"pos": "-1189 2513 -423"}},
		{WRoundStart, nil},
		{MedicDeath, Values{
			"name2": "Dzefersons14", "pid2": "8", "sid2": "[U:1:1080653073]", "team2": "Blue",
			"healing": "135", "uber": "0"}},
		{KilledCustom, Values{
			"name2": "Dzefersons14", "pid2": "8", "sid2": "[U:1:1080653073]", "team2": "Blue",
			"apos": "217 -54 -302", "vpos": "203 -2 -319", "customkill": "backstab"}},
		{KillAssist, Values{
			"name2": "Dzefersons14", "pid2": "8", "sid2": "[U:1:1080653073]", "team2": "Blue",
			"aspos": "-476 154 -254", "apos": "217 -54 -302", "vpos": "203 -2 -319"}},
		{PointCaptured, Values{
			"team": "Red", "cp": "0", "cpname": "#koth_viaduct_cap", "numcappers": "1",
			"body": `(player1 "Hacksaw<12><[U:1:68745073]><Red>") (position1 "101 98 -313")`}},
		{Connected, Values{"address": "139.47.95.130:47949"}},
		{Validated, Values{}},
		{KilledObject, Values{"name2": "idk", "pid2": "9", "sid2": "[U:1:1170132017]", "team2": "Blue",
			"object": "OBJ_SENTRYGUN", "weapon": "obj_attachment_sapper", "apos": "2 -579 -255"}},
		{CarryObject, Values{"object": "OBJ_SENTRYGUN", "pos": "1074 -2279 -423"}},
		{DropObject, Values{"object": "OBJ_SENTRYGUN", "pos": "339 -419 -255"}},
		{BuiltObject, Values{"object": "OBJ_SENTRYGUN", "pos": "880 -152 -255"}},
		{WRoundWin, Values{"winner": "Red"}},
		{WRoundLen, Values{"length": "398.10"}},
		{WTeamScore, Values{"team": "Red", "score": "1", "players": "2"}},
		{Say, Values{"msg": "gg"}},
		{SayTeam, Values{"msg": "gg"}},
		{Domination, Values{"name2": "Dzefersons14", "pid2": "8", "sid2": "[U:1:1080653073]", "team2": "Blue"}},
		{Disconnected, Values{"reason": "Disconnect by user."}},
		{Revenge, Values{"name2": "Desmos Calculator", "pid2": "10", "sid2": "[U:1:1132396177]", "team2": "Red"}},
		{WRoundOvertime, nil},
		{CaptureBlocked, Values{"cp": "0", "cpname": "#koth_viaduct_cap", "pos": "-163 324 -272"}},
		{WGameOver, Values{"reason": "Reached Win Limit"}},
		{WTeamFinalScore, Values{"score": "2", "players": "3"}},
		{UnhandledMsg, nil},
		{UnhandledMsg, nil},
		{LogStop, nil},
	}
	for i, test := range tests {
		foundValue, et := Parse(logEntries[i])
		require.Equalf(t, test.T, et, fmt.Sprintf("[%d] Invalid EventType parsed: %s", i, logEntries[i]))
		for k := range test.E {
			require.Equalf(t, test.E[k], foundValue[k], fmt.Sprintf("[%d] Parsed values dont match: %s", i, logEntries[i]))
		}
	}
}

const exampleLog = `L 02/21/2021 - 06:22:23: asdf
L 02/21/2021 - 06:22:23: Log file started (file "logs/L0221034.log") (game "/home/tf2server/serverfiles/tf") (version "6300758")
L 02/21/2021 - 06:22:23: server_cvar: "sm_nextmap" "pl_frontier_final"
L 02/21/2021 - 06:22:24: RCON from "23.239.22.163:42004": command "status"
L 02/21/2021 - 06:22:31: "Hacksaw<12><[U:1:68745073]><>" Entered the game
L 02/21/2021 - 06:22:35: "Hacksaw<12><[U:1:68745073]><Unassigned>" joined team "Red"
L 02/21/2021 - 06:22:36: "Hacksaw<12><[U:1:68745073]><Red>" changed role to "scout"
L 02/21/2021 - 06:23:04: "Dzefersons14<8><[U:1:1080653073]><Blue>" committed Suicide with "world" (attacker_position "-1189 2513 -423")
L 02/21/2021 - 06:23:11: World triggered "Round_Start"
L 02/21/2021 - 06:23:44: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "medic_death" against "Dzefersons14<8><[U:1:1080653073]><Blue>" (healing "135") (ubercharge "0")
L 02/21/2021 - 06:23:44: "Desmos Calculator<10><[U:1:1132396177]><Red>" Killed "Dzefersons14<8><[U:1:1080653073]><Blue>" with "spy_cicle" (customkill "backstab") (attacker_position "217 -54 -302") (victim_position "203 -2 -319")
L 02/21/2021 - 06:23:44: "Hacksaw<12><[U:1:68745073]><Red>" triggered "kill assist" against "Dzefersons14<8><[U:1:1080653073]><Blue>" (assister_position "-476 154 -254") (attacker_position "217 -54 -302") (victim_position "203 -2 -319")
L 02/21/2021 - 06:24:14: Team "Red" triggered "pointcaptured" (cp "0") (cpname "#koth_viaduct_cap") (numcappers "1") (player1 "Hacksaw<12><[U:1:68745073]><Red>") (position1 "101 98 -313") 
L 02/21/2021 - 06:24:22: "amogus gaming<13><[U:1:1089803558]><>" Connected, address "139.47.95.130:47949"
L 02/21/2021 - 06:24:23: "amogus gaming<13><[U:1:1089803558]><>" STEAM USERID Validated
L 02/21/2021 - 06:26:33: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "killedobject" (object "OBJ_SENTRYGUN") (weapon "obj_attachment_sapper") (objectowner "idk<9><[U:1:1170132017]><Blue>") (attacker_position "2 -579 -255")
L 02/21/2021 - 06:30:45: "idk<9><[U:1:1170132017]><Blue>" triggered "player_carryobject" (object "OBJ_SENTRYGUN") (position "1074 -2279 -423")
L 02/21/2021 - 06:32:00: "idk<9><[U:1:1170132017]><Blue>" triggered "player_dropobject" (object "OBJ_SENTRYGUN") (position "339 -419 -255")
L 02/21/2021 - 06:32:30: "idk<9><[U:1:1170132017]><Blue>" triggered "player_builtobject" (object "OBJ_SENTRYGUN") (position "880 -152 -255")
L 02/21/2021 - 06:29:49: World triggered "Round_Win" (winner "Red")
L 02/21/2021 - 06:29:49: World triggered "Round_Length" (seconds "398.10")
L 02/21/2021 - 06:29:49: Team "Red" current score "1" with "2" players
L 02/21/2021 - 06:29:57: "Hacksaw<12><[U:1:68745073]><Red>" Say "gg"
L 02/21/2021 - 06:29:59: "Desmos Calculator<10><[U:1:1132396177]><Red>" say_team "gg"
L 02/21/2021 - 06:33:41: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "Domination" against "Dzefersons14<8><[U:1:1080653073]><Blue>"
L 02/21/2021 - 06:33:43: "Cybermorphic<15><[U:1:901503117]><Unassigned>" Disconnected (reason "Disconnect by user.")
L 02/21/2021 - 06:35:37: "Dzefersons14<8><[U:1:1080653073]><Blue>" triggered "Revenge" against "Desmos Calculator<10><[U:1:1132396177]><Red>"
L 02/21/2021 - 06:37:20: World triggered "Round_Overtime"
L 02/21/2021 - 06:40:19: "potato<16><[U:1:385661040]><Red>" triggered "captureblocked" (cp "0") (cpname "#koth_viaduct_cap") (position "-163 324 -272")
L 02/21/2021 - 06:42:13: World triggered "Game_Over" reason "Reached Win Limit"
L 02/21/2021 - 06:42:13: Team "Red" final score "2" with "3" players
L 02/21/2021 - 06:42:13: Team "RED" triggered "Intermission_Win_Limit"
L 02/21/2021 - 06:42:33: [META] Loaded 0 plugins (1 already loaded)
L 02/21/2021 - 06:42:33: Log file closed.
`
