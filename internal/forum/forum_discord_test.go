package forum_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/forum"
	"github.com/stretchr/testify/require"
)

func TestForumSavedTemplate(t *testing.T) {
	forum.RegisterDiscordCommands(discord.Discard{})

	view := struct {
		Forum       string
		Description string
	}{Forum: "My Forum", Description: "A forum"}

	out, err := discord.RenderTemplate("forum_saved", view)
	require.NoError(t, err)
	require.Contains(t, out, "My Forum")
	require.NotContains(t, out, "Failed to render")
}
