package app

import (
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/stretchr/testify/require"
	"regexp"
	"testing"
	"time"
)

func TestIsFilteredWord(t *testing.T) {
	l1 := len(wordFilters)
	importFilteredWords([]model.Filter{{WordID: 1, Pattern: regexp.MustCompile("badword"), CreatedOn: time.Now()}})
	require.Equal(t, l1+1, len(wordFilters))
	matched, matchedFilter := ContainsFilteredWord("This is a badword")
	require.True(t, matched)
	require.Equal(t, 1, matchedFilter.WordID)
}
