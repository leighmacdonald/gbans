package discord

import (
	"github.com/bwmarrin/discordgo"
)

type nullDiscordRepository struct{}

func (bot *nullDiscordRepository) RegisterHandler(_ Cmd, _ SlashCommandHandler) error {
	return nil
}

func (bot *nullDiscordRepository) Shutdown() {
}

func (bot *nullDiscordRepository) Start() error {
	return nil
}

func (bot *nullDiscordRepository) SendPayload(_ string, _ *discordgo.MessageEmbed) {
}

func NewNullDiscordRepository() DiscordRepository {
	return &nullDiscordRepository{}
}
