package anticheat_test

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/anticheat"
	"github.com/leighmacdonald/gbans/internal/discord"
	personDomain "github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func TestACLogsTemplate(t *testing.T) {
	anticheat.RegisterDiscordCommands(discord.Discard{}, anticheat.AntiCheat{})

	view := struct {
		Detections map[logparse.Detection]int
		Servers    map[string]int
		Count      int
		Player     personDomain.Info
	}{
		Detections: map[logparse.Detection]int{logparse.Detection("aimbot"): 2},
		Servers:    map[string]int{"us-1": 2},
		Count:      2,
		Player:     personDomain.Core{SteamID: steamid.New(76561198000000013), Name: "Cheater"},
	}

	out, err := discord.RenderTemplate("ac_logs", view)
	require.NoError(t, err)
	require.Contains(t, out, "us-1")
	require.Contains(t, out, "Cheater")
	require.NotContains(t, out, "Failed to render")
}

func TestAnticheatTriggerIncludesPlayerAndLinks(t *testing.T) {
	anticheat.RegisterDiscordCommands(discord.Discard{}, anticheat.AntiCheat{})

	entry := logparse.StacEntry{
		SteamID:    steamid.New(76561198000000013),
		Name:       "Cheater",
		ServerName: "us-1",
		Detection:  logparse.Detection("aimbot"),
	}

	msg := anticheat.NewAnticheatTrigger("detected", anticheat.Action("ban"), entry, 3)
	var content strings.Builder
	var urls []string
	for _, component := range msg.Components {
		switch value := component.(type) {
		case discordgo.Container:
			for _, inner := range value.Components {
				if text, ok := inner.(discordgo.TextDisplay); ok {
					content.WriteString(text.Content)
				}
			}
		case discordgo.ActionsRow:
			for _, inner := range value.Components {
				if button, ok := inner.(discordgo.Button); ok {
					urls = append(urls, button.URL)
				}
			}
		}
	}

	require.Contains(t, content.String(), "Cheater")
	require.Contains(t, content.String(), "us-1")
	require.Contains(t, strings.Join(urls, " "), "/profile/76561198000000013")
	require.Contains(t, strings.Join(urls, " "), "/admin/anticheat")
}
