package stats

import (
	"errors"
	"fmt"
	"strings"
)

// Supported rating algorithm names, as used by the rating_algorithm setting.
const (
	RatingAlgorithmTrueSkill = "trueskill"
	RatingAlgorithmElo       = "elo"
	RatingAlgorithmSynElo    = "synelo"
	// DefaultRatingAlgorithm is used when no algorithm is configured.
	DefaultRatingAlgorithm = RatingAlgorithmTrueSkill
)

// ErrUnknownRatingAlgorithm is returned when the configured algorithm name is
// not registered.
var ErrUnknownRatingAlgorithm = errors.New("unknown rating algorithm")

// NewRatingCalculator resolves an algorithm name to its calculator. An empty
// name selects the default so existing configs without the setting keep their
// current behaviour.
func NewRatingCalculator(name string) (RatingCalculator, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", RatingAlgorithmTrueSkill:
		return TrueSkillCalculator{}, nil
	case RatingAlgorithmElo:
		return EloCalculator{}, nil
	case RatingAlgorithmSynElo:
		return SynEloCalculator{}, nil
	default:
		return nil, fmt.Errorf("%w: %q (supported: %s, %s, %s)",
			ErrUnknownRatingAlgorithm, name, RatingAlgorithmTrueSkill, RatingAlgorithmElo, RatingAlgorithmSynElo)
	}
}
