package helper

import (
	"context"

	"github.com/bwmarrin/discordgo"
)

type SlashCommandHandler func(ctx context.Context, s *discordgo.Session, m *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error)

// type DiscordChannel int

// const (
// 	ChannelMod DiscordChannel = iota
// 	ChannelModLog
// 	ChannelPublicLog
// 	ChannelPublicMatchLog
// 	ChannelModAppealLog
// 	ChannelModVoteLog
// 	ChannelBanLog
// 	ChannelForumLog
// 	ChannelWordFilterLog
// 	ChannelKickLog
// 	ChannelPlayerqueue
// 	ChannelAC
// )

type Cmd string

const (
	CmdACPlayer    Cmd = "player"
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
