package demo

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeDemoFilename(t *testing.T) {
	t.Parallel()

	valid, err := normalizeDemoFilename("20240101-120000-koth_lakeside_f5.dem")
	require.NoError(t, err)
	require.Equal(t, "20240101-120000-koth_lakeside_f5.dem", valid)

	workshop, err := normalizeDemoFilename("20231221-042605-workshop-cp_overgrown_rc8-ugc503939302.dem")
	require.NoError(t, err)
	require.Equal(t, "20231221-042605-workshop-cp_overgrown_rc8-ugc503939302.dem", workshop)

	normalized, err := normalizeDemoFilename("manual_upload.dem")
	require.NoError(t, err)
	require.Regexp(t, `^\d{8}-\d{6}-manual_upload\.dem$`, normalized)

	upperExtension, err := normalizeDemoFilename("manual_upload.DEM")
	require.NoError(t, err)
	require.Regexp(t, `^\d{8}-\d{6}-manual_upload\.dem$`, upperExtension)

	for _, name := range []string{"", ".", "manual_upload.txt", "manual_upload", strings.Repeat("a", 252) + ".dem"} {
		_, err := normalizeDemoFilename(name)
		require.ErrorIs(t, err, ErrDemoFilename, "filename: %q", name)
	}
}
