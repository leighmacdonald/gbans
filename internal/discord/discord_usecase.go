package discord

import (
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/domain"
)

type discordUsecase struct {
	dr domain.DiscordRepository
}

func NewDiscordUsecase(repository domain.DiscordRepository) domain.DiscordUsecase {
	return &discordUsecase{dr: repository}
}

func (d discordUsecase) Shutdown(guildID string) {
	d.dr.Shutdown(guildID)
}

func (d discordUsecase) RegisterHandler(cmd domain.Cmd, handler domain.SlashCommandHandler) error {
	return d.dr.RegisterHandler(cmd, handler)
}

func (d discordUsecase) Start() error {
	return d.dr.Start()
}

func (d discordUsecase) SendPayload(channelID domain.DiscordChannel, embed *discordgo.MessageEmbed) {
	d.dr.SendPayload(channelID, embed)
}
