package app

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/query"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"net"
	"regexp"
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

// respErr creates a common error message embed
func respErr(response *botResponse, message string) {
	response.Value = &discordgo.MessageEmbed{
		URL:      "",
		Type:     discordgo.EmbedTypeRich,
		Title:    "Command Error",
		Color:    int(red),
		Provider: &defaultProvider,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Message",
				Value:  message,
				Inline: false,
			},
		},
		Footer: &defaultFooter,
	}
	response.MsgType = mtEmbed
}

// respOk will set up and allocate a base successful response embed that can be
// further customized
func respOk(response *botResponse, title string) *discordgo.MessageEmbed {
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
	if response != nil {
		response.MsgType = mtEmbed
		response.Value = embed
	}
	return embed
}

type CommandOptions map[optionKey]*discordgo.ApplicationCommandInteractionDataOption

// optionMap will take the recursive discord slash commands and flatten them into a simple
// map.
func optionMap(options []*discordgo.ApplicationCommandInteractionDataOption) CommandOptions {
	optionM := make(CommandOptions, len(options))
	for _, opt := range options {
		optionM[optionKey(opt.Name)] = opt
	}
	return optionM
}

func (opts CommandOptions) String(key optionKey) string {
	root, found := opts[key]
	if !found {
		return ""
	}
	val, ok := root.Value.(string)
	if !ok {
		return ""
	}
	return val
}

func (bot *Discord) onFind(ctx context.Context, _ *discordgo.Session, i *discordgo.InteractionCreate,
	r *botResponse) error {
	opts := optionMap(i.ApplicationCommandData().Options)
	userIdentifier := model.StringSID(opts[OptUserIdentifier].StringValue())
	playerInfo := model.NewPlayerInfo()
	if errFind := bot.app.Find(ctx, userIdentifier, "", &playerInfo); errFind != nil {
		return errCommandFailed
	}
	if !playerInfo.Valid || !playerInfo.InGame {
		return consts.ErrUnknownID
	}
	resp := respOk(r, "Player Found")
	person := model.NewPerson(playerInfo.SteamID)
	if errGetProfile := getOrCreateProfileBySteamID(ctx, bot.app.store, playerInfo.SteamID, &person); errGetProfile != nil {
		return errors.New("Failed to get profile")
	}
	resp.Type = discordgo.EmbedTypeRich
	resp.Image = &discordgo.MessageEmbedImage{URL: person.AvatarFull}
	resp.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: person.Avatar}
	resp.URL = fmt.Sprintf("https://steamcommunity.com/profiles/%d", playerInfo.Player.SID.Int64())
	resp.Title = playerInfo.Player.Name
	addFieldInline(resp, bot.logger, "Server", playerInfo.Server.ServerNameShort)
	addFieldsSteamID(resp, bot.logger, playerInfo.Player.SID)
	addField(resp, bot.logger, "Connect", fmt.Sprintf("steam://connect/%s:%d", playerInfo.Server.Address, playerInfo.Server.Port))
	return nil
}

func (bot *Discord) onMute(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	r *botResponse) error {
	opts := optionMap(interaction.ApplicationCommandData().Options)
	playerID := model.StringSID(opts.String(OptUserIdentifier))
	var reason model.Reason
	reasonValueOpt, ok := opts[OptBanReason]
	if !ok {
		return errors.New("Invalid mute reason")
	}
	reason = model.Reason(reasonValueOpt.IntValue())
	duration := model.Duration(opts[OptDuration].StringValue())
	modNote := opts[OptNote].StringValue()
	author := model.NewPerson(0)
	if errGetAuthor := bot.app.store.GetPersonByDiscordID(ctx, interaction.Interaction.Member.User.ID, &author); errGetAuthor != nil {
		if errGetAuthor == store.ErrNoResult {
			return errors.New("Must set steam id. See /set_steam")
		}
		return errors.New("Error fetching author info")
	}
	var banSteam model.BanSteam
	if errOpts := NewBanSteam(
		model.StringSID(author.SteamID.String()),
		playerID,
		duration,
		reason,
		reason.String(),
		modNote,
		model.Bot,
		0,
		model.NoComm,
		&banSteam,
	); errOpts != nil {
		return errors.Wrapf(errOpts, "Failed to parse options")
	}
	if errBan := bot.app.BanSteam(ctx, &banSteam); errBan != nil {
		return errBan
	}
	response := respOk(r, "Player muted successfully")
	addFieldsSteamID(response, bot.logger, banSteam.TargetId)
	return nil
}

func (bot *Discord) onBanASN(ctx context.Context, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate, response *botResponse) error {
	opts := optionMap(interaction.ApplicationCommandData().Options[0].Options)
	asNumStr := opts[OptASN].StringValue()
	duration := model.Duration(opts[OptDuration].StringValue())
	reason := model.Reason(opts[OptBanReason].IntValue())
	targetId := model.StringSID(opts[OptUserIdentifier].StringValue())
	modNote := opts[OptNote].StringValue()
	author := model.NewPerson(0)
	if errGetPersonByDiscordId := bot.app.store.GetPersonByDiscordID(ctx, interaction.Interaction.Member.User.ID, &author); errGetPersonByDiscordId != nil {
		if errGetPersonByDiscordId == store.ErrNoResult {
			return errors.New("Must set steam id. See /set_steam")
		}
		return errors.New("Error fetching author info")
	}
	asNum, errConv := strconv.ParseInt(asNumStr, 10, 64)
	if errConv != nil {
		return errors.New("Invalid ASN")
	}
	asnRecords, errGetASNRecords := bot.app.store.GetASNRecordsByNum(ctx, asNum)
	if errGetASNRecords != nil {
		if errGetASNRecords == store.ErrNoResult {
			return errors.New("No asnRecords found matching ASN")
		}
		return errors.New("Error fetching asn asnRecords")
	}
	var banASN model.BanASN
	if errOpts := NewBanASN(
		model.StringSID(author.SteamID.String()),
		targetId,
		duration,
		reason,
		reason.String(),
		modNote,
		model.Bot,
		asNum,
		model.Banned,
		&banASN,
	); errOpts != nil {
		return errors.Wrapf(errOpts, "Failed to parse options")
	}
	if errBanASN := bot.app.BanASN(ctx, &banASN); errBanASN != nil {
		if errors.Is(errBanASN, store.ErrDuplicate) {
			return errors.New("Duplicate ASN ban")
		}
		return errCommandFailed
	}
	resp := respOk(response, "ASN BanSteam Created Successfully")
	addField(resp, bot.logger, "ASNum", asNumStr)
	addField(resp, bot.logger, "Total IPs Blocked", fmt.Sprintf("%d", asnRecords.Hosts()))
	return nil
}

func (bot *Discord) onBanIP(ctx context.Context, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate, response *botResponse) error {
	opts := optionMap(interaction.ApplicationCommandData().Options[0].Options)
	target := model.StringSID(opts[OptUserIdentifier].StringValue())
	reason := model.Reason(opts[OptBanReason].IntValue())
	cidr := opts[OptCIDR].StringValue()
	duration := model.Duration(opts[OptDuration].StringValue())
	modNote := opts[OptNote].StringValue()
	author := model.NewPerson(0)
	if errGetPerson := bot.app.store.GetPersonByDiscordID(ctx, interaction.Interaction.Member.User.ID, &author); errGetPerson != nil {
		if errGetPerson == store.ErrNoResult {
			return errors.New("Must set steam id. See /set_steam")
		}
		return errors.New("Error fetching author info")
	}

	var banCIDR model.BanCIDR
	if errOpts := NewBanCIDR(
		model.StringSID(author.SteamID.String()),
		target,
		duration,
		reason,
		reason.String(),
		modNote,
		model.Bot,
		cidr,
		model.Banned,
		&banCIDR,
	); errOpts != nil {
		return errors.Wrapf(errOpts, "Failed to parse options")
	}
	if errBanNet := bot.app.BanCIDR(ctx, &banCIDR); errBanNet != nil {
		return errBanNet
	}

	go func(cidrValue string) {
		_, network, errParseCIDR := net.ParseCIDR(cidrValue)
		if errParseCIDR != nil {
			return
		}
		var playerInfo model.PlayerInfo
		errFindPlayer := bot.app.FindPlayerByCIDR(ctx, network, &playerInfo)
		if errFindPlayer != nil {
			return
		}
		if playerInfo.Valid && playerInfo.InGame {
			if resp, err7 := query.ExecRCON(ctx, *playerInfo.Server, fmt.Sprintf("sm_kick %s", playerInfo.Player.Name)); err7 != nil {
				bot.logger.Debug(resp)
			}
		}
	}(cidr)
	respOk(response, "IP ban created successfully")
	return nil
}

// onBanSteam !ban <id> <duration> [reason]
func (bot *Discord) onBanSteam(ctx context.Context, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate, response *botResponse) error {
	opts := optionMap(interaction.ApplicationCommandData().Options[0].Options)
	target := opts[OptUserIdentifier].StringValue()
	reason := model.Reason(opts[OptBanReason].IntValue())
	modNote := opts[OptNote].StringValue()
	duration := model.Duration(opts[OptDuration].StringValue())
	author := model.NewPerson(0)
	if errGetAuthor := bot.app.store.GetPersonByDiscordID(ctx, interaction.Interaction.Member.User.ID, &author); errGetAuthor != nil {
		if errGetAuthor == store.ErrNoResult {
			return errors.New("Must set steam id. See /set_steam")
		}
		return errors.New("Error fetching author info")
	}
	var banSteam model.BanSteam
	if errOpts := NewBanSteam(
		model.StringSID(author.SteamID.String()),
		model.StringSID(target),
		duration,
		reason,
		reason.String(),
		modNote,
		model.Bot,
		0,
		model.Banned,
		&banSteam,
	); errOpts != nil {
		return errors.Wrapf(errOpts, "Failed to parse options")
	}
	if errBan := bot.app.BanSteam(ctx, &banSteam); errBan != nil {
		if errors.Is(errBan, store.ErrDuplicate) {
			return errors.New("Duplicate ban")
		}
		bot.logger.Error("Failed to execute ban", zap.Error(errBan))
		return errCommandFailed
	}
	createDiscordBanEmbed(banSteam, bot.logger, response)
	return nil
}

func createDiscordBanEmbed(ban model.BanSteam, logger *zap.Logger, response *botResponse) *discordgo.MessageEmbed {
	embed := respOk(response, "User Banned")
	embed.Title = fmt.Sprintf("Ban created successfully (#%d)", ban.BanID)
	embed.Description = ban.Note
	if ban.ReasonText != "" {
		addField(embed, logger, "Reason", ban.ReasonText)
	}
	addFieldsSteamID(embed, logger, ban.TargetId)
	if ban.ValidUntil.Year()-config.Now().Year() > 5 {
		addField(embed, logger, "Expires In", "Permanent")
		addField(embed, logger, "Expires At", "Permanent")
	} else {
		addField(embed, logger, "Expires In", config.FmtDuration(ban.ValidUntil))
		addField(embed, logger, "Expires At", config.FmtTimeShort(ban.ValidUntil))
	}
	return embed
}

func (bot *Discord) onCheck(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *botResponse) error {
	opts := optionMap(interaction.ApplicationCommandData().Options)
	sid, errResolveSID := ResolveSID(ctx, opts[OptUserIdentifier].StringValue())
	if errResolveSID != nil {
		return consts.ErrInvalidSID
	}
	player := model.NewPerson(sid)
	if errGetPlayer := getOrCreateProfileBySteamID(ctx, bot.app.store, sid, &player); errGetPlayer != nil {
		return errCommandFailed
	}
	ban := model.NewBannedPerson()
	if errGetBanBySID := bot.app.store.GetBanBySteamID(ctx, sid, &ban, false); errGetBanBySID != nil {
		if !errors.Is(errGetBanBySID, store.ErrNoResult) {
			bot.logger.Error("Failed to get ban by steamid", zap.Error(errGetBanBySID))
			return errCommandFailed
		}
	}
	q := store.NewBansQueryFilter(sid)
	q.Deleted = true
	// TODO Get count of old bans
	oldBans, errOld := bot.app.store.GetBansSteam(ctx, q)
	if errOld != nil {
		if !errors.Is(errOld, store.ErrNoResult) {
			bot.logger.Error("Failed to fetch old bans", zap.Error(errOld))
		}
	}

	bannedNets, errGetBanNet := bot.app.store.GetBanNetByAddress(ctx, player.IPAddr)
	if errGetBanNet != nil {
		if !errors.Is(errGetBanNet, store.ErrNoResult) {
			bot.logger.Error("Failed to get ban nets by addr", zap.Error(errGetBanNet))
			return errCommandFailed
		}
	}
	var (
		color         = green
		banned        = false
		muted         = false
		reason        = ""
		createdAt     = ""
		authorProfile = model.NewPerson(sid)
		author        *discordgo.MessageEmbedAuthor
		embed         = respOk(response, "")
	)
	var expiry time.Time
	// TODO Show the longest remaining ban.
	if ban.Ban.BanID > 0 {
		banned = ban.Ban.BanType == model.Banned
		muted = ban.Ban.BanType == model.NoComm
		reason = ban.Ban.ReasonText
		if len(reason) == 0 {
			// in case authorProfile ban without authorProfile reason ever makes its way here, we make sure
			// that Discord doesn't shit itself
			reason = "none"
		}
		expiry = ban.Ban.ValidUntil
		createdAt = ban.Ban.CreatedOn.Format(time.RFC3339)
		if ban.Ban.SourceId > 0 {
			if errGetProfile := getOrCreateProfileBySteamID(ctx, bot.app.store, ban.Ban.SourceId, &authorProfile); errGetProfile != nil {
				bot.logger.Error("Failed to load author for ban", zap.Error(errGetProfile))
			} else {
				author = &discordgo.MessageEmbedAuthor{
					URL:     fmt.Sprintf("https://steamcommunity.com/profiles/%d", authorProfile.SteamID),
					Name:    fmt.Sprintf("<@%s>", authorProfile.DiscordID),
					IconURL: authorProfile.Avatar,
				}
			}
		}
		addLink(embed, bot.logger, ban.Ban)
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
	addFieldInline(embed, bot.logger, "Ban/Muted", banStateStr)
	// TODO move elsewhere
	logData, errLogs := thirdparty.LogsTFOverview(ctx, sid)
	if errLogs != nil {
		bot.logger.Warn("Failed to fetch logTF data", zap.Error(errLogs))
	}
	if len(bannedNets) > 0 {
		//ip = bannedNets[0].CIDR.String()
		reason = fmt.Sprintf("Banned from %d networks", len(bannedNets))
		expiry = bannedNets[0].ValidUntil
		addFieldInline(embed, bot.logger, "Network Bans", fmt.Sprintf("%d", len(bannedNets)))
	}
	var (
		waitGroup = &sync.WaitGroup{}
		asn       ip2location.ASNRecord
		location  ip2location.LocationRecord
		proxy     ip2location.ProxyRecord
	)
	waitGroup.Add(3)
	go func() {
		defer waitGroup.Done()
		if player.IPAddr != nil {
			if errASN := bot.app.store.GetASNRecordByIP(ctx, player.IPAddr, &asn); errASN != nil {
				bot.logger.Error("Failed to fetch ASN record", zap.Error(errASN))
			}
		}
	}()
	go func() {
		defer waitGroup.Done()
		if player.IPAddr != nil {
			if errLoc := bot.app.store.GetLocationRecord(ctx, player.IPAddr, &location); errLoc != nil {
				bot.logger.Error("Failed to fetch Location record", zap.Error(errLoc))
			}
		}
	}()
	go func() {
		defer waitGroup.Done()
		if player.IPAddr != nil {
			if errProxy := bot.app.store.GetProxyRecord(ctx, player.IPAddr, &proxy); errProxy != nil && errProxy != store.ErrNoResult {
				bot.logger.Error("Failed to fetch proxy record", zap.Error(errProxy))
			}
		}
	}()
	waitGroup.Wait()
	title := player.PersonaName
	if ban.Ban.BanID > 0 {
		if ban.Ban.BanType == model.Banned {
			title = fmt.Sprintf("%s (BANNED)", title)
		} else if ban.Ban.BanType == model.NoComm {
			title = fmt.Sprintf("%s (MUTED)", title)
		}
	}
	embed.Title = title
	if player.RealName != "" {
		addFieldInline(embed, bot.logger, "Real Name", player.RealName)
	}
	cd := time.Unix(int64(player.TimeCreated), 0)
	addFieldInline(embed, bot.logger, "Age", config.FmtDuration(cd))
	addFieldInline(embed, bot.logger, "Private", fmt.Sprintf("%v", player.CommunityVisibilityState == 1))
	addFieldsSteamID(embed, bot.logger, player.SteamID)
	if player.VACBans > 0 {
		addFieldInline(embed, bot.logger, "VAC Bans", fmt.Sprintf("count: %d days: %d", player.VACBans, player.DaysSinceLastBan))
	}
	if player.GameBans > 0 {
		addFieldInline(embed, bot.logger, "Game Bans", fmt.Sprintf("count: %d", player.GameBans))
	}
	if player.CommunityBanned {
		addFieldInline(embed, bot.logger, "Com. Ban", "true")
	}
	if player.EconomyBan != "" {
		addFieldInline(embed, bot.logger, "Econ Ban", player.EconomyBan)
	}
	if len(oldBans) > 0 {
		numMutes, numBans := 0, 0
		for _, ob := range oldBans {
			if ob.Ban.BanType == model.Banned {
				numBans++
			} else {
				numMutes++
			}
		}
		addFieldInline(embed, bot.logger, "Total Mutes", fmt.Sprintf("%d", numMutes))
		addFieldInline(embed, bot.logger, "Total Bans", fmt.Sprintf("%d", numBans))
	}
	if ban.Ban.BanID > 0 {
		addFieldInline(embed, bot.logger, "Reason", reason)
		addFieldInline(embed, bot.logger, "Created", config.FmtTimeShort(ban.Ban.CreatedOn))
		if time.Until(expiry) > time.Hour*24*365*5 {
			addFieldInline(embed, bot.logger, "Expires", "Permanent")
		} else {
			addFieldInline(embed, bot.logger, "Expires", config.FmtDuration(expiry))
		}
		addFieldInline(embed, bot.logger, "Author", fmt.Sprintf("<@%s>", authorProfile.DiscordID))
		if ban.Ban.Note != "" {
			addField(embed, bot.logger, "Mod Note", ban.Ban.Note)
		}
	}
	if player.IPAddr != nil {
		addFieldInline(embed, bot.logger, "Last IP", player.IPAddr.String())
	}
	if asn.ASName != "" {
		addFieldInline(embed, bot.logger, "ASN", fmt.Sprintf("(%d) %s", asn.ASNum, asn.ASName))
	}
	if location.CountryCode != "" {
		addFieldInline(embed, bot.logger, "City", location.CityName)
	}
	if location.CountryName != "" {
		addFieldInline(embed, bot.logger, "Country", location.CountryName)
	}
	if proxy.CountryCode != "" {
		addFieldInline(embed, bot.logger, "Proxy Type", string(proxy.ProxyType))
		addFieldInline(embed, bot.logger, "Proxy", string(proxy.Threat))
	}
	if logData != nil && logData.Total > 0 {
		addFieldInline(embed, bot.logger, "Logs.tf", fmt.Sprintf("%d", logData.Total))
	}

	embed.URL = player.ProfileURL
	embed.Timestamp = createdAt
	embed.Color = int(color)
	embed.Image = &discordgo.MessageEmbedImage{URL: player.AvatarFull}
	embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: player.Avatar}
	embed.Video = nil
	embed.Author = author
	return nil
}

func (bot *Discord) onHistory(ctx context.Context, session *discordgo.Session,
	interaction *discordgo.InteractionCreate, response *botResponse) error {
	switch interaction.ApplicationCommandData().Name {
	case string(cmdHistoryIP):
		return bot.onHistoryIP(ctx, session, interaction, response)
	default:
		return errCommandFailed
		//return bot.onHistoryChat(ctx, session, interaction, response)
	}
}

func (bot *Discord) onHistoryIP(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *botResponse) error {
	opts := optionMap(interaction.ApplicationCommandData().Options[0].Options)
	steamId, errResolve := ResolveSID(ctx, opts[OptUserIdentifier].StringValue())
	if errResolve != nil {
		return consts.ErrInvalidSID
	}
	person := model.NewPerson(steamId)
	if errPersonBySID := bot.app.PersonBySID(ctx, bot.app.store, steamId, &person); errPersonBySID != nil {
		return errCommandFailed
	}
	ipRecords, errGetPersonIPHist := bot.app.store.GetPersonIPHistory(ctx, steamId, 20)
	if errGetPersonIPHist != nil && errGetPersonIPHist != store.ErrNoResult {
		return errCommandFailed
	}
	embed := respOk(response, fmt.Sprintf("IP History of: %s", person.PersonaName))
	lastIp := net.IP{}
	for _, ipRecord := range ipRecords {
		if ipRecord.IPAddr.Equal(lastIp) {
			continue
		}
		// TODO Join query for connections and geoip lookup data
		//addField(embed, ipRecord.IpAddr.String(), fmt.Sprintf("%s %s %s %s %s %s %s %s", config.FmtTimeShort(ipRecord.CreatedOn), ipRecord.CountryCode,
		//	ipRecord.CityName, ipRecord.ASName, ipRecord.ISP, ipRecord.UsageType, ipRecord.Threat, ipRecord.DomainUsed))
		//lastIp = ipRecord.IpAddr
	}
	embed.Description = "IP history (20 max)"
	return nil
}

//
//func (bot *discord) onHistoryChat(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, response *botResponse) error {
//	steamId, errResolveSID := ResolveSID(ctx, interaction.Data.Options[0].Options[0].Value.(string))
//	if errResolveSID != nil {
//		return consts.ErrInvalidSID
//	}
//	person := model.NewPerson(steamId)
//	if errPersonBySID := PersonBySID(ctx, bot.database, steamId, "", &person); errPersonBySID != nil {
//		return errCommandFailed
//	}
//	chatHistory, errChatHistory := bot.database.GetChatHistory(ctx, steamId, 25)
//	if errChatHistory != nil && !errors.Is(errChatHistory, store.ErrNoResult) {
//		return errCommandFailed
//	}
//	if errors.Is(errChatHistory, store.ErrNoResult) {
//		return errors.New("No chat history found")
//	}
//	var lines []string
//	for _, sayEvent := range chatHistory {
//		lines = append(lines, fmt.Sprintf("%s: %s", config.FmtTimeShort(sayEvent.CreatedOn), sayEvent.Msg))
//	}
//	embed := respOk(response, fmt.Sprintf("Chat History of: %s", person.PersonaName))
//	embed.Description = strings.Join(lines, "\n")
//	return nil
//}

func (bot *Discord) onSetSteam(ctx context.Context, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate, response *botResponse) error {
	opts := optionMap(interaction.ApplicationCommandData().Options)
	steamId, errResolveSID := ResolveSID(ctx, opts[OptUserIdentifier].StringValue())
	if errResolveSID != nil {
		return consts.ErrInvalidSID
	}
	errSetSteam := bot.app.SetSteam(ctx, steamId, interaction.Member.User.ID)
	if errSetSteam != nil {
		return errSetSteam
	}
	embed := respOk(response, "Steam Account Linked")
	embed.Description = "Your steam and discord accounts are now linked"
	return nil
}

func (bot *Discord) onUnbanSteam(ctx context.Context, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate, response *botResponse) error {
	opts := optionMap(interaction.ApplicationCommandData().Options[0].Options)
	reason := opts[OptUnbanReason].StringValue()
	steamId, errResolveSID := ResolveSID(ctx, opts[OptUserIdentifier].StringValue())
	if errResolveSID != nil {
		return consts.ErrInvalidSID
	}
	found, errUnban := bot.app.Unban(ctx, steamId, reason)
	if errUnban != nil {
		return errUnban
	}
	if !found {
		return errors.New("No ban found")
	}
	embed := respOk(response, "User Unbanned Successfully")
	addFieldsSteamID(embed, bot.logger, steamId)
	return nil
}

func (bot *Discord) onUnbanASN(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *botResponse) error {
	opts := optionMap(interaction.ApplicationCommandData().Options[0].Options)
	asNumStr := opts[OptASN].StringValue()
	banExisted, errUnbanASN := bot.app.UnbanASN(ctx, asNumStr)
	if errUnbanASN != nil {
		if errors.Is(errUnbanASN, store.ErrNoResult) {
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
	asnNetworks, errGetASNRecords := bot.app.store.GetASNRecordsByNum(ctx, asNum)
	if errGetASNRecords != nil {
		if errGetASNRecords == store.ErrNoResult {
			return errors.New("No asnNetworks found matching ASN")
		}
		return errors.New("Error fetching asn asnNetworks")
	}
	embed := respOk(response, "ASN Networks Unbanned Successfully")
	addField(embed, bot.logger, "ASN", asNumStr)
	addField(embed, bot.logger, "Hosts", fmt.Sprintf("%d", asnNetworks.Hosts()))
	return nil
}

func (bot *Discord) onKick(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *botResponse) error {
	opts := optionMap(interaction.ApplicationCommandData().Options)
	target := model.StringSID(opts[OptUserIdentifier].StringValue())
	reason := model.Reason(opts[OptBanReason].IntValue())
	targetSid64, errTarget := target.SID64()
	if errTarget != nil {
		return consts.ErrInvalidSID
	}
	person := model.NewPerson(targetSid64)
	if errPersonBySID := bot.app.PersonBySID(ctx, bot.app.store, targetSid64, &person); errPersonBySID != nil {
		return errCommandFailed
	}
	var playerInfo model.PlayerInfo
	errKick := bot.app.Kick(ctx, model.Bot, target, "", reason, &playerInfo)
	if errKick != nil {
		return errCommandFailed
	}
	if playerInfo.Server != nil && playerInfo.Server.ServerID > 0 {
		embed := respOk(response, "User Kicked")
		addFieldsSteamID(embed, bot.logger, targetSid64)
		addField(embed, bot.logger, "NameShort", playerInfo.Player.Name)
	} else {
		return errors.New("User not found")
	}
	return nil
}

func (bot *Discord) onSay(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *botResponse) error {
	opts := optionMap(interaction.ApplicationCommandData().Options)
	server := opts[OptServerIdentifier].StringValue()
	msg := opts[OptMessage].StringValue()
	if errSay := bot.app.Say(ctx, 0, server, msg); errSay != nil {
		return errCommandFailed
	}
	embed := respOk(response, "Sent center message successfully")
	addField(embed, bot.logger, "Server", server)
	addField(embed, bot.logger, "Message", msg)
	return nil
}

func (bot *Discord) onCSay(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *botResponse) error {
	opts := optionMap(interaction.ApplicationCommandData().Options)
	server := opts[OptServerIdentifier].StringValue()
	msg := opts[OptMessage].StringValue()
	if errCSay := bot.app.CSay(ctx, 0, server, msg); errCSay != nil {
		return errCommandFailed
	}
	embed := respOk(response, "Sent console message successfully")
	addField(embed, bot.logger, "Server", server)
	addField(embed, bot.logger, "Message", msg)
	return nil
}

func (bot *Discord) onPSay(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *botResponse) error {
	opts := optionMap(interaction.ApplicationCommandData().Options)
	player := model.StringSID(opts[OptUserIdentifier].StringValue())
	msg := opts[OptMessage].StringValue()
	if errPSay := bot.app.PSay(ctx, 0, player, msg, nil); errPSay != nil {
		return errCommandFailed
	}
	embed := respOk(response, "Sent private message successfully")
	addField(embed, bot.logger, "Player", string(player))
	addField(embed, bot.logger, "Message", msg)
	return nil
}

// TODO dont hard code this
func mapRegion(region string) string {
	switch region {
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

func (bot *Discord) onServers(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate,
	response *botResponse) error {
	state := bot.app.ServerState().ByRegion()
	stats := map[string]float64{}
	used, total := 0, 0
	embed := respOk(response, "Current Server Populations")
	embed.URL = config.ExtURL("/servers")
	var regionNames []string
	for k := range state {
		regionNames = append(regionNames, k)
	}
	sort.Strings(regionNames)
	for _, region := range regionNames {
		var counts []string
		for _, st := range state[region] {
			_, ok := stats[region]
			if !ok {
				stats[region] = 0
				stats[region+"total"] = 0
			}
			maxPlayers := st.MaxPlayers - st.Reserved
			if maxPlayers <= 0 {
				maxPlayers = 32 - st.Reserved
			}
			stats[region] += float64(st.PlayerCount)
			stats[region+"total"] += float64(maxPlayers)
			used += st.PlayerCount
			total += maxPlayers
			counts = append(counts, fmt.Sprintf("%s: %2d/%2d", st.NameShort, st.PlayerCount, maxPlayers))
		}
		msg := strings.Join(counts, "    ")
		if msg != "" {
			addField(embed, bot.logger, mapRegion(region), fmt.Sprintf("```%s```", msg))
		}
	}
	for statName := range stats {
		if strings.HasSuffix(statName, "total") {
			continue
		}
		addField(embed, bot.logger, mapRegion(statName), fmt.Sprintf("%.2f%%", (stats[statName]/stats[statName+"total"])*100))
	}
	addField(embed, bot.logger, "Global", fmt.Sprintf("%d/%d %.2f%%", used, total, float64(used)/float64(total)*100))
	if total == 0 {
		respErr(response, "No server state available")
	}
	return nil
}

func (bot *Discord) onPlayers(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *botResponse) error {
	opts := optionMap(interaction.ApplicationCommandData().Options)
	serverName := opts[OptServerIdentifier].StringValue()
	var server model.Server
	if errGetServer := bot.app.store.GetServerByName(ctx, serverName, &server); errGetServer != nil {
		if errGetServer == store.ErrNoResult {
			return errors.New("Invalid server name")
		}
		return errCommandFailed
	}
	var state model.ServerState
	serverStates := bot.app.ServerState()
	if !serverStates.ByName(server.ServerNameShort, &state) {
		return consts.ErrUnknownID
	}
	var rows []string
	embed := respOk(response, fmt.Sprintf("Current Players: %s", server.ServerNameShort))
	if len(state.Players) > 0 {
		sort.SliceStable(state.Players, func(i, j int) bool {
			return state.Players[i].Name < state.Players[j].Name
		})
		for _, player := range state.Players {
			var asn ip2location.ASNRecord
			if errASN := bot.app.store.GetASNRecordByIP(ctx, player.IP, &asn); errASN != nil {
				// Will fail for LAN ips
				bot.logger.Warn("Failed to get asn record", zap.Error(errASN))
			}
			var loc ip2location.LocationRecord
			if errLoc := bot.app.store.GetLocationRecord(ctx, player.IP, &loc); errLoc != nil {
				bot.logger.Warn("Failed to get location record: %v", zap.Error(errLoc))
			}
			proxyStr := ""
			var proxy ip2location.ProxyRecord
			if errGetProxyRecord := bot.app.store.GetProxyRecord(ctx, player.IP, &proxy); errGetProxyRecord == nil {
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
				flag, player.SID, asStr, player.Ping, player.Name, player.SID, proxyStr))
		}
		embed.Description = strings.Join(rows, "\n")
	} else {
		embed.Description = "No players :("
	}
	return nil
}

func (bot *Discord) onBan(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *botResponse) error {
	name := interaction.ApplicationCommandData().Options[0].Name
	switch name {
	case "steam":
		return bot.onBanSteam(ctx, session, interaction, response)
	case "ip":
		return bot.onBanIP(ctx, session, interaction, response)
	case "asn":
		return bot.onBanASN(ctx, session, interaction, response)
	default:
		bot.logger.Error("Invalid ban type selected", zap.String("type", name))
		return errCommandFailed
	}
}

func (bot *Discord) onUnban(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *botResponse) error {
	switch interaction.ApplicationCommandData().Options[0].Name {
	case "steam":
		return bot.onUnbanSteam(ctx, session, interaction, response)
	case "ip":
		return errCommandFailed
		//return bot.onUnbanIP(ctx, session, interaction, response)
	case "asn":
		return bot.onUnbanASN(ctx, session, interaction, response)
	default:
		return errCommandFailed
	}
}

func (bot *Discord) onFilter(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *botResponse) error {
	switch interaction.ApplicationCommandData().Options[0].Name {
	case "add":
		return bot.onFilterAdd(ctx, session, interaction, response)
	case "del":
		return bot.onFilterDel(ctx, session, interaction, response)
	case "check":
		return bot.onFilterCheck(ctx, session, interaction, response)
	default:
		return errCommandFailed
	}
}

func (bot *Discord) onFilterAdd(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *botResponse) error {
	opts := optionMap(interaction.ApplicationCommandData().Options[0].Options)
	pattern := opts["pattern"].StringValue()
	isRegex := opts["is_regex"].BoolValue()
	author := model.NewPerson(0)
	if errPersonByDiscordID := bot.app.store.GetPersonByDiscordID(ctx, interaction.Interaction.Member.User.ID, &author); errPersonByDiscordID != nil {
		if errPersonByDiscordID == store.ErrNoResult {
			return errors.New("Must set steam id. See /set_steam")
		}
		return errors.New("Error fetching author info")
	}
	if isRegex {
		_, rxErr := regexp.Compile(pattern)
		if rxErr != nil {
			return errors.Errorf("Invalid regular expression: %v", rxErr)
		}
	}
	filter := model.Filter{
		AuthorId:  author.SteamID,
		Pattern:   pattern,
		IsRegex:   isRegex,
		IsEnabled: true,
		CreatedOn: time.Now(),
		UpdatedOn: time.Now(),
	}
	if errFilterAdd := bot.app.FilterAdd(ctx, &filter); errFilterAdd != nil {
		return errCommandFailed
	}
	embed := respOk(response, "Filter Created Successfully")
	embed.Description = filter.Pattern
	return nil
}

func (bot *Discord) onFilterDel(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *botResponse) error {
	opts := optionMap(interaction.ApplicationCommandData().Options[0].Options)
	wordId := opts["filter"].IntValue()
	if wordId <= 0 {
		return errors.New("Invalid filter id")
	}
	var filter model.Filter
	if errGetFilter := bot.app.store.GetFilterByID(ctx, wordId, &filter); errGetFilter != nil {
		return errCommandFailed
	}
	if errDropFilter := bot.app.store.DropFilter(ctx, &filter); errDropFilter != nil {
		return errCommandFailed
	}
	respOk(response, "Filter Deleted Successfully")
	return nil
}

func (bot *Discord) onFilterCheck(_ context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *botResponse) error {
	opts := optionMap(interaction.ApplicationCommandData().Options[0].Options)
	message := opts[OptMessage].StringValue()
	matches := bot.app.FilterCheck(message)
	title := ""
	if len(matches) == 0 {
		title = "No Match Found"
	} else {
		title = "Matched Found"
	}
	respOk(response, title)
	return nil
}

//func (bot *discord) onStats(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, response *botResponse) error {
//	switch interaction.Data.Options[0].Name {
//	case string(cmdStatsPlayer):
//		return bot.onStatsPlayer(ctx, session, interaction, response)
//	case string(cmdStatsGlobal):
//		return bot.onStatsGlobal(ctx, session, interaction, response)
//	case string(cmdStatsServer):
//		return bot.onStatsServer(ctx, session, interaction, response)
//	default:
//		return errCommandFailed
//	}
//}
//
//func (bot *discord) onStatsPlayer(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, response *botResponse) error {
//	target := model.Target(interaction.Data.Options[0].Options[0].Value.(string))
//	sid, errSid := target.SID64()
//	if errSid != nil {
//		return errSid
//	}
//	person := model.NewPerson(sid)
//	if errPersonBySID := PersonBySID(ctx, bot.database, sid, "", &person); errPersonBySID != nil {
//		return errCommandFailed
//	}
//	var stats model.PlayerStats
//	if errStats := bot.database.GetPlayerStats(ctx, sid, &stats); errStats != nil {
//		return errCommandFailed
//	}
//	kd := 0.0
//	if stats.Deaths > 0 && stats.Kills > 0 {
//		kd = float64(stats.Kills) / float64(stats.Deaths)
//	}
//	kad := 0.0
//	if stats.Deaths > 0 && (stats.Kills+stats.Assists) > 0 {
//		kad = float64(stats.Kills+stats.Assists) / float64(stats.Deaths)
//	}
//	acc := 0.0
//	if stats.Hits > 0 && stats.Shots > 0 {
//		acc = float64(stats.Hits) / float64(stats.Shots) * 100
//	}
//	embed := respOk(response, fmt.Sprintf("Player stats for %s (%d)", person.PersonaName, person.SteamID.Int64()))
//	addFieldInline(embed, "Kills", fmt.Sprintf("%d", stats.Kills))
//	addFieldInline(embed, "Deaths", fmt.Sprintf("%d", stats.Deaths))
//	addFieldInline(embed, "Assists", fmt.Sprintf("%d", stats.Assists))
//	addFieldInline(embed, "K:D", fmt.Sprintf("%.2f", kd))
//	addFieldInline(embed, "KA:D", fmt.Sprintf("%.2f", kad))
//	addFieldInline(embed, "Damage", fmt.Sprintf("%d", stats.Damage))
//	addFieldInline(embed, "DamageTaken", fmt.Sprintf("%d", stats.DamageTaken))
//	addFieldInline(embed, "Healing", fmt.Sprintf("%d", stats.Healing))
//	addFieldInline(embed, "Shots", fmt.Sprintf("%d", stats.Shots))
//	addFieldInline(embed, "Hits", fmt.Sprintf("%d", stats.Hits))
//	addFieldInline(embed, "Accuracy", fmt.Sprintf("%.2f%%", acc))
//	return nil
//}
//
//func (bot *discord) onStatsServer(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, response *botResponse) error {
//	serverIdStr := interaction.Data.Options[0].Options[0].Value.(string)
//	var (
//		server model.Server
//		stats  model.ServerStats
//	)
//	if errServer := bot.database.GetServerByName(ctx, serverIdStr, &server); errServer != nil {
//		return errServer
//	}
//	if errStats := bot.database.GetServerStats(ctx, server.ServerID, &stats); errStats != nil {
//		return errCommandFailed
//	}
//	acc := 0.0
//	if stats.Hits > 0 && stats.Shots > 0 {
//		acc = float64(stats.Hits) / float64(stats.Shots) * 100
//	}
//	embed := respOk(response, fmt.Sprintf("Server stats for %s ", server.ServerNameShort))
//	addFieldInline(embed, "Kills", fmt.Sprintf("%d", stats.Kills))
//	addFieldInline(embed, "Assists", fmt.Sprintf("%d", stats.Assists))
//	addFieldInline(embed, "Damage", fmt.Sprintf("%d", stats.Damage))
//	addFieldInline(embed, "Healing", fmt.Sprintf("%d", stats.Healing))
//	addFieldInline(embed, "Shots", fmt.Sprintf("%d", stats.Shots))
//	addFieldInline(embed, "Hits", fmt.Sprintf("%d", stats.Hits))
//	addFieldInline(embed, "Accuracy", fmt.Sprintf("%.2f%%", acc))
//	return nil
//}
//
//func (bot *discord) onStatsGlobal(ctx context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate, response *botResponse) error {
//	var stats model.GlobalStats
//	errStats := bot.database.GetGlobalStats(ctx, &stats)
//	if errStats != nil {
//		return errCommandFailed
//	}
//	acc := 0.0
//	if stats.Hits > 0 && stats.Shots > 0 {
//		acc = float64(stats.Hits) / float64(stats.Shots) * 100
//	}
//	embed := respOk(response, "Global stats")
//	addFieldInline(embed, "Kills", fmt.Sprintf("%d", stats.Kills))
//	addFieldInline(embed, "Assists", fmt.Sprintf("%d", stats.Assists))
//	addFieldInline(embed, "Damage", fmt.Sprintf("%d", stats.Damage))
//	addFieldInline(embed, "Healing", fmt.Sprintf("%d", stats.Healing))
//	addFieldInline(embed, "Shots", fmt.Sprintf("%d", stats.Shots))
//	addFieldInline(embed, "Hits", fmt.Sprintf("%d", stats.Hits))
//	addFieldInline(embed, "Accuracy", fmt.Sprintf("%.2f%%", acc))
//	return nil
//}

func (bot *Discord) onLog(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *botResponse) error {
	opts := optionMap(interaction.ApplicationCommandData().Options)
	matchId := opts[OptMatchId].IntValue()
	if matchId <= 0 {
		return errCommandFailed
	}
	match, errMatch := bot.app.store.MatchGetById(ctx, int(matchId))
	if errMatch != nil {
		return errCommandFailed
	}
	var server model.Server
	if errServer := bot.app.store.GetServer(ctx, match.ServerId, &server); errServer != nil {
		return errCommandFailed
	}
	embed := respOk(response, fmt.Sprintf("%s - %s", server.ServerNameShort, match.MapName))
	embed.Color = int(green)
	embed.URL = config.ExtURL("/match/%d", match.MatchID)

	redScore := 0
	bluScore := 0
	for _, round := range match.Rounds {
		redScore += round.Score.Red
		bluScore += round.Score.Blu
	}
	top := match.TopPlayers()
	addFieldInline(embed, bot.logger, "Red Score", fmt.Sprintf("%d", redScore))
	addFieldInline(embed, bot.logger, "Blu Score", fmt.Sprintf("%d", bluScore))
	addFieldInline(embed, bot.logger, "Players", fmt.Sprintf("%d", len(top)))
	found := 0
	for _, ts := range match.TeamSums {
		addFieldInline(embed, bot.logger, fmt.Sprintf("%s Kills", ts.Team.String()), fmt.Sprintf("%d", ts.Kills))
		addFieldInline(embed, bot.logger, fmt.Sprintf("%s Damage", ts.Team.String()), fmt.Sprintf("%d", ts.Damage))
		addFieldInline(embed, bot.logger, fmt.Sprintf("%s Ubers", ts.Team.String()), fmt.Sprintf("%d", ts.Caps))
		found++
	}
	desc := "`Top players\n" +
		"N. K:D dmg heal sid\n"
	for i, player := range top {
		desc += fmt.Sprintf("%d %d:%d %d %d %s\n", i+1, player.Kills, player.Deaths, player.Damage, player.Healing, player.SteamId.String())
		if i == 9 {
			break
		}
	}
	desc += "`"
	embed.Description = desc
	return nil
}
