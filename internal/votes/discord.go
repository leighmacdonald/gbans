package votes

import (
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/votes"
)

func VoteResultMessage(conf *config.Config, result votes.VoteResult, source person.Person, target person.Person) *discordgo.MessageEmbed {
	avatarSource := domain.NewAvatar(source.AvatarHash)
	avatarTarget := domain.NewAvatar(target.AvatarHash)

	msgEmbed := NewEmbed("Vote Result")
	if result.Success {
		msgEmbed.emb.Color = ColourSuccess
	} else {
		msgEmbed.emb.Color = ColourWarn
	}

	msgEmbed.emb.Thumbnail = &discordgo.MessageEmbedThumbnail{
		URL: avatarTarget.Full(),
	}

	msgEmbed.emb.Author = &discordgo.MessageEmbedAuthor{
		URL:     conf.ExtURL(source),
		Name:    source.PersonaName,
		IconURL: avatarSource.Full(),
	}

	msgEmbed.Embed().
		AddField("Caller SID", result.SourceID.String()).
		AddField("Target", target.PersonaName).
		AddField("Target SID", result.TargetID.String()).
		AddField("Code", fmt.Sprintf("%d", result.Code)).
		AddField("Success", strconv.FormatBool(result.Success)).
		AddField("Server", strconv.FormatInt(int64(result.ServerID), 10)).
		InlineAllFields()

	return msgEmbed.Embed().Truncate().MessageEmbed
}
