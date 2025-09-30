package playerqueue

import (
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain/person"
)

func NewPlayerqueueChatStatus(author person.Info, target person.Info, status ChatStatus, reason string) *discordgo.MessageEmbed {
	colour := discord.ColourError
	switch status {
	case Readwrite:
		colour = discord.ColourSuccess
	case Readonly:
		colour = discord.ColourWarn
	}

	sid := target.GetSteamID()

	return discord.NewEmbed("Updated chat status").
		Embed().
		SetColor(colour).
		SetAuthor(author.GetName(), author.GetAvatar().Small()).
		AddField("Status", string(status)).
		AddField("Reason", reason).
		AddField("Name", target.GetName()).
		AddField("SteamID", sid.String()).
		SetThumbnail(target.GetAvatar().Medium()).
		MessageEmbed
}

func NewPlayerqueueMessage(author person.Info, msg string) *discordgo.MessageEmbed {
	return discord.NewEmbed().
		Embed().
		SetColor(discord.ColourInfo).
		SetAuthor(author.GetName(), author.GetAvatar().Small()).
		SetDescription(msg).MessageEmbed
}

func NewPlayerqueuePurge(author person.Info, target person.Info, chatLog ChatLog, count int) *discordgo.MessageEmbed {
	sid := target.GetSteamID()

	return discord.NewEmbed().
		Embed().
		SetColor(discord.ColourInfo).
		SetAuthor(author.GetName(), author.GetAvatar().Small()).
		SetThumbnail(target.GetAvatar().Medium()).
		AddField("Message", chatLog.BodyMD).
		AddField("Count", strconv.Itoa(count)).
		AddField("Name", target.GetName()).
		AddField("SteamID", sid.String()).
		MessageEmbed
}
