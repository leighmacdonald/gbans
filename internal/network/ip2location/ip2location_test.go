package ip2location_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/network/ip2location"
	"github.com/stretchr/testify/require"
)

func TestStale(t *testing.T) {
	t.Parallel()

	files := ip2location.ImportedFiles

	t.Run("missing cache dir is stale", func(t *testing.T) {
		t.Parallel()

		require.True(t, ip2location.Stale(filepath.Join(t.TempDir(), "nope"), time.Hour, files...))
	})

	t.Run("missing file is stale", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, string(files[0])), []byte("x"), 0o600))

		require.True(t, ip2location.Stale(dir, time.Hour, files...))
	})

	t.Run("fresh files are not stale", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		for _, file := range files {
			require.NoError(t, os.WriteFile(filepath.Join(dir, string(file)), []byte("x"), 0o600))
		}

		require.False(t, ip2location.Stale(dir, time.Hour, files...))
	})

	t.Run("old file is stale", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		for _, file := range files {
			require.NoError(t, os.WriteFile(filepath.Join(dir, string(file)), []byte("x"), 0o600))
		}

		old := time.Now().Add(-time.Hour * 48)
		require.NoError(t, os.Chtimes(filepath.Join(dir, string(files[0])), old, old))

		require.True(t, ip2location.Stale(dir, time.Hour*24, files...))
	})
}
