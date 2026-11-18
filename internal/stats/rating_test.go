//nolint:testpackage // white-box tests for the rating algorithm.
package stats

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// goldenTolerance accounts for the reference implementation (Python `trueskill`)
// converging its expectation-propagation schedule to a small residual instead
// of evaluating the two-team update in closed form.
const goldenTolerance = 1e-4

func skillsOf(pairs ...Skill) []Skill {
	return pairs
}

func requireSkillsClose(t *testing.T, want, got []Skill) {
	t.Helper()
	require.Len(t, got, len(want), "skill count")
	for idx := range want {
		require.InDeltaf(t, want[idx].Mu, got[idx].Mu, goldenTolerance, "mu[%d]", idx)
		require.InDeltaf(t, want[idx].Sigma, got[idx].Sigma, goldenTolerance, "sigma[%d]", idx)
	}
}

// TestUpdateTeamRatingsGolden compares the closed-form update against vectors
// generated with the Python `trueskill` reference implementation.
func TestUpdateTeamRatingsGolden(t *testing.T) {
	t.Parallel()

	initial := func() Skill { return Skill{Mu: RatingMu, Sigma: RatingSigma} }

	testCases := []struct {
		name    string
		red     []Skill
		blu     []Skill
		outcome RatingOutcome
		wantRed []Skill
		wantBlu []Skill
	}{
		{
			name:    "1v1 even win",
			red:     skillsOf(initial()),
			blu:     skillsOf(initial()),
			outcome: RatingWin,
			wantRed: skillsOf(Skill{Mu: 29.3958320200, Sigma: 7.1714755873}),
			wantBlu: skillsOf(Skill{Mu: 20.6041679800, Sigma: 7.1714755873}),
		},
		{
			name:    "1v1 even draw",
			red:     skillsOf(initial()),
			blu:     skillsOf(initial()),
			outcome: RatingDraw,
			wantRed: skillsOf(Skill{Mu: 25.0000000000, Sigma: 6.4575196623}),
			wantBlu: skillsOf(Skill{Mu: 25.0000000000, Sigma: 6.4575196623}),
		},
		{
			name:    "1v1 upset",
			red:     skillsOf(initial()),
			blu:     skillsOf(Skill{Mu: 30, Sigma: 5}),
			outcome: RatingWin,
			wantRed: skillsOf(Skill{Mu: 31.9953065775, Sigma: 6.4889392682}),
			wantBlu: skillsOf(Skill{Mu: 27.4812419772, Sigma: 4.6326844413}),
		},
		{
			name:    "1v1 uneven draw",
			red:     skillsOf(Skill{Mu: 30, Sigma: 5}),
			blu:     skillsOf(initial()),
			outcome: RatingDraw,
			wantRed: skillsOf(Skill{Mu: 29.0334622586, Sigma: 4.4914266064}),
			wantBlu: skillsOf(Skill{Mu: 27.6843498895, Sigma: 5.6714213803}),
		},
		{
			name:    "2v2 even win",
			red:     skillsOf(initial(), initial()),
			blu:     skillsOf(initial(), initial()),
			outcome: RatingWin,
			wantRed: skillsOf(
				Skill{Mu: 28.1083226303, Sigma: 7.7743633498},
				Skill{Mu: 28.1083226303, Sigma: 7.7743633498},
			),
			wantBlu: skillsOf(
				Skill{Mu: 21.8916773697, Sigma: 7.7743633498},
				Skill{Mu: 21.8916773697, Sigma: 7.7743633498},
			),
		},
		{
			name:    "2v2 red loss",
			red:     skillsOf(initial(), initial()),
			blu:     skillsOf(Skill{Mu: 28, Sigma: 5}, Skill{Mu: 28, Sigma: 5}),
			outcome: RatingLoss,
			wantRed: skillsOf(
				Skill{Mu: 22.3529526309, Sigma: 7.6759537855},
				Skill{Mu: 22.3529526309, Sigma: 7.6759537855},
			),
			wantBlu: skillsOf(
				Skill{Mu: 28.9531064470, Sigma: 4.8622654922},
				Skill{Mu: 28.9531064470, Sigma: 4.8622654922},
			),
		},
		{
			name:    "3v3 even win",
			red:     skillsOf(initial(), initial(), initial()),
			blu:     skillsOf(initial(), initial(), initial()),
			outcome: RatingWin,
			wantRed: skillsOf(
				Skill{Mu: 27.5379348001, Sigma: 7.9651917716},
				Skill{Mu: 27.5379348001, Sigma: 7.9651917716},
				Skill{Mu: 27.5379348001, Sigma: 7.9651917716},
			),
			wantBlu: skillsOf(
				Skill{Mu: 22.4620651999, Sigma: 7.9651917716},
				Skill{Mu: 22.4620651999, Sigma: 7.9651917716},
				Skill{Mu: 22.4620651999, Sigma: 7.9651917716},
			),
		},
		{
			name:    "2v1 uneven win",
			red:     skillsOf(Skill{Mu: 28, Sigma: 6}, Skill{Mu: 22, Sigma: 7}),
			blu:     skillsOf(Skill{Mu: 25, Sigma: 8}),
			outcome: RatingWin,
			wantRed: skillsOf(
				Skill{Mu: 28.2504053117, Sigma: 5.9045785355},
				Skill{Mu: 22.3408120125, Sigma: 6.8476129980},
			),
			wantBlu: skillsOf(Skill{Mu: 24.5548725635, Sigma: 7.7714500463}),
		},
		{
			name:    "2v2 uneven draw",
			red:     skillsOf(Skill{Mu: 30, Sigma: 4}, Skill{Mu: 24, Sigma: 6}),
			blu:     skillsOf(Skill{Mu: 26, Sigma: 7}, Skill{Mu: 20, Sigma: 8.3}),
			outcome: RatingDraw,
			wantRed: skillsOf(
				Skill{Mu: 29.4658299962, Sigma: 3.8649890718},
				Skill{Mu: 22.7984071715, Sigma: 5.5316121106},
			),
			wantBlu: skillsOf(
				Skill{Mu: 27.6354176646, Sigma: 6.2440843629},
				Skill{Mu: 22.2991696638, Sigma: 7.0068694687},
			),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			redBefore := cloneSkills(testCase.red)
			bluBefore := cloneSkills(testCase.blu)

			gotRed, gotBlu := updateTeamRatings(testCase.red, testCase.blu, testCase.outcome)

			requireSkillsClose(t, testCase.wantRed, gotRed)
			requireSkillsClose(t, testCase.wantBlu, gotBlu)
			require.Equal(t, redBefore, testCase.red, "red input must not be mutated")
			require.Equal(t, bluBefore, testCase.blu, "blu input must not be mutated")
		})
	}
}

func TestPredictedWinProbability(t *testing.T) {
	t.Parallel()

	team := []Skill{{Mu: RatingMu, Sigma: RatingSigma}, {Mu: RatingMu, Sigma: RatingSigma}}

	require.InDelta(t, 0.5, PredictedWinProbability(team, team), 1e-9)
	require.InDelta(t, 0.5, PredictedWinProbability(team, nil), 1e-9)
	require.InDelta(t, 0.5, PredictedWinProbability([]Skill{{Mu: RatingMu, Sigma: 0}}, []Skill{{Mu: RatingMu, Sigma: 0}}), 1e-9)

	strong := []Skill{{Mu: RatingMu + 10, Sigma: RatingSigma}}
	weak := []Skill{{Mu: RatingMu, Sigma: RatingSigma}}
	require.Greater(t, PredictedWinProbability(strong, weak), 0.5)
	require.Less(t, PredictedWinProbability(weak, strong), 0.5)
	require.InDelta(t, 1, PredictedWinProbability(strong, weak)+PredictedWinProbability(weak, strong), 1e-9)
}

func TestDrawReducesDeviation(t *testing.T) {
	t.Parallel()

	red := []Skill{{Mu: RatingMu, Sigma: RatingSigma}}
	blu := []Skill{{Mu: RatingMu, Sigma: RatingSigma}}

	gotRed, gotBlu := updateTeamRatings(red, blu, RatingDraw)

	require.InDelta(t, RatingMu, gotRed[0].Mu, 1e-9)
	require.InDelta(t, RatingMu, gotBlu[0].Mu, 1e-9)
	require.Less(t, gotRed[0].Sigma, RatingSigma)
	require.Equal(t, gotRed[0], gotBlu[0])
}

func TestUpsetMovesMoreThanExpectedWin(t *testing.T) {
	t.Parallel()

	even := []Skill{{Mu: RatingMu, Sigma: RatingSigma}}
	equalWinRed, _ := updateTeamRatings(even, skillsOf(even[0]), RatingWin)
	expectedGain := equalWinRed[0].Mu - RatingMu

	underdog := []Skill{{Mu: RatingMu, Sigma: RatingSigma}}
	favourite := []Skill{{Mu: RatingMu + 6, Sigma: 4}}
	upsetRed, upsertBlu := updateTeamRatings(underdog, favourite, RatingWin)
	upsetGain := upsetRed[0].Mu - RatingMu

	require.Greater(t, upsetGain, expectedGain)
	require.Less(t, upsertBlu[0].Mu, favourite[0].Mu)
}

func TestSkillStaysWithinBounds(t *testing.T) {
	t.Parallel()

	extreme := []struct {
		name    string
		red     []Skill
		blu     []Skill
		outcome RatingOutcome
	}{
		{
			name:    "huge gap",
			red:     []Skill{{Mu: 1000, Sigma: 1}},
			blu:     []Skill{{Mu: -1000, Sigma: 1}},
			outcome: RatingDraw,
		},
		{
			name:    "huge gap win",
			red:     []Skill{{Mu: 1000, Sigma: 1}},
			blu:     []Skill{{Mu: -1000, Sigma: 1}},
			outcome: RatingWin,
		},
		{
			name:    "huge gap loss",
			red:     []Skill{{Mu: 1000, Sigma: 1}},
			blu:     []Skill{{Mu: -1000, Sigma: 1}},
			outcome: RatingLoss,
		},
		{
			name:    "confident teams",
			red:     []Skill{{Mu: RatingMu, Sigma: minRatingSigma}},
			blu:     []Skill{{Mu: RatingMu, Sigma: minRatingSigma}},
			outcome: RatingWin,
		},
		{
			name:    "large teams",
			red:     skillsOf(make([]Skill, 32)...),
			blu:     skillsOf(make([]Skill, 32)...),
			outcome: RatingWin,
		},
	}

	for _, testCase := range extreme {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			gotRed, gotBlu := updateTeamRatings(testCase.red, testCase.blu, testCase.outcome)

			for _, team := range [][]Skill{gotRed, gotBlu} {
				for _, skill := range team {
					require.False(t, math.IsNaN(skill.Mu), "mu must be finite")
					require.False(t, math.IsNaN(skill.Sigma), "sigma must be finite")
					require.False(t, math.IsInf(skill.Mu, 0), "mu must be finite")
					require.False(t, math.IsInf(skill.Sigma, 0), "sigma must be finite")
					require.GreaterOrEqual(t, skill.Sigma, minRatingSigma)
					require.LessOrEqual(t, skill.Sigma, RatingSigma)
				}
			}
		})
	}
}

func TestUpdateTeamRatingsDeterministic(t *testing.T) {
	t.Parallel()

	red := []Skill{{Mu: 27, Sigma: 6}, {Mu: 22, Sigma: 8}}
	blu := []Skill{{Mu: 26, Sigma: 7}, {Mu: 24, Sigma: 5}}

	firstRed, firstBlu := updateTeamRatings(red, blu, RatingWin)
	for range 10 {
		gotRed, gotBlu := updateTeamRatings(red, blu, RatingWin)
		require.Equal(t, firstRed, gotRed)
		require.Equal(t, firstBlu, gotBlu)
	}
}

func TestEmptyTeamsAreNoOps(t *testing.T) {
	t.Parallel()

	team := []Skill{{Mu: RatingMu, Sigma: RatingSigma}}

	gotRed, gotBlu := updateTeamRatings(team, nil, RatingWin)

	require.Equal(t, team, gotRed)
	require.Empty(t, gotBlu)
}

func TestRatingDisplayScale(t *testing.T) {
	t.Parallel()

	require.Equal(t, DefaultRating, RatingDisplay(RatingMu))
	require.Equal(t, 1200, RatingDisplay(RatingMu+5))
	require.Equal(t, DefaultRating, RatingDisplayConservative(RatingMu, 0))
	require.Equal(t, 600, RatingDisplayConservative(RatingMu+5, 5))
	require.Equal(t, 40, RatingDisplaySigma(1))
}

func TestNormalInvCDF(t *testing.T) {
	t.Parallel()

	require.InDelta(t, 0, normalInvCDF(0.5), 1e-9)
	require.InDelta(t, 1.6448536269514722, normalInvCDF(0.95), 1e-8)
	require.InDelta(t, -1.6448536269514722, normalInvCDF(0.05), 1e-8)
	require.InDelta(t, 0.12566134685507402, normalInvCDF(0.55), 1e-9)
	require.True(t, math.IsInf(normalInvCDF(0), -1))
	require.True(t, math.IsInf(normalInvCDF(1), 1))
}
