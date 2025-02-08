package discord

import (
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/domain"
)

type discordUsecase struct {
	repository domain.DiscordRepository
	config     domain.ConfigUsecase
}

func NewDiscordUsecase(repository domain.DiscordRepository, config domain.ConfigUsecase) domain.DiscordUsecase {
	return &discordUsecase{repository: repository, config: config}
}

func (d discordUsecase) Shutdown(guildID string) {
	d.repository.Shutdown(guildID)
}

func (d discordUsecase) RegisterHandler(cmd domain.Cmd, handler domain.SlashCommandHandler) error {
	return d.repository.RegisterHandler(cmd, handler)
}

func (d discordUsecase) Start() error {
	return d.repository.Start()
}

func (d discordUsecase) SendPayload(channel domain.DiscordChannel, embed *discordgo.MessageEmbed) {
	conf := d.config.Config()

	var channelID string

	switch channel {
	case domain.ChannelMod:
		channelID = conf.Discord.LogChannelID
	case domain.ChannelModLog:
		channelID = conf.Discord.LogChannelID
	case domain.ChannelPublicLog:
		channelID = conf.Discord.PublicLogChannelID
	case domain.ChannelPublicMatchLog:
		channelID = conf.Discord.PublicMatchLogChannelID
	case domain.ChannelModAppealLog:
		channelID = conf.Discord.AppealLogChannelID
	case domain.ChannelModVoteLog:
		channelID = conf.Discord.VoteLogChannelID
	case domain.ChannelBanLog:
		channelID = conf.Discord.BanLogChannelID
	case domain.ChannelForumLog:
		channelID = conf.Discord.ForumLogChannelID
	case domain.ChannelWordFilterLog:
		channelID = conf.Discord.WordFilterLogChannelID
	case domain.ChannelKickLog:
		channelID = conf.Discord.KickLogChannelID
	case domain.ChannelPlayerqueue:
		channelID = conf.Discord.PlayerqueueChannelID
	}

	if channelID == "" {
		channelID = conf.Discord.LogChannelID
	}

	d.repository.SendPayload(channelID, embed)
}
