package app

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAddWarning(t *testing.T) {
	addWarning(76561197961279983, warnLanguage)
	addWarning(76561197961279983, warnLanguage)
	require.True(t, len(warnings[76561197961279983]) == 2)
}
