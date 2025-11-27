package wiki

import (
	_ "embed"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/discord"
)

//go:embed wiki_discord.tmpl
var templateBody []byte

func pageCreated(page Page) *discordgo.MessageSend {
	content, errContent := discord.Render("wiki_created", templateBody, page, discord.HydrateLinks())
	if errContent != nil {
		return nil
	}

	return discord.NewMessage(
		discord.Heading("Wiki Created: "+page.Slug),
		discord.BodyText(content),
		discord.Buttons(discordgo.Button{Label: "View", Style: discordgo.LinkButton, URL: link.Path(page)}))
}

func pageEdited(page Page, _ Page) *discordgo.MessageSend {
	content, errContent := discord.Render("wiki_edited", templateBody, page, discord.HydrateLinks())
	if errContent != nil {
		return nil
	}

	return discord.NewMessage(
		discord.Heading("Wiki Edited: "+page.Slug),
		discord.BodyText(content),
		discord.Buttons(discordgo.Button{Label: "View", Style: discordgo.LinkButton, URL: link.Path(page)}),
	)
}
