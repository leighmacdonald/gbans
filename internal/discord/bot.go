package discord

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "github.com/joho/godotenv/autoload"
	"github.com/leighmacdonald/gbans/internal/ptr"
)

var (
	ErrConfig           = errors.New("configuration error")
	ErrCommandInvalid   = errors.New("command invalid")
	ErrSession          = errors.New("failed to start session")
	ErrCommandSend      = errors.New("failed to send response")
	ErrCommandDuplicate = errors.New("duplicate command")
)

const (
	IDSteamID = iota + 1
	IDCIDR
	IDReason
	IDDuration
	IDNotes
	IDUnbanReason
)

type CommandType int

const (
	CommandTypeCLI CommandType = iota
	CommandTypeModal
)

// Handler is a handler for responding to slash command interactions.
type Handler func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error)

// Responder is responsible for handling responses to Handler messages. This includes both modal data responses
// and button component handlers.
type Responder func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error

type Opts struct {
	// Token must be set to the discord bot toke, without any "Bot " prefix.
	Token string
	// AppID must be set to the bots application ID
	AppID string
	// GuildID should be set to the guildID of your main server. If unset, all commands are registered globally instead.
	GuildID string
	// Unregister UnregisterOnClose when true, will unregister all the previously registered commands on shutdown.
	UnregisterOnClose bool
	// UserAgent defines a optional custom user agent.
	UserAgent string
}

func New(opts Opts) (*Discord, error) {
	if opts.AppID == "" {
		return nil, fmt.Errorf("%w: invalid discord app id", ErrConfig)
	}

	if opts.Token == "" {
		return nil, fmt.Errorf("%w: invalid discord token", ErrConfig)
	}

	bot := &Discord{
		appID:            opts.AppID,
		guildID:          opts.GuildID,
		unregister:       opts.UnregisterOnClose,
		commandHandlers:  make(map[string]Handler),
		modalHandlers:    make(map[string]Handler),
		responseHandlers: make(map[string]Responder),
		buttonHandler:    make(map[string]Responder),
	}

	session, errSession := discordgo.New("Bot " + opts.Token)
	if errSession != nil {
		return nil, errors.Join(errSession, ErrConfig)
	}

	if opts.UserAgent != "" {
		session.UserAgent = opts.UserAgent
	} else {
		session.UserAgent = "discordgo-lipstick (https://github.com/leighmacdonald/discordgo-lipstick)"
	}

	session.Identify.Intents |= discordgo.IntentsGuildMessages
	session.Identify.Intents |= discordgo.IntentMessageContent
	session.Identify.Intents |= discordgo.IntentGuildMembers

	session.AddHandler(bot.onReady)
	session.AddHandler(bot.onConnect)
	session.AddHandler(bot.onDisconnect)
	session.AddHandler(bot.onInteractionCreate)

	bot.session = session

	return bot, nil
}

type Discord struct {
	appID              string
	guildID            string
	session            *discordgo.Session
	commandHandlers    map[string]Handler
	modalHandlers      map[string]Handler
	responseHandlers   map[string]Responder
	buttonHandler      map[string]Responder
	commands           []*discordgo.ApplicationCommand
	running            atomic.Bool
	registeredCommands []*discordgo.ApplicationCommand
	unregister         bool
}

func (b *Discord) SendNext(channelID string, payload *discordgo.MessageSend) error {
	if !b.running.Load() || b.session == nil {
		return nil
	}

	if _, errSend := b.session.ChannelMessageSendComplex(channelID, payload); errSend != nil {
		return errors.Join(errSend, ErrCommandSend)
	}

	return nil
}

func (b *Discord) Send(channelID string, payload *discordgo.MessageEmbed) error {
	if !b.running.Load() || b.session == nil {
		return nil
	}

	if _, errSend := b.session.ChannelMessageSendEmbed(channelID, payload); errSend != nil {
		return errors.Join(errSend, ErrCommandSend)
	}

	return nil
}

func (b *Discord) Start(_ context.Context) error {
	if b.running.Load() {
		return nil
	}

	b.running.Store(true)

	if errStart := b.session.Open(); errStart != nil {
		return errors.Join(errStart, ErrSession)
	}

	return nil
}

func (b *Discord) Close() {
	b.running.Store(false)

	if b.unregister {
		for _, cmd := range b.registeredCommands {
			if err := b.session.ApplicationCommandDelete(b.appID, b.guildID, cmd.ID); err != nil {
				slog.Error("Could not unregister command", slog.String("error", err.Error()), slog.String("name", cmd.Name))
			}
		}
	}

	if err := b.session.Close(); err != nil {
		slog.Error("failed to close discord session cleanly", slog.String("error", err.Error()))
	}
}

func (b *Discord) Session() *discordgo.Session {
	return b.session
}

func (b *Discord) MustRegisterButton(prefix string, responder Responder) {
	b.buttonHandler[prefix] = responder
}

// MustRegisterHandler handles registering a slash command, and associated handler.
// Calling this does not immediately register the command, but instead adds it to the list of
// commands that will be bulk registered upon connection.
func (b *Discord) MustRegisterHandler(cmdName string, appCommand *discordgo.ApplicationCommand, handler Handler,
	commandType CommandType, responseHandler ...Responder,
) {
	switch commandType {
	case CommandTypeCLI:
		_, found := b.commandHandlers[cmdName]
		if found {
			panic(ErrCommandDuplicate)
		}
		for _, cmd := range b.commands {
			if cmd.Name == appCommand.Name {
				panic(ErrCommandDuplicate)
			}
		}

		b.commandHandlers[cmdName] = handler
	case CommandTypeModal:
		if len(responseHandler) == 0 {
			panic("discord.CommandTypeModal handlers must also supply a response handler")
		}
		_, found := b.modalHandlers[cmdName]
		if found {
			panic(ErrCommandDuplicate)
		}
		for _, cmd := range b.commands {
			if cmd.Name == appCommand.Name {
				panic(ErrCommandDuplicate)
			}
		}

		b.modalHandlers[cmdName] = handler
	}
	if len(responseHandler) > 0 {
		b.responseHandlers[cmdName+"_resp"] = responseHandler[0]
	}
	b.commands = append(b.commands, appCommand)
}

func (b *Discord) onReady(session *discordgo.Session, _ *discordgo.Ready) {
	slog.Info("Logged in successfully", slog.String("name", session.State.User.Username),
		slog.String("discriminator", session.State.User.Discriminator))
}

func (b *Discord) onDisconnect(_ *discordgo.Session, _ *discordgo.Disconnect) {
	slog.Info("Discord state changed", slog.String("state", "disconnected"))
}

func (b *Discord) handleModalCmd(handler Handler, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	ctx, cancelCommand := context.WithTimeout(context.TODO(), time.Second*300)
	defer cancelCommand()

	if _, err := handler(ctx, session, interaction); err != nil {
		if _, errFollow := session.FollowupMessageCreate(interaction.Interaction, true, &discordgo.WebhookParams{
			Components: []discordgo.MessageComponent{
				discordgo.Container{
					Components: []discordgo.MessageComponent{
						discordgo.TextDisplay{
							Content: "test",
						},
					},
				},
			},
		}); errFollow != nil {
			slog.Error("Failed sending error response for interaction", slog.String("error", errFollow.Error()))
		}

		return
	}
}

func (b *Discord) handleCLICmd(handler Handler, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
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
			slog.Error("Failed sending error response for interaction", slog.String("error", errFollow.Error()))
		}

		return
	}

	commandCtx, cancelCommand := context.WithTimeout(context.Background(), time.Second*30)
	defer cancelCommand()

	response, errHandleCommand := handler(commandCtx, session, interaction)
	if errHandleCommand != nil || response == nil {
		if _, errFollow := session.FollowupMessageCreate(interaction.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{{Title: "Error", Description: errHandleCommand.Error()}},
		}); errFollow != nil {
			slog.Error("Failed sending error response for interaction", slog.String("error", errFollow.Error()))
		}

		return
	}

	if sendSendResponse := b.sendInteractionResponse(session, interaction.Interaction, response); sendSendResponse != nil {
		slog.Error("Failed sending success response for interaction", slog.String("error", sendSendResponse.Error()))
	}
}

func (b *Discord) onAppCommand(_ context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	command := interaction.ApplicationCommandData().Name
	if handler, handlerFound := b.modalHandlers[command]; handlerFound {
		b.handleModalCmd(handler, session, interaction)

		return
	}
	if handler, handlerFound := b.commandHandlers[command]; handlerFound {
		b.handleCLICmd(handler, session, interaction)
	}

	slog.Error("Got unknown discord command", slog.String("command", command))
}

// onModalSubmit handles responding to modal interaction.
func (b *Discord) onModalSubmit(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	data := interaction.ModalSubmitData()
	var handler Responder
	for commandPrefix, responseHandler := range b.responseHandlers {
		if strings.HasPrefix(data.CustomID, commandPrefix) {
			handler = responseHandler

			break
		}
	}

	if handler == nil {
		slog.Error("No modal submit handler found for command", slog.String("command", data.CustomID))

		return
	}

	//if err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
	//	Type: discordgo.InteractionResponsePong,
	//	Data: &discordgo.InteractionResponseData{
	//		//Content: "Calculating numberwang...",
	//		Components: []discordgo.MessageComponent{
	//			discordgo.Container{
	//				Components: []discordgo.MessageComponent{
	//					discordgo.TextDisplay{
	//						Content: "test",
	//					},
	//				},
	//			},
	//		},
	//	},
	//}); err != nil {
	//	slog.Error("Failed sending error response for interaction", slog.String("error", err.Error()))
	//}

	if errHandler := handler(ctx, session, interaction); errHandler != nil {
		slog.Error("Failed sending error response for handler", slog.String("error", errHandler.Error()))
		if _, errFollow := session.InteractionResponseEdit(interaction.Interaction, &discordgo.WebhookEdit{
			Components: &[]discordgo.MessageComponent{
				discordgo.Container{
					AccentColor: ptr.To(ColourError),
					Components: []discordgo.MessageComponent{
						discordgo.TextDisplay{
							Content: fmt.Sprintf("Command Failed\n\n```%s```", errHandler.Error()),
						},
					},
				},
			},
		}); errFollow != nil {
			slog.Error("Failed sending error response for interaction", slog.String("error", errFollow.Error()))
		}

		return
	}
}

func (b *Discord) onButton(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	var (
		data    = interaction.MessageComponentData()
		handler Responder
	)
	for prefix, buttonHandler := range b.buttonHandler {
		if strings.HasPrefix(data.CustomID, prefix) {
			handler = buttonHandler

			break
		}
	}

	if handler == nil {
		slog.Info("No button handler found for command", slog.String("command", data.CustomID))

		return
	}

	if err := handler(ctx, session, interaction); err != nil {
		slog.Error("Failed sending error response for interaction", slog.String("error", err.Error()))
	}
}

func (b *Discord) onInteractionCreate(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	switch interaction.Type {
	case discordgo.InteractionApplicationCommand:
		b.onAppCommand(ctx, session, interaction)
	case discordgo.InteractionMessageComponent:
		b.onButton(ctx, session, interaction)
	case discordgo.InteractionApplicationCommandAutocomplete:
	case discordgo.InteractionModalSubmit:
		b.onModalSubmit(ctx, session, interaction)
	}
}

func (b *Discord) sendInteractionResponse(session *discordgo.Session, interaction *discordgo.Interaction, response *discordgo.MessageEmbed) error {
	resp := &discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{response},
	}

	_, errResponseErr := session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{
		Embeds: &resp.Embeds,
	})

	if errResponseErr != nil {
		if _, errResp := session.FollowupMessageCreate(interaction, true, &discordgo.WebhookParams{
			Content: "Something went wrong: " + errResponseErr.Error(),
		}); errResp != nil {
			return errors.Join(errResp, ErrCommandSend)
		}

		return nil
	}

	return nil
}

func (b *Discord) onConnect(_ *discordgo.Session, _ *discordgo.Connect) {
	slog.Info("Discord state changed", slog.String("state", "connected"))

	if errRegister := b.overwriteCommands(); errRegister != nil {
		slog.Error("Failed to register discord slash commands", slog.String("error", errRegister.Error()))
	}
}

func (b *Discord) overwriteCommands() error {
	// When guildID is empty, it registers the commands globally instead of per guild.
	commands, errBulk := b.session.ApplicationCommandBulkOverwrite(b.appID, b.guildID, b.commands)
	if errBulk != nil {
		return errors.Join(errBulk, ErrCommandInvalid)
	}

	b.registeredCommands = commands

	return nil
}

type CommandOptions map[string]*discordgo.ApplicationCommandInteractionDataOption

func (opts CommandOptions) String(key string) string {
	root, found := opts[key]
	if !found {
		return ""
	}

	val, ok := root.Value.(string)
	if !ok {
		return ""
	}

	return val
}
