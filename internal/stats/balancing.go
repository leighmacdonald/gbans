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

// GosperBalancer uses Gosper's hack to enumerate k-subsets exactly. The
// highest-rated player is anchored to one team, which halves the search space
// without discarding any optimal split.
type GosperBalancer struct{}

func (GosperBalancer) Name() string {
	return "gosper"
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
		if teamSize > available {
			continue
		}

		var subset uint = (uint(1) << teamSize) - 1
		limit := uint(1) << available
		for subset < limit {
			red := []int{order[0]}
			blu := make([]int, 0, teamSize)
			for idx := 0; idx < available; idx++ {
				if subset&(uint(1)<<idx) != 0 {
					blu = append(blu, order[idx+1])
				} else {
					red = append(red, order[idx+1])
				}
			}

			imbalance := math.Abs(splitSum(ratings, red) - splitSum(ratings, blu))
			if imbalance < best.Imbalance {
				best = TeamSplit{Red: red, Blu: blu, Imbalance: imbalance}
				if imbalance == 0 {
					return best
				}
			}

			subset = gosperNext(subset)
		}
	}

	fillWaiting(&best, order)

	return best
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
