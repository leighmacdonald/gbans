package anticheat_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/anticheat"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/stretchr/testify/require"
)

func TestACLogsTemplate(t *testing.T) {
	anticheat.RegisterDiscordCommands(discord.Discard{}, anticheat.AntiCheat{})

	view := struct {
		Detections map[logparse.Detection]int
		Servers    map[string]int
		Count      int
	}{
		Detections: map[logparse.Detection]int{logparse.Detection("aimbot"): 2},
		Servers:    map[string]int{"us-1": 2},
		Count:      2,
	}

	out, err := discord.RenderTemplate("ac_logs", view)
	require.NoError(t, err)
	require.Contains(t, out, "us-1")
	require.NotContains(t, out, "Failed to render")
}
