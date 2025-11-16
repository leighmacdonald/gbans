package discord

import (
	"context"
	"errors"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/discordgo-lipstick/bot"
)

var (
	ErrCommandFailed = errors.New("command failed")
	DmPerms          = false                                  //nolint:gochecknoglobals
	ModPerms         = int64(discordgo.PermissionBanMembers)  //nolint:gochecknoglobals
	UserPerms        = int64(discordgo.PermissionViewChannel) //nolint:gochecknoglobals
)

// Service provides a interface for controlling the discord backend.
type Service interface {
	// Send handles sending messages to a channel.
	Send(channelID string, payload *discordgo.MessageEmbed) error

	// Start initiates the bot service. This is a blocking call.
	Start(ctx context.Context) error

	// Close the bot session.
	Close()

	// MustRegisterHandler allows the caller to register discord slash commands.
	MustRegisterHandler(cmd string, appCommand *discordgo.ApplicationCommand, handler bot.Handler)

	// Session returns the underlying discordgo session.
	Session() *discordgo.Session
}

// Discard implements a dummy service that can be used when discord bot support is disabled.
type Discard struct{}

func (d Discard) Send(_ string, _ *discordgo.MessageEmbed) error                               { return nil }
func (d Discard) Start(_ context.Context) error                                                { return nil }
func (d Discard) Session() *discordgo.Session                                                  { return nil }
func (d Discard) Close()                                                                       {}
func (d Discard) MustRegisterHandler(_ string, _ *discordgo.ApplicationCommand, _ bot.Handler) {}

type Config struct {
	Enabled                 bool   `json:"enabled"`
	BotEnabled              bool   `json:"bot_enabled"`
	IntegrationsEnabled     bool   `json:"integrations_enabled"`
	AppID                   string `json:"app_id"`
	AppSecret               string `json:"app_secret"`
	LinkID                  string `json:"link_id"`
	Token                   string `json:"token"`
	GuildID                 string `json:"guild_id"`
	LogChannelID            string `json:"log_channel_id"`
	PublicLogChannelEnable  bool   `json:"public_log_channel_enable"`
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

const (
	OptUserIdentifier   = "user_identifier"
	OptServerIdentifier = "server_identifier"
	OptMessage          = "message"
	OptDuration         = "duration"
	OptASN              = "asn"
	OptIP               = "ip"
	OptMatchID          = "match_id"
	OptBanReason        = "ban_reason"
	OptUnbanReason      = "unban_reason"
	OptBan              = "ban"
	OptSteam            = "steam"
	OptNote             = "note"
	OptCIDR             = "cidr"
	OptPattern          = "pattern"
	OptIsRegex          = "is_regex"
)
