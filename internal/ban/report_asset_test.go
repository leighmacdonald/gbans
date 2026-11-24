//nolint:testpackage // exercises unexported Discord message builders directly.
package ban

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func discordMessageText(t *testing.T, message *discordgo.MessageSend) string {
	t.Helper()

	var parts []string
	var walk func([]discordgo.MessageComponent)

	walk = func(components []discordgo.MessageComponent) {
		for _, component := range components {
			switch typed := component.(type) {
			case discordgo.TextDisplay:
				parts = append(parts, typed.Content)
			case discordgo.Container:
				walk(typed.Components)
			case discordgo.Section:
				walk(typed.Components)
			}
		}
	}

	walk(message.Components)

	return strings.Join(parts, "\n")
}

func TestReportMessageHydratesAssetLinks(t *testing.T) {
	link.SetBaseURL("http://localhost:6006")
	t.Cleanup(func() { link.SetBaseURL("") })

	RegisterDiscordCommands(discord.Discard{}, Bans{}, nil, nil)

	base := Report{
		ReportID:    1,
		ReasonText:  "see attached ![](/asset/5c4d0c19-66c6-4700-a0c9-e9eb72d638f)",
		Description: "repeated abuse ![](/asset/6f8aea56-25ad-4cab-8676-8f4fc6369f3b)",
	}
	report := ReportWithAuthor{
		Report:  base,
		Author:  person.Core{SteamID: steamid.New(76561198031303282), Name: "Author"},
		Subject: person.Core{SteamID: steamid.New(76561198166136463), Name: "Target"},
	}

	content := discordMessageText(t, newInGameReportResponse(report))
	require.Contains(t, content, "![](http://localhost:6006/asset/5c4d0c19-66c6-4700-a0c9-e9eb72d638f)")
	require.Contains(t, content, "![](http://localhost:6006/asset/6f8aea56-25ad-4cab-8676-8f4fc6369f3b)")
	require.NotContains(t, content, "](/asset/") // no relative asset links remain
}
