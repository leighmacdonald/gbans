package demo_parser_test

import (
	"github.com/leighmacdonald/gbans/pkg/demo_parser"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	path, _ := filepath.Abs("../../testdata/test.dem")
	var info demo_parser.DemoInfo

	require.NoError(t, demo_parser.Parse(path, &info))
	require.Equal(t, 0, len(info.Chat))
	require.Equal(t, 0, len(info.Deaths))
	require.Equal(t, 0, len(info.Rounds))
	require.Equal(t, 4, len(info.Users))
	require.Equal(t, 25827, info.StartTick)
	require.Equal(t, 0.015, info.IntervalPerTick)
}
