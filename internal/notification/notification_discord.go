package notification

import (
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/discord/message"
)

func discordMessage(msg string, link string) *discordgo.MessageEmbed {
	msgEmbed := message.NewEmbed("Notification", msg)
	if link != "" {
		msgEmbed.Embed().SetURL(link)
	}

	return msgEmbed.Embed().Truncate().MessageEmbed
}
