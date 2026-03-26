package stats_test

import (
	"os"
	"testing"

	"github.com/leighmacdonald/gbans/internal/json"
	"github.com/leighmacdonald/gbans/internal/maps"
	"github.com/leighmacdonald/gbans/internal/stats"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/demoparse"
	"github.com/stretchr/testify/require"
)

func TestImport(t *testing.T) {
	testFixture := tests.NewFixture()
	defer testFixture.Close()

	server := testFixture.CreateTestServer(t.Context())

	demoJSON, err := os.Open("testdata/demo-1427611.json")
	require.NoError(t, err)
	demo, errDemo := json.Decode[demoparse.Demo](demoJSON)
	require.NoError(t, errDemo)

	st := stats.New(stats.NewRepository(testFixture.Database), maps.New(maps.NewRepository(testFixture.Database)))
<<<<<<< HEAD
	demoStats, importErr := st.Import(t.Context(), server.ServerID, demo)
||||||| parent of 9bb2102c (alltime stats table)
	st := stats.New(stats.NewRepository(testFixture.Database))

	importErr := st.ImportDemo(t.Context(), demo)
=======
	_, importErr := st.ImportDemo(t.Context(), server.ServerID, demo)
>>>>>>> 9bb2102c (alltime stats table)
	require.NoError(t, importErr)
	require.Len(t, demoStats.Chat, 48)
}
