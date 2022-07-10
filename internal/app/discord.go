package app

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"sync"
)

const (
	maxEmbedFields      = 25
	minUsernameChars    = 2
	maxUsernameChars    = 32
	maxAuthorChars      = 256
	maxFieldNameChars   = 256
	maxFieldValueChars  = 1024
	maxDescriptionChars = 2048
)

var (
	errCommandFailed = errors.New("Command failed")
	errTooLarge      = errors.Errorf("Max message length is %d", discordMaxMsgLen)
)

func (bot *discord) SendEmbed(channelId string, message *discordgo.MessageEmbed) error {
	if _, errSend := bot.session.ChannelMessageSendEmbed(channelId, message); errSend != nil {
		return errSend
	}
	return nil
}

// discord implements the ChatBot interface for the discord chat platform.
type discord struct {
	session            *discordgo.Session
	ctx                context.Context
	database           store.Store
	connectedMu        *sync.RWMutex
	connected          bool
	commandHandlers    map[botCmd]botCommandHandler
	botSendMessageChan chan discordPayload
}

// NewDiscord instantiates a new, unconnected, discord instance
func NewDiscord(ctx context.Context, database store.Store) (*discord, error) {
	bot := discord{
		ctx:         ctx,
		session:     nil,
		database:    database,
		connectedMu: &sync.RWMutex{},
		connected:   false,
	}
	bot.commandHandlers = map[botCmd]botCommandHandler{
		cmdBan:      bot.onBan,
		cmdCheck:    bot.onCheck,
		cmdCSay:     bot.onCSay,
		cmdFind:     bot.onFind,
		cmdKick:     bot.onKick,
		cmdMute:     bot.onMute,
		cmdPlayers:  bot.onPlayers,
		cmdPSay:     bot.onPSay,
		cmdSay:      bot.onSay,
		cmdServers:  bot.onServers,
		cmdUnban:    bot.onUnban,
		cmdSetSteam: bot.onSetSteam,
		cmdHistory:  bot.onHistory,
		cmdFilter:   bot.onFilter,
		cmdLog:      bot.onLog,
		//cmdStats:    bot.onStats,
	}
	return &bot, nil
}

func (bot *discord) Start(ctx context.Context, token string) error {
	// Immediately connects, so we connect within the Start func
	session, errNewSession := discordgo.New("Bot " + token)
	if errNewSession != nil {
		return errors.Wrapf(errNewSession, "Failed to connect to discord. discord unavailable")

	}
	defer func() {
		if errDisc := bot.session.Close(); errDisc != nil {
			log.Errorf("Failed to cleanly shutdown discord: %v", errDisc)
		}
	}()
	bot.session = session
	bot.session.UserAgent = "gbans (https://github.com/leighmacdonald/gbans)"
	bot.session.AddHandler(bot.onReady)
	bot.session.AddHandler(bot.onConnect)
	bot.session.AddHandler(bot.onDisconnect)
	bot.session.AddHandler(bot.onInteractionCreate)
	bot.session.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)

	// Open a websocket connection to discord and begin listening.
	if errSessionOpen := bot.session.Open(); errSessionOpen != nil {
		return errors.Wrap(errSessionOpen, "Error opening discord connection")
	}
	if errRegister := bot.botRegisterSlashCommands(); errRegister != nil {
		log.Errorf("Failed to register discord slash commands: %v", errRegister)
	}
	<-ctx.Done()
	return nil
}

func (bot *discord) onReady(_ *discordgo.Session, _ *discordgo.Ready) {
	log.WithFields(log.Fields{"service": "discord", "status": "ready"}).Infof("Service status changed")
}

func (bot *discord) onConnect(session *discordgo.Session, _ *discordgo.Connect) {
	log.Tracef("Connected to session ws API")
	status := discordgo.UpdateStatusData{
		IdleSince: nil,
		Activities: []*discordgo.Activity{
			{
				Name:     "Cheeseburgers",
				Type:     discordgo.ActivityTypeListening,
				URL:      "https://" + config.HTTP.Addr(),
				State:    "state field",
				Details:  "Blah",
				Instance: true,
				Flags:    1 << 0,
			},
		},
		AFK:    false,
		Status: "https://github.com/leighmacdonald/gbans",
	}
	if errUpdateStatus := session.UpdateStatusComplex(status); errUpdateStatus != nil {
		log.WithError(errUpdateStatus).Errorf("Failed to update status complex")
	}
	bot.connectedMu.Lock()
	bot.connected = true
	bot.connectedMu.Unlock()
}

func (bot *discord) onDisconnect(_ *discordgo.Session, _ *discordgo.Disconnect) {
	bot.connectedMu.Lock()
	bot.connected = false
	bot.connectedMu.Unlock()
	log.Info("Disconnected from session ws API")
}

func (bot *discord) sendChannelMessage(session *discordgo.Session, channelId string, msg string, wrap bool) error {
	bot.connectedMu.RLock()
	if !bot.connected {
		bot.connectedMu.RUnlock()
		log.Warnf("Tried to send message to disconnected client")
		return nil
	}
	bot.connectedMu.RUnlock()
	if wrap {
		msg = discordMsgWrapper + msg + discordMsgWrapper
	}
	if len(msg) > discordMaxMsgLen {
		return errTooLarge
	}
	_, errChannelMessageSend := session.ChannelMessageSend(channelId, msg)
	if errChannelMessageSend != nil {
		return errors.Wrapf(errChannelMessageSend, "Failed sending success (paged) response for interaction")
	}
	return nil
}

func (bot *discord) sendInteractionMessageEdit(session *discordgo.Session, interaction *discordgo.Interaction, response botResponse) error {
	bot.connectedMu.RLock()
	if !bot.connected {
		bot.connectedMu.RUnlock()
		log.Warnf("Tried to send message to disconnected client")
		return nil
	}
	bot.connectedMu.RUnlock()
	edit := &discordgo.WebhookEdit{
		Content:         "",
		Embeds:          nil,
		AllowedMentions: nil,
	}
	switch response.MsgType {
	case mtString:
		val, ok := response.Value.(string)
		if ok && val != "" {
			edit.Content = val
			if len(edit.Content) > discordMaxMsgLen {
				return errTooLarge
			}
		}
	case mtEmbed:
		edit.Embeds = append(edit.Embeds, response.Value.(*discordgo.MessageEmbed))
	}
	return session.InteractionResponseEdit(config.Discord.AppID, interaction, edit)
}

func (bot *discord) Send(channelId string, message string, wrap bool) error {
	return bot.sendChannelMessage(bot.session, channelId, message, wrap)
}

func addFieldInline(embed *discordgo.MessageEmbed, title string, value string) {
	addFieldRaw(embed, title, value, true)
}

func addField(embed *discordgo.MessageEmbed, title string, value string) {
	addFieldRaw(embed, title, value, false)
}

func addFieldRaw(embed *discordgo.MessageEmbed, title string, value string, inline bool) {
	if len(embed.Fields) >= maxEmbedFields {
		log.Warnf("Dropping embed fields. Already at max count: %d", maxEmbedFields)
		return
	}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   truncate(title, maxFieldNameChars),
		Value:  truncate(value, maxFieldValueChars),
		Inline: inline,
	})
}

func truncate(str string, maxLen int) string {
	if len(str) > maxLen {
		return str[:maxLen]
	}
	return str
}

func addFieldsSteamID(embed *discordgo.MessageEmbed, steamId steamid.SID64) {
	addFieldInline(embed, "STEAM", string(steamid.SID64ToSID(steamId)))
	addFieldInline(embed, "STEAM3", string(steamid.SID64ToSID3(steamId)))
	addFieldInline(embed, "SID64", steamId.String())
}

func addFieldFilter(embed *discordgo.MessageEmbed, filter model.Filter) {
	addFieldInline(embed, "Pattern", filter.Pattern.String())
	addFieldInline(embed, "ID", fmt.Sprintf("%d", filter.WordID))
}

// ChatBot defines a interface for communication with 3rd party service bots
// Currently this is only used for discord, but other providers such as
// Guilded, Matrix, IRC, etc. are planned.
// TODO decouple embed's from discordgo
type ChatBot interface {
	Start(ctx context.Context, token string, eventChan chan model.ServerEvent) error
	Send(channelId string, message string, wrap bool) error
	SendEmbed(channelId string, message *discordgo.MessageEmbed) error
}

type DiscordLogHook struct {
	MinLevel    log.Level
	messageChan chan discordPayload
}

func NewDiscordLogHook(messageChan chan discordPayload) *DiscordLogHook {
	return &DiscordLogHook{
		messageChan: messageChan,
		MinLevel:    log.DebugLevel,
	}
}

func (hook *DiscordLogHook) Fire(entry *log.Entry) error {
	//title := entry.Message
	//if title == "" {
	//	title = "Log Message"
	//}
	embed := &discordgo.MessageEmbed{
		Type: discordgo.EmbedTypeRich,
		//Title:       title,
		Description: truncate(entry.Message, maxDescriptionChars),
		Color:       int(green),
		//Footer:      &defaultFooter,
		Provider: &defaultProvider,
		//Author:   &discordgo.MessageEmbedAuthor{Name: "gbans"},
	}
	fieldCount := 0
	for name, value := range entry.Data {
		var msg string
		switch value.(type) {
		case string:
			msg = value.(string)
		case int:
			msg = fmt.Sprintf("%d", value)
		case int64:
			msg = fmt.Sprintf("%d", value)
		case uint:
			msg = fmt.Sprintf("%d", value)
		case uint64:
			msg = fmt.Sprintf("%d", value)
		default:
			msg = fmt.Sprintf("%v", value)
		}
		if len(msg) > 40 {
			addField(embed, name, msg)
		} else {
			addFieldInline(embed, name, msg)
		}
		fieldCount++
		if fieldCount == maxEmbedFields {
			break
		}
	}
	select {
	case hook.messageChan <- discordPayload{
		channelId: config.Discord.LogChannelID,
		message:   embed,
	}:
	default:
		return errors.New("Failed to write discord logger msg: chan full")
	}
	return nil
}

func (hook *DiscordLogHook) Levels() []log.Level {
	return LevelThreshold(hook.MinLevel)
}

// LevelColors is a struct of the possible colors used in Discord color format (0x[RGB] converted to int)
type LevelColors struct {
	Trace int
	Debug int
	Info  int
	Warn  int
	Error int
	Panic int
	Fatal int
}

// DefaultLevelColors is a struct of the default colors used
var DefaultLevelColors = LevelColors{
	Trace: 3092790,
	Debug: 10170623,
	Info:  3581519,
	Warn:  14327864,
	Error: 13631488,
	Panic: 13631488,
	Fatal: 13631488,
}

// LevelThreshold returns a slice of all the levels above and including the level specified
func LevelThreshold(level log.Level) []log.Level {
	return log.AllLevels[:level+1]
}

// LevelColor returns the respective color for the logrus level
func (lc LevelColors) LevelColor(level log.Level) int {
	switch level {
	case log.TraceLevel:
		return lc.Trace
	case log.DebugLevel:
		return lc.Debug
	case log.InfoLevel:
		return lc.Info
	case log.WarnLevel:
		return lc.Warn
	case log.ErrorLevel:
		return lc.Error
	case log.PanicLevel:
		return lc.Panic
	case log.FatalLevel:
		return lc.Fatal
	default:
		return lc.Warn
	}
}
