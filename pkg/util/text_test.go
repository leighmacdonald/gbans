package util

import (
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

func TestIsFilteredWord(t *testing.T) {
	l1 := len(wordFilters)
	ImportFilteredWords([]*model.Filter{{1, regexp.MustCompile("badword"), time.Now()}})
	require.Equal(t, l1+1, len(wordFilters))
	matched, matchedFilter := IsFilteredWord("This is a badword")
	require.True(t, matched)
	require.Equal(t, 1, matchedFilter.WordID)
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
