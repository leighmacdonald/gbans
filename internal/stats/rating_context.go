package stats

import (
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

// RatingObservation is one recorded rating event for a player. It is the input
// an algorithm needs to compute history-dependent quantities such as a
// convergence score or a Brier score.
type RatingObservation struct {
	// Rating is the player's rating before the event was applied.
	Rating float64
	// Expected is the probability of the player's team winning, from their
	// perspective, as estimated before the event.
	Expected float64
	// Outcome is the result from the player's team perspective.
	Outcome RatingOutcome
}

// PlayerRatingState is the per-player context passed to a calculator for one
// rating event. History is ordered oldest first and holds at most
// RatingCalculator.HistoryWindow observations.
type PlayerRatingState struct {
	SteamID steamid.SteamID
	Skill   Skill
	Matches int
	Rounds  int
	History []RatingObservation
}

// MatchUpdate is a player's state after one rating event. Expected is the
// pre-event win probability the calculator used, recorded for later history.
type MatchUpdate struct {
	Skill    Skill
	Expected float64
}

// RatingContext is the context for converting a skill estimate to its public
// display form. Visibility and staleness are algorithm policy rather than
// fixed database rules.
type RatingContext struct {
	Skill       Skill
	Matches     int
	Rounds      int
	LastMatchAt time.Time
	// Now is the replay's reference time, used for deterministic staleness.
	Now time.Time
}

// DisplayRating is the public, display-scaled form of a Skill. Each calculator
// owns this conversion so alternatives are not tied to the TrueSkill scale.
type DisplayRating struct {
	Rating       int
	Sigma        int
	Conservative int
	// Visible reports whether the rating should be exposed publicly and ranked.
	Visible bool
	// Stale reports whether the player has been inactive long enough to flag.
	Stale bool
}
