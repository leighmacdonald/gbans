package discord

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/ptr"
)

const (
	ModPerms  = int64(discordgo.PermissionBanMembers)
	UserPerms = int64(discordgo.PermissionViewChannel)
)

const (
	iconURL      = "https://cdn.discordapp.com/avatars/758536119397646370/6a371d1a481a72c512244ba9853f7eff.webp?size=128"
	providerName = "gbans"
)

// Service provides a interface for controlling the discord backend.
type Service interface {
	// Send handles sending messages to a channel.
	Send(channelID string, message *discordgo.MessageSend) error

	// Start initiates the bot service. This is a blocking call.
	Start(ctx context.Context) error

	// Close the bot session.
	Close()

	// MustRegisterCommandHandler allows the caller to register discord slash commands.
	// When using discord.CommandTypeModal, the responder must be defined. It will be called when responding
	// to the modal data submission.
	MustRegisterCommandHandler(command *discordgo.ApplicationCommand, handler Handler)

	// MustRegisterPrefixHandler is similar to MustRegisterCommandHandler, however instead of exact command names
	// it matches IDs in the various response types.co
	MustRegisterPrefixHandler(prefix string, responder Handler)

	// CreateRole handles creating a new role within the guild.
	CreateRole(name string) (string, error)

	// Roles returns a slice of all roles within the guild.
	Roles() ([]*discordgo.Role, error)

	MustRegisterTemplate(namespace string, body []byte)
	RenderTemplate(namespace string, name string, args any) (string, error)
}

// Discard implements a dummy service that can be used when discord bot support is disabled or for testing.
type Discard struct{}

func (d Discard) Send(_ string, _ *discordgo.MessageSend) error { return nil }
func (d Discard) Start(_ context.Context) error                 { return nil }
func (d Discard) Close()                                        {}
func (d Discard) MustRegisterCommandHandler(_ *discordgo.ApplicationCommand, _ Handler) {
}
func (d Discard) MustRegisterPrefixHandler(_ string, _ Handler)            {}
func (d Discard) CreateRole(_ string) (string, error)                      { return "", nil }
func (d Discard) Roles() ([]*discordgo.Role, error)                        { return nil, nil }
func (d Discard) MustRegisterTemplate(string, []byte)                      {}
func (d Discard) RenderTemplate(_ string, _ string, _ any) (string, error) { return "", nil }

const (
	OptUserIdentifier   = "user_identifier"
	OptServerIdentifier = "server_identifier"
	OptMessage          = "message"
	OptPattern          = "pattern"
	OptIsRegex          = "is_regex"
)

// OptionMap will take the recursive discord slash commands and flatten them into a simple
// map.
func OptionMap(options []*discordgo.ApplicationCommandInteractionDataOption) CommandOptions {
	optionM := make(CommandOptions, len(options))
	for _, opt := range options {
		optionM[opt.Name] = opt
	}

	return optionM
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

// CustomIDInt64 pulls out the suffix value as a int64.
// eg: ban_unban_button_resp_1234 -> 1234
func CustomIDInt64(idString string) (int64, error) {
	parts := strings.Split(idString, "_")
	if len(parts) < 2 {
		return 0, ErrCustomIDInvalid
	}
	value, errID := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if errID != nil {
		return 0, errors.Join(errID, ErrCustomIDInvalid)
	}

	return value, nil
}

// AckInteraction acknowledges the interation immediately. It should be followed up by
// an RespondUpdate to complete the response.
func AckInteraction(session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	if err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:      discordgo.MessageFlagsIsComponentsV2,
			Components: []discordgo.MessageComponent{discordgo.TextDisplay{Content: "Computering..."}},
		},
	}); err != nil {
		return errors.Join(err, ErrCommandSend)
	}

	return nil
}

func RespondModal(session *discordgo.Session, interaction *discordgo.InteractionCreate, cid string, title string, components ...discordgo.MessageComponent) error {
	if err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   cid,
			Title:      title,
			Flags:      discordgo.MessageFlagsIsComponentsV2 | discordgo.MessageFlagsEphemeral,
			Components: components,
		},
	}); err != nil {
		return errors.Join(err, ErrCommandSend)
	}

	return nil
}

func Respond(session *discordgo.Session, interaction *discordgo.InteractionCreate, components ...discordgo.MessageComponent) error {
	if err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:      discordgo.MessageFlagsIsComponentsV2,
			Components: components,
		},
	}); err != nil {
		return errors.Join(err, ErrCommandSend)
	}

	return nil
}

func RespondPrivate(session *discordgo.Session, interaction *discordgo.InteractionCreate, components ...discordgo.MessageComponent) error {
	if err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:      discordgo.MessageFlagsIsComponentsV2 | discordgo.MessageFlagsEphemeral,
			Components: components,
		},
	}); err != nil {
		return errors.Join(err, ErrCommandSend)
	}

	return nil
}

// RespondUpdate handles updaing a previous acknowledged interaction with a new set of components.
func RespondUpdate(session *discordgo.Session, interaction *discordgo.InteractionCreate, components ...discordgo.MessageComponent) error {
	if _, err := session.InteractionResponseEdit(interaction.Interaction, &discordgo.WebhookEdit{
		Flags:           discordgo.MessageFlagsIsComponentsV2 | discordgo.MessageFlagsSuppressNotifications,
		AllowedMentions: &discordgo.MessageAllowedMentions{},
		Components:      &components,
	}); err != nil {
		return errors.Join(err, ErrCommandSend)
	}

	return nil
}

// Error is responsible for responding with a generic error message format.
func Error(session *discordgo.Session, interaction *discordgo.InteractionCreate, err error) {
	if errResponse := RespondPrivate(session, interaction, discordgo.Container{
		AccentColor: ptr.To(ColourError),
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: fmt.Sprintf(`🚨 Command Error 🚨

    %s
`, err.Error())},
		},
	}); errResponse != nil {
		slog.Error("Failed to send error response", slog.String("error", errResponse.Error()))
	}
}

// Success sends a generic success response.
func Success(session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	return SuccessMsg(session, interaction, "✨ Command Successful ✨")
}

// SuccessMsg sends a success response with a custom message.
func SuccessMsg(session *discordgo.Session, interaction *discordgo.InteractionCreate, msg string) error {
	return RespondPrivate(session, interaction, discordgo.Container{
		AccentColor: ptr.To(ColourSuccess),
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: msg},
		},
	})
}

func NewMessage(components ...discordgo.MessageComponent) *discordgo.MessageSend {
	return &discordgo.MessageSend{
		Flags:      discordgo.MessageFlagsIsComponentsV2,
		Components: components,
	}
}
