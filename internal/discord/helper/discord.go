package helper

import (
	"context"

	"github.com/bwmarrin/discordgo"
)

type SlashCommandHandler func(ctx context.Context, s *discordgo.Session, m *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error)

type Cmd string
