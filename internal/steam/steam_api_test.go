package steam

import (
	"context"
	"os"
	"testing"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	config.Read()
	config.General.Mode = config.TestMode
	os.Exit(m.Run())
}

//func TestFetchPlayerBans(t *testing.T) {
//	reqIds := steamid.Collection{
//		76561198044052046,
//		76561198059958958,
//		76561197999702457,
//		76561198189957966,
//	}
//	bans, errFetch := FetchPlayerBans(reqIds)
//	assert.NoError(t, errFetch, "HTTP error fetching Player bans")
//	assert.Equal(t, len(bans), len(reqIds))
//}

func TestSteamWebAPI(t *testing.T) {
	if config.General.SteamKey == "" {
		t.Skip("No steamkey set")
		return
	}
	friends, errFetch := FetchFriends(context.Background(), 76561197961279983)
	assert.NoError(t, errFetch)
	assert.True(t, len(friends) > 100)
	summaries, errFetchSummaries := FetchSummaries(friends)
	assert.NoError(t, errFetchSummaries)
	assert.Equal(t, len(friends), len(summaries))
}
