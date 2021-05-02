package service

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	dg                 *discordgo.Session
	messageQueue       chan logMessage
	connected          bool
	errUnknownBan      = errors.New("Unknown ban")
	errInvalidSID      = errors.New("Invalid steamid")
	errUnknownID       = errors.New("Could not find matching player/steamid")
	errCommandFailed   = errors.New("Command failed")
	errDuplicateBan    = errors.New("Duplicate ban")
	errInvalidDuration = errors.New("Invalid duration")
)

func init() {
	messageQueue = make(chan logMessage)
}

func startDiscord(ctx context.Context, token string) {
	d, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Errorf("Failed to connect to dg. Bot unavailable")
		return
	}
	defer func() {
		if err := dg.Close(); err != nil {
			log.Errorf("Failed to cleanly shutdown discord: %v", err)
		}
	}()
	dg = d
	dg.UserAgent = "gbans (https://github.com/leighmacdonald/gbans, " + BuildVersion + ")"
	dg.AddHandler(onReady)
	dg.AddHandler(onConnect)
	dg.AddHandler(onDisconnect)
	dg.AddHandler(onInteractionCreate)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		log.Fatalf("Error opening discord connection: %v,", err)
		return
	}
	go discordMessageQueueReader(ctx)

	if err2 := botRegisterSlashCommands(); err2 != nil {
		log.Errorf("Failed to register discord slash commands: %v", err2)
	}

	<-ctx.Done()
}

func discordMessageQueueReader(ctx context.Context) {
	events := make(chan logEvent)
	if err := registerLogEventReader(events); err != nil {
		log.Warnf("logWriter Tried to register duplicate reader channel")
	}
	for {
		select {
		case dm := <-events:
			switch dm.Type {
			case logparse.SayTeam:
				fallthrough
			case logparse.Say:
				prefix := ""
				if dm.Type == logparse.SayTeam {
					prefix = "(team) "
				}
				for _, c := range config.Relay.ChannelIDs {
					sendMessage(newMessage(c, fmt.Sprintf("[%s] %d **%s** %s%s",
						dm.Server.ServerName, dm.Player1.SteamID, dm.Event["name"], prefix, dm.Event["msg"])))
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func onReady(_ *discordgo.Session, _ *discordgo.Ready) {
	log.Infof("Bot is connected & ready")
}

func onConnect(s *discordgo.Session, _ *discordgo.Connect) {
	log.Info("Connected to session ws API")
	d := discordgo.UpdateStatusData{
		//Game: &discordgo.Game{
		//	Name:    `Uncletopia`,
		//	URL:     "git@github.com/leighmacdonald/gbans",
		//	Details: "Mr. Authority",
		//},
	}
	if err := s.UpdateStatusComplex(d); err != nil {
		log.WithError(err).Errorf("Failed to update status complex")
	}
	connected = true
}

func onDisconnect(_ *discordgo.Session, _ *discordgo.Disconnect) {
	connected = false
	log.Info("Disconnected from session ws API")
}

func sendMsg(s *discordgo.Session, i *discordgo.Interaction, msg string, args ...interface{}) error {
	if !connected {
		log.Warnf("Tried to send message to disconnected client")
		return nil
	}
	return s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionApplicationCommandResponseData{
			Content: fmt.Sprintf(msg, args...),
		},
	})
}

type logMessage struct {
	ServerID string
	Body     string
}

func newMessage(channel string, body string) logMessage {
	return logMessage{
		ServerID: channel,
		Body:     body,
	}
}

func sendMessage(message logMessage) {
	if config.Discord.Enabled {
		messageQueue <- message
	}
}
