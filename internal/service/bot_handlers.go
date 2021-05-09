package service

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/external"
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
		sidFS, errFS := steamid.ResolveSID64(context.Background(), playerStr)
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

func onFind(_ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	const f = "Found player `%s` (%d) @ %s"
	userIdentifier := m.Data.Options[0].Value.(string)
	pi := findPlayer(userIdentifier, "")
	if !pi.valid || !pi.inGame {
		return "", errUnknownID
	}
	return fmt.Sprintf(f, pi.player.Name, pi.sid.Int64(), pi.server.ServerName), nil
}

func onMute(_ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	var (
		err      error
		duration = time.Duration(0)
	)
	playerID := m.Data.Options[0].Value.(string)
	pi := findPlayer(playerID, "")
	if !pi.valid {
		return "", errUnknownID
	}
	duration, err = config.ParseDuration(m.Data.Options[1].Value.(string))
	if err != nil {
		return "", err
	}
	reasonStr := model.Custom.String()
	if len(m.Data.Options) > 2 {
		reasonStr = m.Data.Options[2].Value.(string)
	}
	ban, err2 := getBanBySteamID(pi.sid, false)
	if err2 != nil && dbErr(err2) != errNoResult {
		log.Errorf("Error getting ban from db: %v", err2)
		return "", errors.New("Internal DB Error")
	} else if err2 != nil {
		ban = &model.BannedPerson{
			Ban:    model.NewBan(pi.sid, 0, duration),
			Person: model.NewPerson(pi.sid),
		}
	}
	if ban.Ban.BanType == model.Banned {
		return "", errors.New("Person is already banned")
	}
	ban.Ban.BanType = model.NoComm
	ban.Ban.ReasonText = reasonStr
	ban.Ban.ValidUntil = config.Now().Add(duration)
	if err3 := saveBan(ban.Ban); err3 != nil {
		log.Errorf("Failed to save ban: %v", err3)
		return "", errors.New("Failed to save mute state")
	}
	if pi.inGame {
		resp, err4 := execServerRCON(*pi.server, fmt.Sprintf(`sm_gag "#%s"`, steamid.SID64ToSID3(pi.sid)))
		if err4 != nil {
			log.Errorf("Failed to gag active user: %v", err4)
		} else {
			if strings.Contains(resp, "[SM] Gagged") {
				var dStr string
				if duration.Seconds() == 0 {
					dStr = "Forever"
				} else {
					dStr = duration.String()
				}
				return "", errors.Errorf("Person gagged successfully for: %s", dStr)
			} else {
				return "", errors.New("Failed to gag player in-game")
			}
		}
	}
	return "Player muted successfully", nil
}

func onBanIP(_ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	cidrStr := m.Data.Options[0].Value.(string)
	duration, err := config.ParseDuration(m.Data.Options[1].Value.(string))
	if err != nil {
		return "", err
	}
	reason := model.Custom.String()
	if len(m.Data.Options) > 2 {
		reason = m.Data.Options[2].Value.(string)
	}
	_, err2 := getBanNet(net.ParseIP(cidrStr))
	if err2 != nil && dbErr(err2) != errNoResult {
		return "", errCommandFailed
	}
	if err2 == nil {
		return "", errDuplicateBan
	}
	ban, err3 := model.NewBanNet(cidrStr, reason, duration, model.Bot)
	if err3 != nil {
		return "", errCommandFailed
	}
	if err4 := saveBanNet(&ban); err4 != nil {
		return "", errCommandFailed
	}
	_, n, err5 := net.ParseCIDR(cidrStr)
	if err5 != nil {
		return "", errCommandFailed
	}
	pi, srv, err6 := findPlayerByCIDR(n)
	if err6 == nil {
		if resp, err7 := execServerRCON(*srv, fmt.Sprintf("sm_kick %s", pi.Name)); err7 != nil {
			log.Debug(resp)
		}
	}
	return "IP ban created successfully", nil
}

// onBan !ban <id> <duration> [reason]
func onBan(_ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	uid := m.Data.Options[0].Value.(string)
	duration, err := config.ParseDuration(m.Data.Options[1].Value.(string))
	if err != nil {
		return "", errInvalidDuration
	}
	reason := ""
	if len(m.Data.Options) > 2 {
		reason = m.Data.Options[2].Value.(string)
	}
	reporter, errR := getPersonByDiscordID(m.Interaction.Member.User.ID)
	if errR != nil {
		return "", errUnlinkedAccount
	}
	pi := findPlayer(uid, "")
	if !pi.valid {
		return "", errUnknownID
	}
	ban, err2 := BanPlayer(gCtx, pi.sid, reporter.SteamID, duration, model.Custom, reason, model.Bot)
	if err2 != nil {
		return "", err2
	}
	return fmt.Sprintf("Ban created successfully (#%d)", ban.BanID), nil
}

func onCheck(s *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	pId := m.Data.Options[0].Value.(string)
	sid, err := steamid.ResolveSID64(context.Background(), pId)
	if err != nil {
		return "", errUnknownID
	}
	if !sid.Valid() {
		return "", errUnknownID
	}
	bannedPlayer, err2 := getOrCreateProfileBySteamID(sid, "")
	if err2 != nil {
		return "", errCommandFailed
	}
	ban, err3 := getBanBySteamID(sid, false)
	if err3 != nil && err3 != errNoResult {
		return "", errCommandFailed
	}
	bannedNets, err4 := getBanNet(net.ParseIP(pId))
	if err4 != nil && err4 != errNoResult {
		return "", errCommandFailed
	}
	if err3 == errNoResult && err4 == errNoResult {
		return "", sendMsg(s, m.Interaction, "No ban for user in db")
	}
	banned := false
	muted := false
	reason := ""
	var expiry time.Time
	// TODO Show the longest remaining ban.
	if ban != nil && ban.Ban.BanID > 0 {
		banned = ban.Ban.BanType == model.Banned
		muted = ban.Ban.BanType == model.NoComm
		reason = ban.Ban.ReasonText
		expiry = ban.Ban.ValidUntil
	}
	logData, errLogs := external.LogsTFOverview(sid)
	if errLogs != nil {
		log.Warnf("Failed to fetch logTF data: %v", errLogs)
	}
	//ip := "N/A"
	if len(bannedNets) > 0 {
		//ip = bannedNets[0].CIDR.String()
		reason = fmt.Sprintf("Banned from %d networks", len(bannedNets))
		expiry = bannedNets[0].ValidUntil
	}
	asn, _ := getASNRecord(bannedPlayer.IPAddr)
	location, _ := getLocationRecord(bannedPlayer.IPAddr)
	proxy, _ := getProxyRecord(bannedPlayer.IPAddr)
	t := defaultTable(fmt.Sprintf("Profile of: %s", bannedPlayer.PersonaName))
	t.AppendSeparator()
	t.SuppressEmptyColumns()
	t.AppendRow(table.Row{
		"Real Name", bannedPlayer.RealName,
		"Profile", strings.Replace(bannedPlayer.ProfileURL, "https://", "", 1)})
	cd := time.Unix(int64(bannedPlayer.TimeCreated), 0)
	t.AppendRow(table.Row{
		"Account Age", time.Since(cd).String(),
		"Private", bannedPlayer.CommunityVisibilityState == 1,
	})
	t.AppendRow(table.Row{
		"STEAM64", bannedPlayer.SteamID.String(),
		"STEAM", steamid.SID64ToSID(bannedPlayer.SteamID),
	})
	t.AppendRow(table.Row{
		"STEAM3", steamid.SID64ToSID3(bannedPlayer.SteamID),
		"STEAM32", steamid.SID64ToSID32(bannedPlayer.SteamID),
	})
	t.AppendRow(table.Row{"Banned", banned, "Muted", muted})
	if ban != nil && ban.Ban.BanID > 0 {
		author, e := getPersonBySteamID(ban.Ban.AuthorID)
		r := table.Row{"Reason", reason}
		if e == nil && author != nil && author.DiscordID != "" {
			r = append(r, "Author", fmt.Sprintf("<@%s>", author.DiscordID))
		}
		t.AppendRow(r)
		t.AppendRow(table.Row{
			"Ban Created", config.FmtTimeShort(ban.Ban.CreatedOn),
			"Ban Expires", config.FmtDuration(expiry)})
	}
	if bannedPlayer.IPAddr != nil {
		t.AppendRow(table.Row{
			"Last IP", bannedPlayer.IPAddr,
		})
	}
	if asn != nil {
		t.AppendRow(table.Row{
			"ASN", fmt.Sprintf("(%d) %s", asn.ASNum, asn.ASName),
		})
	}
	if location != nil {
		t.AppendRow(table.Row{
			"City", location.CityName,
			"Country", location.CountryName,
		})
	}
	if proxy != nil {
		t.AppendRow(table.Row{
			"Proxy Type", proxy.ProxyType,
			"Proxy", proxy.Threat,
		})
	}
	if errLogs == nil && logData.Success {
		t.AppendRow(table.Row{
			"Logs.tf Count", logData.Total,
		})
	}
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("```\n%s\n```", t.Render()))
	return msg.String(), nil
}

func onHistory(s *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	switch m.Data.Options[0].Name {
	case string(cmdHistoryIP):
		return onHistoryIP(s, m)
	default:
		return onHistoryChat(s, m)
	}
}

func onHistoryIP(s *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	pId := m.Data.Options[0].Options[0].Value.(string)
	sid, err := steamid.ResolveSID64(context.Background(), pId)
	if err != nil || !sid.Valid() {
		return "", errInvalidSID
	}
	p, err2 := GetOrCreatePersonBySteamID(sid)
	if err2 != nil || !sid.Valid() {
		return "", errCommandFailed
	}
	records, err := getPersonIPHistory(sid)
	if err != nil {
		return "", errCommandFailed
	}
	t := defaultTable(fmt.Sprintf("IP History of: %s", p.PersonaName))
	t.AppendSeparator()
	t.SuppressEmptyColumns()
	lastIp := net.IP{}
	for _, rec := range records {
		if rec.IP.Equal(lastIp) {
			continue
		}
		t.AppendRow(table.Row{
			rec.IP.String(),
			rec.CreatedOn.Format("Mon Jan 2 15:04:05"),
			fmt.Sprintf("%s, %s", rec.CityName, rec.CountryCode),
			fmt.Sprintf("(%d) %s", rec.ASNum, rec.ASName),
			fmt.Sprintf("%s, %s, %s, %s", rec.ISP, rec.UsageType, rec.Threat, rec.DomainUsed),
		})
		lastIp = rec.IP
	}
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("```\n%s\n```", t.Render()))
	return msg.String(), nil
}

func onHistoryChat(_ *discordgo.Session, _ *discordgo.InteractionCreate) (string, error) {
	return "", errors.New("hi")
}

func onSetSteam(_ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	pId := m.Data.Options[0].Value.(string)
	sid, err := steamid.ResolveSID64(context.Background(), pId)
	if err != nil || !sid.Valid() {
		return "", errInvalidSID
	}
	p, errP := GetOrCreatePersonBySteamID(sid)
	if errP != nil || !sid.Valid() {
		return "", errCommandFailed
	}
	if (p.DiscordID) != "" {
		return "", errors.Errorf("Discord account already linked to steam account: %d", p.SteamID.Int64())
	}
	p.DiscordID = m.Interaction.Member.User.ID
	if err := SavePerson(p); err != nil {
		return "", errCommandFailed
	}
	return "Successfully linked your account", nil
}

func onUnban(_ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	pId := m.Data.Options[0].Value.(string)
	sid, err := steamid.ResolveSID64(context.Background(), pId)
	if err != nil || !sid.Valid() {
		return "", errInvalidSID
	}
	ban, err2 := getBanBySteamID(sid, false)
	if err2 != nil {
		if err2 == errNoResult {
			return "", errUnknownBan
		} else {
			return "", errCommandFailed
		}
	}
	if err3 := dropBan(ban.Ban); err3 != nil {
		return "", errCommandFailed
	}
	return "User ban is now inactive", nil
}

func onKick(_ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	pId := m.Data.Options[0].Value.(string)
	pi := findPlayer(pId, "")
	if !pi.valid || !pi.inGame {
		return "", errUnknownID
	}
	reason := ""
	if len(m.Data.Options) > 1 {
		reason = m.Data.Options[1].Value.(string)
	}
	if _, err := execServerRCON(*pi.server, fmt.Sprintf("sm_kick #%d %s", pi.player.UserID, reason)); err != nil {
		return "", err
	}
	return fmt.Sprintf("[%s] User kicked: %s", pi.server.ServerName, pi.player.Name), nil
}

func onSay(_ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	sId := m.Data.Options[0].Value.(string)
	server, err := getServerByName(sId)
	if err != nil {
		return "", errors.Errorf("Failed to fetch server: %s", sId)
	}
	msg := fmt.Sprintf(`sm_say %s`, m.Data.Options[1].Value.(string))
	resp, err2 := execServerRCON(server, msg)
	if err2 != nil {
		return "", err2
	}
	rp := strings.Split(resp, "\n")
	if len(rp) < 2 {
		return "", errors.Errorf("Invalid response")
	}
	return fmt.Sprintf("`%s`", rp[0]), nil
}

func onCSay(_ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	sId := m.Data.Options[0].Value.(string)
	var (
		servers []model.Server
		err     error
	)
	if sId == "*" {
		servers, err = getServers()
		if err != nil {
			return "", errors.Wrapf(err, "Failed to fetch servers")
		}
	} else {
		server, errS := getServerByName(sId)
		if errS != nil {
			return "", errors.Wrapf(errS, "Failed to fetch server: %s", sId)
		}
		servers = append(servers, server)
	}
	msg := fmt.Sprintf(`sm_csay %s`, m.Data.Options[1].Value.(string))
	_ = queryRCON(context.Background(), servers, msg)
	return fmt.Sprintf("Message sent to %d server(s)", len(servers)), nil
}

func onPSay(_ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	pId := m.Data.Options[0].Value.(string)
	pi := findPlayer(pId, "")
	if !pi.valid || !pi.inGame {
		return "", errUnknownID
	}
	msg := fmt.Sprintf(`sm_psay %d "%s"`, pi.player.UserID, m.Data.Options[1].Value.(string))
	resp, err := execServerRCON(*pi.server, msg)
	if err != nil {
		return "", errors.Errorf("Failed to exec psay command: %v", err)
	}
	rp := strings.Split(resp, "\n")
	return fmt.Sprintf("`%s`", rp[0]), nil
}

func onServers(_ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	full := false
	if len(m.Data.Options) > 0 {
		full = m.Data.Options[0].Value.(bool)
	}
	servers, err := getServers()
	if err != nil {
		return "", errors.New("Failed to fetch servers")
	}
	mu := &sync.RWMutex{}
	results := make(map[string]extra.Status)
	serverMap := make(map[string]model.Server)
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
			serverMap[server.ServerName] = server
		}(s)
	}
	wg.Wait()
	t := defaultTable("")
	if full {
		t.AppendHeader(table.Row{
			"ID", "Name", "Current Map", "Players", "Version", "Tags",
		})
	} else {
		t.AppendHeader(table.Row{
			"ID", "Name", "Current Map", "Players",
		})
	}
	t.AppendSeparator()
	for name, r := range results {
		if full {
			t.AppendRow(table.Row{
				name, r.ServerName, r.Map, fmt.Sprintf("%d/%d", r.PlayersCount,
					r.PlayersMax-serverMap[name].ReservedSlots), r.Version, strings.Join(r.Tags, ", "),
			})
		} else {
			t.AppendRow(table.Row{name, r.ServerName, r.Map, fmt.Sprintf("%d/%d",
				r.PlayersCount, r.PlayersMax-serverMap[name].ReservedSlots)})
		}
	}
	for _, name := range failed {
		if full {
			t.AppendRow(table.Row{name, "OFFLINE", "", "", "", ""})
		} else {
			t.AppendRow(table.Row{name, "OFFLINE", "", ""})
		}
	}
	t.SortBy([]table.SortBy{{Name: "name", Number: 2, Mode: table.Asc}})
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("```\n%s\n```", t.Render()))
	return msg.String(), nil
}

func getServerStatus(server model.Server) (extra.Status, error) {
	resp, err := execServerRCON(server, "status")
	if err != nil {
		log.Errorf("Failed to exec rcon command: %v", err)
		return extra.Status{}, err
	}
	status, err2 := extra.ParseStatus(resp, true)
	if err2 != nil {
		log.Errorf("Failed to parse status output: %v", err2)
		return extra.Status{}, err2
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
			status, err2 := getServerStatus(server)
			if err2 != nil {
				log.Errorf("Failed to parse status output: %v", err2)
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
	t.SetAllowedRowLength(150)
	t.SuppressEmptyColumns()
	if title != "" {
		t.SetTitle(title)
	}
	t.SetStyle(table.StyleRounded)
	return t
}

func onPlayers(s *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	sId := m.Data.Options[0].Value.(string)
	server, err := getServerByName(sId)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("Invalid server name")
		}
		return "", dbErr(err)
	}
	status, err2 := getServerStatus(server)
	if err2 != nil {
		log.Errorf("Failed to parse status output: %v", err2)
		return "", model.ErrRCON
	}
	t := defaultTable("")
	t.AppendHeader(table.Row{"IP", "steam64", "Name"})
	t.AppendSeparator()
	for _, p := range status.Players {
		t.AppendRow(table.Row{p.IP, p.SID.String(), p.Name})
	}
	t.SortBy([]table.SortBy{{Name: "name", Number: 2, Mode: table.Asc}})
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("```\n%s\n```", t.Render()))
	return "", sendMsg(s, m.Interaction, msg.String())
}
