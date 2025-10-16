package asset_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

func TestAssets(t *testing.T) {
	testFixture := tests.NewFixture()
	defer testFixture.Close()

	tempRoot := t.TempDir()
	owner := testFixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)
	data := []byte(stringutil.SecureRandomString(100))
	assetCase := asset.NewAssets(asset.NewLocalRepository(testFixture.Database, tempRoot))
	saved, errCreate := assetCase.Create(t.Context(), owner.SteamID, asset.BucketDemo, stringutil.SecureRandomString(10), bytes.NewReader(data))
	require.NoError(t, errCreate)
	require.FileExists(t, saved.LocalPath)
	contents, _ := os.ReadFile(saved.LocalPath)
	require.Equal(t, data, contents)

	fetched, reader, errFetched := assetCase.Get(t.Context(), saved.AssetID)
	require.NoError(t, errFetched)
	content2, _ := io.ReadAll(reader)
	require.Equal(t, data, content2)
	require.Equal(t, saved, fetched)

	_, errDelete := assetCase.Delete(t.Context(), saved.AssetID)
	require.NoError(t, errDelete)
	require.NoFileExists(t, saved.LocalPath)

	_, _, errFetchedNotFound := assetCase.Get(t.Context(), saved.AssetID)
	require.Error(t, errFetchedNotFound)
}
