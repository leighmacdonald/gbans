package discord

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/discord/message"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type discordService struct {
	session         *discordgo.Session
	isReady         atomic.Bool
	commandHandlers map[Cmd]SlashCommandHandler
	commands        []*discordgo.ApplicationCommand
	token           string
	appID           string
	guildID         string
	externalURL     string
	config          *config.ConfigUsecase
	tfAPI           *thirdparty.TFAPI
}

func NewDiscordHandler(appID string, guildID string, token string, externalURL string, tfAPI *thirdparty.TFAPI,
) (*discordService, error) {
	if appID == "" || guildID == "" || token == "" || tfAPI == nil {
		return nil, ErrDiscordConfig
	}

	bot := &discordService{
		tfAPI:           tfAPI,
		isReady:         atomic.Bool{},
		commandHandlers: map[Cmd]SlashCommandHandler{},
		appID:           appID,
		guildID:         guildID,
		token:           token,
		externalURL:     externalURL,
	}

	return bot, nil
}

func (h *discordService) Start(_ context.Context) error {
	session, errNewSession := discordgo.New("Bot " + h.token)
	if errNewSession != nil {
		return errors.Join(errNewSession, ErrDiscordCreate)
	}
	session.UserAgent = "gbans (https://github.com/leighmacdonald/gbans)"
	session.Identify.Intents |= discordgo.IntentsGuildMessages
	session.Identify.Intents |= discordgo.IntentMessageContent
	session.Identify.Intents |= discordgo.IntentGuildMembers
	session.AddHandler(h.onReady)
	session.AddHandler(h.onConnect)
	session.AddHandler(h.onDisconnect)
	session.AddHandler(h.onInteractionCreate)

	h.session = session

	// cmdMap := map[Cmd]func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error){
	// 	CmdBan:     h.makeOnBan(),
	// 	CmdCheck:   h.makeOnCheck(),
	// 	CmdCSay:    h.makeOnCSay(),
	// 	CmdFilter:  h.makeOnFilter(),
	// 	CmdFind:    h.makeOnFind(),
	// 	CmdHistory: h.makeOnHistory(),
	// 	CmdKick:    h.makeOnKick(),
	// 	CmdLog:     h.makeOnLog(),
	// 	CmdLogs:    h.makeOnLogs(),
	// 	CmdMute:    h.makeOnMute(),
	// 	// domain.CmdCheckIP:  h.onCheckIp,
	// 	CmdPlayers: h.makeOnPlayers(),
	// 	CmdPSay:    h.makeOnPSay(),
	// 	CmdSay:     h.makeOnSay(),
	// 	CmdServers: h.makeOnServers(),
	// 	CmdUnban:   h.makeOnUnban(),
	// 	CmdStats:   h.makeOnStats(),
	// 	CmdAC:      h.makeOnAC(),
	// }

	// for k, v := range cmdMap {
	// 	if errRegister := h.discord.RegisterHandler(k, v); errRegister != nil {
	// 		slog.Error("Failed to register handler", log.ErrAttr(errRegister))
	// 	}
	// }
	//

	// Open a websocket connection to discord and begin listening.
	if errSessionOpen := session.Open(); errSessionOpen != nil {
		return errors.Join(errSessionOpen, ErrDiscordOpen)
	}

	return nil

}

//
// func (discord *Discord) onHistoryChat(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, response *botResponse) error {
//	steamId, errResolveSID := resolveSID(ctx, interaction.Data.Options[0].Options[0].Value.(string))
//	if errResolveSID != nil {
//		return consts.ErrInvalidSID
//	}
//	Person := model.NewPerson(steamId)
//	if errPersonBySID := PersonBySID(ctx, discord.database, steamId, "", &Person); errPersonBySID != nil {
//		return errCommandFailed
//	}
//	chatHistory, errChatHistory := discord.database.GetChatHistory(ctx, steamId, 25)
//	if errChatHistory != nil && !errors.Is(errChatHistory, db.ErrNoResult) {
//		return errCommandFailed
//	}
//	if errors.Is(errChatHistory, db.ErrNoResult) {
//		return errors.New("No chat history found")
//	}
//	var lines []string
//	for _, sayEvent := range chatHistory {
//		lines = append(lines, fmt.Sprintf("%s: %s", Config.FmtTimeShort(sayEvent.CreatedOn), sayEvent.Msg))
//	}
//	embed := respOk(response, fmt.Sprintf("Chat History of: %s", Person.PersonaName))
//	embed.Description = strings.Join(lines, "\n")
//	return nil
// }

func (h *discordService) getDiscordAuthor(ctx context.Context, interaction *discordgo.InteractionCreate) (person.Person, error) {
	author, errPersonByDiscordID := h.persons.GetPersonByDiscordID(ctx, interaction.Member.User.ID)
	if errPersonByDiscordID != nil {
		if errors.Is(errPersonByDiscordID, database.ErrNoResult) {
			return author, domain.ErrSteamUnset
		}

		return author, domain.ErrFetchSource
	}

	return author, nil
}

func (bot *discordService) RegisterHandler(cmd Cmd, handler SlashCommandHandler) error {
	_, found := bot.commandHandlers[cmd]
	if found {
		return ErrDuplicateCommand
	}

	bot.commandHandlers[cmd] = handler

	return nil
}

func (bot *discordService) Shutdown() {
	if bot.session != nil {
		defer log.Closer(bot.session)
	}
}

func (bot *discordService) onReady(session *discordgo.Session, _ *discordgo.Ready) {
	slog.Info("Discord state changed", slog.String("state", "ready"), slog.String("username",
		fmt.Sprintf("%v#%v", session.State.User.Username, session.State.User.Discriminator)))
}

func (bot *discordService) onConnect(_ *discordgo.Session, _ *discordgo.Connect) {
	if errRegister := bot.botRegisterSlashCommands(bot.appID); errRegister != nil {
		slog.Error("Failed to register discord slash commands", log.ErrAttr(errRegister))
	}

	status := discordgo.UpdateStatusData{
		IdleSince: nil,
		Activities: []*discordgo.Activity{
			{
				Name:     "Cheeseburgers",
				Type:     discordgo.ActivityTypeListening,
				URL:      bot.externalURL,
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

func (bot *discordService) onDisconnect(_ *discordgo.Session, _ *discordgo.Disconnect) {
	bot.isReady.Store(false)

	slog.Info("Discord state changed", slog.String("state", "disconnected"))
}

// onInteractionCreate is called when a user initiates an application command. All commands are sent
// through this interface.
// https://discord.com/developers/docs/interactions/receiving-and-responding#receiving-an-interaction
func (bot *discordService) onInteractionCreate(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
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
				slog.Error("Failed sending error response for interaction", log.ErrAttr(errFollow))
			}

			return
		}

		commandCtx, cancelCommand := context.WithTimeout(context.TODO(), time.Second*30)
		defer cancelCommand()

		response, errHandleCommand := handler(commandCtx, session, interaction)
		if errHandleCommand != nil || response == nil {
			if _, errFollow := session.FollowupMessageCreate(interaction.Interaction, true, &discordgo.WebhookParams{
				Embeds: []*discordgo.MessageEmbed{message.ErrorMessage(string(command), errHandleCommand)},
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

func (bot *discordService) sendInteractionResponse(session *discordgo.Session, interaction *discordgo.Interaction, response *discordgo.MessageEmbed) error {
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
			return errors.Join(errResp, ErrDiscordMessageSen)
		}

		return nil
	}

	return nil
}

func (bot *discordService) SendPayload(channelID string, payload *discordgo.MessageEmbed) {
	if !bot.isReady.Load() {
		return
	}

	if _, errSend := bot.session.ChannelMessageSendEmbed(channelID, payload); errSend != nil {
		slog.Error("Failed to send discord payload", log.ErrAttr(errSend))
	}
}

//nolint:funlen,maintidx
func (bot *discordService) botRegisterSlashCommands(appID string) error {
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
	// }
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
			Name:                     string(CmdAC),
			Description:              "Query Anticheat Logs",
			DefaultMemberPermissions: &modPerms,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        string(CmdACPlayer),
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

	commands, errBulk := bot.session.ApplicationCommandBulkOverwrite(appID, bot.guildID, slashCommands)
	if errBulk != nil {
		return errors.Join(errBulk, ErrDiscordOverwriteCommands)
	}

	bot.commands = commands

	slog.Debug("Registered discord commands", slog.Int("count", len(slashCommands)))

	return nil
}
