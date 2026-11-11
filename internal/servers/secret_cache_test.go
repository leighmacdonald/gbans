//nolint:testpackage // Exercises the unexported secret cache internals.
package servers

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSecretAuthCache(t *testing.T) {
	servers := &Servers{
		secretMu:    &sync.RWMutex{},
		secretCache: make(map[int64]secretAuthEntry),
	}

	entry := secretAuthEntry{
		serverID:   7,
		serverName: "test-server",
		ip:         net.ParseIP("192.0.2.10"),
		expiresAt:  time.Now().Add(time.Minute),
	}
	servers.storeSecretAuth(1234, entry)

	cached, found := servers.cachedSecretAuth(1234)
	require.True(t, found)
	require.Equal(t, entry, cached)

	_, found = servers.cachedSecretAuth(4321)
	require.False(t, found)

	servers.invalidateSecretCache()

	_, found = servers.cachedSecretAuth(1234)
	require.False(t, found)
}

func TestSecretAuthCacheExpiry(t *testing.T) {
	servers := &Servers{
		secretMu:    &sync.RWMutex{},
		secretCache: make(map[int64]secretAuthEntry),
	}

	servers.storeSecretAuth(1234, secretAuthEntry{
		serverID:  7,
		expiresAt: time.Now().Add(-time.Second),
	})

	_, found := servers.cachedSecretAuth(1234)
	require.False(t, found)
}
