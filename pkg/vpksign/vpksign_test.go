package vpksign_test

import (
	"context"
	"testing"

	"github.com/leighmacdonald/gbans/pkg/vpksign"
	"github.com/stretchr/testify/require"
)

func TestSign(t *testing.T) {
	require.NoError(t, vpksign.Sign(context.Background(), "./vpk_bin",
		"", ""))
}
