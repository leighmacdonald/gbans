package discord

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
	"net/http"
	"time"
)

type botCmd string

const (
	cmdBan         botCmd = "ban"
	cmdBanIP       botCmd = "banip"
	cmdFind        botCmd = "find"
	cmdMute        botCmd = "mute"
	cmdCheck       botCmd = "check"
	cmdUnban       botCmd = "unban"
	cmdKick        botCmd = "kick"
	cmdPlayers     botCmd = "players"
	cmdPSay        botCmd = "psay"
	cmdCSay        botCmd = "csay"
	cmdSay         botCmd = "say"
	cmdServers     botCmd = "servers"
	cmdSetSteam    botCmd = "set_steam"
	cmdHistory     botCmd = "history"
	cmdHistoryIP   botCmd = "ip"
	cmdHistoryChat botCmd = "chat"
	cmdFilter      botCmd = "filter"
	cmdFilterAdd   botCmd = "add"
	cmdFilterDel   botCmd = "del"
	cmdFilterCheck botCmd = "check"
)

func (b *DiscordClient) botRegisterSlashCommands() error {
	// TODO register the commands again upon adding new servers to update autocomplete opts
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
		Description: "Reason for the ban (shown to users on kick)",
		Required:    true,
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
		Description: "Duration [s,m,h,d,w,M,y]N|0",
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
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "note",
					Description: "Mod only notes for the ban reason",
					Required:    false,
				},
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
				optUserID,
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
				optReason,
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
		{
			ApplicationID: config.Discord.AppID,
			Name:          string(cmdSetSteam),
			Description:   "Set your steam ID so gbans can link your account to discord",
			Options: []*discordgo.ApplicationCommandOption{
				optUserID,
			},
		},
		{
			ApplicationID: config.Discord.AppID,
			Name:          string(cmdHistory),
			Description:   "Query user history",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        string(cmdHistoryIP),
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Description: "Get the ip history",
					Options: []*discordgo.ApplicationCommandOption{
						optUserID,
					},
				},
				{
					Name:        string(cmdHistoryChat),
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Description: "Get the chat history of the user",
					Options: []*discordgo.ApplicationCommandOption{
						optUserID,
					},
				},
			},
		},
		{
			ApplicationID: config.Discord.AppID,
			Name:          string(cmdFilter),
			Description:   "Manage and test global word filters",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        string(cmdFilterAdd),
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Description: "Add a new filtered word",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "filter",
							Description: "Regular expression for matching word(s)",
							Required:    true,
						},
					},
				},
				{
					Name:        string(cmdFilterDel),
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Description: "Remove a filtered word",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "filter",
							Description: "Filter ID",
							Required:    true,
						},
					},
				},
				{
					Name:        string(cmdFilterCheck),
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Description: "Check if a string has a matching filter",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "string",
							Description: "String to check filters against",
							Required:    true,
						},
					},
				},
			},
		},
	}
	var modPerms []*discordgo.ApplicationCommandPermission
	for _, roleId := range config.Discord.ModRoleIDs {
		modPerms = append(modPerms, &discordgo.ApplicationCommandPermission{
			ID:         roleId,
			Type:       1,
			Permission: true,
		})
	}
	// NOTE
	// We are manually calling the API to set permissions as this is not yet a feature for the discordgo library
	// This should be removed whenever support gets merged
	var perms []permissionRequest
	for _, cmd := range slashCommands {
		command, errC := b.dg.ApplicationCommandCreate(config.Discord.AppID, config.Discord.GuildID, cmd)
		if errC != nil {
			return errors.Wrapf(errC, "Failed to register command: %s", cmd.Name)
		}
		if !command.DefaultPermission && len(modPerms) > 0 {
			perms = append(perms, permissionRequest{
				ID:          command.ID,
				Permissions: modPerms,
			})
		}
	}

	return registerCommandPermissions(perms)
}

type permissionRequest struct {
	ID          string                                    `json:"id"`
	Permissions []*discordgo.ApplicationCommandPermission `json:"permissions"`
}

// registerCommandPermissions is used to additionally apply further restrictions to
// application commands that discordgo itself does not support yet.
func registerCommandPermissions(perms []permissionRequest) error {
	httpClient := util.NewHTTPClient()
	body, err := json.Marshal(perms)
	if err != nil {
		return errors.Wrapf(err, "Failed to set command permissions")
	}
	permUrl := fmt.Sprintf("https://discord.com/api/v8/applications/%s/guilds/%s/commands/permissions",
		config.Discord.AppID, config.Discord.GuildID)
	req, err2 := http.NewRequestWithContext(context.Background(), "PUT", permUrl, bytes.NewReader(body))
	if err2 != nil {
		return errors.Wrapf(err2, "Failed to create http request for discord permissions")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bot %s", config.Discord.Token))
	resp, err3 := httpClient.Do(req)
	if err3 != nil {
		return errors.Wrapf(err3, "Failed to perform http request for discord permissions")
	}
	if resp.StatusCode != http.StatusOK {
		return errors.Wrapf(err3, "Error response code trying to perform http request for discord permissions: %d", resp.StatusCode)
	}
	return nil
}

type responseMsgType int

const (
	mtString responseMsgType = iota
	mtEmbed
	//mtImage
)

type botResponse struct {
	MsgType responseMsgType
	Value   interface{}
}

type botCommandHandler func(ctx context.Context, s *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error

const (
	discordMaxMsgLen  = 2000
	discordMsgWrapper = "```"
	// Accounts for the char lens: ``````
	discordWrapperTotalLen = discordMaxMsgLen - (len(discordMsgWrapper) * 2)
)

// onInteractionCreate is called when a user initiates an application command. All commands are sent
// through this interface.
// https://discord.com/developers/docs/interactions/receiving-and-responding#receiving-an-interaction
func (b *DiscordClient) onInteractionCreate(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	cmd := botCmd(interaction.Data.Name)
	response := botResponse{MsgType: mtString}
	if handler, ok := b.commandHandlers[cmd]; ok {
		// sendPreResponse should be called for any commands that call external services or otherwise
		// could not return a response instantly. Discord will time out commands that don't respond within a
		// very short timeout windows, ~2-3 seconds.
		if err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionApplicationCommandResponseData{
				Content: "Calculating numberwang...",
			},
		}); err != nil {
			RespErr(&response, fmt.Sprintf("Error: %session", err.Error()))
			if sendE := b.sendInteractionMessageEdit(session, interaction.Interaction, response); sendE != nil {
				log.Errorf("Failed sending error message for pre-interaction: %v", sendE)
			}
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		if err := handler(ctx, session, interaction, &response); err != nil {
			// TODO User facing errors only
			RespErr(&response, fmt.Sprintf("Error: %session", err.Error()))
			if sendE := b.sendInteractionMessageEdit(session, interaction.Interaction, response); sendE != nil {
				log.Errorf("Failed sending error message for interaction: %v", sendE)
			}
			log.Errorf("User command error: %v", err)
			return
		}
		if sendE := b.sendInteractionMessageEdit(session, interaction.Interaction, response); sendE != nil {
			log.Errorf("Failed sending success response for interaction: %v", sendE)
		} else {
			log.Debugf("Sent message embed")
		}
	}
}
