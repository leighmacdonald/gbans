package util

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIsFilteredWord(t *testing.T) {
	ImportFilteredWords([]string{"badword", "superbadword", "badword"})
	require.Equal(t, 2, len(filteredWords))
	ok, word := IsFilteredWord("This is a badword")
	require.True(t, ok)
	require.Equal(t, "badword", word)
}

func TestStringChunkDelimited(t *testing.T) {
	s := `aaaaaaaaaa
bbbbbbbbbb
cccccccccc
dddddddddd
`
	v := StringChunkDelimited(s, 30, "\n")
	require.Equal(t, 2, len(v))
}
