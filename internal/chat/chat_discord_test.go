package chat_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/gbans/internal/discord"
	personDomain "github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func TestChatMessageTemplate(t *testing.T) {
	chat.RegisterDiscordCommands(discord.Discard{}, chat.WordFilters{})

	view := map[string]any{
		"Msg": chat.Message{
			ServerName:  "us-1",
			PersonaName: "Cheater",
			Body:        "hello",
			Team:        true,
		},
		"Player": personDomain.Core{SteamID: steamid.New(76561198000000012), Name: "Cheater"},
	}

	out, err := discord.RenderTemplate("chat_message", view)
	require.NoError(t, err)
	require.Contains(t, out, "(Team)")
	require.Contains(t, out, "hello")
	require.NotContains(t, out, "Failed to render")
}
