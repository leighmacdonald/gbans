package auth

import (
	_ "embed"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/discord"
	personDomain "github.com/leighmacdonald/gbans/internal/domain/person"
)

//go:embed auth_discord.tmpl
var templateBody []byte

func RegisterDiscordCommands(_ discord.Service) {
	discord.MustRegisterTemplate(templateBody)
}

type loginInfo struct {
	Person personDomain.Info
}

func loginMessage(fetchedPerson personDomain.Info) *discordgo.MessageSend {
	return discord.NewMessage(
		discord.Heading("User logged in"),
		discord.BodyTextWithThumbnail(discord.ColourInfo,
			discord.PlayerThumbnail(fetchedPerson), "login_info", loginInfo{Person: fetchedPerson}))
}

func logoutMessage(fetchedPerson personDomain.Info) *discordgo.MessageSend {
	return discord.NewMessage(
		discord.Heading("User logged out"),
		discord.BodyTextWithThumbnail(discord.ColourInfo,
			discord.PlayerThumbnail(fetchedPerson), "login_info", loginInfo{Person: fetchedPerson}))
}
