package discord

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/discord/message"
	"github.com/leighmacdonald/gbans/pkg/log"
)

var (
	ErrDiscordOverwriteCommands = errors.New("failed to bulk overwrite discord commands")
	ErrDiscordConfig            = errors.New("invalid config")
	ErrDiscordCreate            = errors.New("failed to connect to discord")
	ErrDiscordOpen              = errors.New("failed to open discord connection")
	ErrDiscordMessageSend       = errors.New("failed to send discord message")
	ErrDuplicateCommand         = errors.New("duplicate command registration")
)

type SlashCommandHandler func(ctx context.Context, s *discordgo.Session, m *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error)

type Discord struct {
	session         *discordgo.Session
	isReady         atomic.Bool
	commandHandlers map[string]SlashCommandHandler
	commands        []*discordgo.ApplicationCommand
	appID           string
	guildID         string
	externalURL     string
}

func NewDiscord(appID string, guildID string, token string, externalURL string) (*Discord, error) {
	if appID == "" || guildID == "" || token == "" {
		return nil, ErrDiscordConfig
	}

	session, errNewSession := discordgo.New("Bot " + token)
	if errNewSession != nil {
		return nil, errors.Join(errNewSession, ErrDiscordCreate)
	}

	session.UserAgent = "gbans (https://github.com/leighmacdonald/gbans)"
	session.Identify.Intents |= discordgo.IntentsGuildMessages
	session.Identify.Intents |= discordgo.IntentMessageContent
	session.Identify.Intents |= discordgo.IntentGuildMembers

	bot := &Discord{
		isReady:         atomic.Bool{},
		commandHandlers: map[string]SlashCommandHandler{},
		appID:           appID,
		guildID:         guildID,
		externalURL:     externalURL,
		session:         session,
	}

	session.AddHandler(bot.onReady)
	session.AddHandler(bot.onConnect)
	session.AddHandler(bot.onDisconnect)
	session.AddHandler(bot.onInteractionCreate)

	return bot, nil
}

func (bot *Discord) Start(_ context.Context) error {
	// Open a websocket connection to discord and begin listening.
	if errSessionOpen := bot.session.Open(); errSessionOpen != nil {
		return errors.Join(errSessionOpen, ErrDiscordOpen)
	}

	return nil
}

func (bot *Discord) MustRegisterHandler(cmd string, handler SlashCommandHandler, appCommand *discordgo.ApplicationCommand) {
	_, found := bot.commandHandlers[cmd]
	if found {
		panic(ErrDuplicateCommand)
	}
	for _, cmd := range bot.commands {
		if cmd.Name == appCommand.Name {
			panic(ErrDuplicateCommand)
		}
	}

	bot.commandHandlers[cmd] = handler
	bot.commands = append(bot.commands, appCommand)
}

func (bot *Discord) Shutdown() {
	if bot.session != nil {
		log.Closer(bot.session)
	}
}

func (bot *Discord) onReady(session *discordgo.Session, _ *discordgo.Ready) {
	slog.Info("Discord state changed", slog.String("state", "ready"), slog.String("username",
		fmt.Sprintf("%v#%v", session.State.User.Username, session.State.User.Discriminator)))
}

func (bot *Discord) onConnect(_ *discordgo.Session, _ *discordgo.Connect) {
	if errRegister := bot.botRegisterSlashCommands(bot.appID); errRegister != nil {
		slog.Error("Failed to register discord slash commands", log.ErrAttr(errRegister))
	}

	status := discordgo.UpdateStatusData{
		IdleSince: nil,
		Activities: []*discordgo.Activity{
			{
				Name:     "Cheeseburgers",
				Type:     discordgo.ActivityTypeCompeting,
				URL:      bot.externalURL,
				State:    "Nom Nom", //nolint:dupword
				Details:  "Blah",
				Instance: true,
				Flags:    1 << 0,
			},
			{
				Name:     "Hot Dogs",
				Type:     discordgo.ActivityTypeCompeting,
				URL:      bot.externalURL,
				State:    "Chomp Chomp", //nolint:dupword
				Details:  "Blah",
				Instance: true,
				Flags:    1 << 0,
			},
		},
		AFK:    false,
		Status: "https://github.com/leighmacdonald/gbans",
	}
	if errUpdateStatus := bot.session.UpdateStatusComplex(status); errUpdateStatus != nil {
		slog.Error("Failed to update status complex", log.ErrAttr(errUpdateStatus))
	}

	slog.Info("Discord state changed", slog.String("state", "connected"))

	bot.isReady.Store(true)
}

func (bot *Discord) onDisconnect(_ *discordgo.Session, _ *discordgo.Disconnect) {
	bot.isReady.Store(false)

	slog.Info("Discord state changed", slog.String("state", "disconnected"))
}

// onInteractionCreate is called when a user initiates an application command. All commands are sent
// through this interface.
// https://discord.com/developers/docs/interactions/receiving-and-responding#receiving-an-interaction
func (bot *Discord) onInteractionCreate(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	var (
		data    = interaction.ApplicationCommandData()
		command = data.Name
	)

	if handler, handlerFound := bot.commandHandlers[command]; handlerFound {
		// sendPreResponse should be called for any commands that call external services or otherwise
		// could not return a response instantly. discord will time out commands that don't respond within a
		// very short timeout windows, ~2-3 seconds.
		initialResponse := &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Calculating numberwang...",
			},
		}

		if errRespond := session.InteractionRespond(interaction.Interaction, initialResponse); errRespond != nil {
			if _, errFollow := session.FollowupMessageCreate(interaction.Interaction, true, &discordgo.WebhookParams{
				Content: errRespond.Error(),
			}); errFollow != nil {
				slog.Error("Failed sending error response for interaction", log.ErrAttr(errFollow))
			}

			return
		}

		commandCtx, cancelCommand := context.WithTimeout(context.TODO(), time.Second*30)
		defer cancelCommand()

		response, errHandleCommand := handler(commandCtx, session, interaction)
		if errHandleCommand != nil || response == nil {
			if _, errFollow := session.FollowupMessageCreate(interaction.Interaction, true, &discordgo.WebhookParams{
				Embeds: []*discordgo.MessageEmbed{message.ErrorMessage(command, errHandleCommand)},
			}); errFollow != nil {
				slog.Error("Failed sending error response for interaction", log.ErrAttr(errFollow))
			}

			return
		}

		if sendSendResponse := bot.sendInteractionResponse(session, interaction.Interaction, response); sendSendResponse != nil {
			slog.Error("Failed sending success response for interaction", log.ErrAttr(sendSendResponse))
		}
	}
}

func (bot *Discord) sendInteractionResponse(session *discordgo.Session, interaction *discordgo.Interaction, response *discordgo.MessageEmbed) error {
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
			return errors.Join(errResp, ErrDiscordMessageSend)
		}

		return nil
	}

	return nil
}

func (bot *Discord) SendPayload(channelID string, payload *discordgo.MessageEmbed) {
	if !bot.isReady.Load() {
		return
	}

	if _, errSend := bot.session.ChannelMessageSendEmbed(channelID, payload); errSend != nil {
		slog.Error("Failed to send discord payload", log.ErrAttr(errSend))
	}
}

//nolint:funlen,maintidx
func (bot *Discord) botRegisterSlashCommands(appID string) error {
	commands, errBulk := bot.session.ApplicationCommandBulkOverwrite(appID, bot.guildID, bot.commands)
	if errBulk != nil {
		return errors.Join(errBulk, ErrDiscordOverwriteCommands)
	}

	bot.commands = commands

	slog.Debug("Registered discord commands", slog.Int("count", len(commands)))

	return nil
}

type NullDiscordRepository struct{}

func (bot *NullDiscordRepository) RegisterHandler(_ string, _ SlashCommandHandler) error {
	return nil
}

func (bot *NullDiscordRepository) Shutdown() {
}

func (bot *NullDiscordRepository) Start() error {
	return nil
}

func (bot *NullDiscordRepository) SendPayload(_ string, _ *discordgo.MessageEmbed) {
}

func NewNullDiscordRepository() *NullDiscordRepository {
	return &NullDiscordRepository{}
}
