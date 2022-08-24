package logparse

import (
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"strings"
	"testing"
	"time"
)

func TestParseTime(t *testing.T) {
	var t0 time.Time
	require.True(t, parseDateTime("02/21/2021 - 06:22:23", &t0))
	require.Equal(t, time.Date(2021, 2, 21, 6, 22, 23, 0, time.UTC), t0)
}

func TestParseAlt(t *testing.T) {
	p := golib.FindFile(path.Join("test_data", "log_1.log"), "gbans")
	f, e := os.ReadFile(p)
	if e != nil {
		t.Fatalf("Failed to open test file: %s", p)
	}
	results := make(map[int]Results)
	for i, line := range strings.Split(string(f), "\n") {
		v := Parse(line)
		results[i] = v
	}
	expected := map[EventType]int{
		SayTeam: 6,
		Say:     18,
	}
	for mt, expectedCount := range expected {
		found := 0
		for _, result := range results {
			if result.MsgType == mt {
				found++
			}
		}
		require.Equal(t, expectedCount, found, "Invalid count for type: %v %d/%d", mt, found, expectedCount)
	}
}

func TestParseUnhandledMsgEvt(t *testing.T) {
	var value1 UnhandledMsgEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: asdf`, IgnoredMsg), &value1))
	require.Equal(t, UnhandledMsgEvt{
		CreatedOn: time.Date(2021, 02, 21, 06, 22, 23, 0, time.UTC),
	}, value1)

	var value2 UnhandledMsgEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: Team "RED" triggered "Intermission_Win_Limit"`, IgnoredMsg), &value2))
	require.EqualValues(t, UnhandledMsgEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)}, value2)

	var value3 UnhandledMsgEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: [META] Loaded 0 plugins (1 already loaded)`, IgnoredMsg), &value3))
	require.EqualValues(t, UnhandledMsgEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)}, value3)
}

func TestParseLogStartEvt(t *testing.T) {
	var value1 LogStartEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: Log file started (file "logs/L0221034.log") (game "/home/tf2server/serverfiles/tf") (version "6300758")`,
		LogStart), &value1))
	require.Equal(t, LogStartEvt{
		EmptyEvt: EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		File:     "logs/L0221034.log", Game: "/home/tf2server/serverfiles/tf", Version: "6300758"}, value1)
}

func TestParseCVAREvt(t *testing.T) {
	var value1 CVAREvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: server_cvar: "sm_nextmap" "pl_frontier_final"`, CVAR), &value1))
	require.Equal(t, CVAREvt{
		EmptyEvt: EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		CVAR:     "sm_nextmap", Value: "pl_frontier_final"}, value1)
}

func TestParseRCONEvt(t *testing.T) {
	var value1 RCONEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: rcon from "23.239.22.163:42004": command "status"`, RCON), &value1))
	require.EqualValues(t, RCONEvt{EmptyEvt: EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)}, Cmd: "status"}, value1)

}

func TestParseEnteredEvt(t *testing.T) {
	var value1 EnteredEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "Hacksaw<12><[U:1:68745073]><>" Entered the game`, Entered), &value1))
	require.EqualValues(t, EnteredEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)}, value1)

}

func TestParseJoinedTeamEvt(t *testing.T) {
	var value1 JoinedTeamEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "Hacksaw<12><[U:1:68745073]><Unassigned>" joined team "Red"`, JoinedTeam), &value1))
	require.EqualValues(t, JoinedTeamEvt{
		EmptyEvt: EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		Team:     RED,
		SourcePlayer: SourcePlayer{
			Name: "Hacksaw", PID: 12, SID: steamid.SID3ToSID64("[U:1:68745073]"), Team: 1,
		}}, value1)
}

func TestParseChangeClassEvt(t *testing.T) {
	var value1 ChangeClassEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "Hacksaw<12><[U:1:68745073]><Red>" changed role to "scout"`, ChangeClass), &value1))
	require.EqualValues(t, ChangeClassEvt{
		EmptyEvt: EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)}, SourcePlayer: SourcePlayer{
			Name: "Hacksaw", PID: 12, SID: steamid.SID3ToSID64("[U:1:68745073]"), Team: RED}, Class: Scout}, value1)

	var value2 ChangeClassEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "var<3><[U:1:204626678]><Blue>" changed role to "scout"`, ChangeClass), &value2))
	require.EqualValues(t, ChangeClassEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "var", PID: 3, SID: steamid.SID3ToSID64("[U:1:204626678]"), Team: BLU},
		Class:        Scout}, value2)
}

func TestParseSuicideEvt(t *testing.T) {
	var value1 SuicideEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "Dzefersons14<8><[U:1:1080653073]><Blue>" committed suicide with "world" (attacker_position "-1189 2513 -423")`, Suicide), &value1))
	require.EqualValues(t, SuicideEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "Dzefersons14", PID: 8, SID: 0x110000140697511, Team: 2},
		Pos:          Pos{X: -1189, Y: 2513, Z: -423},
		Weapon:       World}, value1)

	var value2 SuicideEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "DaDakka!<3602><[U:1:911555463]><Blue>" committed suicide with "world" (attacker_position "1537 7316 -268")`, Suicide), &value2))
	require.EqualValues(t, SuicideEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "DaDakka!", PID: 3602, SID: steamid.SID3ToSID64("[U:1:911555463]"), Team: BLU},
		Pos:          Pos{X: 1537, Y: 7316, Z: -268},
		Weapon:       World,
	}, value2)
}

func TestParseWRoundStartEvt(t *testing.T) {
	var value1 WRoundStartEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: World triggered "Round_Start"`, WRoundStart), &value1))
	require.EqualValues(t, WRoundStartEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)}, value1)
}

func TestParseMedicDeathEvt(t *testing.T) {
	var value1 MedicDeathEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "medic_death" against "Dzefersons14<8><[U:1:1080653073]><Blue>" (healing "135") (ubercharge "0")`, MedicDeath), &value1))
	require.EqualValues(t, MedicDeathEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "Desmos Calculator", PID: 10, SID: 0x1100001437efe91, Team: 1},
		TargetPlayer: TargetPlayer{Name2: "Dzefersons14", PID2: 8,
			SID2: steamid.SID3ToSID64("[U:1:1080653073]"), Team2: BLU,
		},
		Healing: 135,
		Uber:    0}, value1)
}

func TestParseKilledEvt(t *testing.T) {
	var value1 KilledEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "Desmos Calculator<10><[U:1:1132396177]><Red>" killed "Dzefersons14<8><[U:1:1080653073]><Blue>" with "brass_beast" (attacker_position "217 -54 -302") (victim_position "203 -2 -319")`, Killed), &value1))
	require.EqualValues(t, KilledEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "Desmos Calculator", PID: 10, SID: 0x1100001437efe91, Team: 1},
		TargetPlayer: TargetPlayer{Name2: "Dzefersons14", PID2: 8,
			SID2: steamid.SID3ToSID64("[U:1:1080653073]"), Team2: BLU,
		},
		APos:   Pos{X: 217, Y: -54, Z: -302},
		VPos:   Pos{X: 203, Y: -2, Z: -319},
		Weapon: BrassBeast}, value1)

	var value2 KilledEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "Five<636><[U:1:66374745]><Blue>" killed "2-D<658><[U:1:126712178]><Red>" with "scattergun" (attacker_position "803 -693 -235") (victim_position "663 -899 -165")`, Killed), &value2))
	require.EqualValues(t, KilledEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "Five", PID: 636, SID: steamid.SID3ToSID64("[U:1:66374745]"), Team: BLU},
		TargetPlayer: TargetPlayer{Name2: "2-D", PID2: 658, SID2: steamid.SID3ToSID64("[U:1:126712178]"), Team2: RED},
		APos:         Pos{X: 803, Y: -693, Z: -235},
		VPos:         Pos{X: 663, Y: -899, Z: -165},
		Weapon:       Scattergun,
	}, value2)
}

func TestParseCustomKilledEvt(t *testing.T) {
	var value1 CustomKilledEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "Desmos Calculator<10><[U:1:1132396177]><Red>" killed "Dzefersons14<8><[U:1:1080653073]><Blue>" with "spy_cicle" (customkill "backstab") (attacker_position "217 -54 -302") (victim_position "203 -2 -319")`, KilledCustom), &value1))
	require.EqualValues(t, CustomKilledEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "Desmos Calculator", PID: 10, SID: 0x1100001437efe91, Team: 1},
		TargetPlayer: TargetPlayer{Name2: "Dzefersons14", PID2: 8,
			SID2: steamid.SID3ToSID64("[U:1:1080653073]"), Team2: BLU,
		},
		APos:       Pos{X: 217, Y: -54, Z: -302},
		VPos:       Pos{X: 203, Y: -2, Z: -319},
		Weapon:     Spycicle,
		CustomKill: "backstab"}, value1)
}

func TestParseKillAssistEvt(t *testing.T) {
	var value1 KillAssistEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "Hacksaw<12><[U:1:68745073]><Red>" triggered "kill assist" against "Dzefersons14<8><[U:1:1080653073]><Blue>" (assister_position "-476 154 -254") (attacker_position "217 -54 -302") (victim_position "203 -2 -319")`, KillAssist), &value1))
	require.EqualValues(t, KillAssistEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "Hacksaw", PID: 12, SID: 0x11000010418f771, Team: 1},
		TargetPlayer: TargetPlayer{Name2: "Dzefersons14", PID2: 8,
			SID2: steamid.SID3ToSID64("[U:1:1080653073]"), Team2: BLU,
		},
		ASPos: Pos{X: -476, Y: 154, Z: -254},
		APos:  Pos{X: 217, Y: -54, Z: -302},
		VPos:  Pos{X: 203, Y: -2, Z: -319}}, value1)

}

func TestParsePointCapturedEvt(t *testing.T) {
	var value1 PointCapturedEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: Team "Red" triggered "pointcaptured" (cp "0") (cpname "#koth_viaduct_cap") (numcappers "1") (player1 "Hacksaw<12><[U:1:68745073]><Red>") (position1 "101 98 -313")`, PointCaptured), &value1))
	require.EqualValues(t, PointCapturedEvt{
		EmptyEvt: EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		Team:     RED, CP: 0, CPName: "#koth_viaduct_cap", NumCappers: 1,
		Player1: "Hacksaw<12><[U:1:68745073]><Red>", Position1: Pos{X: 101, Y: 98, Z: -313}}, value1)

}

func TestParseConnectedEvt(t *testing.T) {
	var value1 ConnectedEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "amogus gaming<13><[U:1:1089803558]><>" Connected, address "139.47.95.130:47949"`, Connected), &value1))
	require.EqualValues(t, ConnectedEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "amogus gaming", PID: 13, SID: 0x110000140f51526, Team: 0},
		Address:      "139.47.95.130",
		Port:         47949,
	}, value1)

}
func TestParseEmptyEvt(t *testing.T) {
	var value1 EmptyEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "amogus gaming<13><[U:1:1089803558]><>" STEAM USERID Validated`, Validated), &value1))
	require.EqualValues(t, EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)}, value1)

}
func TestParseKilledObjectEvt(t *testing.T) {
	var value1 KilledObjectEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "killedobject" (object "OBJ_SENTRYGUN") (weapon "obj_attachment_sapper") (objectowner "idk<9><[U:1:1170132017]><Blue>") (attacker_position "2 -579 -255")`, KilledObject), &value1))
	require.EqualValues(t, KilledObjectEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "Desmos Calculator", PID: 10, SID: 0x1100001437efe91, Team: 1},
		TargetPlayer: TargetPlayer{Name2: "idk", PID2: 9, SID2: 76561199130397745, Team2: BLU},
		Object:       "OBJ_SENTRYGUN",
		Weapon:       Sapper,
		APos:         Pos{X: 2, Y: -579, Z: -255}}, value1)
}

func TestParseCarryObjectEvt(t *testing.T) {
	var value1 CarryObjectEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "idk<9><[U:1:1170132017]><Blue>" triggered "player_carryobject" (object "OBJ_SENTRYGUN") (position "1074 -2279 -423")`, CarryObject), &value1))
	require.EqualValues(t, CarryObjectEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "idk", PID: 9, SID: 0x110000145becc31, Team: 2},
		Object:       "OBJ_SENTRYGUN", Pos: Pos{X: 1074, Y: -2279, Z: -423}}, value1)
}

func TestParseDropObjectEvt(t *testing.T) {
	var value1 DropObjectEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "idk<9><[U:1:1170132017]><Blue>" triggered "player_dropobject" (object "OBJ_SENTRYGUN") (position "339 -419 -255")`, DropObject), &value1))
	require.EqualValues(t, DropObjectEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "idk", PID: 9, SID: 0x110000145becc31, Team: 2},
		Object:       "OBJ_SENTRYGUN", Pos: Pos{X: 339, Y: -419, Z: -255}}, value1)

}

func TestParseBuiltObjectEvt(t *testing.T) {
	var value1 BuiltObjectEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "idk<9><[U:1:1170132017]><Blue>" triggered "player_builtobject" (object "OBJ_SENTRYGUN") (position "880 -152 -255")`, BuiltObject), &value1))
	require.EqualValues(t, BuiltObjectEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "idk", PID: 9, SID: 0x110000145becc31, Team: 2},
		Object:       "OBJ_SENTRYGUN",
		Pos:          Pos{X: 880, Y: -152, Z: -255},
	}, value1)
}

func TestParseWRoundWinEvt(t *testing.T) {
	var value1 WRoundWinEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: World triggered "Round_Win" (winner "Red")`, WRoundWin), &value1))
	require.EqualValues(t, WRoundWinEvt{
		EmptyEvt: EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		Winner:   RED}, value1)
}

func TestParseWRoundLenEvt(t *testing.T) {
	var value1 WRoundLenEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: World triggered "Round_Length" (seconds "398.10")`, WRoundLen), &value1))
	require.EqualValues(t, WRoundLenEvt{
		EmptyEvt: EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		Length:   398.10}, value1)

}
func TestParseWTeamScoreEvt(t *testing.T) {
	var value1 WTeamScoreEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: Team "Red" current score "1" with "2" players`, WTeamScore), &value1))
	require.EqualValues(t, WTeamScoreEvt{
		EmptyEvt: EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		Team:     RED, Score: 1, Players: 2}, value1)

}
func TestParseSayEvt(t *testing.T) {
	var value1 SayEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "Hacksaw<12><[U:1:68745073]><Red>" say "gg"`, Say), &value1))
	require.EqualValues(t, SayEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "Hacksaw", PID: 12, SID: 0x11000010418f771, Team: 1},
		Msg:          "gg"}, value1)

}

func TestParseSayTeamEvt(t *testing.T) {
	var value1 SayTeamEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "Desmos Calculator<10><[U:1:1132396177]><Red>" say_team "gg"`, SayTeam), &value1))
	require.EqualValues(t, SayTeamEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "Desmos Calculator", PID: 10, SID: 0x1100001437efe91, Team: 1},
		Msg:          "gg"}, value1)
}

func TestParseDominationEvt(t *testing.T) {
	var value1 DominationEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "Domination" against "Dzefersons14<8><[U:1:1080653073]><Blue>"`, Domination), &value1))
	require.EqualValues(t, DominationEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "Desmos Calculator", PID: 10, SID: 0x1100001437efe91, Team: 1},
		TargetPlayer: TargetPlayer{Name2: "Dzefersons14", PID2: 8, SID2: steamid.SID3ToSID64("[U:1:1080653073]"), Team2: BLU}}, value1)
}

func TestParseDisconnectedEvt(t *testing.T) {
	var value1 DisconnectedEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "Cybermorphic<15><[U:1:901503117]><Unassigned>" Disconnected (reason "Disconnect by user.")`, Disconnected), &value1))
	require.EqualValues(t, DisconnectedEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "Cybermorphic", PID: 15, SID: 0x110000135bbd88d, Team: 0},
		Reason:       "Disconnect by user."}, value1)

	var value2 DisconnectedEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "Imperi<248><[U:1:1008044562]><Red>" disconnected (reason "Client left game (Steam auth ticket has been canceled)`, Disconnected), &value2))
	require.EqualValues(t, DisconnectedEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "Imperi", PID: 248, SID: steamid.SID3ToSID64("[U:1:1008044562]"), Team: RED},
		Reason:       "Client left game (Steam auth ticket has been canceled)"}, value2)
}

func TestParseRevengeEvt(t *testing.T) {
	var value1 RevengeEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "Dzefersons14<8><[U:1:1080653073]><Blue>" triggered "Revenge" against "Desmos Calculator<10><[U:1:1132396177]><Red>"`, Revenge), &value1))
	require.EqualValues(t, RevengeEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "Dzefersons14", PID: 8, SID: 0x110000140697511, Team: 2},
		TargetPlayer: TargetPlayer{
			Name2: "Desmos Calculator", PID2: 10,
			SID2: steamid.SID3ToSID64("[U:1:1132396177]"), Team2: RED}}, value1)
}

func TestParseWRoundOvertimeEvt(t *testing.T) {
	var value1 WRoundOvertimeEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: World triggered "Round_Overtime"`, WRoundOvertime), &value1))
	require.EqualValues(t, WRoundOvertimeEvt{
		CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)}, value1)

}

func TestParseCaptureBlockedEvt(t *testing.T) {
	var value1 CaptureBlockedEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "potato<16><[U:1:385661040]><Red>" triggered "captureblocked" (cp "0") (cpname "#koth_viaduct_cap") (position "-163 324 -272")`, CaptureBlocked), &value1))
	require.EqualValues(t, CaptureBlockedEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "potato", PID: 16, SID: 0x110000116fcb870, Team: 1},
		CP:           0,
		CPName:       "#koth_viaduct_cap",
		Pos:          Pos{X: -163, Y: 324, Z: -272}}, value1)
}

func TestParseWGameOverEvt(t *testing.T) {
	var value1 WGameOverEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: World triggered "Game_Over" reason "Reached Win Limit"`, WGameOver), &value1))
	require.EqualValues(t, WGameOverEvt{
		EmptyEvt: EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		Reason:   "Reached Win Limit"}, value1)
}

func TestParseWTeamFinalScoreEvt(t *testing.T) {
	var value1 WTeamFinalScoreEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: Team "Red" final score "2" with "3" players`, WTeamFinalScore), &value1))
	require.EqualValues(t, WTeamFinalScoreEvt{
		EmptyEvt: EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		Score:    2,
		Players:  3}, value1)
}

func TestParseLogStopEvt(t *testing.T) {
	var value1 LogStopEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: Log file closed.`, LogStop), &value1))
	require.EqualValues(t, LogStopEvt{
		CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)}, value1)
}

func TestParseWPausedEvt(t *testing.T) {
	var value1 WPausedEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: World triggered "Game_Paused"`, WPaused), &value1))
	require.EqualValues(t, WPausedEvt{
		CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)}, value1)

}

func TestParseWResumedEvt(t *testing.T) {
	var value1 WResumedEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: World triggered "Game_Unpaused"`, WResumed), &value1))
	require.EqualValues(t, WResumedEvt{
		CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)}, value1)

}

func TestParseFirstHealAfterSpawnEvt(t *testing.T) {
	var value1 FirstHealAfterSpawnEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "SCOTTY T<27><[U:1:97282856]><Blue>" triggered "first_heal_after_spawn" (time "1.6")`, FirstHealAfterSpawn), &value1))
	require.EqualValues(t, FirstHealAfterSpawnEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "SCOTTY T", PID: 27, SID: 0x110000105cc6b28, Team: 2}, HealTime: 1.6}, value1)
}

func TestParseChargeReadyEvt(t *testing.T) {
	var value1 ChargeReadyEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "wonder<7><[U:1:34284979]><Red>" triggered "chargeready"`, ChargeReady), &value1))
	require.EqualValues(t, ChargeReadyEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "wonder", PID: 7, SID: 0x1100001020b25b3, Team: 1}}, value1)
}

func TestParseChargeDeployedEvt(t *testing.T) {
	var value1 ChargeDeployedEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "wonder<7><[U:1:34284979]><Red>" triggered "chargedeployed" (medigun "medigun")`, ChargeDeployed), &value1))
	require.EqualValues(t, ChargeDeployedEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "wonder", PID: 7, SID: 0x1100001020b25b3, Team: 1},
		Medigun:      Uber}, value1)
}

func TestParseChargeEndedEvt(t *testing.T) {
	var value1 ChargeEndedEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "wonder<7><[U:1:34284979]><Red>" triggered "chargeended" (duration "7.5")`, ChargeEnded), &value1))
	require.EqualValues(t, ChargeEndedEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "wonder", PID: 7, SID: 0x1100001020b25b3, Team: 1},
		Duration:     7.5}, value1)

}

func TestParseMedicDeathExEvt(t *testing.T) {
	var value1 MedicDeathExEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "wonder<7><[U:1:34284979]><Red>" triggered "medic_death_ex" (uberpct "32")`, MedicDeathEx), &value1))
	require.Equal(t, MedicDeathExEvt{
		EmptyEvt: EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		UberPct:  32}, value1)
}

func TestParseLostUberAdvantageEvt(t *testing.T) {
	var value1 LostUberAdvantageEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "SEND HELP<16><[U:1:84528002]><Blue>" triggered "lost_uber_advantage" (time "44")`, LostUberAdv), &value1))
	require.Equal(t, LostUberAdvantageEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "SEND HELP", PID: 16, SID: 0x11000010509cb82, Team: 2},
		AdvTime:      44,
	}, value1)

}

func TestParseEmptyUberEvt(t *testing.T) {
	var value1 EmptyUberEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "Kwq<9><[U:1:96748980]><Blue>" triggered "empty_uber"`, EmptyUber), &value1))
	require.Equal(t, EmptyUberEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "Kwq", PID: 9, SID: 0x110000105c445b4, Team: 2}}, value1)
}

func TestParsePickupEvt(t *testing.T) {
	var value1 PickupEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "g о а т z<13><[U:1:41435165]><Red>" picked up item "ammopack_small"`, Pickup), &value1))
	require.EqualValues(t, PickupEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "g о а т z", PID: 13, SID: 0x11000010278401d, Team: 1},
		Item:         ItemAmmoSmall,
		Healing:      0,
	}, value1)
	var value2 PickupEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "g о а т z<13><[U:1:41435165]><Red>" picked up item "medkit_medium" (healing "47")`, Pickup), &value2))
	require.EqualValues(t, PickupEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "g о а т z", PID: 13, SID: 0x11000010278401d, Team: 1},
		Item:         ItemHPMedium,
		Healing:      47,
	}, value2)
}

func TestParseShotFiredEvt(t *testing.T) {
	var value1 ShotFiredEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "rad<6><[U:1:57823119]><Red>" triggered "shot_fired" (weapon "syringegun_medic")`, ShotFired), &value1))
	require.EqualValues(t, ShotFiredEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "rad", PID: 6, SID: 0x110000103724f8f, Team: 1},
		Weapon:       SyringeGun,
	}, value1)
}

func TestParseShotHitEvt(t *testing.T) {
	var value1 ShotHitEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "z/<14><[U:1:66656848]><Blue>" triggered "shot_hit" (weapon "blackbox")`, ShotHit), &value1))
	require.EqualValues(t, ShotHitEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "z/", PID: 14, SID: 0x110000103f91a50, Team: 2},
		Weapon:       Blackbox,
	}, value1)
}

func TestParseDamageEvt(t *testing.T) {
	var value1 DamageEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "rad<6><[U:1:57823119]><Red>" triggered "damage" against "z/<14><[U:1:66656848]><Blue>" (damage "11") (weapon "syringegun_medic")`, Damage), &value1))
	require.EqualValues(t, DamageEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "rad", PID: 6, SID: 0x110000103724f8f, Team: 1},
		TargetPlayer: TargetPlayer{Name2: "z/", PID2: 14, SID2: steamid.SID3ToSID64("[U:1:66656848]"), Team2: BLU},
		Weapon:       SyringeGun,
		Damage:       11,
	}, value1)

	var value2 DamageEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "rad<6><[U:1:57823119]><Red>" triggered "damage" against "z/<14><[U:1:66656848]><Blue>" (damage "88") (realdamage "32") (weapon "ubersaw") (healing "110")`, Damage), &value2))
	require.EqualValues(t, DamageEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "rad", PID: 6, SID: 0x110000103724f8f, Team: 1},
		TargetPlayer: TargetPlayer{Name2: "z/", PID2: 14, SID2: steamid.SID3ToSID64("[U:1:66656848]"), Team2: BLU},
		Damage:       88,
		RealDamage:   32,
		Weapon:       Ubersaw,
		Healing:      110,
	}, value2)

	var value3 DamageEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "Lochlore<22><[U:1:127176886]><Blue>" triggered "damage" against "Doctrine<20><[U:1:1090182064]><Red>" (damage "762") (realdamage "127") (weapon "knife") (crit "crit")`, Damage), &value3))
	require.EqualValues(t, DamageEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "Lochlore", PID: 22, SID: steamid.SID3ToSID64("[U:1:127176886]"), Team: BLU},
		TargetPlayer: TargetPlayer{Name2: "Doctrine", PID2: 20, SID2: steamid.SID3ToSID64("[U:1:1090182064]"), Team2: RED},
		Damage:       762,
		RealDamage:   127,
		Weapon:       Knife,
		Crit:         Crit,
	}, value3)
}

func TestParseJarateAttackEvt(t *testing.T) {
	var value1 JarateAttackEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "Banfield<2796><[U:1:958890744]><Blue>" triggered "jarate_attack" against "Legs™<2818><[U:1:42871337]><Red>" with "tf_weapon_jar" (attacker_position "1881 -1521 264") (victim_position "1729 -301 457")`, JarateAttack), &value1))
	require.EqualValues(t, JarateAttackEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "Banfield", PID: 2796, SID: steamid.SID3ToSID64("[U:1:958890744]"), Team: BLU},
		TargetPlayer: TargetPlayer{Name2: "Legs™", PID2: 2818, SID2: steamid.SID3ToSID64("[U:1:42871337]"), Team2: RED},
		Weapon:       JarBased,
		APos:         Pos{X: 1881, Y: -1521, Z: 264},
		VPos:         Pos{X: 1729, Y: -301, Z: 457},
	}, value1)
}

func TestParseWMiniRoundWinEvt(t *testing.T) {
	var value1 WMiniRoundWinEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: World triggered "Mini_Round_Win" (winner "Blue") (round "round_b")`, WMiniRoundWin), &value1))
	require.EqualValues(t, WMiniRoundWinEvt{
		CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)}, value1)

}

func TestParseWMiniRoundLenEvt(t *testing.T) {
	var value1 WMiniRoundLenEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: World triggered "Mini_Round_Length" (seconds "340.62")`, WMiniRoundLen), &value1))
	require.EqualValues(t, WMiniRoundLenEvt{
		CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)}, value1)
}

func TestParsWRoundSetupBeginEvt(t *testing.T) {
	var value1 WRoundSetupBeginEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: World triggered "Round_Setup_Begin"`, WRoundSetupBegin), &value1))
	require.EqualValues(t, WRoundSetupBeginEvt{
		CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)}, value1)
}

func TestParseWMiniRoundSelectedEvt(t *testing.T) {
	var value1 WMiniRoundSelectedEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: World triggered "Mini_Round_Selected" (round "Round_A")`, WMiniRoundSelected), &value1))
	require.EqualValues(t, WMiniRoundSelectedEvt{
		CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)}, value1)
}

func TestParseWMiniRoundStartEvt(t *testing.T) {
	var value1 WMiniRoundStartEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: World triggered "Mini_Round_Start"`, WMiniRoundStart), &value1))
	require.EqualValues(t, WMiniRoundStartEvt{
		CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)}, value1)
}

func TestParseMilkAttackEvt(t *testing.T) {
	var value1 MilkAttackEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "✪lil vandal<2953><[U:1:178417727]><Blue>" triggered "milk_attack" against "Darth Jar Jar<2965><[U:1:209106507]><Red>" with "tf_weapon_jar" (attacker_position "-1040 -854 128") (victim_position "-1516 -382 128")`, MilkAttack), &value1))
	require.EqualValues(t, MilkAttackEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "✪lil vandal", PID: 2953, SID: steamid.SID3ToSID64("[U:1:178417727]"), Team: BLU},
		TargetPlayer: TargetPlayer{Name2: "Darth Jar Jar", PID2: 2965, SID2: steamid.SID3ToSID64("[U:1:209106507]"), Team2: RED},
		Weapon:       JarBased,
		APos:         Pos{X: -1040, Y: -854, Z: 128},
		VPos:         Pos{X: -1516, Y: -382, Z: 128},
	}, value1)
}

func TestParseGasAttackEvt(t *testing.T) {
	var value1 MilkAttackEvt
	require.NoError(t, Unmarshal(pt(t, `L 02/21/2021 - 06:22:23: "UnEpic<6760><[U:1:132169058]><Blue>" triggered "gas_attack" against "Johnny Blaze<6800><[U:1:33228413]><Red>" with "tf_weapon_jar" (attacker_position "-4539 2731 156") (victim_position "-4384 1527 128")`, GasAttack), &value1))
	require.EqualValues(t, GasAttackEvt{
		EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		SourcePlayer: SourcePlayer{Name: "UnEpic", PID: 6760, SID: steamid.SID3ToSID64("[U:1:132169058]"), Team: BLU},
		TargetPlayer: TargetPlayer{Name2: "Johnny Blaze", PID2: 6800, SID2: steamid.SID3ToSID64("[U:1:33228413]"), Team2: RED},
		Weapon:       JarBased,
		APos:         Pos{X: -4539, Y: 2731, Z: 156},
		VPos:         Pos{X: -4384, Y: 1527, Z: 128},
	}, value1)
}

func pt(t *testing.T, s string, msgType EventType) map[string]any {
	v := Parse(s)
	require.Equal(t, msgType, v.MsgType)
	return v.Values
}

//func TestParse(t *testing.T) {
// TODO
//L 02/05/2022 - 06:39:37: Team "RED" triggered "Intermission_Win_Limit"
//L 02/05/2022 - 01:48:25: "Cudgeon<7124><[U:1:89643594]><Red>" triggered "flagevent" (event "dropped") (position "218 -601 -250")
//L 02/05/2022 - 01:49:51: World triggered "Round_Stalemate"
//L 02/05/2022 - 01:50:19: Started map "pl_frontier_final" (CRC "4a7ee13f724b4abd41219f056539c53c")
//L 02/05/2022 - 01:50:33: "Myst<291><[U:1:102591589]><Red>" disconnected (reason "Client left game (Steam auth ticket has been canceled)
//")
//}

func TestParseKVs(t *testing.T) {
	m1 := map[string]any{}
	require.True(t, parseKVs(`(damage "88") (realdamage "32") (weapon "ubersaw") (healing "110")`, m1))
	require.Equal(t, map[string]any{"damage": "88", "realdamage": "32", "weapon": "ubersaw", "healing": "110"}, m1)

	m2 := map[string]any{}
	require.True(t, parseKVs(`L 01/16/2022 - 21:24:11: Team "Red" triggered "pointcaptured" (cp "0") (cpname "#koth_viaduct_cap") (numcappers "2") (player1 "cube elegy<15><[U:1:84002473]><Red>") (position1 "-156 -105 1601") (player2 "bink<24><[U:1:164995715]><Red>") (position2 "57 78 1602")`, m2))
	require.Equal(t, map[string]any{
		"cp":         "0",
		"cpname":     "#koth_viaduct_cap",
		"numcappers": "2",
		"player1":    "cube elegy<15><[U:1:84002473]><Red>",
		"position1":  "-156 -105 1601",
		"player2":    "bink<24><[U:1:164995715]><Red>",
		"position2":  "57 78 1602",
	}, m2)
}
