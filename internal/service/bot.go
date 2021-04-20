package service

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"strings"
)

type cmdHandler func(s *discordgo.Session, _ *discordgo.MessageCreate, args ...string) error

type cmdDef struct {
	help    string
	handler cmdHandler
	minArgs int
	maxArgs int
}

var (
	dg                 *discordgo.Session
	messageQueue       chan logMessage
	modChannelIDs      []string
	cmdMap             map[string]cmdDef
	connected          bool
	errUnknownCommand  = errors.New("Unknown command")
	errInvalidSID      = errors.New("Invalid steamid")
	errUnknownID       = errors.New("Could not find matching player/steamid")
	errCommandFailed   = errors.New("Command failed")
	errDuplicateBan    = errors.New("Duplicate ban")
	errInvalidDuration = errors.New("Invalid duration")
	//errInvalidArguments = errors.New("Invalid arguments")
	//errInvalidIP        = errors.New("Invalid ip")
)

func newCmd(help string, args string, handler cmdHandler, minArgs int, maxArgs int) cmdDef {
	return cmdDef{
		help:    fmt.Sprintf("%s -- `%s%s`", help, config.Discord.Prefix, args),
		handler: handler,
		minArgs: minArgs,
		maxArgs: maxArgs,
	}
}

func init() {
	messageQueue = make(chan logMessage)
	cmdMap = map[string]cmdDef{
		"help":    newCmd("Returns the command list", "help [command]", onHelp, 0, 1),
		"ban":     newCmd("Ban a player", "ban <name/id> <duration> [reason]", onBan, 1, -1),
		"banip":   newCmd("Ban an IP", "banip <CIDR> <duration> [reason]", onBanIP, 1, -1),
		"find":    newCmd("Find a user on the servers", "find <id>", onFind, 1, 1),
		"mute":    newCmd("Mute a player", "mute <name/id> <duration> [reason]", onMute, 1, -1),
		"check":   newCmd("Check if a user is banned", "check <id>", onCheck, 1, 1),
		"unban":   newCmd("Unban a player", "unban <id>", onUnban, 1, 1),
		"kick":    newCmd("Kick a player", "kick <id> [reason]", onKick, 1, -1),
		"players": newCmd("Get the players in the server", "players <server>", onPlayers, 1, 1),
		"psay":    newCmd("sendMessage a private message to the user", "psay <server> <id> <message>", onPSay, 3, -1),
		"csay":    newCmd("sendMessage a centered message to the server", "csay <server> <message>", onCSay, 2, -1),
		"say":     newCmd("sendMessage a message to the server", "say <server> <message>", onSay, 3, -1),
		"servers": newCmd("Get the server status for all servers", "servers", onServers, 0, 1),
	}
}

func startDiscord(ctx context.Context, token string, channelIDs []string) {
	modChannelIDs = channelIDs
	d, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Errorf("Failed to connect to dg. Bot unavailable")
		return
	}
	defer func() {
		if err := dg.Close(); err != nil {
			log.Errorf("Failed to cleanly shutdown discord: %v", err)
		}
	}()
	dg = d
	dg.AddHandler(onConnect)
	dg.AddHandler(onDisconnect)
	dg.AddHandler(onMessageCreate)
	dg.AddHandler(onHandleCommand)
	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}
	go discordMessageQueueReader(ctx)
	// Wait here until CTRL-C or other term signal is received.
	log.Infof("Bot is now running.  Press CTRL-C to exit.")
	go func() {
		if err2 := botRegisterSlashCommands(config.Discord.AppID, token); err2 != nil {
			log.Errorf("Failed to register discord slash commands: %v", err2)
		}
	}()
	<-ctx.Done()
}

func discordMessageQueueReader(ctx context.Context) {
	events := make(chan logEvent)
	if err := registerLogEventReader(events); err != nil {
		log.Warnf("logWriter Tried to register duplicate reader channel")
	}
	for {
		select {
		case dm := <-events:
			switch dm.Type {
			case logparse.SayTeam:
				fallthrough
			case logparse.Say:
				prefix := ""
				if dm.Type == logparse.SayTeam {
					prefix = "(team) "
				}
				for _, c := range config.Relay.ChannelIDs {
					sendMessage(newMessage(c, fmt.Sprintf("[%s] %d **%s** %s%s",
						dm.Server.ServerName, dm.Player1.SteamID, dm.Event["name"], prefix, dm.Event["msg"])))
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func onConnect(s *discordgo.Session, _ *discordgo.Connect) {
	log.Info("Connected to session ws API")
	d := discordgo.UpdateStatusData{
		Game: &discordgo.Game{
			Name:    `Uncletopia`,
			URL:     "git@github.com/leighmacdonald/gbans",
			Details: "Mr. Authority",
		},
	}
	if err := s.UpdateStatusComplex(d); err != nil {
		log.WithError(err).Errorf("Failed to update status complex")
	}
	connected = true
}

func onDisconnect(_ *discordgo.Session, _ *discordgo.Disconnect) {
	connected = false
	log.Info("Disconnected from session ws API")
}

func onHandleCommand(_ *discordgo.Session, m *discordgo.InteractionCreate) {
	log.Debugf("Got message create event: %v", m)
}

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}
	chanOK := false
	for _, cid := range modChannelIDs {
		if m.ChannelID == cid {
			chanOK = true
			break
		}
	}
	if !chanOK {
		return
	}
	if !strings.HasPrefix(m.Content, config.Discord.Prefix) {
		return
	}
	parts := strings.SplitN(m.Content, " ", 2)
	cmdStr := strings.ToLower(strings.TrimPrefix(parts[0], config.Discord.Prefix))

	cmd, ok := cmdMap[cmdStr]
	if !ok {
		sendErr(s, m.ChannelID,
			errors.Errorf("Invalid command: %s (%shelp)", cmdStr, config.Discord.Prefix))
	}
	var args []string
	if len(parts) == 2 {
		args = strings.Split(parts[1], " ")
	}
	if (cmd.minArgs != -1 && len(args) < cmd.minArgs) || (cmd.maxArgs != -1 && len(args) > cmd.maxArgs) {
		e := errors.Errorf("Invalid number of arguments, see %shelp %s for syntax", config.Discord.Prefix, cmdStr)
		sendErr(s, m.ChannelID, e)
		return
	}
	if err := cmd.handler(s, m, args...); err != nil {
		sendErr(s, m.ChannelID, err)
		return
	}
}

func sendMsg(s *discordgo.Session, cid string, msg string, args ...interface{}) error {
	if !connected {
		log.Warnf("Tried to send message to disconnected client")
		return nil
	}
	_, err := s.ChannelMessageSend(cid, fmt.Sprintf(msg, args...))
	return err
}

func sendErr(s *discordgo.Session, cid string, err error) {
	if !connected {
		log.Warnf("Tried to send error to disconnected client")
		return
	}
	if _, err := s.ChannelMessageSend(cid, err.Error()); err != nil {
		log.Errorf("Failed to send error message: %v", err)
	}
}

type logMessage struct {
	ServerID string
	Body     string
}

func newMessage(channel string, body string) logMessage {
	return logMessage{
		ServerID: channel,
		Body:     body,
	}
}

func sendMessage(message logMessage) {
	if config.Discord.Enabled {
		messageQueue <- message
	}
}
