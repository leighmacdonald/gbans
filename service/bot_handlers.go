package service

import (
	"database/sql"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/leighmacdonald/gbans/config"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"sort"
	"strings"
	"sync"
	"time"
)

type PlayerInfo struct {
	player *extra.Player
	server *model.Server
	sid    steamid.SID64
	inGame bool
	valid  bool
}

// findPlayer will attempt to match a input string to a steam id and if connected, a
// matching active player.
//
// Will accept SteamID or partial player names. When using a partial player name, the
// first instance that contains the partial match will be returned.
//
// valid will be set to true if the value is a valid steamid, even if the player is not
// actively connected
func findPlayer(playerStr string, ip string) PlayerInfo {
	var (
		player   *extra.Player
		server   *model.Server
		err      error
		sid      steamid.SID64
		inGame   = false
		foundSid steamid.SID64
		valid    = false
	)
	if ip != "" {
		player, server, err = findPlayerByIP(net.ParseIP(ip))
		if err == nil {
			foundSid = player.SID
			inGame = true
		}
	} else {
		sid, err := steamid.SID64FromString(playerStr)
		if err == nil && sid.Valid() {
			foundSid = sid
			player, server, err = findPlayerBySID(sid)
			if err == nil {
				inGame = true
				foundSid = sid
			}
		} else {
			player, server, err = findPlayerByName(playerStr)
			if err == nil {
				foundSid = player.SID
				inGame = true
			}
		}
	}
	if sid.Valid() || foundSid.Valid() {
		valid = true
	}

	return PlayerInfo{player, server, foundSid, inGame, valid}
}

func onFind(s *discordgo.Session, m *discordgo.MessageCreate, args ...string) error {
	const f = "Found player `%s` (%d) @ %s"
	pi := findPlayer(args[0], "")
	if !pi.valid || !pi.inGame {
		return errUnknownID
	}
	return sendMsg(s, m.ChannelID, fmt.Sprintf(f, pi.player.Name, pi.sid.Int64(), pi.server.ServerName))
}

func onMute(s *discordgo.Session, m *discordgo.MessageCreate, args ...string) error {
	var (
		err      error
		duration = time.Duration(0)
	)
	pi := findPlayer(args[0], "")
	if !pi.valid {
		return errUnknownID
	}
	if len(args) > 1 {
		duration, err = config.ParseDuration(args[1])
		if err != nil {
			return err
		}
	}
	reasonStr := model.ReasonString(model.Custom)
	if len(args) > 2 {
		reasonStr = strings.Join(args[2:], " ")
	}
	ban, err := GetBan(pi.sid)
	if err != nil && DBErr(err) != errNoResult {
		log.Errorf("Error getting ban from db: %v", err)
		return errors.New("Internal DB Error")
	} else if err != nil {
		ban = model.Ban{
			SteamID:  pi.sid,
			AuthorID: 0,
			Reason:   1,
			Note:     "",
		}
	}
	if ban.BanType == model.Banned {
		return errors.New("Person is already banned")
	}
	ban.BanType = model.NoComm
	ban.ReasonText = reasonStr
	ban.ValidUntil = config.Now().Add(duration)
	if err := SaveBan(&ban); err != nil {
		log.Errorf("Failed to save ban: %v", err)
		return errors.New("Failed to save mute state")
	}
	if pi.inGame {
		resp, err := execServerRCON(*pi.server, fmt.Sprintf(`sm_gag "#%s"`, steamid.SID64ToSID3(pi.sid)))
		if err != nil {
			log.Errorf("Failed to gag active user: %v", err)
		} else {
			if strings.Contains(resp, "[SM] Gagged") {
				var dStr string
				if duration.Seconds() == 0 {
					dStr = "Forever"
				} else {
					dStr = duration.String()
				}
				return sendMsg(s, m.ChannelID, fmt.Sprintf("Person gagged successfully for: %s", dStr))
			} else {
				return sendMsg(s, m.ChannelID, "Failed to gag player in-game")
			}
		}
	}
	return nil
}

func onBanIP(s *discordgo.Session, m *discordgo.MessageCreate, args ...string) error {
	var reason string
	if len(args) > 2 {
		reason = strings.Join(args[2:], " ")
	}
	duration, err := config.ParseDuration(args[1])
	if err != nil {
		return errInvalidDuration
	}
	_, err = getBanNet(net.ParseIP(args[0]))
	if err != nil && DBErr(err) != errNoResult {
		return errCommandFailed
	}
	if err == nil {
		return errDuplicateBan
	}
	ban, err := model.NewBanNet(args[0], reason, duration, model.Bot)
	if err != nil {
		return errCommandFailed
	}
	if err := SaveBanNet(&ban); err != nil {
		return errCommandFailed
	}
	_, n, err := net.ParseCIDR(args[0])
	if err != nil {
		return errCommandFailed
	}
	pi, srv, err := findPlayerByCIDR(n)
	if err == nil {
		if resp, err := execServerRCON(*srv, fmt.Sprintf("sm_kick %s", pi.Name)); err != nil {
			log.Debug(resp)
		}
	}
	return sendMsg(s, m.ChannelID, "IP ban created successfully")
}

// onBan !ban <id> <duration> [reason]
func onBan(s *discordgo.Session, m *discordgo.MessageCreate, args ...string) error {
	var reason string
	if len(args) > 2 {
		reason = strings.Join(args[2:], " ")
	}
	duration, err := config.ParseDuration(args[1])
	if err != nil {
		return errInvalidDuration
	}
	pi := findPlayer(args[0], "")
	if !pi.valid {
		return errUnknownID
	}
	err = BanPlayer(ctx, pi.sid, config.General.Owner, duration, model.Custom, reason, model.Bot)
	if err != nil {
		if err == errDuplicate {
			return sendMsg(s, m.ChannelID, "ID already banned")
		} else {
			return sendMsg(s, m.ChannelID, "Error banning: %s", err)
		}
	}
	return sendMsg(s, m.ChannelID, "Ban created successfully")
}

//goland:noinspection ALL
func onCheck(s *discordgo.Session, m *discordgo.MessageCreate, args ...string) error {
	const f = "[%s] Banned: `%v` -- Muted: `%v` -- IP: `%s` -- Expires In: `%s` Reason: `%s`"
	pi := findPlayer(args[0], "")
	if !pi.valid {
		return errUnknownID
	}
	ban, err1 := GetBan(pi.sid)
	if err1 != nil && err1 != errNoResult {
		return errCommandFailed
	}
	bannedNets, err2 := getBanNet(net.ParseIP(args[0]))
	if err2 != nil && err2 != errNoResult {
		return errCommandFailed
	}
	if err1 == errNoResult && err2 == errNoResult {
		return sendMsg(s, m.ChannelID, "No ban for user in db")
	}
	sid := ""
	reason := ""
	var remaining time.Duration
	// TODO Show the longest remaining ban.
	if ban.BanID > 0 {
		sid = pi.sid.String()
		reason = ban.ReasonText
		remaining = ban.ValidUntil.Sub(config.Now())
	}
	ip := "N/A"
	if len(bannedNets) > 0 {
		ip = bannedNets[0].CIDR.String()
		reason = fmt.Sprintf("Banned from %d networks", len(bannedNets))
		remaining = bannedNets[0].ValidUntil.Sub(config.Now())
	}
	r := strings.Split(remaining.String(), ".")
	return sendMsg(s, m.ChannelID, f, sid,
		ban.BanType == model.Banned, ban.BanType == model.NoComm, ip, r[0], reason)
}

func onUnban(s *discordgo.Session, m *discordgo.MessageCreate, args ...string) error {
	sid, err := steamid.SID64FromString(args[0])
	if err != nil || !sid.Valid() {
		return errInvalidSID
	}
	ban, err := GetBan(sid)
	if err != nil {
		if err == errNoResult {
			return errors.New("SteamID does not exist in database")
		} else {
			return errCommandFailed
		}
	}
	if err := DropBan(ban); err != nil {
		return errCommandFailed
	}
	return sendMsg(s, m.ChannelID, "User ban is now inactive")
}

func onKick(s *discordgo.Session, m *discordgo.MessageCreate, args ...string) error {
	pi := findPlayer(args[0], "")
	if !pi.valid || !pi.inGame {
		return errUnknownID
	}
	reason := ""
	if len(args) > 1 {
		reason = strings.Join(args[1:], " ")
	}
	if _, err := execServerRCON(*pi.server, fmt.Sprintf("sm_kick #%d %s", pi.player.UserID, reason)); err != nil {
		return err
	}
	return sendMsg(s, m.ChannelID, "[%s] User kicked: %s", pi.server.ServerName, pi.player.Name)
}

func onSay(s *discordgo.Session, m *discordgo.MessageCreate, args ...string) error {
	server, err := GetServerByName(args[0])
	if err != nil {
		return errors.Errorf("Failed to fetch server: %s", args[0])
	}
	msg := fmt.Sprintf(`sm_say %s`, strings.Join(args[1:], " "))
	resp, err2 := execServerRCON(server, msg)
	if err2 != nil {
		return err2
	}
	rp := strings.Split(resp, "\n")
	if len(rp) < 2 {
		return errors.Errorf("Invalid response")
	}
	return sendMsg(s, m.ChannelID, fmt.Sprintf("`%s`", rp[0]))
}

func onCSay(s *discordgo.Session, m *discordgo.MessageCreate, args ...string) error {
	server, err := GetServerByName(args[0])
	if err != nil {
		return errors.Errorf("Failed to fetch server: %s", args[0])
	}
	msg := fmt.Sprintf(`sm_csay %s`, strings.Join(args[1:], " "))
	resp, err := execServerRCON(server, msg)
	if err != nil {
		return err
	}
	rp := strings.Split(resp, "triggered ")
	if len(rp) < 2 {
		return errors.Errorf("Invalid response")
	}
	return sendMsg(s, m.ChannelID, fmt.Sprintf("`%s`", rp[1]))
}

func onPSay(s *discordgo.Session, m *discordgo.MessageCreate, args ...string) error {
	server, err := GetServerByName(args[0])
	if err != nil {
		return errors.Errorf("Failed to fetch server: %s", args[0])
	}
	msg := fmt.Sprintf(`sm_psay %s "%s"`, args[1], strings.Join(args[2:], " "))
	resp, err := execServerRCON(server, msg)
	if err != nil {
		return errors.Errorf("Failed to exec psay command: %v", err)
	}
	rp := strings.Split(resp, "\n")
	return sendMsg(s, m.ChannelID, fmt.Sprintf("`%s`", rp[0]))
}

func onServers(s *discordgo.Session, m *discordgo.MessageCreate, args ...string) error {
	servers, err := GetServers()
	if err != nil {
		return errors.New("Failed to fetch servers")
	}
	mu := &sync.RWMutex{}
	results := make(map[string]extra.Status)
	var failed []string
	wg := &sync.WaitGroup{}
	for _, s := range servers {
		wg.Add(1)
		go func(server model.Server) {
			defer wg.Done()
			status, err := getServerStatus(server)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				failed = append(failed, server.ServerName)
				return
			}
			results[server.ServerName] = status
		}(s)
	}
	wg.Wait()
	var msg strings.Builder
	msg.WriteString("```ini\n")
	maxLenSn := 0
	maxLenMap := 0
	maxLenName := 0
	for name, r := range results {
		if len(r.ServerName) > maxLenSn {
			maxLenSn = len(r.ServerName)
		}
		if len(r.Map) > maxLenMap {
			maxLenMap = len(r.Map)
		}
		if len(name) > maxLenName {
			maxLenName = len(name)
		}
	}
	for name, r := range results {
		snPad := fmt.Sprintf("%s%s", r.ServerName, strings.Repeat(" ", maxLenSn-len(r.ServerName)))
		snMap := fmt.Sprintf("%s%s", r.Map, strings.Repeat(" ", maxLenMap-len(r.Map)))
		snName := fmt.Sprintf("%s%s", name, strings.Repeat(" ", maxLenName-len(name)))
		msg.WriteString(fmt.Sprintf("%s -- %s -- %s -- %d/%d\n", snName, snPad, snMap, r.PlayersCount, r.PlayersMax))
	}
	if len(failed) > 0 {
		msg.WriteString(fmt.Sprintf("Servers Down: %s\n", strings.Join(failed, ", ")))
	}
	msg.WriteString("```")
	return sendMsg(s, m.ChannelID, msg.String())
}

func getServerStatus(server model.Server) (extra.Status, error) {
	resp, err := execServerRCON(server, "status")
	if err != nil {
		log.Errorf("Failed to exec rcon command: %v", err)
		return extra.Status{}, err
	}
	status, err := extra.ParseStatus(resp, true)
	if err != nil {
		log.Errorf("Failed to parse status output: %v", err)
		return extra.Status{}, err
	}
	return status, nil
}

func getAllServerStatus() (map[model.Server]extra.Status, error) {
	servers, err := GetServers()
	if err != nil {
		return nil, err
	}
	statuses := make(map[model.Server]extra.Status)
	mu := &sync.RWMutex{}
	wg := &sync.WaitGroup{}
	for _, s := range servers {
		wg.Add(1)
		go func(server model.Server) {
			defer wg.Done()
			status, err := getServerStatus(server)
			if err != nil {
				log.Errorf("Failed to parse status output: %v", err)
				return
			}
			mu.Lock()
			statuses[server] = status
			mu.Unlock()
		}(s)
	}
	wg.Wait()
	return statuses, nil
}

func findPlayerByName(name string) (*extra.Player, *model.Server, error) {
	name = strings.ToLower(name)
	statuses, err := getAllServerStatus()
	if err != nil {
		return nil, nil, err
	}
	for server, status := range statuses {
		for _, player := range status.Players {
			if strings.Contains(strings.ToLower(player.Name), name) {
				return &player, &server, nil
			}
		}
	}
	return nil, nil, errUnknownID
}

func findPlayerBySID(sid steamid.SID64) (*extra.Player, *model.Server, error) {
	statuses, err := getAllServerStatus()
	if err != nil {
		return nil, nil, err
	}
	for server, status := range statuses {
		for _, player := range status.Players {
			if player.SID == sid {
				return &player, &server, nil
			}
		}
	}
	return nil, nil, errUnknownID
}

func findPlayerByIP(ip net.IP) (*extra.Player, *model.Server, error) {
	statuses, err := getAllServerStatus()
	if err != nil {
		return nil, nil, err
	}
	for server, status := range statuses {
		for _, player := range status.Players {
			if ip.Equal(player.IP) {
				return &player, &server, nil
			}
		}
	}
	return nil, nil, errUnknownID
}

func findPlayerByCIDR(ipNet *net.IPNet) (*extra.Player, *model.Server, error) {
	statuses, err := getAllServerStatus()
	if err != nil {
		return nil, nil, err
	}
	for server, status := range statuses {
		for _, player := range status.Players {
			if ipNet.Contains(player.IP) {
				return &player, &server, nil
			}
		}
	}
	return nil, nil, errUnknownID
}

func defaultTable(title string) table.Writer {
	t := table.NewWriter()
	t.SuppressEmptyColumns()
	if title != "" {
		t.SetTitle(title)
	}
	t.SetStyle(table.StyleLight)
	return t
}

func onPlayers(s *discordgo.Session, m *discordgo.MessageCreate, args ...string) error {
	server, err := GetServerByName(args[0])
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("Invalid server name")
		}
		return DBErr(err)
	}
	status, err := getServerStatus(server)
	if err != nil {
		log.Errorf("Failed to parse status output: %v", err)
		return model.ErrRCON
	}
	t := defaultTable("")
	t.AppendHeader(table.Row{
		"IP", "steam64", "Name",
	})
	t.AppendSeparator()
	for _, p := range status.Players {
		t.AppendRow(table.Row{
			p.IP, p.SID.String(), p.Name,
		})
	}
	t.SortBy([]table.SortBy{{Name: "name", Number: 2, Mode: table.Asc}})
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("```\n%s\n```", t.Render()))
	return sendMsg(s, m.ChannelID, msg.String())
}

func onHelp(s *discordgo.Session, m *discordgo.MessageCreate, args ...string) error {
	var msg string
	if len(args) == 0 {
		var m []string
		for k := range cmdMap {
			m = append(m, fmt.Sprintf("`%s%s`", config.Discord.Prefix, k))
		}
		sort.Strings(m)
		msg = fmt.Sprintf("Available commands (`!help <command>`): %s", strings.Join(m, ", "))
	} else {
		cmd, found := cmdMap[args[0]]
		if !found {
			return errUnknownCommand
		}
		msg = cmd.help
	}
	return sendMsg(s, m.ChannelID, msg)
}
