package logparse_test

import (
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func TestParseDateTime(t *testing.T) {
	t.Parallel()

	var t0 time.Time

	require.True(t, logparse.ParseDateTime("02/21/2021 - 06:22:23", &t0))
	require.Equal(t, time.Date(2021, 2, 21, 6, 22, 23, 0, time.UTC), t0)
}

func TestParseSourcePlayer(t *testing.T) {
	t.Parallel()

	// (player1 "var<3><[U:1:204626678]><Blue>") (position1 "194 60 -767")
	var src logparse.SourcePlayer

	require.True(t, logparse.ParseSourcePlayer("var<3><[U:1:204626678]><Blue>", &src))
	require.Equal(t, steamid.New("[U:1:204626678]"), src.SID)
}

func TestParseFile(t *testing.T) {
	t.Parallel()

	logFilePath := path.Join("testdata", "log_1.log")

	openFile, e := os.ReadFile(logFilePath)
	if e != nil {
		t.Fatalf("Failed to open test file: %s", logFilePath)
	}

	var (
		parser  = logparse.NewLogParser()
		results = make(map[int]*logparse.Results)
	)

	for i, line := range strings.Split(string(openFile), "\n") {
		v, err := parser.Parse(line)
		require.NoError(t, err)

		results[i] = v
	}

	expected := map[logparse.EventType]int{
		logparse.SayTeam: 6,
		logparse.Say:     18,
	}

	for event, expectedCount := range expected {
		found := 0

		for _, result := range results {
			if result.EventType == event {
				found++
			}
		}

		require.Equal(t, expectedCount, found, "Invalid count for type: %v %d/%d", event, found, expectedCount)
	}
}

func TestParseUnhandledMsgEvt(t *testing.T) {
	t.Parallel()

	m := `L 02/21/2021 - 06:22:23: asdf`
	testLogLine(t, m, logparse.IgnoredMsgEvt{
		TimeStamp: logparse.TimeStamp{CreatedOn: time.Date(2021, 0o2, 21, 0o6, 22, 23, 0, time.UTC)},
		Message:   m,
	})
}

func testLogLine(t *testing.T, line string, expected any) {
	t.Helper()

	parser := logparse.NewLogParser()
	value1, err := parser.Parse(line)
	require.NoError(t, err, "Failed to parse log line: %s", line)
	require.Equal(t, expected, value1.Event, "Value mismatch")
}

func TestParseLogStartEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: Log file started (file "logs/L0221034.log") (game "/home/tf2server/serverfiles/tf") (version "6300758")`, logparse.LogStartEvt{
		TimeStamp: logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		File:      "logs/L0221034.log", Game: "/home/tf2server/serverfiles/tf", Version: "6300758",
	})
}

func TestParseCVAREvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: server_cvar: "sm_nextmap" "pl_frontier_final"`, logparse.CVAREvt{
		TimeStamp: logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		CVAR:      "sm_nextmap", Value: "pl_frontier_final",
	})
}

func TestParseRCONEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: rcon from "23.239.22.163:42004": command "status"`,
		logparse.RCONEvt{
			TimeStamp: logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			Cmd:       "status",
		})
}

func TestParseEnteredEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "Hacksaw<12><[U:1:68745073]><>" Entered the game`,
		logparse.EnteredEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "Hacksaw", PID: 12, SID: steamid.New("[U:1:68745073]"), Team: logparse.UNASSIGNED},
		})
}

func TestParseJoinedTeamEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "Hacksaw<12><[U:1:68745073]><Unassigned>" joined team "Red"`,
		logparse.JoinedTeamEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			NewTeam:      logparse.RED,
			SourcePlayer: logparse.SourcePlayer{Name: "Hacksaw", PID: 12, SID: steamid.New("[U:1:68745073]"), Team: logparse.SPEC},
		})
}

func TestParseChangeClassEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "Hacksaw<12><[U:1:68745073]><Red>" changed role to "scout"`,
		logparse.ChangeClassEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "Hacksaw", PID: 12, SID: steamid.New("[U:1:68745073]"), Team: logparse.RED},
			Class:        logparse.Scout,
		})
	testLogLine(t, `L 02/21/2021 - 06:22:23: "var<3><[U:1:204626678]><Blue>" changed role to "scout"`,
		logparse.ChangeClassEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "var", PID: 3, SID: steamid.New("[U:1:204626678]"), Team: logparse.BLU},
			Class:        logparse.Scout,
		})
}

func TestParseSuicideEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "Dzefersons14<8><[U:1:1080653073]><Blue>" committed suicide with "world" (attacker_position "-1189 2513 -423")`,
		logparse.SuicideEvt{
			TimeStamp:        logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer:     logparse.SourcePlayer{Name: "Dzefersons14", PID: 8, SID: steamid.New("[U:1:1080653073]"), Team: logparse.BLU},
			AttackerPosition: logparse.Pos{X: -1189, Y: 2513, Z: -423},
			Weapon:           logparse.World,
		})

	testLogLine(t, `L 02/21/2021 - 06:22:23: "DaDakka!<3602><[U:1:911555463]><Blue>" committed suicide with "world" (attacker_position "1537 7316 -268")`,
		logparse.SuicideEvt{
			TimeStamp:        logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer:     logparse.SourcePlayer{Name: "DaDakka!", PID: 3602, SID: steamid.New("[U:1:911555463]"), Team: logparse.BLU},
			AttackerPosition: logparse.Pos{X: 1537, Y: 7316, Z: -268},
			Weapon:           logparse.World,
		})
}

func TestParseWRoundStartEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: World triggered "Round_Start"`,
		logparse.WRoundStartEvt{
			CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC),
		})
}

func TestParseMedicDeathEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "medic_death" against "Dzefersons14<8><[U:1:1080653073]><Blue>" (healing "135") (ubercharge "0")`,
		logparse.MedicDeathEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "Desmos Calculator", PID: 10, SID: steamid.New("[U:1:1132396177]"), Team: logparse.RED},
			TargetPlayer: logparse.TargetPlayer{Name2: "Dzefersons14", PID2: 8, SID2: steamid.New("[U:1:1080653073]"), Team2: logparse.BLU},
			Healing:      135,
			Ubercharge:   false,
		})
	testLogLine(t, `L 02/21/2021 - 06:22:23: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "medic_death" against "Dzefersons14<8><[U:1:1080653073]><Blue>" (healing "135") (ubercharge "1")`,
		logparse.MedicDeathEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "Desmos Calculator", PID: 10, SID: steamid.New("[U:1:1132396177]"), Team: logparse.RED},
			TargetPlayer: logparse.TargetPlayer{Name2: "Dzefersons14", PID2: 8, SID2: steamid.New("[U:1:1080653073]"), Team2: logparse.BLU},
			Healing:      135,
			Ubercharge:   true,
		})
}

func TestParseKilledEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "Desmos Calculator<10><[U:1:1132396177]><Red>" killed "Dzefersons14<8><[U:1:1080653073]><Blue>" with "brass_beast" (attacker_position "217 -54 -302") (victim_position "203 -2 -319")`,
		logparse.KilledEvt{
			TimeStamp:        logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer:     logparse.SourcePlayer{Name: "Desmos Calculator", PID: 10, SID: steamid.New("[U:1:1132396177]"), Team: logparse.RED},
			TargetPlayer:     logparse.TargetPlayer{Name2: "Dzefersons14", PID2: 8, SID2: steamid.New("[U:1:1080653073]"), Team2: logparse.BLU},
			AttackerPosition: logparse.Pos{X: 217, Y: -54, Z: -302},
			VictimPosition:   logparse.Pos{X: 203, Y: -2, Z: -319},
			Weapon:           logparse.BrassBeast,
		})
	testLogLine(t, `L 02/21/2021 - 06:22:23: "Five<636><[U:1:66374745]><Blue>" killed "2-D<658><[U:1:126712178]><Red>" with "scattergun" (attacker_position "803 -693 -235") (victim_position "663 -899 -165")`,
		logparse.KilledEvt{
			TimeStamp:        logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer:     logparse.SourcePlayer{Name: "Five", PID: 636, SID: steamid.New("[U:1:66374745]"), Team: logparse.BLU},
			TargetPlayer:     logparse.TargetPlayer{Name2: "2-D", PID2: 658, SID2: steamid.New("[U:1:126712178]"), Team2: logparse.RED},
			AttackerPosition: logparse.Pos{X: 803, Y: -693, Z: -235},
			VictimPosition:   logparse.Pos{X: 663, Y: -899, Z: -165},
			Weapon:           logparse.Scattergun,
		})
}

func TestParseCustomKilledEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "Desmos Calculator<10><[U:1:1132396177]><Red>" killed "Dzefersons14<8><[U:1:1080653073]><Blue>" with "spy_cicle" (customkill "backstab") (attacker_position "217 -54 -302") (victim_position "203 -2 -319")`,
		logparse.CustomKilledEvt{
			TimeStamp:        logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer:     logparse.SourcePlayer{Name: "Desmos Calculator", PID: 10, SID: steamid.New("[U:1:1132396177]"), Team: logparse.RED},
			TargetPlayer:     logparse.TargetPlayer{Name2: "Dzefersons14", PID2: 8, SID2: steamid.New("[U:1:1080653073]"), Team2: logparse.BLU},
			AttackerPosition: logparse.Pos{X: 217, Y: -54, Z: -302},
			VictimPosition:   logparse.Pos{X: 203, Y: -2, Z: -319},
			Weapon:           logparse.Spycicle,
			Customkill:       "backstab",
		})
}

func TestParseKillAssistEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "Hacksaw<12><[U:1:68745073]><Red>" triggered "kill assist" against "Dzefersons14<8><[U:1:1080653073]><Blue>" (assister_position "-476 154 -254") (attacker_position "217 -54 -302") (victim_position "203 -2 -319")`,
		logparse.KillAssistEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "Hacksaw", PID: 12, SID: steamid.New("[U:1:68745073]"), Team: logparse.RED},
			TargetPlayer: logparse.TargetPlayer{
				Name2: "Dzefersons14", PID2: 8,
				SID2: steamid.New("[U:1:1080653073]"), Team2: logparse.BLU,
			},
			AssisterPosition: logparse.Pos{X: -476, Y: 154, Z: -254},
			AttackerPosition: logparse.Pos{X: 217, Y: -54, Z: -302},
			VictimPosition:   logparse.Pos{X: 203, Y: -2, Z: -319},
		})
}

func TestParsePointCapturedEvt(t *testing.T) {
	t.Parallel()

	evt := logparse.PointCapturedEvt{
		TimeStamp: logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
		Team:      logparse.RED, CP: 0, Cpname: "#koth_viaduct_cap", Numcappers: 2,
		Player1: "Hacksaw<12><[U:1:68745073]><Red>", Position1: logparse.Pos{X: 101, Y: 98, Z: -313},
		Player2: "El Sur<35><[U:1:423376881]><Red>", Position2: logparse.Pos{X: -95, Y: 152, Z: -767},
	}

	testLogLine(t, `L 02/21/2021 - 06:22:23: Team "Red" triggered "pointcaptured" (cp "0") (cpname "#koth_viaduct_cap") (numcappers "2") (player1 "Hacksaw<12><[U:1:68745073]><Red>") (position1 "101 98 -313") (player2 "El Sur<35><[U:1:423376881]><Red>") (position2 "-95 152 -767")`, evt)

	expectedPlayers := []logparse.SourcePlayerPosition{
		{SourcePlayer: logparse.SourcePlayer{Name: "Hacksaw", PID: 12, SID: steamid.New("[U:1:68745073]"), Team: logparse.RED}, Pos: logparse.Pos{X: 101, Y: 98, Z: -313}},
		{SourcePlayer: logparse.SourcePlayer{Name: "El Sur", PID: 35, SID: steamid.New("[U:1:423376881]"), Team: logparse.RED}, Pos: logparse.Pos{X: -95, Y: 152, Z: -767}},
	}

	require.Equal(t, expectedPlayers, evt.Players())
}

func TestParseConnectedEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "amogus gaming<13><[U:1:1089803558]><>" Connected, address "139.47.95.130:47949"`,
		logparse.ConnectedEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "amogus gaming", PID: 13, SID: steamid.New("[U:1:1089803558]"), Team: 0},
			Address:      "139.47.95.130",
			Port:         47949,
		})
}

// func TestParseEmptyEvt(t *testing.T) {
//	testLogLine(t, `L 02/21/2021 - 06:22:23: "amogus gaming<13><[U:1:1089803558]><>" STEAM USERID Validated`,
//		TimeStamp{
//			CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)})
//
//}

func TestParseKilledObjectEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "killedobject" (object "OBJ_SENTRYGUN") (weapon "obj_attachment_sapper") (objectowner "idk<9><[U:1:1170132017]><Blue>") (attacker_position "2 -579 -255")`,
		logparse.KilledObjectEvt{
			TimeStamp:        logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer:     logparse.SourcePlayer{Name: "Desmos Calculator", PID: 10, SID: steamid.New("[U:1:1132396177]"), Team: logparse.RED},
			TargetPlayer:     logparse.TargetPlayer{Name2: "idk", PID2: 9, SID2: steamid.New("[U:1:1170132017]"), Team2: logparse.BLU},
			Object:           "OBJ_SENTRYGUN",
			Weapon:           logparse.Sapper,
			AttackerPosition: logparse.Pos{X: 2, Y: -579, Z: -255},
		})
	testLogLine(t, `L 02/21/2021 - 06:22:23: "Uncle Grain<387><BOT><Red>" triggered "killedobject" (object "OBJ_ATTACHMENT_SAPPER") (weapon "wrench") (objectowner "Doug<382><[U:1:1203081575]><Blue>") (attacker_position "-6889 -1367 -63")`,
		logparse.KilledObjectEvt{
			TimeStamp:        logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer:     logparse.SourcePlayer{Name: "Uncle Grain", PID: 387, SID: logparse.BotSid, Team: logparse.RED},
			TargetPlayer:     logparse.TargetPlayer{Name2: "Doug", PID2: 382, SID2: steamid.New("[U:1:1203081575]"), Team2: logparse.BLU},
			Object:           "OBJ_ATTACHMENT_SAPPER",
			Weapon:           logparse.Wrench,
			AttackerPosition: logparse.Pos{X: -6889, Y: -1367, Z: -63},
		})
}

func TestParseCarryObjectEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "idk<9><[U:1:1170132017]><Blue>" triggered "player_carryobject" (object "OBJ_SENTRYGUN") (position "1074 -2279 -423")`,
		logparse.CarryObjectEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "idk", PID: 9, SID: steamid.New("[U:1:1170132017]"), Team: logparse.BLU},
			Object:       "OBJ_SENTRYGUN", Position: logparse.Pos{X: 1074, Y: -2279, Z: -423},
		})
}

func TestParseDropObjectEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "idk<9><[U:1:1170132017]><Blue>" triggered "player_dropobject" (object "OBJ_SENTRYGUN") (position "339 -419 -255")`,
		logparse.DropObjectEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "idk", PID: 9, SID: steamid.New("[U:1:1170132017]"), Team: logparse.BLU},
			Object:       "OBJ_SENTRYGUN", Position: logparse.Pos{X: 339, Y: -419, Z: -255},
		})
}

func TestParseBuiltObjectEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "idk<9><[U:1:1170132017]><Blue>" triggered "player_builtobject" (object "OBJ_SENTRYGUN") (position "880 -152 -255")`,
		logparse.BuiltObjectEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "idk", PID: 9, SID: steamid.New("[U:1:1170132017]"), Team: logparse.BLU},
			Object:       "OBJ_SENTRYGUN",
			Position:     logparse.Pos{X: 880, Y: -152, Z: -255},
		})
}

func TestParseWRoundWinEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: World triggered "Round_Win" (winner "Red")`,
		logparse.WRoundWinEvt{
			TimeStamp: logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			Winner:    logparse.RED,
		})
}

func TestParseWRoundLenEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: World triggered "Round_Length" (seconds "398.10")`,
		logparse.WRoundLenEvt{
			TimeStamp: logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			Seconds:   398.10,
		})
}

func TestParseWTeamScoreEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: Team "Red" current score "1" with "2" players`,
		logparse.WTeamScoreEvt{
			TimeStamp: logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			Team:      logparse.RED, Score: 1, Players: 2,
		})
}

func TestParseSayEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "Hacksaw<12><[U:1:68745073]><Red>" say "gg"`,
		logparse.SayEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "Hacksaw", PID: 12, SID: steamid.New("[U:1:68745073]"), Team: logparse.RED},
			Msg:          "gg",
			Team:         false,
		})
}

func TestParseWIntermissionWinLimitEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: Team "RED" triggered "Intermission_Win_Limit"`,
		logparse.WIntermissionWinLimitEvt{
			TimeStamp: logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			Team:      logparse.RED,
		})
}

func TestParseSayTeamEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "Desmos Calculator<10><[U:1:1132396177]><Red>" say_team "gg"`,
		logparse.SayEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "Desmos Calculator", PID: 10, SID: steamid.New("[U:1:1132396177]"), Team: logparse.RED},
			Msg:          "gg",
			Team:         true,
		})
}

func TestParseDominationEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "Desmos Calculator<10><[U:1:1132396177]><Red>" triggered "Domination" against "Dzefersons14<8><[U:1:1080653073]><Blue>"`,
		logparse.DominationEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "Desmos Calculator", PID: 10, SID: steamid.New("[U:1:1132396177]"), Team: logparse.RED},
			TargetPlayer: logparse.TargetPlayer{Name2: "Dzefersons14", PID2: 8, SID2: steamid.New("[U:1:1080653073]"), Team2: logparse.BLU},
		})
}

func TestParseDisconnectedEvt(t *testing.T) {
	t.Parallel()

	// testLogLine(t, `L 02/21/2021 - 06:22:23: "Imperi<248><[U:1:1008044562]><Red>" disconnected (reason "Client left game (Steam auth ticket has been canceled)`,
	//	logparse.DisconnectedEvt{
	//		TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
	//		SourcePlayer: logparse.SourcePlayer{Name: "Imperi", PID: 248, SID: steamid.SID3ToSID64("[U:1:1008044562]"), Team: logparse.RED},
	//		Reason:       "Client left game (Steam auth ticket has been canceled)",
	//	})

	testLogLine(t, `L 02/21/2021 - 06:22:23: "Cybermorphic<15><[U:1:901503117]><Unassigned>" Disconnected (reason "Disconnect by user.")`,
		logparse.DisconnectedEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "Cybermorphic", PID: 15, SID: steamid.New("[U:1:901503117]"), Team: logparse.SPEC},
			Reason:       "Disconnect by user.",
		})
}

func TestParseRevengeEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "Dzefersons14<8><[U:1:1080653073]><Blue>" triggered "Revenge" against "Desmos Calculator<10><[U:1:1132396177]><Red>"`,
		logparse.RevengeEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "Dzefersons14", PID: 8, SID: steamid.New("[U:1:1080653073]"), Team: logparse.BLU},
			TargetPlayer: logparse.TargetPlayer{Name2: "Desmos Calculator", PID2: 10, SID2: steamid.New("[U:1:1132396177]"), Team2: logparse.RED},
		})
}

func TestParseWRoundOvertimeEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: World triggered "Round_Overtime"`,
		logparse.WRoundOvertimeEvt{
			CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC),
		})
}

func TestParseCaptureBlockedEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "potato<16><[U:1:385661040]><Red>" triggered "captureblocked" (cp "0") (cpname "#koth_viaduct_cap") (position "-163 324 -272")`,
		logparse.CaptureBlockedEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "potato", PID: 16, SID: steamid.New("[U:1:385661040]"), Team: logparse.RED},
			CP:           0,
			Cpname:       "#koth_viaduct_cap",
			Position:     logparse.Pos{X: -163, Y: 324, Z: -272},
		},
	)
}

func TestParseWGameOverEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: World triggered "Game_Over" reason "Reached Win Limit"`,
		logparse.WGameOverEvt{
			TimeStamp: logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			Reason:    "Reached Win Limit",
		})
}

func TestParseWTeamFinalScoreEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: Team "Red" final score "2" with "3" players`,
		logparse.WTeamFinalScoreEvt{
			TimeStamp: logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			Score:     2,
			Players:   3,
			Team:      logparse.RED,
		})
}

func TestParseLogStopEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: Log file closed.`,
		logparse.LogStopEvt{
			CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC),
		})
}

func TestParseWPausedEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: World triggered "Game_Paused"`,
		logparse.WPausedEvt{
			CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC),
		})
}

func TestParseWResumedEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: World triggered "Game_Unpaused"`,
		logparse.WResumedEvt{
			CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC),
		})
}

func TestParseFirstHealAfterSpawnEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "SCOTTY T<27><[U:1:97282856]><Blue>" triggered "first_heal_after_spawn" (time "1.6")`,
		logparse.FirstHealAfterSpawnEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "SCOTTY T", PID: 27, SID: steamid.New("[U:1:97282856]"), Team: logparse.BLU}, Time: 1.6,
		})
}

func TestParseChargeReadyEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "wonder<7><[U:1:34284979]><Red>" triggered "chargeready"`,
		logparse.ChargeReadyEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "wonder", PID: 7, SID: steamid.New("[U:1:34284979]"), Team: logparse.RED},
		})
}

func TestParseChargeDeployedEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "wonder<7><[U:1:34284979]><Red>" triggered "chargedeployed" (medigun "medigun")`,
		logparse.ChargeDeployedEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "wonder", PID: 7, SID: steamid.New("[U:1:34284979]"), Team: logparse.RED},
			Medigun:      logparse.Uber,
		})
}

func TestParseChargeEndedEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "wonder<7><[U:1:34284979]><Red>" triggered "chargeended" (duration "7.5")`,
		logparse.ChargeEndedEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "wonder", PID: 7, SID: steamid.New("[U:1:34284979]"), Team: logparse.RED},
			Duration:     7.5,
		})
}

func TestParseMedicDeathExEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "wonder<7><[U:1:34284979]><Red>" triggered "medic_death_ex" (uberpct "32")`,
		logparse.MedicDeathExEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "wonder", PID: 7, SID: steamid.New("[U:1:34284979]"), Team: logparse.RED},
			Uberpct:      32,
		})
}

func TestParseLostUberAdvantageEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "SEND HELP<16><[U:1:84528002]><Blue>" triggered "lost_uber_advantage" (time "44")`,
		logparse.LostUberAdvantageEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "SEND HELP", PID: 16, SID: steamid.New("[U:1:84528002]"), Team: logparse.BLU},
			Time:         44,
		})
}

func TestParseEmptyUberEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "Kwq<9><[U:1:96748980]><Blue>" triggered "empty_uber"`,
		logparse.EmptyUberEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "Kwq", PID: 9, SID: steamid.New("[U:1:96748980]"), Team: logparse.BLU},
		})
}

func TestParsePickupEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "g о а т z<13><[U:1:41435165]><Red>" picked up item "ammopack_small"`,
		logparse.PickupEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "g о а т z", PID: 13, SID: steamid.New("[U:1:41435165]"), Team: logparse.RED},
			Item:         logparse.ItemAmmoSmall,
			Healing:      0,
		})
	testLogLine(t, `L 02/21/2021 - 06:22:23: "g о а т z<13><[U:1:41435165]><Red>" picked up item "medkit_medium" (healing "47")`,
		logparse.PickupEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "g о а т z", PID: 13, SID: steamid.New("[U:1:41435165]"), Team: logparse.RED},
			Item:         logparse.ItemHPMedium,
			Healing:      47,
		})
}

func TestParseShotFiredEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "rad<6><[U:1:57823119]><Red>" triggered "shot_fired" (weapon "syringegun_medic")`,
		logparse.ShotFiredEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "rad", PID: 6, SID: steamid.New("[U:1:57823119]"), Team: logparse.RED},
			Weapon:       logparse.SyringeGun,
		})
}

func TestParseShotHitEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "z/<14><[U:1:66656848]><Blue>" triggered "shot_hit" (weapon "blackbox")`,
		logparse.ShotHitEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "z/", PID: 14, SID: steamid.New("[U:1:66656848]"), Team: logparse.BLU},
			Weapon:       logparse.BlackBox,
		})
}

func TestParseDamageEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "rad<6><[U:1:57823119]><Red>" triggered "damage" against "z/<14><[U:1:66656848]><Blue>" (damage "11") (weapon "syringegun_medic")`,
		logparse.DamageEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "rad", PID: 6, SID: steamid.New("[U:1:57823119]"), Team: logparse.RED},
			TargetPlayer: logparse.TargetPlayer{Name2: "z/", PID2: 14, SID2: steamid.New("[U:1:66656848]"), Team2: logparse.BLU},
			Weapon:       logparse.SyringeGun,
			Damage:       11,
		})
	testLogLine(t, `L 02/21/2021 - 06:22:23: "rad<6><[U:1:57823119]><Red>" triggered "damage" against "z/<14><[U:1:66656848]><Blue>" (damage "88") (realdamage "32") (weapon "ubersaw") (healing "110")`,
		logparse.DamageEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "rad", PID: 6, SID: steamid.New("[U:1:57823119]"), Team: logparse.RED},
			TargetPlayer: logparse.TargetPlayer{Name2: "z/", PID2: 14, SID2: steamid.New("[U:1:66656848]"), Team2: logparse.BLU},
			Damage:       88,
			Realdamage:   32,
			Weapon:       logparse.Ubersaw,
			Healing:      110,
		})
	testLogLine(t, `L 02/21/2021 - 06:22:23: "Lochlore<22><[U:1:127176886]><Blue>" triggered "damage" against "Doctrine<20><[U:1:1090182064]><Red>" (damage "762") (realdamage "127") (weapon "knife") (crit "crit")`,
		logparse.DamageEvt{
			TimeStamp:    logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer: logparse.SourcePlayer{Name: "Lochlore", PID: 22, SID: steamid.New("[U:1:127176886]"), Team: logparse.BLU},
			TargetPlayer: logparse.TargetPlayer{Name2: "Doctrine", PID2: 20, SID2: steamid.New("[U:1:1090182064]"), Team2: logparse.RED},
			Damage:       762,
			Realdamage:   127,
			Weapon:       logparse.Knife,
			Crit:         logparse.Crit,
		})
}

func TestParseJarateAttackEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "Banfield<2796><[U:1:958890744]><Blue>" triggered "jarate_attack" against "Legs™<2818><[U:1:42871337]><Red>" with "tf_weapon_jar" (attacker_position "1881 -1521 264") (victim_position "1729 -301 457")`,
		logparse.JarateAttackEvt{
			TimeStamp:        logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer:     logparse.SourcePlayer{Name: "Banfield", PID: 2796, SID: steamid.New("[U:1:958890744]"), Team: logparse.BLU},
			TargetPlayer:     logparse.TargetPlayer{Name2: "Legs™", PID2: 2818, SID2: steamid.New("[U:1:42871337]"), Team2: logparse.RED},
			Weapon:           logparse.JarBased,
			AttackerPosition: logparse.Pos{X: 1881, Y: -1521, Z: 264},
			VictimPosition:   logparse.Pos{X: 1729, Y: -301, Z: 457},
		})
}

func TestParseWMiniRoundWinEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: World triggered "Mini_Round_Win" (winner "Blue") (round "round_b")`,
		logparse.WMiniRoundWinEvt{
			TimeStamp: logparse.TimeStamp{
				CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC),
			},
		})
}

func TestParseWMiniRoundLenEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: World triggered "Mini_Round_Length" (seconds "340.62")`,
		logparse.WMiniRoundLenEvt{
			TimeStamp: logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			Seconds:   340.62,
		})
}

func TestParsWRoundSetupBeginEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: World triggered "Round_Setup_Begin"`,
		logparse.WRoundSetupBeginEvt{
			CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC),
		})
}

func TestParseWMiniRoundSelectedEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: World triggered "Mini_Round_Selected" (round "Round_A")`,
		logparse.WMiniRoundSelectedEvt{
			CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC),
		})
}

func TestParseWMiniRoundStartEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: World triggered "Mini_Round_Start"`,
		logparse.WMiniRoundStartEvt{
			CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC),
		})
}

func TestParseVoteSuccess(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: Vote succeeded "Kick Pain in a Box"`,
		logparse.VoteSuccessEvt{
			TimeStamp: logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			Name:      "Pain in a Box",
		})
}

func TestParseVoteFailed(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: Vote failed "Kick Flower" with code 3`,
		logparse.VoteFailEvt{
			TimeStamp: logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			Name:      "Flower",
			Code:      logparse.VoteCodeFailNoOutnumberYes,
		})
}

func TestParseVoteDetails(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: Kick Vote details:  VoteInitiatorSteamID: [U:1:0000001]  VoteTargetSteamID: [U:1:0000002]  Valid: 1  BIndividual: 1  Name: Disconnected  Proxy: 0"`,
		logparse.VoteKickDetailsEvt{
			TimeStamp: logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SID:       steamid.New("[U:1:0000001]"),
			SID2:      steamid.New("[U:1:0000002]"),
			Valid:     1,
			Name:      "Disconnected",
			Proxy:     0,
		})
}

func TestParseMilkAttackEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "✪lil vandal<2953><[U:1:178417727]><Blue>" triggered "milk_attack" against "Darth Jar Jar<2965><[U:1:209106507]><Red>" with "tf_weapon_jar" (attacker_position "-1040 -854 128") (victim_position "-1516 -382 128")`,
		logparse.MilkAttackEvt{
			TimeStamp:        logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer:     logparse.SourcePlayer{Name: "✪lil vandal", PID: 2953, SID: steamid.New("[U:1:178417727]"), Team: logparse.BLU},
			TargetPlayer:     logparse.TargetPlayer{Name2: "Darth Jar Jar", PID2: 2965, SID2: steamid.New("[U:1:209106507]"), Team2: logparse.RED}, //nolint:dupword
			Weapon:           logparse.JarBased,
			AttackerPosition: logparse.Pos{X: -1040, Y: -854, Z: 128},
			VictimPosition:   logparse.Pos{X: -1516, Y: -382, Z: 128},
		})
}

func TestParseGasAttackEvt(t *testing.T) {
	t.Parallel()

	testLogLine(t, `L 02/21/2021 - 06:22:23: "UnEpic<6760><[U:1:132169058]><Blue>" triggered "gas_attack" against "Johnny Blaze<6800><[U:1:33228413]><Red>" with "tf_weapon_jar" (attacker_position "-4539 2731 156") (victim_position "-4384 1527 128")`,
		logparse.GasAttackEvt{
			TimeStamp:        logparse.TimeStamp{CreatedOn: time.Date(2021, time.February, 21, 6, 22, 23, 0, time.UTC)},
			SourcePlayer:     logparse.SourcePlayer{Name: "UnEpic", PID: 6760, SID: steamid.New("[U:1:132169058]"), Team: logparse.BLU},
			TargetPlayer:     logparse.TargetPlayer{Name2: "Johnny Blaze", PID2: 6800, SID2: steamid.New("[U:1:33228413]"), Team2: logparse.RED},
			Weapon:           logparse.JarBased,
			AttackerPosition: logparse.Pos{X: -4539, Y: 2731, Z: 156},
			VictimPosition:   logparse.Pos{X: -4384, Y: 1527, Z: 128},
		})
}

func TestParseKVs(t *testing.T) {
	t.Parallel()

	var (
		parser  = logparse.NewLogParser()
		keyMapA = map[string]any{}
		keyMapB = map[string]any{}
	)

	require.True(t, parser.ParseKVs(`(damage "88") (realdamage "32") (weapon "ubersaw") (healing "110")`, keyMapA))
	require.Equal(t, map[string]any{"damage": "88", "realdamage": "32", "weapon": "ubersaw", "healing": "110"}, keyMapA)
	require.True(t, parser.ParseKVs(`L 01/16/2022 - 21:24:11: NewTeam "Red" triggered "pointcaptured" (cp "0") (cpname "#koth_viaduct_cap") (numcappers "2") (player1 "cube elegy<15><[U:1:84002473]><Red>") (position1 "-156 -105 1601") (player2 "bink<24><[U:1:164995715]><Red>") (position2 "57 78 1602")`, keyMapB))
	require.Equal(t, map[string]any{
		"cp":         "0",
		"cpname":     "#koth_viaduct_cap",
		"numcappers": "2",
		"player1":    "cube elegy<15><[U:1:84002473]><Red>",
		"position1":  "-156 -105 1601",
		"player2":    "bink<24><[U:1:164995715]><Red>",
		"position2":  "57 78 1602",
	}, keyMapB)
}
