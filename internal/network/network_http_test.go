package network_test

import (
	"net/netip"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

func TestNetworkHTTP(t *testing.T) {
	var (
		authenticator = &tests.UserAuth{Profile: fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.Moderator)}
		networks      = network.NewNetworks(nil, network.NewRepository(fixture.Database, fixture.Persons),
			fixture.Config.Config().Network, fixture.Config.Config().GeoLocation)
		router = fixture.CreateRouter()
		server = fixture.CreateTestServer(t.Context())
	)
	network.NewHandler(router, authenticator, networks)
	for _, addr := range []string{"1.2.3.4", "5.6.7.8"} {
		require.NoError(t, networks.AddConnectionHistory(t.Context(), &network.PersonConnection{
			ServerID:        server.ServerID,
			IPAddr:          netip.MustParseAddr(addr),
			SteamID:         tests.GuestSID,
			PersonaName:     stringutil.SecureRandomString(10),
			ServerName:      server.Name,
			ServerNameShort: server.ShortName,
			CreatedOn:       time.Now(),
		}))
	}

	conns := tests.PostGOK[httphelper.LazyResult[network.PersonConnection]](t, router, "/api/connections", network.ConnectionHistoryQuery{
		Sid64: tests.GuestSID.String(),
	})
	require.Len(t, conns.Data, 2)
	for _, entry := range conns.Data {
		require.True(t, tests.GuestSID.Equal(entry.SteamID))
	}
}
