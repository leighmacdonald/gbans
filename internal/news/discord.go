package news

import "github.com/bwmarrin/discordgo"

func NewNewsMessage(body string, title string) *discordgo.MessageEmbed {
	return NewEmbed("News Created").
		Embed().
		SetDescription(body).
		AddField("Title", title).MessageEmbed
}

func EditNewsMessages(title string, body string) *discordgo.MessageEmbed {
	return NewEmbed("News Updated").
		Embed().
		AddField("Title", title).
		SetDescription(body).
		MessageEmbed
}
