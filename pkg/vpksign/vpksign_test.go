package vpksign_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/pkg/vpksign"
	"github.com/stretchr/testify/require"
)

func TestSign(t *testing.T) {
	require.NoError(t, vpksign.Sign(t.Context(), "./vpk_bin",
		"", ""))
}
