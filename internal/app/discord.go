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
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (app *App) registerDiscordHandlers() error {
	cmdMap := map[discord.Cmd]discord.CommandHandler{
		discord.CmdBan:     makeOnBan(app),
		discord.CmdCheck:   makeOnCheck(app),
		discord.CmdCSay:    makeOnCSay(app),
		discord.CmdFilter:  makeOnFilter(app),
		discord.CmdFind:    makeOnFind(app),
		discord.CmdHistory: makeOnHistory(app),
		discord.CmdKick:    makeOnKick(app),
		discord.CmdLog:     makeOnLog(app),
		discord.CmdMute:    makeOnMute(app),
		// discord.CmdCheckIP: onCheckIp,
		discord.CmdPlayers:  makeOnPlayers(app),
		discord.CmdPSay:     makeOnPSay(app),
		discord.CmdSay:      makeOnSay(app),
		discord.CmdServers:  makeOnServers(app),
		discord.CmdSetSteam: makeOnSetSteam(app),
		discord.CmdUnban:    makeOnUnban(app),
		// discord.CmdStats: onServers,
		// discord.CmdStatsGlobal: onServers,
		// discord.CmdStatsPlayer: onServers,
		// discord.CmdStatsServer: ,

	}
	for k, v := range cmdMap {
		if errRegister := app.bot.RegisterHandler(k, v); errRegister != nil {
			return errors.Wrap(errRegister, "Failed to register discord command")
		}
	}

	return nil
}

func makeOnBan(app *App) discord.CommandHandler {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate,
		response *discord.Response,
	) error {
		name := interaction.ApplicationCommandData().Options[0].Name
		switch name {
		case "steam":
			return onBanSteam(ctx, app, session, interaction, response)
		case "ip":
			return onBanIP(ctx, app, session, interaction, response)
		case "asn":
			return onBanASN(ctx, app, session, interaction, response)
		default:
			return discord.ErrCommandFailed
		}
	}
}

func makeOnUnban(app *App) discord.CommandHandler {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate,
		response *discord.Response,
	) error {
		switch interaction.ApplicationCommandData().Options[0].Name {
		case "steam":
			return onUnbanSteam(ctx, app, session, interaction, response)
		case "ip":
			return discord.ErrCommandFailed
			// return bot.onUnbanIP(ctx, session, interaction, response)
		case "asn":
			return onUnbanASN(ctx, app, session, interaction, response)
		default:
			return discord.ErrCommandFailed
		}
	}
}

func makeOnFilter(app *App) discord.CommandHandler {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate,
		response *discord.Response,
	) error {
		switch interaction.ApplicationCommandData().Options[0].Name {
		case "add":
			return onFilterAdd(ctx, app, session, interaction, response)
		case "del":
			return onFilterDel(ctx, app, session, interaction, response)
		case "check":
			return onFilterCheck(ctx, app, session, interaction, response)
		default:
			return discord.ErrCommandFailed
		}
	}
}

func makeOnCheck(app *App) discord.CommandHandler { //nolint:maintidx
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, //nolint:maintidx
		response *discord.Response,
	) error {
		opts := discord.OptionMap(interaction.ApplicationCommandData().Options)
		sid, errResolveSID := query.ResolveSID(ctx, opts[discord.OptUserIdentifier].StringValue())

		if errResolveSID != nil {
			return consts.ErrInvalidSID
		}

		player := store.NewPerson(sid)
		if errGetPlayer := app.PersonBySID(ctx, sid, &player); errGetPlayer != nil {
			return discord.ErrCommandFailed
		}

		ban := store.NewBannedPerson()
		if errGetBanBySID := app.db.GetBanBySteamID(ctx, sid, &ban, false); errGetBanBySID != nil {
			if !errors.Is(errGetBanBySID, store.ErrNoResult) {
				app.log.Error("Failed to get ban by steamid", zap.Error(errGetBanBySID))

				return discord.ErrCommandFailed
			}
		}

		q := store.NewBansQueryFilter(sid)
		q.Deleted = true

		// TODO Get count of old bans
		oldBans, errOld := app.db.GetBansSteam(ctx, q)
		if errOld != nil {
			if !errors.Is(errOld, store.ErrNoResult) {
				app.log.Error("Failed to fetch old bans", zap.Error(errOld))
			}
		}

		bannedNets, errGetBanNet := app.db.GetBanNetByAddress(ctx, player.IPAddr)
		if errGetBanNet != nil {
			if !errors.Is(errGetBanNet, store.ErrNoResult) {
				app.log.Error("Failed to get ban nets by addr", zap.Error(errGetBanNet))

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
			expiry        time.Time
		)

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

			if ban.Ban.SourceID.Valid() {
				if errGetProfile := app.PersonBySID(ctx, ban.Ban.SourceID, &authorProfile); errGetProfile != nil {
					app.log.Error("Failed to load author for ban", zap.Error(errGetProfile))
				} else {
					author = &discordgo.MessageEmbedAuthor{
						URL:     fmt.Sprintf("https://steamcommunity.com/profiles/%s", authorProfile.SteamID),
						Name:    fmt.Sprintf("<@%s>", authorProfile.DiscordID),
						IconURL: authorProfile.Avatar,
					}
				}
			}

			discord.AddLink(embed, app.conf, ban.Ban)
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
			app.log.Warn("Failed to fetch logTF data", zap.Error(errLogs))
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
				if errASN := app.db.GetASNRecordByIP(ctx, player.IPAddr, &asn); errASN != nil {
					app.log.Error("Failed to fetch ASN record", zap.Error(errASN))
				}
			}
		}()

		go func() {
			defer waitGroup.Done()

			if player.IPAddr != nil {
				if errLoc := app.db.GetLocationRecord(ctx, player.IPAddr, &location); errLoc != nil {
					app.log.Error("Failed to fetch Location record", zap.Error(errLoc))
				}
			}
		}()

		go func() {
			defer waitGroup.Done()

			if player.IPAddr != nil {
				if errProxy := app.db.GetProxyRecord(ctx, player.IPAddr, &proxy); errProxy != nil && !errors.Is(errProxy, store.ErrNoResult) {
					app.log.Error("Failed to fetch proxy record", zap.Error(errProxy))
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

			for _, oldBan := range oldBans {
				if oldBan.Ban.BanType == store.Banned {
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
}

func makeOnHistory(app *App) discord.CommandHandler {
	return func(ctx context.Context, session *discordgo.Session,
		interaction *discordgo.InteractionCreate, response *discord.Response,
	) error {
		switch interaction.ApplicationCommandData().Name {
		case string(discord.CmdHistoryIP):
			return onHistoryIP(ctx, app, session, interaction, response)
		default:
			// return bot.onHistoryChat(ctx, session, interaction, response)
			return discord.ErrCommandFailed
		}
	}
}

func onHistoryIP(ctx context.Context, app *App, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discord.Response,
) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)

	steamID, errResolve := query.ResolveSID(ctx, opts[discord.OptUserIdentifier].StringValue())
	if errResolve != nil {
		return consts.ErrInvalidSID
	}

	person := store.NewPerson(steamID)
	if errPersonBySID := app.PersonBySID(ctx, steamID, &person); errPersonBySID != nil {
		return discord.ErrCommandFailed
	}

	ipRecords, errGetPersonIPHist := app.db.GetPersonIPHistory(ctx, steamID, 20)
	if errGetPersonIPHist != nil && !errors.Is(errGetPersonIPHist, store.ErrNoResult) {
		return discord.ErrCommandFailed
	}

	embed := discord.RespOk(response, fmt.Sprintf("IP History of: %s", person.PersonaName))
	lastIP := net.IP{}

	for _, ipRecord := range ipRecords {
		// TODO Join query for connections and geoip lookup data
		// addField(embed, ipRecord.IPAddr.String(), fmt.Sprintf("%s %s %s %s %s %s %s %s", config.FmtTimeShort(ipRecord.CreatedOn), ipRecord.CC,
		//	ipRecord.CityName, ipRecord.ASName, ipRecord.ISP, ipRecord.UsageType, ipRecord.Threat, ipRecord.DomainUsed))
		// lastIP = ipRecord.IPAddr
		if ipRecord.IPAddr.Equal(lastIP) {
			continue
		}
	}

	embed.Description = "IP history (20 max)"

	return nil
}

//
// func (bot *Discord) onHistoryChat(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, response *botResponse) error {
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
// }

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

func makeOnSetSteam(app *App) discord.CommandHandler {
	return func(ctx context.Context, _ *discordgo.Session,
		interaction *discordgo.InteractionCreate, response *discord.Response,
	) error {
		opts := discord.OptionMap(interaction.ApplicationCommandData().Options)

		steamID, errResolveSID := query.ResolveSID(ctx, opts[discord.OptUserIdentifier].StringValue())
		if errResolveSID != nil {
			return consts.ErrInvalidSID
		}

		errSetSteam := app.SetSteam(ctx, steamID, interaction.Member.User.ID)
		if errSetSteam != nil {
			return errSetSteam
		}

		embed := discord.RespOk(response, "Steam Account Linked")
		embed.Description = "Your steam and discord accounts are now linked"

		return nil
	}
}

func onUnbanSteam(ctx context.Context, app *App, _ *discordgo.Session, interaction *discordgo.InteractionCreate, response *discord.Response) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	reason := opts[discord.OptUnbanReason].StringValue()

	steamID, errResolveSID := query.ResolveSID(ctx, opts[discord.OptUserIdentifier].StringValue())
	if errResolveSID != nil {
		return consts.ErrInvalidSID
	}

	found, errUnban := app.Unban(ctx, steamID, reason)
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

func onUnbanASN(ctx context.Context, app *App, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discord.Response,
) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	asNumStr := opts[discord.OptASN].StringValue()

	banExisted, errUnbanASN := app.UnbanASN(ctx, asNumStr)
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

	asnNetworks, errGetASNRecords := app.db.GetASNRecordsByNum(ctx, asNum)
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

func getDiscordAuthor(ctx context.Context, db *store.Store, interaction *discordgo.InteractionCreate) (store.Person, error) {
	author := store.NewPerson("")
	if errPersonByDiscordID := db.GetPersonByDiscordID(ctx, interaction.Interaction.Member.User.ID, &author); errPersonByDiscordID != nil {
		if errors.Is(errPersonByDiscordID, store.ErrNoResult) {
			return author, errors.New("Must set steam id. See /set_steam")
		}

		return author, errors.New("Error fetching author info")
	}

	return author, nil
}

func makeOnKick(app *App) discord.CommandHandler {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, response *discord.Response) error {
		var (
			opts   = discord.OptionMap(interaction.ApplicationCommandData().Options)
			target = store.StringSID(opts[discord.OptUserIdentifier].StringValue())
			reason = store.Reason(opts[discord.OptBanReason].IntValue())
		)

		targetSid64, errTarget := target.SID64(ctx)
		if errTarget != nil {
			return consts.ErrInvalidSID
		}

		author, errAuthor := getDiscordAuthor(ctx, app.db, interaction)
		if errAuthor != nil {
			return errAuthor
		}

		players, found := app.Find(FindOpts{SteamID: targetSid64})
		if !found {
			return consts.ErrPlayerNotFound
		}

		var err error

		for _, player := range players {
			if errKick := app.Kick(ctx, store.Bot, player.Player.SID, author.SteamID, reason); errKick != nil {
				err = gerrors.Join(err, errKick)

				continue
			}

			embed := discord.RespOk(response, "User Kicked")
			discord.AddFieldsSteamID(embed, targetSid64)
			discord.AddField(embed, "NameShort", player.Player.Name)
		}

		return err
	}
}

func makeOnSay(app *App) discord.CommandHandler {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, response *discord.Response) error {
		opts := discord.OptionMap(interaction.ApplicationCommandData().Options)
		server := opts[discord.OptServerIdentifier].StringValue()
		msg := opts[discord.OptMessage].StringValue()

		if errSay := app.Say(ctx, "", server, msg); errSay != nil {
			return discord.ErrCommandFailed
		}

		embed := discord.RespOk(response, "Sent center message successfully")

		discord.AddField(embed, "Server", server)
		discord.AddField(embed, "Message", msg)

		return nil
	}
}

func makeOnCSay(app *App) discord.CommandHandler {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
		response *discord.Response,
	) error {
		opts := discord.OptionMap(interaction.ApplicationCommandData().Options)
		server := opts[discord.OptServerIdentifier].StringValue()
		msg := opts[discord.OptMessage].StringValue()

		if errCSay := app.CSay(ctx, "", server, msg); errCSay != nil {
			return discord.ErrCommandFailed
		}

		embed := discord.RespOk(response, "Sent console message successfully")

		discord.AddField(embed, "Server", server)
		discord.AddField(embed, "Message", msg)

		return nil
	}
}

func makeOnPSay(app *App) discord.CommandHandler {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, response *discord.Response) error {
		opts := discord.OptionMap(interaction.ApplicationCommandData().Options)
		player := store.StringSID(opts[discord.OptUserIdentifier].StringValue())
		msg := opts[discord.OptMessage].StringValue()

		playerSid, errPlayerSid := player.SID64(ctx)
		if errPlayerSid != nil {
			return errors.Wrap(errPlayerSid, "Failed to get player sid")
		}

		author, errAuthor := getDiscordAuthor(ctx, app.db, interaction)
		if errAuthor != nil {
			return errAuthor
		}

		if errPSay := app.PSay(ctx, author.SteamID, playerSid, msg); errPSay != nil {
			return discord.ErrCommandFailed
		}

		embed := discord.RespOk(response, "Sent private message successfully")

		discord.AddField(embed, "Player", string(player))
		discord.AddField(embed, "Message", msg)

		return nil
	}
}

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

func makeOnServers(app *App) discord.CommandHandler {
	return func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate,
		response *discord.Response,
	) error {
		var (
			currentState = app.state().ByRegion()
			stats        = map[string]float64{}
			used, total  = 0, 0
			embed        = discord.RespOk(response, "Current Server Populations")
			regionNames  []string
		)

		embed.URL = app.conf.ExtURL("/servers")

		for k := range currentState {
			regionNames = append(regionNames, k)
		}

		sort.Strings(regionNames)

		for _, region := range regionNames {
			var counts []string

			for _, curState := range currentState[region] {
				_, ok := stats[region]
				if !ok {
					stats[region] = 0
					stats[region+"total"] = 0
				}

				maxPlayers := curState.MaxPlayers - curState.Reserved
				if maxPlayers <= 0 {
					maxPlayers = 32 - curState.Reserved
				}

				stats[region] += float64(curState.PlayerCount)
				stats[region+"total"] += float64(maxPlayers)
				used += curState.PlayerCount
				total += maxPlayers
				counts = append(counts, fmt.Sprintf("%s: %2d/%2d", curState.NameShort, curState.PlayerCount, maxPlayers))
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
}

func makeOnPlayers(app *App) discord.CommandHandler {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, response *discord.Response) error {
		opts := discord.OptionMap(interaction.ApplicationCommandData().Options)
		serverName := opts[discord.OptServerIdentifier].StringValue()

		curState := app.state()

		serverState, found := curState.ByName(strings.ToLower(serverName))
		if !found {
			return consts.ErrUnknownID
		}

		var rows []string

		embed := discord.RespOk(response, fmt.Sprintf("Current Players: %d", len(serverState.Players)))

		if len(serverState.Players) > 0 {
			sort.SliceStable(serverState.Players, func(i, j int) bool {
				return serverState.Players[i].Name < serverState.Players[j].Name
			})

			for _, player := range serverState.Players {
				var asn ip2location.ASNRecord
				if errASN := app.db.GetASNRecordByIP(ctx, player.IP, &asn); errASN != nil {
					// Will fail for LAN ips
					app.log.Warn("Failed to get asn record", zap.Error(errASN))
				}

				var loc ip2location.LocationRecord
				if errLoc := app.db.GetLocationRecord(ctx, player.IP, &loc); errLoc != nil {
					app.log.Warn("Failed to get location record: %v", zap.Error(errLoc))
				}

				proxyStr := ""

				var proxy ip2location.ProxyRecord
				if errGetProxyRecord := app.db.GetProxyRecord(ctx, player.IP, &proxy); errGetProxyRecord == nil {
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

				rows = append(rows, fmt.Sprintf("%s`%s` %s`%3dms` [%s](https://steamcommunity.com/profiles/%s)%s",
					flag, player.SID, asStr, player.Ping, player.Name, player.SID, proxyStr))
			}

			embed.Description = strings.Join(rows, "\n")
		} else {
			embed.Description = "No players :("
		}

		return nil
	}
}

func onFilterAdd(ctx context.Context, app *App, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discord.Response,
) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	pattern := opts[discord.OptPattern].StringValue()
	isRegex := opts[discord.OptIsRegex].BoolValue()

	if isRegex {
		_, rxErr := regexp.Compile(pattern)
		if rxErr != nil {
			return errors.Errorf("Invalid regular expression: %v", rxErr)
		}
	}

	author, errAuthor := getDiscordAuthor(ctx, app.db, interaction)
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
	if errFilterAdd := app.FilterAdd(ctx, &filter); errFilterAdd != nil {
		return discord.ErrCommandFailed
	}

	embed := discord.RespOk(response, "Filter Created Successfully")
	embed.Description = filter.Pattern

	return nil
}

func onFilterDel(ctx context.Context, app *App, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	response *discord.Response,
) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	wordID := opts["filter"].IntValue()

	if wordID <= 0 {
		return errors.New("Invalid filter id")
	}

	var filter store.Filter
	if errGetFilter := app.db.GetFilterByID(ctx, wordID, &filter); errGetFilter != nil {
		return discord.ErrCommandFailed
	}

	if errDropFilter := app.db.DropFilter(ctx, &filter); errDropFilter != nil {
		return discord.ErrCommandFailed
	}

	discord.RespOk(response, "Filter Deleted Successfully")

	return nil
}

func onFilterCheck(_ context.Context, app *App, _ *discordgo.Session, interaction *discordgo.InteractionCreate, response *discord.Response) error {
	var (
		opts    = discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
		message = opts[discord.OptMessage].StringValue()
		matches = app.FilterCheck(message)
	)

	var title string
	if len(matches) == 0 {
		title = "No Match Found"
	} else {
		title = "Matched Found"
	}

	discord.RespOk(response, title)

	return nil
}

//	func (bot *discord) onStats(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, response *botResponse) error {
//		switch interaction.Data.Options[0].Name {
//		case string(cmdStatsPlayer):
//			return bot.onStatsPlayer(ctx, session, interaction, response)
//		case string(cmdStatsGlobal):
//			return bot.onStatsGlobal(ctx, session, interaction, response)
//		case string(cmdStatsServer):
//			return bot.onStatsServer(ctx, session, interaction, response)
//		default:
//			return errCommandFailed
//		}
//	}
//
//	func (bot *discord) onStatsPlayer(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, response *botResponse) error {
//		target := model.Target(interaction.Data.Options[0].Options[0].Value.(string))
//		sid, errSid := target.SID64()
//		if errSid != nil {
//			return errSid
//		}
//		person := model.NewPerson(sid)
//		if errPersonBySID := PersonBySID(ctx, bot.database, sid, "", &person); errPersonBySID != nil {
//			return errCommandFailed
//		}
//		var stats model.PlayerStats
//		if errStats := bot.database.GetPlayerStats(ctx, sid, &stats); errStats != nil {
//			return errCommandFailed
//		}
//		kd := 0.0
//		if stats.Deaths > 0 && stats.Kills > 0 {
//			kd = float64(stats.Kills) / float64(stats.Deaths)
//		}
//		kad := 0.0
//		if stats.Deaths > 0 && (stats.Kills+stats.Assists) > 0 {
//			kad = float64(stats.Kills+stats.Assists) / float64(stats.Deaths)
//		}
//		acc := 0.0
//		if stats.Hits > 0 && stats.Shots > 0 {
//			acc = float64(stats.Hits) / float64(stats.Shots) * 100
//		}
//		embed := respOk(response, fmt.Sprintf("Player stats for %s (%d)", person.PersonaName, person.SteamID.Int64()))
//		addFieldInline(embed, "Kills", fmt.Sprintf("%d", stats.Kills))
//		addFieldInline(embed, "Deaths", fmt.Sprintf("%d", stats.Deaths))
//		addFieldInline(embed, "Assists", fmt.Sprintf("%d", stats.Assists))
//		addFieldInline(embed, "K:D", fmt.Sprintf("%.2f", kd))
//		addFieldInline(embed, "KA:D", fmt.Sprintf("%.2f", kad))
//		addFieldInline(embed, "Damage", fmt.Sprintf("%d", stats.Damage))
//		addFieldInline(embed, "DamageTaken", fmt.Sprintf("%d", stats.DamageTaken))
//		addFieldInline(embed, "Healing", fmt.Sprintf("%d", stats.Healing))
//		addFieldInline(embed, "Shots", fmt.Sprintf("%d", stats.Shots))
//		addFieldInline(embed, "Hits", fmt.Sprintf("%d", stats.Hits))
//		addFieldInline(embed, "Accuracy", fmt.Sprintf("%.2f%%", acc))
//		return nil
//	}
//
//	func (bot *discord) onStatsServer(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, response *botResponse) error {
//		serverIdStr := interaction.Data.Options[0].Options[0].Value.(string)
//		var (
//			server model.Server
//			stats  model.ServerStats
//		)
//		if errServer := bot.database.GetServerByName(ctx, serverIdStr, &server); errServer != nil {
//			return errServer
//		}
//		if errStats := bot.database.GetServerStats(ctx, server.ServerID, &stats); errStats != nil {
//			return errCommandFailed
//		}
//		acc := 0.0
//		if stats.Hits > 0 && stats.Shots > 0 {
//			acc = float64(stats.Hits) / float64(stats.Shots) * 100
//		}
//		embed := respOk(response, fmt.Sprintf("Server stats for %s ", server.ServerName))
//		addFieldInline(embed, "Kills", fmt.Sprintf("%d", stats.Kills))
//		addFieldInline(embed, "Assists", fmt.Sprintf("%d", stats.Assists))
//		addFieldInline(embed, "Damage", fmt.Sprintf("%d", stats.Damage))
//		addFieldInline(embed, "Healing", fmt.Sprintf("%d", stats.Healing))
//		addFieldInline(embed, "Shots", fmt.Sprintf("%d", stats.Shots))
//		addFieldInline(embed, "Hits", fmt.Sprintf("%d", stats.Hits))
//		addFieldInline(embed, "Accuracy", fmt.Sprintf("%.2f%%", acc))
//		return nil
//	}
//
//	func (bot *discord) onStatsGlobal(ctx context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate, response *botResponse) error {
//		var stats model.GlobalStats
//		errStats := bot.database.GetGlobalStats(ctx, &stats)
//		if errStats != nil {
//			return errCommandFailed
//		}
//		acc := 0.0
//		if stats.Hits > 0 && stats.Shots > 0 {
//			acc = float64(stats.Hits) / float64(stats.Shots) * 100
//		}
//		embed := respOk(response, "Global stats")
//		addFieldInline(embed, "Kills", fmt.Sprintf("%d", stats.Kills))
//		addFieldInline(embed, "Assists", fmt.Sprintf("%d", stats.Assists))
//		addFieldInline(embed, "Damage", fmt.Sprintf("%d", stats.Damage))
//		addFieldInline(embed, "Healing", fmt.Sprintf("%d", stats.Healing))
//		addFieldInline(embed, "Shots", fmt.Sprintf("%d", stats.Shots))
//		addFieldInline(embed, "Hits", fmt.Sprintf("%d", stats.Hits))
//		addFieldInline(embed, "Accuracy", fmt.Sprintf("%.2f%%", acc))
//		return nil
//	}
func makeOnLog(app *App) discord.CommandHandler {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, response *discord.Response) error {
		opts := discord.OptionMap(interaction.ApplicationCommandData().Options)

		matchID := opts[discord.OptMatchID].IntValue()
		if matchID <= 0 {
			return discord.ErrCommandFailed
		}

		match, errMatch := app.db.MatchGetByID(ctx, int(matchID))
		if errMatch != nil {
			return discord.ErrCommandFailed
		}

		var server store.Server
		if errServer := app.db.GetServer(ctx, match.ServerID, &server); errServer != nil {
			return discord.ErrCommandFailed
		}

		embed := discord.RespOk(response, fmt.Sprintf("%s - %s", server.ServerName, match.MapName))
		embed.Color = int(discord.Green)
		embed.URL = app.conf.ExtURL("/match/%d", match.MatchID)

		redScore := 0
		bluScore := 0

		for _, round := range match.Rounds {
			redScore += round.Score.Red
			bluScore += round.Score.Blu
		}

		top := match.TopPlayers()
		found := 0

		discord.AddFieldInline(embed, "Red Score", fmt.Sprintf("%d", redScore))
		discord.AddFieldInline(embed, "Blu Score", fmt.Sprintf("%d", bluScore))
		discord.AddFieldInline(embed, "Players", fmt.Sprintf("%d", len(top)))

		for _, ts := range match.TeamSums {
			discord.AddFieldInline(embed, fmt.Sprintf("%s Kills", ts.Team.String()), fmt.Sprintf("%d", ts.Kills))
			discord.AddFieldInline(embed, fmt.Sprintf("%s Damage", ts.Team.String()), fmt.Sprintf("%d", ts.Damage))
			discord.AddFieldInline(embed, fmt.Sprintf("%s Ubers", ts.Team.String()), fmt.Sprintf("%d", ts.Caps))
			found++
		}

		description := "`Top players\n" +
			"N. K:D dmg heal sid\n"

		for index, player := range top {
			description += fmt.Sprintf("%d %d:%d %d %d %s\n", index+1, player.Kills, player.Deaths, player.Damage, player.Healing, player.SteamID.String())

			if index == 9 {
				break
			}
		}

		description += "`"
		embed.Description = description

		return nil
	}
}

func makeOnFind(app *App) discord.CommandHandler {
	return func(ctx context.Context, _ *discordgo.Session, i *discordgo.InteractionCreate,
		response *discord.Response,
	) error {
		opts := discord.OptionMap(i.ApplicationCommandData().Options)
		userIdentifier := store.StringSID(opts[discord.OptUserIdentifier].StringValue())

		sid, errSid := userIdentifier.SID64(ctx)
		if errSid != nil {
			return consts.ErrInvalidSID
		}

		players, found := app.Find(FindOpts{SteamID: sid})
		if !found {
			return consts.ErrUnknownID
		}

		for _, player := range players {
			var server store.Server
			if errServer := app.db.GetServer(ctx, player.ServerID, &server); errServer != nil {
				return errors.Wrapf(errServer, "Failed to get server")
			}

			person := store.NewPerson(player.Player.SID)
			if errPerson := app.PersonBySID(ctx, player.Player.SID, &person); errPerson != nil {
				return errPerson
			}

			resp := discord.RespOk(response, "Player Found")
			resp.Type = discordgo.EmbedTypeRich
			resp.Image = &discordgo.MessageEmbedImage{URL: person.AvatarFull}
			resp.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: person.Avatar}
			resp.URL = fmt.Sprintf("https://steamcommunity.com/profiles/%d", player.Player.SID.Int64())
			resp.Title = player.Player.Name

			discord.AddFieldInline(resp, "Server", server.ServerName)
			discord.AddFieldsSteamID(resp, player.Player.SID)
			discord.AddField(resp, "Connect", fmt.Sprintf("steam://connect/%s", server.Addr()))
		}

		return nil
	}
}

func makeOnMute(app *App) discord.CommandHandler {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
		resp *discord.Response,
	) error {
		var (
			opts     = discord.OptionMap(interaction.ApplicationCommandData().Options)
			playerID = store.StringSID(opts.String(discord.OptUserIdentifier))
			reason   store.Reason
		)

		reasonValueOpt, ok := opts[discord.OptBanReason]
		if !ok {
			return errors.New("Invalid mute reason")
		}

		reason = store.Reason(reasonValueOpt.IntValue())
		duration := store.Duration(opts[discord.OptDuration].StringValue())
		modNote := opts[discord.OptNote].StringValue()

		author, errAuthor := getDiscordAuthor(ctx, app.db, interaction)
		if errAuthor != nil {
			return errAuthor
		}

		var banSteam store.BanSteam
		if errOpts := store.NewBanSteam(ctx,
			store.StringSID(author.SteamID.String()),
			playerID,
			duration,
			reason,
			store.ReasonString(reason),
			modNote,
			store.Bot,
			0,
			store.NoComm,
			&banSteam,
		); errOpts != nil {
			return errors.Wrapf(errOpts, "Failed to parse options")
		}

		if errBan := app.BanSteam(ctx, &banSteam); errBan != nil {
			return errBan
		}

		response := discord.RespOk(resp, "Player muted successfully")
		discord.AddFieldsSteamID(response, banSteam.TargetID)

		return nil
	}
}

func onBanASN(ctx context.Context, app *App, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate, response *discord.Response,
) error {
	var (
		opts     = discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
		asNumStr = opts[discord.OptASN].StringValue()
		duration = store.Duration(opts[discord.OptDuration].StringValue())
		reason   = store.Reason(opts[discord.OptBanReason].IntValue())
		targetID = store.StringSID(opts[discord.OptUserIdentifier].StringValue())
		modNote  = opts[discord.OptNote].StringValue()
		author   = store.NewPerson("")
	)

	if errGetPersonByDiscordID := app.db.GetPersonByDiscordID(ctx, interaction.Interaction.Member.User.ID, &author); errGetPersonByDiscordID != nil {
		if errors.Is(errGetPersonByDiscordID, store.ErrNoResult) {
			return errors.New("Must set steam id. See /set_steam")
		}

		return errors.New("Error fetching author info")
	}

	asNum, errConv := strconv.ParseInt(asNumStr, 10, 64)
	if errConv != nil {
		return errors.New("Invalid ASN")
	}

	asnRecords, errGetASNRecords := app.db.GetASNRecordsByNum(ctx, asNum)
	if errGetASNRecords != nil {
		if errors.Is(errGetASNRecords, store.ErrNoResult) {
			return errors.New("No asnRecords found matching ASN")
		}

		return errors.New("Error fetching asn asnRecords")
	}

	var banASN store.BanASN
	if errOpts := store.NewBanASN(ctx,
		store.StringSID(author.SteamID.String()),
		targetID,
		duration,
		reason,
		store.ReasonString(reason),
		modNote,
		store.Bot,
		asNum,
		store.Banned,
		&banASN,
	); errOpts != nil {
		return errors.Wrapf(errOpts, "Failed to parse options")
	}

	if errBanASN := app.BanASN(ctx, &banASN); errBanASN != nil {
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

func onBanIP(ctx context.Context, app *App, _ *discordgo.Session,
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

	author := store.NewPerson("")
	if errGetPerson := app.db.GetPersonByDiscordID(ctx, interaction.Interaction.Member.User.ID, &author); errGetPerson != nil {
		if errors.Is(errGetPerson, store.ErrNoResult) {
			return errors.New("Must set steam id. See /set_steam")
		}

		return errors.New("Error fetching author info")
	}

	var banCIDR store.BanCIDR
	if errOpts := store.NewBanCIDR(ctx,
		store.StringSID(author.SteamID.String()),
		target,
		duration,
		reason,
		store.ReasonString(reason),
		modNote,
		store.Bot,
		cidr,
		store.Banned,
		&banCIDR,
	); errOpts != nil {
		return errors.Wrapf(errOpts, "Failed to parse options")
	}

	if errBanNet := app.BanCIDR(ctx, &banCIDR); errBanNet != nil {
		return errBanNet
	}

	players, found := app.Find(FindOpts{CIDR: network})
	if !found {
		return consts.ErrPlayerNotFound
	}

	for _, player := range players {
		if errKick := app.Kick(ctx, store.Bot, player.Player.SID, author.SteamID, reason); errKick != nil {
			app.log.Error("Failed to perform kick", zap.Error(errKick))
		}
	}

	discord.RespOk(response, "IP ban created successfully")

	return nil
}

// onBanSteam !ban <id> <duration> [reason].
func onBanSteam(ctx context.Context, app *App, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate, response *discord.Response,
) error {
	var (
		opts     = discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
		target   = opts[discord.OptUserIdentifier].StringValue()
		reason   = store.Reason(opts[discord.OptBanReason].IntValue())
		modNote  = opts[discord.OptNote].StringValue()
		duration = store.Duration(opts[discord.OptDuration].StringValue())
	)

	author, errAuthor := getDiscordAuthor(ctx, app.db, interaction)
	if errAuthor != nil {
		return errAuthor
	}

	var banSteam store.BanSteam
	if errOpts := store.NewBanSteam(ctx,
		store.StringSID(author.SteamID.String()),
		store.StringSID(target),
		duration,
		reason,
		store.ReasonString(reason),
		modNote,
		store.Bot,
		0,
		store.Banned,
		&banSteam,
	); errOpts != nil {
		return errors.Wrapf(errOpts, "Failed to parse options")
	}

	if errBan := app.BanSteam(ctx, &banSteam); errBan != nil {
		if errors.Is(errBan, store.ErrDuplicate) {
			return errors.New("Duplicate ban")
		}

		return discord.ErrCommandFailed
	}

	createDiscordBanEmbed(banSteam, response)

	return nil
}
