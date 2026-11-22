package votes_test

import (
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/discord"
	personDomain "github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/votes"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func TestVoteResultTemplate(t *testing.T) {
	votes.RegisterDiscordCommands(discord.Discard{})

	view := struct {
		Source     personDomain.Core
		SourceID   string
		TargetName string
		TargetSID  string
		Code       logparse.VoteCode
		Success    bool
		ServerID   int32
	}{
		Source:     personDomain.Core{SteamID: steamid.New(76561198000000001), Name: "Voter"},
		SourceID:   "76561198000000001",
		TargetName: "Target",
		TargetSID:  "76561198000000002",
		Code:       logparse.VoteCodeFailGeneric,
		Success:    true,
		ServerID:   42,
	}

	out, err := discord.RenderTemplate("vote_result", view)
	require.NoError(t, err)
	require.Contains(t, out, "Target")
	require.Contains(t, out, "Voter")
	require.Contains(t, out, "42")
	require.NotContains(t, out, "Failed to render")
}

func TestVoteResultMessageIncludesVoter(t *testing.T) {
	votes.RegisterDiscordCommands(discord.Discard{})

	source := personDomain.Core{SteamID: steamid.New(76561198000000001), Name: "Voter"}
	target := personDomain.Core{SteamID: steamid.New(76561198000000002), Name: "Target"}
	result := votes.Result{
		SourceID:  source.SteamID,
		TargetID:  target.SteamID,
		Code:      logparse.VoteCodeFailGeneric,
		Success:   true,
		CreatedOn: time.Now(),
	}

	msg := votes.VoteResultMessage(result, source, target)
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

	require.Contains(t, content.String(), "Voter")
	require.Contains(t, strings.Join(urls, " "), "/profile/76561198000000001")
}
