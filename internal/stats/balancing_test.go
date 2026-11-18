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

func TestAlternatingBalancer(t *testing.T) {
	t.Parallel()

	ratings := []Skill{{Mu: 100}, {Mu: 90}, {Mu: 80}, {Mu: 70}, {Mu: 60}}
	split, errSplit := (AlternatingBalancer{}).BestSplit(ratings, 0)
	require.NoError(t, errSplit)
	require.Equal(t, []int{0, 2}, split.Red)
	require.Equal(t, []int{1, 3}, split.Blu)
	require.Equal(t, []int{4}, split.Waiting)
	require.Equal(t, BalanceAlgorithmAlternating, AlternatingBalancer{}.Name())
}

func TestSerpentineBalancer(t *testing.T) {
	t.Parallel()

	ratings := []Skill{{Mu: 100}, {Mu: 90}, {Mu: 80}, {Mu: 70}, {Mu: 60}, {Mu: 50}}
	split, errSplit := (SerpentineBalancer{}).BestSplit(ratings, 0)
	require.NoError(t, errSplit)
	require.Equal(t, []int{0, 3, 4}, split.Red)
	require.Equal(t, []int{1, 2, 5}, split.Blu)
	require.Empty(t, split.Waiting)
	require.Equal(t, BalanceAlgorithmSerpentine, SerpentineBalancer{}.Name())
}

func TestNewTeamBalancer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		configured string
		wantName   string
		wantErr    bool
	}{
		{name: "empty selects default", configured: "", wantName: BalanceAlgorithmAlternating},
		{name: "alternating", configured: BalanceAlgorithmAlternating, wantName: BalanceAlgorithmAlternating},
		{name: "serpentine", configured: " SERPENTINE ", wantName: BalanceAlgorithmSerpentine},
		{name: "gosper", configured: BalanceAlgorithmGosper, wantName: BalanceAlgorithmGosper},
		{name: "unknown", configured: "random", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			balancer, errBalancer := NewTeamBalancer(test.configured)
			if test.wantErr {
				require.ErrorIs(t, errBalancer, ErrUnknownTeamBalancer)
				require.Nil(t, balancer)

				return
			}

			require.NoError(t, errBalancer)
			require.Equal(t, test.wantName, balancer.Name())
		})
	}
}

func mathAbs(value float64) float64 {
	if value < 0 {
		return -value
	}

	return value
}
