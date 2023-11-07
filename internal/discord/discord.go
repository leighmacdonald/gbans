package discord

import (
	"fmt"
	"sync/atomic"

	"github.com/bwmarrin/discordgo"
	embed "github.com/leighmacdonald/discordgo-embed"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var ErrCommandFailed = errors.New("Command failed")

type Bot struct {
	log               *zap.Logger
	session           *discordgo.Session
	isReady           atomic.Bool
	commandHandlers   map[Cmd]CommandHandler
	Colour            LevelColors
	unregisterOnStart bool
	appID             string
	extURL            string
}

const (
	iconURL      = "https://cdn.discordapp.com/avatars/758536119397646370/6a371d1a481a72c512244ba9853f7eff.webp?size=128"
	providerName = "gbans"
)

type Payload struct {
	ChannelID string
	Embed     *discordgo.MessageEmbed
}

func NewEmbed(args ...string) *embed.Embed {
	newEmbed := embed.
		NewEmbed().
		SetFooter(providerName, iconURL)

	if len(args) == 2 {
		newEmbed = newEmbed.SetTitle(args[0]).
			SetDescription(args[1])
	} else if len(args) == 1 {
		newEmbed = newEmbed.SetTitle(args[0])
	}

	return newEmbed
}

func AddFieldsSteamID(embed *embed.Embed, steamID steamid.SID64) *embed.Embed {
	embed.AddField("STEAM", string(steamid.SID64ToSID(steamID))).MakeFieldInline()
	embed.AddField("STEAM3", string(steamid.SID64ToSID3(steamID))).MakeFieldInline()
	embed.AddField("SID64", steamID.String()).MakeFieldInline()

	return embed
}

func New(logger *zap.Logger, token string, appID string, unregisterOnStart bool, extURL string) (*Bot, error) {
	// Immediately connects
	session, errNewSession := discordgo.New("Bot " + token)
	if errNewSession != nil {
		return nil, errors.Wrapf(errNewSession, "Failed to connect to discord. discord unavailable")
	}

	session.UserAgent = "gbans (https://github.com/leighmacdonald/gbans)"
	session.Identify.Intents |= discordgo.IntentsGuildMessages
	session.Identify.Intents |= discordgo.IntentMessageContent
	session.Identify.Intents |= discordgo.IntentGuildMembers
	bot := &Bot{
		log:               logger.Named("discord"),
		session:           session,
		isReady:           atomic.Bool{},
		unregisterOnStart: unregisterOnStart,
		appID:             appID,
		extURL:            extURL,
		commandHandlers:   map[Cmd]CommandHandler{},
		Colour: LevelColors{
			Success: 302673,
			Debug:   10170623,
			Info:    3581519,
			Warn:    14327864,
			Error:   13631488,
			Fatal:   13631488,
		},
	}
	bot.session.AddHandler(bot.onReady)
	bot.session.AddHandler(bot.onConnect)
	bot.session.AddHandler(bot.onDisconnect)
	bot.session.AddHandler(bot.onInteractionCreate)

	return bot, nil
}

func (bot *Bot) RegisterHandler(cmd Cmd, handler CommandHandler) error {
	_, found := bot.commandHandlers[cmd]
	if found {
		return errors.New("Duplicate command")
	}

	bot.commandHandlers[cmd] = handler

	return nil
}

func (bot *Bot) Shutdown(guildID string) {
	if bot.session != nil {
		defer util.LogCloser(bot.session, bot.log)
		bot.botUnregisterSlashCommands(guildID)
	}
}

func (bot *Bot) botUnregisterSlashCommands(guildID string) {
	registeredCommands, err := bot.session.ApplicationCommands(bot.session.State.User.ID, guildID)
	if err != nil {
		bot.log.Error("Could not fetch registered commands", zap.Error(err))

		return
	}

	for _, v := range registeredCommands {
		if errDel := bot.session.ApplicationCommandDelete(bot.session.State.User.ID, guildID, v.ID); errDel != nil {
			bot.log.Error("Cannot delete command", zap.String("name", v.Name), zap.Error(err))

			return
		}
	}

	bot.log.Info("Unregistered discord commands", zap.Int("count", len(registeredCommands)))
}

func (bot *Bot) Start() error {
	// Open a websocket connection to discord and begin listening.
	if errSessionOpen := bot.session.Open(); errSessionOpen != nil {
		return errors.Wrap(errSessionOpen, "Error opening discord connection")
	}

	if bot.unregisterOnStart {
		bot.botUnregisterSlashCommands("")
	}

	return nil
}

func (bot *Bot) onReady(session *discordgo.Session, _ *discordgo.Ready) {
	bot.log.Info("Service state changed", zap.String("state", "ready"), zap.String("username",
		fmt.Sprintf("%v#%v", session.State.User.Username, session.State.User.Discriminator)))
}

func (bot *Bot) onConnect(_ *discordgo.Session, _ *discordgo.Connect) {
	if errRegister := bot.botRegisterSlashCommands(bot.appID); errRegister != nil {
		bot.log.Error("Failed to register discord slash commands", zap.Error(errRegister))
	}

	status := discordgo.UpdateStatusData{
		IdleSince: nil,
		Activities: []*discordgo.Activity{
			{
				Name:     "Cheeseburgers",
				Type:     discordgo.ActivityTypeListening,
				URL:      bot.extURL,
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
		bot.log.Error("Failed to update status complex", zap.Error(errUpdateStatus))
	}

	bot.log.Info("Service state changed", zap.String("state", "connected"))

	bot.isReady.Store(true)
}

func (bot *Bot) onDisconnect(_ *discordgo.Session, _ *discordgo.Disconnect) {
	bot.isReady.Store(false)

	bot.log.Info("Service state changed", zap.String("state", "disconnected"))
}

// func sendChannelMessage(session *discordgo.Session, channelId string, msg string, wrap bool) error {
//	if !isReady.Load() {
//		log.Error("Tried to send message to disconnected client")
//		return nil
//	}
//	if wrap {
//		msg = discordMsgWrapper + msg + discordMsgWrapper
//	}
//	if len(msg) > discordMaxMsgLen {
//		return ErrTooLarge
//	}
//	_, errChannelMessageSend := session.ChannelMessageSend(channelId, msg)
//	if errChannelMessageSend != nil {
//		return errors.Wrapf(errChannelMessageSend, "Failed sending success (paged) response for interaction")
//	}
//	return nil
//}

func (bot *Bot) sendInteractionResponse(session *discordgo.Session, interaction *discordgo.Interaction, response *discordgo.MessageEmbed) error {
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
			return errors.Wrap(errResp, "Failed to send error response")
		}

		return nil
	}

	return nil
}

func (bot *Bot) SendPayload(payload Payload) {
	if !bot.isReady.Load() {
		return
	}

	if _, errSend := bot.session.ChannelMessageSendEmbed(payload.ChannelID, payload.Embed); errSend != nil {
		bot.log.Error("Failed to send discord payload", zap.Error(errSend))
	}
}

// LevelColors is a struct of the possible colors used in Discord color format (0x[RGB] converted to int).
type LevelColors struct {
	Debug   int
	Success int
	Info    int
	Warn    int
	Error   int
	Fatal   int
}
