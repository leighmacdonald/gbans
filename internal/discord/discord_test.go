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

func TestHydrateLinksHandlesImages(t *testing.T) {
	result := discord.HydrateLinks()("![](/asset/abc) ![alt text](/asset/def) [link](/wiki/Test)")
	require.Equal(t,
		"![](http://localhost:6006/asset/abc) "+
			"![alt text](http://localhost:6006/asset/def) "+
			"[link](http://localhost:6006/wiki/Test)", result)
}
