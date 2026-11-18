//nolint:testpackage // white-box tests for the pluggable rating algorithms.
package stats

import (
	"math"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func TestNewRatingCalculator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		configured string
		wantName   string
		wantErr    bool
	}{
		{name: "empty selects default", configured: "", wantName: RatingAlgorithmTrueSkill},
		{name: "trueskill", configured: RatingAlgorithmTrueSkill, wantName: RatingAlgorithmTrueSkill},
		{name: "case and whitespace insensitive", configured: "  ELO ", wantName: RatingAlgorithmElo},
		{name: "elo", configured: RatingAlgorithmElo, wantName: RatingAlgorithmElo},
		{name: "synelo", configured: RatingAlgorithmSynElo, wantName: RatingAlgorithmSynElo},
		{name: "unknown", configured: "glicko", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			calculator, errCalc := NewRatingCalculator(test.configured)
			if test.wantErr {
				require.ErrorIs(t, errCalc, ErrUnknownRatingAlgorithm)
				require.Nil(t, calculator)

				return
			}

			require.NoError(t, errCalc)
			require.Equal(t, test.wantName, calculator.Name())
		})
	}
}

func states(skills ...Skill) []PlayerRatingState {
	result := make([]PlayerRatingState, len(skills))
	for idx, skill := range skills {
		result[idx] = PlayerRatingState{Skill: skill}
	}

	return result
}

func TestEloCalculatorUpdateMatch(t *testing.T) {
	t.Parallel()

	calculator := EloCalculator{}
	red := states(Skill{Mu: 1000, Sigma: 0}, Skill{Mu: 1000, Sigma: 0})
	blu := states(Skill{Mu: 1000, Sigma: 0}, Skill{Mu: 1000, Sigma: 0})

	newRed, newBlu := calculator.UpdateMatch(red, blu, RatingWin)

	// Equal teams: expected 0.5, so each winner gains K/4 and each loser drops K/4.
	for _, update := range newRed {
		require.InDelta(t, 1000+EloKFactor/4, update.Skill.Mu, 1e-9)
		require.InDelta(t, 0.5, update.Expected, 1e-9)
	}
	for _, update := range newBlu {
		require.InDelta(t, 1000-EloKFactor/4, update.Skill.Mu, 1e-9)
	}

	require.InDelta(t, 1000.0, red[0].Skill.Mu, 1e-9)
	require.InDelta(t, 1000.0, blu[0].Skill.Mu, 1e-9)

	emptyRed, emptyBlu := calculator.UpdateMatch(nil, blu, RatingWin)
	require.Empty(t, emptyRed)
	require.Len(t, emptyBlu, len(blu))

	drawnRed, drawnBlu := calculator.UpdateMatch(red, blu, RatingDraw)
	require.InDelta(t, 1000.0, drawnRed[0].Skill.Mu, 1e-9)
	require.InDelta(t, 1000.0, drawnBlu[0].Skill.Mu, 1e-9)
}

func TestEloCalculatorFavouriteGainsLess(t *testing.T) {
	t.Parallel()

	calculator := EloCalculator{}
	stronger := states(Skill{Mu: 1200, Sigma: 0}, Skill{Mu: 1200, Sigma: 0})
	weaker := states(Skill{Mu: 1000, Sigma: 0}, Skill{Mu: 1000, Sigma: 0})

	newRed, _ := calculator.UpdateMatch(stronger, weaker, RatingWin)
	expected := 1 / (1 + math.Pow(10, -200/eloScale))
	wantDelta := EloKFactor * (1 - expected) / 2
	require.InDelta(t, 1200+wantDelta, newRed[0].Skill.Mu, 1e-9)
	require.Less(t, wantDelta, EloKFactor/2)
}

func TestEloCalculatorDisplayAndWinProbability(t *testing.T) {
	t.Parallel()

	calculator := EloCalculator{}
	display := calculator.Display(RatingContext{Skill: Skill{Mu: 1234.6, Sigma: 0.5}})
	require.Equal(t, 1235, display.Rating)
	require.Zero(t, display.Sigma)
	require.Equal(t, 1235, display.Conservative)
	require.True(t, display.Visible)

	even := []Skill{{Mu: EloInitialRating, Sigma: 0}}
	require.InDelta(t, 0.5, calculator.WinProbability(even, even), 1e-9)
	require.InDelta(t, 0.5, calculator.WinProbability(nil, even), 1e-9)
	require.Greater(t, calculator.WinProbability([]Skill{{Mu: 1400, Sigma: 0}}, even), 0.5)
	require.Less(t, calculator.WinProbability(even, []Skill{{Mu: 1400, Sigma: 0}}), 0.5)
}

func TestReplayRatingsWithElo(t *testing.T) {
	t.Parallel()

	playerA := steamid.New("[U:1:7200000101]")
	playerB := steamid.New("[U:1:7200000102]")
	matchID, errUUID := uuid.NewV4()
	require.NoError(t, errUUID)

	rows := []ratedOutcomeRow{
		ratingRow(matchID, time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC), 2, 1, playerA, 1, "red", 2, 2),
		ratingRow(matchID, time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC), 2, 1, playerB, 1, "blu", 2, 1),
	}

	ratings, _ := replayRatings(rows, EloCalculator{})
	byPlayer := ratingsByPlayer(ratings)

	// One player per team in this replay, so the whole team adjustment goes to one player.
	require.InDelta(t, EloInitialRating+EloKFactor/2, byPlayer[playerA].Mu, 1e-9)
	require.InDelta(t, EloInitialRating-EloKFactor/2, byPlayer[playerB].Mu, 1e-9)
	require.Equal(t, int64(EloInitialRating+EloKFactor/2), int64(byPlayer[playerA].Display.Rating))
	require.Equal(t, int64(EloInitialRating-EloKFactor/2), int64(byPlayer[playerB].Display.Conservative))
}
