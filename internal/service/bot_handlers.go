package service

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"strings"
	"sync"
	"time"
)

type playerInfo struct {
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
func findPlayer(playerStr string, ip string) playerInfo {
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
		sidFS, errFS := steamid.SID64FromString(playerStr)
		if errFS == nil && sidFS.Valid() {
			foundSid = sidFS
			player, server, err = findPlayerBySID(sidFS)
			if err == nil {
				inGame = true
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

	return playerInfo{player, server, foundSid, inGame, valid}
}

func onFind(s *discordgo.Session, m *discordgo.InteractionCreate) error {
	const f = "Found player `%s` (%d) @ %s"
	userIdentifier := m.Data.Options[0].Value.(string)
	pi := findPlayer(userIdentifier, "")
	if !pi.valid || !pi.inGame {
		return errUnknownID
	}
	return sendMsg(s, m.Interaction, fmt.Sprintf(f, pi.player.Name, pi.sid.Int64(), pi.server.ServerName))
}

func onMute(s *discordgo.Session, m *discordgo.InteractionCreate) error {
	var (
		err      error
		duration = time.Duration(0)
	)
	playerID := m.Data.Options[0].Value.(string)
	pi := findPlayer(playerID, "")
	if !pi.valid {
		return errUnknownID
	}

	duration, err = config.ParseDuration(m.Data.Options[1].Value.(string))
	if err != nil {
		return err
	}
	reasonStr := model.Custom.String()
	if len(m.Data.Options) > 2 {
		reasonStr = m.Data.Options[2].Value.(string)
	}
	ban, err := getBanBySteamID(pi.sid, false)
	if err != nil && dbErr(err) != errNoResult {
		log.Errorf("Error getting ban from db: %v", err)
		return errors.New("Internal DB Error")
	} else if err != nil {
		ban = &model.BannedPerson{
			Ban:    model.NewBan(pi.sid, 0, duration),
			Person: model.NewPerson(pi.sid),
		}
	}
	if ban.Ban.BanType == model.Banned {
		return errors.New("Person is already banned")
	}
	ban.Ban.BanType = model.NoComm
	ban.Ban.ReasonText = reasonStr
	ban.Ban.ValidUntil = config.Now().Add(duration)
	if errB := saveBan(ban.Ban); errB != nil {
		log.Errorf("Failed to save ban: %v", errB)
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
				return sendMsg(s, m.Interaction, fmt.Sprintf("Person gagged successfully for: %s", dStr))
			} else {
				return sendMsg(s, m.Interaction, "Failed to gag player in-game")
			}
		}
	}
	return nil
}

func onBanIP(s *discordgo.Session, m *discordgo.InteractionCreate) error {
	cidrStr := m.Data.Options[0].Value.(string)
	duration, err := config.ParseDuration(m.Data.Options[1].Value.(string))
	if err != nil {
		return err
	}
	reason := model.Custom.String()
	if len(m.Data.Options) > 2 {
		reason = m.Data.Options[2].Value.(string)
	}
	_, err = getBanNet(net.ParseIP(cidrStr))
	if err != nil && dbErr(err) != errNoResult {
		return errCommandFailed
	}
	if err == nil {
		return errDuplicateBan
	}
	ban, err := model.NewBanNet(cidrStr, reason, duration, model.Bot)
	if err != nil {
		return errCommandFailed
	}
	if err := saveBanNet(&ban); err != nil {
		return errCommandFailed
	}
	_, n, err := net.ParseCIDR(cidrStr)
	if err != nil {
		return errCommandFailed
	}
	pi, srv, err := findPlayerByCIDR(n)
	if err == nil {
		if resp, err := execServerRCON(*srv, fmt.Sprintf("sm_kick %s", pi.Name)); err != nil {
			log.Debug(resp)
		}
	}
	return sendMsg(s, m.Interaction, "IP ban created successfully")
}

// onBan !ban <id> <duration> [reason]
func onBan(s *discordgo.Session, m *discordgo.InteractionCreate) error {
	uid := m.Data.Options[0].Value.(string)
	duration, err := config.ParseDuration(m.Data.Options[1].Value.(string))
	if err != nil {
		return errInvalidDuration
	}
	reason := ""
	if len(m.Data.Options) > 2 {
		reason = m.Data.Options[2].Value.(string)
	}
	pi := findPlayer(uid, "")
	if !pi.valid {
		return errUnknownID
	}
	ban, err2 := BanPlayer(gCtx, pi.sid, config.General.Owner, duration, model.Custom, reason, model.Bot)
	if err2 != nil {
		if err2 == errDuplicate {
			return sendMsg(s, m.Interaction, "ID already banned")
		} else {
			return sendMsg(s, m.Interaction, "Error banning: %s", err2)
		}
	}
	return sendMsg(s, m.Interaction, "Ban created successfully (#%d)", ban.BanID)
}

//goland:noinspection ALL
func onCheck(s *discordgo.Session, m *discordgo.InteractionCreate) error {
	const f = "[%s] Banned: `%v` -- Muted: `%v` -- IP: `%s` -- Expires In: `%s` Reason: `%s`"
	pId := m.Data.Options[0].Value.(string)
	pi := findPlayer(pId, "")
	if !pi.valid {
		return errUnknownID
	}
	ban, err1 := getBanBySteamID(pi.sid, false)
	if err1 != nil && err1 != errNoResult {
		return errCommandFailed
	}
	bannedNets, err2 := getBanNet(net.ParseIP(pId))
	if err2 != nil && err2 != errNoResult {
		return errCommandFailed
	}
	if err1 == errNoResult && err2 == errNoResult {
		return sendMsg(s, m.Interaction, "No ban for user in db")
	}
	sid := ""
	reason := ""
	var remaining time.Duration
	// TODO Show the longest remaining ban.
	if ban.Ban.BanID > 0 {
		sid = pi.sid.String()
		reason = ban.Ban.ReasonText
		remaining = ban.Ban.ValidUntil.Sub(config.Now())
	}
	ip := "N/A"
	if len(bannedNets) > 0 {
		ip = bannedNets[0].CIDR.String()
		reason = fmt.Sprintf("Banned from %d networks", len(bannedNets))
		remaining = bannedNets[0].ValidUntil.Sub(config.Now())
	}
	r := strings.Split(remaining.String(), ".")
	return sendMsg(s, m.Interaction, f, sid,
		ban.Ban.BanType == model.Banned, ban.Ban.BanType == model.NoComm, ip, r[0], reason)
}

func onUnban(s *discordgo.Session, m *discordgo.InteractionCreate) error {
	pId := m.Data.Options[0].Value.(string)
	sid, err := steamid.SID64FromString(pId)
	if err != nil || !sid.Valid() {
		return errInvalidSID
	}
	ban, err := getBanBySteamID(sid, false)
	if err != nil {
		if err == errNoResult {
			return errUnknownBan
		} else {
			return errCommandFailed
		}
	}
	if err := dropBan(ban.Ban); err != nil {
		return errCommandFailed
	}
	return sendMsg(s, m.Interaction, "User ban is now inactive")
}

func onKick(s *discordgo.Session, m *discordgo.InteractionCreate) error {
	pId := m.Data.Options[0].Value.(string)
	pi := findPlayer(pId, "")
	if !pi.valid || !pi.inGame {
		return errUnknownID
	}
	reason := ""
	if len(m.Data.Options) > 1 {
		reason = m.Data.Options[1].Value.(string)
	}
	if _, err := execServerRCON(*pi.server, fmt.Sprintf("sm_kick #%d %s", pi.player.UserID, reason)); err != nil {
		return err
	}
	return sendMsg(s, m.Interaction, "[%s] User kicked: %s", pi.server.ServerName, pi.player.Name)
}

func onSay(s *discordgo.Session, m *discordgo.InteractionCreate) error {
	sId := m.Data.Options[0].Value.(string)
	server, err := getServerByName(sId)
	if err != nil {
		return errors.Errorf("Failed to fetch server: %s", sId)
	}
	msg := fmt.Sprintf(`sm_say %s`, m.Data.Options[1].Value.(string))
	resp, err2 := execServerRCON(server, msg)
	if err2 != nil {
		return err2
	}
	rp := strings.Split(resp, "\n")
	if len(rp) < 2 {
		return errors.Errorf("Invalid response")
	}
	return sendMsg(s, m.Interaction, fmt.Sprintf("`%s`", rp[0]))
}

func onCSay(s *discordgo.Session, m *discordgo.InteractionCreate) error {
	sId := m.Data.Options[0].Value.(string)
	var (
		servers []model.Server
		err     error
	)
	if sId == "*" {
		servers, err = getServers()
		if err != nil {
			return errors.Wrapf(err, "Failed to fetch servers")
		}
	} else {
		server, errS := getServerByName(sId)
		if errS != nil {
			return errors.Wrapf(errS, "Failed to fetch server: %s", sId)
		}
		servers = append(servers, server)
	}
	msg := fmt.Sprintf(`sm_csay %s`, m.Data.Options[1].Value.(string))
	_ = queryRCON(context.Background(), servers, msg)
	return sendMsg(s, m.Interaction, fmt.Sprintf("Message sent to %d server(s)", len(servers)))
}

func onPSay(s *discordgo.Session, m *discordgo.InteractionCreate) error {
	pId := m.Data.Options[0].Value.(string)
	pi := findPlayer(pId, "")
	if !pi.valid || !pi.inGame {
		return errUnknownID
	}
	msg := fmt.Sprintf(`sm_psay %d "%s"`, pi.player.UserID, m.Data.Options[1].Value.(string))
	resp, err := execServerRCON(*pi.server, msg)
	if err != nil {
		return errors.Errorf("Failed to exec psay command: %v", err)
	}
	rp := strings.Split(resp, "\n")
	return sendMsg(s, m.Interaction, fmt.Sprintf("`%s`", rp[0]))
}

func onServers(s *discordgo.Session, m *discordgo.InteractionCreate) error {
	servers, err := getServers()
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
	return s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionApplicationCommandResponseData{
			Content: msg.String(),
		},
	})
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
	servers, err := getServers()
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

func onPlayers(s *discordgo.Session, m *discordgo.InteractionCreate) error {
	sId := m.Data.Options[0].Value.(string)
	server, err := getServerByName(sId)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("Invalid server name")
		}
		return dbErr(err)
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
	return sendMsg(s, m.Interaction, msg.String())
}
