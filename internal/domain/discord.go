package domain

import "github.com/bwmarrin/discordgo"

type DiscordUsecase interface {
	SendPayload(channelID string, embed *discordgo.MessageEmbed)
}
