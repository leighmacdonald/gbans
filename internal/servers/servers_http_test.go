package servers_test

import (
	"fmt"
	"testing"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

func TestServersHTTP(t *testing.T) {
	var (
		authenticator = &tests.UserAuth{Profile: fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)}
		serversUC, _  = servers.New(servers.NewRepository(fixture.Database), nil, "")
		router        = fixture.CreateRouter()
		server        = servers.NewServer(stringutil.SecureRandomString(5), "1.2.3.4", 27015)
	)
	servers.NewServersHandler(router, authenticator, serversUC)

	// None exist yet
	require.Empty(t, tests.GetGOK[[]servers.Server](t, router, "/api/servers"))

	// Create one
	createdServer := tests.PostGOK[servers.Server](t, router, "/api/servers", server)
	require.Equal(t, server.ShortName, createdServer.ShortName)
	require.Equal(t, server.Address, createdServer.Address)
	require.Equal(t, server.Port, createdServer.Port)

	// Make sure it exists
	require.Len(t, tests.GetGOK[[]servers.Server](t, router, "/api/servers"), 1)
	require.Len(t, tests.GetGOK[[]servers.Server](t, router, "/api/servers_admin"), 1)

	// Update it
	createdServer.EnableStats = !createdServer.EnableStats
	updated := tests.PutGOK[servers.Server](t, router, fmt.Sprintf("/api/servers/%d", createdServer.ServerID), createdServer)
	require.Equal(t, createdServer.EnableStats, updated.EnableStats)

	// Delete it
	tests.DeleteOK(t, router, fmt.Sprintf("/api/servers/%d", createdServer.ServerID), nil)

	// Fetch them
	require.Empty(t, tests.GetGOK[[]servers.Server](t, router, "/api/servers"))
	require.Empty(t, tests.GetGOK[[]servers.Server](t, router, "/api/servers_admin"))
}
