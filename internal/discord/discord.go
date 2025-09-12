package discord

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/discord/message"
	"github.com/leighmacdonald/gbans/pkg/log"
)

var (
	ErrDiscordOverwriteCommands = errors.New("failed to bulk overwrite discord commands")
	ErrDiscordConfig            = errors.New("invalid config")
	ErrDiscordCreate            = errors.New("failed to connect to discord")
	ErrDiscordOpen              = errors.New("failed to open discord connection")
	ErrDiscordMessageSend       = errors.New("failed to send discord message")
	ErrDuplicateCommand         = errors.New("duplicate command registration")
)

type SlashCommandHandler func(ctx context.Context, s *discordgo.Session, m *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error)

type discordService struct {
	session         *discordgo.Session
	isReady         atomic.Bool
	commandHandlers map[string]SlashCommandHandler
	commands        []*discordgo.ApplicationCommand
	token           string
	appID           string
	guildID         string
	externalURL     string
}

func NewDiscordHandler(appID string, guildID string, token string, externalURL string) (*discordService, error) {
	if appID == "" || guildID == "" || token == "" {
		return nil, ErrDiscordConfig
	}

	bot := &discordService{
		isReady:         atomic.Bool{},
		commandHandlers: map[string]SlashCommandHandler{},
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

// func (h *discordService) getDiscordAuthor(ctx context.Context, interaction *discordgo.InteractionCreate) (person.Person, error) {
// 	author, errPersonByDiscordID := h.persons.GetPersonByDiscordID(ctx, interaction.Member.User.ID)
// 	if errPersonByDiscordID != nil {
// 		if errors.Is(errPersonByDiscordID, database.ErrNoResult) {
// 			return author, domain.ErrSteamUnset
// 		}

// 		return author, domain.ErrFetchSource
// 	}

// 	return author, nil
// }

func (bot *discordService) RegisterHandler(cmd string, handler SlashCommandHandler) error {
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
		command = data.Name
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
			return errors.Join(errResp, ErrDiscordMessageSend)
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
	commands, errBulk := bot.session.ApplicationCommandBulkOverwrite(appID, bot.guildID, bot.commands)
	if errBulk != nil {
		return errors.Join(errBulk, ErrDiscordOverwriteCommands)
	}

	bot.commands = commands

	slog.Debug("Registered discord commands", slog.Int("count", len(commands)))

	return nil
}

type nullDiscordRepository struct{}

func (bot *nullDiscordRepository) RegisterHandler(_ string, _ SlashCommandHandler) error {
	return nil
}

func (bot *nullDiscordRepository) Shutdown() {
}

func (bot *nullDiscordRepository) Start() error {
	return nil
}

func (bot *nullDiscordRepository) SendPayload(_ string, _ *discordgo.MessageEmbed) {
}

func NewNullDiscordRepository() *nullDiscordRepository {
	return &nullDiscordRepository{}
}
