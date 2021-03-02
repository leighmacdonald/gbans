package logparse

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestParse(t *testing.T) {
	type tst struct {
		M string
		T MsgType
		E Values
	}

	tests := []tst{
		{`L 02/21/2021 - 06:22:23: Some random unhandled log format`,
			unhandledMsg, Values{},
		},
		{`L 02/21/2021 - 06:22:23: Log file started (file "logs/L0221034.log") (game "/home/tf2server/serverfiles/tf") (version "6300758")`,
			logStart, Values{
				"file":    "logs/L0221034.log",
				"game":    "/home/tf2server/serverfiles/tf",
				"version": "6300758"},
		},
		{`L 02/21/2021 - 06:22:24: rcon from "23.239.22.163:42004": command "status"`,
			rcon, Values{"cmd": "status"},
		},
	}
	for i, test := range tests {
		foundValue, et := Parse(test.M)
		require.Equalf(t, test.T, et, fmt.Sprintf("[%d] Invalid EventType parsed", i))
		for k := range test.E {
			require.Equalf(t, test.E[k], foundValue[k], fmt.Sprintf("[%d] Parsed values dont match", i))
		}
	}
}

const exampleLog = `L 02/21/2021 - 06:22:23: Log file started (file "logs/L0221034.log") (game "/home/tf2server/serverfiles/tf") (version "6300758")
L 02/21/2021 - 06:22:23: server_cvar: "sm_nextmap" "pl_frontier_final"
L 02/21/2021 - 06:22:23: server_cvar: "mp_timelimit" "30"
L 02/21/2021 - 06:22:24: rcon from "23.239.22.163:42004": command "status"
L 02/21/2021 - 06:22:31: "Hacksaw<12><[U:1:68745073]><>" entered the game
L 02/21/2021 - 06:22:35: "Hacksaw<12><[U:1:68745073]><Unassigned>" joined team "Red"
L 02/21/2021 - 06:22:35: "idk<9><[U:1:1170132017]><>" entered the game
L 02/21/2021 - 06:22:36: "Hacksaw<12><[U:1:68745073]><Red>" changed role to "scout"
L 02/21/2021 - 06:22:46: "idk<9><[U:1:1170132017]><Unassigned>" joined team "Blue"
L 02/21/2021 - 06:22:46: "Desmos Calculator<10><[U:1:1132396177]><>" entered the game
L 02/21/2021 - 06:22:47: "Dzefersons14<8><[U:1:1080653073]><>" entered the game
L 02/21/2021 - 06:22:49: "idk<9><[U:1:1170132017]><Blue>" changed role to "pyro"
L 02/21/2021 - 06:22:50: rcon from "68.144.74.48:64680": command "status"
L 02/21/2021 - 06:22:57: "Dzefersons14<8><[U:1:1080653073]><Unassigned>" joined team "Red"
L 02/21/2021 - 06:22:59: "Dzefersons14<8><[U:1:1080653073]><Red>" changed role to "soldier"
L 02/21/2021 - 06:23:04: "Dzefersons14<8><[U:1:1080653073]><Red>" joined team "Blue"
L 02/21/2021 - 06:23:04: "Dzefersons14<8><[U:1:1080653073]><Blue>" committed suicide with "world" (attacker_position "-1189 2513 -423")
L 02/21/2021 - 06:23:09: "Dzefersons14<8><[U:1:1080653073]><Blue>" changed role to "scout"
L 02/21/2021 - 06:23:11: World triggered "Round_Start"
L 02/21/2021 - 06:23:14: "Dzefersons14<8><[U:1:1080653073]><Blue>" changed role to "medic"
L 02/21/2021 - 06:23:21: "Desmos Calculator<10><[U:1:1132396177]><Unassigned>" joined team "Red"
L 02/21/2021 - 06:23:23: "Desmos Calculator<10><[U:1:1132396177]><Red>" changed role to "spy"
L 02/21/2021 - 06:23:24: rcon from "23.239.22.163:42120": command "status"
L 02/21/2021 - 06:23:44: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "medic_death" against "Dzefersons14<8><[U:1:1080653073]><Blue>" (healing "135") (ubercharge "0")
L 02/21/2021 - 06:23:44: "Desmos Calculator<10><[U:1:1132396177]><Red>" killed "Dzefersons14<8><[U:1:1080653073]><Blue>" with "spy_cicle" (customkill "backstab") (attacker_position "217 -54 -302") (victim_position "203 -2 -319")
L 02/21/2021 - 06:23:44: "Hacksaw<12><[U:1:68745073]><Red>" triggered "kill assist" against "Dzefersons14<8><[U:1:1080653073]><Blue>" (assister_position "-476 154 -254") (attacker_position "217 -54 -302") (victim_position "203 -2 -319")
L 02/21/2021 - 06:23:50: rcon from "68.144.74.48:64700": command "status"
L 02/21/2021 - 06:23:56: "Hacksaw<12><[U:1:68745073]><Red>" killed "idk<9><[U:1:1170132017]><Blue>" with "force_a_nature" (attacker_position "763 693 -255") (victim_position "210 801 -205")
L 02/21/2021 - 06:23:56: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "kill assist" against "idk<9><[U:1:1170132017]><Blue>" (assister_position "-252 793 -255") (attacker_position "763 693 -255") (victim_position "210 801 -205")
L 02/21/2021 - 06:24:04: "idk<9><[U:1:1170132017]><Blue>" changed role to "demoman"
L 02/21/2021 - 06:24:14: Team "Red" triggered "pointcaptured" (cp "0") (cpname "#koth_viaduct_cap") (numcappers "1") (player1 "Hacksaw<12><[U:1:68745073]><Red>") (position1 "101 98 -313") 
L 02/21/2021 - 06:24:22: "amogus gaming<13><[U:1:1089803558]><>" connected, address "139.47.95.130:47949"
L 02/21/2021 - 06:24:23: "amogus gaming<13><[U:1:1089803558]><>" STEAM USERID validated
L 02/21/2021 - 06:24:24: rcon from "23.239.22.163:42242": command "status"
L 02/21/2021 - 06:24:34: "amogus gaming<13><[U:1:1089803558]><>" entered the game
L 02/21/2021 - 06:24:34: "idk<9><[U:1:1170132017]><Blue>" killed "Desmos Calculator<10><[U:1:1132396177]><Red>" with "demokatana" (attacker_position "174 -2453 -255") (victim_position "100 -2447 -255")
L 02/21/2021 - 06:24:34: "Dzefersons14<8><[U:1:1080653073]><Blue>" triggered "kill assist" against "Desmos Calculator<10><[U:1:1132396177]><Red>" (assister_position "147 -2308 -254") (attacker_position "174 -2453 -255") (victim_position "100 -2447 -255")
L 02/21/2021 - 06:24:50: rcon from "68.144.74.48:64727": command "status"
L 02/21/2021 - 06:24:53: "amogus gaming<13><[U:1:1089803558]><Unassigned>" joined team "Blue"
L 02/21/2021 - 06:24:54: "amogus gaming<13><[U:1:1089803558]><Blue>" changed role to "demoman"
L 02/21/2021 - 06:25:09: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "medic_death" against "Dzefersons14<8><[U:1:1080653073]><Blue>" (healing "219") (ubercharge "0")
L 02/21/2021 - 06:25:09: "Desmos Calculator<10><[U:1:1132396177]><Red>" killed "Dzefersons14<8><[U:1:1080653073]><Blue>" with "spy_cicle" (customkill "backstab") (attacker_position "-659 598 -223") (victim_position "-661 529 -223")
L 02/21/2021 - 06:25:11: "amogus gaming<13><[U:1:1089803558]><Blue>" killed "Hacksaw<12><[U:1:68745073]><Red>" with "iron_bomber" (attacker_position "-147 146 -319") (victim_position "-149 583 -208")
L 02/21/2021 - 06:25:11: "idk<9><[U:1:1170132017]><Blue>" triggered "kill assist" against "Hacksaw<12><[U:1:68745073]><Red>" (assister_position "-733 312 -255") (attacker_position "-147 146 -319") (victim_position "-149 583 -208")
L 02/21/2021 - 06:25:24: rcon from "23.239.22.163:42362": command "status"
L 02/21/2021 - 06:25:34: "Desmos Calculator<10><[U:1:1132396177]><Red>" killed "amogus gaming<13><[U:1:1089803558]><Blue>" with "spy_cicle" (attacker_position "-833 1820 -455") (victim_position "-894 1856 -455")
L 02/21/2021 - 06:25:34: "Hacksaw<12><[U:1:68745073]><Red>" triggered "kill assist" against "amogus gaming<13><[U:1:1089803558]><Blue>" (assister_position "-1263 2105 -419") (attacker_position "-833 1820 -455") (victim_position "-894 1856 -455")
L 02/21/2021 - 06:25:40: Team "Blue" triggered "pointcaptured" (cp "0") (cpname "#koth_viaduct_cap") (numcappers "2") (player1 "Dzefersons14<8><[U:1:1080653073]><Blue>") (position1 "51 63 -313") (player2 "idk<9><[U:1:1170132017]><Blue>") (position2 "-10 51 -313") 
L 02/21/2021 - 06:25:50: rcon from "68.144.74.48:64746": command "status"
L 02/21/2021 - 06:25:58: "Desmos Calculator<10><[U:1:1132396177]><Red>" killed "idk<9><[U:1:1170132017]><Blue>" with "spy_cicle" (customkill "backstab") (attacker_position "252 -228 -254") (victim_position "195 -188 -263")
L 02/21/2021 - 06:25:58: "Hacksaw<12><[U:1:68745073]><Red>" triggered "kill assist" against "idk<9><[U:1:1170132017]><Blue>" (assister_position "-290 165 -283") (attacker_position "252 -228 -254") (victim_position "195 -188 -263")
L 02/21/2021 - 06:26:02: "Hacksaw<12><[U:1:68745073]><Red>" triggered "medic_death" against "Dzefersons14<8><[U:1:1080653073]><Blue>" (healing "263") (ubercharge "0")
L 02/21/2021 - 06:26:02: "Hacksaw<12><[U:1:68745073]><Red>" killed "Dzefersons14<8><[U:1:1080653073]><Blue>" with "force_a_nature" (attacker_position "68 -92 -319") (victim_position "382 -582 -255")
L 02/21/2021 - 06:26:02: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "kill assist" against "Dzefersons14<8><[U:1:1080653073]><Blue>" (assister_position "490 -530 -239") (attacker_position "68 -92 -319") (victim_position "382 -582 -255")
L 02/21/2021 - 06:26:06: "idk<9><[U:1:1170132017]><Blue>" changed role to "engineer"
L 02/21/2021 - 06:26:18: Team "Red" triggered "pointcaptured" (cp "0") (cpname "#koth_viaduct_cap") (numcappers "1") (player1 "Hacksaw<12><[U:1:68745073]><Red>") (position1 "17 56 -313") 
L 02/21/2021 - 06:26:24: rcon from "23.239.22.163:42486": command "status"
L 02/21/2021 - 06:26:29: "idk<9><[U:1:1170132017]><Blue>" triggered "player_builtobject" (object "OBJ_SENTRYGUN") (position "291 -426 -255")
L 02/21/2021 - 06:26:30: "Dzefersons14<8><[U:1:1080653073]><Blue>" changed role to "spy"
L 02/21/2021 - 06:26:31: "Desmos Calculator<10><[U:1:1132396177]><Red>" killed "idk<9><[U:1:1170132017]><Blue>" with "spy_cicle" (customkill "backstab") (attacker_position "314 -453 -255") (victim_position "293 -372 -255")
L 02/21/2021 - 06:26:31: "Hacksaw<12><[U:1:68745073]><Red>" triggered "kill assist" against "idk<9><[U:1:1170132017]><Blue>" (assister_position "66 -743 -255") (attacker_position "314 -453 -255") (victim_position "293 -372 -255")
L 02/21/2021 - 06:26:32: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "player_builtobject" (object "OBJ_ATTACHMENT_SAPPER") (position "284 -463 -255")
L 02/21/2021 - 06:26:33: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "killedobject" (object "OBJ_SENTRYGUN") (weapon "obj_attachment_sapper") (objectowner "idk<9><[U:1:1170132017]><Blue>") (attacker_position "2 -579 -255")
L 02/21/2021 - 06:26:50: rcon from "68.144.74.48:64761": command "status"
L 02/21/2021 - 06:26:55: "idk<9><[U:1:1170132017]><Blue>" triggered "player_builtobject" (object "OBJ_SENTRYGUN") (position "453 -617 -255")
L 02/21/2021 - 06:27:01: "Hacksaw<12><[U:1:68745073]><Red>" killed "amogus gaming<13><[U:1:1089803558]><Blue>" with "world" (attacker_position "186 13 -319") (victim_position "193 668 -243")
L 02/21/2021 - 06:27:02: "idk<9><[U:1:1170132017]><Blue>" killed "Hacksaw<12><[U:1:68745073]><Red>" with "obj_minisentry" (attacker_position "407 -481 -255") (victim_position "-131 377 -228")
L 02/21/2021 - 06:27:24: rcon from "23.239.22.163:42602": command "status"
L 02/21/2021 - 06:27:29: "idk<9><[U:1:1170132017]><Blue>" killed "Desmos Calculator<10><[U:1:1132396177]><Red>" with "robot_arm" (attacker_position "450 -179 -255") (victim_position "427 -113 -254")
L 02/21/2021 - 06:27:35: "amogus gaming<13><[U:1:1089803558]><Blue>" killed "Hacksaw<12><[U:1:68745073]><Red>" with "iron_bomber" (attacker_position "14 143 1") (victim_position "-473 718 -255")
L 02/21/2021 - 06:27:49: Team "Blue" triggered "pointcaptured" (cp "0") (cpname "#koth_viaduct_cap") (numcappers "2") (player1 "Dzefersons14<8><[U:1:1080653073]><Blue>") (position1 "74 54 -313") (player2 "amogus gaming<13><[U:1:1089803558]><Blue>") (position2 "-14 -4 -313") 
L 02/21/2021 - 06:27:50: rcon from "68.144.74.48:64774": command "status"
L 02/21/2021 - 06:28:04: "Hacksaw<12><[U:1:68745073]><Red>" killed "amogus gaming<13><[U:1:1089803558]><Blue>" with "force_a_nature" (attacker_position "-1389 1413 -447") (victim_position "-1840 1052 -455")
L 02/21/2021 - 06:28:11: "idk<9><[U:1:1170132017]><Blue>" triggered "player_builtobject" (object "OBJ_DISPENSER") (position "506 -580 -239")
L 02/21/2021 - 06:28:16: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "player_builtobject" (object "OBJ_ATTACHMENT_SAPPER") (position "526 -510 -223")
L 02/21/2021 - 06:28:17: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "killedobject" (object "OBJ_SENTRYGUN") (weapon "obj_attachment_sapper") (objectowner "idk<9><[U:1:1170132017]><Blue>") (attacker_position "448 -458 -255")
L 02/21/2021 - 06:28:18: "Desmos Calculator<10><[U:1:1132396177]><Red>" killed "idk<9><[U:1:1170132017]><Blue>" with "spy_cicle" (customkill "backstab") (attacker_position "484 -726 -255") (victim_position "493 -659 -255")
L 02/21/2021 - 06:28:19: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "player_builtobject" (object "OBJ_ATTACHMENT_SAPPER") (position "415 -691 -255")
L 02/21/2021 - 06:28:20: "Hacksaw<12><[U:1:68745073]><Red>" killed "Dzefersons14<8><[U:1:1080653073]><Blue>" with "force_a_nature" (attacker_position "-862 170 -255") (victim_position "-599 223 -222")
L 02/21/2021 - 06:28:24: rcon from "23.239.22.163:42724": command "status"
L 02/21/2021 - 06:28:25: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "killedobject" (object "OBJ_DISPENSER") (weapon "obj_attachment_sapper") (objectowner "idk<9><[U:1:1170132017]><Blue>") (attacker_position "679 -233 -255")
L 02/21/2021 - 06:28:29: "Dzefersons14<8><[U:1:1080653073]><Blue>" changed role to "demoman"
L 02/21/2021 - 06:28:33: "amogus gaming<13><[U:1:1089803558]><Blue>" killed "Desmos Calculator<10><[U:1:1132396177]><Red>" with "iron_bomber" (attacker_position "-88 -855 -255") (victim_position "-209 -1058 -235")
L 02/21/2021 - 06:28:37: "amogus gaming<13><[U:1:1089803558]><Blue>" triggered "captureblocked" (cp "0") (cpname "#koth_viaduct_cap") (position "63 -214 -129")
L 02/21/2021 - 06:28:50: rcon from "68.144.74.48:64794": command "status"
L 02/21/2021 - 06:28:56: "amogus gaming<13><[U:1:1089803558]><Blue>" killed "Hacksaw<12><[U:1:68745073]><Red>" with "iron_bomber" (attacker_position "-625 73 -255") (victim_position "-1445 189 -367")
L 02/21/2021 - 06:29:15: "Hacksaw<12><[U:1:68745073]><Red>" killed "amogus gaming<13><[U:1:1089803558]><Blue>" with "force_a_nature" (attacker_position "-1623 998 -455") (victim_position "-1615 811 -432")
L 02/21/2021 - 06:29:22: "Dzefersons14<8><[U:1:1080653073]><Blue>" killed "Hacksaw<12><[U:1:68745073]><Red>" with "tf_projectile_pipe_remote" (attacker_position "544 -513 -511") (victim_position "67 179 -319")
L 02/21/2021 - 06:29:24: rcon from "23.239.22.163:42840": command "status"
L 02/21/2021 - 06:29:28: "Desmos Calculator<10><[U:1:1132396177]><Red>" killed "Dzefersons14<8><[U:1:1080653073]><Blue>" with "spy_cicle" (customkill "backstab") (attacker_position "-288 -418 -463") (victim_position "-235 -416 -463")
L 02/21/2021 - 06:29:35: "idk<9><[U:1:1170132017]><Blue>" triggered "captureblocked" (cp "0") (cpname "#koth_viaduct_cap") (position "155 -207 -270")
L 02/21/2021 - 06:29:42: "Hacksaw<12><[U:1:68745073]><Red>" killed "idk<9><[U:1:1170132017]><Blue>" with "force_a_nature" (attacker_position "-145 223 -319") (victim_position "65 61 -313")
L 02/21/2021 - 06:29:47: Team "Red" triggered "pointcaptured" (cp "0") (cpname "#koth_viaduct_cap") (numcappers "1") (player1 "Hacksaw<12><[U:1:68745073]><Red>") (position1 "80 -60 -319") 
L 02/21/2021 - 06:29:49: World triggered "Round_Win" (winner "Red")
L 02/21/2021 - 06:29:49: World triggered "Round_Length" (seconds "398.10")
L 02/21/2021 - 06:29:49: Team "Red" current score "1" with "2" players
L 02/21/2021 - 06:29:49: Team "Blue" current score "0" with "3" players
L 02/21/2021 - 06:29:49: rcon from "68.144.74.48:64815": command "status"
L 02/21/2021 - 06:29:57: "Hacksaw<12><[U:1:68745073]><Red>" say "gg"
L 02/21/2021 - 06:29:59: "Desmos Calculator<10><[U:1:1132396177]><Red>" say "gg"
L 02/21/2021 - 06:30:04: World triggered "Round_Start"
L 02/21/2021 - 06:30:19: "idk<9><[U:1:1170132017]><Blue>" triggered "player_builtobject" (object "OBJ_SENTRYGUN") (position "1039 -2283 -423")
L 02/21/2021 - 06:30:24: rcon from "23.239.22.163:42964": command "status"
L 02/21/2021 - 06:30:30: "amogus gaming<13><[U:1:1089803558]><Blue>" committed suicide with "iron_bomber" (attacker_position "44 1031 -255")
L 02/21/2021 - 06:30:45: "idk<9><[U:1:1170132017]><Blue>" triggered "player_carryobject" (object "OBJ_SENTRYGUN") (position "1074 -2279 -423")
L 02/21/2021 - 06:30:50: rcon from "68.144.74.48:64831": command "status"
L 02/21/2021 - 06:30:56: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "killedobject" (object "OBJ_SENTRYGUN") (weapon "building_carried_destroyed") (objectowner "idk<9><[U:1:1170132017]><Blue>") (attacker_position "702 -2523 -343")
L 02/21/2021 - 06:30:56: "Desmos Calculator<10><[U:1:1132396177]><Red>" killed "idk<9><[U:1:1170132017]><Blue>" with "spy_cicle" (customkill "backstab") (attacker_position "702 -2523 -343") (victim_position "746 -2488 -331")
L 02/21/2021 - 06:31:06: "Hacksaw<12><[U:1:68745073]><Red>" killed "amogus gaming<13><[U:1:1089803558]><Blue>" with "pep_pistol" (attacker_position "-951 117 -255") (victim_position "-971 -122 -255")
L 02/21/2021 - 06:31:11: "Dzefersons14<8><[U:1:1080653073]><Blue>" triggered "captureblocked" (cp "0") (cpname "#koth_viaduct_cap") (position "204 213 -319")
L 02/21/2021 - 06:31:14: "amogus gaming<13><[U:1:1089803558]><Blue>" changed role to "sniper"
L 02/21/2021 - 06:31:17: "idk<9><[U:1:1170132017]><Blue>" triggered "player_builtobject" (object "OBJ_SENTRYGUN") (position "1436 -2358 -447")
L 02/21/2021 - 06:31:20: "Dzefersons14<8><[U:1:1080653073]><Blue>" killed "Hacksaw<12><[U:1:68745073]><Red>" with "tf_projectile_pipe" (attacker_position "83 -35 -318") (victim_position "319 255 -378")
L 02/21/2021 - 06:31:24: rcon from "23.239.22.163:43084": command "status"
L 02/21/2021 - 06:31:45: "idk<9><[U:1:1170132017]><Blue>" triggered "player_carryobject" (object "OBJ_SENTRYGUN") (position "1437 -2329 -447")
L 02/21/2021 - 06:31:50: rcon from "68.144.74.48:64850": command "status"
L 02/21/2021 - 06:32:00: "idk<9><[U:1:1170132017]><Blue>" triggered "player_dropobject" (object "OBJ_SENTRYGUN") (position "339 -419 -255")
L 02/21/2021 - 06:32:00: "idk<9><[U:1:1170132017]><Blue>" triggered "player_builtobject" (object "OBJ_SENTRYGUN") (position "339 -419 -255")
L 02/21/2021 - 06:32:08: "amogus gaming<13><[U:1:1089803558]><Blue>" killed "Hacksaw<12><[U:1:68745073]><Red>" with "sniperrifle" (customkill "headshot") (attacker_position "-1104 191 -259") (victim_position "-232 881 -255")
L 02/21/2021 - 06:32:09: "Desmos Calculator<10><[U:1:1132396177]><Red>" killed "idk<9><[U:1:1170132017]><Blue>" with "spy_cicle" (customkill "backstab") (attacker_position "340 -482 -255") (victim_position "341 -428 -255")
L 02/21/2021 - 06:32:09: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "player_builtobject" (object "OBJ_ATTACHMENT_SAPPER") (position "381 -410 -255")
L 02/21/2021 - 06:32:18: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "killedobject" (object "OBJ_SENTRYGUN") (weapon "obj_attachment_sapper") (objectowner "idk<9><[U:1:1170132017]><Blue>") (attacker_position "-1107 260 -261")
L 02/21/2021 - 06:32:22: "Desmos Calculator<10><[U:1:1132396177]><Red>" killed "amogus gaming<13><[U:1:1089803558]><Blue>" with "spy_cicle" (customkill "backstab") (attacker_position "-1895 603 -367") (victim_position "-1894 668 -367")
L 02/21/2021 - 06:32:24: rcon from "23.239.22.163:43208": command "status"
L 02/21/2021 - 06:32:27: "amogus gaming<13><[U:1:1089803558]><Blue>" changed role to "soldier"
L 02/21/2021 - 06:32:31: "idk<9><[U:1:1170132017]><Blue>" triggered "player_builtobject" (object "OBJ_SENTRYGUN") (position "1365 -2168 -447")
L 02/21/2021 - 06:32:41: "Desmos Calculator<10><[U:1:1132396177]><Red>" killed "Dzefersons14<8><[U:1:1080653073]><Blue>" with "revolver" (attacker_position "324 -554 -510") (victim_position "354 -495 -511")
L 02/21/2021 - 06:32:41: "Hacksaw<12><[U:1:68745073]><Red>" triggered "kill assist" against "Dzefersons14<8><[U:1:1080653073]><Blue>" (assister_position "-392 -357 -463") (attacker_position "324 -554 -510") (victim_position "354 -495 -511")
L 02/21/2021 - 06:32:41: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "domination" against "Dzefersons14<8><[U:1:1080653073]><Blue>"
L 02/21/2021 - 06:32:42: "TheShiniestToast<14><[U:1:1162330687]><>" connected, address "98.209.156.58:27005"
L 02/21/2021 - 06:32:43: "TheShiniestToast<14><[U:1:1162330687]><>" STEAM USERID validated
L 02/21/2021 - 06:32:49: "TheShiniestToast<14><[U:1:1162330687]><>" entered the game
L 02/21/2021 - 06:32:50: rcon from "68.144.74.48:64863": command "status"
L 02/21/2021 - 06:32:53: Team "Red" triggered "pointcaptured" (cp "0") (cpname "#koth_viaduct_cap") (numcappers "2") (player1 "Desmos Calculator<10><[U:1:1132396177]><Red>") (position1 "101 -31 -319") (player2 "Hacksaw<12><[U:1:68745073]><Red>") (position2 "98 140 -316") 
L 02/21/2021 - 06:32:54: "TheShiniestToast<14><[U:1:1162330687]><Unassigned>" joined team "Red"
L 02/21/2021 - 06:32:55: "TheShiniestToast<14><[U:1:1162330687]><Red>" changed role to "scout"
L 02/21/2021 - 06:32:58: "idk<9><[U:1:1170132017]><Blue>" triggered "player_carryobject" (object "OBJ_SENTRYGUN") (position "1410 -2135 -447")
L 02/21/2021 - 06:33:01: "Dzefersons14<8><[U:1:1080653073]><Blue>" changed role to "pyro"
L 02/21/2021 - 06:33:18: "Cybermorphic<15><[U:1:901503117]><>" connected, address "71.105.43.248:27005"
L 02/21/2021 - 06:33:18: "Cybermorphic<15><[U:1:901503117]><>" STEAM USERID validated
L 02/21/2021 - 06:33:24: "amogus gaming<13><[U:1:1089803558]><Blue>" killed "TheShiniestToast<14><[U:1:1162330687]><Red>" with "tf_projectile_rocket" (attacker_position "114 192 -47") (victim_position "70 710 -254")
L 02/21/2021 - 06:33:24: rcon from "23.239.22.163:43340": command "status"
L 02/21/2021 - 06:33:28: "Hacksaw<12><[U:1:68745073]><Red>" killed "amogus gaming<13><[U:1:1089803558]><Blue>" with "force_a_nature" (attacker_position "-48 -120 -49") (victim_position "120 93 15")
L 02/21/2021 - 06:33:30: "idk<9><[U:1:1170132017]><Blue>" triggered "player_dropobject" (object "OBJ_SENTRYGUN") (position "880 -152 -255")
L 02/21/2021 - 06:33:30: "idk<9><[U:1:1170132017]><Blue>" triggered "player_builtobject" (object "OBJ_SENTRYGUN") (position "880 -152 -255")
L 02/21/2021 - 06:33:35: "Cybermorphic<15><[U:1:901503117]><>" entered the game
L 02/21/2021 - 06:33:40: "idk<9><[U:1:1170132017]><Blue>" killed "Hacksaw<12><[U:1:68745073]><Red>" with "obj_sentrygun3" (attacker_position "923 -94 -255") (victim_position "124 629 -255")
L 02/21/2021 - 06:33:43: "Cybermorphic<15><[U:1:901503117]><Unassigned>" disconnected (reason "Disconnect by user.")
L 02/21/2021 - 06:33:50: rcon from "68.144.74.48:64880": command "status"
L 02/21/2021 - 06:33:55: "idk<9><[U:1:1170132017]><Blue>" killed "Desmos Calculator<10><[U:1:1132396177]><Red>" with "shotgun_primary" (attacker_position "175 -1159 -255") (victim_position "242 -1599 -254")
L 02/21/2021 - 06:33:55: "Hacksaw<12><[U:1:68745073]><Red>" changed role to "soldier"
L 02/21/2021 - 06:34:08: "Dzefersons14<8><[U:1:1080653073]><Blue>" killed "TheShiniestToast<14><[U:1:1162330687]><Red>" with "flaregun" (attacker_position "-29 33 -313") (victim_position "635 1746 -446")
L 02/21/2021 - 06:34:13: "TheShiniestToast<14><[U:1:1162330687]><Red>" changed role to "spy"
L 02/21/2021 - 06:34:20: Team "Blue" triggered "pointcaptured" (cp "0") (cpname "#koth_viaduct_cap") (numcappers "2") (player1 "Dzefersons14<8><[U:1:1080653073]><Blue>") (position1 "55 41 -313") (player2 "amogus gaming<13><[U:1:1089803558]><Blue>") (position2 "-39 80 -313") 
L 02/21/2021 - 06:34:22: "Hacksaw<12><[U:1:68745073]><Red>" triggered "killedobject" (object "OBJ_SENTRYGUN") (weapon "quake_rl") (objectowner "idk<9><[U:1:1170132017]><Blue>") (attacker_position "-236 1003 -255")
L 02/21/2021 - 06:34:24: rcon from "23.239.22.163:43468": command "status"
L 02/21/2021 - 06:34:25: "amogus gaming<13><[U:1:1089803558]><Blue>" killed "Hacksaw<12><[U:1:68745073]><Red>" with "tf_projectile_rocket" (attacker_position "-314 1049 -219") (victim_position "197 1127 -235")
L 02/21/2021 - 06:34:25: "idk<9><[U:1:1170132017]><Blue>" triggered "kill assist" against "Hacksaw<12><[U:1:68745073]><Red>" (assister_position "899 -188 -255") (attacker_position "-314 1049 -219") (victim_position "197 1127 -235")
L 02/21/2021 - 06:34:50: rcon from "68.144.74.48:64896": command "status"
L 02/21/2021 - 06:34:55: "Dzefersons14<8><[U:1:1080653073]><Blue>" killed "TheShiniestToast<14><[U:1:1162330687]><Red>" with "flaregun" (attacker_position "379 -186 -254") (victim_position "123 -406 -254")
L 02/21/2021 - 06:35:09: "Desmos Calculator<10><[U:1:1132396177]><Red>" killed "idk<9><[U:1:1170132017]><Blue>" with "revolver" (attacker_position "1913 -1080 -455") (victim_position "1391 -1157 -455")
L 02/21/2021 - 06:35:23: "amogus gaming<13><[U:1:1089803558]><Blue>" killed "TheShiniestToast<14><[U:1:1162330687]><Red>" with "tf_projectile_rocket" (attacker_position "-102 1377 -255") (victim_position "197 1433 -235")
L 02/21/2021 - 06:35:24: rcon from "23.239.22.163:43570": command "status"
L 02/21/2021 - 06:35:28: "amogus gaming<13><[U:1:1089803558]><Blue>" killed "Hacksaw<12><[U:1:68745073]><Red>" with "tf_projectile_rocket" (attacker_position "-239 1360 -236") (victim_position "-393 1490 -255")
L 02/21/2021 - 06:35:28: "TheShiniestToast<14><[U:1:1162330687]><Red>" changed role to "scout"
L 02/21/2021 - 06:35:34: "idk<9><[U:1:1170132017]><Blue>" triggered "player_builtobject" (object "OBJ_SENTRYGUN") (position "1422 -2446 -447")
L 02/21/2021 - 06:35:37: "Dzefersons14<8><[U:1:1080653073]><Blue>" killed "Desmos Calculator<10><[U:1:1132396177]><Red>" with "degreaser" (attacker_position "-1806 2139 -447") (victim_position "-1819 2406 -447")
L 02/21/2021 - 06:35:37: "Dzefersons14<8><[U:1:1080653073]><Blue>" triggered "revenge" against "Desmos Calculator<10><[U:1:1132396177]><Red>"
L 02/21/2021 - 06:35:39: "TheShiniestToast<14><[U:1:1162330687]><Red>" killed "amogus gaming<13><[U:1:1089803558]><Blue>" with "force_a_nature" (attacker_position "-1218 2276 -447") (victim_position "-1241 2186 -406")
L 02/21/2021 - 06:35:50: rcon from "68.144.74.48:64915": command "status"
L 02/21/2021 - 06:35:50: "Dzefersons14<8><[U:1:1080653073]><Blue>" killed "TheShiniestToast<14><[U:1:1162330687]><Red>" with "flaregun" (attacker_position "-1667 1313 -447") (victim_position "-1933 1735 -447")
L 02/21/2021 - 06:35:57: "TheShiniestToast<14><[U:1:1162330687]><Red>" say "fucking crit flares"
L 02/21/2021 - 06:35:58: "potato<16><[U:1:385661040]><>" connected, address "76.242.50.231:27005"
L 02/21/2021 - 06:35:58: "potato<16><[U:1:385661040]><>" STEAM USERID validated
L 02/21/2021 - 06:36:01: "Dzefersons14<8><[U:1:1080653073]><Blue>" say "lol"
L 02/21/2021 - 06:36:02: "idk<9><[U:1:1170132017]><Blue>" triggered "player_carryobject" (object "OBJ_SENTRYGUN") (position "1397 -2439 -447")
L 02/21/2021 - 06:36:11: "Dzefersons14<8><[U:1:1080653073]><Blue>" killed "Hacksaw<12><[U:1:68745073]><Red>" with "flaregun" (attacker_position "454 999 -190") (victim_position "544 729 -255")
L 02/21/2021 - 06:36:16: "idk<9><[U:1:1170132017]><Blue>" triggered "player_dropobject" (object "OBJ_SENTRYGUN") (position "935 -180 -255")
L 02/21/2021 - 06:36:16: "idk<9><[U:1:1170132017]><Blue>" triggered "player_builtobject" (object "OBJ_SENTRYGUN") (position "935 -180 -255")
L 02/21/2021 - 06:36:16: "potato<16><[U:1:385661040]><>" entered the game
L 02/21/2021 - 06:36:17: "Cam?<17><[U:1:862652083]><>" connected, address "24.226.90.125:27005"
L 02/21/2021 - 06:36:18: "Cam?<17><[U:1:862652083]><>" STEAM USERID validated
L 02/21/2021 - 06:36:18: "Hacksaw<12><[U:1:68745073]><Red>" changed role to "pyro"
L 02/21/2021 - 06:36:19: "Cam?<17><[U:1:862652083]><>" disconnected (reason "Disconnect by user.")
L 02/21/2021 - 06:36:19: "amogus gaming<13><[U:1:1089803558]><Blue>" killed "Desmos Calculator<10><[U:1:1132396177]><Red>" with "tf_projectile_rocket" (attacker_position "624 755 -206") (victim_position "933 788 -252")
L 02/21/2021 - 06:36:20: "Cam?<18><[U:1:862652083]><>" connected, address "24.226.90.125:27005"
L 02/21/2021 - 06:36:20: "Cam?<18><[U:1:862652083]><>" STEAM USERID validated
L 02/21/2021 - 06:36:21: "amogus gaming<13><[U:1:1089803558]><Blue>" killed "TheShiniestToast<14><[U:1:1162330687]><Red>" with "tf_projectile_rocket" (attacker_position "582 857 -209") (victim_position "566 552 -247")
L 02/21/2021 - 06:36:24: rcon from "23.239.22.163:43696": command "status"
L 02/21/2021 - 06:36:24: "TheShiniestToast<14><[U:1:1162330687]><Red>" disconnected (reason "Disconnect by user.")
L 02/21/2021 - 06:36:39: "potato<16><[U:1:385661040]><Unassigned>" joined team "Red"
L 02/21/2021 - 06:36:48: "potato<16><[U:1:385661040]><Red>" changed role to "scout"
L 02/21/2021 - 06:36:50: rcon from "68.144.74.48:64936": command "status"
L 02/21/2021 - 06:36:52: "idk<9><[U:1:1170132017]><Blue>" triggered "player_carryobject" (object "OBJ_SENTRYGUN") (position "816 -215 -255")
L 02/21/2021 - 06:36:54: "idk<9><[U:1:1170132017]><Blue>" triggered "player_dropobject" (object "OBJ_SENTRYGUN") (position "713 -252 -239")
L 02/21/2021 - 06:36:54: "idk<9><[U:1:1170132017]><Blue>" triggered "player_builtobject" (object "OBJ_SENTRYGUN") (position "713 -252 -239")
L 02/21/2021 - 06:36:57: "amogus gaming<13><[U:1:1089803558]><Blue>" say "let-s genocide"
L 02/21/2021 - 06:36:58: "potato<16><[U:1:385661040]><Red>" killed "amogus gaming<13><[U:1:1089803558]><Blue>" with "pep_brawlerblaster" (attacker_position "174 731 -255") (victim_position "398 694 -255")
L 02/21/2021 - 06:37:02: "idk<9><[U:1:1170132017]><Blue>" killed "potato<16><[U:1:385661040]><Red>" with "obj_sentrygun3" (attacker_position "656 -269 -255") (victim_position "951 -167 -255")
L 02/21/2021 - 06:37:20: World triggered "Round_Overtime"
L 02/21/2021 - 06:37:23: "Hacksaw<12><[U:1:68745073]><Red>" killed "Dzefersons14<8><[U:1:1080653073]><Blue>" with "powerjack" (attacker_position "112 151 -319") (victim_position "129 84 -318")
L 02/21/2021 - 06:37:23: "amogus gaming<13><[U:1:1089803558]><Blue>" say "let them cap"
L 02/21/2021 - 06:37:24: rcon from "23.239.22.163:43818": command "status"
L 02/21/2021 - 06:37:31: "idk<9><[U:1:1170132017]><Blue>" killed "potato<16><[U:1:385661040]><Red>" with "obj_sentrygun3" (attacker_position "1516 -426 -359") (victim_position "-250 137 -292")
L 02/21/2021 - 06:37:38: "idk<9><[U:1:1170132017]><Blue>" triggered "captureblocked" (cp "0") (cpname "#koth_viaduct_cap") (position "256 -106 -270")
L 02/21/2021 - 06:37:41: "idk<9><[U:1:1170132017]><Blue>" triggered "captureblocked" (cp "0") (cpname "#koth_viaduct_cap") (position "225 167 -319")
L 02/21/2021 - 06:37:41: "idk<9><[U:1:1170132017]><Blue>" killed "Hacksaw<12><[U:1:68745073]><Red>" with "wrench" (attacker_position "225 167 -319") (victim_position "215 124 -319")
L 02/21/2021 - 06:37:43: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "player_builtobject" (object "OBJ_ATTACHMENT_SAPPER") (position "655 -365 -223")
L 02/21/2021 - 06:37:45: "idk<9><[U:1:1170132017]><Blue>" triggered "killedobject" (object "OBJ_ATTACHMENT_SAPPER") (weapon "wrench") (objectowner "Desmos Calculator<10><[U:1:1132396177]><Red>") (attacker_position "606 -289 -255")
L 02/21/2021 - 06:37:45: "amogus gaming<13><[U:1:1089803558]><Blue>" say_team "dude"
L 02/21/2021 - 06:37:48: "amogus gaming<13><[U:1:1089803558]><Blue>" say "let them cap"
L 02/21/2021 - 06:37:50: rcon from "68.144.74.48:64954": command "status"
L 02/21/2021 - 06:37:52: "idk<9><[U:1:1170132017]><Blue>" triggered "player_builtobject" (object "OBJ_DISPENSER") (position "677 -314 -223")
L 02/21/2021 - 06:37:53: World triggered "Round_Win" (winner "Blue")
L 02/21/2021 - 06:37:53: World triggered "Round_Length" (seconds "468.30")
L 02/21/2021 - 06:37:53: Team "Red" current score "1" with "3" players
L 02/21/2021 - 06:37:53: Team "Blue" current score "1" with "3" players
L 02/21/2021 - 06:37:53: "Dzefersons14<8><[U:1:1080653073]><Blue>" say "no"
L 02/21/2021 - 06:37:55: "Cam?<18><[U:1:862652083]><>" entered the game
L 02/21/2021 - 06:38:05: "amogus gaming<13><[U:1:1089803558]><Blue>" killed "Hacksaw<12><[U:1:68745073]><Red>" with "tf_projectile_rocket" (attacker_position "-1322 2355 -447") (victim_position "-2058 2283 -334")
L 02/21/2021 - 06:38:07: "idk<9><[U:1:1170132017]><Blue>" killed "potato<16><[U:1:385661040]><Red>" with "obj_sentrygun3" (attacker_position "556 -225 -255") (victim_position "159 -770 -230")
L 02/21/2021 - 06:38:08: World triggered "Round_Start"
L 02/21/2021 - 06:38:10: "Hacksaw<12><[U:1:68745073]><Red>" changed role to "scout"
L 02/21/2021 - 06:38:10: "Cam?<18><[U:1:862652083]><Unassigned>" joined team "Red"
L 02/21/2021 - 06:38:12: "Cam?<18><[U:1:862652083]><Red>" changed role to "demoman"
L 02/21/2021 - 06:38:13: "amogus gaming<13><[U:1:1089803558]><Blue>" say "gotta go"
L 02/21/2021 - 06:38:15: "Desmos Calculator<10><[U:1:1132396177]><Red>" say_team "ringer timr"
L 02/21/2021 - 06:38:16: "amogus gaming<13><[U:1:1089803558]><Blue>" say "bye"
L 02/21/2021 - 06:38:22: "Desmos Calculator<10><[U:1:1132396177]><Red>" say "bye"
L 02/21/2021 - 06:38:24: rcon from "23.239.22.163:43948": command "status"
L 02/21/2021 - 06:38:24: "idk<9><[U:1:1170132017]><Blue>" triggered "player_builtobject" (object "OBJ_SENTRYGUN") (position "1497 -2233 -447")
L 02/21/2021 - 06:38:26: "Gamemaster19<19><[U:1:407075022]><>" connected, address "72.85.24.173:27005"
L 02/21/2021 - 06:38:26: "Gamemaster19<19><[U:1:407075022]><>" STEAM USERID validated
L 02/21/2021 - 06:38:29: "Cam?<18><[U:1:862652083]><Red>" killed "Dzefersons14<8><[U:1:1080653073]><Blue>" with "iron_bomber" (attacker_position "1194 -2098 -192") (victim_position "1553 -2119 -447")
L 02/21/2021 - 06:38:29: "Cam?<18><[U:1:862652083]><Red>" triggered "killedobject" (object "OBJ_SENTRYGUN") (weapon "iron_bomber") (objectowner "idk<9><[U:1:1170132017]><Blue>") (attacker_position "1138 -2095 -455")
L 02/21/2021 - 06:38:29: "Cam?<18><[U:1:862652083]><Red>" killed "idk<9><[U:1:1170132017]><Blue>" with "iron_bomber" (attacker_position "1138 -2095 -455") (victim_position "1569 -2235 -447")
L 02/21/2021 - 06:38:32: "Dzefersons14<8><[U:1:1080653073]><Blue>" killed "Cam?<18><[U:1:862652083]><Red>" with "degreaser" (attacker_position "1553 -2119 -433") (victim_position "1199 -1475 -447")
L 02/21/2021 - 06:38:41: "Gamemaster19<19><[U:1:407075022]><>" entered the game
L 02/21/2021 - 06:38:44: "Gamemaster19<19><[U:1:407075022]><Unassigned>" joined team "Blue"
L 02/21/2021 - 06:38:47: "Gamemaster19<19><[U:1:407075022]><Blue>" changed role to "demoman"
L 02/21/2021 - 06:38:50: rcon from "68.144.74.48:64984": command "status"
L 02/21/2021 - 06:38:51: "idk<9><[U:1:1170132017]><Blue>" triggered "player_builtobject" (object "OBJ_SENTRYGUN") (position "1502 -2273 -447")
L 02/21/2021 - 06:38:55: Team "Red" triggered "pointcaptured" (cp "0") (cpname "#koth_viaduct_cap") (numcappers "1") (player1 "Hacksaw<12><[U:1:68745073]><Red>") (position1 "-28 126 -313") 
L 02/21/2021 - 06:39:00: "Gamemaster19<19><[U:1:407075022]><Blue>" killed "potato<16><[U:1:385661040]><Red>" with "tf_projectile_pipe_remote" (attacker_position "197 -2249 -255") (victim_position "160 -1602 -255")
L 02/21/2021 - 06:39:04: "Dzefersons14<8><[U:1:1080653073]><Blue>" killed "Desmos Calculator<10><[U:1:1132396177]><Red>" with "degreaser" (customkill "feign_death") (attacker_position "1102 -2041 -455") (victim_position "969 -2251 -423")
L 02/21/2021 - 06:39:12: "idk<9><[U:1:1170132017]><Blue>" killed "Cam?<18><[U:1:862652083]><Red>" with "obj_sentrygun2" (attacker_position "1498 -2262 -447") (victim_position "998 -1702 -455")
L 02/21/2021 - 06:39:21: "Gamemaster19<19><[U:1:407075022]><Blue>" killed "Desmos Calculator<10><[U:1:1132396177]><Red>" with "tf_projectile_pipe_remote" (attacker_position "215 -1701 -255") (victim_position "-152 -1183 -235")
L 02/21/2021 - 06:39:24: rcon from "23.239.22.163:44112": command "status"
L 02/21/2021 - 06:39:25: "Hacksaw<12><[U:1:68745073]><Red>" killed "Gamemaster19<19><[U:1:407075022]><Blue>" with "force_a_nature" (attacker_position "-61 -1212 -235") (victim_position "184 -1210 -255")
L 02/21/2021 - 06:39:25: "idk<9><[U:1:1170132017]><Blue>" triggered "player_carryobject" (object "OBJ_SENTRYGUN") (position "1530 -2292 -447")
L 02/21/2021 - 06:39:47: "Hacksaw<12><[U:1:68745073]><Red>" triggered "killedobject" (object "OBJ_SENTRYGUN") (weapon "building_carried_destroyed") (objectowner "idk<9><[U:1:1170132017]><Blue>") (attacker_position "-43 -713 -255")
L 02/21/2021 - 06:39:47: "Hacksaw<12><[U:1:68745073]><Red>" killed "idk<9><[U:1:1170132017]><Blue>" with "pep_pistol" (attacker_position "-43 -713 -255") (victim_position "492 -763 -255")
L 02/21/2021 - 06:39:49: "Dzefersons14<8><[U:1:1080653073]><Blue>" killed "potato<16><[U:1:385661040]><Red>" with "flaregun" (attacker_position "-249 -619 -255") (victim_position "-550 -663 -255")
L 02/21/2021 - 06:39:50: rcon from "68.144.74.48:65014": command "status"
L 02/21/2021 - 06:40:17: "Cam?<18><[U:1:862652083]><Red>" killed "Dzefersons14<8><[U:1:1080653073]><Blue>" with "iron_bomber" (attacker_position "-856 -455 -255") (victim_position "-257 -640 -255")
L 02/21/2021 - 06:40:19: "idk<9><[U:1:1170132017]><Blue>" triggered "player_builtobject" (object "OBJ_SENTRYGUN") (position "820 -2581 -343")
L 02/21/2021 - 06:40:19: "potato<16><[U:1:385661040]><Red>" triggered "captureblocked" (cp "0") (cpname "#koth_viaduct_cap") (position "-163 324 -272")
L 02/21/2021 - 06:40:21: "potato<16><[U:1:385661040]><Red>" killed "Gamemaster19<19><[U:1:407075022]><Blue>" with "pep_brawlerblaster" (attacker_position "120 -321 -255") (victim_position "281 -471 -233")
L 02/21/2021 - 06:40:21: "Hacksaw<12><[U:1:68745073]><Red>" triggered "kill assist" against "Gamemaster19<19><[U:1:407075022]><Blue>" (assister_position "-210 469 -254") (attacker_position "120 -321 -255") (victim_position "281 -471 -233")
L 02/21/2021 - 06:40:24: rcon from "23.239.22.163:44226": command "status"
L 02/21/2021 - 06:40:28: "idk<9><[U:1:1170132017]><Blue>" killed "potato<16><[U:1:385661040]><Red>" with "obj_sentrygun" (attacker_position "717 -2514 -343") (victim_position "382 -2602 -283")
L 02/21/2021 - 06:40:28: "idk<9><[U:1:1170132017]><Blue>" triggered "domination" against "potato<16><[U:1:385661040]><Red>"
L 02/21/2021 - 06:40:50: rcon from "68.144.74.48:65029": command "status"
L 02/21/2021 - 06:40:52: "Hacksaw<12><[U:1:68745073]><Red>" killed "Gamemaster19<19><[U:1:407075022]><Blue>" with "pep_pistol" (attacker_position "-130 56 -25") (victim_position "-4 -605 -255")
L 02/21/2021 - 06:40:52: "Cam?<18><[U:1:862652083]><Red>" triggered "kill assist" against "Gamemaster19<19><[U:1:407075022]><Blue>" (assister_position "190 -158 -205") (attacker_position "-130 56 -25") (victim_position "-4 -605 -255")
L 02/21/2021 - 06:40:56: "Gamemaster19<19><[U:1:407075022]><Blue>" disconnected (reason "Disconnect by user.")
L 02/21/2021 - 06:41:01: "idk<9><[U:1:1170132017]><Blue>" triggered "player_carryobject" (object "OBJ_SENTRYGUN") (position "793 -2611 -343")
L 02/21/2021 - 06:41:08: "amogus gaming<13><[U:1:1089803558]><Blue>" disconnected (reason "#TF_Idle_kicked")
L 02/21/2021 - 06:41:09: "Desmos Calculator<10><[U:1:1132396177]><Red>" killed "Dzefersons14<8><[U:1:1080653073]><Blue>" with "spy_cicle" (customkill "backstab") (attacker_position "232 -2253 -255") (victim_position "225 -2196 -255")
L 02/21/2021 - 06:41:09: "idk<9><[U:1:1170132017]><Blue>" triggered "player_dropobject" (object "OBJ_SENTRYGUN") (position "112 -2622 -255")
L 02/21/2021 - 06:41:09: "idk<9><[U:1:1170132017]><Blue>" triggered "player_builtobject" (object "OBJ_SENTRYGUN") (position "112 -2622 -255")
L 02/21/2021 - 06:41:09: "potato<16><[U:1:385661040]><Red>" killed "idk<9><[U:1:1170132017]><Blue>" with "pep_brawlerblaster" (attacker_position "169 -2603 -255") (victim_position "130 -2569 -255")
L 02/21/2021 - 06:41:09: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "kill assist" against "idk<9><[U:1:1170132017]><Blue>" (assister_position "235 -2207 -254") (attacker_position "169 -2603 -255") (victim_position "130 -2569 -255")
L 02/21/2021 - 06:41:09: "potato<16><[U:1:385661040]><Red>" triggered "revenge" against "idk<9><[U:1:1170132017]><Blue>"
L 02/21/2021 - 06:41:12: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "player_builtobject" (object "OBJ_ATTACHMENT_SAPPER") (position "176 -2597 -255")
L 02/21/2021 - 06:41:14: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "killedobject" (object "OBJ_SENTRYGUN") (weapon "obj_attachment_sapper") (objectowner "idk<9><[U:1:1170132017]><Blue>") (attacker_position "30 -2018 -412")
L 02/21/2021 - 06:41:24: rcon from "23.239.22.163:44364": command "status"
L 02/21/2021 - 06:41:28: "Desmos Calculator<10><[U:1:1132396177]><Red>" committed suicide with "world" (attacker_position "360 -1747 -450")
L 02/21/2021 - 06:41:40: "potato<16><[U:1:385661040]><Red>" killed "idk<9><[U:1:1170132017]><Blue>" with "pep_brawlerblaster" (attacker_position "956 -2813 -343") (victim_position "1185 -2892 -343")
L 02/21/2021 - 06:41:50: rcon from "68.144.74.48:65044": command "status"
L 02/21/2021 - 06:41:55: World triggered "Round_Overtime"
L 02/21/2021 - 06:41:56: "Cam?<18><[U:1:862652083]><Red>" triggered "captureblocked" (cp "0") (cpname "#koth_viaduct_cap") (position "-181 -351 -222")
L 02/21/2021 - 06:41:56: "Cam?<18><[U:1:862652083]><Red>" killed "Dzefersons14<8><[U:1:1080653073]><Blue>" with "iron_bomber" (attacker_position "-181 -351 -222") (victim_position "-97 197 -319")
L 02/21/2021 - 06:41:56: "Hacksaw<12><[U:1:68745073]><Red>" triggered "kill assist" against "Dzefersons14<8><[U:1:1080653073]><Blue>" (assister_position "554 151 -255") (attacker_position "-181 -351 -222") (victim_position "-97 197 -319")
L 02/21/2021 - 06:41:58: World triggered "Round_Win" (winner "Red")
L 02/21/2021 - 06:41:58: World triggered "Round_Length" (seconds "230.01")
L 02/21/2021 - 06:41:58: Team "Red" current score "2" with "4" players
L 02/21/2021 - 06:41:58: Team "Blue" current score "1" with "2" players
L 02/21/2021 - 06:42:02: "idk<9><[U:1:1170132017]><Blue>" changed role to "soldier"
L 02/21/2021 - 06:42:03: "potato<16><[U:1:385661040]><Red>" killed "idk<9><[U:1:1170132017]><Blue>" with "pep_brawlerblaster" (attacker_position "1426 -2416 -447") (victim_position "1526 -2473 -447")
L 02/21/2021 - 06:42:06: "Desmos Calculator<10><[U:1:1132396177]><Red>" joined team "Blue"
L 02/21/2021 - 06:42:06: "Desmos Calculator<10><[U:1:1132396177]><Blue>" committed suicide with "world" (attacker_position "361 -597 -510")
L 02/21/2021 - 06:42:11: "Cam?<18><[U:1:862652083]><Red>" committed suicide with "iron_bomber" (attacker_position "1754 -796 -446")
L 02/21/2021 - 06:42:13: World triggered "Game_Over" reason "Reached Win Limit"
L 02/21/2021 - 06:42:13: Team "Red" final score "2" with "3" players
L 02/21/2021 - 06:42:13: Team "Blue" final score "1" with "3" players
L 02/21/2021 - 06:42:13: Team "RED" triggered "Intermission_Win_Limit"
L 02/21/2021 - 06:42:21: "Cam?<18><[U:1:862652083]><Red>" disconnected (reason "Disconnect by user.")
L 02/21/2021 - 06:42:24: rcon from "23.239.22.163:44506": command "status"
L 02/21/2021 - 06:42:28: server_cvar: "sm_nextmap" "pl_borneo"
L 02/21/2021 - 06:42:33: [META] Loaded 0 plugins (1 already loaded)
L 02/21/2021 - 06:42:33: Log file closed.
`
