package tests_test

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func srcdsTokens(server domain.Server) *authTokens {
	return &authTokens{
		user:           nil,
		serverPassword: server.Password,
	}
}

func genSpeedrun(players int, bots int) domain.Speedrun {
	run := domain.Speedrun{
		MapDetail:     domain.MapDetail{MapName: "pl_" + stringutil.SecureRandomString(10)},
		PointCaptures: nil,
		ServerID:      testServer.ServerID,
		Players:       make([]domain.SpeedrunParticipant, players),
		Duration:      time.Second * time.Duration(rand.Int32N(10000)), // nolint: gosec
		PlayerCount:   players,
		BotCount:      bots,
		CreatedOn:     time.Now(),
		Category:      domain.Mode24v40,
	}

	for player := range players {
		run.Players[player] = domain.SpeedrunParticipant{
			SteamID:  steamid.RandSID64(),
			Duration: time.Second * time.Duration(rand.Int32N(5000)), // nolint: gosec
		}
	}

	for round := range rand.Int32N(5) + 1 { // nolint: gosec
		capture := domain.SpeedrunPointCaptures{
			RoundID:  int(round) + 1,
			Players:  nil,
			Duration: time.Second * time.Duration(rand.Int32N(1000)), // nolint: gosec
		}

		for j := range rand.Int32N(5) + 1 { // nolint: gosec
			capture.Players = append(capture.Players, domain.SpeedrunParticipant{
				SteamID:  run.Players[j].SteamID,
				Duration: time.Second * time.Duration(rand.Int32N(1000)), // nolint: gosec
			})
		}

		run.PointCaptures = append(run.PointCaptures, capture)
	}

	return run
}

func TestSubmitSpeedrun(t *testing.T) {
	router := testRouter()
	speedrun := genSpeedrun(24, 40)
	var result domain.Speedrun
	testEndpointWithReceiver(t, router, http.MethodPost, "/api/sm/speedruns", speedrun, http.StatusOK, srcdsTokens(testServer), &result)
	require.Equal(t, strings.ToLower(speedrun.MapDetail.MapName), result.MapDetail.MapName)

	var result2 domain.Speedrun
	testEndpointWithReceiver(t, router, http.MethodGet, fmt.Sprintf("/api/speedruns/byid/%d", result.SpeedrunID), speedrun, http.StatusOK, srcdsTokens(testServer), &result2)

	require.Len(t, result2.Players, len(result.Players))
	require.Len(t, result2.PointCaptures, len(result.PointCaptures))
	for i := range result.PointCaptures {
		require.Len(t, result2.PointCaptures[i].Players, len(result.PointCaptures[i].Players))
	}
	require.Equal(t, result.ServerID, result2.ServerID)
	require.Equal(t, result.SpeedrunID, result2.SpeedrunID)

	for range 40 {
		var result3 domain.Speedrun
		sr := genSpeedrun(24, 40)
		sr.MapDetail.MapName = speedrun.MapDetail.MapName

		testEndpointWithReceiver(t, router, http.MethodPost, "/api/sm/speedruns", sr, http.StatusOK, srcdsTokens(testServer), &result3)
		require.Equal(t, strings.ToLower(speedrun.MapDetail.MapName), result.MapDetail.MapName)
	}

	top := map[string][]domain.Speedrun{}
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/speedruns/overall/top?count=10", nil, http.StatusOK,
		nil, &top)
	require.Len(t, top[result.MapDetail.MapName], 10)
}
