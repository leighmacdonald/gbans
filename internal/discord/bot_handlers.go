package discord

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/action"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/external"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/query"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type discordColour int

const (
	green  discordColour = 3066993
	orange discordColour = 15105570
	red    discordColour = 10038562
)

var (
	defaultProvider = discordgo.MessageEmbedProvider{
		URL:  "https://github.com/leighmacdonald/gbans",
		Name: "gbans",
	}
	defaultFooter = discordgo.MessageEmbedFooter{
		Text:         "gbans",
		IconURL:      "https://cdn.discordapp.com/avatars/758536119397646370/6a371d1a481a72c512244ba9853f7eff.webp?size=128",
		ProxyIconURL: "",
	}
)

// RespErr creates a common error message embed
func RespErr(r *botResponse, m string) {
	r.Value = &discordgo.MessageEmbed{
		URL:      "",
		Type:     discordgo.EmbedTypeRich,
		Title:    "Command Error",
		Color:    int(red),
		Provider: &defaultProvider,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Message",
				Value:  m,
				Inline: false,
			},
		},
		Footer: &defaultFooter,
	}
	r.MsgType = mtEmbed
}

// RespOk will set up and allocate a base successful response embed that can be
// further customized
func RespOk(r *botResponse, title string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Type:        discordgo.EmbedTypeRich,
		Title:       title,
		Description: "",
		Color:       int(green),
		Footer:      &defaultFooter,
		Image:       nil,
		Thumbnail:   nil,
		Video:       nil,
		Provider:    &defaultProvider,
		Author:      nil,
		Fields:      nil,
	}
	if r != nil {
		r.MsgType = mtEmbed
		r.Value = embed
	}
	return embed
}

func (b *discord) onFind(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	userIdentifier := m.Data.Options[0].Value.(string)
	pi := model.NewPlayerInfo()
	if err := b.executor.Find(userIdentifier, "", &pi); err != nil {
		return errCommandFailed
	}
	if !pi.Valid || !pi.InGame {
		return consts.ErrUnknownID
	}
	e := RespOk(r, "Player Found")
	p := model.NewPerson(pi.SteamID)
	if errP := b.executor.GetOrCreateProfileBySteamID(ctx, pi.SteamID, "", &p); errP != nil {
		return errors.New("Failed to get profile")
	}
	e.Type = discordgo.EmbedTypeRich
	e.Image = &discordgo.MessageEmbedImage{URL: p.AvatarFull}
	e.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: p.Avatar}
	e.URL = fmt.Sprintf("https://steamcommunity.com/profiles/%d", pi.Player.SID.Int64())
	e.Title = pi.Player.Name
	addFieldInline(e, "Server", pi.Server.ServerName)
	addFieldsSteamID(e, pi.Player.SID)
	addField(e, "Connect", fmt.Sprintf("steam://%s:%d", pi.Server.Address, pi.Server.Port))
	return nil
}

func (b *discord) onMute(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	playerID := m.Data.Options[0].Value.(string)
	reasonStr := model.Custom.String()
	if len(m.Data.Options) > 2 {
		reasonStr = m.Data.Options[2].Value.(string)
	}
	author := model.NewPerson(0)
	if errA := b.db.GetPersonByDiscordID(ctx, m.Interaction.Member.User.ID, &author); errA != nil {
		if errA == store.ErrNoResult {
			return errors.New("Must set steam id. See /set_steam")
		}
		return errors.New("Error fetching author info")
	}
	var ban model.Ban
	if err := b.executor.Ban(action.NewMute(model.Bot, playerID, author.SteamID.String(),
		reasonStr, m.Data.Options[1].Value.(string)), &ban); err != nil {
		return err
	}
	e := RespOk(r, "Player muted successfully")
	addFieldsSteamID(e, ban.SteamID)
	return nil
}

func (b *discord) onBanASN(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	asNumStr := m.Data.Options[0].Options[0].Value.(string)
	duration := m.Data.Options[0].Options[1].Value.(string)
	reason := m.Data.Options[0].Options[2].Value.(string)
	targetId := steamid.SID64(0)
	if len(m.Data.Options[0].Options) > 3 {
		targetId = steamid.SID64(m.Data.Options[0].Options[3].Value.(int64))
	}
	author := model.NewPerson(0)
	if errA := b.db.GetPersonByDiscordID(ctx, m.Interaction.Member.User.ID, &author); errA != nil {
		if errA == store.ErrNoResult {
			return errors.New("Must set steam id. See /set_steam")
		}
		return errors.New("Error fetching author info")
	}
	asNum, errConv := strconv.ParseInt(asNumStr, 10, 64)
	if errConv != nil {
		return errors.New("Invalid ASN")
	}
	networks, errNets := b.db.GetASNRecordsByNum(ctx, asNum)
	if errNets != nil {
		if errNets == store.ErrNoResult {
			return errors.New("No networks found matching ASN")
		}
		return errors.New("Error fetching asn networks")
	}

	req := action.NewBanASN(model.Bot, targetId.String(), author.SteamID.String(), reason, duration, asNum)
	var ba model.BanASN
	if err := b.executor.BanASN(req, &ba); err != nil {
		if errors.Is(err, store.ErrDuplicate) {
			return errors.New("Duplicate ASN ban")
		}
		return errCommandFailed
	}
	e := RespOk(r, "ASN Ban Created Successfully")
	addField(e, "ASNum", asNumStr)
	addField(e, "Total IPs Blocked", fmt.Sprintf("%d", networks.Hosts()))
	return nil
}

func (b *discord) onBanIP(_ context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	reason := model.Custom.String()
	if len(m.Data.Options[0].Options) > 3 {
		reason = m.Data.Options[0].Options[3].Value.(string)
	}
	var bn model.BanNet
	if err := b.executor.BanNetwork(action.NewBanNet(
		model.Bot,
		m.Data.Options[0].Options[1].Value.(string),
		m.Member.User.ID, // FIXME
		reason,
		m.Data.Options[0].Options[2].Value.(string),
		m.Data.Options[0].Options[0].Value.(string)), &bn); err != nil {
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
	}(m.Data.Options[0].Options[0].Value.(string))
	RespOk(r, "IP ban created successfully")
	return nil
}

// onBanSteam !ban <id> <duration> [reason]
func (b *discord) onBanSteam(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	reason := ""
	if len(m.Data.Options[0].Options) > 2 {
		reason = m.Data.Options[0].Options[2].Value.(string)
	}
	author := model.NewPerson(0)
	if errA := b.db.GetPersonByDiscordID(ctx, m.Interaction.Member.User.ID, &author); errA != nil {
		if errA == store.ErrNoResult {
			return errors.New("Must set steam id. See /set_steam")
		}
		return errors.New("Error fetching author info")
	}
	var ban model.Ban
	a := action.NewBan(model.Bot, m.Data.Options[0].Options[0].Value.(string), author.SteamID.String(),
		reason, m.Data.Options[0].Options[1].Value.(string))
	if err := b.executor.Ban(a, &ban); err != nil {
		if errors.Is(err, store.ErrDuplicate) {
			return errors.New("Duplicate ban")
		}
		return errCommandFailed
	}
	e := RespOk(r, "User Banned")
	e.Title = fmt.Sprintf("Ban created successfully (#%d)", ban.BanID)
	e.Description = ban.Note
	if ban.ReasonText != "" {
		addField(e, "Reason", ban.ReasonText)
	}
	addFieldsSteamID(e, ban.SteamID)
	addField(e, "Expires In", config.FmtDuration(ban.ValidUntil))
	addField(e, "Expires At", config.FmtTimeShort(ban.ValidUntil))
	return nil
}

func (b *discord) onCheck(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	sid, err := b.executor.ResolveSID(m.Data.Options[0].Value.(string))
	if err != nil {
		return consts.ErrInvalidSID
	}
	player := model.NewPerson(sid)
	if errP := b.executor.GetOrCreateProfileBySteamID(ctx, sid, "", &player); errP != nil {
		return errCommandFailed
	}
	ban := model.NewBannedPerson()
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
	var (
		color     = green
		banned    = false
		muted     = false
		reason    = ""
		createdAt = ""
		a         = model.NewPerson(sid)
		author    *discordgo.MessageEmbedAuthor
		e         = RespOk(r, "")
	)
	var expiry time.Time
	// TODO Show the longest remaining ban.
	if ban.Ban.BanID > 0 {
		banned = ban.Ban.BanType == model.Banned
		muted = ban.Ban.BanType == model.NoComm
		reason = ban.Ban.ReasonText
		if len(reason) == 0 {
			// in case a ban without a reason ever makes its way here, we make sure that Discord doesn't shit itself
			reason = "none"
		}
		expiry = ban.Ban.ValidUntil
		createdAt = ban.Ban.CreatedOn.Format(time.RFC3339)
		if ban.Ban.AuthorID > 0 {
			if errA := b.executor.GetOrCreateProfileBySteamID(ctx, ban.Ban.AuthorID, "", &a); errA != nil {
				log.Errorf("Failed to load author for ban: %v", errA)
			} else {
				author = &discordgo.MessageEmbedAuthor{
					URL:     fmt.Sprintf("https://steamcommunity.com/profiles/%d", a.SteamID),
					Name:    fmt.Sprintf("<@%s>", a.DiscordID),
					IconURL: a.Avatar,
				}
			}
		}
	}
	banStateStr := "no"
	if banned {
		// #992D22 red
		color = red
		banStateStr = "banned"
	}
	if muted {
		// #E67E22 orange
		color = orange
		banStateStr = "muted"
	}
	addFieldInline(e, "Ban/Muted", banStateStr)
	// TODO move elsewhere
	logData, errLogs := external.LogsTFOverview(sid)
	if errLogs != nil {
		log.Warnf("Failed to fetch logTF data: %v", errLogs)
	}
	if len(bannedNets) > 0 {
		//ip = bannedNets[0].CIDR.String()
		reason = fmt.Sprintf("Banned from %d networks", len(bannedNets))
		expiry = bannedNets[0].ValidUntil
		addFieldInline(e, "Network Bans", fmt.Sprintf("%d", len(bannedNets)))
	}
	var (
		wg       = &sync.WaitGroup{}
		asn      ip2location.ASNRecord
		location ip2location.LocationRecord
		proxy    ip2location.ProxyRecord
	)
	wg.Add(3)
	go func() {
		defer wg.Done()
		if errASN := b.db.GetASNRecordByIP(ctx, player.IPAddr, &asn); errASN != nil {
			log.Warnf("Failed to fetch ASN record: %v", errASN)
		}
	}()
	go func() {
		defer wg.Done()
		if errLoc := b.db.GetLocationRecord(ctx, player.IPAddr, &location); errLoc != nil {
			log.Warnf("Failed to fetch Location record: %v", errLoc)
		}
	}()
	go func() {
		defer wg.Done()
		if errProxy := b.db.GetProxyRecord(ctx, player.IPAddr, &proxy); errProxy != nil && errProxy != store.ErrNoResult {
			log.Errorf("Failed to fetch proxy record: %v", errProxy)
		}
	}()
	wg.Wait()
	title := player.PersonaName
	if ban.Ban.BanID > 0 {
		if ban.Ban.BanType == model.Banned {
			title = fmt.Sprintf("%s (BANNED)", title)
		} else if ban.Ban.BanType == model.NoComm {
			title = fmt.Sprintf("%s (MUTED)", title)
		}
	}
	e.Title = title
	if player.RealName != "" {
		addFieldInline(e, "Real Name", player.RealName)
	}
	cd := time.Unix(int64(player.TimeCreated), 0)
	addFieldInline(e, "Age", config.FmtDuration(cd))
	addFieldInline(e, "Private", fmt.Sprintf("%v", player.CommunityVisibilityState == 1))
	addFieldsSteamID(e, player.SteamID)
	if player.VACBans > 0 {
		addFieldInline(e, "VAC Bans", fmt.Sprintf("count: %d days: %d", player.VACBans, player.DaysSinceLastBan))
	}
	if player.GameBans > 0 {
		addFieldInline(e, "Game Bans", fmt.Sprintf("count: %d", player.GameBans))
	}
	if player.CommunityBanned {
		addFieldInline(e, "Com. Ban", "true")
	}
	if player.EconomyBan != "" {
		addFieldInline(e, "Econ Ban", player.EconomyBan)
	}
	if ban.Ban.BanID > 0 {
		addFieldInline(e, "Reason", reason)
		addFieldInline(e, "Created", config.FmtTimeShort(ban.Ban.CreatedOn))
		if time.Until(expiry) > time.Hour*24*365*5 {
			addFieldInline(e, "Expires", "Permanent")
		} else {
			addFieldInline(e, "Expires", config.FmtDuration(expiry))
		}
		addFieldInline(e, "Author", fmt.Sprintf("<@%s>", a.DiscordID))
	}
	if player.IPAddr != nil {
		addFieldInline(e, "Last IP", player.IPAddr.String())
	}
	if asn.ASName != "" {
		addFieldInline(e, "ASN", fmt.Sprintf("(%d) %s", asn.ASNum, asn.ASName))
	}
	if location.CountryCode != "" {
		addFieldInline(e, "City", location.CityName)
	}
	if location.CountryName != "" {
		addFieldInline(e, "Country", location.CountryName)
	}
	if proxy.CountryCode != "" {
		addFieldInline(e, "Proxy Type", string(proxy.ProxyType))
		addFieldInline(e, "Proxy", string(proxy.Threat))
	}
	if logData.Total > 0 {
		addFieldInline(e, "Logs.tf", fmt.Sprintf("%d", logData.Total))
	}

	e.URL = player.ProfileURL
	e.Timestamp = createdAt
	e.Color = int(color)
	e.Image = &discordgo.MessageEmbedImage{URL: player.AvatarFull}
	e.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: player.Avatar}
	e.Video = nil
	e.Author = author
	return nil
}

func (b *discord) onHistory(ctx context.Context, s *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	switch m.Data.Options[0].Name {
	case string(cmdHistoryIP):
		return b.onHistoryIP(ctx, s, m, r)
	default:
		return b.onHistoryChat(ctx, s, m, r)
	}
}

func (b *discord) onHistoryIP(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	sid, err := b.executor.ResolveSID(m.Data.Options[0].Options[0].Value.(string))
	if err != nil {
		return consts.ErrInvalidSID
	}
	p := model.NewPerson(sid)
	if errP := b.executor.PersonBySID(sid, "", &p); errP != nil {
		return errCommandFailed
	}
	records, errIPH := b.db.GetIPHistory(ctx, sid)
	if errIPH != nil && errIPH != store.ErrNoResult {
		return errCommandFailed
	}
	e := RespOk(r, fmt.Sprintf("Chat History of: %s", p.PersonaName))
	lastIp := net.IP{}
	for _, l := range records {
		if l.IP.Equal(lastIp) {
			continue
		}
		addField(e, l.IP.String(), fmt.Sprintf("%s %s %s %s %s %s %s %s", config.FmtTimeShort(l.CreatedOn), l.CountryCode,
			l.CityName, l.ASName, l.ISP, l.UsageType, l.Threat, l.DomainUsed))
		lastIp = l.IP
	}
	e.Description = "IP history (20 max)"
	return nil
}

func (b *discord) onHistoryChat(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	sid, err := b.executor.ResolveSID(m.Data.Options[0].Options[0].Value.(string))
	if err != nil {
		return consts.ErrInvalidSID
	}
	p := model.NewPerson(sid)
	if errP := b.executor.PersonBySID(sid, "", &p); errP != nil {
		return errCommandFailed
	}
	hist, errC := b.db.GetChatHistory(ctx, sid, 25)
	if errC != nil && !errors.Is(errC, store.ErrNoResult) {
		return errCommandFailed
	}
	if errors.Is(errC, store.ErrNoResult) {
		return errors.New("No chat history found")
	}
	var lines []string
	for _, l := range hist {
		lines = append(lines, fmt.Sprintf("%s: %s", config.FmtTimeShort(l.CreatedOn), l.Msg))
	}
	e := RespOk(r, fmt.Sprintf("Chat History of: %s", p.PersonaName))
	e.Description = strings.Join(lines, "\n")
	return nil
}

func (b *discord) onSetSteam(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	sid, err := b.executor.ResolveSID(m.Data.Options[0].Value.(string))
	if err != nil {
		return consts.ErrInvalidSID
	}
	p := model.NewPerson(sid)
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
	e := RespOk(r, "Steam Account Linked")
	e.Description = "Your steam and discord accounts are now linked"
	return nil
}

func (b *discord) onUnbanSteam(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	sid, err := b.executor.ResolveSID(m.Data.Options[0].Options[0].Value.(string))
	if err != nil {
		return consts.ErrInvalidSID
	}
	p := model.NewPerson(sid)
	if errP := b.executor.PersonBySID(sid, "", &p); errP != nil {
		return errCommandFailed
	}
	ban := model.NewBannedPerson()
	if errB := b.db.GetBanBySteamID(ctx, sid, false, &ban); errB != nil {
		if errors.Is(errB, store.ErrNoResult) {
			return errors.New("No matching ban found")
		}
		return errCommandFailed
	}
	ban.Ban.BanType = model.OK
	if errBS := b.db.SaveBan(ctx, &ban.Ban); errBS != nil {
		return errCommandFailed
	}
	e := RespOk(r, "User Unbanned Successfully")
	addFieldsSteamID(e, sid)
	return nil
}

func (b *discord) onUnbanASN(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	asNumStr, ok := m.Data.Options[0].Options[0].Value.(string)
	if !ok {
		return errors.New("invalid asn")
	}
	req := action.UnbanASNRequest{
		BaseOrigin: action.BaseOrigin{Origin: model.Bot},
		ASNum:      asNumStr,
		Reason:     "",
	}
	banExisted, err := b.executor.UnbanASN(ctx, req)
	if err != nil {
		if errors.Is(err, store.ErrNoResult) {
			return errors.New("Ban for ASN does not exist")
		}
		return errCommandFailed
	}
	if !banExisted {
		return errors.New("Ban for ASN does not exist")
	}
	asNum, errConv := strconv.ParseInt(asNumStr, 10, 64)
	if errConv != nil {
		return errors.New("Invalid ASN")
	}
	networks, errNets := b.db.GetASNRecordsByNum(ctx, asNum)
	if errNets != nil {
		if errNets == store.ErrNoResult {
			return errors.New("No networks found matching ASN")
		}
		return errors.New("Error fetching asn networks")
	}
	e := RespOk(r, "ASN Networks Unbanned Successfully")
	addField(e, "ASN", asNumStr)
	addField(e, "Hosts", fmt.Sprintf("%d", networks.Hosts()))
	return nil
}

func (b *discord) onKick(_ context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	sid, err := b.executor.ResolveSID(m.Data.Options[0].Value.(string))
	if err != nil {
		return consts.ErrInvalidSID
	}
	p := model.NewPerson(sid)
	if errP := b.executor.PersonBySID(sid, "", &p); errP != nil {
		return errCommandFailed
	}
	reason := ""
	if len(m.Data.Options) > 1 {
		reason = m.Data.Options[1].Value.(string)
	}
	var pi model.PlayerInfo
	errPI := b.executor.Kick(action.NewKick(model.Bot, p.SteamID.String(), "", reason), &pi)
	if errPI != nil {
		return errCommandFailed
	}
	if pi.Server != nil && pi.Server.ServerID > 0 {
		e := RespOk(r, "User Kicked")
		addFieldsSteamID(e, sid)
		addField(e, "Name", pi.Player.Name)
	} else {
		return errors.New("User not found")
	}
	return nil
}

func (b *discord) onSay(_ context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	server := m.Data.Options[0].Value.(string)
	msg := m.Data.Options[1].Value.(string)
	if errS := b.executor.Say(action.NewSay(model.Bot, server, msg)); errS != nil {
		return errCommandFailed
	}
	e := RespOk(r, "Sent center message successfully")
	addField(e, "Server", server)
	addField(e, "Message", msg)
	return nil
}

func (b *discord) onCSay(_ context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	server := m.Data.Options[0].Value.(string)
	msg := m.Data.Options[1].Value.(string)
	if errS := b.executor.CSay(action.NewCSay(model.Bot, server, msg)); errS != nil {
		return errCommandFailed
	}
	e := RespOk(r, "Sent console message successfully")
	addField(e, "Server", server)
	addField(e, "Message", msg)
	return nil
}

func (b *discord) onPSay(_ context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	player := m.Data.Options[0].Value.(string)
	msg := m.Data.Options[1].Value.(string)
	if errS := b.executor.PSay(action.NewPSay(model.Bot, player, msg)); errS != nil {
		return errCommandFailed
	}
	e := RespOk(r, "Sent private message successfully")
	addField(e, "Player", player)
	addField(e, "Message", msg)
	return nil
}

// TODO dont hard code this
func mapRegion(n string) string {
	switch n {
	case "asia":
		return "Asia Pacific"
	case "au":
		return "Australia"
	case "eu":
		return "Europe (+UK)"
	case "sa":
		return "South America"
	case "us-east":
		return "NA East"
	case "us-west":
		return "NA West"
	case "us-central":
		return "NA Central"
	case "global":
		return "Global"
	default:
		return "Unknown"
	}
}

func (b *discord) onServers(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate, r *botResponse) error {
	state := b.executor.ServerState().ByRegion()
	stats := map[string]float64{}
	used, total := 0, 0
	e := RespOk(r, "Current Server Populations")
	e.URL = "https://uncletopia.com/servers"
	var regionNames []string
	for k := range state {
		regionNames = append(regionNames, k)
	}
	sort.Strings(regionNames)
	for _, rName := range regionNames {
		var counts []string
		for _, st := range state[rName] {
			_, ok := stats[rName]
			if !ok {
				stats[rName] = 0
				stats[rName+"total"] = 0
			}
			maxPlayers := st.Status.PlayersMax - st.Reserved
			if maxPlayers <= 0 {
				maxPlayers = 32 - st.Reserved
			}
			stats[rName] += float64(st.Status.PlayersCount)
			stats[rName+"total"] += float64(maxPlayers)
			used += st.Status.PlayersCount
			total += maxPlayers
			counts = append(counts, fmt.Sprintf("%s: %2d/%2d", st.Name, st.Status.PlayersCount, maxPlayers))
		}
		msg := strings.Join(counts, "    ")
		if msg != "" {
			addField(e, mapRegion(rName), fmt.Sprintf("```%s```", msg))
		}
	}
	for statName := range stats {
		if strings.HasSuffix(statName, "total") {
			continue
		}
		addField(e, mapRegion(statName), fmt.Sprintf("%.2f%%", (stats[statName]/stats[statName+"total"])*100))
	}
	addField(e, "Global", fmt.Sprintf("%d/%d %.2f%%", used, total, float64(used)/float64(total)*100))
	if total == 0 {
		RespErr(r, "No server state available")
	}
	return nil
}

func (b *discord) onPlayers(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	var server model.Server
	if errS := b.db.GetServerByName(ctx, m.Data.Options[0].Value.(string), &server); errS != nil {
		if errS == store.ErrNoResult {
			return errors.New("Invalid server name")
		}
		return errCommandFailed
	}
	var state model.ServerState
	ss := b.executor.ServerState()
	if !ss.ByName(server.ServerName, &state) {
		return consts.ErrUnknownID
	}
	var rows []string
	e := RespOk(r, fmt.Sprintf("Current Players: %s", server.ServerName))
	if len(state.Status.Players) > 0 {
		sort.SliceStable(state.Status.Players, func(i, j int) bool {
			return state.Status.Players[i].Name < state.Status.Players[j].Name
		})
		for _, p := range state.Status.Players {
			var asn ip2location.ASNRecord
			if errASN := b.db.GetASNRecordByIP(ctx, p.IP, &asn); errASN != nil {
				// Will fail for LAN ips
				log.Warnf("Failed to get asn record: %v", errASN)
			}
			var loc ip2location.LocationRecord
			if errLoc := b.db.GetLocationRecord(ctx, p.IP, &loc); errLoc != nil {
				log.Warnf("Failed to get location record: %v", errLoc)
			}
			proxyStr := ""
			var proxy ip2location.ProxyRecord
			if errLoc := b.db.GetProxyRecord(ctx, p.IP, &proxy); errLoc == nil {
				proxyStr = fmt.Sprintf("Threat: %s | %s | %s", proxy.ProxyType, proxy.Threat, proxy.UsageType)
			}
			flag := ""
			if loc.CountryCode != "" {
				flag = fmt.Sprintf(":flag_%s: ", strings.ToLower(loc.CountryCode))
			}
			asStr := ""
			if asn.ASNum > 0 {
				asStr = fmt.Sprintf("[ASN](https://spyse.com/target/as/%d) ", asn.ASNum)
			}
			rows = append(rows, fmt.Sprintf("%s`%d` %s`%3dms` [%s](https://steamcommunity.com/profiles/%d)%s",
				flag, p.SID, asStr, p.Ping, p.Name, p.SID, proxyStr))
		}
		e.Description = strings.Join(rows, "\n")
	} else {
		e.Description = "No players :("
	}
	return nil
}

func (b *discord) onBan(ctx context.Context, s *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	switch m.Data.Options[0].Name {
	case "steam":
		return b.onBanSteam(ctx, s, m, r)
	case "ip":
		return b.onBanIP(ctx, s, m, r)
	case "asn":
		return b.onBanASN(ctx, s, m, r)
	default:
		return errCommandFailed
	}
}
func (b *discord) onUnban(ctx context.Context, s *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	switch m.Data.Options[0].Name {
	case "steam":
		return b.onUnbanSteam(ctx, s, m, r)
	case "ip":
		return errCommandFailed
		//return b.onUnbanIP(ctx, s, m, r)
	case "asn":
		return b.onUnbanASN(ctx, s, m, r)
	default:
		return errCommandFailed
	}
}
func (b *discord) onFilter(ctx context.Context, s *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
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

func (b *discord) onFilterAdd(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	filter := m.Data.Options[0].Options[0].Value.(string)
	author := model.NewPerson(0)
	if errA := b.db.GetPersonByDiscordID(ctx, m.Interaction.Member.User.ID, &author); errA != nil {
		if errA == store.ErrNoResult {
			return errors.New("Must set steam id. See /set_steam")
		}
		return errors.New("Error fetching author info")
	}
	af, err := b.executor.FilterAdd(action.FilterAddRequest{
		BaseOrigin: action.BaseOrigin{Origin: model.Bot},
		Author:     action.Author(author.SteamID.String()),
		Filter:     filter,
	})
	if err != nil {
		return errCommandFailed
	}
	e := RespOk(r, "Filter Created Successfully")
	addFieldFilter(e, af)
	return nil
}

func (b *discord) onFilterDel(ctx context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
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
	e := RespOk(r, "Filter Deleted Successfully")
	addFieldFilter(e, f)
	return nil
}

func (b *discord) onFilterCheck(_ context.Context, _ *discordgo.Session, m *discordgo.InteractionCreate, r *botResponse) error {
	message := m.Data.Options[0].Options[0].Value.(string)
	matches := b.executor.FilterCheck(action.FilterCheckRequest{
		BaseOrigin: action.BaseOrigin{Origin: model.Bot},
		Message:    message,
	})
	title := ""
	if len(matches) == 0 {
		title = "No Match Found"
	} else {
		title = "Matched Found"
	}
	e := RespOk(r, title)
	for _, filter := range matches {
		addFieldFilter(e, filter)
	}
	return nil
}
