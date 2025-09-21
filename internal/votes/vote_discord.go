package votes

import (
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/discord/message"
	"github.com/leighmacdonald/gbans/internal/domain"
)

func VoteResultMessage(conf *config.Config, result Result, source domain.PersonCore, target domain.PersonCore) *discordgo.MessageEmbed {
	msgEmbed := message.NewEmbed("Vote Result")
	if result.Success {
		msgEmbed.Emb.Color = message.ColourSuccess
	} else {
		msgEmbed.Emb.Color = message.ColourWarn
	}

	msgEmbed.Emb.Thumbnail = &discordgo.MessageEmbedThumbnail{
		URL: target.GetAvatar().Full(),
	}

	msgEmbed.Emb.Author = &discordgo.MessageEmbedAuthor{
		URL:     conf.ExtURL(source),
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
