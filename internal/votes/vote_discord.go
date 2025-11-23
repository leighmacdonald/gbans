package votes

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/pkg/logparse"
)

func VoteResultMessage(result Result, source person.Core, target person.Core) *discordgo.MessageSend {
	var colour int
	if result.Success {
		colour = discord.ColourSuccess
	} else {
		colour = discord.ColourWarn
	}

	const format = `# Vote Result
Caller SID: {{ .SourceID }}
Target Name: {{ .TargetName }}
Target SID: {{ .TargetSID }}
Code: {{ .Code }}
Success: {{ .Success }}
Server: {{ .ServerID }}
`

	body, errBody := discord.Render("vote_result", format, struct {
		SourceID   string
		TargetName string
		TargetSID  string
		Code       logparse.VoteCode
		Success    bool
		Server     int
	}{
		SourceID:   result.SourceID.String(),
		TargetName: target.GetName(),
		TargetSID:  result.TargetID.String(),
		Code:       result.Code,
		Success:    result.Success,
		Server:     result.ServerID,
	})
	if errBody != nil {
		slog.Error("Failed to render vote result message", slog.String("error", errBody.Error()))
	}

	return discord.NewMessageSend(discordgo.Container{
		AccentColor: ptr.To(colour),
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: body},
		},
	})
}
