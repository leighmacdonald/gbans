package auth

import (
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/discord"
	personDomain "github.com/leighmacdonald/gbans/internal/domain/person"
)

func loginMessage(fetchedPerson personDomain.Core) *discordgo.MessageSend {
	content, errContent := discord.Render("auth_login", templateBody, struct {
		Person personDomain.Core
	}{Person: fetchedPerson})
	if errContent != nil {
		return nil
	}

	return discord.NewMessage(discord.BodyColouredText(discord.ColourSuccess, content))
}

func logoutMessage(steamID string) *discordgo.MessageSend {
	content, errContent := discord.Render("auth_logout", templateBody, struct {
		SteamID string
	}{SteamID: steamID})
	if errContent != nil {
		return nil
	}

	return discord.NewMessage(discord.BodyColouredText(discord.ColourSuccess, content))
}
