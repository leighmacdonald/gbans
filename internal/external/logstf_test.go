package external

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLogsTFOverview(t *testing.T) {
	tfResult1, errTFOverview1 := LogsTFOverview(76561198084134025)
	require.NoError(t, errTFOverview1)
	require.True(t, tfResult1.Total > 100)

	tfResult2, errTFOverview2 := LogsTFOverview(123456)
	require.NoError(t, errTFOverview2)
	require.True(t, tfResult2.Total == 0)
}
