package discord

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "github.com/joho/godotenv/autoload"
	"github.com/leighmacdonald/gbans/internal/ptr"
)

var (
	ErrConfig           = errors.New("configuration error")
	ErrCommandInvalid   = errors.New("command invalid")
	ErrCustomIDInvalid  = errors.New("custom_id invalid")
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
	IDBody
)

// Handler is a handler for responding to slash command interactions.
type Handler func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error

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

type Discord struct {
	appID              string
	guildID            string
	session            *discordgo.Session
	commandHandlers    map[string]Handler
	prefixHandlers     map[string]Handler
	commands           []*discordgo.ApplicationCommand
	running            atomic.Bool
	registeredCommands []*discordgo.ApplicationCommand
	unregister         bool
}

func New(opts Opts) (*Discord, error) {
	if opts.AppID == "" {
		return nil, fmt.Errorf("%w: invalid discord app id", ErrConfig)
	}

	if opts.Token == "" {
		return nil, fmt.Errorf("%w: invalid discord token", ErrConfig)
	}

	bot := &Discord{
		appID:           opts.AppID,
		guildID:         opts.GuildID,
		unregister:      opts.UnregisterOnClose,
		commandHandlers: make(map[string]Handler),
		prefixHandlers:  make(map[string]Handler),
	}

	session, errSession := discordgo.New("Bot " + opts.Token)
	if errSession != nil {
		return nil, errors.Join(errSession, ErrConfig)
	}

	if opts.UserAgent != "" {
		session.UserAgent = opts.UserAgent
	} else {
		session.UserAgent = "gbans (https://github.com/leighmacdonald/gbans)"
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

// MustRegisterPrefixHandler takes a prefix and handler to execute when the prefix is matched. The prefix
// is defined ban_unban_button_resp_.
func (b *Discord) MustRegisterPrefixHandler(prefix string, handler Handler) {
	_, found := b.prefixHandlers[prefix]
	if found {
		panic(ErrCommandDuplicate)
	}

	b.prefixHandlers[prefix] = handler
}

// MustRegisterCommandHandler handles registering a slash command, and associated handler.
// Calling this does not immediately register the command, but instead adds it to the list of
// commands that will be bulk registered upon connection.
func (b *Discord) MustRegisterCommandHandler(appCommand *discordgo.ApplicationCommand, handler Handler) {
	_, found := b.commandHandlers[appCommand.Name]
	if found {
		panic(ErrCommandDuplicate)
	}
	for _, cmd := range b.commands {
		if cmd.Name == appCommand.Name {
			panic(ErrCommandDuplicate)
		}
	}

	b.commandHandlers[appCommand.Name] = handler

	b.commands = append(b.commands, appCommand)
}

func (b *Discord) onReady(session *discordgo.Session, _ *discordgo.Ready) {
	slog.Info("Logged in successfully", slog.String("name", session.State.User.Username),
		slog.String("discriminator", session.State.User.Discriminator))
}

func (b *Discord) onDisconnect(_ *discordgo.Session, _ *discordgo.Disconnect) {
	slog.Info("Discord state changed", slog.String("state", "disconnected"))
}

func (b *Discord) handleModalCmd(ctx context.Context, handler Handler, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	if err := handler(ctx, session, interaction); err != nil {
		slog.Error("Failed to handle modal command", slog.String("error", err.Error()))

		if errFollow := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsIsComponentsV2 | discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{
					discordgo.Container{
						AccentColor: ptr.To(ColourError),
						Components: []discordgo.MessageComponent{
							discordgo.TextDisplay{
								Content: "Error executing command",
							},
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

func (b *Discord) handleCLICmd(ctx context.Context, handler Handler, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	errHandleCommand := handler(ctx, session, interaction)
	if errHandleCommand != nil {
		//_, _ = session.FollowupMessageCreate(interaction.Interaction, true, &discordgo.WebhookParams{
		//	Content: "error handling command",
		//})
		slog.Error("Failed handling command", slog.String("error", errHandleCommand.Error()))

		return
	}
}

func (b *Discord) onAppCommand(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	command := interaction.ApplicationCommandData().Name
	if handler, handlerFound := b.commandHandlers[command]; handlerFound {
		b.handleCLICmd(ctx, handler, session, interaction)

		return
	}
}

func (b *Discord) findAndExecPrefixHandler(ctx context.Context, handlerName string, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	for prefix, handler := range b.prefixHandlers {
		if !strings.HasPrefix(handlerName, prefix) {
			continue
		}

		b.handleModalCmd(ctx, handler, session, interaction)

		return
	}

	slog.Error("Got unknown discord command", slog.String("command", handlerName))
}

func (b *Discord) onInteractionCreate(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	switch interaction.Type {
	case discordgo.InteractionApplicationCommand:
		b.onAppCommand(ctx, session, interaction)
	case discordgo.InteractionMessageComponent:
		b.findAndExecPrefixHandler(ctx, interaction.MessageComponentData().CustomID, session, interaction)
	case discordgo.InteractionApplicationCommandAutocomplete:
	case discordgo.InteractionModalSubmit:
		b.findAndExecPrefixHandler(ctx, interaction.ModalSubmitData().CustomID, session, interaction)
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

func Render(name string, templ string, context any) (string, error) {
	var b bytes.Buffer
	tmpl, err := template.New(name).Parse(templ)
	if err != nil {
		return "", err
	}
	if err = tmpl.Execute(&b, context); err != nil {
		return "", err
	}

	return b.String(), nil
}

// CustomIDInt64 pulls out the suffix value as a int64.
// eg: ban_unban_button_resp_1234 -> 1234
func CustomIDInt64(idString string) (int64, error) {
	parts := strings.Split(idString, "_")
	if len(parts) < 2 {
		return 0, ErrCustomIDInvalid
	}
	value, errID := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if errID != nil {
		return 0, errID
	}

	return value, nil
}

// AckInteraction acknowledges the interation immediately. It should be followed up by
// an RespondInteraction to complete the response.
func AckInteraction(session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	return session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsIsComponentsV2,
			Components: []discordgo.MessageComponent{
				discordgo.TextDisplay{Content: "Computering..."},
			},
		},
	})
}

func RespondInteraction(session *discordgo.Session, interaction *discordgo.InteractionCreate, components ...discordgo.MessageComponent) error {
	_, err := session.InteractionResponseEdit(interaction.Interaction, &discordgo.WebhookEdit{
		Flags:           discordgo.MessageFlagsIsComponentsV2 | discordgo.MessageFlagsSuppressNotifications,
		AllowedMentions: &discordgo.MessageAllowedMentions{},
		Components:      &components,
	})

	return err
}
