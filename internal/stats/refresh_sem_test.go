//nolint:testpackage // Exercises the unexported refresh semaphore helpers.
package stats

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func newPoolWithMaxConns(t *testing.T, maxConns int32) *pgxpool.Pool {
	t.Helper()

	poolConfig, err := pgxpool.ParseConfig("postgres://user:pass@localhost:5432/db")
	require.NoError(t, err)
	poolConfig.MaxConns = maxConns

	pool, err := pgxpool.NewWithConfig(t.Context(), poolConfig)
	require.NoError(t, err)

	t.Cleanup(pool.Close)

	return pool
}

func TestViewRefreshConcurrency(t *testing.T) {
	require.Equal(t, int64(1), viewRefreshConcurrency(nil))

	tests := map[int32]int64{
		1:  1,
		2:  1,
		8:  4,
		16: 8,
	}

	for maxConns, want := range tests {
		require.Equal(t, want, viewRefreshConcurrency(newPoolWithMaxConns(t, maxConns)), "pool of %d", maxConns)
	}
}

func TestViewRefreshSlotsSerializeOverlappingCalls(t *testing.T) {
	// The semaphore is process-wide state shared with other tests in this
	// package, so reset it to make this test independent of test ordering.
	viewRefreshMu.Lock()
	viewRefreshSem = nil
	viewRefreshLimit = 0
	viewRefreshMu.Unlock()

	pool := newPoolWithMaxConns(t, 8)
	limit := viewRefreshConcurrency(pool)
	require.Positive(t, limit)

	release, err := acquireViewRefreshSlots(t.Context(), pool, int64(len(StatViews())))
	require.NoError(t, err)

	// All slots are held, so a concurrent invocation must wait rather than
	// interleaving with this one.
	blockCtx, blockCancel := context.WithTimeout(t.Context(), time.Second)
	defer blockCancel()

	_, errBlocked := acquireViewRefreshSlots(blockCtx, pool, int64(len(StatViews())))
	require.ErrorIs(t, errBlocked, blockCtx.Err())

	release()

	releaseSecond, errSecond := acquireViewRefreshSlots(t.Context(), pool, int64(len(StatViews())))
	require.NoError(t, errSecond)

	releaseSecond()
}
