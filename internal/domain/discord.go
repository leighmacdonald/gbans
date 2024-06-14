package domain

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type DiscordCredential struct {
	SteamID      steamid.SteamID `json:"steam_id"`
	DiscordID    string          `json:"discord_id"`
	AccessToken  string          `json:"access_token"`
	RefreshToken string          `json:"refresh_token"`
	ExpiresIn    int             `json:"expires_in"`
	Scope        string          `json:"scope"`
	TokenType    string          `json:"token_type"`
	TimeStamped
}

type DiscordUserDetail struct {
	SteamID          steamid.SteamID `json:"steam_id"`
	ID               string          `json:"id"`
	Username         string          `json:"username"`
	Avatar           string          `json:"avatar"`
	AvatarDecoration interface{}     `json:"avatar_decoration"`
	Discriminator    string          `json:"discriminator"`
	PublicFlags      int             `json:"public_flags"`
	Flags            int             `json:"flags"`
	Banner           interface{}     `json:"banner"`
	BannerColor      interface{}     `json:"banner_color"`
	AccentColor      interface{}     `json:"accent_color"`
	Locale           string          `json:"locale"`
	MfaEnabled       bool            `json:"mfa_enabled"`
	PremiumType      int             `json:"premium_type"`
	TimeStamped
}

type DiscordOAuthUsecase interface {
	CreateStatefulLoginURL(steamID steamid.SteamID) (string, error)
	HandleOAuthCode(ctx context.Context, code string, state string) error
	Logout(ctx context.Context, steamID steamid.SteamID) error
	Start(ctx context.Context)
	GetUserDetail(ctx context.Context, steamID steamid.SteamID) (DiscordUserDetail, error)
}

type DiscordOAuthRepository interface {
	SaveTokens(ctx context.Context, creds DiscordCredential) error
	GetTokens(ctx context.Context, steamID steamid.SteamID) (DiscordCredential, error)
	DeleteTokens(ctx context.Context, steamID steamid.SteamID) error
	OldAuths(ctx context.Context) ([]DiscordCredential, error)
	SaveUserDetail(ctx context.Context, detail DiscordUserDetail) error
	GetUserDetail(ctx context.Context, id steamid.SteamID) (DiscordUserDetail, error)
	DeleteUserDetail(ctx context.Context, steamID steamid.SteamID) error
}

type SlashCommandHandler func(ctx context.Context, s *discordgo.Session, m *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error)

type DiscordRepository interface {
	RegisterHandler(cmd Cmd, handler SlashCommandHandler) error
	Shutdown(guildID string)
	Start() error
	SendPayload(channelID string, payload *discordgo.MessageEmbed)
}

type DiscordChannel int

const (
	ChannelMod DiscordChannel = iota
	ChannelModLog
	ChannelPublicLog
	ChannelPublicMatchLog
	ChannelModAppealLog
	ChannelModVoteLog
	ChannelBanLog
	ChannelForumLog
	ChannelWordFilterLog
)

type DiscordUsecase interface {
	Start() error
	Shutdown(guildID string)
	SendPayload(channelID DiscordChannel, embed *discordgo.MessageEmbed)
	RegisterHandler(cmd Cmd, handler SlashCommandHandler) error
}

type FoundPlayer struct {
	Player PlayerServerInfo
	Server Server
}

type Cmd string

const (
	CmdBan         Cmd = "ban"
	CmdFind        Cmd = "find"
	CmdMute        Cmd = "mute"
	CmdCheck       Cmd = "check"
	CmdCheckIP     Cmd = "checkip"
	CmdUnban       Cmd = "unban"
	CmdKick        Cmd = "kick"
	CmdPlayers     Cmd = "players"
	CmdPSay        Cmd = "psay"
	CmdCSay        Cmd = "csay"
	CmdSay         Cmd = "say"
	CmdServers     Cmd = "servers"
	CmdSetSteam    Cmd = "set_steam"
	CmdStats       Cmd = "stats"
	CmdStatsGlobal Cmd = "global"
	CmdStatsPlayer Cmd = "player"
	CmdStatsServer Cmd = "server"
	CmdHistory     Cmd = "history"
	CmdHistoryIP   Cmd = "ip"
	CmdHistoryChat Cmd = "chat"
	CmdFilter      Cmd = "filter"
	CmdLog         Cmd = "log"
	CmdLogs        Cmd = "logs"
)

// type subCommandKey string
//
// const (
//	CmdBan     = "ban"
//	CmdFilter  = "filter"
//	CmdHistory = "history"
// )

type optionKey string

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

type CommandOptions map[optionKey]*discordgo.ApplicationCommandInteractionDataOption

// OptionMap will take the recursive discord slash commands and flatten them into a simple
// map.
func OptionMap(options []*discordgo.ApplicationCommandInteractionDataOption) CommandOptions {
	optionM := make(CommandOptions, len(options))
	for _, opt := range options {
		optionM[optionKey(opt.Name)] = opt
	}

	return optionM
}

func (opts CommandOptions) String(key optionKey) string {
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
