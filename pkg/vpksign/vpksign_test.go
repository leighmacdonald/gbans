package vpksign

import (
	"context"
	"testing"

	"testing"

	"github.com/stretchr/testify/require"
)

func TestSign(t *testing.T) {
	require.NoError(t, Sign(context.Background(), "./vpk_bin",
		"", ""))
}
