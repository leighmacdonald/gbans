package discord

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/leighmacdonald/gbans/internal/action"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/external"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/query"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"strings"
	"sync"
	"time"
)

func (b *Bot) onFind(_ context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	const f = "Found player `%s` (%d) @ %s"
	userIdentifier := m.Data.Options[0].Value.(string)
	var pi model.PlayerInfo
	if err := b.executor.Find(userIdentifier, "", &pi); err != nil {
		return errCommandFailed
	}
	if !pi.Valid || !pi.InGame {
		return consts.ErrUnknownID
	}
	r.Value = fmt.Sprintf(f, pi.Player.Name, pi.SteamID.Int64(), pi.Server.ServerName)
	return nil
}

func (b *Bot) onMute(_ context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	playerID := m.Data.Options[0].Value.(string)
	reasonStr := model.Custom.String()
	if len(m.Data.Options) > 2 {
		reasonStr = m.Data.Options[2].Value.(string)
	}
	var pi model.PlayerInfo
	if err := b.executor.Mute(action.NewMute(action.Discord, playerID, m.Member.User.ID,
		reasonStr, m.Data.Options[1].Value.(string)), &pi); err != nil {
		return err
	}
	r.Value = "Player muted successfully"
	return nil
}

func (b *Bot) onBanIP(_ context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	reason := model.Custom.String()
	if len(m.Data.Options) > 2 {
		reason = m.Data.Options[2].Value.(string)
	}
	var bn model.BanNet
	if err := b.executor.BanNetwork(action.NewBanNet(action.Discord, "", m.Member.User.ID, reason,
		m.Data.Options[1].Value.(string),
		m.Data.Options[0].Value.(string)), &bn); err != nil {
		return err
	}

	go func(cidr string) {
		_, n, e := net.ParseCIDR(cidr)
		if e != nil {
			return
		}
		var pi model.PlayerInfo
		err := b.executor.FindPlayerByCIDR(n, &pi)
		if err != nil {
			return
		}
		if pi.Valid && pi.InGame {
			if resp, err7 := query.ExecRCON(*pi.Server, fmt.Sprintf("sm_kick %s", pi.Player.Name)); err7 != nil {
				log.Debug(resp)
			}
		}
	}(m.Data.Options[0].Value.(string))
	r.Value = "IP ban created successfully"
	return nil
}

// onBan !ban <id> <duration> [reason]
func (b *Bot) onBan(_ context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	reason := ""
	if len(m.Data.Options) > 2 {
		reason = m.Data.Options[2].Value.(string)
	}
	var ban model.Ban
	a := action.NewBan(action.Discord, m.Data.Options[0].Value.(string), m.Interaction.Member.User.ID,
		reason, m.Data.Options[1].Value.(string))
	if err := b.executor.Ban(a, &ban); err != nil {
		return errCommandFailed
	}
	r.Value = fmt.Sprintf("Ban created successfully (#%d)", ban.BanID)
	return nil
}

func (b *Bot) onCheck(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	sid, err := b.executor.ResolveSID(m.Data.Options[0].Value.(string))
	if err != nil {
		return consts.ErrInvalidSID
	}
	var player model.Person
	if errP := b.executor.PersonBySID(sid, "", &player); errP != nil {
		return errCommandFailed
	}
	var ban model.BannedPerson
	if errBP := b.db.GetBanBySteamID(ctx, sid, true, &ban); errBP != nil {
		if !errors.Is(errBP, store.ErrNoResult) {
			log.Errorf("Failed to get ban by steamid: %v", errBP)
			return errCommandFailed
		}
	}
	bannedNets, errBN := b.db.GetBanNet(ctx, player.IPAddr)
	if errBN != nil {
		if !errors.Is(errBN, store.ErrNoResult) {
			log.Errorf("Failed to get bannets by addr: %v", errBN)
			return errCommandFailed
		}
	}
	banned := false
	muted := false
	reason := ""
	var expiry time.Time
	// TODO Show the longest remaining ban.
	if ban.Ban.BanID > 0 {
		banned = ban.Ban.BanType == model.Banned
		muted = ban.Ban.BanType == model.NoComm
		reason = ban.Ban.ReasonText
		expiry = ban.Ban.ValidUntil
	}

	// TODO move elsewhere
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

	// TODO waitgroup?
	var asn ip2location.ASNRecord
	if errASN := b.db.GetASNRecord(ctx, player.IPAddr, &asn); errASN != nil {
		log.Warnf("Failed to fetch ASN record: %v", errASN)
	}

	var location ip2location.LocationRecord
	if errLoc := b.db.GetLocationRecord(ctx, player.IPAddr, &location); errLoc != nil {
		log.Warnf("Failed to fetch Location record: %v", errLoc)
	}

	var proxy ip2location.ProxyRecord
	if errProxy := b.db.GetProxyRecord(ctx, player.IPAddr, &proxy); errProxy != nil && errProxy != store.ErrNoResult {
		log.Errorf("Failed to fetch proxy record: %v", errProxy)
	}

	title := fmt.Sprintf("Profile of: %s", player.PersonaName)
	if ban.Ban.BanID > 0 {
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
		"Real Name", player.RealName,
		"Profile", strings.Replace(player.ProfileURL, "https://", "", 1)})
	cd := time.Unix(int64(player.TimeCreated), 0)
	t.AppendRow(table.Row{
		"Account Age", config.FmtDuration(cd),
		"Private", player.CommunityVisibilityState == 1,
	})
	t.AppendRow(table.Row{
		"STEAM64", player.SteamID.String(),
		"STEAM", steamid.SID64ToSID(player.SteamID),
	})
	t.AppendRow(table.Row{
		"STEAM3", steamid.SID64ToSID3(player.SteamID),
		"STEAM32", steamid.SID64ToSID32(player.SteamID),
	})
	vacState := "false"
	if player.VACBans > 0 {
		vacState = fmt.Sprintf("true (count: %d) (days: %d)", player.VACBans, player.DaysSinceLastBan)
	}
	gameState := "false"
	if player.GameBans > 0 {
		gameState = fmt.Sprintf("true (count: %d)", player.GameBans)
	}
	t.AppendRow(table.Row{
		"VAC Banned", vacState,
		"Game Banned", gameState,
	})
	ecoBan := "false"
	if player.EconomyBan != "none" {
		ecoBan = player.EconomyBan
	}
	t.AppendRow(table.Row{
		"Community Ban", player.CommunityBanned,
		"Economy Ban", ecoBan,
	})
	t.AppendRow(table.Row{"Banned", banned, "Muted", muted})
	if ban.Ban.BanID > 0 {
		var actAuthor model.Person
		err := b.executor.PersonBySID(ban.Ban.AuthorID, "", &actAuthor)
		if err == nil {
			r := table.Row{"Reason", reason}
			if actAuthor.DiscordID != "" {
				r = append(r, "Origin", fmt.Sprintf("<@%s>", actAuthor.DiscordID))
			}
			t.AppendRow(r)
		}
		t.AppendRow(table.Row{
			"Ban Created", config.FmtTimeShort(ban.Ban.CreatedOn),
			"Ban Expires", config.FmtDuration(expiry)})
	}
	if player.IPAddr != nil {
		t.AppendRow(table.Row{
			"Last IP", player.IPAddr,
		})
	}
	if asn.ASName != "" {
		t.AppendRow(table.Row{
			"ASN", fmt.Sprintf("(%d) %s", asn.ASNum, asn.ASName),
		})
	}
	if location.CountryCode != "" {
		t.AppendRow(table.Row{
			"City", location.CityName,
			"Country", location.CountryName,
		})
	}
	if proxy.CountryCode != "" {
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
	r.Value = t.Render()
	return nil
}

func (b *Bot) onHistory(ctx context.Context, s *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	switch m.Data.Options[0].Name {
	case string(cmdHistoryIP):
		return b.onHistoryIP(ctx, s, m, r)
	default:
		return b.onHistoryChat(ctx, s, m, r)
	}
}

func (b *Bot) onHistoryIP(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	sid, err := b.executor.ResolveSID(m.Data.Options[0].Options[0].Value.(string))
	if err != nil {
		return consts.ErrInvalidSID
	}
	var p model.Person
	if errP := b.executor.PersonBySID(sid, "", &p); errP != nil {
		return errCommandFailed
	}
	records, errIPH := b.db.GetIPHistory(ctx, sid)
	if errIPH != nil && errIPH != store.ErrNoResult {
		return errCommandFailed
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
	r.Value = t.Render()
	return nil
}

func (b *Bot) onHistoryChat(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	sid, err := b.executor.ResolveSID(m.Data.Options[0].Options[0].Value.(string))
	if err != nil {
		return consts.ErrInvalidSID
	}
	var p model.Person
	if errP := b.executor.PersonBySID(sid, "", &p); errP != nil {
		return errCommandFailed
	}
	hist, errC := b.db.GetChatHistory(ctx, sid)
	if errC != nil && errC != store.ErrNoResult {
		return errCommandFailed
	}
	t := defaultTable(fmt.Sprintf("Chat History of: %s", p.PersonaName))
	t.AppendHeader(table.Row{"Time", "Message"})
	t.AppendSeparator()
	t.SuppressEmptyColumns()
	for _, l := range hist {
		t.AppendRow(table.Row{
			config.FmtTimeShort(l.CreatedOn),
			l.Msg,
		})
	}
	r.Value = t.Render()
	return nil
}

func (b *Bot) onSetSteam(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	sid, err := b.executor.ResolveSID(m.Data.Options[0].Options[0].Value.(string))
	if err != nil {
		return consts.ErrInvalidSID
	}
	var p model.Person
	if errP := b.executor.PersonBySID(sid, "", &p); errP != nil {
		return errCommandFailed
	}
	if p.DiscordID != "" {
		return errors.New("Steam ID already set")
	}
	p.DiscordID = m.Member.User.ID
	if errS := b.db.SavePerson(ctx, &p); errS != nil {
		return errCommandFailed
	}
	r.Value = "Successfully linked your account"
	return nil
}

func (b *Bot) onUnban(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	sid, err := b.executor.ResolveSID(m.Data.Options[0].Options[0].Value.(string))
	if err != nil {
		return consts.ErrInvalidSID
	}
	var p model.Person
	if errP := b.executor.PersonBySID(sid, "", &p); errP != nil {
		return errCommandFailed
	}
	var ban model.BannedPerson
	if errB := b.db.GetBanBySteamID(ctx, sid, false, &ban); errB != nil {
		return errCommandFailed
	}
	ban.Ban.BanType = model.OK
	if errBS := b.db.SaveBan(ctx, &ban.Ban); errBS != nil {
		return errCommandFailed
	}
	r.Value = "User ban is now inactive"
	return nil
}

func (b *Bot) onKick(_ context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	sid, err := b.executor.ResolveSID(m.Data.Options[0].Options[0].Value.(string))
	if err != nil {
		return consts.ErrInvalidSID
	}
	var p model.Person
	if errP := b.executor.PersonBySID(sid, "", &p); errP != nil {
		return errCommandFailed
	}

	reason := ""
	if len(m.Data.Options) > 1 {
		reason = m.Data.Options[1].Value.(string)
	}
	var pi model.PlayerInfo
	errPI := b.executor.Kick(action.NewKick(action.Discord, p.SteamID.String(), "", reason), &pi)
	if errPI != nil {
		return errCommandFailed
	}
	r.Value = fmt.Sprintf("[%s] User kicked: %s", pi.Server.ServerName, pi.Player.Name)
	return nil
}

func (b *Bot) onSay(_ context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	server := m.Data.Options[0].Value.(string)
	msg := m.Data.Options[1].Value.(string)
	if errS := b.executor.Say(action.NewSay(action.Discord, server, msg)); errS != nil {
		return errCommandFailed
	}
	r.Value = "Sent message successfully"
	return consts.ErrInvalidSID
}

func (b *Bot) onCSay(_ context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	server := m.Data.Options[0].Value.(string)
	msg := m.Data.Options[1].Value.(string)
	if errS := b.executor.CSay(action.NewCSay(action.Discord, server, msg)); errS != nil {
		return errCommandFailed
	}
	r.Value = "Sent message successfully"
	return consts.ErrInvalidSID
}

func (b *Bot) onPSay(_ context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	player := m.Data.Options[0].Value.(string)
	msg := m.Data.Options[1].Value.(string)
	if errS := b.executor.PSay(action.NewPSay(action.Discord, player, msg)); errS != nil {
		return errCommandFailed
	}
	r.Value = "Sent message successfully"
	return consts.ErrInvalidSID
}

func (b *Bot) onServers(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	servers, err := b.db.GetServers(ctx, false)
	if err != nil {
		return errCommandFailed
	}
	full := false
	if len(m.Data.Options) > 0 {
		full = m.Data.Options[0].Value.(bool)
	}
	used, total := 0, 0
	cl := &sync.RWMutex{}
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
			cl.Lock()
			used += status.PlayersCount
			total += 32 - server.ReservedSlots
			cl.Unlock()
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
	txt := t.Render() + fmt.Sprintf("\nSum: %d/%d (%.2f%% full)", used, total, float64(used)/float64(total)*100)
	r.Value = txt
	return nil
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

func (b *Bot) onPlayers(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	var server model.Server
	if errS := b.db.GetServerByName(ctx, m.Data.Options[0].Value.(string), &server); errS != nil {
		if errS == store.ErrNoResult {
			return errors.New("Invalid server name")
		}
		return errCommandFailed
	}
	status, err2 := query.GetServerStatus(server)
	if err2 != nil {
		log.Errorf("Failed to parse status output: %v", err2)
		return model.ErrRCON
	}
	t := defaultTable("")
	t.AppendHeader(table.Row{"IP", "steam64", "Name"})
	t.AppendSeparator()
	for _, p := range status.Players {
		t.AppendRow(table.Row{p.IP, p.SID.String(), p.Name})
	}
	t.SortBy([]table.SortBy{{Name: "name", Number: 2, Mode: table.Asc}})
	r.Value = t.Render()
	return consts.ErrInvalidSID
}

func (b *Bot) onFilter(ctx context.Context, s *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	switch m.Data.Options[0].Name {
	case string(cmdFilterAdd):
		return b.onFilterAdd(ctx, s, m, r)
	case string(cmdFilterDel):
		return b.onFilterDel(ctx, s, m, r)
	case string(cmdFilterCheck):
		return b.onFilterCheck(ctx, s, m, r)
	default:
		return errCommandFailed
	}
}

func (b *Bot) onFilterAdd(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	f, errF := b.db.InsertFilter(ctx, m.Data.Options[0].Options[0].Value.(string))
	if errF != nil {
		return errCommandFailed
	}
	r.MsgType = mtString
	r.Value = fmt.Sprintf("Filter added: %s (id: %d)", f.Word.String(), f.WordID)
	return nil
}

func (b *Bot) onFilterDel(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	wordId, ok := m.Data.Options[0].Options[0].Value.(int)
	if !ok {
		return errors.New("Invalid filter id")
	}
	var f model.Filter
	if errF := b.db.GetFilterByID(ctx, wordId, &f); errF != nil {
		return errCommandFailed
	}
	if err := b.db.DropFilter(ctx, &f); err != nil {
		return errCommandFailed
	}
	r.Value = fmt.Sprintf("Deleted filter successfully: %d", int(wordId))
	r.MsgType = mtString
	return nil
}

func (b *Bot) onFilterCheck(_ context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	//value := m.Data.Options[0].Options[0].Value.(string)
	//isFiltered, filter := app.IsFilteredWord(value)
	//if !isFiltered {
	//	return fmt.Sprintf("No matching filters found for: %s", value), nil
	//}
	//return fmt.Sprintf("Matched [#%d] %s", filter.WordID, filter.Word.String()), nil
	return errCommandFailed
}
