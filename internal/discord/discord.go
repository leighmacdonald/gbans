package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	ErrCommandFailed = errors.New("Command failed")
	ErrTooLarge      = errors.Errorf("Max message length is %d", discordMaxMsgLen)
)

type Bot struct {
	log               *zap.Logger
	session           *discordgo.Session
	isReady           bool
	onConnectUser     func()
	onDisconnectUser  func()
	commandHandlers   map[Cmd]CommandHandler
	ColourLevels      LevelColors
	unregisterOnStart bool
	appId             string
	extURL            string
}

func New(logger *zap.Logger, token string, appID string, unregisterOnStart bool, extURL string) (*Bot, error) {
	// Immediately connects
	session, errNewSession := discordgo.New("Bot " + token)
	if errNewSession != nil {
		return nil, errors.Wrapf(errNewSession, "Failed to connect to discord. discord unavailable")
	}

	session.UserAgent = "gbans (https://github.com/leighmacdonald/gbans)"
	session.Identify.Intents |= discordgo.IntentsGuildMessages
	session.Identify.Intents |= discordgo.IntentMessageContent
	session.Identify.Intents |= discordgo.IntentGuildMembers
	bot := &Bot{
		log:               logger.Named("discord"),
		session:           session,
		isReady:           false,
		unregisterOnStart: unregisterOnStart,
		appId:             appID,
		extURL:            extURL,
		commandHandlers:   map[Cmd]CommandHandler{},
		ColourLevels: LevelColors{
			Debug: 10170623,
			Info:  3581519,
			Warn:  14327864,
			Error: 13631488,
			Fatal: 13631488,
		},
	}
	bot.session.AddHandler(bot.onReady)
	bot.session.AddHandler(bot.onConnect)
	bot.session.AddHandler(bot.onDisconnect)
	bot.session.AddHandler(bot.onInteractionCreate)

	return bot, nil
}

func (bot *Bot) SetOnConnect(fn func()) {
	bot.onConnectUser = fn
}

func (bot *Bot) SetOnDisconnect(fn func()) {
	bot.onDisconnectUser = fn
}

func (bot *Bot) RegisterHandler(cmd Cmd, handler CommandHandler) error {
	_, found := bot.commandHandlers[cmd]
	if found {
		return errors.New("Duplicate command")
	}

	bot.commandHandlers[cmd] = handler

	return nil
}

func (bot *Bot) Shutdown(guildID string) {
	if bot.session != nil {
		defer util.LogCloser(bot.session, bot.log)
		bot.botUnregisterSlashCommands(guildID)
	}
}

func (bot *Bot) botUnregisterSlashCommands(guildID string) {
	registeredCommands, err := bot.session.ApplicationCommands(bot.session.State.User.ID, guildID)
	if err != nil {
		bot.log.Error("Could not fetch registered commands", zap.Error(err))

		return
	}

	for _, v := range registeredCommands {
		if errDel := bot.session.ApplicationCommandDelete(bot.session.State.User.ID, guildID, v.ID); errDel != nil {
			bot.log.Error("Cannot delete command", zap.String("name", v.Name), zap.Error(err))

			return
		}
	}

	bot.log.Info("Unregistered discord commands", zap.Int("count", len(registeredCommands)))
}

func (bot *Bot) Start() error {
	// Open a websocket connection to discord and begin listening.
	if errSessionOpen := bot.session.Open(); errSessionOpen != nil {
		return errors.Wrap(errSessionOpen, "Error opening discord connection")
	}

	if bot.unregisterOnStart {
		bot.botUnregisterSlashCommands("")
	}

	if errRegister := bot.botRegisterSlashCommands(bot.appId); errRegister != nil {
		bot.log.Error("Failed to register discord slash commands", zap.Error(errRegister))
	}

	return nil
}

func (bot *Bot) onReady(session *discordgo.Session, _ *discordgo.Ready) {
	bot.log.Info("Service state changed", zap.String("state", "ready"), zap.String("username",
		fmt.Sprintf("%v#%v", session.State.User.Username, session.State.User.Discriminator)))

	bot.isReady = true
}

func (bot *Bot) onConnect(_ *discordgo.Session, _ *discordgo.Connect) {
	status := discordgo.UpdateStatusData{
		IdleSince: nil,
		Activities: []*discordgo.Activity{
			{
				Name:     "Cheeseburgers",
				Type:     discordgo.ActivityTypeListening,
				URL:      bot.extURL,
				State:    "state field",
				Details:  "Blah",
				Instance: true,
				Flags:    1 << 0,
			},
		},
		AFK:    false,
		Status: "https://github.com/leighmacdonald/gbans",
	}
	if errUpdateStatus := bot.session.UpdateStatusComplex(status); errUpdateStatus != nil {
		bot.log.Error("Failed to update status complex", zap.Error(errUpdateStatus))
	}

	bot.log.Info("Service state changed", zap.String("state", "connected"))

	if bot.onConnectUser != nil {
		bot.onConnectUser()
	}
}

func (bot *Bot) onDisconnect(_ *discordgo.Session, _ *discordgo.Disconnect) {
	bot.isReady = false

	bot.log.Info("Service state changed", zap.String("state", "disconnected"))

	if bot.onDisconnectUser != nil {
		bot.onDisconnectUser()
	}
}

// func sendChannelMessage(session *discordgo.Session, channelId string, msg string, wrap bool) error {
//	if !isReady.Load() {
//		log.Error("Tried to send message to disconnected client")
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

func (bot *Bot) sendInteractionResponse(session *discordgo.Session, interaction *discordgo.Interaction, response Response) error {
	resp := &discordgo.InteractionResponseData{
		Content: "hi",
	}

	switch response.MsgType {
	case MtString:
		val, ok := response.Value.(string)
		if ok && val != "" {
			resp.Content = val
			if len(resp.Content) > discordMaxMsgLen {
				return ErrTooLarge
			}
		}
	case MtEmbed:
		embed, ok := response.Value.(*discordgo.MessageEmbed)
		if !ok {
			return errors.New("Failed to cast MessageEmbed")
		}

		resp.Embeds = append(resp.Embeds, embed)
	}

	_, errResponseErr := session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{
		Embeds: &resp.Embeds,
	})

	if errResponseErr != nil {
		if _, errResp := session.FollowupMessageCreate(interaction, true, &discordgo.WebhookParams{
			Content: "Something went wrong",
		}); errResp != nil {
			return errors.Wrap(errResp, "Failed to send error response")
		}

		return nil
	}

	return nil
}

func (bot *Bot) SendPayload(payload Payload) {
	if !bot.isReady {
		return
	}

	if _, errSend := bot.session.ChannelMessageSendEmbed(payload.ChannelID, payload.Embed); errSend != nil {
		bot.log.Error("Failed to send discord payload", zap.Error(errSend))
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
