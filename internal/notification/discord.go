package notification

import "github.com/bwmarrin/discordgo"

func NotificationMessage(message string, link string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("Notification", message)
	if link != "" {
		msgEmbed.Embed().SetURL(link)
	}

	return msgEmbed.Embed().Truncate().MessageEmbed
}
