//nolint:testpackage // white-box tests for the SynElo rating algorithm.
package stats

import (
	"math"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func TestSynEloTheta(t *testing.T) {
	t.Parallel()

	require.InDelta(t, 400, synEloTheta(1), 1e-9)
	require.InDelta(t, 2400, synEloTheta(12), 1e-9)
	require.InDelta(t, SynEloScale, synEloTheta(0), 1e-9)
}

func TestSynEloUpdateMatch(t *testing.T) {
	t.Parallel()

	calculator := SynEloCalculator{}
	red := states(Skill{Mu: 1000, Sigma: 0}, Skill{Mu: 1000, Sigma: 0})
	blu := states(Skill{Mu: 1000, Sigma: 0}, Skill{Mu: 1000, Sigma: 0})

	newRed, newBlu := calculator.UpdateMatch(red, blu, RatingWin)

	// With no history, convergence C is zero, so K = K_min and the change is
	// K_min * (1 - 0.5) = 1 point per player (2v2 shares the team adjustment).
	for _, update := range newRed {
		require.InDelta(t, SynEloRatingStart+SynEloKMin*0.5, update.Skill.Mu, 1e-9)
		require.InDelta(t, 0.5, update.Expected, 1e-9)
	}
	for _, update := range newBlu {
		require.InDelta(t, SynEloRatingStart-SynEloKMin*0.5, update.Skill.Mu, 1e-9)
	}
	require.Zero(t, newRed[0].Skill.Sigma)
	require.InDelta(t, SynEloRatingStart, red[0].Skill.Mu, 1e-9)
}

func TestSynEloConvergenceSignals(t *testing.T) {
	t.Parallel()

	// A steadily climbing, well-predicted player has velocity and volatility.
	history := []RatingObservation{
		{Rating: 1000, Expected: 0.5, Outcome: RatingWin},
		{Rating: 1010, Expected: 0.55, Outcome: RatingWin},
		{Rating: 1020, Expected: 0.6, Outcome: RatingWin},
		{Rating: 1030, Expected: 0.65, Outcome: RatingWin},
	}

	convergence := synEloConvergence(history)
	require.Greater(t, convergence, 0.0)
	require.LessOrEqual(t, convergence, 1.0)
	require.Zero(t, synEloConvergence(nil))
}

func TestSynEloFloorAndRounding(t *testing.T) {
	t.Parallel()

	calculator := SynEloCalculator{}
	low := states(Skill{Mu: SynEloRatingFloor, Sigma: 0})
	high := states(Skill{Mu: 5000, Sigma: 0})

	newRed, _ := calculator.UpdateMatch(low, high, RatingLoss)
	require.GreaterOrEqual(t, newRed[0].Skill.Mu, SynEloRatingFloor)
	require.InDelta(t, math.RoundToEven(newRed[0].Skill.Mu), newRed[0].Skill.Mu, 1e-9)
}

func TestSynEloDisplay(t *testing.T) {
	t.Parallel()

	calculator := SynEloCalculator{}
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)

	shadow := calculator.Display(RatingContext{
		Skill:       Skill{Mu: 3000, Sigma: 0},
		Rounds:      SynEloShadowRounds - 1,
		LastMatchAt: now,
		Now:         now,
	})
	require.False(t, shadow.Visible)
	require.Equal(t, int(SynEloShadowGrade), shadow.Rating)
	require.False(t, shadow.Stale)

	graduated := calculator.Display(RatingContext{
		Skill:       Skill{Mu: 3000, Sigma: 0},
		Rounds:      SynEloShadowRounds,
		LastMatchAt: now,
		Now:         now,
	})
	require.True(t, graduated.Visible)
	require.Equal(t, 3000, graduated.Rating)

	stale := calculator.Display(RatingContext{
		Skill:       Skill{Mu: 3000, Sigma: 0},
		Rounds:      SynEloShadowRounds,
		LastMatchAt: now.Add(-31 * 24 * time.Hour),
		Now:         now,
	})
	require.True(t, stale.Stale)
}

func TestSynEloWinProbability(t *testing.T) {
	t.Parallel()

	calculator := SynEloCalculator{}
	even := []Skill{{Mu: 1000, Sigma: 0}, {Mu: 1000, Sigma: 0}}
	require.InDelta(t, 0.5, calculator.WinProbability(even, even), 1e-9)
	require.InDelta(t, 0.5, calculator.WinProbability(nil, even), 1e-9)
	require.Greater(t, calculator.WinProbability([]Skill{{Mu: 1300, Sigma: 0}, {Mu: 1300, Sigma: 0}}, even), 0.5)
}

func TestReplayRatingsWithSynElo(t *testing.T) {
	t.Parallel()

	playerA := steamid.New("[U:1:7200000201]")
	playerB := steamid.New("[U:1:7200000202]")
	rows := make([]ratedOutcomeRow, 0, 52*2)

	for idx := range 52 {
		matchID, errUUID := uuid.NewV4()
		require.NoError(t, errUUID)
		start := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC).Add(time.Duration(idx) * time.Hour)
		rows = append(rows,
			ratingRow(matchID, start, 1, 0, playerA, 1, "red", 1, idx*2+1),
			ratingRow(matchID, start, 1, 0, playerB, 1, "blu", 1, idx*2+2),
		)
	}

	ratings, history := replayRatings(rows, SynEloCalculator{})
	byPlayer := ratingsByPlayer(ratings)

	require.Equal(t, 52, byPlayer[playerA].RoundsPlayed)
	require.True(t, byPlayer[playerA].Display.Visible)
	require.Greater(t, byPlayer[playerA].Mu, SynEloRatingStart)
	require.Less(t, byPlayer[playerB].Mu, SynEloRatingStart)
	require.Len(t, historyByPlayer(history)[playerA], 52)
	require.InDelta(t, 0.5, historyByPlayer(history)[playerA][0].Expected, 1e-9)
}
