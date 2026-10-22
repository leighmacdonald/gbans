package stats_test

import (
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/stats"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

type captureNotifier struct {
	payloads []notification.Payload
}

func (c *captureNotifier) Send(payload notification.Payload) {
	c.payloads = append(c.payloads, payload)
}

func TestMatchMessage(t *testing.T) {
	stats.RegisterDiscordCommands(discord.Discard{})

	match := testMatchDiscordMatch(t)
	message := stats.MatchMessage(match)
	content := matchMessageText(t, message)

	require.Contains(t, content, "cp_process_f7")
	require.Contains(t, content, "RED 3")
	require.Contains(t, content, "2 BLU")
	require.Contains(t, content, "RED wins")
	require.Contains(t, content, "gbans-1")
	require.Contains(t, content, "24m12s")
	require.Contains(t, content, "Match ID: `"+match.MatchID.String()+"`")

	require.Contains(t, content, "[Red One]("+link.Raw("/profile/76561198000000001")+")")
	require.Contains(t, content, "[Red Two]("+link.Raw("/profile/76561198000000002")+")")
	require.Contains(t, content, "[Red Three]("+link.Raw("/profile/76561198000000003")+")")
	require.NotContains(t, content, "Red Four")
	require.NotContains(t, content, "Zero Round")

	require.Contains(t, content, "[Blue One]("+link.Raw("/profile/76561198000000005")+")")
	require.Contains(t, content, "[Blue Two]("+link.Raw("/profile/76561198000000006")+")")
	require.Contains(t, content, "[Blue Three]("+link.Raw("/profile/76561198000000007")+")")
	require.NotContains(t, content, "Blue Four")

	require.Contains(t, content, "**100 pts**")
	require.Contains(t, content, "10 K")

	buttons := matchMessageButtons(t, message)
	require.Len(t, buttons, 2)
	require.Equal(t, "View Match", buttons[0].Label)
	require.Equal(t, link.Path(match), buttons[0].URL)
	require.Equal(t, "Download STV", buttons[1].Label)
	require.Equal(t, link.Raw("/asset/"+match.AssetID.String()), buttons[1].URL)
}

func TestMatchMessageDrawWithoutDemo(t *testing.T) {
	stats.RegisterDiscordCommands(discord.Discard{})

	matchID, errID := uuid.FromString("b233a951-cc14-405e-b92a-6196b17d21c6")
	require.NoError(t, errID)

	match := stats.Match{
		MatchID:   matchID,
		MapName:   "ctf_2fort",
		ScoreRed:  2,
		ScoreBlu:  2,
		StartTime: time.Date(2026, time.October, 21, 14, 0, 0, 0, time.UTC),
	}

	message := stats.MatchMessage(match)
	content := matchMessageText(t, message)

	require.Contains(t, content, "ctf_2fort")
	require.Contains(t, content, "Draw")
	require.Contains(t, content, "No players recorded")

	buttons := matchMessageButtons(t, message)
	require.Len(t, buttons, 1)
	require.Equal(t, "View Match", buttons[0].Label)
	require.Equal(t, link.Path(match), buttons[0].URL)
}

func testMatchDiscordMatch(t *testing.T) stats.Match {
	t.Helper()

	matchID, errID := uuid.FromString("7d59fcf3-2a46-48c7-89a8-c26c0ea93bc0")
	require.NoError(t, errID)

	assetID, errAsset := uuid.FromString("8506bbfd-353f-4332-ae9d-b951306dd393")
	require.NoError(t, errAsset)

	players := []stats.MatchOverallStatsRound{
		matchPlayerRound(1, "red", 76561198000000001, "Red One", 50, 5, 1, 2, 1000),
		matchPlayerRound(2, "red", 76561198000000001, "Red One", 50, 5, 1, 2, 1000),
		matchPlayerRound(1, "red", 76561198000000002, "Red Two", 90, 4, 2, 3, 900),
		matchPlayerRound(1, "red", 76561198000000003, "Red Three", 80, 3, 3, 4, 800),
		matchPlayerRound(1, "red", 76561198000000004, "Red Four", 70, 2, 4, 5, 700),
		matchPlayerRound(1, "blu", 76561198000000005, "Blue One", 95, 6, 1, 2, 1100),
		matchPlayerRound(1, "blu", 76561198000000006, "Blue Two", 85, 5, 2, 3, 1000),
		matchPlayerRound(1, "blu", 76561198000000007, "Blue Three", 75, 4, 3, 4, 900),
		matchPlayerRound(1, "blu", 76561198000000008, "Blue Four", 65, 3, 4, 5, 800),
		matchPlayerRound(3, "red", 76561198000000009, "Zero Round", 999, 99, 99, 0, 99999),
	}

	return stats.Match{
		MatchID:         matchID,
		AssetID:         assetID,
		ServerName:      "gbans Casual #1",
		ServerNameShort: "gbans-1",
		MapName:         "cp_process_f7",
		ScoreRed:        3,
		ScoreBlu:        2,
		StartTime:       time.Date(2026, time.October, 21, 14, 0, 0, 0, time.UTC),
		DurationMs:      1_452_000,
		Rounds: []stats.MatchRound{
			{RoundID: 1, Winner: "red", DurationMs: 600_000},
			{RoundID: 2, Winner: "red", DurationMs: 600_000},
			{RoundID: 3, Winner: "blu", DurationMs: 0},
		},
		Players: players,
	}
}

func matchPlayerRound(roundID uint32, team string, steamID int64, name string, points, kills, assists, deaths, damage uint64) stats.MatchOverallStatsRound {
	return stats.MatchOverallStatsRound{
		OverallStats: stats.OverallStats{ //nolint:modernize // embedded field type is required in a keyed literal.
			SteamID:          steamid.New(steamID),
			Personaname:      name,
			Points:           points,
			Kills:            kills,
			Assists:          assists,
			ScoreboardDeaths: deaths,
			ScoreboardDamage: damage,
		},
		Team:    team,
		RoundID: roundID,
	}
}

func matchMessageText(t *testing.T, message *discordgo.MessageSend) string {
	t.Helper()

	var parts []string

	for _, component := range message.Components {
		switch typed := component.(type) {
		case discordgo.Container:
			for _, child := range typed.Components {
				if text, ok := child.(discordgo.TextDisplay); ok {
					parts = append(parts, text.Content)
				}
			}
		case discordgo.TextDisplay:
			parts = append(parts, typed.Content)
		}
	}

	return strings.Join(parts, "\n")
}

func matchMessageButtons(t *testing.T, message *discordgo.MessageSend) []discordgo.Button {
	t.Helper()

	var buttons []discordgo.Button

	for _, component := range message.Components {
		row, ok := component.(discordgo.ActionsRow)
		if !ok {
			continue
		}

		for _, child := range row.Components {
			if button, okButton := child.(discordgo.Button); okButton {
				buttons = append(buttons, button)
			}
		}
	}

	return buttons
}
