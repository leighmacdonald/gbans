package discord

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/external"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/query"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"strings"
	"sync"
	"time"
)

func onFind(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	const f = "Found player `%s` (%d) @ %s"
	userIdentifier := m.Data.Options[0].Value.(string)
	pi := actions.FindPlayer(ctx, userIdentifier, "")
	if !pi.Valid || !pi.InGame {
		return "", consts.ErrUnknownID
	}
	return fmt.Sprintf(f, pi.Player.Name, pi.SteamID.Int64(), pi.Server.ServerName), nil
}

func onMute(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	au, er := store.GetPersonByDiscordID(ctx, m.Member.User.ID)
	if er != nil {
		return "", errUnlinkedAccount
	}
	playerID := m.Data.Options[0].Value.(string)
	sid, err := steamid.ResolveSID64(ctx, playerID)
	if err != nil {
		return "", consts.ErrUnknownID
	}
	duration, err2 := config.ParseDuration(m.Data.Options[1].Value.(string))
	if err2 != nil {
		return "", err2
	}
	reasonStr := model.Custom.String()
	if len(m.Data.Options) > 2 {
		reasonStr = m.Data.Options[2].Value.(string)
	}
	if _, e := actions.MutePlayer(context.Background(), sid, au.SteamID, duration, model.Custom, reasonStr); e != nil {
		return "", e
	}
	return "Player muted successfully", nil
}

func onBanIP(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	cidrStr := m.Data.Options[0].Value.(string)
	duration, err := config.ParseDuration(m.Data.Options[1].Value.(string))
	if err != nil {
		return "", err
	}
	reason := model.Custom.String()
	if len(m.Data.Options) > 2 {
		reason = m.Data.Options[2].Value.(string)
	}
	_, err2 := store.GetBanNet(ctx, net.ParseIP(cidrStr))
	if err2 != nil && err2 != store.ErrNoResult {
		return "", errCommandFailed
	}
	if err2 == nil {
		return "", consts.ErrDuplicateBan
	}
	ban, err3 := model.NewBanNet(cidrStr, reason, duration, model.Bot)
	if err3 != nil {
		return "", errCommandFailed
	}
	if err4 := store.SaveBanNet(ctx, &ban); err4 != nil {
		return "", errCommandFailed
	}
	_, n, err5 := net.ParseCIDR(cidrStr)
	if err5 != nil {
		return "", errCommandFailed
	}
	pi, srv, err6 := actions.FindPlayerByCIDR(ctx, n)
	if err6 == nil {
		if resp, err7 := query.ExecRCON(*srv, fmt.Sprintf("sm_kick %s", pi.Name)); err7 != nil {
			log.Debug(resp)
		}
	}
	return "IP ban created successfully", nil
}

// onBan !ban <id> <duration> [reason]
func onBan(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	uid := m.Data.Options[0].Value.(string)
	duration, err := config.ParseDuration(m.Data.Options[1].Value.(string))
	if err != nil {
		return "", consts.ErrInvalidDuration
	}
	reason := ""
	if len(m.Data.Options) > 2 {
		reason = m.Data.Options[2].Value.(string)
	}
	reporter, errR := store.GetPersonByDiscordID(ctx, m.Interaction.Member.User.ID)
	if errR != nil {
		return "", errUnlinkedAccount
	}
	pi := actions.FindPlayer(ctx, uid, "")
	if !pi.Valid {
		return "", consts.ErrUnknownID
	}
	ban, err2 := actions.BanPlayer(context.Background(), pi.SteamID, reporter.SteamID, duration, model.Custom, reason, model.Bot)
	if err2 != nil {
		return "", err2
	}
	return fmt.Sprintf("Ban created successfully (#%d)", ban.BanID), nil
}

func onCheck(ctx context.Context, s *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	pId := m.Data.Options[0].Value.(string)
	sid, err := steamid.ResolveSID64(ctx, pId)
	if err != nil {
		return "", consts.ErrUnknownID
	}
	if !sid.Valid() {
		return "", consts.ErrUnknownID
	}
	bannedPlayer, err2 := actions.GetOrCreateProfileBySteamID(ctx, sid, "")
	if err2 != nil {
		return "", errCommandFailed
	}
	ban, err3 := store.GetBanBySteamID(ctx, sid, false)
	if err3 != nil && err3 != store.ErrNoResult {
		return "", errCommandFailed
	}
	bannedNets, err4 := store.GetBanNet(ctx, net.ParseIP(pId))
	if err4 != nil && err4 != store.ErrNoResult {
		return "", errCommandFailed
	}
	if err3 == store.ErrNoResult && err4 == store.ErrNoResult {
		return "", sendInteractionMessageEdit(s, m.Interaction, "No ban for user in db")
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
	asn, _ := store.GetASNRecord(ctx, bannedPlayer.IPAddr)
	location, _ := store.GetLocationRecord(ctx, bannedPlayer.IPAddr)
	proxy, _ := store.GetProxyRecord(ctx, bannedPlayer.IPAddr)
	title := fmt.Sprintf("Profile of: %s", bannedPlayer.PersonaName)
	if ban != nil && ban.Ban.BanID > 0 {
		if ban.Ban.BanType == model.Banned {
			title += " (BANNED)"
		} else if ban.Ban.BanType == model.NoComm {
			title += " (MUTED)"
		}
	}
	t := defaultTable(title)
	t.AppendSeparator()
	t.SuppressEmptyColumns()
	t.AppendRow(table.Row{
		"Real Name", bannedPlayer.RealName,
		"Profile", strings.Replace(bannedPlayer.ProfileURL, "https://", "", 1)})
	cd := time.Unix(int64(bannedPlayer.TimeCreated), 0)
	t.AppendRow(table.Row{
		"Account Age", config.FmtDuration(cd),
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
	vacState := "false"
	if bannedPlayer.VACBans > 0 {
		vacState = fmt.Sprintf("true (count: %d) (days: %d)", bannedPlayer.VACBans, bannedPlayer.DaysSinceLastBan)
	}
	gameState := "false"
	if bannedPlayer.GameBans > 0 {
		gameState = fmt.Sprintf("true (count: %d)", bannedPlayer.GameBans)
	}
	t.AppendRow(table.Row{
		"VAC Banned", vacState,
		"Game Banned", gameState,
	})
	ecoBan := "false"
	if bannedPlayer.EconomyBan != "none" {
		ecoBan = bannedPlayer.EconomyBan
	}
	t.AppendRow(table.Row{
		"Community Ban", bannedPlayer.CommunityBanned,
		"Economy Ban", ecoBan,
	})
	t.AppendRow(table.Row{"Banned", banned, "Muted", muted})
	if ban != nil && ban.Ban.BanID > 0 {
		author, e := store.GetPersonBySteamID(ctx, ban.Ban.AuthorID)
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
	return t.Render(), nil
}

func onHistory(ctx context.Context, s *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	switch m.Data.Options[0].Name {
	case string(cmdHistoryIP):
		return onHistoryIP(ctx, s, m)
	default:
		return onHistoryChat(ctx, s, m)
	}
}

func onHistoryIP(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	pId := m.Data.Options[0].Options[0].Value.(string)
	sid, err := steamid.ResolveSID64(ctx, pId)
	if err != nil || !sid.Valid() {
		return "", consts.ErrInvalidSID
	}
	p, err2 := store.GetOrCreatePersonBySteamID(ctx, sid)
	if err2 != nil || !sid.Valid() {
		return "", errCommandFailed
	}
	records, err := store.GetPersonIPHistory(ctx, sid)
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
	return t.Render(), nil
}

func onHistoryChat(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (string, error) {
	return "", errors.New("hi")
}

func onSetSteam(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	pId := m.Data.Options[0].Value.(string)
	sid, err := steamid.ResolveSID64(ctx, pId)
	if err != nil || !sid.Valid() {
		return "", consts.ErrInvalidSID
	}
	p, errP := store.GetOrCreatePersonBySteamID(ctx, sid)
	if errP != nil || !sid.Valid() {
		return "", errCommandFailed
	}
	if (p.DiscordID) != "" {
		return "", errors.Errorf("Discord account already linked to steam account: %d", p.SteamID.Int64())
	}
	p.DiscordID = m.Interaction.Member.User.ID
	if errS := store.SavePerson(ctx, p); errS != nil {
		return "", errCommandFailed
	}
	return "Successfully linked your account", nil
}

func onUnban(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	author, er := store.GetPersonByDiscordID(ctx, m.Member.User.ID)
	if er != nil {
		return "", errUnlinkedAccount
	}
	pId := m.Data.Options[0].Value.(string)
	reason := m.Data.Options[1].Value.(string)
	sid, err := steamid.ResolveSID64(ctx, pId)
	if err != nil || !sid.Valid() {
		return "", consts.ErrInvalidSID
	}
	if errUB := actions.UnbanPlayer(ctx, sid, author.SteamID, reason); errUB != nil {
		return "", errors.Wrapf(err, "Error trying to execute unban")
	}
	return "User ban is now inactive", nil
}

func onKick(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	pId := m.Data.Options[0].Value.(string)
	pi := actions.FindPlayer(ctx, pId, "")
	if !pi.Valid || !pi.InGame {
		return "", consts.ErrUnknownID
	}
	reason := ""
	if len(m.Data.Options) > 1 {
		reason = m.Data.Options[1].Value.(string)
	}
	if _, err := query.ExecRCON(*pi.Server, fmt.Sprintf("sm_kick #%d %s", pi.Player.UserID, reason)); err != nil {
		return "", err
	}
	return fmt.Sprintf("[%s] User kicked: %s", pi.Server.ServerName, pi.Player.Name), nil
}

func onSay(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	sId := m.Data.Options[0].Value.(string)
	server, err := store.GetServerByName(ctx, sId)
	if err != nil {
		return "", errors.Errorf("Failed to fetch server: %s", sId)
	}
	msg := fmt.Sprintf(`sm_say %s`, m.Data.Options[1].Value.(string))
	resp, err2 := query.ExecRCON(server, msg)
	if err2 != nil {
		return "", err2
	}
	rp := strings.Split(resp, "\n")
	if len(rp) < 2 {
		return "", errors.Errorf("Invalid response")
	}
	return fmt.Sprintf("`%s`", rp[0]), nil
}

func onCSay(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	sId := m.Data.Options[0].Value.(string)
	var (
		servers []model.Server
		err     error
	)
	if sId == "*" {
		servers, err = store.GetServers(ctx)
		if err != nil {
			return "", errors.Wrapf(err, "Failed to fetch servers")
		}
	} else {
		server, errS := store.GetServerByName(ctx, sId)
		if errS != nil {
			return "", errors.Wrapf(errS, "Failed to fetch server: %s", sId)
		}
		servers = append(servers, server)
	}
	msg := fmt.Sprintf(`sm_csay %s`, m.Data.Options[1].Value.(string))
	_ = query.RCON(ctx, servers, msg)
	return fmt.Sprintf("Message sent to %d server(s)", len(servers)), nil
}

func onPSay(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	pId := m.Data.Options[0].Value.(string)
	pi := actions.FindPlayer(ctx, pId, "")
	if !pi.Valid || !pi.InGame {
		return "", consts.ErrUnknownID
	}
	msg := fmt.Sprintf(`sm_psay %d "%s"`, pi.Player.UserID, m.Data.Options[1].Value.(string))
	resp, err := query.ExecRCON(*pi.Server, msg)
	if err != nil {
		return "", errors.Errorf("Failed to exec psay command: %v", err)
	}
	rp := strings.Split(resp, "\n")
	return fmt.Sprintf("`%s`", rp[0]), nil
}

func onServers(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	full := false
	if len(m.Data.Options) > 0 {
		full = m.Data.Options[0].Value.(bool)
	}
	servers, err := store.GetServers(ctx)
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
			status, errSS := query.GetServerStatus(server)
			mu.Lock()
			defer mu.Unlock()
			if errSS != nil {
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
	t.SortBy([]table.SortBy{{Name: "ID", Number: 2, Mode: table.Asc}})
	return t.Render(), nil
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

func onPlayers(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	sId := m.Data.Options[0].Value.(string)
	server, err := store.GetServerByName(ctx, sId)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("Invalid server name")
		}
		return "", err
	}
	status, err2 := query.GetServerStatus(server)
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
	return t.Render(), nil
}

func onFilter(ctx context.Context, s *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	switch m.Data.Options[0].Name {
	case string(cmdFilterAdd):
		return onFilterAdd(ctx, s, m)
	case string(cmdFilterDel):
		return onFilterDel(ctx, s, m)
	case string(cmdFilterCheck):
		return onFilterCheck(ctx, s, m)
	default:
		return "", errCommandFailed
	}
}

func onFilterAdd(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	value := m.Data.Options[0].Options[0].Value.(string)
	f, err := store.InsertFilter(ctx, value)
	if err != nil {
		if err == store.ErrDuplicate {
			return "", store.ErrDuplicate
		}
		log.Errorf("Error saving filter word: %v", err)
		return "", errCommandFailed
	}
	return fmt.Sprintf("Filter added: %s (id: %d)", value, f.WordID), nil
}

func onFilterDel(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	wordId := m.Data.Options[0].Options[0].Value.(float64)
	filter, err := store.GetFilterByID(ctx, int(wordId))
	if err != nil {
		return "", err
	}
	if err2 := store.DropFilter(ctx, filter); err2 != nil {
		return "", err2
	}
	return fmt.Sprintf("Deleted filter successfully: %d", filter.WordID), nil
}

func onFilterCheck(_ context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	//value := m.Data.Options[0].Options[0].Value.(string)
	//isFiltered, filter := app.IsFilteredWord(value)
	//if !isFiltered {
	//	return fmt.Sprintf("No matching filters found for: %s", value), nil
	//}
	//return fmt.Sprintf("Matched [#%d] %s", filter.WordID, filter.Word.String()), nil
	return "", errCommandFailed
}
