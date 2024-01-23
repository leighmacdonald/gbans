package app

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/pkg/util"
	"go.uber.org/zap"
)

var (
	ErrCommandFailed     = errors.New("command failed")
	ErrDiscordCreate     = errors.New("failed to connect to discord")
	ErrDiscordOpen       = errors.New("failed to open discord connection")
	ErrDuplicateCommand  = errors.New("duplicate command registration")
	ErrDiscordMessageSen = errors.New("failed to send discord message")
)

type SlashCommandHandler func(ctx context.Context, s *discordgo.Session, m *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error)

type Bot struct {
	log               *zap.Logger
	session           *discordgo.Session
	isReady           atomic.Bool
	commandHandlers   map[Cmd]SlashCommandHandler
	unregisterOnStart bool
	appID             string
	extURL            string
}

type Payload struct {
	ChannelID string
	Embed     *discordgo.MessageEmbed
}

func NewDiscord(logger *zap.Logger, conf config.Config) (*Bot, error) {
	// Immediately connects
	session, errNewSession := discordgo.New("Bot " + conf.Discord.Token)
	if errNewSession != nil {
		return nil, errors.Join(errNewSession, ErrDiscordCreate)
	}

	session.UserAgent = "gbans (https://github.com/leighmacdonald/gbans)"
	session.Identify.Intents |= discordgo.IntentsGuildMessages
	session.Identify.Intents |= discordgo.IntentMessageContent
	session.Identify.Intents |= discordgo.IntentGuildMembers
	bot := &Bot{
		log:               logger.Named("discord"),
		session:           session,
		isReady:           atomic.Bool{},
		unregisterOnStart: conf.Discord.UnregisterOnStart,
		appID:             conf.Discord.AppID,
		extURL:            conf.General.ExternalURL,
		commandHandlers:   map[Cmd]SlashCommandHandler{},
	}
	bot.session.AddHandler(bot.onReady)
	bot.session.AddHandler(bot.onConnect)
	bot.session.AddHandler(bot.onDisconnect)
	bot.session.AddHandler(bot.onInteractionCreate)

	return bot, nil
}

func (bot *Bot) RegisterHandler(cmd Cmd, handler SlashCommandHandler) error {
	_, found := bot.commandHandlers[cmd]
	if found {
		return ErrDuplicateCommand
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
		return errors.Join(errSessionOpen, ErrDiscordOpen)
	}

	if bot.unregisterOnStart {
		bot.botUnregisterSlashCommands("")
	}

	return nil
}

func (bot *Bot) onReady(session *discordgo.Session, _ *discordgo.Ready) {
	bot.log.Info("Service state changed", zap.String("state", "ready"), zap.String("username",
		fmt.Sprintf("%v#%v", session.State.User.Username, session.State.User.Discriminator)))
}

func (bot *Bot) onConnect(_ *discordgo.Session, _ *discordgo.Connect) {
	if errRegister := bot.botRegisterSlashCommands(bot.appID); errRegister != nil {
		bot.log.Error("Failed to register discord slash commands", zap.Error(errRegister))
	}

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

	bot.isReady.Store(true)
}

func (bot *Bot) onDisconnect(_ *discordgo.Session, _ *discordgo.Disconnect) {
	bot.isReady.Store(false)

	bot.log.Info("Service state changed", zap.String("state", "disconnected"))
}

func (bot *Bot) sendInteractionResponse(session *discordgo.Session, interaction *discordgo.Interaction, response *discordgo.MessageEmbed) error {
	resp := &discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{response},
	}

	_, errResponseErr := session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{
		Embeds: &resp.Embeds,
	})

	if errResponseErr != nil {
		if _, errResp := session.FollowupMessageCreate(interaction, true, &discordgo.WebhookParams{
			Content: "Something went wrong",
		}); errResp != nil {
			return errors.Join(errResp, ErrDiscordMessageSen)
		}

		return nil
	}

	return nil
}

func (bot *Bot) SendPayload(channelID string, payload *discordgo.MessageEmbed) {
	if !bot.isReady.Load() {
		return
	}

	if _, errSend := bot.session.ChannelMessageSendEmbed(channelID, payload); errSend != nil {
		bot.log.Error("Failed to send discord payload", zap.Error(errSend))
	}
}
