package playerqueue

import (
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/discord/message"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/person"
)

func NewPlayerqueueChatStatus(author person.UserProfile, target person.UserProfile, status ChatStatus, reason string) *discordgo.MessageEmbed {
	colour := message.ColourError
	switch status {
	case Readwrite:
		colour = message.ColourSuccess
	case Readonly:
		colour = message.ColourWarn
	}

	sid := target.GetSteamID()

	return message.NewEmbed("Updated chat status").
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

func NewPlayerqueueMessage(author domain.PersonInfo, msg string) *discordgo.MessageEmbed {
	return message.NewEmbed().
		Embed().
		SetColor(message.ColourInfo).
		SetAuthor(author.GetName(), author.GetAvatar().Small()).
		SetDescription(msg).MessageEmbed
}

func NewPlayerqueuePurge(author domain.PersonInfo, target person.UserProfile, chatLog ChatLog, count int) *discordgo.MessageEmbed {
	return message.NewEmbed().
		Embed().
		SetColor(message.ColourInfo).
		SetAuthor(author.GetName(), author.GetAvatar().Small()).
		SetThumbnail(target.GetAvatar().Medium()).
		AddField("Message", chatLog.BodyMD).
		AddField("Count", strconv.Itoa(count)).
		AddField("Name", target.GetName()).
		AddField("SteamID", target.SteamID.String()).
		MessageEmbed
}
