package stats

import (
	"errors"
	"fmt"
	"strings"
)

// Supported team balancing algorithm names, as used by the balance_algorithm
// setting.
const (
	BalanceAlgorithmAlternating = "alternating"
	BalanceAlgorithmSerpentine  = "serpentine"
	BalanceAlgorithmGosper      = "gosper"
	// DefaultBalanceAlgorithm is used when no balancer is configured.
	DefaultBalanceAlgorithm = BalanceAlgorithmAlternating
)

// ErrUnknownTeamBalancer is returned when the configured balancer name is not
// registered.
var ErrUnknownTeamBalancer = errors.New("unknown team balancer")

// NewTeamBalancer resolves an algorithm name to its balancer. An empty name
// selects the default so existing configs without the setting keep their
// current behaviour.
func NewTeamBalancer(name string) (TeamBalancer, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", BalanceAlgorithmAlternating:
		return AlternatingBalancer{}, nil
	case BalanceAlgorithmSerpentine:
		return SerpentineBalancer{}, nil
	case BalanceAlgorithmGosper:
		return GosperBalancer{}, nil
	default:
		return nil, fmt.Errorf("%w: %q (supported: %s, %s, %s)",
			ErrUnknownTeamBalancer, name,
			BalanceAlgorithmAlternating, BalanceAlgorithmSerpentine, BalanceAlgorithmGosper)
	}
}
