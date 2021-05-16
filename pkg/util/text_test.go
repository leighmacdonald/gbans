package util

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStringChunkDelimited(t *testing.T) {
	s := `aaaaaaaaaa
bbbbbbbbbb
cccccccccc
dddddddddd
`
	v := StringChunkDelimited(s, 30, "\n")
	require.Equal(t, 2, len(v))
}
