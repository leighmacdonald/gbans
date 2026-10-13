//nolint:testpackage // exercises unexported seed helpers directly.
package sourcemod

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveSeedChannelID(t *testing.T) {
	const (
		server = "111111111111111111"
		global = "222222222222222222"
		public = "333333333333333333"
	)

	// Server channel wins when valid.
	require.Equal(t, server, resolveSeedChannelID(server, global, public))
	// Invalid server channel falls back to global.
	require.Equal(t, global, resolveSeedChannelID("", global, public))
	require.Equal(t, global, resolveSeedChannelID("not-a-channel", global, public))
	require.Equal(t, global, resolveSeedChannelID("<#111>", global, public))
	// Invalid global falls back to public log channel.
	require.Equal(t, public, resolveSeedChannelID("", "", public))
	require.Equal(t, public, resolveSeedChannelID("junk", "also-junk", public))
}

func TestSanitizeSeedMessage(t *testing.T) {
	require.Empty(t, sanitizeSeedMessage(""))
	require.Equal(t, "please seed", sanitizeSeedMessage("please seed"))

	// Mention syntax must not survive: no pings via user text.
	out := sanitizeSeedMessage("@everyone come seed @here <@123> <@&456> <#789>")
	require.NotContains(t, out, "@everyone")
	require.NotContains(t, out, "@here")
	require.NotContains(t, out, "<@123>")
	require.NotContains(t, out, "<@&456>")
	require.Contains(t, out, "come seed")

	// Overlong messages are capped.
	long := strings.Repeat("a", maxSeedMessageLength+100)
	capped := sanitizeSeedMessage(long)
	require.LessOrEqual(t, len([]rune(capped)), maxSeedMessageLength+1)
	require.True(t, strings.HasSuffix(capped, "…"))
	require.Len(t, []rune(strings.TrimSuffix(capped, "…")), maxSeedMessageLength)
}
