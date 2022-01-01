package steam

import (
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	config.Read()
	config.General.Mode = config.TestMode
	os.Exit(m.Run())
}

func TestFetchPlayerBans(t *testing.T) {
	reqIds := steamid.Collection{
		76561198044052046,
		76561198059958958,
		76561197999702457,
		76561198189957966,
	}
	bans, err := FetchPlayerBans(reqIds)
	require.NoError(t, err, "HTTP error fetching Player bans")
	require.Equal(t, len(bans), len(reqIds))
}

func TestSteamWebAPI(t *testing.T) {
	if config.General.SteamKey == "" {
		t.Skip("No steamkey set")
		return
	}
	friends, err := FetchFriends(76561197961279983)
	require.NoError(t, err)
	require.True(t, len(friends) > 100)
	summaries, err2 := FetchSummaries(friends)
	require.NoError(t, err2)
	require.Equal(t, len(friends), len(summaries))
}
