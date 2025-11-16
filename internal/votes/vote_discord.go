package votes

import (
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain/person"
)

func VoteResultMessage(result Result, source person.Core, target person.Core) *discordgo.MessageEmbed {
	msgEmbed := discord.NewEmbed("Vote Result")
	if result.Success {
		msgEmbed.Emb.Color = discord.ColourSuccess
	} else {
		msgEmbed.Emb.Color = discord.ColourWarn
	}

	msgEmbed.Emb.Thumbnail = &discordgo.MessageEmbedThumbnail{
		URL: target.GetAvatar().Full(),
	}

	msgEmbed.Emb.Author = &discordgo.MessageEmbedAuthor{
		URL:     link.Path(source),
		Name:    source.GetName(),
		IconURL: source.GetAvatar().Full(),
	}

	msgEmbed.Embed().
		AddField("Caller SID", result.SourceID.String()).
		AddField("Target", target.GetName()).
		AddField("Target SID", result.TargetID.String()).
		AddField("Code", fmt.Sprintf("%d", result.Code)).
		AddField("Success", strconv.FormatBool(result.Success)).
		AddField("Server", strconv.FormatInt(int64(result.ServerID), 10)).
		InlineAllFields()

	return msgEmbed.Embed().Truncate().MessageEmbed
}
