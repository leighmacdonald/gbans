package servers_test

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func TestSpeedrunsHTTP(t *testing.T) {
	var (
		userAuth   = &tests.UserAuth{Profile: fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.Moderator)}
		serverAuth = &tests.ServerAuth{}
		router     = fixture.CreateRouter()
		server     = fixture.CreateTestServer(t.Context())
		speedruns  = servers.NewSpeedruns(servers.NewSpeedrunRepository(fixture.Database, fixture.Persons))
	)

	servers.NewSpeedrunsHandler(router, userAuth, serverAuth, speedruns)

	testSR := genSpeedrun(24, 0, server.ServerID)
	createdSR := tests.PostGOK[servers.Speedrun](t, router, "/api/sm/speedruns", testSR)

	fetched := tests.GetGOK[servers.Speedrun](t, router, fmt.Sprintf("/api/speedruns/byid/%d", createdSR.SpeedrunID))
	require.Equal(t, 24, fetched.PlayerCount)

	for i := range createdSR.PointCaptures {
		require.Len(t, fetched.PointCaptures[i].Players, len(createdSR.PointCaptures[i].Players))
	}
	require.Equal(t, createdSR.ServerID, fetched.ServerID)
	require.Equal(t, createdSR.SpeedrunID, fetched.SpeedrunID)

	mapName := "pl_" + stringutil.SecureRandomString(8)
	var srB servers.Speedrun
	for range 40 {
		srB = genSpeedrun(24, 40, server.ServerID)
		srB.MapDetail.MapName = mapName

		res := tests.PostGOK[servers.Speedrun](t, router, "/api/sm/speedruns", srB)
		require.Equal(t, strings.ToLower(srB.MapDetail.MapName), res.MapDetail.MapName)
	}

	// result := tests.GetGOK[map[string][]servers.Speedrun](t, router, "/api/speedruns/overall/top?count=10")
	// require.Len(t, result[mapName], 10)
}

func genSpeedrun(players int, bots int, serverID int) servers.Speedrun {
	run := servers.Speedrun{
		MapDetail:     servers.MapDetail{MapName: "pl_" + stringutil.SecureRandomString(10)},
		PointCaptures: nil,
		ServerID:      serverID,
		Players:       make([]servers.SpeedrunParticipant, players),
		Duration:      time.Second * time.Duration(rand.Int32N(10000)), // nolint: gosec
		PlayerCount:   players,
		BotCount:      bots,
		CreatedOn:     time.Now(),
		Category:      servers.Mode24v40,
	}

	for player := range players {
		run.Players[player] = servers.SpeedrunParticipant{
			SteamID:  steamid.RandSID64(),
			Duration: time.Second * time.Duration(rand.Int32N(5000)), // nolint: gosec
		}
	}

	for round := range rand.Int32N(5) + 1 { // nolint: gosec
		capture := servers.SpeedrunPointCaptures{
			RoundID:  int(round) + 1,
			Players:  nil,
			Duration: time.Second * time.Duration(rand.Int32N(1000)), // nolint: gosec
		}

		for j := range rand.Int32N(5) + 1 { // nolint: gosec
			capture.Players = append(capture.Players, servers.SpeedrunParticipant{
				SteamID:  run.Players[j].SteamID,
				Duration: time.Second * time.Duration(rand.Int32N(1000)), // nolint: gosec
			})
		}

		run.PointCaptures = append(run.PointCaptures, capture)
	}

	return run
}
