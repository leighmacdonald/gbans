//nolint:testpackage // white-box tests for the team balancing solver.
package stats

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGosperBestSplit(t *testing.T) {
	t.Parallel()

	balancer := GosperBalancer{}
	ratings := []Skill{
		{Mu: 1600},
		{Mu: 1500},
		{Mu: 1400},
		{Mu: 1300},
		{Mu: 1200},
		{Mu: 1100},
		{Mu: 1000},
		{Mu: 900},
	}

	split, errSplit := balancer.BestSplit(ratings, 32)
	require.NoError(t, errSplit)
	require.Len(t, split.Red, 4)
	require.Len(t, split.Blu, 4)
	require.Empty(t, split.Waiting)
	require.Contains(t, split.Red, 0)
	require.LessOrEqual(t, split.Imbalance, 100.0)
	require.InDelta(t,
		mathAbs(splitSum(ratings, split.Red)-splitSum(ratings, split.Blu)),
		split.Imbalance, 1e-9)
}

func TestGosperBestSplitPerfectBalance(t *testing.T) {
	t.Parallel()

	ratings := []Skill{{Mu: 4}, {Mu: 3}, {Mu: 2}, {Mu: 1}}
	split, errSplit := (GosperBalancer{}).BestSplit(ratings, 32)
	require.NoError(t, errSplit)
	require.Zero(t, split.Imbalance)
	require.ElementsMatch(t, []int{0, 3}, split.Red)
	require.ElementsMatch(t, []int{1, 2}, split.Blu)
}

func TestGosperGreedyFallback(t *testing.T) {
	t.Parallel()

	ratings := make([]Skill, 20)
	for idx := range ratings {
		ratings[idx] = Skill{Mu: float64(2000 - idx*10)}
	}

	split, errSplit := (GosperBalancer{}).BestSplit(ratings, 8)
	require.NoError(t, errSplit)
	require.Len(t, split.Red, 10)
	require.Len(t, split.Blu, 10)
	require.Less(t, split.Imbalance, 100.0)
}

func TestGosperOddPool(t *testing.T) {
	t.Parallel()

	ratings := []Skill{{Mu: 5}, {Mu: 4}, {Mu: 3}, {Mu: 2}, {Mu: 1}}
	split, errSplit := (GosperBalancer{}).BestSplit(ratings, 32)
	require.NoError(t, errSplit)
	require.Len(t, split.Red, 2)
	require.Len(t, split.Blu, 2)
	require.Len(t, split.Waiting, 1)
}

func TestGosperSmallPool(t *testing.T) {
	t.Parallel()

	split, errSplit := (GosperBalancer{}).BestSplit([]Skill{{Mu: 1}, {Mu: 2}}, 32)
	require.NoError(t, errSplit)
	require.Len(t, split.Waiting, 2)
}

func TestGosperNext(t *testing.T) {
	t.Parallel()

	// Three-bit numbers with two bits set, in ascending order.
	seen := []uint{0b011, 0b101, 0b110}
	value := uint(0b011)
	for _, want := range seen {
		require.Equal(t, want, value)
		value = gosperNext(value)
	}
	require.GreaterOrEqual(t, value, uint(0b1000))
}

func mathAbs(value float64) float64 {
	if value < 0 {
		return -value
	}

	return value
}
