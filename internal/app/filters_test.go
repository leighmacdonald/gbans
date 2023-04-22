package app

import (
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIsFilteredWord(t *testing.T) {
	l1 := len(wordFilters)
	f1 := model.Filter{FilterID: 1, Pattern: ".*word", IsRegex: true, CreatedOn: config.Now()}
	f1.Init()
	importFilteredWords([]model.Filter{f1})
	require.Equal(t, l1+1, len(wordFilters))
	matched, matchedFilter := findFilteredWordMatch("This is a badword")
	require.Equal(t, "badword", matched)
	require.Equal(t, int64(1), matchedFilter.FilterID)
}
