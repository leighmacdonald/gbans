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
	importFilteredWords([]model.Filter{{WordID: 1, Pattern: regexp.MustCompile("badword"), CreatedOn: config.Now()}})
	require.Equal(t, l1+1, len(wordFilters))
	matched, matchedFilter := ContainsFilteredWord("This is a badword")
	require.True(t, matched)
	require.Equal(t, 1, matchedFilter.WordID)
}
