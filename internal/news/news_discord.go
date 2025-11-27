package news

import (
	_ "embed"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/discord"
)

//go:embed news_discord.tmpl
var templateBody []byte

func NewNewsMessage(body string, title string) *discordgo.MessageSend {
	content, errContent := discord.Render("news_update", templateBody, struct {
		Title string
		Body  string
	}{Title: title, Body: body})
	if errContent != nil {
		return nil
	}

	return discord.NewMessage(
		discord.Heading("News posted"),
		discord.BodyText(content))
}

func EditNewsMessages(body string, title string) *discordgo.MessageSend {
	content, errContent := discord.Render("news_update", templateBody, struct {
		Title string
		Body  string
	}{Title: title, Body: body})
	if errContent != nil {
		return nil
	}

	return discord.NewMessage(
		discord.Heading("News edited"),
		discord.BodyText(content))
}
