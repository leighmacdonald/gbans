package stats

import "math"

// TrueSkill-style rating parameters, using the classic TrueSkill defaults.
// A new player starts at mu=25, sigma=25/3, which maps to a display rating of
// 1000.
const (
	// RatingMu is the initial skill mean for a new player.
	RatingMu = 25.0
	// RatingSigma is the initial skill standard deviation for a new player.
	RatingSigma = RatingMu / 3.0
	// RatingBeta is the per-player performance noise.
	RatingBeta = RatingSigma / 2.0
	// RatingTau is the per-match skill drift added before each update.
	RatingTau = RatingSigma / 100.0
	// RatingDrawProbability is the assumed probability of a draw between
	// equally skilled teams.
	RatingDrawProbability = 0.10
	// ProvisionalMatchCount is the number of rated matches a player needs
	// before their computed rating is exposed instead of the default.
	ProvisionalMatchCount = 5
	// DefaultRating is the display rating assigned to provisional players.
	DefaultRating = 1000
	// RatingDisplayScale maps one skill point onto display rating points.
	RatingDisplayScale = 40.0
	// minRatingSigma prevents a rating from becoming impossibly confident.
	minRatingSigma = 0.5
	// minNormalCDF guards the win-probability ratio for extreme skill gaps.
	minNormalCDF = 1e-300
)

// RatingOutcome is a match result from a single player's team perspective.
type RatingOutcome int8

const (
	// RatingDraw means the player's team drew the match.
	RatingDraw RatingOutcome = 0
	// RatingWin means the player's team won the match.
	RatingWin RatingOutcome = 1
	// RatingLoss means the player's team lost the match.
	RatingLoss RatingOutcome = -1
)

// Skill is a Gaussian estimate of a player's contribution to a team.
type Skill struct {
	Mu    float64
	Sigma float64
}

// NewSkill returns the starting skill of an unrated player and clamps the
// deviation into the supported range.
func NewSkill(mu, sigma float64) Skill {
	return newSkill(mu, sigma*sigma)
}

// DisplayRating converts the skill to the public rating scale.
func (s Skill) DisplayRating() int {
	return RatingDisplay(s.Mu)
}

// RatingDisplay converts a skill mean to the public display scale.
func RatingDisplay(mu float64) int {
	return int(math.Round(DefaultRating + RatingDisplayScale*(mu-RatingMu)))
}

// RatingDisplaySigma converts a skill deviation to display points.
func RatingDisplaySigma(sigma float64) int {
	return int(math.Round(RatingDisplayScale * sigma))
}

// RatingDisplayConservative is the lower-confidence-bound display rating.
func RatingDisplayConservative(mu, sigma float64) int {
	return RatingDisplay(mu - 3*sigma)
}

// PredictedWinProbability returns the default calculator's estimate of the red
// team winning. Callers using a configured calculator should call its
// WinProbability method instead.
func PredictedWinProbability(red, blu []Skill) float64 {
	return TrueSkillCalculator{}.WinProbability(red, blu)
}

// RatingCalculator computes per-player skill estimates from two-team match
// outcomes. The default implementation is TrueSkillCalculator.
//
// Implementations must be deterministic: replaying the same match sequence
// must produce identical results, because ratings are rebuilt from scratch on
// every refresh. Skill is the canonical persisted representation, so a
// calculator must map its native state onto mu/sigma and keep it stable across
// refreshes. Display controls the public scale and visibility, which are
// persisted alongside the skill so SQL views do not depend on a specific
// algorithm.
type RatingCalculator interface {
	// Name uniquely identifies the algorithm in configuration.
	Name() string

	// HistoryWindow is the number of past observations a calculator needs.
	// Returning zero avoids building history windows for stateless algorithms.
	HistoryWindow() int

	// NewPlayerSkill returns the skill of a player with no rated matches yet.
	NewPlayerSkill() Skill

	// UpdateMatch applies one match outcome to the skills of both teams and
	// returns each player's updated skill and the pre-match expected score.
	// outcome is from the red team's perspective. Inputs must not be mutated.
	UpdateMatch(red, blu []PlayerRatingState, outcome RatingOutcome) (newRed, newBlu []MatchUpdate)

	// Display converts a skill estimate and its context to the public form.
	Display(ctx RatingContext) DisplayRating

	// WinProbability returns the probability that the red team wins.
	WinProbability(red, blu []Skill) float64
}

// TrueSkillCalculator is the default RatingCalculator: two-team TrueSkill
// (Bradley–Terry) updates with draw support and per-match skill drift.
type TrueSkillCalculator struct{}

func (TrueSkillCalculator) Name() string {
	return RatingAlgorithmTrueSkill
}

func (TrueSkillCalculator) HistoryWindow() int {
	return 0
}

func (TrueSkillCalculator) NewPlayerSkill() Skill {
	return Skill{Mu: RatingMu, Sigma: RatingSigma}
}

func (TrueSkillCalculator) UpdateMatch(red, blu []PlayerRatingState, outcome RatingOutcome) ([]MatchUpdate, []MatchUpdate) {
	redSkills := make([]Skill, len(red))
	bluSkills := make([]Skill, len(blu))
	for idx, player := range red {
		redSkills[idx] = player.Skill
	}
	for idx, player := range blu {
		bluSkills[idx] = player.Skill
	}

	expectedRed := TrueSkillCalculator{}.WinProbability(redSkills, bluSkills)
	newRed, newBlu := updateTeamRatings(redSkills, bluSkills, outcome)

	return toMatchUpdates(newRed, expectedRed), toMatchUpdates(newBlu, 1-expectedRed)
}

func (TrueSkillCalculator) Display(ctx RatingContext) DisplayRating {
	visible := ctx.Matches >= ProvisionalMatchCount
	if !visible {
		return DisplayRating{Visible: false, Rating: DefaultRating}
	}

	return DisplayRating{
		Rating:       RatingDisplay(ctx.Skill.Mu),
		Sigma:        RatingDisplaySigma(ctx.Skill.Sigma),
		Conservative: RatingDisplayConservative(ctx.Skill.Mu, ctx.Skill.Sigma),
		Visible:      true,
	}
}

func toMatchUpdates(skills []Skill, expected float64) []MatchUpdate {
	updates := make([]MatchUpdate, len(skills))
	for idx, skill := range skills {
		updates[idx] = MatchUpdate{Skill: skill, Expected: expected}
	}

	return updates
}

func (TrueSkillCalculator) WinProbability(red, blu []Skill) float64 {
	if len(red) == 0 || len(blu) == 0 {
		return 0.5
	}

	muRed, varRed := teamMoments(red, false)
	muBlu, varBlu := teamMoments(blu, false)
	denom := varRed + varBlu + float64(len(red)+len(blu))*RatingBeta*RatingBeta
	if denom <= 0 {
		return 0.5
	}

	return normalCDF((muRed - muBlu) / math.Sqrt(denom))
}

// updateTeamRatings applies a two-team TrueSkill update. outcome is from the
// red team's perspective. The inputs are not mutated.
func updateTeamRatings(red, blu []Skill, outcome RatingOutcome) ([]Skill, []Skill) {
	if len(red) == 0 || len(blu) == 0 {
		return cloneSkills(red), cloneSkills(blu)
	}

	muRed, varRed := teamMoments(red, true)
	muBlu, varBlu := teamMoments(blu, true)

	playerCount := len(red) + len(blu)
	cSquared := varRed + varBlu + float64(playerCount)*RatingBeta*RatingBeta
	if cSquared <= 0 {
		return cloneSkills(red), cloneSkills(blu)
	}

	diffStdDev := math.Sqrt(cSquared)
	// The draw margin is expressed in performance units, so normalise it by
	// the team-difference deviation to compare it against the skill gap.
	margin := drawMargin(playerCount) / diffStdDev
	diff := (muRed - muBlu) / diffStdDev

	var (
		vFactor, wFactor float64
		// redFactor is the mean multiplier from the red team's perspective: it
		// is positive when red should gain skill.
		redFactor float64
	)

	switch outcome {
	case RatingDraw:
		vFactor, wFactor = drawUpdate(diff, margin)
		redFactor = vFactor
	case RatingWin:
		vFactor, wFactor = winUpdate(diff - margin)
		redFactor = vFactor
	case RatingLoss:
		vFactor, wFactor = winUpdate(-diff - margin)
		redFactor = -vFactor
	default:
		return cloneSkills(red), cloneSkills(blu)
	}

	// Each team shifts in proportion to its own uncertainty, so the update is
	// asymmetric when the teams have different deviations.
	deltaRed := (varRed / diffStdDev) * redFactor
	deltaBlu := -(varBlu / diffStdDev) * redFactor
	scaleRed := 1 - (varRed/cSquared)*wFactor
	scaleBlu := 1 - (varBlu/cSquared)*wFactor

	return applyTeamUpdate(red, varRed, deltaRed, scaleRed), applyTeamUpdate(blu, varBlu, deltaBlu, scaleBlu)
}

// applyTeamUpdate shifts a team's mean by deltaMu and shrinks each player's
// variance relative to the team update, weighted by their share of the team
// uncertainty.
func applyTeamUpdate(team []Skill, teamVariance, deltaMu, varianceScale float64) []Skill {
	updated := make([]Skill, len(team))
	for idx, player := range team {
		variance := player.Sigma*player.Sigma + RatingTau*RatingTau
		share := variance / teamVariance
		newVariance := variance - share*share*teamVariance*(1-varianceScale)
		updated[idx] = newSkill(player.Mu+share*deltaMu, newVariance)
	}

	return updated
}

// teamMoments sums the team mean and variance. When includeDynamics is set,
// each player's per-match drift variance is added first.
func teamMoments(team []Skill, includeDynamics bool) (float64, float64) {
	var totalMu, totalVariance float64

	for _, player := range team {
		totalMu += player.Mu
		totalVariance += player.Sigma * player.Sigma
		if includeDynamics {
			totalVariance += RatingTau * RatingTau
		}
	}

	return totalMu, totalVariance
}

// newSkill builds a clamped skill estimate from a posterior mean and variance.
func newSkill(mu, variance float64) Skill {
	sigma := math.Sqrt(math.Max(variance, minRatingSigma*minRatingSigma))
	sigma = math.Min(sigma, RatingSigma)

	return Skill{Mu: mu, Sigma: sigma}
}

func cloneSkills(team []Skill) []Skill {
	return append([]Skill(nil), team...)
}

// winUpdate returns the TrueSkill v and w values for a decisive result. diff
// is the winner-loser skill gap in standard deviations, already reduced by the
// draw margin.
func winUpdate(diff float64) (float64, float64) {
	denom := normalCDF(diff)

	var vFactor float64
	if denom < minNormalCDF {
		// The result was essentially impossible under the model. The tail
		// ratio approaches -diff, which keeps the update finite.
		vFactor = -diff
	} else {
		vFactor = normalPDF(diff) / denom
	}

	return vFactor, clampW(vFactor * (vFactor + diff))
}

// drawUpdate returns the TrueSkill v and w values for a draw. diff is the red
// minus blue skill gap in standard deviations and margin is the draw margin in
// the same units. v is signed from the red team's perspective: a stronger red
// team is pulled down.
func drawUpdate(diff, margin float64) (float64, float64) {
	absDiff := math.Abs(diff)
	lower := margin - absDiff
	upper := -margin - absDiff
	denom := normalCDF(lower) - normalCDF(upper)
	if denom < minNormalCDF {
		return 0, 1
	}

	vFactor := (normalPDF(upper) - normalPDF(lower)) / denom
	if diff < 0 {
		vFactor = -vFactor
	}

	wFactor := vFactor*vFactor + (lower*normalPDF(lower)-upper*normalPDF(upper))/denom

	return vFactor, clampW(wFactor)
}

// drawMargin returns the draw margin in performance units for a game with
// playerCount total participants.
func drawMargin(playerCount int) float64 {
	return normalInvCDF((RatingDrawProbability+1)/2) * math.Sqrt(float64(playerCount)) * RatingBeta
}

func clampW(w float64) float64 {
	return math.Min(math.Max(w, 1e-12), 1-1e-12)
}

func normalPDF(x float64) float64 {
	return math.Exp(-0.5*x*x) / math.Sqrt(2*math.Pi)
}

func normalCDF(x float64) float64 {
	return 0.5 * math.Erfc(-x/math.Sqrt2)
}

// normalInvCDF is the Acklam approximation of the standard normal inverse CDF.
// It is accurate to roughly 1e-9 over the open unit interval.
//
//nolint:varnamelen // names follow the published Acklam algorithm.
func normalInvCDF(p float64) float64 {
	if p <= 0 {
		return math.Inf(-1)
	}

	if p >= 1 {
		return math.Inf(1)
	}

	// Coefficients from Peter Acklam's inverse normal CDF algorithm.
	a := [...]float64{
		-3.969683028665376e+01, 2.209460984245205e+02, -2.759285104469687e+02,
		1.383577518672690e+02, -3.066479806614716e+01, 2.506628277459239e+00,
	}
	b := [...]float64{
		-5.447609879822406e+01, 1.615858368580409e+02, -1.556989798598866e+02,
		6.680131188771972e+01, -1.328068155288572e+01,
	}
	c := [...]float64{
		-7.784894002430293e-03, -3.223964580411365e-01, -2.400758277161838e+00,
		-2.549732539343734e+00, 4.374664141464968e+00, 2.938163982698783e+00,
	}
	d := [...]float64{
		7.784695709041462e-03, 3.224671290700398e-01, 2.445134137142996e+00,
		3.754408661907416e+00,
	}
	const low = 0.02425

	q := p - 0.5
	if math.Abs(q) <= 0.5-low {
		r := q * q

		return (((((a[0]*r+a[1])*r+a[2])*r+a[3])*r+a[4])*r + a[5]) * q /
			(((((b[0]*r+b[1])*r+b[2])*r+b[3])*r+b[4])*r + 1)
	}

	r := p
	if p > 0.5 {
		r = 1 - p
	}

	r = math.Sqrt(-2 * math.Log(r))
	x := (((((c[0]*r+c[1])*r+c[2])*r+c[3])*r+c[4])*r + c[5]) /
		((((d[0]*r+d[1])*r+d[2])*r+d[3])*r + 1)
	if p <= 0.5 {
		return x
	}

	return -x
}
