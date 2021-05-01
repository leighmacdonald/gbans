package service

import (
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	log "github.com/sirupsen/logrus"
)

type botCmd string

const (
	cmdBan     botCmd = "ban"
	cmdBanIP   botCmd = "banip"
	cmdFind    botCmd = "find"
	cmdMute    botCmd = "mute"
	cmdCheck   botCmd = "check"
	cmdUnban   botCmd = "unban"
	cmdKick    botCmd = "kick"
	cmdPlayers botCmd = "players"
	cmdPSay    botCmd = "psay"
	cmdCSay    botCmd = "csay"
	cmdSay     botCmd = "say"
	cmdServers botCmd = "servers"
)

func botRegisterSlashCommands(appID string) error {
	optUserID := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        "user_identifier",
		Description: "SteamID in any format OR profile url",
		Required:    true,
	}
	optServerID := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        "server_identifier",
		Description: "Short server name",
		Required:    true,
	}
	optReason := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        "reason",
		Description: "Reason for the action",
		Required:    false,
	}
	optMessage := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        "message",
		Description: "Message to send",
		Required:    true,
	}
	optDuration := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        "duration",
		Description: "Duration [s,m,h,w,M,y]N|0",
		Required:    true,
	}
	slashCommands := []*discordgo.ApplicationCommand{
		{
			Name:        string(cmdBan),
			Description: "Ban and kick a user from all servers",
			Options: []*discordgo.ApplicationCommandOption{
				optUserID,
				optDuration,
				optReason,
			},
		},
		{
			Name:        string(cmdBanIP),
			Description: "Ban and kick a network from connecting to all servers",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "cidr",
					Description: "Network range to block eg: 12.34.56.78/32 (1 host) | 12.34.56.0/24 (256 hosts)",
					Required:    true,
				},
				optDuration,
				optReason,
			},
		},
		{
			Name:        string(cmdFind),
			Description: "Find a user on any of the servers",
			Options: []*discordgo.ApplicationCommandOption{
				optUserID,
			},
		},
		{
			Name:        string(cmdMute),
			Description: "Mute a player",
			Options: []*discordgo.ApplicationCommandOption{
				optUserID,
				optDuration,
			},
		},
		{
			ApplicationID: config.Discord.AppID,
			Name:          string(cmdCheck),
			Description:   "Get ban status for a steam id",
			Options: []*discordgo.ApplicationCommandOption{
				optUserID,
			},
		},
		{
			Name:        string(cmdUnban),
			Description: "Unban a previously banned player",
			Options: []*discordgo.ApplicationCommandOption{
				optUserID,
			},
		},
		{
			Name:        string(cmdKick),
			Description: "Kick a user from any server they are playing on",
			Options: []*discordgo.ApplicationCommandOption{
				optUserID,
				optReason,
			},
		},
		{
			Name:        string(cmdPlayers),
			Description: "Show a table of the current players on the server",
			Options: []*discordgo.ApplicationCommandOption{
				optServerID,
			},
		},
		{
			Name:        string(cmdPSay),
			Description: "Privately message a player",
			Options: []*discordgo.ApplicationCommandOption{
				optUserID,
				optMessage,
			},
		},
		{
			Name:        string(cmdCSay),
			Description: "Send a centered message to the whole server",
			Options: []*discordgo.ApplicationCommandOption{
				optServerID,
				optMessage,
			},
		},
		{
			Name:        string(cmdSay),
			Description: "Send a console message to the whole server",
			Options: []*discordgo.ApplicationCommandOption{
				optServerID,
				optMessage,
			},
		},
		{
			Name:        string(cmdServers),
			Description: "Show the high level status of all servers",
		},
	}

	for _, cmd := range slashCommands {
		if _, err := dg.ApplicationCommandCreate(appID, config.Discord.GuildID, cmd); err != nil {
			log.Errorf("Failed to register command: %v", err)
		}
	}
	return nil
}

var commandHandlers = map[botCmd]func(s *discordgo.Session, m *discordgo.InteractionCreate) error{
	cmdBan:     onBan,
	cmdBanIP:   onBanIP,
	cmdCheck:   onCheck,
	cmdCSay:    onCSay,
	cmdFind:    onFind,
	cmdKick:    onKick,
	cmdMute:    onMute,
	cmdPlayers: onPlayers,
	cmdPSay:    onPSay,
	cmdSay:     onSay,
	cmdServers: onServers,
	cmdUnban:   onUnban,
}

func onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	found := false
	for _, roleID := range i.Member.Roles {
		if roleID == config.Discord.ModRoleID {
			found = true
			break
		}
	}
	if !found {
		_ = sendMsg(s, i.Interaction, "Permission denied")
		return
	}
	if h, ok := commandHandlers[botCmd(i.Data.Name)]; ok {
		if err := h(s, i); err != nil {
			// TODO User facing errors only
			_ = sendMsg(s, i.Interaction, "Error: %s", err.Error())
			log.Errorf("User command error: %v", err)
		}
	}
}
