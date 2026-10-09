package demoparse_test

import (
	"context"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	demostatsv1 "github.com/leighmacdonald/gbans/internal/demostats/v1"
	"github.com/leighmacdonald/gbans/internal/demostats/v1/demostatsv1connect"
	"github.com/leighmacdonald/gbans/internal/fs"
	"github.com/leighmacdonald/gbans/internal/json"
	"github.com/leighmacdonald/gbans/pkg/demoparse"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	demoPath := fs.FindFile("testdata/koth_ashville_final.dem.json", "gbans")
	file, err := os.Open(demoPath)
	require.NoError(t, err)
	defer file.Close()
	parsed, err := json.Decode[demoparse.Demo](file)
	require.NoError(t, err)
	require.Equal(t, "tf", parsed.Game)
}

type fixtureParser struct {
	demo *demoparse.Demo
}

func (f fixtureParser) ParseDemo(_ context.Context, req *demostatsv1.ParseDemoRequest) (*demostatsv1.ParseDemoResponse, error) {
	resp := demoparse.ProtoFromDemo(f.demo)
	if req.GetFilename() != "" {
		resp.GetDemo().Filename = req.GetFilename()
	}

	return resp, nil
}

func loadFixtureDemo(t *testing.T) *demoparse.Demo {
	t.Helper()

	demoPath := fs.FindFile("testdata/koth_ashville_final.dem.json", "gbans")
	file, err := os.Open(demoPath)
	require.NoError(t, err)
	defer file.Close()

	parsed, err := json.Decode[demoparse.Demo](file)
	require.NoError(t, err)

	return &parsed
}

// TestSubmitConnect ensures Submit speaks the v0.3.x ConnectRPC API and maps
// the typed response back into the domain model.
func TestSubmitConnect(t *testing.T) {
	want := loadFixtureDemo(t)

	_, handler := demostatsv1connect.NewDemoServiceHandler(fixtureParser{demo: want})
	server := httptest.NewServer(handler)
	defer server.Close()

	got, err := demoparse.Submit(t.Context(), server.URL, "match.dem", strings.NewReader("fake-dem-bytes"))
	require.NoError(t, err)
	require.Equal(t, "match.dem", got.Filename)
	require.Equal(t, want.Server, got.Server)
	require.Equal(t, want.Map, got.Map)
	require.Equal(t, want.Game, got.Game)
	require.Len(t, got.Rounds, len(want.Rounds))
	require.NotEmpty(t, got.SteamIDs())
	require.Equal(t, want.Scores(), got.Scores())
}

// TestProtoRoundTrip ensures domain -> proto -> domain conversions preserve
// the fields consumed by stats import.
func TestProtoRoundTrip(t *testing.T) {
	want := loadFixtureDemo(t)

	got := demoparse.DemoFromProto(demoparse.ProtoFromDemo(want), want.Filename)
	require.Equal(t, want.Filename, got.Filename)
	require.Equal(t, want.Server, got.Server)
	require.Equal(t, want.Map, got.Map)
	require.Equal(t, want.Game, got.Game)
	require.Equal(t, want.DemoType, got.DemoType)
	require.Len(t, got.Rounds, len(want.Rounds))
	require.Equal(t, want.Scores(), got.Scores())

	for roundIdx, round := range want.Rounds {
		require.Equal(t, round.Winner, got.Rounds[roundIdx].Winner)
		require.Len(t, got.Rounds[roundIdx].Players, len(round.Players))

		for playerIdx, player := range round.Players {
			require.Equal(t, player.SteamID, got.Rounds[roundIdx].Players[playerIdx].SteamID)
			require.Equal(t, player.Kills, got.Rounds[roundIdx].Players[playerIdx].Kills)
			require.Equal(t, player.Damage, got.Rounds[roundIdx].Players[playerIdx].Damage)
			require.Equal(t, player.Healing, got.Rounds[roundIdx].Players[playerIdx].Healing)
			require.Equal(t, player.Classes, got.Rounds[roundIdx].Players[playerIdx].Classes)
			require.Equal(t, player.Weapons, got.Rounds[roundIdx].Players[playerIdx].Weapons)
		}
	}
}

// TestEventProtoRoundTrip ensures every match event variant survives the
// domain -> proto -> domain conversion used by fixture test servers.
func TestEventProtoRoundTrip(t *testing.T) {
	want := &demoparse.Demo{
		Filename: "match.dem",
		Events: []demoparse.MatchEvent{
			{
				Tick: 100, Type: demoparse.MatchEventKill,
				Kill: &demoparse.KillEvent{
					Tick: 100, Killer: "[U:1:1]", Victim: "[U:1:2]", Weapon: "scattergun",
					KillerPos:    &demoparse.Position{X: 1, Y: 2, Z: 3},
					VictimPos:    &demoparse.Position{X: 4, Y: 5, Z: 6},
					KillerAngles: &demoparse.EyeAngles{Pitch: 1, Yaw: 2},
					VictimAngles: &demoparse.EyeAngles{Pitch: 3, Yaw: 4},
					IsFirstBlood: true, IsDomination: true, IsRevenge: true,
				},
			},
			{
				Tick: 200, Type: demoparse.MatchEventCaptureStarted,
				CaptureStarted: &demoparse.CaptureStartedEvent{
					Cp: 1, CpName: "cp_1", Team: 2, CapTeam: 3,
					Cappers: []string{"[U:1:1]"}, CapTime: 2.5,
				},
			},
			{
				Tick: 300, Type: demoparse.MatchEventBuildingBuilt,
				BuildingBuilt: &demoparse.BuildingBuiltEvent{
					Owner: "[U:1:3]", Building: "sentry", Level: 3,
					Pos: demoparse.Position{X: 7, Y: 8, Z: 9},
				},
			},
			{
				Tick: 400, Type: demoparse.MatchEventBuildingDestroyed,
				BuildingDestroyed: &demoparse.BuildingDestroyedEvent{
					Owner: "[U:1:3]", Attacker: "[U:1:4]", Assister: "[U:1:5]",
					Weapon: "tf_projectile_rocket", Building: "dispenser",
					Pos: &demoparse.Position{X: 1, Y: 1, Z: 1},
				},
			},
			{
				Tick: 500, Type: demoparse.MatchEventBuildingUpgraded,
				BuildingUpgraded: &demoparse.BuildingLifecycleEvent{
					Player: "[U:1:3]", Building: "teleporter", Index: 9,
				},
			},
			{
				Tick: 600, Type: demoparse.MatchEventRoundWon,
				RoundWon: &demoparse.RoundWonEvent{
					Winner: "red", WinReason: 1, RoundTime: 100.5,
				},
			},
			{Tick: 700, Type: demoparse.MatchEventSetupFinished},
			{
				Tick: 800, Type: demoparse.MatchEventUberDropped,
				UberDropped: &demoparse.UberDroppedEvent{
					Medic: "[U:1:6]", Attacker: "[U:1:7]", Healing: 450,
				},
			},
			{
				Tick: 900, Type: demoparse.MatchEventFlagCaptured,
				FlagCaptured: &demoparse.FlagCapturedEvent{CappingTeam: 3, Score: 2},
			},
			{
				Tick: 1000, Type: demoparse.MatchEventKillstreakEnded,
				KillstreakEnded: &demoparse.KillstreakEndedEvent{
					Player: "[U:1:1]", Streak: 8, Killer: "[U:1:2]",
				},
			},
		},
	}

	got := demoparse.DemoFromProto(demoparse.ProtoFromDemo(want), want.Filename)
	require.Len(t, got.Events, len(want.Events))

	for i, event := range want.Events {
		require.Equal(t, event.Tick, got.Events[i].Tick, "event %d tick", i)
		require.Equal(t, event.Type, got.Events[i].Type, "event %d type", i)
	}

	require.Equal(t, *want.Events[0].Kill, *got.Events[0].Kill)
	require.Equal(t, *want.Events[1].CaptureStarted, *got.Events[1].CaptureStarted)
	require.Equal(t, *want.Events[2].BuildingBuilt, *got.Events[2].BuildingBuilt)
	require.Equal(t, *want.Events[3].BuildingDestroyed, *got.Events[3].BuildingDestroyed)
	require.Equal(t, *want.Events[4].BuildingUpgraded, *got.Events[4].BuildingUpgraded)
	require.Equal(t, *want.Events[5].RoundWon, *got.Events[5].RoundWon)
	require.Equal(t, *want.Events[7].UberDropped, *got.Events[7].UberDropped)
	require.Equal(t, *want.Events[8].FlagCaptured, *got.Events[8].FlagCaptured)
	require.Equal(t, *want.Events[9].KillstreakEnded, *got.Events[9].KillstreakEnded)
}

// TestWinnerMapping ensures the proto Team enum maps onto the winner strings
// consumed by stats import.
func TestWinnerMapping(t *testing.T) {
	resp := demoparse.ProtoFromDemo(&demoparse.Demo{
		Filename: "match.dem",
		DemoType: demoparse.HL2Demo,
		Server:   "server",
		Map:      "cp_process_final",
		Game:     "tf",
		Rounds: []demoparse.RoundSummary{
			{Winner: "red", Time: 100},
			{Winner: "blue", Time: 200},
			{Time: 50},
		},
	})

	got := demoparse.DemoFromProto(resp, "match.dem")
	require.Equal(t, "red", got.Rounds[0].Winner)
	require.Equal(t, "blue", got.Rounds[1].Winner)
	require.Empty(t, got.Rounds[2].Winner)
}
