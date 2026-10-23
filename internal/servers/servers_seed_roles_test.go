package servers_test

import (
	"context"
	"testing"

	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

func TestSeedRoleIDs(t *testing.T) {
	serversCase, _ := servers.New(servers.NewRepository(fixture.Database), nil, "")

	enabledA := servers.NewServer(stringutil.SecureRandomString(10), "192.0.2.1", 27015)
	enabledA.DiscordSeedRoleIDs = []string{"role-a", "role-shared"}

	enabledB := servers.NewServer(stringutil.SecureRandomString(10), "192.0.2.2", 27015)
	enabledB.DiscordSeedRoleIDs = []string{"role-shared", "role-b"}

	disabledC := servers.NewServer(stringutil.SecureRandomString(10), "192.0.2.3", 27015)
	disabledC.IsEnabled = false
	disabledC.DiscordSeedRoleIDs = []string{"role-disabled"}

	deletedD := servers.NewServer(stringutil.SecureRandomString(10), "192.0.2.4", 27015)
	deletedD.DiscordSeedRoleIDs = []string{"role-deleted"}

	enabledA, errSave := serversCase.Save(t.Context(), enabledA)
	require.NoError(t, errSave)

	enabledB, errSave = serversCase.Save(t.Context(), enabledB)
	require.NoError(t, errSave)

	disabledC, errSave = serversCase.Save(t.Context(), disabledC)
	require.NoError(t, errSave)

	deletedD, errSave = serversCase.Save(t.Context(), deletedD)
	require.NoError(t, errSave)

	// Remove the test servers when done so the shared fixture is left as it was found.
	t.Cleanup(func() {
		for _, srv := range []servers.Server{enabledA, enabledB, disabledC} {
			if errDelete := serversCase.Delete(context.Background(), srv.ServerID); errDelete != nil {
				t.Errorf("failed to delete test server: %v", errDelete)
			}
		}
	})

	// Soft-delete one enabled server to prove deleted servers are excluded.
	require.NoError(t, serversCase.Delete(t.Context(), deletedD.ServerID))

	ids, errIDs := serversCase.SeedRoleIDs(t.Context())
	require.NoError(t, errIDs)

	// Only enabled, non-deleted servers count, and ids shared between servers are deduplicated.
	require.ElementsMatch(t, []string{"role-a", "role-shared", "role-b"}, ids)
}
