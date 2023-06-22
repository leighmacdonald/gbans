package app

import (
	"context"
	gerrors "errors"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/query"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func registerDiscordHandlers() error {
	cmdMap := map[discord.Cmd]discord.CommandHandler{
		discord.CmdBan:     onBan,
		discord.CmdCheck:   onCheck,
		discord.CmdCSay:    onCSay,
		discord.CmdFilter:  onFilter,
		discord.CmdFind:    onFind,
		discord.CmdHistory: onHistory,
		discord.CmdKick:    onKick,
		discord.CmdLog:     onLog,
		discord.CmdMute:    onMute,
		// discord.CmdCheckIp: onCheckIp,
		discord.CmdPlayers:  onPlayers,
		discord.CmdPSay:     onPSay,
		discord.CmdSay:      onSay,
		discord.CmdServers:  onServers,
		discord.CmdSetSteam: onSetSteam,
		discord.CmdUnban:    onUnban,
		// discord.CmdStats: onServers,
		// discord.CmdStatsGlobal: onServers,
		// discord.CmdStatsPlayer: onServers,
		// discord.CmdStatsServer: ,

	}
	for k, v := range cmdMap {
		if errRegister := discord.RegisterHandler(k, v); errRegister != nil {
			return errRegister
		}
	}
	return nil
}

func init() {
	if errRegister := registerDiscordHandlers(); errRegister != nil {
		panic(errRegister.Error())
	}
}

func onBan(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discord.Response,
) error {
	name := interaction.ApplicationCommandData().Options[0].Name
	switch name {
	case "steam":
		return onBanSteam(ctx, session, interaction, response)
	case "ip":
		return onBanIP(ctx, session, interaction, response)
	case "asn":
		return onBanASN(ctx, session, interaction, response)
	default:
		logger.Error("Invalid ban type selected", zap.String("type", name))
		return discord.ErrCommandFailed
	}
}

func onUnban(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discord.Response,
) error {
	switch interaction.ApplicationCommandData().Options[0].Name {
	case "steam":
		return onUnbanSteam(ctx, session, interaction, response)
	case "ip":
		return discord.ErrCommandFailed
		// return bot.onUnbanIP(ctx, session, interaction, response)
	case "asn":
		return onUnbanASN(ctx, session, interaction, response)
	default:
		return discord.ErrCommandFailed
	}
}

func onFilter(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discord.Response,
) error {
	switch interaction.ApplicationCommandData().Options[0].Name {
	case "add":
		return onFilterAdd(ctx, session, interaction, response)
	case "del":
		return onFilterDel(ctx, session, interaction, response)
	case "check":
		return onFilterCheck(ctx, session, interaction, response)
	default:
		return discord.ErrCommandFailed
	}
}

func onCheck(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discord.Response,
) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options)
	sid, errResolveSID := query.ResolveSID(ctx, opts[discord.OptUserIdentifier].StringValue())
	if errResolveSID != nil {
		return consts.ErrInvalidSID
	}
	player := store.NewPerson(sid)
	if errGetPlayer := PersonBySID(ctx, sid, &player); errGetPlayer != nil {
		return discord.ErrCommandFailed
	}
	ban := store.NewBannedPerson()
	if errGetBanBySID := store.GetBanBySteamID(ctx, sid, &ban, false); errGetBanBySID != nil {
		if !errors.Is(errGetBanBySID, store.ErrNoResult) {
			logger.Error("Failed to get ban by steamid", zap.Error(errGetBanBySID))
			return discord.ErrCommandFailed
		}
	}
	q := store.NewBansQueryFilter(sid)
	q.Deleted = true
	// TODO Get count of old bans
	oldBans, errOld := store.GetBansSteam(ctx, q)
	if errOld != nil {
		if !errors.Is(errOld, store.ErrNoResult) {
			logger.Error("Failed to fetch old bans", zap.Error(errOld))
		}
	}

	bannedNets, errGetBanNet := store.GetBanNetByAddress(ctx, player.IPAddr)
	if errGetBanNet != nil {
		if !errors.Is(errGetBanNet, store.ErrNoResult) {
			logger.Error("Failed to get ban nets by addr", zap.Error(errGetBanNet))
			return discord.ErrCommandFailed
		}
	}
	var (
		color         = discord.Green
		banned        = false
		muted         = false
		reason        = ""
		createdAt     = ""
		authorProfile = store.NewPerson(sid)
		author        *discordgo.MessageEmbedAuthor
		embed         = discord.RespOk(response, "")
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
		if ban.Ban.SourceID > 0 {
			if errGetProfile := PersonBySID(ctx, ban.Ban.SourceID, &authorProfile); errGetProfile != nil {
				logger.Error("Failed to load author for ban", zap.Error(errGetProfile))
			} else {
				author = &discordgo.MessageEmbedAuthor{
					URL:     fmt.Sprintf("https://steamcommunity.com/profiles/%d", authorProfile.SteamID),
					Name:    fmt.Sprintf("<@%s>", authorProfile.DiscordID),
					IconURL: authorProfile.Avatar,
				}
			}
		}
		discord.AddLink(embed, ban.Ban)
	}
	banStateStr := "no"
	if banned {
		// #992D22 red
		color = discord.Red
		banStateStr = "banned"
	}
	if muted {
		// #E67E22 orange
		color = discord.Orange
		banStateStr = "muted"
	}
	discord.AddFieldInline(embed, "Ban/Muted", banStateStr)
	// TODO move elsewhere
	logData, errLogs := thirdparty.LogsTFOverview(ctx, sid)
	if errLogs != nil {
		logger.Warn("Failed to fetch logTF data", zap.Error(errLogs))
	}
	if len(bannedNets) > 0 {
		// ip = bannedNets[0].CIDR.String()
		reason = fmt.Sprintf("Banned from %d networks", len(bannedNets))
		expiry = bannedNets[0].ValidUntil
		discord.AddFieldInline(embed, "Network Bans", fmt.Sprintf("%d", len(bannedNets)))
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
			if errASN := store.GetASNRecordByIP(ctx, player.IPAddr, &asn); errASN != nil {
				logger.Error("Failed to fetch ASN record", zap.Error(errASN))
			}
		}
	}()
	go func() {
		defer waitGroup.Done()
		if player.IPAddr != nil {
			if errLoc := store.GetLocationRecord(ctx, player.IPAddr, &location); errLoc != nil {
				logger.Error("Failed to fetch Location record", zap.Error(errLoc))
			}
		}
	}()
	go func() {
		defer waitGroup.Done()
		if player.IPAddr != nil {
			if errProxy := store.GetProxyRecord(ctx, player.IPAddr, &proxy); errProxy != nil && !errors.Is(errProxy, store.ErrNoResult) {
				logger.Error("Failed to fetch proxy record", zap.Error(errProxy))
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
		discord.AddFieldInline(embed, "Real Name", player.RealName)
	}
	cd := time.Unix(int64(player.TimeCreated), 0)
	discord.AddFieldInline(embed, "Age", config.FmtDuration(cd))
	discord.AddFieldInline(embed, "Private", fmt.Sprintf("%v", player.CommunityVisibilityState == 1))
	discord.AddFieldsSteamID(embed, player.SteamID)
	if player.VACBans > 0 {
		discord.AddFieldInline(embed, "VAC Bans", fmt.Sprintf("count: %d days: %d", player.VACBans, player.DaysSinceLastBan))
	}
	if player.GameBans > 0 {
		discord.AddFieldInline(embed, "Game Bans", fmt.Sprintf("count: %d", player.GameBans))
	}
	if player.CommunityBanned {
		discord.AddFieldInline(embed, "Com. Ban", "true")
	}
	if player.EconomyBan != "" {
		discord.AddFieldInline(embed, "Econ Ban", string(player.EconomyBan))
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
		discord.AddFieldInline(embed, "Total Mutes", fmt.Sprintf("%d", numMutes))
		discord.AddFieldInline(embed, "Total Bans", fmt.Sprintf("%d", numBans))
	}
	if ban.Ban.BanID > 0 {
		discord.AddFieldInline(embed, "Reason", reason)
		discord.AddFieldInline(embed, "Created", config.FmtTimeShort(ban.Ban.CreatedOn))
		if time.Until(expiry) > time.Hour*24*365*5 {
			discord.AddFieldInline(embed, "Expires", "Permanent")
		} else {
			discord.AddFieldInline(embed, "Expires", config.FmtDuration(expiry))
		}
		discord.AddFieldInline(embed, "Author", fmt.Sprintf("<@%s>", authorProfile.DiscordID))
		if ban.Ban.Note != "" {
			discord.AddField(embed, "Mod Note", ban.Ban.Note)
		}
	}
	if player.IPAddr != nil {
		discord.AddFieldInline(embed, "Last IP", player.IPAddr.String())
	}
	if asn.ASName != "" {
		discord.AddFieldInline(embed, "ASN", fmt.Sprintf("(%d) %s", asn.ASNum, asn.ASName))
	}
	if location.CountryCode != "" {
		discord.AddFieldInline(embed, "City", location.CityName)
	}
	if location.CountryName != "" {
		discord.AddFieldInline(embed, "Country", location.CountryName)
	}
	if proxy.CountryCode != "" {
		discord.AddFieldInline(embed, "Proxy Type", string(proxy.ProxyType))
		discord.AddFieldInline(embed, "Proxy", string(proxy.Threat))
	}
	if logData != nil && logData.Total > 0 {
		discord.AddFieldInline(embed, "Logs.tf", fmt.Sprintf("%d", logData.Total))
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

func onHistory(ctx context.Context, session *discordgo.Session,
	interaction *discordgo.InteractionCreate, response *discord.Response,
) error {
	switch interaction.ApplicationCommandData().Name {
	case string(discord.CmdHistoryIP):
		return onHistoryIP(ctx, session, interaction, response)
	default:
		return discord.ErrCommandFailed
		// return bot.onHistoryChat(ctx, session, interaction, response)
	}
}

func onHistoryIP(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discord.Response,
) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	steamID, errResolve := query.ResolveSID(ctx, opts[discord.OptUserIdentifier].StringValue())
	if errResolve != nil {
		return consts.ErrInvalidSID
	}
	person := store.NewPerson(steamID)
	if errPersonBySID := PersonBySID(ctx, steamID, &person); errPersonBySID != nil {
		return discord.ErrCommandFailed
	}
	ipRecords, errGetPersonIPHist := store.GetPersonIPHistory(ctx, steamID, 20)
	if errGetPersonIPHist != nil && !errors.Is(errGetPersonIPHist, store.ErrNoResult) {
		return discord.ErrCommandFailed
	}
	embed := discord.RespOk(response, fmt.Sprintf("IP History of: %s", person.PersonaName))
	lastIP := net.IP{}
	for _, ipRecord := range ipRecords {
		if ipRecord.IPAddr.Equal(lastIP) {
			continue
		}
		// TODO Join query for connections and geoip lookup data
		// addField(embed, ipRecord.IPAddr.String(), fmt.Sprintf("%s %s %s %s %s %s %s %s", config.FmtTimeShort(ipRecord.CreatedOn), ipRecord.CountryCode,
		//	ipRecord.CityName, ipRecord.ASName, ipRecord.ISP, ipRecord.UsageType, ipRecord.Threat, ipRecord.DomainUsed))
		// lastIP = ipRecord.IPAddr
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

func createDiscordBanEmbed(ban store.BanSteam, response *discord.Response) *discordgo.MessageEmbed {
	embed := discord.RespOk(response, "User Banned")
	embed.Title = fmt.Sprintf("Ban created successfully (#%d)", ban.BanID)
	embed.Description = ban.Note
	if ban.ReasonText != "" {
		discord.AddField(embed, "Reason", ban.ReasonText)
	}
	discord.AddFieldsSteamID(embed, ban.TargetID)
	if ban.ValidUntil.Year()-config.Now().Year() > 5 {
		discord.AddField(embed, "Expires In", "Permanent")
		discord.AddField(embed, "Expires At", "Permanent")
	} else {
		discord.AddField(embed, "Expires In", config.FmtDuration(ban.ValidUntil))
		discord.AddField(embed, "Expires At", config.FmtTimeShort(ban.ValidUntil))
	}
	return embed
}

func onSetSteam(ctx context.Context, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate, response *discord.Response,
) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options)
	steamID, errResolveSID := query.ResolveSID(ctx, opts[discord.OptUserIdentifier].StringValue())
	if errResolveSID != nil {
		return consts.ErrInvalidSID
	}
	errSetSteam := SetSteam(ctx, steamID, interaction.Member.User.ID)
	if errSetSteam != nil {
		return errSetSteam
	}
	embed := discord.RespOk(response, "Steam Account Linked")
	embed.Description = "Your steam and discord accounts are now linked"
	return nil
}

func onUnbanSteam(ctx context.Context, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate, response *discord.Response,
) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	reason := opts[discord.OptUnbanReason].StringValue()
	steamID, errResolveSID := query.ResolveSID(ctx, opts[discord.OptUserIdentifier].StringValue())
	if errResolveSID != nil {
		return consts.ErrInvalidSID
	}
	found, errUnban := Unban(ctx, steamID, reason)
	if errUnban != nil {
		return errUnban
	}
	if !found {
		return errors.New("No ban found")
	}
	embed := discord.RespOk(response, "User Unbanned Successfully")
	discord.AddFieldsSteamID(embed, steamID)
	return nil
}

func onUnbanASN(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discord.Response,
) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	asNumStr := opts[discord.OptASN].StringValue()
	banExisted, errUnbanASN := UnbanASN(ctx, asNumStr)
	if errUnbanASN != nil {
		if errors.Is(errUnbanASN, store.ErrNoResult) {
			return errors.New("Ban for ASN does not exist")
		}
		return discord.ErrCommandFailed
	}
	if !banExisted {
		return errors.New("Ban for ASN does not exist")
	}
	asNum, errConv := strconv.ParseInt(asNumStr, 10, 64)
	if errConv != nil {
		return errors.New("Invalid ASN")
	}
	asnNetworks, errGetASNRecords := store.GetASNRecordsByNum(ctx, asNum)
	if errGetASNRecords != nil {
		if errors.Is(errGetASNRecords, store.ErrNoResult) {
			return errors.New("No asnNetworks found matching ASN")
		}
		return errors.New("Error fetching asn asnNetworks")
	}
	embed := discord.RespOk(response, "ASN Networks Unbanned Successfully")
	discord.AddField(embed, "ASN", asNumStr)
	discord.AddField(embed, "Hosts", fmt.Sprintf("%d", asnNetworks.Hosts()))
	return nil
}

func getDiscordAuthor(ctx context.Context, interaction *discordgo.InteractionCreate) (store.Person, error) {
	author := store.NewPerson(0)
	if errPersonByDiscordID := store.GetPersonByDiscordID(ctx, interaction.Interaction.Member.User.ID, &author); errPersonByDiscordID != nil {
		if errors.Is(errPersonByDiscordID, store.ErrNoResult) {
			return author, errors.New("Must set steam id. See /set_steam")
		}
		return author, errors.New("Error fetching author info")
	}
	return author, nil
}

func onKick(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, response *discord.Response) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options)
	target := store.StringSID(opts[discord.OptUserIdentifier].StringValue())
	reason := store.Reason(opts[discord.OptBanReason].IntValue())
	targetSid64, errTarget := target.SID64()
	if errTarget != nil {
		return consts.ErrInvalidSID
	}
	author, errAuthor := getDiscordAuthor(ctx, interaction)
	if errAuthor != nil {
		return errAuthor
	}
	players, found := state.Find(state.FindOpts{SteamID: targetSid64})
	if !found {
		return nil
	}
	var err error
	for _, player := range players {
		if errKick := Kick(ctx, store.Bot, player.Player.SID, author.SteamID, reason); errKick != nil {
			err = gerrors.Join(err, errKick)
			continue
		}
		embed := discord.RespOk(response, "User Kicked")
		discord.AddFieldsSteamID(embed, targetSid64)
		discord.AddField(embed, "NameShort", player.Player.Name)
	}
	return err
}

func onSay(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discord.Response,
) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options)
	server := opts[discord.OptServerIdentifier].StringValue()
	msg := opts[discord.OptMessage].StringValue()
	if errSay := Say(ctx, 0, server, msg); errSay != nil {
		return discord.ErrCommandFailed
	}
	embed := discord.RespOk(response, "Sent center message successfully")
	discord.AddField(embed, "Server", server)
	discord.AddField(embed, "Message", msg)
	return nil
}

func onCSay(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discord.Response,
) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options)
	server := opts[discord.OptServerIdentifier].StringValue()
	msg := opts[discord.OptMessage].StringValue()
	if errCSay := CSay(ctx, 0, server, msg); errCSay != nil {
		return discord.ErrCommandFailed
	}
	embed := discord.RespOk(response, "Sent console message successfully")
	discord.AddField(embed, "Server", server)
	discord.AddField(embed, "Message", msg)
	return nil
}

func onPSay(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discord.Response,
) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options)
	player := store.StringSID(opts[discord.OptUserIdentifier].StringValue())
	msg := opts[discord.OptMessage].StringValue()
	playerSid, errPlayerSid := player.SID64()
	if errPlayerSid != nil {
		return errPlayerSid
	}
	author, errAuthor := getDiscordAuthor(ctx, interaction)
	if errAuthor != nil {
		return errAuthor
	}
	if errPSay := PSay(ctx, author.SteamID, playerSid, msg); errPSay != nil {
		return discord.ErrCommandFailed
	}
	embed := discord.RespOk(response, "Sent private message successfully")
	discord.AddField(embed, "Player", string(player))
	discord.AddField(embed, "Message", msg)
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

func onServers(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate,
	response *discord.Response,
) error {
	currentState := state.State().ByRegion()
	stats := map[string]float64{}
	used, total := 0, 0
	embed := discord.RespOk(response, "Current Server Populations")
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
			discord.AddField(embed, mapRegion(region), fmt.Sprintf("```%s```", msg))
		}
	}
	for statName := range stats {
		if strings.HasSuffix(statName, "total") {
			continue
		}
		discord.AddField(embed, mapRegion(statName), fmt.Sprintf("%.2f%%", (stats[statName]/stats[statName+"total"])*100))
	}
	discord.AddField(embed, "Global", fmt.Sprintf("%d/%d %.2f%%", used, total, float64(used)/float64(total)*100))
	if total == 0 {
		discord.RespErr(response, "No server currentState available")
	}
	return nil
}

func onPlayers(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discord.Response,
) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options)
	serverName := opts[discord.OptServerIdentifier].StringValue()
	var server store.Server
	if errGetServer := store.GetServerByName(ctx, serverName, &server); errGetServer != nil {
		if errors.Is(errGetServer, store.ErrNoResult) {
			return errors.New("Invalid server name")
		}
		return discord.ErrCommandFailed
	}
	var currentState state.ServerState
	serverStates := state.State()
	if !serverStates.ByName(server.ServerNameShort, &currentState) {
		return consts.ErrUnknownID
	}
	var rows []string
	embed := discord.RespOk(response, fmt.Sprintf("Current Players: %s", server.ServerNameShort))
	if len(currentState.Players) > 0 {
		sort.SliceStable(currentState.Players, func(i, j int) bool {
			return currentState.Players[i].Name < currentState.Players[j].Name
		})
		for _, player := range currentState.Players {
			var asn ip2location.ASNRecord
			if errASN := store.GetASNRecordByIP(ctx, player.IP, &asn); errASN != nil {
				// Will fail for LAN ips
				logger.Warn("Failed to get asn record", zap.Error(errASN))
			}
			var loc ip2location.LocationRecord
			if errLoc := store.GetLocationRecord(ctx, player.IP, &loc); errLoc != nil {
				logger.Warn("Failed to get location record: %v", zap.Error(errLoc))
			}
			proxyStr := ""
			var proxy ip2location.ProxyRecord
			if errGetProxyRecord := store.GetProxyRecord(ctx, player.IP, &proxy); errGetProxyRecord == nil {
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

func onFilterAdd(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discord.Response,
) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	pattern := opts["pattern"].StringValue()
	isRegex := opts["is_regex"].BoolValue()

	if isRegex {
		_, rxErr := regexp.Compile(pattern)
		if rxErr != nil {
			return errors.Errorf("Invalid regular expression: %v", rxErr)
		}
	}
	author, errAuthor := getDiscordAuthor(ctx, interaction)
	if errAuthor != nil {
		return errAuthor
	}

	filter := store.Filter{
		AuthorID:  author.SteamID,
		Pattern:   pattern,
		IsRegex:   isRegex,
		IsEnabled: true,
		CreatedOn: time.Now(),
		UpdatedOn: time.Now(),
	}
	if errFilterAdd := FilterAdd(ctx, &filter); errFilterAdd != nil {
		return discord.ErrCommandFailed
	}
	embed := discord.RespOk(response, "Filter Created Successfully")
	embed.Description = filter.Pattern
	return nil
}

func onFilterDel(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discord.Response,
) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	wordID := opts["filter"].IntValue()
	if wordID <= 0 {
		return errors.New("Invalid filter id")
	}
	var filter store.Filter
	if errGetFilter := store.GetFilterByID(ctx, wordID, &filter); errGetFilter != nil {
		return discord.ErrCommandFailed
	}
	if errDropFilter := store.DropFilter(ctx, &filter); errDropFilter != nil {
		return discord.ErrCommandFailed
	}
	discord.RespOk(response, "Filter Deleted Successfully")
	return nil
}

func onFilterCheck(_ context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discord.Response,
) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	message := opts[discord.OptMessage].StringValue()
	matches := FilterCheck(message)
	var title string
	if len(matches) == 0 {
		title = "No Match Found"
	} else {
		title = "Matched Found"
	}

	discord.RespOk(response, title)

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

func onLog(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discord.Response,
) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options)
	matchID := opts[discord.OptMatchId].IntValue()
	if matchID <= 0 {
		return discord.ErrCommandFailed
	}
	match, errMatch := store.MatchGetByID(ctx, int(matchID))
	if errMatch != nil {
		return discord.ErrCommandFailed
	}
	var server store.Server
	if errServer := store.GetServer(ctx, match.ServerId, &server); errServer != nil {
		return discord.ErrCommandFailed
	}
	embed := discord.RespOk(response, fmt.Sprintf("%s - %s", server.ServerNameShort, match.MapName))
	embed.Color = int(discord.Green)
	embed.URL = config.ExtURL("/match/%d", match.MatchID)

	redScore := 0
	bluScore := 0
	for _, round := range match.Rounds {
		redScore += round.Score.Red
		bluScore += round.Score.Blu
	}
	top := match.TopPlayers()
	discord.AddFieldInline(embed, "Red Score", fmt.Sprintf("%d", redScore))
	discord.AddFieldInline(embed, "Blu Score", fmt.Sprintf("%d", bluScore))
	discord.AddFieldInline(embed, "Players", fmt.Sprintf("%d", len(top)))
	found := 0
	for _, ts := range match.TeamSums {
		discord.AddFieldInline(embed, fmt.Sprintf("%s Kills", ts.Team.String()), fmt.Sprintf("%d", ts.Kills))
		discord.AddFieldInline(embed, fmt.Sprintf("%s Damage", ts.Team.String()), fmt.Sprintf("%d", ts.Damage))
		discord.AddFieldInline(embed, fmt.Sprintf("%s Ubers", ts.Team.String()), fmt.Sprintf("%d", ts.Caps))
		found++
	}
	desc := "`Top players\n" +
		"N. K:D dmg heal sid\n"
	for i, player := range top {
		desc += fmt.Sprintf("%d %d:%d %d %d %s\n", i+1, player.Kills, player.Deaths, player.Damage, player.Healing, player.SteamID.String())
		if i == 9 {
			break
		}
	}
	desc += "`"
	embed.Description = desc
	return nil
}

func onFind(ctx context.Context, _ *discordgo.Session, i *discordgo.InteractionCreate,
	r *discord.Response,
) error {
	opts := discord.OptionMap(i.ApplicationCommandData().Options)
	userIdentifier := store.StringSID(opts[discord.OptUserIdentifier].StringValue())
	sid, errSid := userIdentifier.SID64()
	if errSid != nil {
		return consts.ErrInvalidSID
	}
	playerInfo := state.NewPlayerInfo()
	players, found := state.Find(state.FindOpts{SteamID: sid})
	if !found {
		return consts.ErrUnknownID
	}
	for _, player := range players {
		var server store.Server
		if errServer := store.GetServer(ctx, player.ServerId, &server); errServer != nil {
			return errServer
		}
		person := store.NewPerson(player.Player.SID)
		if errPerson := PersonBySID(ctx, player.Player.SID, &person); errPerson != nil {
			return errPerson
		}
		resp := discord.RespOk(r, "Player Found")
		resp.Type = discordgo.EmbedTypeRich
		resp.Image = &discordgo.MessageEmbedImage{URL: person.AvatarFull}
		resp.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: person.Avatar}
		resp.URL = fmt.Sprintf("https://steamcommunity.com/profiles/%d", playerInfo.Player.SID.Int64())
		resp.Title = playerInfo.Player.Name
		discord.AddFieldInline(resp, "Server", server.ServerNameShort)
		discord.AddFieldsSteamID(resp, playerInfo.Player.SID)
		discord.AddField(resp, "Connect", fmt.Sprintf("steam://connect/%s", server.Addr()))
	}
	return nil
}

func onMute(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	r *discord.Response,
) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options)
	playerID := store.StringSID(opts.String(discord.OptUserIdentifier))
	var reason store.Reason
	reasonValueOpt, ok := opts[discord.OptBanReason]
	if !ok {
		return errors.New("Invalid mute reason")
	}
	reason = store.Reason(reasonValueOpt.IntValue())
	duration := store.Duration(opts[discord.OptDuration].StringValue())
	modNote := opts[discord.OptNote].StringValue()
	author, errAuthor := getDiscordAuthor(ctx, interaction)
	if errAuthor != nil {
		return errAuthor
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
	if errBan := BanSteam(ctx, &banSteam); errBan != nil {
		return errBan
	}
	response := discord.RespOk(r, "Player muted successfully")
	discord.AddFieldsSteamID(response, banSteam.TargetID)
	return nil
}

func onBanASN(ctx context.Context, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate, response *discord.Response,
) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	asNumStr := opts[discord.OptASN].StringValue()
	duration := store.Duration(opts[discord.OptDuration].StringValue())
	reason := store.Reason(opts[discord.OptBanReason].IntValue())
	targetID := store.StringSID(opts[discord.OptUserIdentifier].StringValue())
	modNote := opts[discord.OptNote].StringValue()
	author := store.NewPerson(0)
	if errGetPersonByDiscordID := store.GetPersonByDiscordID(ctx, interaction.Interaction.Member.User.ID, &author); errGetPersonByDiscordID != nil {
		if errGetPersonByDiscordID == store.ErrNoResult {
			return errors.New("Must set steam id. See /set_steam")
		}
		return errors.New("Error fetching author info")
	}
	asNum, errConv := strconv.ParseInt(asNumStr, 10, 64)
	if errConv != nil {
		return errors.New("Invalid ASN")
	}
	asnRecords, errGetASNRecords := store.GetASNRecordsByNum(ctx, asNum)
	if errGetASNRecords != nil {
		if errors.Is(errGetASNRecords, store.ErrNoResult) {
			return errors.New("No asnRecords found matching ASN")
		}
		return errors.New("Error fetching asn asnRecords")
	}
	var banASN store.BanASN
	if errOpts := store.NewBanASN(
		store.StringSID(author.SteamID.String()),
		targetID,
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
	if errBanASN := BanASN(ctx, &banASN); errBanASN != nil {
		if errors.Is(errBanASN, store.ErrDuplicate) {
			return errors.New("Duplicate ASN ban")
		}
		return discord.ErrCommandFailed
	}
	resp := discord.RespOk(response, "ASN BanSteam Created Successfully")
	discord.AddField(resp, "ASNum", asNumStr)
	discord.AddField(resp, "Total IPs Blocked", fmt.Sprintf("%d", asnRecords.Hosts()))
	return nil
}

func onBanIP(ctx context.Context, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate, response *discord.Response,
) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	target := store.StringSID(opts[discord.OptUserIdentifier].StringValue())
	reason := store.Reason(opts[discord.OptBanReason].IntValue())
	cidr := opts[discord.OptCIDR].StringValue()

	_, network, errParseCIDR := net.ParseCIDR(cidr)
	if errParseCIDR != nil {
		return errors.Wrap(errParseCIDR, "Invalid CIDR")
	}

	duration := store.Duration(opts[discord.OptDuration].StringValue())
	modNote := opts[discord.OptNote].StringValue()
	author := store.NewPerson(0)
	if errGetPerson := store.GetPersonByDiscordID(ctx, interaction.Interaction.Member.User.ID, &author); errGetPerson != nil {
		if errors.Is(errGetPerson, store.ErrNoResult) {
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
	if errBanNet := BanCIDR(ctx, &banCIDR); errBanNet != nil {
		return errBanNet
	}
	players, found := state.Find(state.FindOpts{CIDR: network})
	if !found {
		return nil
	}
	for _, player := range players {
		if errKick := Kick(ctx, store.Bot, player.Player.SID, author.SteamID, reason); errKick != nil {
			logger.Error("Failed to perform kick", zap.Error(errKick))
		}
	}
	discord.RespOk(response, "IP ban created successfully")
	return nil
}

// onBanSteam !ban <id> <duration> [reason]
func onBanSteam(ctx context.Context, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate, response *discord.Response,
) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	target := opts[discord.OptUserIdentifier].StringValue()
	reason := store.Reason(opts[discord.OptBanReason].IntValue())
	modNote := opts[discord.OptNote].StringValue()
	duration := store.Duration(opts[discord.OptDuration].StringValue())
	author, errAuthor := getDiscordAuthor(ctx, interaction)
	if errAuthor != nil {
		return errAuthor
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
	if errBan := BanSteam(ctx, &banSteam); errBan != nil {
		if errors.Is(errBan, store.ErrDuplicate) {
			return errors.New("Duplicate ban")
		}
		logger.Error("Failed to execute ban", zap.Error(errBan))
		return discord.ErrCommandFailed
	}
	createDiscordBanEmbed(banSteam, response)
	return nil
}
