package sourcemod_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/leighmacdonald/gbans/internal/sourcemod/v1"
	"github.com/leighmacdonald/gbans/internal/stats"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func TestSMBalanceTeams(t *testing.T) {
	harness := newRPCHarness(t)
	client := harness.pluginClient(t, harness.serverToken(t))
	ctx := context.Background()

	repo := stats.NewRepository(fixture.Database)
	bucketID, errBucket := repo.DefaultBucketID(ctx)
	require.NoError(t, errBucket)

	players := []steamid.SteamID{
		steamid.New(76561198000000101),
		steamid.New(76561198000000102),
		steamid.New(76561198000000103),
		steamid.New(76561198000000104),
	}
	means := []float64{1500, 1400, 1300, 1200}
	ratings := make([]stats.PlayerRating, 0, len(players))
	for idx, player := range players {
		ratings = append(ratings, stats.PlayerRating{
			SteamID:       player,
			StatsBucketID: bucketID,
			Mu:            means[idx],
			Sigma:         50,
			Display:       stats.DisplayRating{Rating: int(means[idx]), Visible: true},
			MatchesPlayed: 10,
			CalculatedOn:  time.Now(),
		})
	}
	require.NoError(t, repo.ReplaceRatings(ctx, ratings, nil))

	request := &v1.SMBalanceTeamsRequest{
		SteamIds: []uint64{
			uint64(players[0].Int64()), //nolint:gosec // valid SteamID64 is positive.
			uint64(players[1].Int64()), //nolint:gosec // valid SteamID64 is positive.
			uint64(players[2].Int64()), //nolint:gosec // valid SteamID64 is positive.
			uint64(players[3].Int64()), //nolint:gosec // valid SteamID64 is positive.
			uint64(76561198000000999),  // Unrated player uses the starting skill.
		},
		MaxExact: new(int32(32)),
	}

	resp, err := client.SMBalanceTeams(ctx, request)
	require.NoError(t, err)
	require.Len(t, resp.GetRedSteamIds(), 3)
	require.Len(t, resp.GetBluSteamIds(), 2)
	require.Empty(t, resp.GetWaitingSteamIds())
	require.Equal(t, stats.BalanceAlgorithmAlternating, resp.GetBalancer())
	require.Equal(t, stats.RatingAlgorithmTrueSkill, resp.GetAlgorithm())

	all := append(append([]uint64{}, resp.GetRedSteamIds()...), resp.GetBluSteamIds()...)
	require.ElementsMatch(t, request.GetSteamIds(), all)
}

func TestSMBalanceTeamsRejectsEmptyAndInvalid(t *testing.T) {
	harness := newRPCHarness(t)
	client := harness.pluginClient(t, harness.serverToken(t))
	ctx := context.Background()

	_, errEmpty := client.SMBalanceTeams(ctx, &v1.SMBalanceTeamsRequest{})
	requireCode(t, connect.CodeInvalidArgument, errEmpty)

	_, errInvalid := client.SMBalanceTeams(ctx, &v1.SMBalanceTeamsRequest{
		SteamIds: []uint64{0},
	})
	requireCode(t, connect.CodeInvalidArgument, errInvalid)
}
