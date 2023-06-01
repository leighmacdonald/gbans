package thirdparty

import (
	"context"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLogsTFOverview(t *testing.T) {
	tfResult1, errTFOverview1 := LogsTFOverview(context.Background(), 76561198084134025)
	if errors.Is(errTFOverview1, context.DeadlineExceeded) {
		t.Skip("Skipping test, network unreachable.")
		return
	}
	require.NoError(t, errTFOverview1)
	require.True(t, tfResult1.Total > 100)

	tfResult2, errTFOverview2 := LogsTFOverview(context.Background(), 123456)
	if errors.Is(errTFOverview2, context.DeadlineExceeded) {
		t.Skip("Skipping test, network unreachable.")
		return
	}
	require.NoError(t, errTFOverview2)
	require.True(t, tfResult2.Total == 0)
}
