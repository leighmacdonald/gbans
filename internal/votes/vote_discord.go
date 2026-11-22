package votes

import (
	_ "embed"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/pkg/logparse"
)

//go:embed vote_discord.gotmpl
var templateBody []byte

func RegisterDiscordCommands(_ discord.Connection) {
	discord.MustRegisterTemplate(templateBody)
}

type voteResultView struct {
	Source     person.Core
	SourceID   string
	TargetName string
	TargetSID  string
	Code       logparse.VoteCode
	Success    bool
	ServerID   int32
}

func VoteResultMessage(result Result, source person.Core, target person.Core) *discordgo.MessageSend {
	var colour int
	if result.Success {
		colour = discord.ColourSuccess
	} else {
		colour = discord.ColourWarn
	}

	body, errBody := discord.RenderTemplate("vote_result", voteResultView{
		Source:     source,
		SourceID:   result.SourceID.String(),
		TargetName: target.GetName(),
		TargetSID:  result.TargetID.String(),
		Code:       result.Code,
		Success:    result.Success,
		ServerID:   result.ServerID,
	})
	if errBody != nil {
		slog.Error("Failed to render vote result message", slog.String("error", errBody.Error()))
	}

	return discord.NewMessage(
		discord.BodyColouredText(colour, body),
		discord.Buttons(discord.Link("👤 Voter", link.Path(source))))
}
