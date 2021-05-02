package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
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
		Description: "SteamID in any format OR profile permUrl",
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
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "server_identifier",
					Description: "Short server name or `*` for all",
					Required:    true,
				},
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
			Name:              string(cmdServers),
			Description:       "Show the high level status of all servers",
			DefaultPermission: true,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "full",
					Description: "Return the full status output including server versions and tags",
				},
			},
		},
	}
	modPerm := []*discordgo.ApplicationCommandPermission{
		{
			ID:         config.Discord.ModRoleID,
			Type:       1,
			Permission: true,
		},
	}
	// NOTE
	// We are manually calling the API to set permissions as this is not yet a feature for the discordgo library
	// This should be removed whenever support gets merged
	var perms []permissionRequest
	for _, cmd := range slashCommands {
		c, errC := dg.ApplicationCommandCreate(config.Discord.AppID, config.Discord.GuildID, cmd)
		if errC != nil {
			return errors.Wrapf(errC, "Failed to register command: %s", cmd.Name)
		}
		perms = append(perms, permissionRequest{
			ID:          c.ID,
			Permissions: modPerm,
		})
	}

	return registerCommandPermissions(perms)
}

type permissionRequest struct {
	ID          string                                    `json:"id"`
	Permissions []*discordgo.ApplicationCommandPermission `json:"permissions"`
}

func registerCommands(cmds []*discordgo.ApplicationCommand) ([]*discordgo.ApplicationCommand, error) {
	hc := util.NewHTTPClient()
	cmdUrl := fmt.Sprintf("https://discord.com/api/v8/applications/%s/guilds/%s/commands",
		config.Discord.AppID, config.Discord.GuildID)
	b, err := json.Marshal(cmds)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to marshal commands for request")
	}
	req, err := http.NewRequestWithContext(context.Background(), "PUT", cmdUrl, bytes.NewReader(b))
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create http request for discord commands")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bot %s", config.Discord.Token))
	resp, err := hc.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to perform http request for discord permissions")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Wrapf(err, "Failed to register slash commands, bad response: %d", resp.StatusCode)
	}
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to read body")
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Errorf("Failed to close resp body")
		}
	}()
	var r []*discordgo.ApplicationCommand
	if errDec := json.Unmarshal(b, &r); errDec != nil {
		return nil, errors.Wrapf(errDec, "Failed to decode json response")
	}
	return r, nil
}

func registerCommandPermissions(perms []permissionRequest) error {
	hc := util.NewHTTPClient()
	b, err := json.Marshal(perms)
	if err != nil {
		return errors.Wrapf(err, "Failed to set command permissions")
	}
	permUrl := fmt.Sprintf("https://discord.com/api/v8/applications/%s/guilds/%s/commands/permissions",
		config.Discord.AppID, config.Discord.GuildID)
	req, err := http.NewRequestWithContext(context.Background(), "PUT", permUrl, bytes.NewReader(b))
	if err != nil {
		return errors.Wrapf(err, "Failed to create http request for discord permissions")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bot %s", config.Discord.Token))
	resp, err := hc.Do(req)
	if err != nil {
		return errors.Wrapf(err, "Failed to perform http request for discord permissions")
	}
	if resp.StatusCode != http.StatusOK {
		return errors.Wrapf(err, "Error response code trying to perform http request for discord permissions: %d", resp.StatusCode)
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
	if h, ok := commandHandlers[botCmd(i.Data.Name)]; ok {
		if err := h(s, i); err != nil {
			// TODO User facing errors only
			_ = sendMsg(s, i.Interaction, "Error: %s", err.Error())
			log.Errorf("User command error: %v", err)
		}
	}
}
