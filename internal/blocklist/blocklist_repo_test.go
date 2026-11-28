package blocklist_test

import (
	"context"
	"net/netip"
	"testing"

	"github.com/leighmacdonald/gbans/internal/blocklist"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/stretchr/testify/require"
)

var fixture *tests.Fixture //nolint:gochecknoglobals

func TestMain(m *testing.M) {
	fixture = tests.NewFixture()
	defer fixture.Close()

	m.Run()
}

func TestInsertCachePropagatesError(t *testing.T) {
	repo := blocklist.NewRepository(fixture.Database)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	prefix := netip.MustParsePrefix("127.0.0.0/8")
	err := repo.InsertCache(ctx, blocklist.CIDRBlockSource{CIDRBlockSourceID: 1}, []netip.Prefix{prefix})
	require.ErrorIs(t, err, context.Canceled)
}
