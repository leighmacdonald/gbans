package discord

import (
	"context"
	"errors"

	"github.com/bwmarrin/discordgo"
)

var (
	ErrCommandFailed = errors.New("command failed")           //nolint:gochecknoglobals
	ModPerms         = int64(discordgo.PermissionBanMembers)  //nolint:gochecknoglobals
	UserPerms        = int64(discordgo.PermissionViewChannel) //nolint:gochecknoglobals
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
	// it matches IDs in the various response types.
	MustRegisterPrefixHandler(prefix string, responder Handler)
}

// Discard implements a dummy service that can be used when discord bot support is disabled or for testing.
type Discard struct{}

func (d Discard) Send(_ string, _ *discordgo.MessageSend) error { return nil }
func (d Discard) Start(_ context.Context) error                 { return nil }
func (d Discard) Close()                                        {}
func (d Discard) MustRegisterCommandHandler(_ *discordgo.ApplicationCommand, _ Handler) {
}
func (d Discard) MustRegisterPrefixHandler(_ string, _ Handler) {}

type Config struct {
	Enabled                 bool   `json:"enabled"`
	BotEnabled              bool   `json:"bot_enabled"`
	IntegrationsEnabled     bool   `json:"integrations_enabled"`
	AppID                   string `json:"app_id"`
	AppSecret               string `json:"app_secret"`
	LinkID                  string `json:"link_id"`
	Token                   string `json:"token"`
	GuildID                 string `json:"guild_id"`
	PublicLogChannelEnable  bool   `json:"public_log_channel_enable"`
	LogChannelID            string `json:"log_channel_id"`
	PublicLogChannelID      string `json:"public_log_channel_id"`
	PublicMatchLogChannelID string `json:"public_match_log_channel_id"`
	VoteLogChannelID        string `json:"vote_log_channel_id"`
	AppealLogChannelID      string `json:"appeal_log_channel_id"`
	BanLogChannelID         string `json:"ban_log_channel_id"`
	ForumLogChannelID       string `json:"forum_log_channel_id"`
	KickLogChannelID        string `json:"kick_log_channel_id"`
	PlayerqueueChannelID    string `json:"playerqueue_channel_id"`
	ModPingRoleID           string `json:"mod_ping_role_id"`
	AnticheatChannelID      string `json:"anticheat_channel_id"`
}

func (c Config) SafePublicLogChannelID() string {
	if c.PublicLogChannelID != "" {
		return c.PublicLogChannelID
	}

	return c.LogChannelID
}

func (c Config) SafePublicMatchLogChannelID() string {
	if c.PublicMatchLogChannelID != "" {
		return c.PublicMatchLogChannelID
	}

	return c.LogChannelID
}

func (c Config) SafeVoteLogChannelID() string {
	if c.VoteLogChannelID != "" {
		return c.VoteLogChannelID
	}

	return c.LogChannelID
}

func (c Config) SafeAppealLogChannelID() string {
	if c.AppealLogChannelID != "" {
		return c.AppealLogChannelID
	}

	return c.LogChannelID
}

func (c Config) SafeBanLogChannelID() string {
	if c.BanLogChannelID != "" {
		return c.BanLogChannelID
	}

	return c.LogChannelID
}

func (c Config) SafeForumLogChannelID() string {
	if c.ForumLogChannelID != "" {
		return c.ForumLogChannelID
	}

	return c.LogChannelID
}

func (c Config) SafeKickLogChannelID() string {
	if c.KickLogChannelID != "" {
		return c.KickLogChannelID
	}

	return c.LogChannelID
}

func (c Config) SafePlayerqueueChannelID() string {
	if c.PlayerqueueChannelID != "" {
		return c.PlayerqueueChannelID
	}

	return c.LogChannelID
}

func (c Config) SafeModPingRoleID() string {
	if c.ModPingRoleID != "" {
		return c.ModPingRoleID
	}

	return c.LogChannelID
}

func (c Config) SafeAnticheatChannelID() string {
	if c.AnticheatChannelID != "" {
		return c.AnticheatChannelID
	}

	return c.LogChannelID
}

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

func NewMessageSend(components ...discordgo.MessageComponent) *discordgo.MessageSend {
	return &discordgo.MessageSend{
		Flags:      discordgo.MessageFlagsIsComponentsV2,
		Components: components,
	}
}
