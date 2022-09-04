package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type botCmd string

const (
	cmdBan         botCmd = "ban"
	cmdFind        botCmd = "find"
	cmdMute        botCmd = "mute"
	cmdCheck       botCmd = "check"
	cmdCheckIp     botCmd = "checkip"
	cmdUnban       botCmd = "unban"
	cmdKick        botCmd = "kick"
	cmdPlayers     botCmd = "players"
	cmdPSay        botCmd = "psay"
	cmdCSay        botCmd = "csay"
	cmdSay         botCmd = "say"
	cmdServers     botCmd = "servers"
	cmdSetSteam    botCmd = "set_steam"
	cmdStats       botCmd = "stats"
	cmdStatsGlobal botCmd = "global"
	cmdStatsPlayer botCmd = "player"
	cmdStatsServer botCmd = "server"
	cmdHistory     botCmd = "history"
	cmdHistoryIP   botCmd = "ip"
	cmdHistoryChat botCmd = "chat"
	cmdFilter      botCmd = "filter"
	cmdLog         botCmd = "log"
)

//type subCommandKey string
//
//const (
//	CmdBan     = "ban"
//	CmdFilter  = "filter"
//	CmdHistory = "history"
//)

type optionKey string

const (
	OptUserIdentifier   = "user_identifier"
	OptServerIdentifier = "server_identifier"
	OptMessage          = "message"
	OptDuration         = "duration"
	OptASN              = "asn"
	OptIP               = "ip"
	OptMatchId          = "match_id"
	OptBanReason        = "ban_reason"
	OptUnbanReason      = "unban_reason"
	OptBan              = "ban"
	OptSteam            = "steam"
	OptNote             = "note"
	OptCIDR             = "cidr"
)

func (bot *discord) botRegisterSlashCommands() error {
	optUserID := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        OptUserIdentifier,
		Description: "SteamID in any format OR profile url",
		Required:    true,
	}
	optUserIDOptional := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        OptUserIdentifier,
		Description: "Optional SteamID in any format OR profile url to attach to a command",
		Required:    false,
	}
	optServerID := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        OptServerIdentifier,
		Description: "Short server name",
		Required:    true,
	}
	//optReason := &discordgo.ApplicationCommandOption{
	//	Type:        discordgo.ApplicationCommandOptionString,
	//	Name:        "reason",
	//	Description: "Reason for the ban (shown to users on kick)",
	//	Required:    true,
	//}
	optMessage := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        OptMessage,
		Description: "Message to send",
		Required:    true,
	}
	optDuration := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        OptDuration,
		Description: "Duration [s,m,h,d,w,M,y]N|0",
		Required:    true,
	}
	optAsn := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        OptASN,
		Description: "An Autonomous System (AS) is a group of one or more IP prefixes run by one or more network operators",
		Required:    true,
	}
	optIpAddr := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        OptIP,
		Description: "IP address to check",
		Required:    true,
	}
	optMatchId := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionInteger,
		Name:        OptMatchId,
		Description: "MatchID of any previously uploaded match",
		Required:    true,
	}
	var reasons []*discordgo.ApplicationCommandOptionChoice
	for _, r := range []model.Reason{model.External, model.Cheating, model.Racism, model.Harassment, model.Exploiting,
		model.WarningsExceeded, model.Spam, model.Language, model.Profile, model.ItemDescriptions, model.BotHost, model.Custom,
	} {
		reasons = append(reasons, &discordgo.ApplicationCommandOptionChoice{
			Name:  r.String(),
			Value: r,
		})
	}
	enabledDefault := true
	optBanReason := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionInteger,
		Name:        OptBanReason,
		Description: "Reason for the ban/mute",
		Required:    true,
		Choices:     reasons,
	}
	slashCommands := []*discordgo.ApplicationCommand{
		{
			Name:        string(cmdLog),
			Description: "Show a match log summary",
			Options: []*discordgo.ApplicationCommandOption{
				optMatchId,
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
				optBanReason,
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        OptNote,
					Description: "Mod only notes for the mute reason",
					Required:    true,
				},
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
			ApplicationID: config.Discord.AppID,
			Name:          string(cmdCheckIp),
			Description:   "Check if a ip is banned",
			Options: []*discordgo.ApplicationCommandOption{
				optIpAddr,
			},
		},
		{
			Name:        string(cmdKick),
			Description: "Kick a user from any server they are playing on",
			Options: []*discordgo.ApplicationCommandOption{
				optUserID,
				optBanReason,
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
					Name:        OptServerIdentifier,
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
			DefaultPermission: &enabledDefault,
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
			Name:          OptBan,
			Description:   "Manage steam, ip and ASN bans",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        OptSteam,
					Description: "BanSteam and kick a user from all servers",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						optUserID,
						optDuration,
						optBanReason,
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        OptNote,
							Description: "Mod only notes for the ban reason",
							Required:    true,
						},
					},
				},
				{
					Name:        "asn",
					Description: "BanSteam network(s) via their parent ASN (Autonomous System Number) from connecting to all servers",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						optAsn,
						optDuration,
						optBanReason,
						optUserIDOptional,
					},
				},
				{
					Name:        "ip",
					Description: "BanSteam and kick a network from connecting to all servers",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        OptCIDR,
							Description: "Network range to block eg: 12.34.56.78/32 (1 host) | 12.34.56.0/24 (256 hosts)",
							Required:    true,
						},
						optUserID,
						optDuration,
						optBanReason,
					},
				},
			},
		},
		{
			ApplicationID: config.Discord.AppID,
			Name:          "unban",
			Description:   "Manage steam, ip and ASN bans",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        OptSteam,
					Description: "Unban a previously banned player",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						optUserID,
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        OptUnbanReason,
							Description: "Reason for unbanning",
							Required:    true,
						},
					},
				}, // TODO ip
				{
					Name:        OptASN,
					Description: "Unban a previously banned ASN",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						optAsn,
					},
				},
			},
		},
		{
			ApplicationID:     config.Discord.AppID,
			Name:              string(cmdStats),
			Description:       "Query stats",
			DefaultPermission: &enabledDefault,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        string(cmdStatsPlayer),
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Description: "Get a players stats",
					Options: []*discordgo.ApplicationCommandOption{
						optUserID,
					},
				},
				{
					Name:        string(cmdStatsServer),
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Description: "Get a servers stats",
					Options: []*discordgo.ApplicationCommandOption{
						optServerID,
					},
				},
				{
					Name:        string(cmdStatsGlobal),
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Description: "Get a global stats",
					Options:     []*discordgo.ApplicationCommandOption{},
				},
			},
		},
		{
			ApplicationID: config.Discord.AppID,
			Name:          string(cmdFilter),
			Description:   "Manage and test global word filters",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
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
					Name:        "del",
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
					Name:        "check",
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
	var modPerms []*discordgo.ApplicationCommandPermissions
	for _, roleId := range config.Discord.ModRoleIDs {
		modPerms = append(modPerms, &discordgo.ApplicationCommandPermissions{
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
		command, errC := bot.session.ApplicationCommandCreate(config.Discord.AppID, config.Discord.GuildID, cmd)
		if errC != nil {
			return errors.Wrapf(errC, "Failed to register command: %s", cmd.Name)
		}
		if !*command.DefaultPermission && len(modPerms) > 0 {
			perms = append(perms, permissionRequest{
				ID:          command.ID,
				Permissions: modPerms,
			})
		}
	}

	return registerCommandPermissions(bot.ctx, perms)
}

type permissionRequest struct {
	ID          string                                     `json:"id"`
	Permissions []*discordgo.ApplicationCommandPermissions `json:"permissions"`
}

// registerCommandPermissions is used to additionally apply further restrictions to
// application commands that discordgo itself does not support yet.
// FIXME I suspect this is not required anymore
func registerCommandPermissions(ctx context.Context, perms []permissionRequest) error {
	httpClient := util.NewHTTPClient()
	body, errUnmarshal := json.Marshal(perms)
	if errUnmarshal != nil {
		return errors.Wrapf(errUnmarshal, "Failed to set command permissions")
	}
	permUrl := fmt.Sprintf("https://discord.com/api/v8/applications/%s/guilds/%s/commands/permissions", config.Discord.AppID, config.Discord.GuildID)
	reqCtx, cancelReq := context.WithTimeout(ctx, time.Second*10)
	defer cancelReq()
	req, errNewReq := http.NewRequestWithContext(reqCtx, "PUT", permUrl, bytes.NewReader(body))
	if errNewReq != nil {
		return errors.Wrapf(errNewReq, "Failed to create http request for discord permissions")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bot %s", config.Discord.Token))
	resp, errDo := httpClient.Do(req)
	if errDo != nil {
		return errors.Wrapf(errDo, "Failed to perform http request for discord permissions")
	}
	if resp.StatusCode != http.StatusOK {
		return errors.Wrapf(errDo, "Error response code trying to perform http request for discord permissions: %d", resp.StatusCode)
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
	Value   any
}

type botCommandHandler func(ctx context.Context, s *discordgo.Session,
	m *discordgo.InteractionCreate, r *botResponse) error

const (
	discordMaxMsgLen  = 2000
	discordMsgWrapper = "```"
)

// onInteractionCreate is called when a user initiates an application command. All commands are sent
// through this interface.
// https://discord.com/developers/docs/interactions/receiving-and-responding#receiving-an-interaction
func (bot *discord) onInteractionCreate(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	command := botCmd(interaction.ApplicationCommandData().Name)
	response := botResponse{MsgType: mtString}
	if handler, handlerFound := bot.commandHandlers[command]; handlerFound {
		// sendPreResponse should be called for any commands that call external services or otherwise
		// could not return a response instantly. discord will time out commands that don't respond within a
		// very short timeout windows, ~2-3 seconds.
		if errRespond := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Calculating numberwang...",
			},
		}); errRespond != nil {
			respErr(&response, fmt.Sprintf("Error: %session", errRespond.Error()))
			if errSendInteraction := bot.sendInteractionMessageEdit(session, interaction.Interaction, response); errSendInteraction != nil {
				log.Errorf("Failed sending error message for pre-interaction: %v", errSendInteraction)
			}
			return
		}
		commandCtx, cancelCommand := context.WithTimeout(bot.ctx, time.Second*30)
		defer cancelCommand()

		if errHandleCommand := handler(commandCtx, session, interaction, &response); errHandleCommand != nil {
			// TODO User facing errors only
			respErr(&response, errHandleCommand.Error())
			if errSendInteraction := bot.sendInteractionMessageEdit(session, interaction.Interaction, response); errSendInteraction != nil {
				log.Errorf("Failed sending error message for interaction: %v", errSendInteraction)
			}
			log.Errorf("User command error: %v", errHandleCommand)
			return
		}
		if sendSendResponse := bot.sendInteractionMessageEdit(session, interaction.Interaction, response); sendSendResponse != nil {
			log.Errorf("Failed sending success response for interaction: %v", sendSendResponse)
		} else {
			log.Tracef("Sent message embed")
		}
	}
}
