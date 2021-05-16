package service

import (
	"context"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFetchPlayerBans(t *testing.T) {
	reqIds := []steamid.SID64{
		76561198044052046,
		76561198059958958,
		76561197999702457,
		76561198189957966,
	}
	bans, err := fetchPlayerBans(context.Background(), reqIds)
	require.NoError(t, err, "HTTP error fetching player bans")
	require.Equal(t, len(bans), len(reqIds))
}
