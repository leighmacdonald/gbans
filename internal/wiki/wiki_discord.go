package wiki

import (
	_ "embed"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/discord"
)

//go:embed wiki_discord.tmpl
var templateBody []byte

func RegisterDiscordCommands(_ discord.Service) {
	discord.MustRegisterTemplate(templateBody)
}

func pageCreated(page Page) *discordgo.MessageSend {
	content, errContent := discord.RenderTemplate("wiki_created", page, discord.HydrateLinks())
	if errContent != nil {
		return nil
	}

	return discord.NewMessage(
		discord.Heading("Wiki Created: %s", page.Slug),
		discord.BodyText(content),
		discord.Buttons(discordgo.Button{Label: "View", Style: discordgo.LinkButton, URL: link.Path(page)}))
}

func pageEdited(page Page, _ Page) *discordgo.MessageSend {
	content, errContent := discord.RenderTemplate("wiki_edited", page, discord.HydrateLinks())
	if errContent != nil {
		return nil
	}

	return discord.NewMessage(
		discord.Heading("Wiki Edited: %s", page.Slug),
		discord.BodyText(content),
		discord.Buttons(discordgo.Button{Label: "View", Style: discordgo.LinkButton, URL: link.Path(page)}),
	)
}
