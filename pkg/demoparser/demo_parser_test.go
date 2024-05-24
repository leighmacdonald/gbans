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
	require.Len(t, info.Chat, 20)
	require.Len(t, info.Deaths, 243)
	require.Len(t, info.Rounds, 2)
	require.Len(t, info.Users, 45)
	require.Equal(t, 509, info.StartTick)
	require.InEpsilon(t, 0.015, info.IntervalPerTick, 0.001)
}
