package domain

import (
	"context"

	"github.com/bwmarrin/discordgo"
)

type SlashCommandHandler func(ctx context.Context, s *discordgo.Session, m *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error)

type DiscordRepository interface {
	RegisterHandler(cmd Cmd, handler SlashCommandHandler) error
	Shutdown(guildID string)
	Start() error
	SendPayload(channelID DiscordChannel, payload *discordgo.MessageEmbed)
}

type DiscordChannel int

const (
	ChannelMod DiscordChannel = iota
	ChannelModLog
	ChannelPublicLog
	ChannelPublicMatchLog
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
