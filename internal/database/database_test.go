package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/stretchr/testify/require"
)

var fixture *tests.Fixture //nolint:gochecknoglobals

func TestMain(m *testing.M) {
	fixture = tests.NewFixture()
	defer fixture.Close()

	m.Run()
}

func TestConnectUsesDefaultMaxConns(t *testing.T) {
	db := database.New(fixture.DSN, false, false)
	require.NoError(t, db.Connect(t.Context()))
	defer db.Close()

	require.Equal(t, database.DefaultMaxConns, db.Pool().Config().MaxConns)
}

func TestConnectRespectsDSNMaxConns(t *testing.T) {
	store := database.New(fixture.DSN+"?pool_max_conns=4", false, false)
	require.NoError(t, store.Connect(t.Context()))
	defer store.Close()

	require.Equal(t, int32(4), store.Pool().Config().MaxConns)
}

func TestQueryFailsFastWhenPoolSaturated(t *testing.T) {
	store := database.New(fixture.DSN, false, false)
	require.NoError(t, store.Connect(t.Context()))
	defer store.Close()

	pool := store.Pool()

	// Exhaust the pool so new queries have no free connection to acquire.
	conns := make([]*pgxpool.Conn, pool.Config().MaxConns)
	for i := range conns {
		conn, err := pool.Acquire(t.Context())
		require.NoError(t, err)
		conns[i] = conn
	}
	defer func() {
		for _, conn := range conns {
			conn.Release()
		}
	}()

	// While the pool is saturated a query must fail (the caller's deadline
	// bounds the wait) instead of blocking indefinitely.
	queryCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	start := time.Now()
	_, err := store.Query(queryCtx, "select 1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "acquiring database connection")
	require.Less(t, time.Since(start), 5*time.Second)
}
