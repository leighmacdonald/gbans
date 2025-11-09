package network_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

var fixture *tests.Fixture //nolint:gochecknoglobals

func TestMain(m *testing.M) {
	fixture = tests.NewFixture()
	defer fixture.Close()

	m.Run()
}

type dummyCache struct{}

func (d dummyCache) UpdateCache(_ context.Context) error {
	return nil
}

func TestSources(t *testing.T) {
	var (
		authenticator = &tests.StaticAuth{Profile: fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.Moderator)}
		blocklists    = network.NewBlocklists(network.NewBlocklistRepository(fixture.Database), dummyCache{})
		networks      = network.NewNetworks(nil, network.NewRepository(fixture.Database, fixture.Persons),
			fixture.Config.Config().Network, fixture.Config.Config().GeoLocation)
		router = fixture.CreateRouter()
	)
	network.NewBlocklistHandler(router, blocklists, networks, authenticator)

	// No results
	require.Empty(t, tests.GetGOK[[]network.CIDRBlockSource](t, router, "/api/block_list/sources"))

	// Permission denied to create as a non-admin
	req := network.BlocklistCreateRequest{
		Name:    stringutil.SecureRandomString(10),
		URL:     "https://raw.githubusercontent.com/X4BNet/lists_vpn/main/output/datacenter/ipv4.txt",
		Enabled: true,
	}
	tests.PostForbidden(t, router, "/api/block_list/sources", req)

	// Retry as a admin
	authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)
	list := tests.PostGCreated[network.CIDRBlockSource](t, router, "/api/block_list/sources", req)
	require.Positive(t, list.CIDRBlockSourceID)

	// Update it
	list.Enabled = false
	tests.PostGOK[network.CIDRBlockSource](t, router, fmt.Sprintf("/api/block_list/sources/%d", list.CIDRBlockSourceID), list)

	// Fetch updated list
	updatedLists := tests.GetGOK[[]network.CIDRBlockSource](t, router, "/api/block_list/sources")
	require.Len(t, updatedLists, 1)

	// Delete it
	tests.DeleteOK(t, router, fmt.Sprintf("/api/block_list/sources/%d", updatedLists[0].CIDRBlockSourceID), nil)

	// Confirm delete
	require.Empty(t, tests.GetGOK[[]network.CIDRBlockSource](t, router, "/api/block_list/sources"))
}

func TestWhitelistIP(t *testing.T) {
	var (
		authenticator = &tests.StaticAuth{Profile: fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.Moderator)}
		blocklists    = network.NewBlocklists(network.NewBlocklistRepository(fixture.Database), dummyCache{})
		networks      = network.NewNetworks(nil, network.NewRepository(fixture.Database, fixture.Persons),
			fixture.Config.Config().Network, fixture.Config.Config().GeoLocation)
		router = fixture.CreateRouter()
	)
	network.NewBlocklistHandler(router, blocklists, networks, authenticator)

	require.Empty(t, tests.GetGOK[[]network.CIDRBlockWhitelistExport](t, router, "/api/block_list/whitelist/ip"))

	req := network.CreateWhitelistIPRequest{Address: "1.2.3.4"}
	ipBlock := tests.PostGCreated[network.CIDRBlockWhitelistExport](t, router, "/api/block_list/whitelist/ip", req)
	require.Equal(t, req.Address+"/32", ipBlock.Address)

	fetched := tests.GetGOK[[]network.CIDRBlockWhitelistExport](t, router, "/api/block_list/whitelist/ip")
	require.Len(t, fetched, 1)

	updateReq := network.UpdateWhitelistIPRequest{Address: "2.3.4.5"}
	updated := tests.PostGOK[network.WhitelistIP](t, router, fmt.Sprintf("/api/block_list/whitelist/ip/%d", fetched[0].CIDRBlockWhitelistID), updateReq)
	require.Equal(t, updateReq.Address+"/32", updated.Address.String())

	tests.DeleteOK(t, router, fmt.Sprintf("/api/block_list/whitelist/ip/%d", fetched[0].CIDRBlockWhitelistID), nil)

	require.Empty(t, tests.GetGOK[[]network.CIDRBlockWhitelistExport](t, router, "/api/block_list/whitelist/ip"))
}

func TestWhitelistSteam(t *testing.T) {
	var (
		authenticator = &tests.StaticAuth{Profile: fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.Moderator)}
		router        = fixture.CreateRouter()
		guest         = fixture.CreateTestPerson(t.Context(), tests.GuestSID, permission.User)
	)
	network.NewBlocklistHandler(router,
		network.NewBlocklists(network.NewBlocklistRepository(fixture.Database), dummyCache{}),
		network.NewNetworks(nil, network.NewRepository(fixture.Database, fixture.Persons),
			fixture.Config.Config().Network, fixture.Config.Config().GeoLocation),
		authenticator)

	// Get empty set
	require.Empty(t, tests.GetGOK[[]network.WhitelistSteam](t, router, "/api/block_list/whitelist/steam"))

	// Create a entry
	req := network.CreateSteamWhitelistRequest{SteamIDField: httphelper.SteamIDField{SteamIDValue: guest.SteamID.String()}}
	steamBlock := tests.PostGOK[network.WhitelistSteam](t, router, "/api/block_list/whitelist/steam", req)
	require.Equal(t, req.SteamIDField, steamBlock.SteamIDField)

	// Fetch new entries
	require.Len(t, tests.GetGOK[[]network.WhitelistSteam](t, router, "/api/block_list/whitelist/steam"), 1)

	// Delete it
	tests.DeleteOK(t, router, fmt.Sprintf("/api/block_list/whitelist/steam/%d", guest.SteamID.Int64()), nil)

	// Confirm delete
	require.Empty(t, tests.GetGOK[[]network.WhitelistSteam](t, router, "/api/block_list/whitelist/steam"))
}
