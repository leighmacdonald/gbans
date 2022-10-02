package vpksign

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSign(t *testing.T) {
	require.NoError(t, Sign(context.Background(), "./vpk_bin",
		"", ""))
}
