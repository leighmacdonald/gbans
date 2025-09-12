package playerqueue

import (
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/playerqueue"
)

func NewPlayerqueueChatStatus(author person.UserProfile, target person.UserProfile, status playerqueue.ChatStatus, reason string) *discordgo.MessageEmbed {
	colour := ColourError
	switch status {
	case playerqueue.Readwrite:
		colour = ColourSuccess
	case playerqueue.Readonly:
		colour = ColourWarn
	}

	sid := target.GetSteamID()

	return NewEmbed("Updated chat status").
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

func NewPlayerqueueMessage(author person.UserProfile, msg string) *discordgo.MessageEmbed {
	return NewEmbed().
		Embed().
		SetColor(ColourInfo).
		SetAuthor(author.GetName(), author.GetAvatar().Small()).
		SetDescription(msg).MessageEmbed
}

func NewPlayerqueuePurge(author person.UserProfile, target person.UserProfile, message playerqueue.ChatLog, count int) *discordgo.MessageEmbed {
	return NewEmbed().
		Embed().
		SetColor(ColourInfo).
		SetAuthor(author.GetName(), author.GetAvatar().Small()).
		SetThumbnail(target.GetAvatar().Medium()).
		AddField("Message", message.BodyMD).
		AddField("Count", strconv.Itoa(count)).
		AddField("Name", target.GetName()).
		AddField("SteamID", target.SteamID.String()).
		MessageEmbed
}
