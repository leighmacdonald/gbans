package auth

import (
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/discord"
	personDomain "github.com/leighmacdonald/gbans/internal/domain/person"
)

type loginInfo struct {
	Person personDomain.Info
}

func loginMessage(fetchedPerson personDomain.Info) *discordgo.MessageSend {
	content, errContent := discord.Render("login_info", templateBody, loginInfo{Person: fetchedPerson})
	if errContent != nil {
		return nil
	}

	return discord.NewMessage(
		discord.Heading("User logged in"),
		discord.BodyTextWithThumbnail(discord.ColourInfo, discord.PlayerThumbnail(fetchedPerson), content))
}

func logoutMessage(fetchedPerson personDomain.Info) *discordgo.MessageSend {
	content, errContent := discord.Render("login_info", templateBody, loginInfo{Person: fetchedPerson})
	if errContent != nil {
		return nil
	}

	return discord.NewMessage(
		discord.Heading("User logged out"),
		discord.BodyTextWithThumbnail(discord.ColourInfo, discord.PlayerThumbnail(fetchedPerson), content))
}
