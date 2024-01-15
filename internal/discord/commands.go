package discord

import (
	"context"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

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

//nolint:funlen,maintidx
func (bot *Bot) botRegisterSlashCommands(appID string) error {
	dmPerms := false
	modPerms := int64(discordgo.PermissionBanMembers)
	userPerms := int64(discordgo.PermissionViewChannel)
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
		Required:    true,
	}
	optServerID := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        OptServerIdentifier,
		Description: "Short server name",
		Required:    true,
	}
	// optReason := &discordgo.ApplicationCommandOption{
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
	optIPAddr := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        OptIP,
		Description: "IP address to check",
		Required:    true,
	}
	optMatchID := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        OptMatchID,
		Description: "MatchID of any previously uploaded match",
		Required:    true,
	}

	reasonCollection := []store.Reason{
		store.External, store.Cheating, store.Racism, store.Harassment, store.Exploiting,
		store.WarningsExceeded, store.Spam, store.Language, store.Profile, store.ItemDescriptions, store.BotHost, store.Custom,
	}

	reasons := make([]*discordgo.ApplicationCommandOptionChoice, len(reasonCollection))

	for index, r := range reasonCollection {
		reasons[index] = &discordgo.ApplicationCommandOptionChoice{
			Name:  r.String(),
			Value: r,
		}
	}

	optBanReason := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionInteger,
		Name:        OptBanReason,
		Description: "Reason for the ban/mute",
		Required:    true,
		Choices:     reasons,
	}

	slashCommands := []*discordgo.ApplicationCommand{
		{
			Name:        string(CmdLog),
			Description: "Show a match log summary",
			Options: []*discordgo.ApplicationCommandOption{
				optMatchID,
			},
		},
		{
			Name:        string(CmdLogs),
			Description: "Show a list of your recent logs",
			Options:     []*discordgo.ApplicationCommandOption{},
		},
		{
			Name:                     string(CmdFind),
			DMPermission:             &dmPerms,
			DefaultMemberPermissions: &modPerms,
			Description:              "Find a user on any of the servers",
			Options: []*discordgo.ApplicationCommandOption{
				optUserID,
			},
		},
		{
			Name:                     string(CmdMute),
			Description:              "Mute a player",
			DMPermission:             &dmPerms,
			DefaultMemberPermissions: &modPerms,
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
			ApplicationID:            appID,
			Name:                     string(CmdCheck),
			DMPermission:             &dmPerms,
			DefaultMemberPermissions: &modPerms,
			Description:              "Get ban status for a steam id",
			Options: []*discordgo.ApplicationCommandOption{
				optUserID,
			},
		},
		{
			ApplicationID:            appID,
			Name:                     string(CmdCheckIP),
			DMPermission:             &dmPerms,
			DefaultMemberPermissions: &modPerms,
			Description:              "Check if a ip is banned",
			Options: []*discordgo.ApplicationCommandOption{
				optIPAddr,
			},
		},
		{
			Name:                     string(CmdKick),
			DMPermission:             &dmPerms,
			DefaultMemberPermissions: &modPerms,
			Description:              "Kick a user from any server they are playing on",
			Options: []*discordgo.ApplicationCommandOption{
				optUserID,
				optBanReason,
			},
		},
		{
			Name:                     string(CmdPlayers),
			DMPermission:             &dmPerms,
			DefaultMemberPermissions: &modPerms,
			Description:              "Show a table of the current players on the server",
			Options: []*discordgo.ApplicationCommandOption{
				optServerID,
			},
		},
		{
			Name:                     string(CmdPSay),
			Description:              "Privately message a player",
			DMPermission:             &dmPerms,
			DefaultMemberPermissions: &modPerms,
			Options: []*discordgo.ApplicationCommandOption{
				optUserID,
				optMessage,
			},
		},
		{
			Name:                     string(CmdCSay),
			Description:              "Send a centered message to the whole server",
			DMPermission:             &dmPerms,
			DefaultMemberPermissions: &modPerms,
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
			Name:                     string(CmdSay),
			Description:              "Send a console message to the whole server",
			DMPermission:             &dmPerms,
			DefaultMemberPermissions: &modPerms,
			Options: []*discordgo.ApplicationCommandOption{
				optServerID,
				optMessage,
			},
		},
		{
			Name:                     string(CmdServers),
			Description:              "Show the high level status of all servers",
			DefaultMemberPermissions: &userPerms,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "full",
					Description: "Return the full status output including server versions and tags",
				},
			},
		},
		{
			ApplicationID:            appID,
			Name:                     string(CmdSetSteam),
			DMPermission:             &dmPerms,
			DefaultMemberPermissions: &modPerms,
			Description:              "Set your steam ID so gbans can link your account to discord",
			Options: []*discordgo.ApplicationCommandOption{
				optUserID,
			},
		},
		{
			ApplicationID:            appID,
			Name:                     string(CmdHistory),
			DMPermission:             &dmPerms,
			DefaultMemberPermissions: &modPerms,
			Description:              "Query user history",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        string(CmdHistoryIP),
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Description: "Get the ip history",
					Options: []*discordgo.ApplicationCommandOption{
						optUserID,
					},
				},
				{
					Name:        string(CmdHistoryChat),
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Description: "Get the chat history of the user",
					Options: []*discordgo.ApplicationCommandOption{
						optUserID,
					},
				},
			},
		},
		{
			ApplicationID:            appID,
			Name:                     OptBan,
			Description:              "Manage steam, ip and ASN bans",
			DMPermission:             &dmPerms,
			DefaultMemberPermissions: &modPerms,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        OptSteam,
					Description: "Ban and kick a user from all servers",
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
					Description: "Ban network(s) via their parent ASN (Autonomous System Number) from connecting to all servers",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						optAsn,
						optDuration,
						optBanReason,
						optUserIDOptional,
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        OptNote,
							Description: "Mod only notes for the mute reason",
							Required:    true,
						},
					},
				},
				{
					Name:        "ip",
					Description: "Ban and kick a network from connecting to all servers",
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
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        OptNote,
							Description: "Mod only notes for the mute reason",
							Required:    true,
						},
					},
				},
			},
		},
		{
			ApplicationID:            appID,
			Name:                     "unban",
			Description:              "Manage steam, ip and ASN bans",
			DMPermission:             &dmPerms,
			DefaultMemberPermissions: &modPerms,
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
			ApplicationID:            appID,
			Name:                     string(CmdStats),
			Description:              "Query stats",
			DefaultMemberPermissions: &userPerms,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        string(CmdStatsPlayer),
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Description: "Get a players stats",
					Options: []*discordgo.ApplicationCommandOption{
						optUserID,
					},
				},
				// {
				//	Name:        string(CmdStatsServer),
				//	Type:        discordgo.ApplicationCommandOptionSubCommand,
				//	Description: "Get a servers stats",
				//	Options: []*discordgo.ApplicationCommandOption{
				//		optServerID,
				//	},
				// },
				// {
				//	Name:        string(CmdStatsGlobal),
				//	Type:        discordgo.ApplicationCommandOptionSubCommand,
				//	Description: "Get a global stats",
				//	Options:     []*discordgo.ApplicationCommandOption{},
				// },
			},
		},
		{
			ApplicationID:            appID,
			Name:                     string(CmdFilter),
			Description:              "Manage and test global word filters",
			DMPermission:             &dmPerms,
			DefaultMemberPermissions: &modPerms,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Description: "Add a new filtered word",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionBoolean,
							Name:        OptIsRegex,
							Description: "Is the pattern a regular expression?",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        OptPattern,
							Description: "Regular expression or word for matching",
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
							Name:        OptMessage,
							Description: "String to check filters against",
							Required:    true,
						},
					},
				},
			},
		},
	}

	_, errBulk := bot.session.ApplicationCommandBulkOverwrite(appID, "", slashCommands)
	if errBulk != nil {
		return errors.Wrap(errBulk, "Failed to bulk overwrite application commands")
	}

	bot.log.Info("Registered discord commands", zap.Int("count", len(slashCommands)))

	return nil
}

type CommandOptions map[optionKey]*discordgo.ApplicationCommandInteractionDataOption

type CommandHandler func(ctx context.Context, s *discordgo.Session, m *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error)

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

// onInteractionCreate is called when a user initiates an application command. All commands are sent
// through this interface.
// https://discord.com/developers/docs/interactions/receiving-and-responding#receiving-an-interaction

func (bot *Bot) onInteractionCreate(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	var (
		data    = interaction.ApplicationCommandData()
		command = Cmd(data.Name)
	)

	if handler, handlerFound := bot.commandHandlers[command]; handlerFound {
		// sendPreResponse should be called for any commands that call external services or otherwise
		// could not return a response instantly. discord will time out commands that don't respond within a
		// very short timeout windows, ~2-3 seconds.
		initialResponse := &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Calculating numberwang...",
			},
		}

		if errRespond := session.InteractionRespond(interaction.Interaction, initialResponse); errRespond != nil {
			if _, errFollow := session.FollowupMessageCreate(interaction.Interaction, true, &discordgo.WebhookParams{
				Content: errRespond.Error(),
			}); errFollow != nil {
				bot.log.Error("Failed sending error response for interaction", zap.Error(errFollow))
			}

			return
		}

		commandCtx, cancelCommand := context.WithTimeout(context.TODO(), time.Second*30)
		defer cancelCommand()

		response, errHandleCommand := handler(commandCtx, session, interaction)
		if errHandleCommand != nil || response == nil {
			errEmbed := NewEmbed(config.Config{}, "Error Returned").Embed().
				SetColor(bot.Colour.Error).
				AddField("command", string(command)).
				SetDescription(errHandleCommand.Error())
			if _, errFollow := session.FollowupMessageCreate(interaction.Interaction, true, &discordgo.WebhookParams{
				Embeds: []*discordgo.MessageEmbed{errEmbed.MessageEmbed},
			}); errFollow != nil {
				bot.log.Error("Failed sending error response for interaction", zap.Error(errFollow))
			}

			return
		}

		if sendSendResponse := bot.sendInteractionResponse(session, interaction.Interaction, response); sendSendResponse != nil {
			bot.log.Error("Failed sending success response for interaction", zap.Error(sendSendResponse))
		}
	}
}
