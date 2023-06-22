package util_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/pkg/util"

	"github.com/stretchr/testify/require"
)

func TestStringChunkDelimited(t *testing.T) {
	s := `aaaaaaaaaa
bbbbbbbbbb
cccccccccc
dddddddddd
`
	v := util.StringChunkDelimited(s, 30, "\n")
	require.Equal(t, 2, len(v))
}
