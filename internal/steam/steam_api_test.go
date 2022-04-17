package steam

import (
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/stretchr/testify/assert"
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
	assert.NoError(t, err, "HTTP error fetching Player bans")
	assert.Equal(t, len(bans), len(reqIds))
}

func TestSteamWebAPI(t *testing.T) {
	if config.General.SteamKey == "" {
		t.Skip("No steamkey set")
		return
	}
	friends, err := FetchFriends(76561197961279983)
	assert.NoError(t, err)
	assert.True(t, len(friends) > 100)
	summaries, err2 := FetchSummaries(friends)
	assert.NoError(t, err2)
	assert.Equal(t, len(friends), len(summaries))
}
