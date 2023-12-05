package demoparser_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/leighmacdonald/gbans/pkg/demoparser"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	path, _ := filepath.Abs("../../testdata/test.dem")

	var info demoparser.DemoInfo

	require.NoError(t, demoparser.Parse(context.Background(), path, &info))
	require.Equal(t, 0, len(info.Chat))
	require.Equal(t, 0, len(info.Deaths))
	require.Equal(t, 0, len(info.Rounds))
	require.Equal(t, 4, len(info.Users))
	require.Equal(t, 25827, info.StartTick)
	require.Equal(t, 0.015, info.IntervalPerTick)
}
