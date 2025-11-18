package asset_test

import (
	"bytes"
	"mime/multipart"
	"testing"

	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
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

func TestHTTPSaveAsset(t *testing.T) {
	var (
		auth = &tests.UserAuth{
			Profile: fixture.CreateTestPerson(t.Context(), tests.UserSID, permission.User),
		}
		router   = fixture.CreateRouter()
		assets   = asset.NewAssets(asset.NewLocalRepository(fixture.Database, t.TempDir()))
		name     = stringutil.SecureRandomString(10) + ".bin"
		contentA = []byte(stringutil.SecureRandomString(20))
	)

	asset.NewAssetHandler(router, auth, assets)

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	// create a new form-data header name data and filename data.txt
	dataPart, err := writer.CreateFormFile("file", name)
	require.NoError(t, err)
	_, errWrite := dataPart.Write(contentA)
	require.NoError(t, errWrite)
	_ = writer.WriteField("name", name)
	require.NoError(t, writer.Close())

	require.NoError(t, err)
	// saved := tests.PostCreatedForm[asset.Asset](t, router, "/api/asset", body, map[string]string{
	// 	"Content-Type": writer.FormDataContentType(),
	// })
	// require.NotEmpty(t, saved.Name)
}

func TestHTTPGetAsset(t *testing.T) {
	var (
		auth = &tests.UserAuth{
			Profile: fixture.CreateTestPerson(t.Context(), tests.UserSID, permission.User),
		}
		router   = fixture.CreateRouter()
		assets   = asset.NewAssets(asset.NewLocalRepository(fixture.Database, t.TempDir()))
		name     = stringutil.SecureRandomString(10) + ".bin"
		contentA = []byte(stringutil.SecureRandomString(20))
		contentB = []byte(stringutil.SecureRandomString(20))
	)

	asset.NewAssetHandler(router, auth, assets)

	saved, errCreate := assets.Create(t.Context(), auth.Profile.SteamID, asset.BucketMedia, name, bytes.NewReader(contentA), false)
	require.NoError(t, errCreate)

	// var fetched asset.Asset
	require.Equal(t, contentA, tests.GetOKBytes(t, router, "/asset/"+saved.AssetID.String()))

	// Create a private asset
	savedPrivate, errCreatePrivate := assets.Create(t.Context(), auth.Profile.SteamID, asset.BucketMedia, name, bytes.NewReader(contentB), true)
	require.NoError(t, errCreatePrivate)

	// Fetch it as the author
	require.Equal(t, contentB, tests.GetOKBytes(t, router, "/asset/"+savedPrivate.AssetID.String()))

	// Fetch it as a mod
	auth.Profile = fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.Moderator)
	require.Equal(t, contentB, tests.GetOKBytes(t, router, "/asset/"+savedPrivate.AssetID.String()))

	// Permission denied for non author
	auth.Profile = fixture.CreateTestPerson(t.Context(), tests.GuestSID, permission.User)
	tests.GetForbidden(t, router, "/asset/"+savedPrivate.AssetID.String())
}
