package chat_test

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/gbans/internal/discord"
	personDomain "github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func TestWarningFlaggedMessageIncludesStatus(t *testing.T) {
	chat.RegisterDiscordCommands(discord.Discard{}, chat.WordFilters{})

	warn := chat.UserWarning{
		SteamID:       76561198000000012,
		Personaname:   "Cheater",
		Matched:       "badword",
		MatchedFilter: chat.Filter{FilterID: 7, Pattern: "badword"},
	}
	warning := chat.NewUserWarning{
		UserWarning: warn,
		UserMessage: chat.Message{ServerName: "us-1"},
	}
	status := chat.WarningStatus{
		WarningCount:   3,
		CurrentWeight:  9,
		MaxWeight:      5,
		WindowResetsIn: "45s",
	}

	message := chat.WarningFlaggedMessage(warning, status)
	var parts []string
	for _, component := range message.Components {
		if container, ok := component.(discordgo.Container); ok {
			for _, inner := range container.Components {
				if text, ok := inner.(discordgo.TextDisplay); ok {
					parts = append(parts, text.Content)
				}
			}
		}
	}
	content := strings.Join(parts, "\n")

	require.Contains(t, content, "Cheater")
	require.Contains(t, content, "us-1")
	require.Contains(t, content, "Current Warnings: **3**")
	require.Contains(t, content, "Weight: **9/5**")
	require.Contains(t, content, "45s")
	require.NotContains(t, content, "Failed to render")
}

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
