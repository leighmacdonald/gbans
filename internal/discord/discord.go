package discord

import (
	"context"
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"sync/atomic"
)

var (
	ErrCommandFailed = errors.New("Command failed")
	ErrTooLarge      = errors.Errorf("Max message length is %d", discordMaxMsgLen)
)

var (
	logger    *zap.Logger
	session   *discordgo.Session
	connected atomic.Bool
	//ctx             context.Context
	commandHandlers map[Cmd]CommandHandler
)

func init() {
	commandHandlers = map[Cmd]CommandHandler{}
}

func RegisterHandler(cmd Cmd, handler CommandHandler) error {
	_, found := commandHandlers[cmd]
	if found {
		return errors.New("Duplicate command")
	}
	commandHandlers[cmd] = handler
	return nil
}

func SendEmbed(channelId string, message *discordgo.MessageEmbed) error {
	if session == nil {
		return nil
	}
	if _, errSend := session.ChannelMessageSendEmbed(channelId, message); errSend != nil {
		return errSend
	}
	return nil
}

func Start(ctx context.Context, l *zap.Logger) error {
	logger = l.Named("discord")

	// Immediately connects, so we connect within the Start func
	botSession, errNewSession := discordgo.New("Bot " + config.Discord.Token)
	if errNewSession != nil {
		return errors.Wrapf(errNewSession, "Failed to connect to discordutil. discordutil unavailable")
	}
	defer func() {
		if botSession != nil {
			if errDisc := botSession.Close(); errDisc != nil {
				logger.Error("Failed to cleanly shutdown discord", zap.Error(errDisc))
			}
		}
	}()

	botSession.UserAgent = "gbans (https://github.com/leighmacdonald/gbans)"
	botSession.AddHandler(onReady)
	botSession.AddHandler(onConnect)
	botSession.AddHandler(onDisconnect)
	botSession.AddHandler(onInteractionCreate)

	botSession.Identify.Intents |= discordgo.IntentsGuildMessages
	botSession.Identify.Intents |= discordgo.IntentMessageContent
	botSession.Identify.Intents |= discordgo.IntentGuildMembers
	//botSession.Identify.Intents |= discordgo.IntentGuildPresences

	// Open a websocket connection to discord and begin listening.
	if errSessionOpen := botSession.Open(); errSessionOpen != nil {
		return errors.Wrap(errSessionOpen, "Error opening discord connection")
	}

	session = botSession
	if errRegister := botRegisterSlashCommands(ctx); errRegister != nil {
		logger.Error("Failed to register discord slash commands", zap.Error(errRegister))
	}
	return nil
}

func onReady(_ *discordgo.Session, _ *discordgo.Ready) {
	connected.Store(true)
	logger.Info("Service state changed", zap.String("state", "ready"))
}

func onConnect(session *discordgo.Session, _ *discordgo.Connect) {
	status := discordgo.UpdateStatusData{
		IdleSince: nil,
		Activities: []*discordgo.Activity{
			{
				Name:     "Cheeseburgers",
				Type:     discordgo.ActivityTypeListening,
				URL:      config.General.ExternalUrl,
				State:    "state field",
				Details:  "Blah",
				Instance: true,
				Flags:    1 << 0,
			},
		},
		AFK:    false,
		Status: "https://github.com/leighmacdonald/gbans",
	}
	if errUpdateStatus := session.UpdateStatusComplex(status); errUpdateStatus != nil {
		logger.Error("Failed to update status complex", zap.Error(errUpdateStatus))
	}
	logger.Info("Service state changed", zap.String("state", "connected"))
}

func onDisconnect(_ *discordgo.Session, _ *discordgo.Disconnect) {
	connected.Store(false)

	logger.Info("Service state changed", zap.String("state", "disconnected"))
}

func sendChannelMessage(session *discordgo.Session, channelId string, msg string, wrap bool) error {
	if !connected.Load() {
		logger.Error("Tried to send message to disconnected client")
		return nil
	}
	if wrap {
		msg = discordMsgWrapper + msg + discordMsgWrapper
	}
	if len(msg) > discordMaxMsgLen {
		return ErrTooLarge
	}
	_, errChannelMessageSend := session.ChannelMessageSend(channelId, msg)
	if errChannelMessageSend != nil {
		return errors.Wrapf(errChannelMessageSend, "Failed sending success (paged) response for interaction")
	}
	return nil
}

func sendInteractionMessageEdit(session *discordgo.Session, interaction *discordgo.Interaction, response Response) error {
	if !connected.Load() {
		logger.Error("Tried to send message edit to disconnected client")
		return nil
	}
	edit := &discordgo.WebhookEdit{
		Embeds:          nil,
		AllowedMentions: nil,
	}
	var embeds []*discordgo.MessageEmbed
	switch response.MsgType {
	case MtString:
		val, ok := response.Value.(string)
		if ok && val != "" {
			edit.Content = &val
			if len(*edit.Content) > discordMaxMsgLen {
				return ErrTooLarge
			}
		}
	case MtEmbed:
		embeds = append(embeds, response.Value.(*discordgo.MessageEmbed))
		edit.Embeds = &embeds
	}
	_, errResp := session.InteractionResponseEdit(interaction, edit)
	return errResp
}

func SendPayload(payload Payload) error {
	panic("x")
	return nil
}

func Send(channelId string, message string, wrap bool) error {
	return sendChannelMessage(session, channelId, message, wrap)
}

// LevelColors is a struct of the possible colors used in Discord color format (0x[RGB] converted to int)
type LevelColors struct {
	Debug int
	Info  int
	Warn  int
	Error int
	Fatal int
}

// DefaultLevelColors is a struct of the default colors used
var DefaultLevelColors = LevelColors{
	Debug: 10170623,
	Info:  3581519,
	Warn:  14327864,
	Error: 13631488,
	Fatal: 13631488,
}
