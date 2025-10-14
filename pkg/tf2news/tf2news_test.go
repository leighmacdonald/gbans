package tf2news_test

import (
	"os"
	"testing"

	"github.com/leighmacdonald/gbans/pkg/tf2news"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	testFeed, _ := os.Open("testdata/feed.xml")
	entries, err := tf2news.Parse(t.Context(), testFeed)
	require.NoError(t, err)
	require.Len(t, entries, 25)
	require.True(t, entries[0].GameUpdate)
	require.False(t, entries[1].GameUpdate)
}
