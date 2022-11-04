package app

import (
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
)

func TestIsFilteredWord(t *testing.T) {
	l1 := len(wordFilters)
	importFilteredWords([]model.Filter{{WordID: 1, Patterns: []*regexp.Regexp{regexp.MustCompile(".*word")}, CreatedOn: config.Now()}})
	require.Equal(t, l1+1, len(wordFilters))
	matched, matchedFilter := findFilteredWordMatch("This is a badword")
	require.Equal(t, "badword", matched)
	require.Equal(t, 1, matchedFilter.WordID)
}
