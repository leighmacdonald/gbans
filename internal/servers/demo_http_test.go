package servers_test

import (
	"context"
	"io"
	"os"
	"path"
	"testing"

	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/fs"
	"github.com/leighmacdonald/gbans/internal/json"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/demostats"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

type MockDemoParser struct{}

// Submit implements [servers.DemoParser].
func (m MockDemoParser) Submit(ctx context.Context, name string, reader io.Reader) (*demostats.Demo, error) {
	statsPath := path.Join("../../pkg/demostats/testdata", name+".json")
	if !fs.Exists(statsPath) {
		return nil, demostats.ErrDemoSubmit
	}

	fp, err := os.Open(statsPath)
	if err != nil {
		return nil, err
	}
	defer fp.Close()

	return json.Decode[*demostats.Demo](fp)
}

func TestDemos(t *testing.T) {
	var (
		authenticator = &tests.UserAuth{Profile: fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.Moderator)}
		assets        = asset.NewAssets(asset.NewLocalRepository(fixture.Database, t.TempDir()))
		demos         = servers.NewDemos(asset.BucketDemo, servers.NewDemoRepository(fixture.Database),
			assets, fixture.Config.Config().Demo, steamid.New(fixture.Config.Config().Owner), MockDemoParser{})
		router = fixture.CreateRouter()
		server = fixture.CreateTestServer(t.Context())
	)

	asset.NewAssetHandler(router, authenticator, assets)
	servers.NewDemoHandler(router, authenticator, demos)

	// No demos
	require.Empty(t, tests.PostGOK[[]servers.DemoFile](t, router, "/api/demos", nil))

	// Create one
	demoFile, errOpen := os.Open("../../pkg/demostats/testdata/1423552-koth_ashville_final.dem")
	require.NoError(t, errOpen)
	demoAsset, errAsset := assets.Create(t.Context(), authenticator.Profile.SteamID, asset.BucketDemo, "1423552-koth_ashville_final.dem", demoFile, false)
	require.NoError(t, errAsset)
	// TODO mock demo parser
	testDemo, errCreate := demos.CreateFromAsset(t.Context(), &demoAsset, server.ServerID)
	require.NoError(t, errCreate)

	// Query it
	queried := tests.PostGOK[[]servers.DemoFile](t, router, "/api/demos", nil)
	require.Len(t, queried, 1)
	require.Equal(t, testDemo.AssetID, queried[0].AssetID)

	demoFile.Seek(0, 0)
	expected, errRead := io.ReadAll(demoFile)
	require.NoError(t, errRead)

	// Test over HTTP to ensure zstd decompression on the fly for demos
	require.Equal(t, expected, tests.GetOKBytes(t, router, "/asset/"+testDemo.AssetID.String()))
}
