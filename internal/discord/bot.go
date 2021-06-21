package discord

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"strings"
	"sync"
	"time"
)

var (
	dg               *discordgo.Session
	connectedMu      *sync.RWMutex
	connected        = false
	errCommandFailed = errors.New("Command failed")
	errTooLarge      = errors.Errorf("Max message length is %d", discordMaxMsgLen)
)

func Start(ctx context.Context, token string, eventChan chan model.ServerEvent) {
	d, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Errorf("Failed to connect to dg. Bot unavailable")
		return
	}
	defer func() {
		if errDisc := dg.Close(); errDisc != nil {
			log.Errorf("Failed to cleanly shutdown discord: %v", errDisc)
		}
	}()
	dg = d
	dg.UserAgent = "gbans (https://github.com/leighmacdonald/gbans)"
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
	go discordMessageQueueReader(ctx, eventChan)

	if err2 := botRegisterSlashCommands(); err2 != nil {
		log.Errorf("Failed to register discord slash commands: %v", err2)
	}

	<-ctx.Done()
}

// discordMessageQueueReader functions by registering event handlers for the two user message events
// Discord will rate limit you once you start approaching 5-10 servers of active users. Because of this
// we queue messages and periodically send them out as multiline string blocks instead.
func discordMessageQueueReader(ctx context.Context, eventChan chan model.ServerEvent) {
	messageTicker := time.NewTicker(time.Second * 10)
	var sendQueue []string
	for {
		select {
		case dm := <-eventChan:
			prefix := ""
			if dm.EventType == logparse.SayTeam {
				prefix = "(team) "
			}
			name := ""
			sid := steamid.SID64(0)
			if dm.Source != nil && dm.Source.SteamID.Valid() {
				sid = dm.Source.SteamID
				name = dm.Source.PersonaName
			}
			sendQueue = append(sendQueue, fmt.Sprintf("[%s] %d **%s** %s%s",
				dm.Server.ServerName, sid, name, prefix, dm.Extra))
		case <-messageTicker.C:
			if len(sendQueue) == 0 {
				continue
			}
			msg := strings.Join(sendQueue, "\n")
			for _, m := range util.StringChunkDelimited(msg, discordWrapperTotalLen) {
				for _, channelID := range config.Relay.ChannelIDs {
					if err := sendChannelMessage(dg, channelID, m, true); err != nil {
						log.Errorf("Failed to send bulk message log: %v", err)
					}
				}
			}
			sendQueue = nil
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
		IdleSince: nil,
		Activities: []*discordgo.Activity{
			{
				Name:     "Cheeseburgers",
				Type:     discordgo.ActivityTypeStreaming,
				URL:      "https://" + config.HTTP.Addr(),
				State:    "state field",
				Details:  "Blah",
				Instance: true,
				Flags:    1 << 0,
			},
		},
		AFK:    false,
		Status: "https://github.com/leighmacdonald/gbans",
	}
	if err := s.UpdateStatusComplex(d); err != nil {
		log.WithError(err).Errorf("Failed to update status complex")
	}
	connectedMu.Lock()
	connected = true
	connectedMu.Unlock()
}

func onDisconnect(_ *discordgo.Session, _ *discordgo.Disconnect) {
	connectedMu.Lock()
	connected = false
	connectedMu.Unlock()
	log.Info("Disconnected from session ws API")
}

func sendChannelMessage(s *discordgo.Session, c string, msg string, wrap bool) error {
	connectedMu.RLock()
	if !connected {
		connectedMu.RUnlock()
		log.Warnf("Tried to send message to disconnected client")
		return nil
	}
	connectedMu.RUnlock()
	if wrap {
		msg = discordMsgWrapper + msg + discordMsgWrapper
	}
	if len(msg) > discordMaxMsgLen {
		return errTooLarge
	}
	_, err := s.ChannelMessageSend(c, msg)
	if err != nil {
		return errors.Wrapf(err, "Failed sending success (paged) response for interaction")
	}
	return nil
}

func sendInteractionMessageEdit(s *discordgo.Session, i *discordgo.Interaction, msg string) error {
	connectedMu.RLock()
	if !connected {
		connectedMu.RUnlock()
		log.Warnf("Tried to send message to disconnected client")
		return nil
	}
	connectedMu.RUnlock()
	msg = discordMsgWrapper + msg + discordMsgWrapper
	if len(msg) > discordMaxMsgLen {
		return errTooLarge
	}
	return s.InteractionResponseEdit(config.Discord.AppID, i, &discordgo.WebhookEdit{Content: msg})
}

func Send(channelId string, message string, wrap bool) error {
	return sendChannelMessage(dg, channelId, message, wrap)
}

func init() {
	connectedMu = &sync.RWMutex{}
}
