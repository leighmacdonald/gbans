package bot

import (
	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

var (
	discord *discordgo.Session
)

func Start(token string) {
	d, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Errorf("Failed to connect to discord. Bot unavailable")
		return
	}
	discord = d
}
