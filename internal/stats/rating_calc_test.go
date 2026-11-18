//nolint:testpackage // white-box tests for the rating replay.
package stats

import (
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/maps"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func ratingRow(
	matchID uuid.UUID,
	start time.Time,
	scoreRed, scoreBlu int,
	steamID steamid.SteamID,
	bucketID int32,
	team string,
	rounds, roundID int,
) ratedOutcomeRow {
	row := ratedOutcomeRow{
		MatchID:       matchID,
		StatsBucketID: bucketID,
		StartTime:     start,
		ScoreRed:      scoreRed,
		ScoreBlu:      scoreBlu,
		SteamID:       steamID,
		RedRounds:     0,
		BluRounds:     0,
	}

	if team == "red" {
		row.RedRounds = rounds
		row.LastRedRound = roundID
	} else {
		row.BluRounds = rounds
		row.LastBluRound = roundID
	}

	return row
}

func ratingsByPlayer(ratings []PlayerRating) map[steamid.SteamID]PlayerRating {
	out := make(map[steamid.SteamID]PlayerRating, len(ratings))
	for _, rating := range ratings {
		out[rating.SteamID] = rating
	}

	return out
}

func historyByPlayer(history []RatingHistory) map[steamid.SteamID][]RatingHistory {
	out := make(map[steamid.SteamID][]RatingHistory)
	for _, entry := range history {
		out[entry.SteamID] = append(out[entry.SteamID], entry)
	}

	return out
}

func TestReplayRatingsProgression(t *testing.T) {
	t.Parallel()

	playerA := steamid.New("[U:1:7200000001]")
	playerB := steamid.New("[U:1:7200000002]")
	start := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)

	rows := make([]ratedOutcomeRow, 0, 12)
	for match := range 6 {
		matchID, errUUID := uuid.NewV4()
		require.NoError(t, errUUID)

		matchStart := start.Add(time.Duration(match) * time.Hour)
		// Alternate the winner so both players stay close to the default.
		scoreRed, scoreBlu := 2, 1
		if match%2 == 1 {
			scoreRed, scoreBlu = 1, 2
		}

		rows = append(rows,
			ratingRow(matchID, matchStart, scoreRed, scoreBlu, playerA, 1, "red", 3, match*10+1),
			ratingRow(matchID, matchStart, scoreRed, scoreBlu, playerB, 1, "blu", 3, match*10+2),
		)
	}

	ratings, history := replayRatings(rows, TrueSkillCalculator{})

	require.Len(t, ratings, 2)
	require.Len(t, history, 12)

	byPlayer := ratingsByPlayer(ratings)
	for _, steamID := range []steamid.SteamID{playerA, playerB} {
		rating := byPlayer[steamID]
		require.Equal(t, 6, rating.MatchesPlayed)
		require.Equal(t, 3, rating.Wins)
		require.Equal(t, 3, rating.Losses)
		require.Zero(t, rating.Draws)
		require.InDelta(t, RatingMu, rating.Mu, 1.5)
		require.Less(t, rating.Sigma, RatingSigma)
		require.False(t, rating.LastMatchAt.IsZero())
		require.Equal(t, start.Add(5*time.Hour), rating.LastMatchAt)
	}

	// The winner of the most recent match must end up ahead.
	require.Greater(t, byPlayer[playerB].Mu, byPlayer[playerA].Mu)

	historyByID := historyByPlayer(history)
	for _, steamID := range []steamid.SteamID{playerA, playerB} {
		entries := historyByID[steamID]
		require.Len(t, entries, 6)
		for idx, entry := range entries {
			require.Equal(t, idx+1, entry.MatchesPlayed)
		}
	}
}

func TestReplayRatingsDraw(t *testing.T) {
	t.Parallel()

	playerA := steamid.New("[U:1:7200000011]")
	playerB := steamid.New("[U:1:7200000012]")
	matchID, errUUID := uuid.NewV4()
	require.NoError(t, errUUID)
	start := time.Date(2026, 2, 2, 12, 0, 0, 0, time.UTC)

	rows := []ratedOutcomeRow{
		ratingRow(matchID, start, 2, 2, playerA, 1, "red", 4, 1),
		ratingRow(matchID, start, 2, 2, playerB, 1, "blu", 4, 2),
	}

	ratings, history := replayRatings(rows, TrueSkillCalculator{})

	require.Len(t, ratings, 2)
	require.Len(t, history, 2)

	for _, rating := range ratings {
		require.Equal(t, 1, rating.MatchesPlayed)
		require.Equal(t, 1, rating.Draws)
		require.InDelta(t, RatingMu, rating.Mu, 1e-9)
		require.Less(t, rating.Sigma, RatingSigma)
	}

	for _, entry := range history {
		require.Equal(t, RatingDraw, entry.Outcome)
	}
}

func TestReplayRatingsSkipsIncompleteMatches(t *testing.T) {
	t.Parallel()

	playerA := steamid.New("[U:1:7200000021]")
	playerB := steamid.New("[U:1:7200000022]")
	start := time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)

	// One-sided match.
	oneSided, errUUID := uuid.NewV4()
	require.NoError(t, errUUID)

	// A rated match must still count normally.
	rated, errRated := uuid.NewV4()
	require.NoError(t, errRated)

	rows := []ratedOutcomeRow{
		ratingRow(oneSided, start, 2, 0, playerA, 1, "red", 2, 1),
		ratingRow(rated, start.Add(time.Hour), 2, 0, playerA, 1, "red", 2, 2),
		ratingRow(rated, start.Add(time.Hour), 2, 0, playerB, 1, "blu", 2, 3),
	}

	ratings, history := replayRatings(rows, TrueSkillCalculator{})

	require.Len(t, ratings, 2)
	require.Len(t, history, 2)

	byPlayer := ratingsByPlayer(ratings)
	require.Equal(t, 1, byPlayer[playerA].MatchesPlayed)
	require.Equal(t, 1, byPlayer[playerB].MatchesPlayed)
}

func TestReplayRatingsBucketIsolation(t *testing.T) {
	t.Parallel()

	player := steamid.New("[U:1:7200000031]")
	opponent := steamid.New("[U:1:7200000032]")
	start := time.Date(2026, 2, 4, 12, 0, 0, 0, time.UTC)

	rows := make([]ratedOutcomeRow, 0, 4)
	for _, bucket := range []int32{1, 2} {
		matchID, errUUID := uuid.NewV4()
		require.NoError(t, errUUID)

		rows = append(rows,
			ratingRow(matchID, start, 2, 0, player, bucket, "red", 2, 1),
			ratingRow(matchID, start, 2, 0, opponent, bucket, "blu", 2, 2),
		)
	}

	ratings, _ := replayRatings(rows, TrueSkillCalculator{})

	require.Len(t, ratings, 4)
	for _, rating := range ratings {
		require.Equal(t, 1, rating.MatchesPlayed)
	}
}

func TestReplayRatingsDeterministic(t *testing.T) {
	t.Parallel()

	playerA := steamid.New("[U:1:7200000041]")
	playerB := steamid.New("[U:1:7200000042]")
	start := time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC)

	rows := make([]ratedOutcomeRow, 0, 8)
	for match := range 4 {
		matchID, errUUID := uuid.NewV4()
		require.NoError(t, errUUID)

		rows = append(rows,
			ratingRow(matchID, start.Add(time.Duration(match)*time.Hour), 2, 0, playerA, 1, "red", 2, 1),
			ratingRow(matchID, start.Add(time.Duration(match)*time.Hour), 2, 0, playerB, 1, "blu", 2, 2),
		)
	}

	firstRatings, firstHistory := replayRatings(rows, TrueSkillCalculator{})
	secondRatings, secondHistory := replayRatings(rows, TrueSkillCalculator{})

	// The run timestamp is the only field allowed to differ between runs.
	for idx := range firstRatings {
		firstRatings[idx].CalculatedOn = time.Time{}
	}
	for idx := range secondRatings {
		secondRatings[idx].CalculatedOn = time.Time{}
	}

	require.Equal(t, firstRatings, secondRatings)
	require.Equal(t, firstHistory, secondHistory)
}

func TestMatchTeams(t *testing.T) {
	t.Parallel()

	majority := steamid.New("[U:1:7200000051]")
	tieRed := steamid.New("[U:1:7200000052]")
	tieBlu := steamid.New("[U:1:7200000053]")
	unassigned := steamid.New("[U:1:7200000054]")
	start := time.Date(2026, 2, 6, 12, 0, 0, 0, time.UTC)
	matchID, errUUID := uuid.NewV4()
	require.NoError(t, errUUID)

	rows := []ratedOutcomeRow{
		// Played two rounds red and one round blu: the majority wins.
		{MatchID: matchID, StatsBucketID: 1, StartTime: start, ScoreRed: 1, ScoreBlu: 1, SteamID: majority, RedRounds: 2, BluRounds: 1, LastRedRound: 5, LastBluRound: 6},
		// Equal rounds: the last round decides.
		{MatchID: matchID, StatsBucketID: 1, StartTime: start, ScoreRed: 1, ScoreBlu: 1, SteamID: tieRed, RedRounds: 1, BluRounds: 1, LastRedRound: 9, LastBluRound: 8},
		{MatchID: matchID, StatsBucketID: 1, StartTime: start, ScoreRed: 1, ScoreBlu: 1, SteamID: tieBlu, RedRounds: 1, BluRounds: 1, LastRedRound: 7, LastBluRound: 8},
		// Never on a team: ignored.
		{MatchID: matchID, StatsBucketID: 1, StartTime: start, ScoreRed: 1, ScoreBlu: 1, SteamID: unassigned},
	}

	red, blu := matchTeams(rows)

	require.ElementsMatch(t, []steamid.SteamID{majority, tieRed}, red)
	require.ElementsMatch(t, []steamid.SteamID{tieBlu}, blu)
}

// stepCalculator is a minimal RatingCalculator that moves skill by a fixed
// step per result. It exists to prove the replay honours an injected
// calculator instead of the TrueSkill maths.
type stepCalculator struct {
	step float64
}

func (c stepCalculator) Name() string {
	return "step"
}

func (c stepCalculator) NewPlayerSkill() Skill {
	return Skill{Mu: 0, Sigma: 1}
}

func (c stepCalculator) Display(skill Skill) DisplayRating {
	return DisplayRating{Rating: int(skill.Mu), Conservative: int(skill.Mu)}
}

func (stepCalculator) WinProbability(_, _ []Skill) float64 {
	return 0.5
}

func (c stepCalculator) UpdateMatch(red, blu []Skill, outcome RatingOutcome) ([]Skill, []Skill) {
	newRed := make([]Skill, len(red))
	newBlu := make([]Skill, len(blu))
	copy(newRed, red)
	copy(newBlu, blu)

	for idx := range newRed {
		switch outcome {
		case RatingWin:
			newRed[idx].Mu += c.step
		case RatingLoss:
			newRed[idx].Mu -= c.step
		}
	}

	for idx := range newBlu {
		switch outcome {
		case RatingWin:
			newBlu[idx].Mu -= c.step
		case RatingLoss:
			newBlu[idx].Mu += c.step
		}
	}

	return newRed, newBlu
}

func TestReplayRatingsWithCustomCalculator(t *testing.T) {
	t.Parallel()

	playerA := steamid.New("[U:1:7200000061]")
	playerB := steamid.New("[U:1:7200000062]")
	start := time.Date(2026, 2, 7, 12, 0, 0, 0, time.UTC)

	rows := make([]ratedOutcomeRow, 0, 6)
	for match := range 3 {
		matchID, errUUID := uuid.NewV4()
		require.NoError(t, errUUID)

		rows = append(rows,
			ratingRow(matchID, start.Add(time.Duration(match)*time.Hour), 2, 0, playerA, 1, "red", 2, 1),
			ratingRow(matchID, start.Add(time.Duration(match)*time.Hour), 2, 0, playerB, 1, "blu", 2, 2),
		)
	}

	ratings, history := replayRatings(rows, stepCalculator{step: 1.5})

	byPlayer := ratingsByPlayer(ratings)
	// The calculator starts players at mu=0 and moves them a fixed step per
	// result, so the outcome is exact regardless of the TrueSkill maths.
	require.InDelta(t, 4.5, byPlayer[playerA].Mu, 1e-9)
	require.InDelta(t, -4.5, byPlayer[playerB].Mu, 1e-9)
	require.InDelta(t, 1, byPlayer[playerA].Sigma, 1e-9)

	byHistory := historyByPlayer(history)
	require.InDelta(t, 1.5, byHistory[playerA][0].Mu, 1e-9)
	require.InDelta(t, -1.5, byHistory[playerB][0].Mu, 1e-9)
	require.InDelta(t, 4.5, byHistory[playerA][2].Mu, 1e-9)
}

func TestNewRatingCalculatorWiring(t *testing.T) {
	t.Parallel()

	custom := stepCalculator{step: 2}

	var repo Repository
	svc := New(repo, maps.Maps{}, WithRatingCalculator(custom))
	require.Equal(t, custom, svc.calculator)

	require.IsType(t, TrueSkillCalculator{}, New(repo, maps.Maps{}).calculator)

	// A nil calculator is ignored so the default is retained.
	require.IsType(t, TrueSkillCalculator{}, New(repo, maps.Maps{}, WithRatingCalculator(nil)).calculator)
}

func TestRatingOutcomeFor(t *testing.T) {
	t.Parallel()

	require.Equal(t, RatingWin, ratingOutcomeFor(ratedOutcomeRow{ScoreRed: 2, ScoreBlu: 1}))
	require.Equal(t, RatingLoss, ratingOutcomeFor(ratedOutcomeRow{ScoreRed: 1, ScoreBlu: 2}))
	require.Equal(t, RatingDraw, ratingOutcomeFor(ratedOutcomeRow{ScoreRed: 1, ScoreBlu: 1}))

	redOutcome, bluOutcome := teamOutcomes(RatingWin)
	require.Equal(t, RatingWin, redOutcome)
	require.Equal(t, RatingLoss, bluOutcome)

	redOutcome, bluOutcome = teamOutcomes(RatingLoss)
	require.Equal(t, RatingLoss, redOutcome)
	require.Equal(t, RatingWin, bluOutcome)

	redOutcome, bluOutcome = teamOutcomes(RatingDraw)
	require.Equal(t, RatingDraw, redOutcome)
	require.Equal(t, RatingDraw, bluOutcome)
}
