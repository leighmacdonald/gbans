package votes

import (
	_ "embed"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/pkg/logparse"
)

//go:embed vote_discord.tmpl
var templateBody []byte

type voteResultView struct {
	SourceID   string
	TargetName string
	TargetSID  string
	Code       logparse.VoteCode
	Success    bool
	Server     int
}

func VoteResultMessage(result Result, _ person.Core, target person.Core) *discordgo.MessageSend {
	var colour int
	if result.Success {
		colour = discord.ColourSuccess
	} else {
		colour = discord.ColourWarn
	}

	body, errBody := discord.Render("vote_result", templateBody, voteResultView{
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

	return discord.NewMessage(discord.BodyColouredText(colour, body))
}
