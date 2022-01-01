// Package discord implements the ChatBot interface using discord as the underlying chat service
package app

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

//const (
//	embedIconURL = "https://raw.githubusercontent.com/leighmacdonald/gbans/master/frontend/src/icons/logo.svg"
//)

var (
	errCommandFailed = errors.New("Command failed")
	errTooLarge      = errors.Errorf("Max message length is %d", discordMaxMsgLen)
)

// discord implements the ChatBot interface for the discord chat platform.
type discord struct {
	dg              *discordgo.Session
	connectedMu     *sync.RWMutex
	connected       bool
	commandHandlers map[botCmd]botCommandHandler
}

// NewDiscord instantiates a new, unconnected, discord instance
func NewDiscord() (*discord, error) {
	b := discord{
		dg:          nil,
		connectedMu: &sync.RWMutex{},
		connected:   false,
	}
	var commandHandlers = map[botCmd]botCommandHandler{
		cmdBan:      b.onBan,
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

func (b *discord) Start(ctx context.Context, token string, eventChan chan model.ServerEvent) error {
	// Immediately connects, so we connect within the Start func
	d, err := discordgo.New("Bot " + token)
	if err != nil {
		return errors.Wrapf(err, "Failed to connect to discord. discord unavailable")

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

	b.dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)

	// Open a websocket connection to discord and begin listening.
	err = b.dg.Open()
	if err != nil {
		return errors.Wrap(err, "Error opening discord connection")
	}

	if len(config.Discord.LogChannelID) > 0 {
		go b.discordMessageQueueReader(ctx, eventChan)
	}

	if err2 := b.botRegisterSlashCommands(); err2 != nil {
		log.Errorf("Failed to register discord slash commands: %v", err2)
	}

	<-ctx.Done()
	return nil
}

// discordMessageQueueReader functions by registering event handlers for the two user message events
// discord will rate limit you once you start approaching 5-10 servers of active users. Because of this
// we queue messages and periodically send them out as multiline string blocks instead.
func (b *discord) discordMessageQueueReader(ctx context.Context, eventChan chan model.ServerEvent) {
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

func (b *discord) onReady(_ *discordgo.Session, _ *discordgo.Ready) {
	log.WithFields(log.Fields{"service": "discord", "status": "ready"}).Infof("Service status changed")
}

func (b *discord) onConnect(s *discordgo.Session, _ *discordgo.Connect) {
	log.Tracef("Connected to session ws API")
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

func (b *discord) onDisconnect(_ *discordgo.Session, _ *discordgo.Disconnect) {
	b.connectedMu.Lock()
	b.connected = false
	b.connectedMu.Unlock()
	log.Info("Disconnected from session ws API")
}

func (b *discord) sendChannelMessage(s *discordgo.Session, c string, msg string, wrap bool) error {
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

func (b *discord) sendInteractionMessageEdit(s *discordgo.Session, i *discordgo.Interaction, r botResponse) error {
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

func (b *discord) Send(channelId string, message string, wrap bool) error {
	return b.sendChannelMessage(b.dg, channelId, message, wrap)
}

func (b *discord) SendEmbed(channelId string, message *discordgo.MessageEmbed) error {
	if _, errSend := b.dg.ChannelMessageSendEmbed(channelId, message); errSend != nil {
		return errSend
	}
	return nil
}

func addFieldInline(e *discordgo.MessageEmbed, title string, value string) {
	addFieldRaw(e, title, value, true)
}

func addField(e *discordgo.MessageEmbed, title string, value string) {
	addFieldRaw(e, title, value, false)
}

func addFieldRaw(e *discordgo.MessageEmbed, title string, value string, inline bool) {
	const maxEmbedFields = 25
	if len(e.Fields) >= maxEmbedFields {
		log.Warnf("Dropping embed fields. Already at max count: %d", maxEmbedFields)
		return
	}
	e.Fields = append(e.Fields, &discordgo.MessageEmbedField{
		Name:   title,
		Value:  value,
		Inline: inline,
	})
}

func addFieldsSteamID(e *discordgo.MessageEmbed, sid steamid.SID64) {
	addFieldInline(e, "STEAM", string(steamid.SID64ToSID(sid)))
	addFieldInline(e, "STEAM3", string(steamid.SID64ToSID3(sid)))
	addFieldInline(e, "SID64", sid.String())
}

func addFieldFilter(e *discordgo.MessageEmbed, filter model.Filter) {
	addFieldInline(e, "Pattern", filter.Pattern.String())
	addFieldInline(e, "ID", fmt.Sprintf("%d", filter.WordID))
}

// ChatBot defines a interface for communication with 3rd party service bots
// Currently this is only used for discord, but other providers such as
// Guilded, Matrix, IRC, etc. are planned.
// TODO decouple embed's from discordgo
type ChatBot interface {
	Start(ctx context.Context, token string, eventChan chan model.ServerEvent) error
	Send(channelId string, message string, wrap bool) error
	SendEmbed(channelId string, message *discordgo.MessageEmbed) error
}
