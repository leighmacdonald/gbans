package discord

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/query"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/discordutil"
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
	r *discordutil.Response) error {
	opts := optionMap(i.ApplicationCommandData().Options)
	userIdentifier := store.StringSID(opts[OptUserIdentifier].StringValue())
	playerInfo := state.NewPlayerInfo()
	if errFind := bot.app.Find(ctx, userIdentifier, "", &playerInfo); errFind != nil {
		return errCommandFailed
	}
	if !playerInfo.Valid || !playerInfo.InGame {
		return consts.ErrUnknownID
	}
	resp := discordutil.RespOk(r, "Player Found")
	person := store.NewPerson(playerInfo.SteamID)
	if errGetProfile := bot.app.PersonBySID(ctx, playerInfo.SteamID, &person); errGetProfile != nil {
		return errors.Wrapf(errGetProfile, "Failed to get profile: %d", playerInfo.SteamID)
	}
	resp.Type = discordgo.EmbedTypeRich
	resp.Image = &discordgo.MessageEmbedImage{URL: person.AvatarFull}
	resp.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: person.Avatar}
	resp.URL = fmt.Sprintf("https://steamcommunity.com/profiles/%d", playerInfo.Player.SID.Int64())
	resp.Title = playerInfo.Player.Name
	discordutil.AddFieldInline(resp, bot.logger, "Server", playerInfo.Server.ServerNameShort)
	discordutil.AddFieldsSteamID(resp, bot.logger, playerInfo.Player.SID)
	discordutil.AddField(resp, bot.logger, "Connect", fmt.Sprintf("steam://connect/%s:%d", playerInfo.Server.Address, playerInfo.Server.Port))
	return nil
}

func (bot *Discord) onMute(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	r *discordutil.Response) error {
	opts := optionMap(interaction.ApplicationCommandData().Options)
	playerID := store.StringSID(opts.String(OptUserIdentifier))
	var reason store.Reason
	reasonValueOpt, ok := opts[OptBanReason]
	if !ok {
		return errors.New("Invalid mute reason")
	}
	reason = store.Reason(reasonValueOpt.IntValue())
	duration := store.Duration(opts[OptDuration].StringValue())
	modNote := opts[OptNote].StringValue()
	author := store.NewPerson(0)
	if errGetAuthor := bot.app.Store().GetPersonByDiscordID(ctx, interaction.Interaction.Member.User.ID, &author); errGetAuthor != nil {
		if errGetAuthor == store.ErrNoResult {
			return errors.New("Must set steam id. See /set_steam")
		}
		return errors.New("Error fetching author info")
	}
	var banSteam store.BanSteam
	if errOpts := store.NewBanSteam(
		store.StringSID(author.SteamID.String()),
		playerID,
		duration,
		reason,
		reason.String(),
		modNote,
		store.Bot,
		0,
		store.NoComm,
		&banSteam,
	); errOpts != nil {
		return errors.Wrapf(errOpts, "Failed to parse options")
	}
	if errBan := bot.app.BanSteam(ctx, &banSteam); errBan != nil {
		return errBan
	}
	response := discordutil.RespOk(r, "Player muted successfully")
	discordutil.AddFieldsSteamID(response, bot.logger, banSteam.TargetId)
	return nil
}

func (bot *Discord) onBanASN(ctx context.Context, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate, response *discordutil.Response) error {
	opts := optionMap(interaction.ApplicationCommandData().Options[0].Options)
	asNumStr := opts[OptASN].StringValue()
	duration := store.Duration(opts[OptDuration].StringValue())
	reason := store.Reason(opts[OptBanReason].IntValue())
	targetId := store.StringSID(opts[OptUserIdentifier].StringValue())
	modNote := opts[OptNote].StringValue()
	author := store.NewPerson(0)
	if errGetPersonByDiscordId := bot.app.Store().GetPersonByDiscordID(ctx, interaction.Interaction.Member.User.ID, &author); errGetPersonByDiscordId != nil {
		if errGetPersonByDiscordId == store.ErrNoResult {
			return errors.New("Must set steam id. See /set_steam")
		}
		return errors.New("Error fetching author info")
	}
	asNum, errConv := strconv.ParseInt(asNumStr, 10, 64)
	if errConv != nil {
		return errors.New("Invalid ASN")
	}
	asnRecords, errGetASNRecords := bot.app.Store().GetASNRecordsByNum(ctx, asNum)
	if errGetASNRecords != nil {
		if errGetASNRecords == store.ErrNoResult {
			return errors.New("No asnRecords found matching ASN")
		}
		return errors.New("Error fetching asn asnRecords")
	}
	var banASN store.BanASN
	if errOpts := store.NewBanASN(
		store.StringSID(author.SteamID.String()),
		targetId,
		duration,
		reason,
		reason.String(),
		modNote,
		store.Bot,
		asNum,
		store.Banned,
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
	resp := discordutil.RespOk(response, "ASN BanSteam Created Successfully")
	discordutil.AddField(resp, bot.logger, "ASNum", asNumStr)
	discordutil.AddField(resp, bot.logger, "Total IPs Blocked", fmt.Sprintf("%d", asnRecords.Hosts()))
	return nil
}

func (bot *Discord) onBanIP(ctx context.Context, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate, response *discordutil.Response) error {
	opts := optionMap(interaction.ApplicationCommandData().Options[0].Options)
	target := store.StringSID(opts[OptUserIdentifier].StringValue())
	reason := store.Reason(opts[OptBanReason].IntValue())
	cidr := opts[OptCIDR].StringValue()
	duration := store.Duration(opts[OptDuration].StringValue())
	modNote := opts[OptNote].StringValue()
	author := store.NewPerson(0)
	if errGetPerson := bot.app.Store().GetPersonByDiscordID(ctx, interaction.Interaction.Member.User.ID, &author); errGetPerson != nil {
		if errGetPerson == store.ErrNoResult {
			return errors.New("Must set steam id. See /set_steam")
		}
		return errors.New("Error fetching author info")
	}

	var banCIDR store.BanCIDR
	if errOpts := store.NewBanCIDR(
		store.StringSID(author.SteamID.String()),
		target,
		duration,
		reason,
		reason.String(),
		modNote,
		store.Bot,
		cidr,
		store.Banned,
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
		var playerInfo state.PlayerInfo
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
	discordutil.RespOk(response, "IP ban created successfully")
	return nil
}

// onBanSteam !ban <id> <duration> [reason]
func (bot *Discord) onBanSteam(ctx context.Context, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate, response *discordutil.Response) error {
	opts := optionMap(interaction.ApplicationCommandData().Options[0].Options)
	target := opts[OptUserIdentifier].StringValue()
	reason := store.Reason(opts[OptBanReason].IntValue())
	modNote := opts[OptNote].StringValue()
	duration := store.Duration(opts[OptDuration].StringValue())
	author := store.NewPerson(0)
	if errGetAuthor := bot.app.Store().GetPersonByDiscordID(ctx, interaction.Interaction.Member.User.ID, &author); errGetAuthor != nil {
		if errGetAuthor == store.ErrNoResult {
			return errors.New("Must set steam id. See /set_steam")
		}
		return errors.New("Error fetching author info")
	}
	var banSteam store.BanSteam
	if errOpts := store.NewBanSteam(
		store.StringSID(author.SteamID.String()),
		store.StringSID(target),
		duration,
		reason,
		reason.String(),
		modNote,
		store.Bot,
		0,
		store.Banned,
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

func createDiscordBanEmbed(ban store.BanSteam, logger *zap.Logger, response *discordutil.Response) *discordgo.MessageEmbed {
	embed := discordutil.RespOk(response, "User Banned")
	embed.Title = fmt.Sprintf("Ban created successfully (#%d)", ban.BanID)
	embed.Description = ban.Note
	if ban.ReasonText != "" {
		discordutil.AddField(embed, logger, "Reason", ban.ReasonText)
	}
	discordutil.AddFieldsSteamID(embed, logger, ban.TargetId)
	if ban.ValidUntil.Year()-config.Now().Year() > 5 {
		discordutil.AddField(embed, logger, "Expires In", "Permanent")
		discordutil.AddField(embed, logger, "Expires At", "Permanent")
	} else {
		discordutil.AddField(embed, logger, "Expires In", config.FmtDuration(ban.ValidUntil))
		discordutil.AddField(embed, logger, "Expires At", config.FmtTimeShort(ban.ValidUntil))
	}
	return embed
}

func (bot *Discord) onCheck(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discordutil.Response) error {
	opts := optionMap(interaction.ApplicationCommandData().Options)
	sid, errResolveSID := query.ResolveSID(ctx, opts[OptUserIdentifier].StringValue())
	if errResolveSID != nil {
		return consts.ErrInvalidSID
	}
	player := store.NewPerson(sid)
	if errGetPlayer := bot.app.PersonBySID(ctx, sid, &player); errGetPlayer != nil {
		return errCommandFailed
	}
	ban := store.NewBannedPerson()
	if errGetBanBySID := bot.app.Store().GetBanBySteamID(ctx, sid, &ban, false); errGetBanBySID != nil {
		if !errors.Is(errGetBanBySID, store.ErrNoResult) {
			bot.logger.Error("Failed to get ban by steamid", zap.Error(errGetBanBySID))
			return errCommandFailed
		}
	}
	q := store.NewBansQueryFilter(sid)
	q.Deleted = true
	// TODO Get count of old bans
	oldBans, errOld := bot.app.Store().GetBansSteam(ctx, q)
	if errOld != nil {
		if !errors.Is(errOld, store.ErrNoResult) {
			bot.logger.Error("Failed to fetch old bans", zap.Error(errOld))
		}
	}

	bannedNets, errGetBanNet := bot.app.Store().GetBanNetByAddress(ctx, player.IPAddr)
	if errGetBanNet != nil {
		if !errors.Is(errGetBanNet, store.ErrNoResult) {
			bot.logger.Error("Failed to get ban nets by addr", zap.Error(errGetBanNet))
			return errCommandFailed
		}
	}
	var (
		color         = discordutil.Green
		banned        = false
		muted         = false
		reason        = ""
		createdAt     = ""
		authorProfile = store.NewPerson(sid)
		author        *discordgo.MessageEmbedAuthor
		embed         = discordutil.RespOk(response, "")
	)
	var expiry time.Time
	// TODO Show the longest remaining ban.
	if ban.Ban.BanID > 0 {
		banned = ban.Ban.BanType == store.Banned
		muted = ban.Ban.BanType == store.NoComm
		reason = ban.Ban.ReasonText
		if len(reason) == 0 {
			// in case authorProfile ban without authorProfile reason ever makes its way here, we make sure
			// that Discord doesn't shit itself
			reason = "none"
		}
		expiry = ban.Ban.ValidUntil
		createdAt = ban.Ban.CreatedOn.Format(time.RFC3339)
		if ban.Ban.SourceId > 0 {
			if errGetProfile := bot.app.PersonBySID(ctx, ban.Ban.SourceId, &authorProfile); errGetProfile != nil {
				bot.logger.Error("Failed to load author for ban", zap.Error(errGetProfile))
			} else {
				author = &discordgo.MessageEmbedAuthor{
					URL:     fmt.Sprintf("https://steamcommunity.com/profiles/%d", authorProfile.SteamID),
					Name:    fmt.Sprintf("<@%s>", authorProfile.DiscordID),
					IconURL: authorProfile.Avatar,
				}
			}
		}
		discordutil.AddLink(embed, bot.logger, ban.Ban)
	}
	banStateStr := "no"
	if banned {
		// #992D22 red
		color = discordutil.Red
		banStateStr = "banned"
	}
	if muted {
		// #E67E22 orange
		color = discordutil.Orange
		banStateStr = "muted"
	}
	discordutil.AddFieldInline(embed, bot.logger, "Ban/Muted", banStateStr)
	// TODO move elsewhere
	logData, errLogs := thirdparty.LogsTFOverview(ctx, sid)
	if errLogs != nil {
		bot.logger.Warn("Failed to fetch logTF data", zap.Error(errLogs))
	}
	if len(bannedNets) > 0 {
		//ip = bannedNets[0].CIDR.String()
		reason = fmt.Sprintf("Banned from %d networks", len(bannedNets))
		expiry = bannedNets[0].ValidUntil
		discordutil.AddFieldInline(embed, bot.logger, "Network Bans", fmt.Sprintf("%d", len(bannedNets)))
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
			if errASN := bot.app.Store().GetASNRecordByIP(ctx, player.IPAddr, &asn); errASN != nil {
				bot.logger.Error("Failed to fetch ASN record", zap.Error(errASN))
			}
		}
	}()
	go func() {
		defer waitGroup.Done()
		if player.IPAddr != nil {
			if errLoc := bot.app.Store().GetLocationRecord(ctx, player.IPAddr, &location); errLoc != nil {
				bot.logger.Error("Failed to fetch Location record", zap.Error(errLoc))
			}
		}
	}()
	go func() {
		defer waitGroup.Done()
		if player.IPAddr != nil {
			if errProxy := bot.app.Store().GetProxyRecord(ctx, player.IPAddr, &proxy); errProxy != nil && errProxy != store.ErrNoResult {
				bot.logger.Error("Failed to fetch proxy record", zap.Error(errProxy))
			}
		}
	}()
	waitGroup.Wait()
	title := player.PersonaName
	if ban.Ban.BanID > 0 {
		if ban.Ban.BanType == store.Banned {
			title = fmt.Sprintf("%s (BANNED)", title)
		} else if ban.Ban.BanType == store.NoComm {
			title = fmt.Sprintf("%s (MUTED)", title)
		}
	}
	embed.Title = title
	if player.RealName != "" {
		discordutil.AddFieldInline(embed, bot.logger, "Real Name", player.RealName)
	}
	cd := time.Unix(int64(player.TimeCreated), 0)
	discordutil.AddFieldInline(embed, bot.logger, "Age", config.FmtDuration(cd))
	discordutil.AddFieldInline(embed, bot.logger, "Private", fmt.Sprintf("%v", player.CommunityVisibilityState == 1))
	discordutil.AddFieldsSteamID(embed, bot.logger, player.SteamID)
	if player.VACBans > 0 {
		discordutil.AddFieldInline(embed, bot.logger, "VAC Bans", fmt.Sprintf("count: %d days: %d", player.VACBans, player.DaysSinceLastBan))
	}
	if player.GameBans > 0 {
		discordutil.AddFieldInline(embed, bot.logger, "Game Bans", fmt.Sprintf("count: %d", player.GameBans))
	}
	if player.CommunityBanned {
		discordutil.AddFieldInline(embed, bot.logger, "Com. Ban", "true")
	}
	if player.EconomyBan != "" {
		discordutil.AddFieldInline(embed, bot.logger, "Econ Ban", player.EconomyBan)
	}
	if len(oldBans) > 0 {
		numMutes, numBans := 0, 0
		for _, ob := range oldBans {
			if ob.Ban.BanType == store.Banned {
				numBans++
			} else {
				numMutes++
			}
		}
		discordutil.AddFieldInline(embed, bot.logger, "Total Mutes", fmt.Sprintf("%d", numMutes))
		discordutil.AddFieldInline(embed, bot.logger, "Total Bans", fmt.Sprintf("%d", numBans))
	}
	if ban.Ban.BanID > 0 {
		discordutil.AddFieldInline(embed, bot.logger, "Reason", reason)
		discordutil.AddFieldInline(embed, bot.logger, "Created", config.FmtTimeShort(ban.Ban.CreatedOn))
		if time.Until(expiry) > time.Hour*24*365*5 {
			discordutil.AddFieldInline(embed, bot.logger, "Expires", "Permanent")
		} else {
			discordutil.AddFieldInline(embed, bot.logger, "Expires", config.FmtDuration(expiry))
		}
		discordutil.AddFieldInline(embed, bot.logger, "Author", fmt.Sprintf("<@%s>", authorProfile.DiscordID))
		if ban.Ban.Note != "" {
			discordutil.AddField(embed, bot.logger, "Mod Note", ban.Ban.Note)
		}
	}
	if player.IPAddr != nil {
		discordutil.AddFieldInline(embed, bot.logger, "Last IP", player.IPAddr.String())
	}
	if asn.ASName != "" {
		discordutil.AddFieldInline(embed, bot.logger, "ASN", fmt.Sprintf("(%d) %s", asn.ASNum, asn.ASName))
	}
	if location.CountryCode != "" {
		discordutil.AddFieldInline(embed, bot.logger, "City", location.CityName)
	}
	if location.CountryName != "" {
		discordutil.AddFieldInline(embed, bot.logger, "Country", location.CountryName)
	}
	if proxy.CountryCode != "" {
		discordutil.AddFieldInline(embed, bot.logger, "Proxy Type", string(proxy.ProxyType))
		discordutil.AddFieldInline(embed, bot.logger, "Proxy", string(proxy.Threat))
	}
	if logData != nil && logData.Total > 0 {
		discordutil.AddFieldInline(embed, bot.logger, "Logs.tf", fmt.Sprintf("%d", logData.Total))
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
	interaction *discordgo.InteractionCreate, response *discordutil.Response) error {
	switch interaction.ApplicationCommandData().Name {
	case string(cmdHistoryIP):
		return bot.onHistoryIP(ctx, session, interaction, response)
	default:
		return errCommandFailed
		//return bot.onHistoryChat(ctx, session, interaction, response)
	}
}

func (bot *Discord) onHistoryIP(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discordutil.Response) error {
	opts := optionMap(interaction.ApplicationCommandData().Options[0].Options)
	steamId, errResolve := query.ResolveSID(ctx, opts[OptUserIdentifier].StringValue())
	if errResolve != nil {
		return consts.ErrInvalidSID
	}
	person := store.NewPerson(steamId)
	if errPersonBySID := bot.app.PersonBySID(ctx, steamId, &person); errPersonBySID != nil {
		return errCommandFailed
	}
	ipRecords, errGetPersonIPHist := bot.app.Store().GetPersonIPHistory(ctx, steamId, 20)
	if errGetPersonIPHist != nil && errGetPersonIPHist != store.ErrNoResult {
		return errCommandFailed
	}
	embed := discordutil.RespOk(response, fmt.Sprintf("IP History of: %s", person.PersonaName))
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
//func (bot *Discord) onHistoryChat(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, response *botResponse) error {
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
	interaction *discordgo.InteractionCreate, response *discordutil.Response) error {
	opts := optionMap(interaction.ApplicationCommandData().Options)
	steamId, errResolveSID := query.ResolveSID(ctx, opts[OptUserIdentifier].StringValue())
	if errResolveSID != nil {
		return consts.ErrInvalidSID
	}
	errSetSteam := bot.app.SetSteam(ctx, steamId, interaction.Member.User.ID)
	if errSetSteam != nil {
		return errSetSteam
	}
	embed := discordutil.RespOk(response, "Steam Account Linked")
	embed.Description = "Your steam and discord accounts are now linked"
	return nil
}

func (bot *Discord) onUnbanSteam(ctx context.Context, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate, response *discordutil.Response) error {
	opts := optionMap(interaction.ApplicationCommandData().Options[0].Options)
	reason := opts[OptUnbanReason].StringValue()
	steamId, errResolveSID := query.ResolveSID(ctx, opts[OptUserIdentifier].StringValue())
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
	embed := discordutil.RespOk(response, "User Unbanned Successfully")
	discordutil.AddFieldsSteamID(embed, bot.logger, steamId)
	return nil
}

func (bot *Discord) onUnbanASN(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discordutil.Response) error {
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
	asnNetworks, errGetASNRecords := bot.app.Store().GetASNRecordsByNum(ctx, asNum)
	if errGetASNRecords != nil {
		if errGetASNRecords == store.ErrNoResult {
			return errors.New("No asnNetworks found matching ASN")
		}
		return errors.New("Error fetching asn asnNetworks")
	}
	embed := discordutil.RespOk(response, "ASN Networks Unbanned Successfully")
	discordutil.AddField(embed, bot.logger, "ASN", asNumStr)
	discordutil.AddField(embed, bot.logger, "Hosts", fmt.Sprintf("%d", asnNetworks.Hosts()))
	return nil
}

func (bot *Discord) onKick(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discordutil.Response) error {
	opts := optionMap(interaction.ApplicationCommandData().Options)
	target := store.StringSID(opts[OptUserIdentifier].StringValue())
	reason := store.Reason(opts[OptBanReason].IntValue())
	targetSid64, errTarget := target.SID64()
	if errTarget != nil {
		return consts.ErrInvalidSID
	}
	person := store.NewPerson(targetSid64)
	if errPersonBySID := bot.app.PersonBySID(ctx, targetSid64, &person); errPersonBySID != nil {
		return errCommandFailed
	}
	var playerInfo state.PlayerInfo
	errKick := bot.app.Kick(ctx, store.Bot, target, "", reason, &playerInfo)
	if errKick != nil {
		return errCommandFailed
	}
	if playerInfo.Server != nil && playerInfo.Server.ServerID > 0 {
		embed := discordutil.RespOk(response, "User Kicked")
		discordutil.AddFieldsSteamID(embed, bot.logger, targetSid64)
		discordutil.AddField(embed, bot.logger, "NameShort", playerInfo.Player.Name)
	} else {
		return errors.New("User not found")
	}
	return nil
}

func (bot *Discord) onSay(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discordutil.Response) error {
	opts := optionMap(interaction.ApplicationCommandData().Options)
	server := opts[OptServerIdentifier].StringValue()
	msg := opts[OptMessage].StringValue()
	if errSay := bot.app.Say(ctx, 0, server, msg); errSay != nil {
		return errCommandFailed
	}
	embed := discordutil.RespOk(response, "Sent center message successfully")
	discordutil.AddField(embed, bot.logger, "Server", server)
	discordutil.AddField(embed, bot.logger, "Message", msg)
	return nil
}

func (bot *Discord) onCSay(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discordutil.Response) error {
	opts := optionMap(interaction.ApplicationCommandData().Options)
	server := opts[OptServerIdentifier].StringValue()
	msg := opts[OptMessage].StringValue()
	if errCSay := bot.app.CSay(ctx, 0, server, msg); errCSay != nil {
		return errCommandFailed
	}
	embed := discordutil.RespOk(response, "Sent console message successfully")
	discordutil.AddField(embed, bot.logger, "Server", server)
	discordutil.AddField(embed, bot.logger, "Message", msg)
	return nil
}

func (bot *Discord) onPSay(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discordutil.Response) error {
	opts := optionMap(interaction.ApplicationCommandData().Options)
	player := store.StringSID(opts[OptUserIdentifier].StringValue())
	msg := opts[OptMessage].StringValue()
	if errPSay := bot.app.PSay(ctx, 0, player, msg, nil); errPSay != nil {
		return errCommandFailed
	}
	embed := discordutil.RespOk(response, "Sent private message successfully")
	discordutil.AddField(embed, bot.logger, "Player", string(player))
	discordutil.AddField(embed, bot.logger, "Message", msg)
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
	response *discordutil.Response) error {
	currentState := bot.app.ServerState().ByRegion()
	stats := map[string]float64{}
	used, total := 0, 0
	embed := discordutil.RespOk(response, "Current Server Populations")
	embed.URL = config.ExtURL("/servers")
	var regionNames []string
	for k := range currentState {
		regionNames = append(regionNames, k)
	}
	sort.Strings(regionNames)
	for _, region := range regionNames {
		var counts []string
		for _, st := range currentState[region] {
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
			discordutil.AddField(embed, bot.logger, mapRegion(region), fmt.Sprintf("```%s```", msg))
		}
	}
	for statName := range stats {
		if strings.HasSuffix(statName, "total") {
			continue
		}
		discordutil.AddField(embed, bot.logger, mapRegion(statName), fmt.Sprintf("%.2f%%", (stats[statName]/stats[statName+"total"])*100))
	}
	discordutil.AddField(embed, bot.logger, "Global", fmt.Sprintf("%d/%d %.2f%%", used, total, float64(used)/float64(total)*100))
	if total == 0 {
		discordutil.RespErr(response, "No server currentState available")
	}
	return nil
}

func (bot *Discord) onPlayers(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discordutil.Response) error {
	opts := optionMap(interaction.ApplicationCommandData().Options)
	serverName := opts[OptServerIdentifier].StringValue()
	var server store.Server
	if errGetServer := bot.app.Store().GetServerByName(ctx, serverName, &server); errGetServer != nil {
		if errGetServer == store.ErrNoResult {
			return errors.New("Invalid server name")
		}
		return errCommandFailed
	}
	var currentState state.ServerState
	serverStates := bot.app.ServerState()
	if !serverStates.ByName(server.ServerNameShort, &currentState) {
		return consts.ErrUnknownID
	}
	var rows []string
	embed := discordutil.RespOk(response, fmt.Sprintf("Current Players: %s", server.ServerNameShort))
	if len(currentState.Players) > 0 {
		sort.SliceStable(currentState.Players, func(i, j int) bool {
			return currentState.Players[i].Name < currentState.Players[j].Name
		})
		for _, player := range currentState.Players {
			var asn ip2location.ASNRecord
			if errASN := bot.app.Store().GetASNRecordByIP(ctx, player.IP, &asn); errASN != nil {
				// Will fail for LAN ips
				bot.logger.Warn("Failed to get asn record", zap.Error(errASN))
			}
			var loc ip2location.LocationRecord
			if errLoc := bot.app.Store().GetLocationRecord(ctx, player.IP, &loc); errLoc != nil {
				bot.logger.Warn("Failed to get location record: %v", zap.Error(errLoc))
			}
			proxyStr := ""
			var proxy ip2location.ProxyRecord
			if errGetProxyRecord := bot.app.Store().GetProxyRecord(ctx, player.IP, &proxy); errGetProxyRecord == nil {
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
	response *discordutil.Response) error {
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
	response *discordutil.Response) error {
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
	response *discordutil.Response) error {
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
	response *discordutil.Response) error {
	opts := optionMap(interaction.ApplicationCommandData().Options[0].Options)
	pattern := opts["pattern"].StringValue()
	isRegex := opts["is_regex"].BoolValue()
	author := store.NewPerson(0)
	if errPersonByDiscordID := bot.app.Store().GetPersonByDiscordID(ctx, interaction.Interaction.Member.User.ID, &author); errPersonByDiscordID != nil {
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
	filter := store.Filter{
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
	embed := discordutil.RespOk(response, "Filter Created Successfully")
	embed.Description = filter.Pattern
	return nil
}

func (bot *Discord) onFilterDel(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discordutil.Response) error {
	opts := optionMap(interaction.ApplicationCommandData().Options[0].Options)
	wordId := opts["filter"].IntValue()
	if wordId <= 0 {
		return errors.New("Invalid filter id")
	}
	var filter store.Filter
	if errGetFilter := bot.app.Store().GetFilterByID(ctx, wordId, &filter); errGetFilter != nil {
		return errCommandFailed
	}
	if errDropFilter := bot.app.Store().DropFilter(ctx, &filter); errDropFilter != nil {
		return errCommandFailed
	}
	discordutil.RespOk(response, "Filter Deleted Successfully")
	return nil
}

func (bot *Discord) onFilterCheck(_ context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discordutil.Response) error {
	opts := optionMap(interaction.ApplicationCommandData().Options[0].Options)
	message := opts[OptMessage].StringValue()
	matches := bot.app.FilterCheck(message)
	title := ""
	if len(matches) == 0 {
		title = "No Match Found"
	} else {
		title = "Matched Found"
	}
	discordutil.RespOk(response, title)
	return nil
}

//func (bot *discordutil) onStats(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, response *botResponse) error {
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
//func (bot *discordutil) onStatsPlayer(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, response *botResponse) error {
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
//func (bot *discordutil) onStatsServer(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, response *botResponse) error {
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
//func (bot *discordutil) onStatsGlobal(ctx context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate, response *botResponse) error {
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
	response *discordutil.Response) error {
	opts := optionMap(interaction.ApplicationCommandData().Options)
	matchId := opts[OptMatchId].IntValue()
	if matchId <= 0 {
		return errCommandFailed
	}
	match, errMatch := bot.app.Store().MatchGetById(ctx, int(matchId))
	if errMatch != nil {
		return errCommandFailed
	}
	var server store.Server
	if errServer := bot.app.Store().GetServer(ctx, match.ServerId, &server); errServer != nil {
		return errCommandFailed
	}
	embed := discordutil.RespOk(response, fmt.Sprintf("%s - %s", server.ServerNameShort, match.MapName))
	embed.Color = int(discordutil.Green)
	embed.URL = config.ExtURL("/match/%d", match.MatchID)

	redScore := 0
	bluScore := 0
	for _, round := range match.Rounds {
		redScore += round.Score.Red
		bluScore += round.Score.Blu
	}
	top := match.TopPlayers()
	discordutil.AddFieldInline(embed, bot.logger, "Red Score", fmt.Sprintf("%d", redScore))
	discordutil.AddFieldInline(embed, bot.logger, "Blu Score", fmt.Sprintf("%d", bluScore))
	discordutil.AddFieldInline(embed, bot.logger, "Players", fmt.Sprintf("%d", len(top)))
	found := 0
	for _, ts := range match.TeamSums {
		discordutil.AddFieldInline(embed, bot.logger, fmt.Sprintf("%s Kills", ts.Team.String()), fmt.Sprintf("%d", ts.Kills))
		discordutil.AddFieldInline(embed, bot.logger, fmt.Sprintf("%s Damage", ts.Team.String()), fmt.Sprintf("%d", ts.Damage))
		discordutil.AddFieldInline(embed, bot.logger, fmt.Sprintf("%s Ubers", ts.Team.String()), fmt.Sprintf("%d", ts.Caps))
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
