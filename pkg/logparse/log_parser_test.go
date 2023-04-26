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

func TestParseSourcePlayer(t *testing.T) {
	// (player1 "var<3><[U:1:204626678]><Blue>") (position1 "194 60 -767")
	var src SourcePlayer
	require.True(t, parseSourcePlayer("var<3><[U:1:204626678]><Blue>", &src))
	require.Equal(t, src.SID, steamid.SID3ToSID64("[U:1:204626678]"))
}

func TestParseAlt(t *testing.T) {
	p := golib.FindFile(path.Join("test_data", "log_1.log"), "gbans")
	f, e := os.ReadFile(p)
	if e != nil {
		t.Fatalf("Failed to open test file: %s", p)
	}
	results := make(map[int]*Results)
	for i, line := range strings.Split(string(f), "\n") {
		v, err := Parse(line)
		require.NoError(t, err)
		results[i] = v
	}
	expected := map[EventType]int{
		SayTeam: 6,
		Say:     18,
	}
	for mt, expectedCount := range expected {
		found := 0
		for _, result := range results {
			if result.EventType == mt {
				found++
			}
		}
		require.Equal(t, expectedCount, found, "Invalid count for type: %v %d/%d", mt, found, expectedCount)
	}
}

func TestParseUnhandledMsgEvt(t *testing.T) {
	m := `L 02/21/2021 - 06:22:23: asdf`
	testLogLine(t, m, UnhandledMsgEvt{
		EmptyEvt: EmptyEvt{CreatedOn: time.Date(2021, 02, 21, 06, 22, 23, 0, time.UTC)},
		Message:  m})
}

func testLogLine(t *testing.T, line string, expected any) {
	value1, err := Parse(line)
	require.NoError(t, err, "Failed to parse log line: %s", line)
	require.EqualValues(t, expected, value1.Event, "Value mismatch")
}

func TestParseLogStartEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: Log file started (file "logs/L0221034.log") (game "/home/tf2server/serverfiles/tf") (version "6300758")`, LogStartEvt{
		EmptyEvt: EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		File:     "logs/L0221034.log", Game: "/home/tf2server/serverfiles/tf", Version: "6300758"})
}

func TestParseCVAREvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: server_cvar: "sm_nextmap" "pl_frontier_final"`, CVAREvt{
		EmptyEvt: EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		CVAR:     "sm_nextmap", Value: "pl_frontier_final"})
}

func TestParseRCONEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: rcon from "23.239.22.163:42004": command "status"`,
		RCONEvt{EmptyEvt: EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			Cmd: "status"})
}

func TestParseEnteredEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "Hacksaw<12><[U:1:68745073]><>" Entered the game`,
		EnteredEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "Hacksaw", PID: 12, SID: steamid.SID3ToSID64("[U:1:68745073]"), Team: UNASSIGNED}})
}

func TestParseJoinedTeamEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "Hacksaw<12><[U:1:68745073]><Unassigned>" joined team "Red"`,
		JoinedTeamEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			Team:         RED,
			SourcePlayer: SourcePlayer{Name: "Hacksaw", PID: 12, SID: steamid.SID3ToSID64("[U:1:68745073]"), Team: SPEC}})
}

func TestParseChangeClassEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "Hacksaw<12><[U:1:68745073]><Red>" changed role to "scout"`,
		ChangeClassEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "Hacksaw", PID: 12, SID: steamid.SID3ToSID64("[U:1:68745073]"), Team: RED},
			Class:        Scout})
	testLogLine(t, `L 02/21/2021 - 06:22:23: "var<3><[U:1:204626678]><Blue>" changed role to "scout"`,
		ChangeClassEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "var", PID: 3, SID: steamid.SID3ToSID64("[U:1:204626678]"), Team: BLU},
			Class:        Scout})
}

func TestParseSuicideEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "Dzefersons14<8><[U:1:1080653073]><Blue>" committed suicide with "world" (attacker_position "-1189 2513 -423")`,
		SuicideEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "Dzefersons14", PID: 8, SID: 0x110000140697511, Team: BLU},
			Pos:          Pos{X: -1189, Y: 2513, Z: -423},
			Weapon:       World})

	testLogLine(t, `L 02/21/2021 - 06:22:23: "DaDakka!<3602><[U:1:911555463]><Blue>" committed suicide with "world" (attacker_position "1537 7316 -268")`,
		SuicideEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "DaDakka!", PID: 3602, SID: steamid.SID3ToSID64("[U:1:911555463]"), Team: BLU},
			Pos:          Pos{X: 1537, Y: 7316, Z: -268},
			Weapon:       World})
}

func TestParseWRoundStartEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: World triggered "Round_Start"`,
		WRoundStartEvt{
			CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)})
}

func TestParseMedicDeathEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "medic_death" against "Dzefersons14<8><[U:1:1080653073]><Blue>" (healing "135") (ubercharge "0")`,
		MedicDeathEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "Desmos Calculator", PID: 10, SID: 0x1100001437efe91, Team: RED},
			TargetPlayer: TargetPlayer{Name2: "Dzefersons14", PID2: 8, SID2: steamid.SID3ToSID64("[U:1:1080653073]"), Team2: BLU},
			Healing:      135,
			HadUber:      false})
	testLogLine(t, `L 02/21/2021 - 06:22:23: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "medic_death" against "Dzefersons14<8><[U:1:1080653073]><Blue>" (healing "135") (ubercharge "1")`,
		MedicDeathEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "Desmos Calculator", PID: 10, SID: 0x1100001437efe91, Team: RED},
			TargetPlayer: TargetPlayer{Name2: "Dzefersons14", PID2: 8, SID2: steamid.SID3ToSID64("[U:1:1080653073]"), Team2: BLU},
			Healing:      135,
			HadUber:      true})
}

func TestParseKilledEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "Desmos Calculator<10><[U:1:1132396177]><Red>" killed "Dzefersons14<8><[U:1:1080653073]><Blue>" with "brass_beast" (attacker_position "217 -54 -302") (victim_position "203 -2 -319")`,
		KilledEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "Desmos Calculator", PID: 10, SID: 0x1100001437efe91, Team: RED},
			TargetPlayer: TargetPlayer{Name2: "Dzefersons14", PID2: 8, SID2: steamid.SID3ToSID64("[U:1:1080653073]"), Team2: BLU},
			APos:         Pos{X: 217, Y: -54, Z: -302},
			VPos:         Pos{X: 203, Y: -2, Z: -319},
			Weapon:       BrassBeast})
	testLogLine(t, `L 02/21/2021 - 06:22:23: "Five<636><[U:1:66374745]><Blue>" killed "2-D<658><[U:1:126712178]><Red>" with "scattergun" (attacker_position "803 -693 -235") (victim_position "663 -899 -165")`,
		KilledEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "Five", PID: 636, SID: steamid.SID3ToSID64("[U:1:66374745]"), Team: BLU},
			TargetPlayer: TargetPlayer{Name2: "2-D", PID2: 658, SID2: steamid.SID3ToSID64("[U:1:126712178]"), Team2: RED},
			APos:         Pos{X: 803, Y: -693, Z: -235},
			VPos:         Pos{X: 663, Y: -899, Z: -165},
			Weapon:       Scattergun,
		})
}

func TestParseCustomKilledEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "Desmos Calculator<10><[U:1:1132396177]><Red>" killed "Dzefersons14<8><[U:1:1080653073]><Blue>" with "spy_cicle" (customkill "backstab") (attacker_position "217 -54 -302") (victim_position "203 -2 -319")`,
		CustomKilledEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "Desmos Calculator", PID: 10, SID: 0x1100001437efe91, Team: RED},
			TargetPlayer: TargetPlayer{Name2: "Dzefersons14", PID2: 8, SID2: steamid.SID3ToSID64("[U:1:1080653073]"), Team2: BLU},
			APos:         Pos{X: 217, Y: -54, Z: -302},
			VPos:         Pos{X: 203, Y: -2, Z: -319},
			Weapon:       Spycicle,
			CustomKill:   "backstab"})
}

func TestParseKillAssistEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "Hacksaw<12><[U:1:68745073]><Red>" triggered "kill assist" against "Dzefersons14<8><[U:1:1080653073]><Blue>" (assister_position "-476 154 -254") (attacker_position "217 -54 -302") (victim_position "203 -2 -319")`,
		KillAssistEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "Hacksaw", PID: 12, SID: 0x11000010418f771, Team: RED},
			TargetPlayer: TargetPlayer{Name2: "Dzefersons14", PID2: 8,
				SID2: steamid.SID3ToSID64("[U:1:1080653073]"), Team2: BLU,
			},
			ASPos: Pos{X: -476, Y: 154, Z: -254},
			APos:  Pos{X: 217, Y: -54, Z: -302},
			VPos:  Pos{X: 203, Y: -2, Z: -319}})
}

func TestParsePointCapturedEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: Team "Red" triggered "pointcaptured" (cp "0") (cpname "#koth_viaduct_cap") (numcappers "1") (player1 "Hacksaw<12><[U:1:68745073]><Red>") (position1 "101 98 -313")`,
		PointCapturedEvt{
			EmptyEvt: EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			Team:     RED, CP: 0, CPName: "#koth_viaduct_cap", NumCappers: 1,
			Player1: "Hacksaw<12><[U:1:68745073]><Red>", Position1: Pos{X: 101, Y: 98, Z: -313}})

}

func TestParseConnectedEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "amogus gaming<13><[U:1:1089803558]><>" Connected, address "139.47.95.130:47949"`,
		ConnectedEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "amogus gaming", PID: 13, SID: 0x110000140f51526, Team: 0},
			Address:      "139.47.95.130",
			Port:         47949})
}

//func TestParseEmptyEvt(t *testing.T) {
//	testLogLine(t, `L 02/21/2021 - 06:22:23: "amogus gaming<13><[U:1:1089803558]><>" STEAM USERID Validated`,
//		EmptyEvt{
//			CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)})
//
//}

func TestParseKilledObjectEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "killedobject" (object "OBJ_SENTRYGUN") (weapon "obj_attachment_sapper") (objectowner "idk<9><[U:1:1170132017]><Blue>") (attacker_position "2 -579 -255")`,
		KilledObjectEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "Desmos Calculator", PID: 10, SID: 0x1100001437efe91, Team: RED},
			TargetPlayer: TargetPlayer{Name2: "idk", PID2: 9, SID2: 76561199130397745, Team2: BLU},
			Object:       "OBJ_SENTRYGUN",
			Weapon:       Sapper,
			APos:         Pos{X: 2, Y: -579, Z: -255}})
	testLogLine(t, `L 02/21/2021 - 06:22:23: "Uncle Grain<387><BOT><Red>" triggered "killedobject" (object "OBJ_ATTACHMENT_SAPPER") (weapon "wrench") (objectowner "Doug<382><[U:1:1203081575]><Blue>") (attacker_position "-6889 -1367 -63")`,
		KilledObjectEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "Uncle Grain", PID: 387, SID: BotSid, Team: RED},
			TargetPlayer: TargetPlayer{Name2: "Doug", PID2: 382, SID2: steamid.SID3ToSID64("[U:1:1203081575]"), Team2: BLU},
			Object:       "OBJ_ATTACHMENT_SAPPER",
			Weapon:       Wrench,
			APos:         Pos{X: -6889, Y: -1367, Z: -63}})
}

func TestParseCarryObjectEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "idk<9><[U:1:1170132017]><Blue>" triggered "player_carryobject" (object "OBJ_SENTRYGUN") (position "1074 -2279 -423")`,
		CarryObjectEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "idk", PID: 9, SID: 0x110000145becc31, Team: BLU},
			Object:       "OBJ_SENTRYGUN", Pos: Pos{X: 1074, Y: -2279, Z: -423}})
}

func TestParseDropObjectEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "idk<9><[U:1:1170132017]><Blue>" triggered "player_dropobject" (object "OBJ_SENTRYGUN") (position "339 -419 -255")`,
		DropObjectEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "idk", PID: 9, SID: 0x110000145becc31, Team: BLU},
			Object:       "OBJ_SENTRYGUN", Pos: Pos{X: 339, Y: -419, Z: -255}})

}

func TestParseBuiltObjectEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "idk<9><[U:1:1170132017]><Blue>" triggered "player_builtobject" (object "OBJ_SENTRYGUN") (position "880 -152 -255")`,
		BuiltObjectEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "idk", PID: 9, SID: 0x110000145becc31, Team: BLU},
			Object:       "OBJ_SENTRYGUN",
			Pos:          Pos{X: 880, Y: -152, Z: -255},
		})
}

func TestParseWRoundWinEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: World triggered "Round_Win" (winner "Red")`,
		WRoundWinEvt{
			EmptyEvt: EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			Winner:   RED})
}

func TestParseWRoundLenEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: World triggered "Round_Length" (seconds "398.10")`,
		WRoundLenEvt{
			EmptyEvt: EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			Length:   398.10})

}
func TestParseWTeamScoreEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: Team "Red" current score "1" with "2" players`,
		WTeamScoreEvt{
			EmptyEvt: EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			Team:     RED, Score: 1, Players: 2})

}
func TestParseSayEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "Hacksaw<12><[U:1:68745073]><Red>" say "gg"`,
		SayEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "Hacksaw", PID: 12, SID: 0x11000010418f771, Team: RED},
			Msg:          "gg"})

}

func TestParseSayTeamEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "Desmos Calculator<10><[U:1:1132396177]><Red>" say_team "gg"`,
		SayTeamEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "Desmos Calculator", PID: 10, SID: 0x1100001437efe91, Team: RED},
			Msg:          "gg"})
}

func TestParseDominationEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "Domination" against "Dzefersons14<8><[U:1:1080653073]><Blue>"`,
		DominationEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "Desmos Calculator", PID: 10, SID: 0x1100001437efe91, Team: RED},
			TargetPlayer: TargetPlayer{Name2: "Dzefersons14", PID2: 8, SID2: steamid.SID3ToSID64("[U:1:1080653073]"), Team2: BLU}})
}

func TestParseDisconnectedEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "Cybermorphic<15><[U:1:901503117]><Unassigned>" Disconnected (reason "Disconnect by user.")`,
		DisconnectedEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "Cybermorphic", PID: 15, SID: 0x110000135bbd88d, Team: SPEC},
			Reason:       "Disconnect by user."})

	testLogLine(t, `L 02/21/2021 - 06:22:23: "Imperi<248><[U:1:1008044562]><Red>" disconnected (reason "Client left game (Steam auth ticket has been canceled)`,
		DisconnectedEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "Imperi", PID: 248, SID: steamid.SID3ToSID64("[U:1:1008044562]"), Team: RED},
			Reason:       "Client left game (Steam auth ticket has been canceled)"})
}

func TestParseRevengeEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "Dzefersons14<8><[U:1:1080653073]><Blue>" triggered "Revenge" against "Desmos Calculator<10><[U:1:1132396177]><Red>"`,
		RevengeEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "Dzefersons14", PID: 8, SID: 0x110000140697511, Team: BLU},
			TargetPlayer: TargetPlayer{Name2: "Desmos Calculator", PID2: 10, SID2: steamid.SID3ToSID64("[U:1:1132396177]"), Team2: RED}})
}

func TestParseWRoundOvertimeEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: World triggered "Round_Overtime"`,
		WRoundOvertimeEvt{
			CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)})

}

func TestParseCaptureBlockedEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "potato<16><[U:1:385661040]><Red>" triggered "captureblocked" (cp "0") (cpname "#koth_viaduct_cap") (position "-163 324 -272")`,
		CaptureBlockedEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "potato", PID: 16, SID: 0x110000116fcb870, Team: RED},
			CP:           0,
			CPName:       "#koth_viaduct_cap",
			Pos:          Pos{X: -163, Y: 324, Z: -272}},
	)
}

func TestParseWGameOverEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: World triggered "Game_Over" reason "Reached Win Limit"`,
		WGameOverEvt{
			EmptyEvt: EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			Reason:   "Reached Win Limit"})
}

func TestParseWTeamFinalScoreEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: Team "Red" final score "2" with "3" players`,
		WTeamFinalScoreEvt{
			EmptyEvt: EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			Score:    2,
			Players:  3})
}

func TestParseLogStopEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: Log file closed.`,
		LogStopEvt{
			CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)})
}

func TestParseWPausedEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: World triggered "Game_Paused"`,
		WPausedEvt{
			CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)})
}

func TestParseWResumedEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: World triggered "Game_Unpaused"`,
		WResumedEvt{
			CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)})
}

func TestParseFirstHealAfterSpawnEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "SCOTTY T<27><[U:1:97282856]><Blue>" triggered "first_heal_after_spawn" (time "1.6")`,
		FirstHealAfterSpawnEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "SCOTTY T", PID: 27, SID: 0x110000105cc6b28, Team: BLU}, HealTime: 1.6})
}

func TestParseChargeReadyEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "wonder<7><[U:1:34284979]><Red>" triggered "chargeready"`,
		ChargeReadyEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "wonder", PID: 7, SID: 0x1100001020b25b3, Team: RED}})
}

func TestParseChargeDeployedEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "wonder<7><[U:1:34284979]><Red>" triggered "chargedeployed" (medigun "medigun")`,
		ChargeDeployedEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "wonder", PID: 7, SID: 0x1100001020b25b3, Team: RED},
			Medigun:      Uber})
}

func TestParseChargeEndedEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "wonder<7><[U:1:34284979]><Red>" triggered "chargeended" (duration "7.5")`,
		ChargeEndedEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "wonder", PID: 7, SID: 0x1100001020b25b3, Team: RED},
			Duration:     7.5})
}

func TestParseMedicDeathExEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "wonder<7><[U:1:34284979]><Red>" triggered "medic_death_ex" (uberpct "32")`,
		MedicDeathExEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "wonder", PID: 7, SID: steamid.SID3ToSID64("[U:1:34284979]"), Team: RED},
			UberPct:      32})
}

func TestParseLostUberAdvantageEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "SEND HELP<16><[U:1:84528002]><Blue>" triggered "lost_uber_advantage" (time "44")`,
		LostUberAdvantageEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "SEND HELP", PID: 16, SID: 0x11000010509cb82, Team: BLU},
			AdvTime:      44,
		})
}

func TestParseEmptyUberEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "Kwq<9><[U:1:96748980]><Blue>" triggered "empty_uber"`,
		EmptyUberEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "Kwq", PID: 9, SID: 0x110000105c445b4, Team: BLU}})
}

func TestParsePickupEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "g о а т z<13><[U:1:41435165]><Red>" picked up item "ammopack_small"`,
		PickupEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "g о а т z", PID: 13, SID: 0x11000010278401d, Team: RED},
			Item:         ItemAmmoSmall,
			Healing:      0,
		})
	testLogLine(t, `L 02/21/2021 - 06:22:23: "g о а т z<13><[U:1:41435165]><Red>" picked up item "medkit_medium" (healing "47")`,
		PickupEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "g о а т z", PID: 13, SID: 0x11000010278401d, Team: RED},
			Item:         ItemHPMedium,
			Healing:      47,
		})
}

func TestParseShotFiredEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "rad<6><[U:1:57823119]><Red>" triggered "shot_fired" (weapon "syringegun_medic")`,
		ShotFiredEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "rad", PID: 6, SID: 0x110000103724f8f, Team: RED},
			Weapon:       SyringeGun,
		})
}

func TestParseShotHitEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "z/<14><[U:1:66656848]><Blue>" triggered "shot_hit" (weapon "blackbox")`,
		ShotHitEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "z/", PID: 14, SID: 0x110000103f91a50, Team: BLU},
			Weapon:       Blackbox,
		})
}

func TestParseDamageEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "rad<6><[U:1:57823119]><Red>" triggered "damage" against "z/<14><[U:1:66656848]><Blue>" (damage "11") (weapon "syringegun_medic")`,
		DamageEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "rad", PID: 6, SID: 0x110000103724f8f, Team: RED},
			TargetPlayer: TargetPlayer{Name2: "z/", PID2: 14, SID2: steamid.SID3ToSID64("[U:1:66656848]"), Team2: BLU},
			Weapon:       SyringeGun,
			Damage:       11})
	testLogLine(t, `L 02/21/2021 - 06:22:23: "rad<6><[U:1:57823119]><Red>" triggered "damage" against "z/<14><[U:1:66656848]><Blue>" (damage "88") (realdamage "32") (weapon "ubersaw") (healing "110")`,
		DamageEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "rad", PID: 6, SID: 0x110000103724f8f, Team: RED},
			TargetPlayer: TargetPlayer{Name2: "z/", PID2: 14, SID2: steamid.SID3ToSID64("[U:1:66656848]"), Team2: BLU},
			Damage:       88,
			RealDamage:   32,
			Weapon:       Ubersaw,
			Healing:      110})
	testLogLine(t, `L 02/21/2021 - 06:22:23: "Lochlore<22><[U:1:127176886]><Blue>" triggered "damage" against "Doctrine<20><[U:1:1090182064]><Red>" (damage "762") (realdamage "127") (weapon "knife") (crit "crit")`,
		DamageEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "Lochlore", PID: 22, SID: steamid.SID3ToSID64("[U:1:127176886]"), Team: BLU},
			TargetPlayer: TargetPlayer{Name2: "Doctrine", PID2: 20, SID2: steamid.SID3ToSID64("[U:1:1090182064]"), Team2: RED},
			Damage:       762,
			RealDamage:   127,
			Weapon:       Knife,
			Crit:         Crit})
}

func TestParseJarateAttackEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "Banfield<2796><[U:1:958890744]><Blue>" triggered "jarate_attack" against "Legs™<2818><[U:1:42871337]><Red>" with "tf_weapon_jar" (attacker_position "1881 -1521 264") (victim_position "1729 -301 457")`,
		JarateAttackEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "Banfield", PID: 2796, SID: steamid.SID3ToSID64("[U:1:958890744]"), Team: BLU},
			TargetPlayer: TargetPlayer{Name2: "Legs™", PID2: 2818, SID2: steamid.SID3ToSID64("[U:1:42871337]"), Team2: RED},
			Weapon:       JarBased,
			APos:         Pos{X: 1881, Y: -1521, Z: 264},
			VPos:         Pos{X: 1729, Y: -301, Z: 457}})
}

func TestParseWMiniRoundWinEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: World triggered "Mini_Round_Win" (winner "Blue") (round "round_b")`,
		WMiniRoundWinEvt{
			CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)})
}

func TestParseWMiniRoundLenEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: World triggered "Mini_Round_Length" (seconds "340.62")`,
		WMiniRoundLenEvt{
			CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)})
}

func TestParsWRoundSetupBeginEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: World triggered "Round_Setup_Begin"`,
		WRoundSetupBeginEvt{
			CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)})
}

func TestParseWMiniRoundSelectedEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: World triggered "Mini_Round_Selected" (round "Round_A")`,
		WMiniRoundSelectedEvt{
			CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)})
}

func TestParseWMiniRoundStartEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: World triggered "Mini_Round_Start"`,
		WMiniRoundStartEvt{
			CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)})
}

func TestParseMilkAttackEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "✪lil vandal<2953><[U:1:178417727]><Blue>" triggered "milk_attack" against "Darth Jar Jar<2965><[U:1:209106507]><Red>" with "tf_weapon_jar" (attacker_position "-1040 -854 128") (victim_position "-1516 -382 128")`,
		MilkAttackEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "✪lil vandal", PID: 2953, SID: steamid.SID3ToSID64("[U:1:178417727]"), Team: BLU},
			TargetPlayer: TargetPlayer{Name2: "Darth Jar Jar", PID2: 2965, SID2: steamid.SID3ToSID64("[U:1:209106507]"), Team2: RED},
			Weapon:       JarBased,
			APos:         Pos{X: -1040, Y: -854, Z: 128},
			VPos:         Pos{X: -1516, Y: -382, Z: 128}})
}

func TestParseGasAttackEvt(t *testing.T) {
	testLogLine(t, `L 02/21/2021 - 06:22:23: "UnEpic<6760><[U:1:132169058]><Blue>" triggered "gas_attack" against "Johnny Blaze<6800><[U:1:33228413]><Red>" with "tf_weapon_jar" (attacker_position "-4539 2731 156") (victim_position "-4384 1527 128")`,
		GasAttackEvt{
			EmptyEvt:     EmptyEvt{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: SourcePlayer{Name: "UnEpic", PID: 6760, SID: steamid.SID3ToSID64("[U:1:132169058]"), Team: BLU},
			TargetPlayer: TargetPlayer{Name2: "Johnny Blaze", PID2: 6800, SID2: steamid.SID3ToSID64("[U:1:33228413]"), Team2: RED},
			Weapon:       JarBased,
			APos:         Pos{X: -4539, Y: 2731, Z: 156},
			VPos:         Pos{X: -4384, Y: 1527, Z: 128},
		})
}

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
