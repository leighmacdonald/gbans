package demoparser_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/leighmacdonald/gbans/pkg/demoparser"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	path, _ := filepath.Abs("testdata/test.dem")
	if !demoparser.Exists(path) {
		path, _ = filepath.Abs("../../testdata/test.dem")
		if !demoparser.Exists(path) {
			return
		}
	}

	var info demoparser.DemoInfo

	require.NoError(t, demoparser.Parse(context.Background(), path, &info))
	require.Equal(t, 20, len(info.Chat))
	require.Equal(t, 243, len(info.Deaths))
	require.Equal(t, 2, len(info.Rounds))
	require.Equal(t, 45, len(info.Users))
	require.Equal(t, 509, info.StartTick)
	require.Equal(t, 0.015, info.IntervalPerTick)
}
