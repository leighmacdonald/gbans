package discord

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type nullDiscordRepository struct{}

func (bot *nullDiscordRepository) RegisterHandler(_ domain.Cmd, _ domain.SlashCommandHandler) error {
	return nil
}

func (bot *nullDiscordRepository) Shutdown(_ string) {
}

func (bot *nullDiscordRepository) Start() error {
	return nil
}

func (bot *nullDiscordRepository) SendPayload(_ string, _ *discordgo.MessageEmbed) {
}

func NewNullDiscordRepository() domain.DiscordRepository {
	return &nullDiscordRepository{}
}

type discordRepository struct {
	session           *discordgo.Session
	isReady           atomic.Bool
	commandHandlers   map[domain.Cmd]domain.SlashCommandHandler
	unregisterOnStart bool
	conf              domain.Config
}

func NewDiscordRepository(conf domain.Config) (domain.DiscordRepository, error) {
	if !conf.Discord.Enabled || !conf.Discord.BotEnabled {
		return &nullDiscordRepository{}, nil
	}

	// Immediately connects
	session, errNewSession := discordgo.New("Bot " + conf.Discord.Token)
	if errNewSession != nil {
		return nil, errors.Join(errNewSession, domain.ErrDiscordCreate)
	}

	session.UserAgent = "gbans (https://github.com/leighmacdonald/gbans)"
	session.Identify.Intents |= discordgo.IntentsGuildMessages
	session.Identify.Intents |= discordgo.IntentMessageContent
	session.Identify.Intents |= discordgo.IntentGuildMembers
	bot := &discordRepository{
		session:           session,
		conf:              conf,
		isReady:           atomic.Bool{},
		unregisterOnStart: conf.Discord.UnregisterOnStart,
		commandHandlers:   map[domain.Cmd]domain.SlashCommandHandler{},
	}

	bot.session.AddHandler(bot.onReady)
	bot.session.AddHandler(bot.onConnect)
	bot.session.AddHandler(bot.onDisconnect)
	bot.session.AddHandler(bot.onInteractionCreate)

	return bot, nil
}

func (bot *discordRepository) RegisterHandler(cmd domain.Cmd, handler domain.SlashCommandHandler) error {
	_, found := bot.commandHandlers[cmd]
	if found {
		return domain.ErrDuplicateCommand
	}

	bot.commandHandlers[cmd] = handler

	return nil
}

func (bot *discordRepository) Shutdown(guildID string) {
	if bot.session != nil {
		defer log.Closer(bot.session)
		bot.botUnregisterSlashCommands(guildID)
	}
}

func (bot *discordRepository) botUnregisterSlashCommands(guildID string) {
	registeredCommands, err := bot.session.ApplicationCommands(bot.session.State.User.ID, guildID)
	if err != nil {
		slog.Error("Could not fetch registered commands", log.ErrAttr(err))

		return
	}

	for _, v := range registeredCommands {
		if errDel := bot.session.ApplicationCommandDelete(bot.session.State.User.ID, guildID, v.ID); errDel != nil {
			slog.Error("Cannot delete command", slog.String("name", v.Name), log.ErrAttr(err))

			return
		}
	}

	slog.Info("Unregistered discord commands", slog.Int("count", len(registeredCommands)))
}

func (bot *discordRepository) Start() error {
	// Open a websocket connection to discord and begin listening.
	if errSessionOpen := bot.session.Open(); errSessionOpen != nil {
		return errors.Join(errSessionOpen, domain.ErrDiscordOpen)
	}

	if bot.unregisterOnStart {
		bot.botUnregisterSlashCommands("")
	}

	return nil
}

func (bot *discordRepository) onReady(session *discordgo.Session, _ *discordgo.Ready) {
	slog.Info("Discord state changed", slog.String("state", "ready"), slog.String("username",
		fmt.Sprintf("%v#%v", session.State.User.Username, session.State.User.Discriminator)))
}

func (bot *discordRepository) onConnect(_ *discordgo.Session, _ *discordgo.Connect) {
	if errRegister := bot.botRegisterSlashCommands(bot.conf.Discord.AppID); errRegister != nil {
		slog.Error("Failed to register discord slash commands", log.ErrAttr(errRegister))
	}

	status := discordgo.UpdateStatusData{
		IdleSince: nil,
		Activities: []*discordgo.Activity{
			{
				Name:     "Cheeseburgers",
				Type:     discordgo.ActivityTypeListening,
				URL:      bot.conf.ExternalURL,
				State:    "state field",
				Details:  "Blah",
				Instance: true,
				Flags:    1 << 0,
			},
		},
		AFK:    false,
		Status: "https://github.com/leighmacdonald/gbans",
	}
	if errUpdateStatus := bot.session.UpdateStatusComplex(status); errUpdateStatus != nil {
		slog.Error("Failed to update status complex", log.ErrAttr(errUpdateStatus))
	}

	slog.Info("Discord state changed", slog.String("state", "connected"))

	bot.isReady.Store(true)
}

func (bot *discordRepository) onDisconnect(_ *discordgo.Session, _ *discordgo.Disconnect) {
	bot.isReady.Store(false)

	slog.Info("Discord state changed", slog.String("state", "disconnected"))
}

// onInteractionCreate is called when a user initiates an application command. All commands are sent
// through this interface.
// https://discord.com/developers/docs/interactions/receiving-and-responding#receiving-an-interaction
func (bot *discordRepository) onInteractionCreate(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	var (
		data    = interaction.ApplicationCommandData()
		command = domain.Cmd(data.Name)
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
				slog.Error("Failed sending error response for interaction", log.ErrAttr(errFollow))
			}

			return
		}

		commandCtx, cancelCommand := context.WithTimeout(context.TODO(), time.Second*30)
		defer cancelCommand()

		response, errHandleCommand := handler(commandCtx, session, interaction)
		if errHandleCommand != nil || response == nil {
			if _, errFollow := session.FollowupMessageCreate(interaction.Interaction, true, &discordgo.WebhookParams{
				Embeds: []*discordgo.MessageEmbed{ErrorMessage(string(command), errHandleCommand)},
			}); errFollow != nil {
				slog.Error("Failed sending error response for interaction", log.ErrAttr(errFollow))
			}

			return
		}

		if sendSendResponse := bot.sendInteractionResponse(session, interaction.Interaction, response); sendSendResponse != nil {
			slog.Error("Failed sending success response for interaction", log.ErrAttr(sendSendResponse))
		}
	}
}

func (bot *discordRepository) sendInteractionResponse(session *discordgo.Session, interaction *discordgo.Interaction, response *discordgo.MessageEmbed) error {
	resp := &discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{response},
	}

	_, errResponseErr := session.InteractionResponseEdit(interaction, &discordgo.WebhookEdit{
		Embeds: &resp.Embeds,
	})

	if errResponseErr != nil {
		if _, errResp := session.FollowupMessageCreate(interaction, true, &discordgo.WebhookParams{
			Content: "Something went wrong",
		}); errResp != nil {
			return errors.Join(errResp, domain.ErrDiscordMessageSen)
		}

		return nil
	}

	return nil
}

func (bot *discordRepository) SendPayload(channelID string, payload *discordgo.MessageEmbed) {
	if !bot.isReady.Load() {
		return
	}

	if _, errSend := bot.session.ChannelMessageSendEmbed(channelID, payload); errSend != nil {
		slog.Error("Failed to send discord payload", log.ErrAttr(errSend))
	}
}

//nolint:funlen,maintidx
func (bot *discordRepository) botRegisterSlashCommands(appID string) error {
	dmPerms := false
	modPerms := int64(discordgo.PermissionBanMembers)
	userPerms := int64(discordgo.PermissionViewChannel)
	optUserID := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        domain.OptUserIdentifier,
		Description: "SteamID in any format OR profile url",
		Required:    true,
	}
	optUserIDOptional := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        domain.OptUserIdentifier,
		Description: "Optional SteamID in any format OR profile url to attach to a command",
		Required:    true,
	}
	optServerID := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        domain.OptServerIdentifier,
		Description: "Short server name",
		Required:    true,
	}
	// optReason := &discordgo.ApplicationCommandOption{
	//	Type:        discordgo.ApplicationCommandOptionString,
	//	Name:        "reason",
	//	Description: "Reason for the ban (shown to users on kick)",
	//	Required:    true,
	// }
	optMessage := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        domain.OptMessage,
		Description: "Message to send",
		Required:    true,
	}
	optDuration := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        domain.OptDuration,
		Description: "Duration [s,m,h,d,w,M,y]N|0",
		Required:    true,
	}
	optAsn := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        domain.OptASN,
		Description: "An Autonomous System (AS) is a group of one or more IP prefixes run by one or more network operators",
		Required:    true,
	}
	optIPAddr := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        domain.OptIP,
		Description: "IP address to check",
		Required:    true,
	}
	optMatchID := &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        domain.OptMatchID,
		Description: "MatchID of any previously uploaded match",
		Required:    true,
	}

	reasonCollection := []domain.Reason{
		domain.External, domain.Cheating, domain.Racism, domain.Harassment, domain.Exploiting,
		domain.WarningsExceeded, domain.Spam, domain.Language, domain.Profile, domain.ItemDescriptions,
		domain.BotHost, domain.Evading, domain.Username, domain.Custom,
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
		Name:        domain.OptBanReason,
		Description: "Reason for the ban/mute",
		Required:    true,
		Choices:     reasons,
	}

	slashCommands := []*discordgo.ApplicationCommand{
		{
			Name:        string(domain.CmdLog),
			Description: "Show a match log summary",
			Options: []*discordgo.ApplicationCommandOption{
				optMatchID,
			},
		},
		{
			Name:        string(domain.CmdLogs),
			Description: "Show a list of your recent logs",
			Options:     []*discordgo.ApplicationCommandOption{},
		},
		{
			Name:                     string(domain.CmdFind),
			DMPermission:             &dmPerms,
			DefaultMemberPermissions: &modPerms,
			Description:              "Find a user on any of the servers",
			Options: []*discordgo.ApplicationCommandOption{
				optUserID,
			},
		},
		{
			Name:                     string(domain.CmdMute),
			Description:              "Mute a player",
			DMPermission:             &dmPerms,
			DefaultMemberPermissions: &modPerms,
			Options: []*discordgo.ApplicationCommandOption{
				optUserID,
				optDuration,
				optBanReason,
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        domain.OptNote,
					Description: "Mod only notes for the mute reason",
					Required:    true,
				},
			},
		},
		{
			ApplicationID:            appID,
			Name:                     string(domain.CmdCheck),
			DMPermission:             &dmPerms,
			DefaultMemberPermissions: &modPerms,
			Description:              "Get ban status for a steam id",
			Options: []*discordgo.ApplicationCommandOption{
				optUserID,
			},
		},
		{
			ApplicationID:            appID,
			Name:                     string(domain.CmdCheckIP),
			DMPermission:             &dmPerms,
			DefaultMemberPermissions: &modPerms,
			Description:              "Check if a ip is banned",
			Options: []*discordgo.ApplicationCommandOption{
				optIPAddr,
			},
		},
		{
			Name:                     string(domain.CmdKick),
			DMPermission:             &dmPerms,
			DefaultMemberPermissions: &modPerms,
			Description:              "Kick a user from any server they are playing on",
			Options: []*discordgo.ApplicationCommandOption{
				optUserID,
				optBanReason,
			},
		},
		{
			Name:                     string(domain.CmdPlayers),
			DMPermission:             &dmPerms,
			DefaultMemberPermissions: &modPerms,
			Description:              "Show a table of the current players on the server",
			Options: []*discordgo.ApplicationCommandOption{
				optServerID,
			},
		},
		{
			Name:                     string(domain.CmdPSay),
			Description:              "Privately message a player",
			DMPermission:             &dmPerms,
			DefaultMemberPermissions: &modPerms,
			Options: []*discordgo.ApplicationCommandOption{
				optUserID,
				optMessage,
			},
		},
		{
			Name:                     string(domain.CmdCSay),
			Description:              "Send a centered message to the whole server",
			DMPermission:             &dmPerms,
			DefaultMemberPermissions: &modPerms,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        domain.OptServerIdentifier,
					Description: "Short server name or `*` for all",
					Required:    true,
				},
				optMessage,
			},
		},
		{
			Name:                     string(domain.CmdSay),
			Description:              "Send a console message to the whole server",
			DMPermission:             &dmPerms,
			DefaultMemberPermissions: &modPerms,
			Options: []*discordgo.ApplicationCommandOption{
				optServerID,
				optMessage,
			},
		},
		{
			Name:                     string(domain.CmdServers),
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
			Name:                     string(domain.CmdHistory),
			DMPermission:             &dmPerms,
			DefaultMemberPermissions: &modPerms,
			Description:              "Query user history",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        string(domain.CmdHistoryIP),
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Description: "Get the ip history",
					Options: []*discordgo.ApplicationCommandOption{
						optUserID,
					},
				},
				{
					Name:        string(domain.CmdHistoryChat),
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
			Name:                     domain.OptBan,
			Description:              "Manage steam, ip and ASN bans",
			DMPermission:             &dmPerms,
			DefaultMemberPermissions: &modPerms,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        domain.OptSteam,
					Description: "Ban and kick a user from all servers",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						optUserID,
						optDuration,
						optBanReason,
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        domain.OptNote,
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
							Name:        domain.OptNote,
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
							Name:        domain.OptCIDR,
							Description: "Network range to block eg: 12.34.56.78/32 (1 host) | 12.34.56.0/24 (256 hosts)",
							Required:    true,
						},
						optUserID,
						optDuration,
						optBanReason,
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        domain.OptNote,
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
					Name:        domain.OptSteam,
					Description: "Unban a previously banned player",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						optUserID,
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        domain.OptUnbanReason,
							Description: "Reason for unbanning",
							Required:    true,
						},
					},
				}, // TODO ip
				{
					Name:        domain.OptASN,
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
			Name:                     string(domain.CmdStats),
			Description:              "Query stats",
			DefaultMemberPermissions: &userPerms,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        string(domain.CmdStatsPlayer),
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
			Name:                     string(domain.CmdAC),
			Description:              "Query Anticheat Logs",
			DefaultMemberPermissions: &modPerms,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        string(domain.CmdACPlayer),
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Description: "Query a players anticheat logs by steam id",
					Options: []*discordgo.ApplicationCommandOption{
						optUserID,
					},
				},
			},
		},

		{
			ApplicationID:            appID,
			Name:                     string(domain.CmdFilter),
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
							Name:        domain.OptIsRegex,
							Description: "Is the pattern a regular expression?",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        domain.OptPattern,
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
							Name:        domain.OptMessage,
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
		return errors.Join(errBulk, domain.ErrDiscordOverwriteCommands)
	}

	slog.Debug("Registered discord commands", slog.Int("count", len(slashCommands)))

	return nil
}
