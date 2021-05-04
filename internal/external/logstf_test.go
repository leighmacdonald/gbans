package external

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLogsTFOverview(t *testing.T) {
	r1, err := LogsTFOverview(76561198084134025)
	require.NoError(t, err)
	require.True(t, r1.Total > 100)

	r2, err2 := LogsTFOverview(123456)
	require.NoError(t, err2)
	require.True(t, r2.Total == 0)
}
