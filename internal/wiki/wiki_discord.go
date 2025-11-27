package wiki

import (
	_ "embed"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/discord"
)

//go:embed wiki_discord.tmpl
var templateBody []byte

func pageCreated(page Page) *discordgo.MessageSend {
	content, errContent := discord.Render("wiki_created", templateBody, page)
	if errContent != nil {
		return nil
	}

	return discord.NewMessageSend(
		discordgo.Label{
			Label:       page.Slug,
			Description: "New wiki page created",
			Component:   discordgo.TextDisplay{Content: content},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{Label: "View", Style: discordgo.LinkButton, URL: link.Path(page)},
			},
		})
}

func pageEdited(page Page, _ Page) *discordgo.MessageSend {
	content, errContent := discord.Render("wiki_edited", templateBody, page)
	if errContent != nil {
		return nil
	}

	return discord.NewMessageSend(
		discordgo.Label{
			Label:       page.Slug,
			Description: fmt.Sprintf("Wiki page updated [revision: %d]", page.Revision),
			Component:   discordgo.TextDisplay{Content: content},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{Label: "View", Style: discordgo.LinkButton, URL: link.Path(page)},
			},
		})
}
