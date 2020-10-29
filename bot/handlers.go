package bot

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/config"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/gbans/store"
	"github.com/leighmacdonald/gbans/util"
	"github.com/leighmacdonald/rcon/rcon"
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
		duration, err = util.ParseDuration(args[1])
		if err != nil {
			return err
		}
	}
	reasonStr := model.ReasonString(model.Custom)
	if len(args) > 2 {
		reasonStr = strings.Join(args[2:], " ")
	}
	ban, err := store.GetBan(pi.sid)
	if err != nil && store.DBErr(err) != store.ErrNoResult {
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
	ban.Until = time.Now().Add(duration).Unix()
	if err := store.SaveBan(&ban); err != nil {
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
	duration, err := util.ParseDuration(args[1])
	if err != nil {
		return errInvalidDuration
	}
	_, err = store.GetBanNet(args[0])
	if err != nil && store.DBErr(err) != store.ErrNoResult {
		return errCommandFailed
	}
	if err == nil {
		return errDuplicateBan
	}
	ban, err := model.NewBanNet(args[0], reason, duration, model.Bot)
	if err != nil {
		return errCommandFailed
	}
	if err := store.SaveBanNet(&ban); err != nil {
		return errCommandFailed
	}
	pi, srv, err := findPlayerByCIDR(args[0])
	if err == nil {
		if resp, err := execServerRCON(*srv, fmt.Sprintf("sm_kick %s", pi.Name)); err != nil {
			log.Debug(resp)
		}
	}
	return sendMsg(s, m.ChannelID, "IP ban created successfully")
}

func isIP4(ip net.IP) bool {
	if ip.To4() != nil {
		return true
	}
	return false
}

// onBan !ban <id> <duration> [reason]
func onBan(s *discordgo.Session, m *discordgo.MessageCreate, args ...string) error {
	var reason string
	if len(args) > 2 {
		reason = strings.Join(args[2:], " ")
	}
	duration, err := util.ParseDuration(args[1])
	if err != nil {
		return errInvalidDuration
	}
	pi := findPlayer(args[0], "")
	if !pi.valid {
		return errUnknownID
	}
	exists := false
	ban, err := store.GetBan(pi.sid)
	if err != nil && store.DBErr(err) != store.ErrNoResult {
		return errCommandFailed
	}
	if ban.BanID > 0 {
		exists = true
	}
	if ban.BanType == model.Banned {
		return errDuplicateBan
	}
	ban.SteamID = pi.sid
	ban.BanType = model.Banned
	if duration > 0 {
		ban.Until = time.Now().Add(duration).Unix()
	} else {
		ban.Until = 0
	}
	ban.ReasonText = reason
	ban.Source = model.Bot
	if err := store.SaveBan(&ban); err != nil {
		return errCommandFailed
	}
	pi2, srv, err := findPlayerBySID(pi.sid)
	if err == nil {
		if resp, err := execServerRCON(*srv, fmt.Sprintf("sm_kick %s", pi2.Name)); err != nil {
			log.Debug(resp)
		}
	}
	if exists {
		return sendMsg(s, m.ChannelID, "Ban updated successfully")
	} else {
		return sendMsg(s, m.ChannelID, "Ban created successfully")
	}
}

//goland:noinspection ALL
func onCheck(s *discordgo.Session, m *discordgo.MessageCreate, args ...string) error {
	const f = "[%s] Banned: `%v` -- Muted: `%v` -- IP: `%s` -- Expires In: `%s` Reason: `%s`"
	pi := findPlayer(args[0], "")
	if !pi.valid {
		return errUnknownID
	}
	ban, err1 := store.GetBan(pi.sid)
	if err1 != nil && err1 != store.ErrNoResult {
		return errCommandFailed
	}
	banIp, err2 := store.GetBanNet(args[0])
	if err2 != nil && err2 != store.ErrNoResult {
		return errCommandFailed
	}
	if err1 == store.ErrNoResult && err2 == store.ErrNoResult {
		return sendMsg(s, m.ChannelID, "No ban for user in db")
	}
	sid := ""
	var until time.Time
	reason := ""
	var remaining time.Duration
	if ban.BanID > 0 {
		sid = pi.sid.String()
		until = time.Unix(ban.Until, 0)
		reason = ban.ReasonText
		remaining = until.Sub(time.Now())
	}
	ip := "N/A"
	if banIp.NetID > 0 {
		ip = banIp.CIDR
		until = time.Unix(banIp.Until, 0)
		reason = banIp.Reason
		remaining = until.Sub(time.Now())
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
	ban, err := store.GetBan(sid)
	if err != nil {
		if err == store.ErrNoResult {
			return errors.New("SteamID does not exist in database")
		} else {
			return errCommandFailed
		}
	}
	if err := store.DropBan(ban); err != nil {
		return errCommandFailed
	}
	return sendMsg(s, m.ChannelID, "User ban is now inactive")
}

func onKick(s *discordgo.Session, m *discordgo.MessageCreate, args ...string) error {
	return nil
}

func onSay(s *discordgo.Session, m *discordgo.MessageCreate, args ...string) error {
	server, err := store.GetServerByName(args[0])
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
	server, err := store.GetServerByName(args[0])
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
	server, err := store.GetServerByName(args[0])
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
	servers, err := store.GetServers()
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
	servers, err := store.GetServers()
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

func findPlayerByCIDR(ipNet string) (*extra.Player, *model.Server, error) {
	_, n, err := net.ParseCIDR(ipNet)
	if err != nil {
		return nil, nil, err
	}
	statuses, err := getAllServerStatus()
	if err != nil {
		return nil, nil, err
	}
	for server, status := range statuses {
		for _, player := range status.Players {
			if n.Contains(player.IP) {
				return &player, &server, nil
			}
		}
	}
	return nil, nil, errUnknownID
}

func onPlayers(s *discordgo.Session, m *discordgo.MessageCreate, args ...string) error {
	server, err := store.GetServerByName(args[0])
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("Invalid server name")
		}
		return store.DBErr(err)
	}
	status, err := getServerStatus(server)
	if err != nil {
		log.Errorf("Failed to parse status output: %v", err)
		return model.ErrRCON
	}
	var msg strings.Builder
	msg.WriteString("```ini\n")
	for _, p := range status.Players {
		ipStr := p.IP.String()
		ipStrPad := fmt.Sprintf("%s%s", ipStr, strings.Repeat(" ", 15-len(ipStr)))
		msg.WriteString(fmt.Sprintf("%d %d %s [%s]\n", p.UserID, p.SID.Int64(), ipStrPad, p.Name))
	}
	msg.WriteString("\n```")
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

func execServerRCON(server model.Server, cmd string) (string, error) {
	r, err := rcon.Dial(context.Background(), server.Addr(), server.RCON, time.Second*10)
	if err != nil {
		return "", errors.Errorf("Failed to dial server: %s", server.ServerName)
	}
	resp, err2 := r.Exec(cmd)
	if err2 != nil {
		return "", errors.Errorf("Failed to exec command: %v", err)
	}
	return resp, nil
}
