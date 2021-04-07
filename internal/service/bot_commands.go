package service

import (
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	log "github.com/sirupsen/logrus"
)

func botRegisterSlashCommands(appID string, authToken string) error {
	slashCommands := []*discordgo.ApplicationCommand{
		{
			ApplicationID: config.Discord.AppID,
			Name:          "check",
			Description:   "Get ban status for a steam id",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "user_identifier",
					Description: "Steam ID of the user",
					Required:    true,
				},
			},
		},
	}
	for _, cmd := range slashCommands {
		if _, err := dg.ApplicationCommandCreate(appID, "491843276849020939", cmd); err != nil {
			log.Errorf("Failed to register command: %v", err)
		}
	}
	return nil
}
