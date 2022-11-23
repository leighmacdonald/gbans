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
	"time"
)

const (
	maxEmbedFields = 25
	// TODO ensure these
	//minUsernameChars    = 2
	//maxUsernameChars    = 32
	//maxAuthorChars      = 256
	maxFieldNameChars   = 256
	maxFieldValueChars  = 1024
	maxDescriptionChars = 2048
)

var (
	errCommandFailed = errors.New("Command failed")
	errTooLarge      = errors.Errorf("Max message length is %d", discordMaxMsgLen)
)

func (bot *Discord) SendEmbed(channelId string, message *discordgo.MessageEmbed) error {
	if bot.session == nil {
		return nil
	}
	if _, errSend := bot.session.ChannelMessageSendEmbed(channelId, message); errSend != nil {
		return errSend
	}
	return nil
}

// Discord implements the ChatBot interface for the discord chat platform.
type Discord struct {
	session            *discordgo.Session
	app                *App
	ctx                context.Context
	database           store.Store
	connectedMu        *sync.RWMutex
	commandHandlers    map[botCmd]botCommandHandler
	botSendMessageChan chan discordPayload
	initReadySent      bool
	Ready              bool
	retryCount         int64
	lastRetry          time.Time
}

// NewDiscord instantiates a new, unconnected, discord instance
func NewDiscord(ctx context.Context, app *App, database store.Store) (*Discord, error) {
	bot := Discord{
		ctx:         ctx,
		app:         app,
		session:     nil,
		database:    database,
		connectedMu: &sync.RWMutex{},
		// Only update the automod on first connect
		initReadySent: false,
		Ready:         false,
		retryCount:    -1,
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

func (bot *Discord) Start(ctx context.Context, token string) error {
	// Immediately connects, so we connect within the Start func
	session, errNewSession := discordgo.New("Bot " + token)
	if errNewSession != nil {
		return errors.Wrapf(errNewSession, "Failed to connect to discord. discord unavailable")
	}
	defer func() {
		if bot.session != nil {
			if errDisc := bot.session.Close(); errDisc != nil {
				log.Errorf("Failed to cleanly shutdown discord: %v", errDisc)
			}
		}
	}()

	session.UserAgent = "gbans (https://github.com/leighmacdonald/gbans)"
	session.AddHandler(bot.onReady)
	session.AddHandler(bot.onConnect)
	session.AddHandler(bot.onDisconnect)
	session.AddHandler(bot.onInteractionCreate)

	session.Identify.Intents |= discordgo.IntentsGuildMessages
	session.Identify.Intents |= discordgo.IntentAutoModerationExecution
	session.Identify.Intents |= discordgo.IntentMessageContent
	session.Identify.Intents |= discordgo.PermissionModerateMembers

	// Open a websocket connection to discord and begin listening.
	if errSessionOpen := session.Open(); errSessionOpen != nil {
		return errors.Wrap(errSessionOpen, "Error opening discord connection")
	}

	bot.session = session
	if errRegister := bot.botRegisterSlashCommands(); errRegister != nil {
		log.Errorf("Failed to register discord slash commands: %v", errRegister)
	}
	<-ctx.Done()
	return nil
}

func (bot *Discord) onReady(session *discordgo.Session, _ *discordgo.Ready) {
	log.WithFields(log.Fields{"service": "discord", "state": "ready"}).Infof("Discord state changed")
	bot.connectedMu.RLock()
	ready := bot.initReadySent
	bot.connectedMu.RUnlock()
	if !ready && config.Discord.AutoModEnable {
		bot.connectedMu.Lock()
		bot.initReadySent = true
		bot.connectedMu.Unlock()

		filters, errFilters := bot.database.GetFilters(bot.ctx)
		if errFilters != nil {
			return
		}
		var patterns []string
		for _, filter := range filters {
			for _, p := range filter.Patterns {
				patterns = append(patterns, p.String())
			}
		}
		if len(patterns) > 10 {
			patterns = patterns[0:10]
		}
		create := &discordgo.AutoModerationRule{
			Name:        "gbans automod",
			EventType:   discordgo.AutoModerationEventMessageSend,
			TriggerType: discordgo.AutoModerationEventTriggerKeyword,
			TriggerMetadata: &discordgo.AutoModerationTriggerMetadata{
				RegexPatterns: patterns,
			},
			Enabled: &config.Discord.AutoModEnable,
			Actions: []discordgo.AutoModerationAction{
				{Type: discordgo.AutoModerationRuleActionBlockMessage},
			},
			ExemptChannels: &config.Discord.ModChannels,
			ExemptRoles:    &[]string{},
		}
		_, errRule := session.AutoModerationRuleCreate(config.Discord.GuildID, create)
		if errRule != nil {
			log.Errorf("Failed to register word filter rule: %v", errRule)
		}
	}
	bot.connectedMu.Lock()
	bot.Ready = true
	bot.connectedMu.Unlock()
}

func (bot *Discord) onConnect(session *discordgo.Session, _ *discordgo.Connect) {
	status := discordgo.UpdateStatusData{
		IdleSince: nil,
		Activities: []*discordgo.Activity{
			{
				Name:     "Cheeseburgers",
				Type:     discordgo.ActivityTypeListening,
				URL:      config.General.ExternalUrl,
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
	log.WithFields(log.Fields{"service": "discord", "state": "connected"}).Infof("Discord state changed")
}

func (bot *Discord) onDisconnect(_ *discordgo.Session, _ *discordgo.Disconnect) {
	bot.connectedMu.Lock()
	bot.Ready = false
	bot.retryCount++
	bot.connectedMu.Unlock()
	log.WithFields(log.Fields{"service": "discord", "state": "disconnected"}).Infof("Discord state changed")
	if bot.retryCount > 0 {
		time.Sleep(time.Duration(bot.retryCount * int64(time.Second) * 5))
	}
}

func (bot *Discord) sendChannelMessage(session *discordgo.Session, channelId string, msg string, wrap bool) error {
	bot.connectedMu.RLock()
	if !bot.Ready {
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

func (bot *Discord) sendInteractionMessageEdit(session *discordgo.Session, interaction *discordgo.Interaction, response botResponse) error {
	bot.connectedMu.RLock()
	if !bot.Ready {
		bot.connectedMu.RUnlock()
		log.Warnf("Tried to send message to disconnected client")
		return nil
	}
	bot.connectedMu.RUnlock()
	edit := &discordgo.WebhookEdit{
		Embeds:          nil,
		AllowedMentions: nil,
	}
	var embeds []*discordgo.MessageEmbed
	switch response.MsgType {
	case mtString:
		val, ok := response.Value.(string)
		if ok && val != "" {
			edit.Content = &val
			if len(*edit.Content) > discordMaxMsgLen {
				return errTooLarge
			}
		}
	case mtEmbed:
		embeds = append(embeds, response.Value.(*discordgo.MessageEmbed))
		edit.Embeds = &embeds
	}
	_, errResp := session.InteractionResponseEdit(interaction, edit)
	return errResp
}

func (bot *Discord) Send(channelId string, message string, wrap bool) error {
	return bot.sendChannelMessage(bot.session, channelId, message, wrap)
}

func addFieldInline(embed *discordgo.MessageEmbed, title string, value string) {
	addFieldRaw(embed, title, value, true)
}

func addField(embed *discordgo.MessageEmbed, title string, value string) {
	addFieldRaw(embed, title, value, false)
}

//func addAuthor(embed *discordgo.MessageEmbed, person model.Person) {
//	name := person.PersonaName
//	if name == "" {
//		name = person.SteamID.String()
//	}
//	embed.Author = &discordgo.MessageEmbedAuthor{URL: person.ToURL(), Name: name}
//}

func addAuthorProfile(embed *discordgo.MessageEmbed, person model.UserProfile) {
	name := person.Name
	if name == "" {
		name = person.SteamID.String()
	}
	embed.Author = &discordgo.MessageEmbedAuthor{URL: person.ToURL(), Name: name}
}

func addLink(embed *discordgo.MessageEmbed, value model.Linkable) {
	url := value.ToURL()
	if len(url) > 0 {
		addFieldRaw(embed, "Link", url, false)
	}
}

func addFieldRaw(embed *discordgo.MessageEmbed, title string, value string, inline bool) {
	if len(embed.Fields) >= maxEmbedFields {
		log.Warnf("Dropping embed fields. Already at max count: %d", maxEmbedFields)
		return
	}
	if len(title) == 0 {
		log.Warnf("Title cannot be empty, dropping field")
		return
	}
	if len(value) == 0 {
		log.Warnf("Value cannot be empty, dropping field: %s", title)
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
	addFieldInline(embed, "Patterns", filter.Patterns.String())
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
		Color:       DefaultLevelColors.LevelColor(entry.Level),
		//Footer:      &defaultFooter,
		Provider: &defaultProvider,
		//Author:   &discordgo.MessageEmbedAuthor{Name: "gbans"},
	}
	fieldCount := 0
	for name, value := range entry.Data {
		var msg string
		switch typedValue := value.(type) {
		case string:
			msg = typedValue
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
		embed:     embed,
	}:
	default:
		// errors.New("Failed to write discord logger msg: chan full")
		return nil
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
