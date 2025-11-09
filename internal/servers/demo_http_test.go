package servers_test

import (
	"os"
	"testing"

	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func TestDemos(t *testing.T) {
	var (
		authenticator = &tests.StaticAuth{Profile: fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.Moderator)}
		assets        = asset.NewAssets(asset.NewLocalRepository(fixture.Database, t.TempDir()))
		demos         = servers.NewDemos(asset.BucketDemo, servers.NewDemoRepository(fixture.Database),
			assets, fixture.Config.Config().Demo, steamid.New(fixture.Config.Config().Owner))
		router = fixture.CreateRouter()
		// server = fixture.CreateTestServer(t.Context())
	)

	servers.NewDemoHandler(router, demos, authenticator)

	// No demos
	require.Empty(t, tests.PostGOK[[]servers.DemoFile](t, router, "/api/demos", nil))

	// Create one
	demoFile, errOpen := os.Open("testdata/test.dem")
	require.NoError(t, errOpen)
	_, errAsset := assets.Create(t.Context(), authenticator.Profile.SteamID, asset.BucketDemo, "20231112-063943-koth_harvest_final.dem", demoFile, false)
	require.NoError(t, errAsset)
	// TODO mock demo parser
	// testDemo, errCreate := demos.CreateFromAsset(t.Context(), testAsset, server.ServerID)
	// require.NoError(t, errCreate)

	// // Query it
	// queried := tests.PostGOK[[]servers.DemoFile](t, router, "/api/demos", nil)
	// require.Len(t, queried, 1)
	// require.Equal(t, testDemo.AssetID, queried[0].AssetID)
}
