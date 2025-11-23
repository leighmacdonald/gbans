package discord_test

import (
	"net/netip"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/ban/reason"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/sosodev/duration"
	"github.com/stretchr/testify/require"
)

func TestBind(t *testing.T) {
	type t1 struct {
		SteamID  steamid.SteamID    `id:"1"`
		CIDR     *netip.Prefix      `id:"2"`
		Reason   reason.Reason      `id:"3"`
		Duration *duration.Duration `id:"4"`
		String   string             `id:"5"`
	}

	data := map[int]string{
		1: "76561198084134027",
		2: "12.3.4.5/24",
		3: "3", // cheating
		4: "PT1H",
		5: "blah",
	}

	req, errReq := discord.BindValues[t1](t.Context(), data)
	require.NoError(t, errReq)

	prefix := netip.MustParsePrefix("12.3.4.5/24")
	require.Equal(t, t1{
		SteamID:  steamid.New(76561198084134027),
		CIDR:     &prefix,
		Reason:   reason.Cheating,
		Duration: duration.FromTimeDuration(time.Hour),
		String:   "blah",
	}, req)
}
