package discord

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/action"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"strings"
	"sync"
	"time"
)

const (
	embedIconURL = "https://raw.githubusercontent.com/leighmacdonald/gbans/master/frontend/src/icons/logo.svg"
)

var (
	errCommandFailed = errors.New("Command failed")
	errTooLarge      = errors.Errorf("Max message length is %d", discordMaxMsgLen)
)

type Bot struct {
	dg              *discordgo.Session
	connectedMu     *sync.RWMutex
	connected       bool
	commandHandlers map[botCmd]botCommandHandler
	executor        action.Executor
	db              store.Store
}

// New instantiates a new, unconnected, Bot instance
func New(executor action.Executor, s store.Store) (*Bot, error) {
	b := Bot{
		dg:          nil,
		connectedMu: &sync.RWMutex{},
		connected:   false,
		executor:    executor,
		db:          s,
	}
	var commandHandlers = map[botCmd]botCommandHandler{
		cmdBan:      b.onBan,
		cmdBanIP:    b.onBanIP,
		cmdCheck:    b.onCheck,
		cmdCSay:     b.onCSay,
		cmdFind:     b.onFind,
		cmdKick:     b.onKick,
		cmdMute:     b.onMute,
		cmdPlayers:  b.onPlayers,
		cmdPSay:     b.onPSay,
		cmdSay:      b.onSay,
		cmdServers:  b.onServers,
		cmdUnban:    b.onUnban,
		cmdSetSteam: b.onSetSteam,
		cmdHistory:  b.onHistory,
		cmdFilter:   b.onFilter,
	}
	b.commandHandlers = commandHandlers
	return &b, nil
}

func (b *Bot) Start(ctx context.Context, token string, eventChan chan model.ServerEvent) error {
	d, err := discordgo.New("Bot " + token)
	if err != nil {
		return errors.Wrapf(err, "Failed to connect to discord. Bot unavailable")

	}
	defer func() {
		if errDisc := b.dg.Close(); errDisc != nil {
			log.Errorf("Failed to cleanly shutdown discord: %v", errDisc)
		}
	}()
	b.dg = d
	b.dg.UserAgent = "gbans (https://github.com/leighmacdonald/gbans)"
	b.dg.AddHandler(b.onReady)
	b.dg.AddHandler(b.onConnect)
	b.dg.AddHandler(b.onDisconnect)
	b.dg.AddHandler(b.onInteractionCreate)

	// In this example, we only care about receiving message events.
	b.dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)

	// Open a websocket connection to Discord and begin listening.
	err = b.dg.Open()
	if err != nil {
		return errors.Wrap(err, "Error opening discord connection")
	}
	go b.discordMessageQueueReader(ctx, eventChan)

	if err2 := b.botRegisterSlashCommands(); err2 != nil {
		log.Errorf("Failed to register discord slash commands: %v", err2)
	}

	<-ctx.Done()
	return nil
}

// discordMessageQueueReader functions by registering event handlers for the two user message events
// Discord will rate limit you once you start approaching 5-10 servers of active users. Because of this
// we queue messages and periodically send them out as multiline string blocks instead.
func (b *Bot) discordMessageQueueReader(ctx context.Context, eventChan chan model.ServerEvent) {
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
					if err := b.sendChannelMessage(b.dg, channelID, m, true); err != nil {
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

func (b *Bot) onReady(_ *discordgo.Session, _ *discordgo.Ready) {
	log.Infof("Bot is connected & ready")
}

func (b *Bot) onConnect(s *discordgo.Session, _ *discordgo.Connect) {
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
	b.connectedMu.Lock()
	b.connected = true
	b.connectedMu.Unlock()
}

func (b *Bot) onDisconnect(_ *discordgo.Session, _ *discordgo.Disconnect) {
	b.connectedMu.Lock()
	b.connected = false
	b.connectedMu.Unlock()
	log.Info("Disconnected from session ws API")
}

func (b *Bot) sendChannelMessage(s *discordgo.Session, c string, msg string, wrap bool) error {
	b.connectedMu.RLock()
	if !b.connected {
		b.connectedMu.RUnlock()
		log.Warnf("Tried to send message to disconnected client")
		return nil
	}
	b.connectedMu.RUnlock()
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

func (b *Bot) sendInteractionMessageEdit(s *discordgo.Session, i *discordgo.Interaction, r botResponse) error {
	b.connectedMu.RLock()
	if !b.connected {
		b.connectedMu.RUnlock()
		log.Warnf("Tried to send message to disconnected client")
		return nil
	}
	b.connectedMu.RUnlock()

	e := &discordgo.WebhookEdit{
		Content:         "",
		Embeds:          nil,
		AllowedMentions: nil,
	}
	switch r.MsgType {
	case mtString:
		e.Content = r.Value.(string)
		if len(e.Content) > discordMaxMsgLen {
			return errTooLarge
		}
	case mtEmbed:
		e.Embeds = append(e.Embeds, r.Value.(*discordgo.MessageEmbed))
	}
	return s.InteractionResponseEdit(config.Discord.AppID, i, e)
}

func (b *Bot) Send(channelId string, message string, wrap bool) error {
	return b.sendChannelMessage(b.dg, channelId, message, wrap)
}
func (b *Bot) SendEmbed(channelId string, message *discordgo.MessageEmbed) error {
	if _, errSend := b.dg.ChannelMessageSendEmbed(channelId, message); errSend != nil {
		return errSend
	}
	return nil
}

type ChatBot interface {
	Start(ctx context.Context, token string, eventChan chan model.ServerEvent) error
	Send(channelId string, message string, wrap bool) error
	SendEmbed(channelId string, message *discordgo.MessageEmbed) error
}
