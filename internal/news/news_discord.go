package news

import (
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/discord/message"
)

func NewNewsMessage(body string, title string) *discordgo.MessageEmbed {
	return message.NewEmbed("News Created").
		Embed().
		SetDescription(body).
		AddField("Title", title).MessageEmbed
}

func EditNewsMessages(title string, body string) *discordgo.MessageEmbed {
	return message.NewEmbed("News Updated").
		Embed().
		AddField("Title", title).
		SetDescription(body).
		MessageEmbed
}
