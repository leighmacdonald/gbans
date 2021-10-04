package depotdownloader

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"
)

func TestDepotDownloader(t *testing.T) {
	cd := "."
	if os.Getenv("TEST_DEPOT_DIR") != "" {
		cd = os.Getenv("TEST_DEPOT_DIR")
	}
	v, ok := os.LookupEnv("DEPOT_DOWNLOADER")
	if !ok || v == "" {
		t.Skipf("Skipped depot test, DEPOT_DOWNLOADER not set")
		return
	}
	o := path.Join(cd, fmt.Sprintf("%d", tf2server))
	d, errDL := New(tf2server, o)
	require.NoError(t, errDL)
	require.NoError(t, d.Start())
}
