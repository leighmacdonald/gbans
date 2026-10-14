//nolint:testpackage // Needs access to the unexported update guard.
package network

import (
	"context"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/network/ip2location"
	"github.com/stretchr/testify/require"
)

func TestRefreshLocationData_AlreadyUpdating(t *testing.T) {
	t.Parallel()

	networks := NewNetworks(nil, Repository{}, &Config{}, &ip2location.Config{
		Enabled:   true,
		CachePath: t.TempDir(),
		Token:     "token",
	})
	networks.updating.Store(true)

	err := networks.RefreshLocationData(context.Background())

	require.ErrorIs(t, err, ErrUpdateInProgress)
	// The in progress flag belongs to the other updater, not us.
	require.True(t, networks.updating.Load())
}

func TestStartGeoUpdater_ReturnsWhenDisabled(t *testing.T) {
	t.Parallel()

	tests := map[string]func(cachePath string) *ip2location.Config{
		"nil config": func(_ string) *ip2location.Config {
			return nil
		},
		"disabled": func(cachePath string) *ip2location.Config {
			return &ip2location.Config{Enabled: false, CachePath: cachePath, Token: "token", UpdateIntervalDays: 30}
		},
		"missing token": func(cachePath string) *ip2location.Config {
			return &ip2location.Config{Enabled: true, CachePath: cachePath, UpdateIntervalDays: 30}
		},
		"zero interval": func(cachePath string) *ip2location.Config {
			return &ip2location.Config{Enabled: true, CachePath: cachePath, Token: "token"}
		},
		"negative interval": func(cachePath string) *ip2location.Config {
			return &ip2location.Config{Enabled: true, CachePath: cachePath, Token: "token", UpdateIntervalDays: -1}
		},
	}

	for name, newConf := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			networks := NewNetworks(nil, Repository{}, &Config{}, newConf(t.TempDir()))

			done := make(chan struct{})
			go func() {
				networks.StartGeoUpdater(context.Background())
				close(done)
			}()

			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("StartGeoUpdater did not return for disabled config")
			}
		})
	}
}
