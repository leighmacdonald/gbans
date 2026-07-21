package stats_test

import (
	"os"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/json"
	"github.com/leighmacdonald/gbans/internal/maps"
	"github.com/leighmacdonald/gbans/internal/stats"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/demoparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
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

	ctx := t.Context()

	seen := map[steamid.SteamID]struct{}{}
	for _, round := range demo.Rounds {
		for _, player := range round.Players {
			sid := steamid.New(player.SteamID)
			if !sid.Valid() {
				continue
			}
			if _, ok := seen[sid]; ok {
				continue
			}
			seen[sid] = struct{}{}
			err := testFixture.Database.Exec(ctx,
				`INSERT INTO person (steam_id, created_on, updated_on, personaname, avatarhash, profilestate, personastate,
				                    realname, timecreated, loccountrycode, locstatecode, loccityid)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
				 ON CONFLICT DO NOTHING`,
				sid.Int64(), time.Now(), time.Now(), player.Name, "", 0, 0, "", 0, "", "", 0)
			require.NoError(t, err)
		}
	}

	var demoID int32
	errDemo = testFixture.Database.QueryRow(ctx,
		`INSERT INTO demo (server_id, title, map_name, created_on) VALUES ($1, $2, $3, $4) RETURNING demo_id`,
		server.ServerID, demo.Filename, demo.Map, time.Now()).Scan(&demoID)
	require.NoError(t, errDemo)

	st := stats.New(stats.NewRepository(testFixture.Database), maps.New(maps.NewRepository(testFixture.Database)))
	matchID, importErr := st.Import(ctx, server.ServerID, demoID, &demo, time.Now())
	require.NoError(t, importErr)
	require.NotNil(t, matchID)
}
