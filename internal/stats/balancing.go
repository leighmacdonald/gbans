package stats

import (
	"errors"
	"fmt"
	"math"
	"math/bits"
)

// ErrTooManyPlayers is returned when an exact split is requested for a pool
// larger than the bitmask solver can represent.
var ErrTooManyPlayers = errors.New("player pool too large for exact split")

// TeamSplit is an assignment of pool indices (sorted by rating, strongest
// first) to two teams. Waiting holds players left out of an odd-sized pool.
type TeamSplit struct {
	Red     []int
	Blu     []int
	Waiting []int
	// Imbalance is the absolute difference between the teams' rating sums.
	Imbalance float64
}

// TeamBalancer produces balanced team assignments from player ratings.
type TeamBalancer interface {
	Name() string
	// BestSplit returns the best k-versus-k split for every supported team size
	// and keeps the best overall result. maxExact caps the exact solver; larger
	// pools use the greedy swap fallback.
	BestSplit(ratings []Skill, maxExact int) (TeamSplit, error)
}

// AlternatingBalancer is the default balancer. It sorts players by rating and
// assigns them red/blue/red/blue in descending order. It is fast and
// predictable; an odd last player is left in Waiting for the service to place.
type AlternatingBalancer struct{}

func (AlternatingBalancer) Name() string {
	return BalanceAlgorithmAlternating
}

func (AlternatingBalancer) BestSplit(ratings []Skill, _ int) (TeamSplit, error) {
	order := rankedIndices(ratings)
	split := TeamSplit{Red: make([]int, 0, (len(order)+1)/2), Blu: make([]int, 0, len(order)/2)}

	for idx, player := range order {
		if idx%2 == 0 {
			split.Red = append(split.Red, player)
		} else {
			split.Blu = append(split.Blu, player)
		}
	}

	// An unpaired last player is benched so the two teams stay equal; the
	// service reassigns them to the smaller team before returning results.
	if len(split.Red) > len(split.Blu) {
		split.Waiting = split.Red[len(split.Red)-1:]
		split.Red = split.Red[:len(split.Red)-1]
	}

	split.Imbalance = math.Abs(splitSum(ratings, split.Red) - splitSum(ratings, split.Blu))

	return split, nil
}

// SerpentineBalancer assigns players in a snake draft (red/blue/blue/red)
// after sorting by rating. It evens out the teams more than simple alternating
// while staying O(n log n).
type SerpentineBalancer struct{}

func (SerpentineBalancer) Name() string {
	return BalanceAlgorithmSerpentine
}

func (SerpentineBalancer) BestSplit(ratings []Skill, _ int) (TeamSplit, error) {
	order := rankedIndices(ratings)
	split := TeamSplit{Red: make([]int, 0, (len(order)+1)/2), Blu: make([]int, 0, len(order)/2)}

	for idx, player := range order {
		switch idx % 4 {
		case 0, 3:
			split.Red = append(split.Red, player)
		default:
			split.Blu = append(split.Blu, player)
		}
	}

	if len(split.Red) > len(split.Blu) {
		split.Waiting = split.Red[len(split.Red)-1:]
		split.Red = split.Red[:len(split.Red)-1]
	}

	split.Imbalance = math.Abs(splitSum(ratings, split.Red) - splitSum(ratings, split.Blu))

	return split, nil
}

// GosperBalancer uses Gosper's hack to enumerate k-subsets exactly. The
// highest-rated player is anchored to one team, which halves the search space
// without discarding any optimal split.
type GosperBalancer struct{}

func (GosperBalancer) Name() string {
	return BalanceAlgorithmGosper
}

func (GosperBalancer) BestSplit(ratings []Skill, maxExact int) (TeamSplit, error) {
	order := rankedIndices(ratings)
	if len(order) < 4 {
		return TeamSplit{Waiting: order}, nil
	}

	if maxExact > 0 && len(order) <= maxExact {
		if len(order) > bits.UintSize-1 {
			return TeamSplit{}, fmt.Errorf("%w: %d players", ErrTooManyPlayers, len(order))
		}

		return gosperBestSplit(ratings, order), nil
	}

	return greedyBestSplit(ratings, order, max(len(order)/2, 2)), nil
}

func rankedIndices(ratings []Skill) []int {
	order := make([]int, len(ratings))
	for idx := range order {
		order[idx] = idx
	}

	sortByRating(order, ratings)

	return order
}

func sortByRating(order []int, ratings []Skill) {
	for i := 1; i < len(order); i++ {
		for j := i; j > 0 && ratings[order[j]].Mu > ratings[order[j-1]].Mu; j-- {
			order[j], order[j-1] = order[j-1], order[j]
		}
	}
}

func gosperBestSplit(ratings []Skill, order []int) TeamSplit {
	available := len(order) - 1
	best := TeamSplit{Imbalance: math.Inf(1)}

	for teamSize := len(order) / 2; teamSize >= 2; teamSize-- {
		if 2*teamSize-1 > available {
			continue
		}

		outer := (uint(1) << (2*teamSize - 1)) - 1
		outerLimit := uint(1) << available
		for outer < outerLimit {
			active := setBitIndices(outer, 1)

			inner := (uint(1) << teamSize) - 1
			innerLimit := uint(1) << len(active)
			for inner < innerLimit {
				red := []int{order[0]}
				blu := make([]int, 0, teamSize)
				for idx, player := range active {
					if inner&(uint(1)<<idx) != 0 {
						blu = append(blu, player)
					} else {
						red = append(red, player)
					}
				}

				imbalance := math.Abs(splitSum(ratings, red) - splitSum(ratings, blu))
				if imbalance < best.Imbalance {
					best = TeamSplit{Red: red, Blu: blu, Imbalance: imbalance}
					if imbalance == 0 {
						return bestWithWaiting(best, order)
					}
				}

				inner = gosperNext(inner)
			}

			outer = gosperNext(outer)
		}
	}

	return bestWithWaiting(best, order)
}

func bestWithWaiting(split TeamSplit, order []int) TeamSplit {
	fillWaiting(&split, order)

	return split
}

// setBitIndices returns the sorted indices of the set bits of value, offset by
// start. The anchored player (index zero) is excluded from the enumeration.
func setBitIndices(value uint, start int) []int {
	var indices []int
	for idx := range bits.UintSize {
		if value&(uint(1)<<idx) != 0 {
			indices = append(indices, idx+start)
		}
	}

	return indices
}

// gosperNext advances a bitmask to the next k-subset of the same cardinality.
func gosperNext(value uint) uint {
	lowest := value & -value
	ripple := value + lowest

	return ((ripple^value)/lowest)>>2 | ripple
}

func greedyBestSplit(ratings []Skill, order []int, teamSize int) TeamSplit {
	red := append([]int(nil), order[:teamSize]...)
	blu := append([]int(nil), order[teamSize:teamSize*2]...)
	waiting := append([]int(nil), order[teamSize*2:]...)

	best := TeamSplit{
		Red:       red,
		Blu:       blu,
		Waiting:   waiting,
		Imbalance: math.Abs(splitSum(ratings, red) - splitSum(ratings, blu)),
	}

	for improved := true; improved; {
		improved = false
		for redIdx := range red {
			for bluIdx := range blu {
				red[redIdx], blu[bluIdx] = blu[bluIdx], red[redIdx]
				imbalance := math.Abs(splitSum(ratings, red) - splitSum(ratings, blu))
				if imbalance < best.Imbalance {
					best = TeamSplit{
						Red:       append([]int(nil), red...),
						Blu:       append([]int(nil), blu...),
						Waiting:   waiting,
						Imbalance: imbalance,
					}
					improved = true
				} else {
					red[redIdx], blu[bluIdx] = blu[bluIdx], red[redIdx]
				}
			}
		}
	}

	return best
}

func splitSum(ratings []Skill, indices []int) float64 {
	var total float64
	for _, idx := range indices {
		total += ratings[idx].Mu
	}

	return total
}

func fillWaiting(split *TeamSplit, order []int) {
	used := make(map[int]bool, len(split.Red)+len(split.Blu))
	for _, idx := range split.Red {
		used[idx] = true
	}
	for _, idx := range split.Blu {
		used[idx] = true
	}

	for _, idx := range order {
		if !used[idx] {
			split.Waiting = append(split.Waiting, idx)
		}
	}
}
