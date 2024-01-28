package domain

import "github.com/bwmarrin/discordgo"

type DiscordChannel int

const (
	ChannelMod DiscordChannel = iota
	ChannelModLog
	ChannelPublicLog
	ChannelPublicMatchLog
)

type DiscordUsecase interface {
	SendPayload(channelID DiscordChannel, embed *discordgo.MessageEmbed)
}
