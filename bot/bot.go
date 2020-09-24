package bot

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/config"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"regexp"
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
	modChannelIDs      []string
	cmdMap             map[string]cmdDef
	errUnknownCommand  = errors.New("Unknown command")
	errCommandFailed   = errors.New("Command failed")
	reStatusPlayerFull *regexp.Regexp
	reStatusPlayer     *regexp.Regexp
)

func newCmd(help string, handler cmdHandler, minArgs int, maxArgs int) cmdDef {
	return cmdDef{
		help:    help,
		handler: handler,
		minArgs: minArgs,
		maxArgs: maxArgs,
	}
}

func init() {
	reStatusPlayerFull = regexp.MustCompile(`^#\s+(\d+)\s+"(.+?)"\s+(\[U:\d:\d+])\s+(.+?)\s+(\d+)\s+(\d+)\s+(.+?)\s(.+?):(.+?)$`)
	reStatusPlayer = regexp.MustCompile(`^#\s+(\d+)\s+"(.+?)"\s+(\[U:\d:\d+])\s+(\d+:\d+)\s+(\d+)\s+(\d+)\s+(.+?)$`)
	cmdMap = map[string]cmdDef{
		"help":    newCmd("Returns the command list", onHelp, 0, 1),
		"ban":     newCmd("Ban a player | !ban <steam_id|name>", onBan, 1, 3),
		"banIP":   newCmd("Ban a IP", onBanIP, 1, 3),
		"check":   newCmd("Check if a user is banned -- !check <steam_id|ip>", onCheck, 1, 1),
		"unban":   newCmd("Unban a player -- !unban <steam_id>", onUnban, 1, 1),
		"kick":    newCmd("Kick a player -- !kick <server> <steam_id|name> [reason]", onKick, 1, 2),
		"players": newCmd("Get the players in the server -- !players <server>", onPlayers, 1, 1),
		"psay":    newCmd("Send a private message to the user", onPSay, 3, -1),
		"csay":    newCmd("Send a centered message to the server", onCSay, 3, -1),
		"say":     newCmd("Send a message to the server", onSay, 3, -1),
		"servers": newCmd("Get the server status for all servers", onServers, 0, 1),
	}
}

func Start(ctx context.Context, token string, channelIDs []string) {
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

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	log.Infof("Bot is now running.  Press CTRL-C to exit.")
	<-ctx.Done()
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
}

func onDisconnect(_ *discordgo.Session, _ *discordgo.Disconnect) {
	log.Info("Disconnected from session ws API")
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

func sendMsg(s *discordgo.Session, cid string, msg string) error {
	_, err := s.ChannelMessageSend(cid, msg)
	return err
}

func sendErr(s *discordgo.Session, cid string, err error) {
	if _, err := s.ChannelMessageSend(cid, err.Error()); err != nil {
		log.Errorf("Failed to send error message: %v", err)
	}
}
