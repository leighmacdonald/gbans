//nolint:testpackage // exercises the unexported status parser directly.
package servers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseStatusResponse(t *testing.T) {
	t.Parallel()

	const botsStatus = `hostname: Uncletopia | NA | One Thousand Uncles | 1
version : 11138802/24 11138802 secure
udp/ip  : ?.?.?.?:?  (public IP from Steam: 148.113.198.21)
map     : ctf_sawmill at: 0 x, 0 y, 0 z
players : 24 humans, 40 bots (64 max)
edicts  : 940 used of 2048 max
# userid name                uniqueid            connected ping loss state  adr
#    317 "Nova"              [U:1:227567390]     06:19      165    0 active 77.102.13.60:27005
#    319 "Uncle Funkle"      BOT                                     active
`

	const sourceTVOnlyStatus = `hostname: Uncletopia | LAX | 24/7 Odyssey
version : 11138802/24 11138802 secure
udp/ip  : ?.?.?.?:?  (public IP from Steam: 108.181.3.247)
map     : pl_odyssey at: 0 x, 0 y, 0 z
tags    : nocrits,nodmgspread,payload,uncletopia
sourcetv:  ?.?.?.?:?, delay 0.0s  (local: 0.0.0.0:27016)
players : 0 humans, 1 bots (33 max)
edicts  : 983 used of 2048 max
# userid name                uniqueid            connected ping loss state  adr
#      2 "Uncletopia | LAX | 24/7 Odyssey" BOT                       active
`

	const sourceTVHumansStatus = `hostname: Uncletopia | CHI | 1
sourcetv:  ?.?.?.?:?, delay 0.0s  (local: 0.0.0.0:27016)
players : 25 humans, 1 bots (33 max)
`

	const humansStatus = `hostname: Uncletopia | LAX | 24/7 Odyssey
players : 5 humans, 0 bots (24 max)
`

	testCases := []struct {
		name         string
		status       string
		humans       int
		bots         int
		playersCount int
		maxPlayers   int
	}{
		{name: "bots are players", status: botsStatus, humans: 24, bots: 40, playersCount: 64, maxPlayers: 64},
		{name: "sourcetv not counted", status: sourceTVOnlyStatus, humans: 0, bots: 0, playersCount: 0, maxPlayers: 32},
		{name: "sourcetv excluded with humans", status: sourceTVHumansStatus, humans: 25, bots: 0, playersCount: 25, maxPlayers: 32},
		{name: "humans only", status: humansStatus, humans: 5, bots: 0, playersCount: 5, maxPlayers: 24},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			status, errParse := parseStatusResponse(testCase.status)
			require.NoError(t, errParse)
			require.Equal(t, testCase.humans, status.Humans)
			require.Equal(t, testCase.bots, status.Bots)
			require.Equal(t, testCase.playersCount, status.PlayersCount)
			require.Equal(t, testCase.maxPlayers, status.PlayersMax)
		})
	}
}
