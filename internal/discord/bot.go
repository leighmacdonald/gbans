package discord

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/ptr"
)

var (
	ErrConfig           = errors.New("configuration error")
	ErrCommandInvalid   = errors.New("command invalid")
	ErrCustomIDInvalid  = errors.New("custom_id invalid")
	ErrSession          = errors.New("failed to start session")
	ErrCommandSend      = errors.New("failed to send response")
	ErrCommandDuplicate = errors.New("duplicate command")
	ErrTemplate         = errors.New("template error")
	ErrCommandFailed    = errors.New("command failed")
	ErrRole             = errors.New("failed to create/fetch roles")
)

const (
	ColourSuccess = 302673
	ColourInfo    = 3581519
	ColourWarn    = 14327864
	ColourError   = 13631488
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
	templates          map[string]*template.Template
	mutex              *sync.RWMutex
}

func New(opts Opts) (*Discord, error) {
	if opts.AppID == "" {
		return nil, fmt.Errorf("%w: invalid discord app id", ErrConfig)
	}

	if opts.Token == "" {
		return nil, fmt.Errorf("%w: invalid discord token", ErrConfig)
	}

	bot := &Discord{
		mutex:           &sync.RWMutex{},
		appID:           opts.AppID,
		guildID:         opts.GuildID,
		unregister:      opts.UnregisterOnClose,
		commandHandlers: make(map[string]Handler),
		prefixHandlers:  make(map[string]Handler),
		templates:       make(map[string]*template.Template),
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

func (b *Discord) MustRegisterTemplate(namespace string, body []byte) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	textTemplate, errParse := template.New(namespace).Parse(string(body))
	if errParse != nil {
		panic(errParse)
	}

	b.templates[namespace] = textTemplate
}

func (b *Discord) RenderTemplate(namespace string, name string, args any) (string, error) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	textTemplate, found := b.templates[namespace]
	if !found {
		return "", fmt.Errorf("%w: unknown template namespace %s", ErrTemplate, namespace)
	}
	var outBuff bytes.Buffer
	if errExec := textTemplate.ExecuteTemplate(&outBuff, name, args); errExec != nil {
		return "", errors.Join(errExec, ErrCommandFailed)
	}

	return outBuff.String(), nil
}

func (b *Discord) Send(channelID string, payload *discordgo.MessageSend) error {
	if !b.running.Load() || b.session == nil {
		return nil
	}

	if _, errSend := b.session.ChannelMessageSendComplex(channelID, payload); errSend != nil {
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

func (b *Discord) Roles() ([]*discordgo.Role, error) {
	roles, errRoles := b.session.GuildRoles(b.guildID)
	if errRoles != nil {
		return nil, errors.Join(errRoles, ErrRole)
	}

	return roles, errors.Join(errRoles, ErrRole)
}

func (b *Discord) CreateRole(name string) (string, error) {
	role, errRole := b.session.GuildRoleCreate(b.guildID, &discordgo.RoleParams{
		Name:        name,
		Mentionable: ptr.To(true),
	})
	if errRole != nil {
		return "", errors.Join(errRole, ErrRole)
	}

	return role.ID, nil
}

func (b *Discord) onReady(session *discordgo.Session, _ *discordgo.Ready) {
	slog.Info("Logged in successfully", slog.String("name", session.State.User.Username),
		slog.String("discriminator", session.State.User.Discriminator))
}

func (b *Discord) onDisconnect(_ *discordgo.Session, _ *discordgo.Disconnect) {
	slog.Info("Discord state changed", slog.String("state", "disconnected"))
}

func (b *Discord) onAppCommand(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	command := interaction.ApplicationCommandData().Name
	handler, handlerFound := b.commandHandlers[command]
	if !handlerFound {
		return
	}
	if errHandleCommand := handler(ctx, session, interaction); errHandleCommand != nil {
		slog.Error("Failed handling command",
			slog.String("command", command),
			slog.String("error", errHandleCommand.Error()))
	}
}

func (b *Discord) findAndExecPrefixHandler(ctx context.Context, handlerName string, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	for prefix, handler := range b.prefixHandlers {
		if !strings.HasPrefix(handlerName, prefix) {
			continue
		}

		if err := handler(ctx, session, interaction); err != nil {
			Error(session, interaction, err)
		}

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
		b.onAutoComplete(ctx, session, interaction)
	case discordgo.InteractionModalSubmit:
		b.findAndExecPrefixHandler(ctx, interaction.ModalSubmitData().CustomID, session, interaction)
	}
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

func (b *Discord) onAutoComplete(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	data := interaction.ApplicationCommandData()
	if len(data.Options) == 0 || data.Options[0].Name == "" {
		return
	}

	b.findAndExecPrefixHandler(ctx, data.Options[0].Name, session, interaction)
}
