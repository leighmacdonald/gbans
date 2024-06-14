package stringutil_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

func TestStringChunkDelimited(t *testing.T) {
	s := `aaaaaaaaaa
bbbbbbbbbb
cccccccccc
dddddddddd
`
	v := stringutil.StringChunkDelimited(s, 30, "\n")
	require.Len(t, v, 2)
}
