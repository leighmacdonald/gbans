package discord_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/stretchr/testify/require"
)

func TestExtURLReplacer(t *testing.T) {
	result := discord.HydrateLinks()("asdf [All Maps](/wiki/maps_all) asdf\n[All Maps](https://example.com/wiki/maps_all)")
	require.Equal(t, "asdf [All Maps](http://localhost:6006/wiki/maps_all) asdf\n[All Maps](https://example.com/wiki/maps_all)", result)
}
