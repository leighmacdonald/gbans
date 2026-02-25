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
	"github.com/leighmacdonald/gbans/pkg/zstd"
	"github.com/stretchr/testify/require"
)

func TestAssets(t *testing.T) {
	testFixture := tests.NewFixture()
	defer testFixture.Close()

	tempRoot := t.TempDir()
	owner := testFixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)
	data := []byte(stringutil.SecureRandomString(100))
	assetCase := asset.NewAssets(asset.NewLocalRepository(testFixture.Database, tempRoot))
	saved, errCreate := assetCase.Create(t.Context(), owner.SteamID, asset.BucketDemo, stringutil.SecureRandomString(10), bytes.NewReader(data), false)
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

func TestAssetReader(t *testing.T) {
	for idx := range 2 {
		data := []byte(stringutil.SecureRandomString(25_000_000))
		tempFile, err := os.CreateTemp(t.TempDir(), "test.dem")
		require.NoError(t, err)
		filename := "test.dem"
		if idx == 0 {
			compressed := new(bytes.Buffer)
			require.NoError(t, zstd.Compress(bytes.NewReader(data), compressed))
			filename = "test.dem.zstd"
			_, e := tempFile.Write(compressed.Bytes())
			require.NoError(t, e)
		} else {
			_, e := tempFile.Write(data)
			require.NoError(t, e)
		}
		require.NoError(t, tempFile.Close())

		testAsset := asset.Asset{
			Name:      filename,
			Size:      int64(len(data)),
			LocalPath: tempFile.Name(),
		}

		require.Equal(t, idx == 0, testAsset.IsCompressed())

		for range 2 {
			fetchedData, err := io.ReadAll(&testAsset)
			require.NoError(t, err)
			require.Equal(t, data, fetchedData)
		}
	}
}
