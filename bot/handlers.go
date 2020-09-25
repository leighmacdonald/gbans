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
	"github.com/leighmacdonald/steamid/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type PlayerInfo struct {
	player model.Player
	server model.Server
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
func findPlayer(playerStr string) PlayerInfo {
	var (
		player   model.Player
		server   model.Server
		err      error
		inGame   = false
		foundSid steamid.SID64
	)
	sid := steamid.SID64FromString(playerStr)
	if sid.Valid() {
		foundSid = sid
		player, server, err = findPlayerBySID(sid)
		if err != nil {
			return PlayerInfo{player, server, sid, false, true}
		}
		inGame = true
	} else {
		player, server, err = findPlayerByName(playerStr)
		if err != nil {
			return PlayerInfo{player, server, 0, inGame, false}
		}
		foundSid = player.SID
		inGame = true
	}
	return PlayerInfo{player, server, foundSid, inGame, true}
}

func onFind(s *discordgo.Session, m *discordgo.MessageCreate, args ...string) error {
	const f = "Found player `%s` (%d) @ %s"
	pi := findPlayer(args[0])
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
	pi := findPlayer(args[0])
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
			IP:       "",
			Note:     "",
		}
	}
	if ban.BanType == model.Banned {
		return errors.New("Player is already banned")
	}
	ban.BanType = model.NoComm
	ban.ReasonText = reasonStr
	ban.Until = time.Now().Add(duration).Unix()
	if err := store.SaveBan(&ban); err != nil {
		log.Errorf("Failed to save ban: %v", err)
		return errors.New("Failed to save mute state")
	}
	if pi.inGame {
		resp, err := execServerRCON(pi.server, fmt.Sprintf(`sm_gag "#%s"`, steamid.SID64ToSID3(pi.sid)))
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
				return sendMsg(s, m.ChannelID, fmt.Sprintf("Player gagged successfully for: %s", dStr))
			} else {
				return sendMsg(s, m.ChannelID, "Failed to gag player in-game")
			}
		}
	}
	return nil
}

func onBanIP(s *discordgo.Session, m *discordgo.MessageCreate, args ...string) error {
	return nil
}

func onBan(s *discordgo.Session, m *discordgo.MessageCreate, args ...string) error {

	return nil
}

func onCheck(s *discordgo.Session, m *discordgo.MessageCreate, args ...string) error {
	const f = "[%d] Banned: `%v` -- IsMuted: `%v` -- IP: `%s` -- Updated: `%s`"
	pi := findPlayer(args[0])
	if !pi.valid {
		return errUnknownID
	}
	ban, err := store.GetBan(pi.sid)
	if err != nil {
		if err == store.ErrNoResult {
			return sendMsg(s, m.ChannelID, "[%d] No record found", pi.sid.Int64())
		} else {
			return errCommandFailed
		}
	}
	ip := ban.IP
	if ip == "" {
		ip = "Unknown"
	}
	return sendMsg(s, m.ChannelID, f,
		pi.sid.Int64(),
		ban.BanType == model.Banned,
		ban.BanType == model.NoComm,
		ip,
		time.Unix(ban.UpdatedOn, 0).String(),
	)
}

func onUnban(s *discordgo.Session, m *discordgo.MessageCreate, args ...string) error {
	sid := steamid.SID64FromString(args[0])
	if !sid.Valid() {
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
	if !ban.Active {
		return errors.New("Ban is already inactive")
	}
	ban.Active = false
	if err := store.SaveBan(&ban); err != nil {
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
	results := make(map[string]model.Status)
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

func getServerStatus(server model.Server) (model.Status, error) {
	resp, err := execServerRCON(server, "status")
	if err != nil {
		log.Errorf("Failed to exec rcon command: %v", err)
		return model.Status{}, err
	}
	status, err := ParseStatus(resp, true)
	if err != nil {
		log.Errorf("Failed to parse status output: %v", err)
		return model.Status{}, err
	}
	return status, nil
}

func getAllServerStatus() (map[model.Server]model.Status, error) {
	servers, err := store.GetServers()
	if err != nil {
		return nil, err
	}
	statuses := make(map[model.Server]model.Status)
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

func findPlayerByName(name string) (model.Player, model.Server, error) {
	name = strings.ToLower(name)
	statuses, err := getAllServerStatus()
	if err != nil {
		return model.Player{}, model.Server{}, err
	}
	for server, status := range statuses {
		for _, player := range status.Players {
			if strings.Contains(strings.ToLower(player.Name), name) {
				return player, server, nil
			}
		}
	}
	return model.Player{}, model.Server{}, errUnknownID
}

func findPlayerBySID(sid steamid.SID64) (model.Player, model.Server, error) {
	statuses, err := getAllServerStatus()
	if err != nil {
		return model.Player{}, model.Server{}, err
	}
	for server, status := range statuses {
		for _, player := range status.Players {
			if player.SID == sid {
				return player, server, nil
			}
		}
	}
	return model.Player{}, model.Server{}, errUnknownID
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

// ParseStatus will parse a status command output into a struct
// If full is true, it will also parse the address/port of the player.
// This only works for status commands via RCON/CLI
func ParseStatus(status string, full bool) (model.Status, error) {
	var s model.Status
	for _, line := range strings.Split(status, "\n") {
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			switch strings.TrimRight(parts[0], " ") {
			case "hostname":
				s.ServerName = parts[1]
			case "version":
				s.Version = parts[1]
			case "map":
				s.Map = strings.Split(parts[1], " ")[0]
			case "tags":
				s.Tags = strings.Split(parts[1], ",")
			case "players":
				ps := strings.Split(strings.ReplaceAll(parts[1], "(", ""), " ")
				m, err := strconv.ParseUint(ps[4], 10, 64)
				if err != nil {
					return model.Status{}, err
				}
				s.PlayersMax = int(m)
			case "edicts":
				ed := strings.Split(parts[1], " ")
				l, err := strconv.ParseUint(ed[0], 10, 64)
				if err != nil {
					return model.Status{}, err
				}
				m, err := strconv.ParseUint(ed[3], 10, 64)
				if err != nil {
					return model.Status{}, err
				}
				s.Edicts = []int{int(l), int(m)}
			}
			continue
		} else {
			var m []string
			if full {
				m = reStatusPlayerFull.FindStringSubmatch(line)
			} else {
				m = reStatusPlayer.FindStringSubmatch(line)
			}
			if (!full && len(m) == 8) || (full && len(m) == 10) {
				userID, err := strconv.ParseUint(m[1], 10, 64)
				if err != nil {
					return model.Status{}, err
				}
				ping, err := strconv.ParseUint(m[5], 10, 64)
				if err != nil {
					return model.Status{}, err
				}
				loss, err := strconv.ParseUint(m[6], 10, 64)
				if err != nil {
					return model.Status{}, err
				}
				tp := strings.Split(m[4], ":")
				for i, j := 0, len(tp)-1; i < j; i, j = i+1, j-1 {
					tp[i], tp[j] = tp[j], tp[i]
				}
				var totalSec int
				for i, vStr := range tp {
					v, err := strconv.ParseUint(vStr, 10, 64)
					if err != nil {
						return model.Status{}, err
					}
					totalSec += int(v) * []int{1, 60, 3600}[i]
				}
				dur, err := time.ParseDuration(fmt.Sprintf("%ds", totalSec))
				if err != nil {
					return model.Status{}, err
				}
				p := model.Player{
					UserID:        int(userID),
					Name:          m[2],
					SID:           steamid.SID3ToSID64(steamid.SID3(m[3])),
					ConnectedTime: dur,
					Ping:          int(ping),
					Loss:          int(loss),
					State:         m[7],
				}
				if full {
					port, err := strconv.ParseUint(m[9], 10, 64)
					if err != nil {
						return model.Status{}, err
					}
					p.IP = net.ParseIP(m[8])
					p.Port = int(port)
				}
				s.Players = append(s.Players, p)
			}
		}
	}
	s.PlayersCount = len(s.Players)
	return s, nil
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
