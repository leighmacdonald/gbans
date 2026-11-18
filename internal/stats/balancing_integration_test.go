package stats_test

import (
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/maps"
	"github.com/leighmacdonald/gbans/internal/stats"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func seedBalanceRatings(t *testing.T, fixture *tests.Fixture, bucketID int32) []steamid.SteamID {
	t.Helper()

	players := []steamid.SteamID{
		steamid.New(76561198000000001),
		steamid.New(76561198000000002),
		steamid.New(76561198000000003),
		steamid.New(76561198000000004),
		steamid.New(76561198000000005),
	}
	mus := []float64{1000, 900, 800, 700}

	ratings := make([]stats.PlayerRating, 0, len(mus))
	for idx, mean := range mus {
		ratings = append(ratings, stats.PlayerRating{
			SteamID:       players[idx],
			StatsBucketID: bucketID,
			Mu:            mean,
			Sigma:         50,
			Display:       stats.DisplayRating{Rating: int(mean), Visible: true},
			MatchesPlayed: 10,
			CalculatedOn:  time.Now(),
		})
	}

	require.NoError(t, stats.NewRepository(fixture.Database).ReplaceRatings(t.Context(), ratings, nil))

	return players
}

func TestBalanceTeamsIntegration(t *testing.T) {
	t.Parallel()

	fixture := tests.NewFixture()
	defer fixture.Close()

	ctx := t.Context()
	repo := stats.NewRepository(fixture.Database)
	bucketID, errBucket := repo.DefaultBucketID(ctx)
	require.NoError(t, errBucket)

	players := seedBalanceRatings(t, fixture, bucketID)

	svc := stats.New(repo, maps.Maps{})
	result, errBalance := svc.BalanceTeams(ctx, players, 0)
	require.NoError(t, errBalance)

	require.Equal(t, stats.BalanceAlgorithmAlternating, result.Balancer)
	require.Equal(t, stats.RatingAlgorithmTrueSkill, result.Algorithm)
	require.Len(t, result.Red, 3)
	require.Len(t, result.Blu, 2)
	require.Empty(t, result.Waiting)
	require.ElementsMatch(t, players, append(append([]steamid.SteamID{}, result.Red...), result.Blu...))

	// The unrated player (index 4) uses TrueSkill's starting mu and is still
	// assigned to a team rather than left out.
	require.Contains(t, append(result.Red, result.Blu...), players[4])
}

func TestBalanceTeamsWithGosper(t *testing.T) {
	t.Parallel()

	fixture := tests.NewFixture()
	defer fixture.Close()

	ctx := t.Context()
	repo := stats.NewRepository(fixture.Database)
	bucketID, errBucket := repo.DefaultBucketID(ctx)
	require.NoError(t, errBucket)

	players := seedBalanceRatings(t, fixture, bucketID)
	svc := stats.New(repo, maps.Maps{}, stats.WithTeamBalancer(stats.GosperBalancer{}))

	result, errBalance := svc.BalanceTeams(ctx, players, 0)
	require.NoError(t, errBalance)
	require.Equal(t, stats.BalanceAlgorithmGosper, result.Balancer)
	require.ElementsMatch(t, players, append(append([]steamid.SteamID{}, result.Red...), result.Blu...))
	require.Empty(t, result.Waiting)
}
