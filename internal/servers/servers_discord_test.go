package servers_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func TestFindTemplate(t *testing.T) {
	servers.RegisterDiscordCommands(discord.Discard{}, nil, nil, network.Networks{}, nil, "")

	found := []servers.FindResult{
		{
			Player: &servers.Player{Name: "Tester", SID: steamid.New(76561198000000013)},
			Server: &servers.Server{ShortName: "us-1", Address: "10.0.0.1", Port: 27015},
		},
	}

	view := struct {
		Found []servers.FindResult
	}{Found: found}

	out, err := discord.RenderTemplate("find", view)
	require.NoError(t, err)
	require.Contains(t, out, "Tester")
	require.Contains(t, out, "us-1")
	require.NotContains(t, out, "Failed to render")
}
