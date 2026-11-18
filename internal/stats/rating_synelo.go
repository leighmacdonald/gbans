package stats

import (
	"math"
	"time"
)

const (
	// SynEloRatingStart is R_0, the starting rating of an unrated player.
	SynEloRatingStart = 1000.0
	// SynEloRatingFloor is R_min, the absolute rating floor.
	SynEloRatingFloor = 100.0
	// SynEloShadowGrade is the rating ceiling while a player is in shadow.
	SynEloShadowGrade = 2200.0
	// SynEloShadowRounds is T_shadow, the number of rounds before a rating is
	// made visible.
	SynEloShadowRounds = 50
	// SynEloStaleDays is T_stale, the inactivity threshold for the stale flag.
	SynEloStaleDays = 30
	// SynEloKMin and SynEloKMax bound the dynamic K-factor.
	SynEloKMin = 2.0
	SynEloKMax = 72.0
	// SynEloScale is the logistic scale for a single player per team.
	SynEloScale = 400.0
	// SynEloWindow is W, the number of past observations used for convergence.
	SynEloWindow = 500
	// SynEloSigma is the Gaussian compression width for the K-factor.
	SynEloSigma = 400.0

	synEloVelWeight = 0.25
	synEloWREWeight = 0.25
	synEloVolWeight = 0.50
	synEloWRENorm   = 0.10

	synEloStaleDuration = SynEloStaleDays * 24 * time.Hour
)

// SynEloCalculator implements the rating algorithm described in issue #696:
// team Elo with a dynamic K-factor derived from each player's convergence
// score, a Gaussian compression term, Banker's rounding, a rating floor and a
// display-only shadow period. The scalar rating is stored in Skill.Mu.
type SynEloCalculator struct{}

func (SynEloCalculator) Name() string {
	return RatingAlgorithmSynElo
}

func (SynEloCalculator) HistoryWindow() int {
	return SynEloWindow
}

func (SynEloCalculator) NewPlayerSkill() Skill {
	return Skill{Mu: SynEloRatingStart, Sigma: 0}
}

func (SynEloCalculator) UpdateMatch(red, blu []PlayerRatingState, outcome RatingOutcome) ([]MatchUpdate, []MatchUpdate) {
	redSkills := playerSkills(red)
	bluSkills := playerSkills(blu)
	if len(red) == 0 || len(blu) == 0 {
		return toMatchUpdates(cloneSkills(redSkills), 0.5), toMatchUpdates(cloneSkills(bluSkills), 0.5)
	}

	theta := synEloTheta(max(len(red), len(blu)))
	redExpected := synEloWinProbability(teamSum(redSkills), teamSum(bluSkills), theta)
	redActual := synEloActualScore(outcome)

	return synEloTeamUpdate(red, redExpected, redActual, theta),
		synEloTeamUpdate(blu, 1-redExpected, 1-redActual, theta)
}

func (SynEloCalculator) Display(ctx RatingContext) DisplayRating {
	visible := ctx.Rounds >= SynEloShadowRounds
	rating := int(math.RoundToEven(ctx.Skill.Mu))
	if !visible {
		rating = min(rating, int(SynEloShadowGrade))
	}

	stale := !ctx.LastMatchAt.IsZero() && ctx.Now.Sub(ctx.LastMatchAt) > synEloStaleDuration

	return DisplayRating{
		Rating:       rating,
		Sigma:        0,
		Conservative: rating,
		Visible:      visible,
		Stale:        stale,
	}
}

func (SynEloCalculator) WinProbability(red, blu []Skill) float64 {
	if len(red) == 0 || len(blu) == 0 {
		return 0.5
	}

	theta := synEloTheta(max(len(red), len(blu)))

	return synEloWinProbability(teamSum(red), teamSum(blu), theta)
}

func synEloTeamUpdate(team []PlayerRatingState, expected, actual, theta float64) []MatchUpdate {
	updates := make([]MatchUpdate, len(team))
	for idx, player := range team {
		// F3: convergence from the player's rolling window.
		convergence := synEloConvergence(player.History)
		// F4: Gaussian compression around the starting rating.
		compression := math.Exp(-math.Pow(player.Skill.Mu-SynEloRatingStart, 2) / (2 * SynEloSigma * SynEloSigma))
		// F5: dynamic K-factor.
		kFactor := SynEloKMin + (SynEloKMax-SynEloKMin)*convergence*compression
		updated := math.Max(SynEloRatingFloor, player.Skill.Mu+kFactor*(actual-expected))

		updates[idx] = MatchUpdate{
			Skill:    Skill{Mu: math.RoundToEven(updated), Sigma: 0},
			Expected: expected,
		}
	}

	return updates
}

func synEloConvergence(history []RatingObservation) float64 {
	if len(history) < 2 {
		return 0
	}

	mid := len(history) / 2
	older := history[:mid]
	newer := history[mid:]

	var oldSum, newSum, outcomeSum, expectedSum float64
	for _, observation := range older {
		oldSum += observation.Rating
	}
	for _, observation := range newer {
		newSum += observation.Rating
	}
	for _, observation := range history {
		outcomeSum += synEloScore(observation.Outcome)
		expectedSum += observation.Expected
	}

	velocity := math.Abs(newSum/float64(len(newer))-oldSum/float64(len(older))) / SynEloKMax
	winRateExcess := math.Abs(outcomeSum/float64(len(history))-expectedSum/float64(len(history))) / synEloWRENorm

	mean := (oldSum + newSum) / float64(len(history))
	var variance float64
	for _, observation := range history {
		variance += math.Pow(observation.Rating-mean, 2)
	}
	volatility := math.Sqrt(variance/float64(len(history))) / SynEloKMax

	score := synEloVelWeight*velocity + synEloWREWeight*winRateExcess + synEloVolWeight*volatility

	return math.Min(1, math.Max(0, score))
}

func synEloWinProbability(sumRed, sumBlu, theta float64) float64 {
	if theta <= 0 {
		return 0.5
	}

	return 1 / (1 + math.Exp(-(sumRed-sumBlu)/theta))
}

// synEloTheta derives the pool's logistic scale from the largest team size.
// For a single player per team it matches standard Elo's 400; for 12v12 it
// gives 2400, as described in the issue.
func synEloTheta(maxTeamSize int) float64 {
	if maxTeamSize < 1 {
		return SynEloScale
	}

	median := (maxTeamSize + 1) / 2

	return SynEloScale * float64(median)
}

func synEloActualScore(outcome RatingOutcome) float64 {
	switch outcome {
	case RatingWin:
		return 1
	case RatingDraw:
		return 0.5
	default:
		return 0
	}
}

func synEloScore(outcome RatingOutcome) float64 {
	return synEloActualScore(outcome)
}

func teamSum(team []Skill) float64 {
	var total float64
	for _, skill := range team {
		total += skill.Mu
	}

	return total
}
