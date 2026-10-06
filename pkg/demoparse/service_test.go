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
