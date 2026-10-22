package votes_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/votes"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/stretchr/testify/require"
)

func TestVoteResultTemplate(t *testing.T) {
	votes.RegisterDiscordCommands(discord.Discard{})

	view := struct {
		SourceID   string
		TargetName string
		TargetSID  string
		Code       logparse.VoteCode
		Success    bool
		ServerID   int32
	}{
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
	require.Contains(t, out, "42")
	require.NotContains(t, out, "Failed to render")
}
