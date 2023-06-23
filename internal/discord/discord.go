package discord

import (
	"sync/atomic"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	ErrCommandFailed = errors.New("Command failed")
	ErrTooLarge      = errors.Errorf("Max message length is %d", discordMaxMsgLen)
)

var (
	logger  *zap.Logger
	session *discordgo.Session
	isReady atomic.Bool

	commandHandlers map[Cmd]CommandHandler

	onConnectUser    func()
	onDisconnectUser func()
)

func init() {
	commandHandlers = map[Cmd]CommandHandler{}
}

func SetOnConnect(fn func()) {
	onConnectUser = fn
}

func SetOnDisconnect(fn func()) {
	onDisconnectUser = fn
}

func RegisterHandler(cmd Cmd, handler CommandHandler) error {
	_, found := commandHandlers[cmd]
	if found {
		return errors.New("Duplicate command")
	}
	commandHandlers[cmd] = handler
	return nil
}

func Shutdown() {
	if session != nil {
		defer util.LogCloser(session, logger)
		botUnregisterSlashCommands()
	}
}

func botUnregisterSlashCommands() {
	registeredCommands, err := session.ApplicationCommands(session.State.User.ID, config.Discord.GuildID)
	if err != nil {
		logger.Error("Could not fetch registered commands", zap.Error(err))
		return
	}
	for _, v := range registeredCommands {
		if errDel := session.ApplicationCommandDelete(session.State.User.ID, config.Discord.GuildID, v.ID); errDel != nil {
			logger.Error("Cannot delete command", zap.String("name", v.Name), zap.Error(err))
			return
		}
	}
	logger.Info("Unregistered discord commands", zap.Int("count", len(registeredCommands)))
}

func Start(l *zap.Logger) error {
	logger = l.Named("discord")

	// Immediately connects, so we connect within the Start func
	botSession, errNewSession := discordgo.New("Bot " + config.Discord.Token)
	if errNewSession != nil {
		return errors.Wrapf(errNewSession, "Failed to connect to discord. discord unavailable")
	}

	session = botSession

	session.UserAgent = "gbans (https://github.com/leighmacdonald/gbans)"
	session.AddHandler(onReady)
	session.AddHandler(onConnect)
	session.AddHandler(onDisconnect)
	session.AddHandler(onInteractionCreate)

	session.Identify.Intents |= discordgo.IntentsGuildMessages
	session.Identify.Intents |= discordgo.IntentMessageContent
	session.Identify.Intents |= discordgo.IntentGuildMembers

	// Open a websocket connection to discord and begin listening.
	if errSessionOpen := session.Open(); errSessionOpen != nil {
		return errors.Wrap(errSessionOpen, "Error opening discord connection")
	}
	if errRegister := botRegisterSlashCommands(); errRegister != nil {
		logger.Error("Failed to register discord slash commands", zap.Error(errRegister))
	}

	return nil
}

func onReady(_ *discordgo.Session, _ *discordgo.Ready) {
	logger.Info("Service state changed", zap.String("state", "ready"))
	isReady.Store(true)
}

func onConnect(session *discordgo.Session, _ *discordgo.Connect) {
	status := discordgo.UpdateStatusData{
		IdleSince: nil,
		Activities: []*discordgo.Activity{
			{
				Name:     "Cheeseburgers",
				Type:     discordgo.ActivityTypeListening,
				URL:      config.General.ExternalURL,
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
	if onConnectUser != nil {
		onConnectUser()
	}
}

func onDisconnect(_ *discordgo.Session, _ *discordgo.Disconnect) {
	isReady.Store(false)

	logger.Info("Service state changed", zap.String("state", "disconnected"))
	if onDisconnectUser != nil {
		onDisconnectUser()
	}
}

// func sendChannelMessage(session *discordgo.Session, channelId string, msg string, wrap bool) error {
//	if !isReady.Load() {
//		logger.Error("Tried to send message to disconnected client")
//		return nil
//	}
//	if wrap {
//		msg = discordMsgWrapper + msg + discordMsgWrapper
//	}
//	if len(msg) > discordMaxMsgLen {
//		return ErrTooLarge
//	}
//	_, errChannelMessageSend := session.ChannelMessageSend(channelId, msg)
//	if errChannelMessageSend != nil {
//		return errors.Wrapf(errChannelMessageSend, "Failed sending success (paged) response for interaction")
//	}
//	return nil
//}

func sendInteractionResponse(session *discordgo.Session, interaction *discordgo.Interaction, response Response) error {
	if !isReady.Load() {
		return nil
	}
	edit := &discordgo.InteractionResponseData{
		Content: "hi",
	}
	switch response.MsgType {
	case MtString:
		val, ok := response.Value.(string)
		if ok && val != "" {
			edit.Content = val
			if len(edit.Content) > discordMaxMsgLen {
				return ErrTooLarge
			}
		}
	case MtEmbed:
		embed, ok := response.Value.(*discordgo.MessageEmbed)
		if !ok {
			return errors.New("Failed to cast MessageEmbed")
		}
		edit.Embeds = append(edit.Embeds, embed)
	}
	_, err := session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{
		Embeds: &edit.Embeds,
	})
	return err
}

func SendPayload(payload Payload) {
	if !isReady.Load() {
		return
	}
	if _, errSend := session.ChannelMessageSendEmbed(payload.ChannelID, payload.Embed); errSend != nil {
		logger.Error("Failed to send discord payload", zap.Error(errSend))
	}
}

// LevelColors is a struct of the possible colors used in Discord color format (0x[RGB] converted to int).
type LevelColors struct {
	Debug int
	Info  int
	Warn  int
	Error int
	Fatal int
}

// DefaultLevelColors is a struct of the default colors used.
var DefaultLevelColors = LevelColors{
	Debug: 10170623,
	Info:  3581519,
	Warn:  14327864,
	Error: 13631488,
	Fatal: 13631488,
}
