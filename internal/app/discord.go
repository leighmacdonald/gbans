package app

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
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

var (
	errCommandFailed = errors.New("Command failed")
	errTooLarge      = errors.Errorf("Max message length is %d", discordMaxMsgLen)
)

func (bot *discord) SendEmbed(channelId string, message *discordgo.MessageEmbed) error {
	if _, errSend := bot.session.ChannelMessageSendEmbed(channelId, message); errSend != nil {
		return errSend
	}
	return nil
}

// discord implements the ChatBot interface for the discord chat platform.
type discord struct {
	session            *discordgo.Session
	database           store.Store
	connectedMu        *sync.RWMutex
	connected          bool
	commandHandlers    map[botCmd]botCommandHandler
	botSendMessageChan chan discordPayload
}

// NewDiscord instantiates a new, unconnected, discord instance
func NewDiscord(database store.Store) (*discord, error) {
	bot := discord{
		session:     nil,
		database:    database,
		connectedMu: &sync.RWMutex{},
		connected:   false,
	}
	bot.commandHandlers = map[botCmd]botCommandHandler{
		cmdBan:      bot.onBan,
		cmdCheck:    bot.onCheck,
		cmdCSay:     bot.onCSay,
		cmdFind:     bot.onFind,
		cmdKick:     bot.onKick,
		cmdMute:     bot.onMute,
		cmdPlayers:  bot.onPlayers,
		cmdPSay:     bot.onPSay,
		cmdSay:      bot.onSay,
		cmdServers:  bot.onServers,
		cmdUnban:    bot.onUnban,
		cmdSetSteam: bot.onSetSteam,
		cmdHistory:  bot.onHistory,
		cmdFilter:   bot.onFilter,
		cmdStats:    bot.onStats,
	}
	return &bot, nil
}

func (bot *discord) Start(ctx context.Context, token string, eventChan chan model.ServerEvent) error {
	// Immediately connects, so we connect within the Start func
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return errors.Wrapf(err, "Failed to connect to discord. discord unavailable")

	}
	defer func() {
		if errDisc := bot.session.Close(); errDisc != nil {
			log.Errorf("Failed to cleanly shutdown discord: %v", errDisc)
		}
	}()
	bot.session = session
	bot.session.UserAgent = "gbans (https://github.com/leighmacdonald/gbans)"
	bot.session.AddHandler(bot.onReady)
	bot.session.AddHandler(bot.onConnect)
	bot.session.AddHandler(bot.onDisconnect)
	bot.session.AddHandler(bot.onInteractionCreate)

	bot.session.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)

	// Open a websocket connection to discord and begin listening.
	err = bot.session.Open()
	if err != nil {
		return errors.Wrap(err, "Error opening discord connection")
	}

	go bot.discordMessageQueueReader(ctx, eventChan)

	if errRegister := bot.botRegisterSlashCommands(); errRegister != nil {
		log.Errorf("Failed to register discord slash commands: %v", errRegister)
	}

	<-ctx.Done()
	return nil
}

// discordMessageQueueReader functions by registering event handlers for the two user message events
// discord will rate limit you once you start approaching 5-10 servers of active users. Because of this
// we queue messages and periodically send them out as multiline string blocks instead.
func (bot *discord) discordMessageQueueReader(ctx context.Context, eventChan chan model.ServerEvent) {
	messageTicker := time.NewTicker(time.Second * 10)
	var sendQueue []string
	for {
		select {
		case serverEvent := <-eventChan:
			prefix := ""
			if serverEvent.EventType == logparse.SayTeam {
				prefix = "(team) "
			}
			name := ""
			sid := steamid.SID64(0)
			if serverEvent.Source != nil && serverEvent.Source.SteamID.Valid() {
				sid = serverEvent.Source.SteamID
				name = serverEvent.Source.PersonaName
			}
			msg, found := serverEvent.MetaData["msg"]
			if found {
				sendQueue = append(sendQueue, fmt.Sprintf("[%s] %d **%s** %s%s",
					serverEvent.Server.ServerName, sid, name, prefix, msg))
			}

		case <-messageTicker.C:
			if len(sendQueue) == 0 {
				continue
			}
			msg := strings.Join(sendQueue, "\n")
			for _, m := range util.StringChunkDelimited(msg, discordWrapperTotalLen) {
				for _, channelID := range config.Relay.ChannelIDs {
					if err := bot.sendChannelMessage(bot.session, channelID, m, true); err != nil {
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

func (bot *discord) onReady(_ *discordgo.Session, _ *discordgo.Ready) {
	log.WithFields(log.Fields{"service": "discord", "status": "ready"}).Infof("Service status changed")
}

func (bot *discord) onConnect(session *discordgo.Session, _ *discordgo.Connect) {
	log.Tracef("Connected to session ws API")
	status := discordgo.UpdateStatusData{
		IdleSince: nil,
		Activities: []*discordgo.Activity{
			{
				Name:     "Cheeseburgers",
				Type:     discordgo.ActivityTypeListening,
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
	if err := session.UpdateStatusComplex(status); err != nil {
		log.WithError(err).Errorf("Failed to update status complex")
	}
	bot.connectedMu.Lock()
	bot.connected = true
	bot.connectedMu.Unlock()
}

func (bot *discord) onDisconnect(_ *discordgo.Session, _ *discordgo.Disconnect) {
	bot.connectedMu.Lock()
	bot.connected = false
	bot.connectedMu.Unlock()
	log.Info("Disconnected from session ws API")
}

func (bot *discord) sendChannelMessage(session *discordgo.Session, channelId string, msg string, wrap bool) error {
	bot.connectedMu.RLock()
	if !bot.connected {
		bot.connectedMu.RUnlock()
		log.Warnf("Tried to send message to disconnected client")
		return nil
	}
	bot.connectedMu.RUnlock()
	if wrap {
		msg = discordMsgWrapper + msg + discordMsgWrapper
	}
	if len(msg) > discordMaxMsgLen {
		return errTooLarge
	}
	_, err := session.ChannelMessageSend(channelId, msg)
	if err != nil {
		return errors.Wrapf(err, "Failed sending success (paged) response for interaction")
	}
	return nil
}

func (bot *discord) sendInteractionMessageEdit(session *discordgo.Session, interaction *discordgo.Interaction, response botResponse) error {
	bot.connectedMu.RLock()
	if !bot.connected {
		bot.connectedMu.RUnlock()
		log.Warnf("Tried to send message to disconnected client")
		return nil
	}
	bot.connectedMu.RUnlock()

	edit := &discordgo.WebhookEdit{
		Content:         "",
		Embeds:          nil,
		AllowedMentions: nil,
	}
	switch response.MsgType {
	case mtString:
		edit.Content = response.Value.(string)
		if len(edit.Content) > discordMaxMsgLen {
			return errTooLarge
		}
	case mtEmbed:
		edit.Embeds = append(edit.Embeds, response.Value.(*discordgo.MessageEmbed))
	}
	return session.InteractionResponseEdit(config.Discord.AppID, interaction, edit)
}

func (bot *discord) Send(channelId string, message string, wrap bool) error {
	return bot.sendChannelMessage(bot.session, channelId, message, wrap)
}

func addFieldInline(embed *discordgo.MessageEmbed, title string, value string) {
	addFieldRaw(embed, title, value, true)
}

func addField(embed *discordgo.MessageEmbed, title string, value string) {
	addFieldRaw(embed, title, value, false)
}

func addFieldRaw(embed *discordgo.MessageEmbed, title string, value string, inline bool) {
	const maxEmbedFields = 25
	if len(embed.Fields) >= maxEmbedFields {
		log.Warnf("Dropping embed fields. Already at max count: %d", maxEmbedFields)
		return
	}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   title,
		Value:  value,
		Inline: inline,
	})
}

func addFieldsSteamID(embed *discordgo.MessageEmbed, steamId steamid.SID64) {
	addFieldInline(embed, "STEAM", string(steamid.SID64ToSID(steamId)))
	addFieldInline(embed, "STEAM3", string(steamid.SID64ToSID3(steamId)))
	addFieldInline(embed, "SID64", steamId.String())
}

func addFieldFilter(embed *discordgo.MessageEmbed, filter model.Filter) {
	addFieldInline(embed, "Pattern", filter.Pattern.String())
	addFieldInline(embed, "ID", fmt.Sprintf("%d", filter.WordID))
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
