package discord

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/leighmacdonald/gbans/internal/action"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/external"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/query"
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

func onFind(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	const f = "Found player `%s` (%d) @ %s"
	userIdentifier := m.Data.Options[0].Value.(string)
	act := action.NewFind(userIdentifier)
	res := <-act.Enqueue().Done()
	if res.Err != nil {
		return "", consts.ErrInternal
	}
	pi := res.Value.(model.PlayerInfo)
	if !pi.Valid || !pi.InGame {
		return "", consts.ErrUnknownID
	}
	return fmt.Sprintf(f, pi.Player.Name, pi.SteamID.Int64(), pi.Server.ServerName), nil
}

func onMute(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	playerID := m.Data.Options[0].Value.(string)
	reasonStr := model.Custom.String()
	if len(m.Data.Options) > 2 {
		reasonStr = m.Data.Options[2].Value.(string)
	}
	act := action.NewMute(playerID, m.Member.User.ID, reasonStr, m.Data.Options[1].Value.(string))
	res := <-act.Enqueue().Done()
	if res.Err != nil {
		return "", res.Err
	}
	return "Player muted successfully", nil
}

func onBanIP(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	reason := model.Custom.String()
	if len(m.Data.Options) > 2 {
		reason = m.Data.Options[2].Value.(string)
	}
	act := action.NewBanNet("", m.Member.User.ID, reason,
		m.Data.Options[1].Value.(string),
		m.Data.Options[0].Value.(string))
	res := <-act.Enqueue().Done()
	if res.Err != nil {
		return "", errCommandFailed
	}
	go func(cidr string) {
		_, n, e := net.ParseCIDR(cidr)
		if e != nil {
			return
		}
		fAct := action.NewFindByCIDR(n)
		fRes := <-fAct.Enqueue().Done()
		pi, ok := fRes.Value.(model.FindResult)
		if ok {
			if resp, err7 := query.ExecRCON(*pi.Server, fmt.Sprintf("sm_kick %s", pi.Player.Name)); err7 != nil {
				log.Debug(resp)
			}
		}
	}(m.Data.Options[0].Value.(string))
	return "IP ban created successfully", nil
}

// onBan !ban <id> <duration> [reason]
func onBan(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	reason := ""
	if len(m.Data.Options) > 2 {
		reason = m.Data.Options[2].Value.(string)
	}
	act := action.NewBan(
		m.Data.Options[0].Value.(string),
		m.Interaction.Member.User.ID,
		reason,
		m.Data.Options[1].Value.(string))
	res := <-act.Enqueue().Done()
	if res.Err != nil {
		return "", errCommandFailed
	}
	ban, ok := res.Value.(*model.Ban)
	if !ok {
		return "", errCommandFailed
	}
	return fmt.Sprintf("Ban created successfully (#%d)", ban.BanID), nil
}

func onCheck(ctx context.Context, s *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	sid := m.Data.Options[0].Value.(string)
	act := action.NewProfile(sid, "")
	res := <-act.Enqueue().Done()
	bannedPlayer, bannedPlayerOk := res.Value.(*model.Person)
	if !bannedPlayerOk {
		return "", errCommandFailed
	}
	br := action.NewGetBan(sid)
	brRes := <-br.Enqueue().Done()
	ban, banOk := brRes.Value.(*model.BannedPerson)

	bnr := action.NewGetBanNet(sid)
	bnrRes := <-bnr.Enqueue().Done()
	bannedNets, banNetsOk := bnrRes.Value.([]*model.BanNet)
	banned := false
	muted := false
	reason := ""
	var expiry time.Time
	// TODO Show the longest remaining ban.
	if banOk && ban != nil && ban.Ban.BanID > 0 {
		banned = ban.Ban.BanType == model.Banned
		muted = ban.Ban.BanType == model.NoComm
		reason = ban.Ban.ReasonText
		expiry = ban.Ban.ValidUntil
	}
	id, e := steamid.ResolveSID64(ctx, sid)
	if e != nil {
		return "", consts.ErrInvalidSID
	}
	// TODO move elsewhere
	logData, errLogs := external.LogsTFOverview(id)
	if errLogs != nil {
		log.Warnf("Failed to fetch logTF data: %v", errLogs)
	}
	//ip := "N/A"
	if banNetsOk && len(bannedNets) > 0 {
		//ip = bannedNets[0].CIDR.String()
		reason = fmt.Sprintf("Banned from %d networks", len(bannedNets))
		expiry = bannedNets[0].ValidUntil
	}

	actASN := action.NewGetASNRecord(bannedPlayer.IPAddr.String())
	actASN.Enqueue()
	actLocation := action.NewGetLocationRecord(bannedPlayer.IPAddr.String())
	actLocation.Enqueue()
	actProxy := action.NewGetProxyRecord(bannedPlayer.IPAddr.String())
	actProxy.Enqueue()

	// TODO waitgroup
	actASNRes := <-actASN.Done()
	actLocationRes := <-actLocation.Done()
	actProxyRes := <-actProxy.Done()

	asn, asnOk := actASNRes.Value.(*ip2location.ASNRecord)
	location, locOk := actLocationRes.Value.(*ip2location.LocationRecord)
	proxy, proOk := actProxyRes.Value.(*ip2location.ProxyRecord)

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
		actAuthor := action.NewGetPersonByID(ban.Ban.AuthorID.String())
		actRes := <-actAuthor.Enqueue().Done()
		if actRes.Err == nil {
			author, authorOK := actRes.Value.(*model.Person)
			r := table.Row{"Reason", reason}
			if authorOK && author != nil && author.DiscordID != "" {
				r = append(r, "Source", fmt.Sprintf("<@%s>", author.DiscordID))
			}
			t.AppendRow(r)
		}
		t.AppendRow(table.Row{
			"Ban Created", config.FmtTimeShort(ban.Ban.CreatedOn),
			"Ban Expires", config.FmtDuration(expiry)})
	}
	if bannedPlayer.IPAddr != nil {
		t.AppendRow(table.Row{
			"Last IP", bannedPlayer.IPAddr,
		})
	}
	if asnOk && asn != nil {
		t.AppendRow(table.Row{
			"ASN", fmt.Sprintf("(%d) %s", asn.ASNum, asn.ASName),
		})
	}
	if locOk && location != nil {
		t.AppendRow(table.Row{
			"City", location.CityName,
			"Country", location.CountryName,
		})
	}
	if proOk && proxy != nil {
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

	actP := action.NewGetPersonByID(pId)
	resP := <-actP.Enqueue().Done()
	if resP.Err != nil {
		return "", errCommandFailed
	}
	p, ok := resP.Value.(*model.Person)
	if !ok {
		return "No results", nil
	}

	act := action.NewGetHistoryIP(pId)
	res := <-act.Enqueue().Done()
	if res.Err != nil {
		return "", errCommandFailed
	}
	records, okIP := res.Value.([]model.PersonIPRecord)
	if !okIP || len(records) == 0 {
		return "No results", nil
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
	act := action.NewSetSteamID(pId, m.Member.User.ID)
	res := <-act.Enqueue().Done()
	if res.Err != nil {
		return "", res.Err
	}
	return "Successfully linked your account", nil
}

func onUnban(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	pId := m.Data.Options[0].Value.(string)
	reason := ""
	if len(m.Data.Options) > 1 {
		reason = m.Data.Options[1].Value.(string)
	}
	act := action.NewUnban(pId, m.Member.User.ID, reason)
	res := <-act.Enqueue().Done()
	if res.Err != nil {
		return "", res.Err
	}
	return "User ban is now inactive", nil
}

func onKick(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	pId := m.Data.Options[0].Value.(string)
	reason := ""
	if len(m.Data.Options) > 1 {
		reason = m.Data.Options[1].Value.(string)
	}
	act := action.NewKick(pId, m.Member.User.ID, reason)
	res := <-act.Enqueue().Done()
	pi, ok := res.Value.(*model.PlayerInfo)
	if ok {
		return fmt.Sprintf("[%s] User kicked: %s", pi.Server.ServerName, pi.Player.Name), nil
	} else {
		return "", consts.ErrInvalidSID
	}
}

func onSay(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	act := action.NewSay(m.Data.Options[0].Value.(string), m.Data.Options[1].Value.(string))
	res := <-act.Enqueue().Done()
	if res.Err != nil {
		return "", res.Err
	}
	return "Sent message successfully", nil
}

func onCSay(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	act := action.NewCSay(m.Data.Options[0].Value.(string), m.Data.Options[1].Value.(string))
	res := <-act.Enqueue().Done()
	if res.Err != nil {
		return "", res.Err
	}
	return "Sent message successfully", nil
}

func onPSay(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	act := action.NewPSay(m.Data.Options[0].Value.(string), m.Data.Options[1].Value.(string))
	res := <-act.Enqueue().Done()
	if res.Err != nil {
		return "", res.Err
	}
	return "Sent message successfully", nil
}

func onServers(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	act := action.NewServers()
	res := <-act.Enqueue().Done()
	servers, ok := res.Value.([]model.Server)
	if !ok {
		return "", errCommandFailed
	}
	full := false
	if len(m.Data.Options) > 0 {
		full = m.Data.Options[0].Value.(bool)
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
	act := action.NewServerByID(m.Data.Options[0].Value.(string))
	res := <-act.Enqueue().Done()
	if res.Err != nil {
		if res.Err == sql.ErrNoRows {
			return "", errors.New("Invalid server name")
		}
		return "", res.Err
	}
	server, ok := res.Value.(model.Server)
	if !ok {
		return "", errCommandFailed
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
	act := action.NewFilterAdd(m.Data.Options[0].Options[0].Value.(string))
	res := <-act.Enqueue().Done()
	if res.Err != nil {
		return "", res.Err
	}
	f, ok := res.Value.(*model.Filter)
	if !ok {
		return "", errCommandFailed
	}
	return fmt.Sprintf("Filter added: %s (id: %d)", f.Word.String(), f.WordID), nil
}

func onFilterDel(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate) (string, error) {
	wordId, ok := m.Data.Options[0].Options[0].Value.(float64)
	if !ok {
		return "", errors.New("Invalid filter id")
	}
	act := action.NewFilterDel(int(wordId))
	res := <-act.Enqueue().Done()
	if res.Err != nil {
		return "", res.Err
	}
	return fmt.Sprintf("Deleted filter successfully: %d", int(wordId)), nil
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
