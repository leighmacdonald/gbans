package stats

import "math"

const (
	// EloInitialRating is the starting rating for an unrated player.
	EloInitialRating = 1000.0
	// EloKFactor controls how many rating points a single match can move.
	EloKFactor = 32.0
	// eloScale is the rating difference that corresponds to a 10x odds ratio.
	eloScale = 400.0
)

// EloCalculator is a classic Elo rating algorithm with team support. It is an
// alternative to the default TrueSkillCalculator and demonstrates that the
// update, display and win-probability logic are all pluggable through the
// RatingCalculator interface. The scalar Elo rating is stored in Skill.Mu;
// Skill.Sigma is unused and stays zero.
type EloCalculator struct{}

func (EloCalculator) Name() string {
	return RatingAlgorithmElo
}

func (EloCalculator) NewPlayerSkill() Skill {
	return Skill{Mu: EloInitialRating, Sigma: 0}
}

func (EloCalculator) UpdateMatch(red, blu []Skill, outcome RatingOutcome) ([]Skill, []Skill) {
	updatedRed := cloneSkills(red)
	updatedBlu := cloneSkills(blu)
	if len(red) == 0 || len(blu) == 0 {
		return updatedRed, updatedBlu
	}

	redExpected := eloExpectedScore(teamAverage(red), teamAverage(blu))
	var redActual float64
	switch outcome {
	case RatingWin:
		redActual = 1
	case RatingDraw:
		redActual = 0.5
	case RatingLoss:
		redActual = 0
	default:
		return updatedRed, updatedBlu
	}

	// Each team's total adjustment is K * (actual - expected); dividing by the
	// roster size keeps the team change independent of how many players it has.
	redDelta := EloKFactor * (redActual - redExpected) / float64(len(red))
	bluDelta := EloKFactor * ((1 - redActual) - (1 - redExpected)) / float64(len(blu))

	for idx := range updatedRed {
		updatedRed[idx].Mu += redDelta
	}

	for idx := range updatedBlu {
		updatedBlu[idx].Mu += bluDelta
	}

	return updatedRed, updatedBlu
}

func (EloCalculator) Display(skill Skill) DisplayRating {
	rating := int(math.Round(skill.Mu))

	return DisplayRating{Rating: rating, Sigma: 0, Conservative: rating}
}

func (EloCalculator) WinProbability(red, blu []Skill) float64 {
	if len(red) == 0 || len(blu) == 0 {
		return 0.5
	}

	return eloExpectedScore(teamAverage(red), teamAverage(blu))
}

func eloExpectedScore(redAverage, bluAverage float64) float64 {
	return 1 / (1 + math.Pow(10, (bluAverage-redAverage)/eloScale))
}

func teamAverage(team []Skill) float64 {
	var total float64
	for _, skill := range team {
		total += skill.Mu
	}

	return total / float64(len(team))
}
