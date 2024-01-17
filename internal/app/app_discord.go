package app

import (
	"context"
	gerrors "errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gofrs/uuid/v5"
	embed "github.com/leighmacdonald/discordgo-embed"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/olekukonko/tablewriter"
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
		discord.CmdLogs:    makeOnLogs(app),
		discord.CmdMute:    makeOnMute(app),
		// discord.CmdCheckIP:  onCheckIp,
		discord.CmdPlayers:  makeOnPlayers(app),
		discord.CmdPSay:     makeOnPSay(app),
		discord.CmdSay:      makeOnSay(app),
		discord.CmdServers:  makeOnServers(app),
		discord.CmdSetSteam: makeOnSetSteam(app),
		discord.CmdUnban:    makeOnUnban(app),
		discord.CmdStats:    makeOnStats(app),
	}
	for k, v := range cmdMap {
		if errRegister := app.discord.RegisterHandler(k, v); errRegister != nil {
			return errors.Wrap(errRegister, "Failed to register discord command")
		}
	}

	return nil
}

func makeOnBan(app *App) discord.CommandHandler {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		name := interaction.ApplicationCommandData().Options[0].Name
		switch name {
		case "steam":
			return onBanSteam(ctx, app, session, interaction)
		case "ip":
			return onBanIP(ctx, app, session, interaction)
		case "asn":
			return onBanASN(ctx, app, session, interaction)
		default:
			return nil, discord.ErrCommandFailed
		}
	}
}

func makeOnUnban(app *App) discord.CommandHandler {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		switch interaction.ApplicationCommandData().Options[0].Name {
		case "steam":
			return onUnbanSteam(ctx, app, session, interaction)
		case "ip":
			return nil, discord.ErrCommandFailed
			// return discord.onUnbanIP(ctx, session, interaction, response)
		case "asn":
			return onUnbanASN(ctx, app, session, interaction)
		default:
			return nil, discord.ErrCommandFailed
		}
	}
}

func makeOnFilter(app *App) discord.CommandHandler {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		switch interaction.ApplicationCommandData().Options[0].Name {
		case "add":
			return onFilterAdd(ctx, app, session, interaction)
		case "del":
			return onFilterDel(ctx, app, session, interaction)
		case "check":
			return onFilterCheck(ctx, app, session, interaction)
		default:
			return nil, discord.ErrCommandFailed
		}
	}
}

func makeOnCheck(app *App) discord.CommandHandler { //nolint:maintidx
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, //nolint:maintidx
	) (*discordgo.MessageEmbed, error) {
		opts := discord.OptionMap(interaction.ApplicationCommandData().Options)
		sid, errResolveSID := resolveSID(ctx, opts[discord.OptUserIdentifier].StringValue())

		if errResolveSID != nil {
			return nil, consts.ErrInvalidSID
		}

		player := model.NewPerson(sid)
		if errGetPlayer := app.PersonBySID(ctx, sid, &player); errGetPlayer != nil {
			return nil, discord.ErrCommandFailed
		}

		ban := model.NewBannedPerson()
		if errGetBanBySID := app.db.GetBanBySteamID(ctx, sid, &ban, false); errGetBanBySID != nil {
			if !errors.Is(errGetBanBySID, store.ErrNoResult) {
				app.log.Error("Failed to get ban by steamid", zap.Error(errGetBanBySID))

				return nil, discord.ErrCommandFailed
			}
		}

		oldBans, _, errOld := app.db.GetBansSteam(ctx, store.SteamBansQueryFilter{
			BansQueryFilter: store.BansQueryFilter{
				QueryFilter: store.QueryFilter{Deleted: true},
				TargetID:    model.StringSID(sid),
			},
		})
		if errOld != nil {
			if !errors.Is(errOld, store.ErrNoResult) {
				app.log.Error("Failed to fetch old bans", zap.Error(errOld))
			}
		}

		bannedNets, errGetBanNet := app.db.GetBanNetByAddress(ctx, player.IPAddr)
		if errGetBanNet != nil {
			if !errors.Is(errGetBanNet, store.ErrNoResult) {
				app.log.Error("Failed to get ban nets by addr", zap.Error(errGetBanNet))

				return nil, discord.ErrCommandFailed
			}
		}

		var (
			conf          = app.config()
			color         = conf.Discord.ColourSuccess
			banned        = false
			muted         = false
			reason        = ""
			createdAt     = ""
			authorProfile = model.NewPerson(sid)
			msgEmbed      = discord.NewEmbed(conf)
			expiry        time.Time
		)

		// TODO Show the longest remaining ban.
		if ban.BanID > 0 {
			banned = ban.BanType == model.Banned
			muted = ban.BanType == model.NoComm
			reason = ban.ReasonText

			if len(reason) == 0 {
				// in case authorProfile ban without authorProfile reason ever makes its way here, we make sure
				// that Discord doesn't shit itself
				reason = "none"
			}

			expiry = ban.ValidUntil
			createdAt = ban.CreatedOn.Format(time.RFC3339)

			if ban.SourceID.Valid() {
				if errGetProfile := app.PersonBySID(ctx, ban.SourceID, &authorProfile); errGetProfile != nil {
					app.log.Error("Failed to load author for ban", zap.Error(errGetProfile))
				} else {
					msgEmbed.AddAuthorPersonInfo(authorProfile)
				}
			}

			msgEmbed.Embed().SetURL(conf.ExtURL(ban.BanSteam))
		}

		banStateStr := "no"

		if banned {
			// #992D22 red
			color = conf.Discord.ColourError
			banStateStr = "banned"
		}

		if muted {
			// #E67E22 orange
			color = conf.Discord.ColourWarn
			banStateStr = "muted"
		}

		msgEmbed.Embed().AddField("Ban/Muted", banStateStr)

		// TODO move elsewhere
		logData, errLogs := thirdparty.LogsTFOverview(ctx, sid)
		if errLogs != nil {
			app.log.Warn("Failed to fetch logTF data", zap.Error(errLogs))
		}

		if len(bannedNets) > 0 {
			// ip = bannedNets[0].CIDR.String()
			reason = fmt.Sprintf("Banned from %d networks", len(bannedNets))
			expiry = bannedNets[0].ValidUntil
			msgEmbed.Embed().AddField("Network Bans", fmt.Sprintf("%d", len(bannedNets)))
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

		if ban.BanID > 0 {
			if ban.BanType == model.Banned {
				title = fmt.Sprintf("%s (BANNED)", title)
			} else if ban.BanType == model.NoComm {
				title = fmt.Sprintf("%s (MUTED)", title)
			}
		}

		msgEmbed.Embed().SetTitle(title)

		if player.RealName != "" {
			msgEmbed.Embed().AddField("Real Name", player.RealName)
		}

		cd := time.Unix(int64(player.TimeCreated), 0)
		msgEmbed.Embed().AddField("Age", util.FmtDuration(cd))
		msgEmbed.Embed().AddField("Private", fmt.Sprintf("%v", player.CommunityVisibilityState == 1))
		msgEmbed.AddFieldsSteamID(player.SteamID)

		if player.VACBans > 0 {
			msgEmbed.Embed().AddField("VAC Bans", fmt.Sprintf("count: %d days: %d", player.VACBans, player.DaysSinceLastBan))
		}

		if player.GameBans > 0 {
			msgEmbed.Embed().AddField("Game Bans", fmt.Sprintf("count: %d", player.GameBans))
		}

		if player.CommunityBanned {
			msgEmbed.Embed().AddField("Com. Ban", "true")
		}

		if player.EconomyBan != "" {
			msgEmbed.Embed().AddField("Econ Ban", string(player.EconomyBan))
		}

		if len(oldBans) > 0 {
			numMutes, numBans := 0, 0

			for _, oldBan := range oldBans {
				if oldBan.BanType == model.Banned {
					numBans++
				} else {
					numMutes++
				}
			}

			msgEmbed.Embed().AddField("Total Mutes", fmt.Sprintf("%d", numMutes))
			msgEmbed.Embed().AddField("Total Bans", fmt.Sprintf("%d", numBans))
		}

		msgEmbed.Embed().InlineAllFields()

		if ban.BanID > 0 {
			msgEmbed.Embed().AddField("Reason", reason)
			msgEmbed.Embed().AddField("Created", util.FmtTimeShort(ban.CreatedOn)).MakeFieldInline()

			if time.Until(expiry) > time.Hour*24*365*5 {
				msgEmbed.Embed().AddField("Expires", "Permanent").MakeFieldInline()
			} else {
				msgEmbed.Embed().AddField("Expires", util.FmtDuration(expiry)).MakeFieldInline()
			}

			msgEmbed.Embed().AddField("Author", fmt.Sprintf("<@%s>", authorProfile.DiscordID)).MakeFieldInline()

			if ban.Note != "" {
				msgEmbed.Embed().AddField("Mod Note", ban.Note).MakeFieldInline()
			}

			msgEmbed.AddAuthorPersonInfo(authorProfile)
		}

		if player.IPAddr != nil {
			msgEmbed.Embed().AddField("Last IP", player.IPAddr.String()).MakeFieldInline()
		}

		if asn.ASName != "" {
			msgEmbed.Embed().AddField("ASN", fmt.Sprintf("(%d) %s", asn.ASNum, asn.ASName)).MakeFieldInline()
		}

		if location.CountryCode != "" {
			msgEmbed.Embed().AddField("City", location.CityName).MakeFieldInline()
		}

		if location.CountryName != "" {
			msgEmbed.Embed().AddField("Country", location.CountryName).MakeFieldInline()
		}

		if proxy.CountryCode != "" {
			msgEmbed.Embed().AddField("Proxy Type", string(proxy.ProxyType)).MakeFieldInline()
			msgEmbed.Embed().AddField("Proxy", string(proxy.Threat)).MakeFieldInline()
		}

		if logData != nil && logData.Total > 0 {
			msgEmbed.Embed().AddField("Logs.tf", fmt.Sprintf("%d", logData.Total)).MakeFieldInline()
		}

		if createdAt != "" {
			msgEmbed.Embed().AddField("created at", createdAt).MakeFieldInline()
		}

		msgEmbed.Embed().
			SetURL(player.ProfileURL).
			SetColor(color).
			SetImage(player.AvatarFull).
			SetThumbnail(player.Avatar).
			Truncate()

		return msgEmbed.Embed().MessageEmbed, nil
	}
}

func makeOnHistory(app *App) discord.CommandHandler {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		switch interaction.ApplicationCommandData().Name {
		case string(discord.CmdHistoryIP):
			return onHistoryIP(ctx, app, session, interaction)
		default:
			// return discord.onHistoryChat(ctx, session, interaction, response)
			return nil, discord.ErrCommandFailed
		}
	}
}

func onHistoryIP(ctx context.Context, app *App, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)

	steamID, errResolve := resolveSID(ctx, opts[discord.OptUserIdentifier].StringValue())
	if errResolve != nil {
		return nil, consts.ErrInvalidSID
	}

	person := model.NewPerson(steamID)
	if errPersonBySID := app.PersonBySID(ctx, steamID, &person); errPersonBySID != nil {
		return nil, discord.ErrCommandFailed
	}

	ipRecords, errGetPersonIPHist := app.db.GetPersonIPHistory(ctx, steamID, 20)
	if errGetPersonIPHist != nil && !errors.Is(errGetPersonIPHist, store.ErrNoResult) {
		return nil, discord.ErrCommandFailed
	}

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

	conf := app.config()

	msgEmbed := discord.NewEmbed(conf, fmt.Sprintf("IP History of: %s", person.PersonaName)).Embed().
		SetDescription("IP history (20 max)").
		Truncate()

	return msgEmbed.MessageEmbed, nil
}

//
// func (discord *Discord) onHistoryChat(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, response *botResponse) error {
//	steamId, errResolveSID := resolveSID(ctx, interaction.Data.Options[0].Options[0].Value.(string))
//	if errResolveSID != nil {
//		return consts.ErrInvalidSID
//	}
//	person := model.NewPerson(steamId)
//	if errPersonBySID := PersonBySID(ctx, discord.database, steamId, "", &person); errPersonBySID != nil {
//		return errCommandFailed
//	}
//	chatHistory, errChatHistory := discord.database.GetChatHistory(ctx, steamId, 25)
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

func (app *App) createDiscordBanEmbed(ctx context.Context, ban model.BanSteam) (*embed.Embed, error) {
	conf := app.config()
	msgEmbed := discord.NewEmbed(conf)
	msgEmbed.Embed().
		SetTitle(fmt.Sprintf("Ban created successfully (#%d)", ban.BanID)).
		SetDescription(ban.Note).
		SetURL(conf.ExtURL(ban)).
		SetColor(conf.Discord.ColourSuccess)

	if ban.ReasonText != "" {
		msgEmbed.Embed().AddField("Reason", ban.ReasonText)
	}

	var target model.Person
	if errTarget := app.PersonBySID(ctx, ban.TargetID, &target); errTarget != nil {
		return nil, errTarget
	}

	msgEmbed.Embed().SetImage(target.AvatarFull)

	var author model.Person

	if errTarget := app.PersonBySID(ctx, ban.SourceID, &target); errTarget != nil {
		return nil, errTarget
	}

	msgEmbed.AddAuthorPersonInfo(author)

	if ban.ValidUntil.Year()-time.Now().Year() > 5 {
		msgEmbed.Embed().AddField("Expires In", "Permanent")
		msgEmbed.Embed().AddField("Expires At", "Permanent")
	} else {
		msgEmbed.Embed().AddField("Expires In", util.FmtDuration(ban.ValidUntil))
		msgEmbed.Embed().AddField("Expires At", util.FmtTimeShort(ban.ValidUntil))
	}

	msgEmbed.AddFieldsSteamID(ban.TargetID)

	return msgEmbed.Embed().Truncate(), nil
}

func makeOnSetSteam(app *App) discord.CommandHandler {
	return func(ctx context.Context, _ *discordgo.Session,
		interaction *discordgo.InteractionCreate,
	) (*discordgo.MessageEmbed, error) {
		opts := discord.OptionMap(interaction.ApplicationCommandData().Options)

		steamID, errResolveSID := resolveSID(ctx, opts[discord.OptUserIdentifier].StringValue())
		if errResolveSID != nil {
			return nil, consts.ErrInvalidSID
		}

		errSetSteam := app.SetSteam(ctx, steamID, interaction.Member.User.ID)
		if errSetSteam != nil {
			return nil, errSetSteam
		}

		conf := app.config()

		msgEmbed := discord.NewEmbed(conf).Embed().
			SetTitle("Steam Account Linked").
			SetDescription("Your steam and discord accounts are now linked").
			SetColor(conf.Discord.ColourSuccess).
			Truncate()

		return msgEmbed.MessageEmbed, nil
	}
}

func onUnbanSteam(ctx context.Context, app *App, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	reason := opts[discord.OptUnbanReason].StringValue()

	steamID, errResolveSID := resolveSID(ctx, opts[discord.OptUserIdentifier].StringValue())
	if errResolveSID != nil {
		return nil, consts.ErrInvalidSID
	}

	found, errUnban := app.Unban(ctx, steamID, reason)
	if errUnban != nil {
		return nil, errUnban
	}

	if !found {
		return nil, errors.New("No ban found")
	}

	conf := app.config()

	msgEmbed := discord.NewEmbed(conf, "User Unbanned Successfully")
	msgEmbed.Embed().SetColor(conf.Discord.ColourSuccess)

	var user model.Person
	if errUser := app.PersonBySID(ctx, steamID, &user); errUser != nil {
		app.log.Warn("Could not fetch unbanned person", zap.String("steam_id", steamID.String()), zap.Error(errUser))
	} else {
		msgEmbed.Embed().SetImage(user.AvatarFull).
			SetURL(user.AvatarFull)
	}

	msgEmbed.AddFieldsSteamID(steamID)

	return msgEmbed.Embed().Truncate().MessageEmbed, nil
}

func onUnbanASN(ctx context.Context, app *App, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	asNumStr := opts[discord.OptASN].StringValue()

	banExisted, errUnbanASN := app.UnbanASN(ctx, asNumStr)
	if errUnbanASN != nil {
		if errors.Is(errUnbanASN, store.ErrNoResult) {
			return nil, errors.New("Ban for ASN does not exist")
		}

		return nil, discord.ErrCommandFailed
	}

	if !banExisted {
		return nil, errors.New("Ban for ASN does not exist")
	}

	asNum, errConv := strconv.ParseInt(asNumStr, 10, 64)
	if errConv != nil {
		return nil, errors.New("Invalid ASN")
	}

	asnNetworks, errGetASNRecords := app.db.GetASNRecordsByNum(ctx, asNum)
	if errGetASNRecords != nil {
		if errors.Is(errGetASNRecords, store.ErrNoResult) {
			return nil, errors.New("No asnNetworks found matching ASN")
		}

		return nil, errors.New("Error fetching asn asnNetworks")
	}

	conf := app.config()

	msgEmbed := discord.
		NewEmbed(app.config(), "ASN Networks Unbanned Successfully").
		Embed().
		SetColor(conf.Discord.ColourSuccess).
		AddField("ASN", asNumStr).
		AddField("Hosts", fmt.Sprintf("%d", asnNetworks.Hosts())).
		Truncate()

	return msgEmbed.MessageEmbed, nil
}

func getDiscordAuthor(ctx context.Context, db *store.Store, interaction *discordgo.InteractionCreate) (model.Person, error) {
	author := model.NewPerson("")
	if errPersonByDiscordID := db.GetPersonByDiscordID(ctx, interaction.Interaction.Member.User.ID, &author); errPersonByDiscordID != nil {
		if errors.Is(errPersonByDiscordID, store.ErrNoResult) {
			return author, errors.New("Must set steam id. See /set_steam")
		}

		return author, errors.New("Error fetching author info")
	}

	return author, nil
}

func makeOnKick(app *App) discord.CommandHandler {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		var (
			opts   = discord.OptionMap(interaction.ApplicationCommandData().Options)
			target = model.StringSID(opts[discord.OptUserIdentifier].StringValue())
			reason = model.Reason(opts[discord.OptBanReason].IntValue())
		)

		targetSid64, errTarget := target.SID64(ctx)
		if errTarget != nil {
			return nil, consts.ErrInvalidSID
		}

		author, errAuthor := getDiscordAuthor(ctx, app.db, interaction)
		if errAuthor != nil {
			return nil, errAuthor
		}

		state := app.state.current()
		players := state.find(findOpts{SteamID: targetSid64})

		if len(players) == 0 {
			return nil, consts.ErrPlayerNotFound
		}

		msgEmbed := discord.NewEmbed(app.config(), "Users Kicked")

		var err error

		for _, player := range players {
			if errKick := app.Kick(ctx, model.Bot, player.Player.SID, author.SteamID, reason); errKick != nil {
				err = gerrors.Join(err, errKick)

				continue
			}

			msgEmbed.Embed().AddField("Name", player.Player.Name)
			msgEmbed.AddFieldsSteamID(targetSid64)
		}

		return msgEmbed.Embed().Truncate().MessageEmbed, err
	}
}

func makeOnSay(app *App) discord.CommandHandler {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		opts := discord.OptionMap(interaction.ApplicationCommandData().Options)
		server := opts[discord.OptServerIdentifier].StringValue()
		msg := opts[discord.OptMessage].StringValue()

		if errSay := app.Say(ctx, "", server, msg); errSay != nil {
			return nil, discord.ErrCommandFailed
		}

		conf := app.config()

		msgEmbed := discord.
			NewEmbed(conf, "Sent center message successfully").Embed().
			SetColor(conf.Discord.ColourSuccess).
			AddField("Server", server).
			AddField("Message", msg).
			Truncate()

		return msgEmbed.MessageEmbed, nil
	}
}

func makeOnCSay(app *App) discord.CommandHandler {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	) (*discordgo.MessageEmbed, error) {
		opts := discord.OptionMap(interaction.ApplicationCommandData().Options)
		server := opts[discord.OptServerIdentifier].StringValue()
		msg := opts[discord.OptMessage].StringValue()

		if errCSay := app.CSay(ctx, "", server, msg); errCSay != nil {
			return nil, discord.ErrCommandFailed
		}

		conf := app.config()

		msgEmbed := discord.
			NewEmbed(conf, "Sent console message successfully").Embed().
			SetColor(conf.Discord.ColourSuccess).
			AddField("Server", server).
			AddField("Message", msg).
			Truncate()

		return msgEmbed.MessageEmbed, nil
	}
}

func makeOnPSay(app *App) discord.CommandHandler {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		opts := discord.OptionMap(interaction.ApplicationCommandData().Options)
		player := model.StringSID(opts[discord.OptUserIdentifier].StringValue())
		msg := opts[discord.OptMessage].StringValue()

		playerSid, errPlayerSid := player.SID64(ctx)
		if errPlayerSid != nil {
			return nil, errors.Wrap(errPlayerSid, "Failed to get player sid")
		}

		if errPSay := app.PSay(ctx, playerSid, msg); errPSay != nil {
			return nil, discord.ErrCommandFailed
		}

		conf := app.config()

		msgEmbed := discord.
			NewEmbed(conf, "Sent private message successfully").Embed().
			SetColor(conf.Discord.ColourSuccess).
			AddField("Player", string(player)).
			AddField("Message", msg).
			Truncate()

		return msgEmbed.MessageEmbed, nil
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
	return func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		var (
			conf         = app.config()
			state        = app.state.current()
			currentState = state.sortRegion()
			stats        = map[string]float64{}
			used, total  = 0, 0
			msgEmbed     = discord.NewEmbed(conf, "Current Server Populations")
			regionNames  []string
		)

		msgEmbed.Embed().SetURL(conf.ExtURLRaw("/servers"))

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
				counts = append(counts, fmt.Sprintf("%s:   %2d/%2d  ", curState.NameShort, curState.PlayerCount, maxPlayers))
			}

			msg := strings.Join(counts, "    ")
			if msg != "" {
				msgEmbed.Embed().AddField(mapRegion(region), fmt.Sprintf("```%s```", msg))
			}
		}

		for statName := range stats {
			if strings.HasSuffix(statName, "total") {
				continue
			}

			msgEmbed.Embed().AddField(mapRegion(statName), fmt.Sprintf("%.2f%%", (stats[statName]/stats[statName+"total"])*100)).MakeFieldInline()
		}

		msgEmbed.Embed().AddField("Global", fmt.Sprintf("%d/%d %.2f%%", used, total, float64(used)/float64(total)*100)).MakeFieldInline()

		if total == 0 {
			msgEmbed.Embed().SetColor(conf.Discord.ColourError)
			msgEmbed.Embed().SetDescription("No server states available")
		}

		return msgEmbed.Embed().Truncate().MessageEmbed, nil
	}
}

func makeOnPlayers(app *App) discord.CommandHandler {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		opts := discord.OptionMap(interaction.ApplicationCommandData().Options)
		serverName := opts[discord.OptServerIdentifier].StringValue()
		state := app.state.current()
		serverStates := state.byName(serverName, false)

		if len(serverStates) != 1 {
			return nil, errUnknownServer
		}

		serverState := serverStates[0]

		var rows []string

		conf := app.config()

		msgEmbed := discord.NewEmbed(conf, fmt.Sprintf("%s Current Players: %d / %d", serverState.Name, len(serverState.Players), serverState.MaxPlayers))

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

			msgEmbed.Embed().SetDescription(strings.Join(rows, "\n"))
			msgEmbed.Embed().SetColor(conf.Discord.ColourSuccess)
		} else {
			msgEmbed.Embed().SetDescription("No players :(")
			msgEmbed.Embed().SetColor(conf.Discord.ColourError)
		}

		return msgEmbed.Embed().MessageEmbed, nil
	}
}

func onFilterAdd(ctx context.Context, app *App, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
) (*discordgo.MessageEmbed, error) {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	pattern := opts[discord.OptPattern].StringValue()
	isRegex := opts[discord.OptIsRegex].BoolValue()

	if isRegex {
		_, rxErr := regexp.Compile(pattern)
		if rxErr != nil {
			return nil, errors.Errorf("Invalid regular expression: %v", rxErr)
		}
	}

	author, errAuthor := getDiscordAuthor(ctx, app.db, interaction)
	if errAuthor != nil {
		return nil, errAuthor
	}

	filter := model.Filter{
		AuthorID:  author.SteamID,
		Pattern:   pattern,
		IsRegex:   isRegex,
		IsEnabled: true,
		CreatedOn: time.Now(),
		UpdatedOn: time.Now(),
	}
	if errFilterAdd := app.FilterAdd(ctx, &filter); errFilterAdd != nil {
		return nil, discord.ErrCommandFailed
	}

	conf := app.config()

	msgEmbed := discord.
		NewEmbed(conf, "Filter Created Successfully").Embed().
		SetColor(conf.Discord.ColourSuccess).
		AddField("pattern", filter.Pattern).
		Truncate()

	return msgEmbed.MessageEmbed, nil
}

func onFilterDel(ctx context.Context, app *App, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	wordID := opts["filter"].IntValue()

	if wordID <= 0 {
		return nil, errors.New("Invalid filter id")
	}

	var filter model.Filter
	if errGetFilter := app.db.GetFilterByID(ctx, wordID, &filter); errGetFilter != nil {
		return nil, discord.ErrCommandFailed
	}

	if errDropFilter := app.db.DropFilter(ctx, &filter); errDropFilter != nil {
		return nil, discord.ErrCommandFailed
	}

	conf := app.config()

	msgEmbed := discord.
		NewEmbed(conf, "Filter Deleted Successfully").Embed().
		SetColor(conf.Discord.ColourSuccess).
		AddField("filter", filter.Pattern).
		Truncate()

	return msgEmbed.MessageEmbed, nil
}

func onFilterCheck(_ context.Context, app *App, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	var (
		opts    = discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
		message = opts[discord.OptMessage].StringValue()
		matches = app.FilterCheck(message)
	)

	msgEmbed := discord.NewEmbed(app.config())

	conf := app.config()

	if len(matches) == 0 {
		msgEmbed.Embed().SetTitle("No Matches Found")
		msgEmbed.Embed().SetColor(conf.Discord.ColourSuccess)
	} else {
		msgEmbed.Embed().SetTitle("Matched Found")
		msgEmbed.Embed().SetColor(conf.Discord.ColourWarn)
		for _, match := range matches {
			msgEmbed.Embed().AddField(fmt.Sprintf("Matched ID: %d", match.FilterID), match.Pattern)
		}
	}

	return msgEmbed.Embed().Truncate().MessageEmbed, nil
}

func makeOnStats(app *App) discord.CommandHandler {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		name := interaction.ApplicationCommandData().Options[0].Name
		switch name {
		case "player":
			return onStatsPlayer(ctx, app, session, interaction)
		// case string(cmdStatsGlobal):
		//	return discord.onStatsGlobal(ctx, session, interaction, response)
		// case string(cmdStatsServer):
		//	return discord.onStatsServer(ctx, session, interaction, response)
		default:
			return nil, discord.ErrCommandFailed
		}
	}
}

func makeClassStatsTable(classes model.PlayerClassStatsCollection) string {
	writer := &strings.Builder{}
	table := defaultTable(writer)
	table.SetHeader([]string{"Class", "K", "A", "D", "KD", "KAD", "DA", "DT", "Dom", "Time"})

	table.Append([]string{
		"total",
		fmt.Sprintf("%d", classes.Kills()),
		fmt.Sprintf("%d", classes.Assists()),
		fmt.Sprintf("%d", classes.Deaths()),
		infString(classes.KDRatio()),
		infString(classes.KDARatio()),
		fmt.Sprintf("%d", classes.Damage()),
		// fmt.Sprintf("%d", classes.DamagePerMin()),
		fmt.Sprintf("%d", classes.DamageTaken()),
		// fmt.Sprintf("%d", classes.Captures()),
		fmt.Sprintf("%d", classes.Dominations()),
		// fmt.Sprintf("%d", classes.Dominated()),
		fmt.Sprintf("%.1fh", (time.Duration(int64(classes.Playtime())) * time.Second).Hours()),
	})

	sort.SliceStable(classes, func(i, j int) bool {
		return classes[i].Playtime > classes[j].Playtime
	})

	for _, player := range classes {
		table.Append([]string{
			player.ClassName,
			fmt.Sprintf("%d", player.Kills),
			fmt.Sprintf("%d", player.Assists),
			fmt.Sprintf("%d", player.Deaths),
			infString(player.KDRatio()),
			infString(player.KDARatio()),
			fmt.Sprintf("%d", player.Damage),
			// fmt.Sprintf("%d", player.DamagePerMin()),
			fmt.Sprintf("%d", player.DamageTaken),
			// fmt.Sprintf("%d", player.Captures),
			fmt.Sprintf("%d", player.Dominations),
			// fmt.Sprintf("%d", player.Dominated),
			fmt.Sprintf("%.1fh", (time.Duration(int64(player.Playtime)) * time.Second).Hours()),
		})
	}

	table.Render()

	return strings.Trim(writer.String(), "\n")
}

func makeWeaponStatsTable(weapons []model.PlayerWeaponStats) string {
	writer := &strings.Builder{}
	table := defaultTable(writer)
	table.SetHeader([]string{"Weapon", "K", "Dmg", "Sh", "Hi", "Acc", "B", "H", "A"})

	sort.SliceStable(weapons, func(i, j int) bool {
		return weapons[i].Kills > weapons[j].Kills
	})

	for i, weapon := range weapons {
		if i == 10 {
			break
		}

		table.Append([]string{
			weapon.WeaponName,
			fmt.Sprintf("%d", weapon.Kills),
			fmt.Sprintf("%d", weapon.Damage),
			fmt.Sprintf("%d", weapon.Shots),
			fmt.Sprintf("%d", weapon.Hits),
			fmt.Sprintf("%.1f", weapon.Accuracy()),
			fmt.Sprintf("%d", weapon.Backstabs),
			fmt.Sprintf("%d", weapon.Headshots),
			fmt.Sprintf("%d", weapon.Airshots),
		})
	}

	table.Render()

	return writer.String()
}

func makeKillstreakStatsTable(killstreaks []model.PlayerKillstreakStats) string {
	writer := &strings.Builder{}
	table := defaultTable(writer)
	table.SetHeader([]string{"Ks", "Class", "Dur", "Date"})

	sort.SliceStable(killstreaks, func(i, j int) bool {
		return killstreaks[i].Kills > killstreaks[j].Kills
	})

	for index, killstreak := range killstreaks {
		if index == 3 {
			break
		}

		table.Append([]string{
			fmt.Sprintf("%d", killstreak.Kills),
			killstreak.Class.String(),
			fmt.Sprintf("%d", killstreak.Duration),
			killstreak.CreatedOn.Format(time.DateOnly),
		})
	}

	table.Render()

	return writer.String()
}

func makeMedicStatsTable(stats []model.PlayerMedicStats) string {
	writer := &strings.Builder{}
	table := defaultTable(writer)
	table.SetHeader([]string{"Healing", "Drop", "NearFull", "AvgLen", "U", "K", "V", "Q"})

	sort.SliceStable(stats, func(i, j int) bool {
		return stats[i].Healing > stats[j].Healing
	})

	for index, medicStats := range stats {
		if index == 3 {
			break
		}

		table.Append([]string{
			fmt.Sprintf("%d", medicStats.Healing),
			fmt.Sprintf("%d", medicStats.Drops),
			fmt.Sprintf("%d", medicStats.NearFullChargeDeath),
			fmt.Sprintf("%.2f", medicStats.AvgUberLength),
			fmt.Sprintf("%d", medicStats.ChargesUber),
			fmt.Sprintf("%d", medicStats.ChargesKritz),
			fmt.Sprintf("%d", medicStats.ChargesVacc),
			fmt.Sprintf("%d", medicStats.ChargesQuickfix),
		})
	}

	table.Render()

	return writer.String()
}

func onStatsPlayer(ctx context.Context, app *App, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	steamID, errResolveSID := resolveSID(ctx, opts[discord.OptUserIdentifier].StringValue())

	if errResolveSID != nil {
		return nil, consts.ErrInvalidSID
	}

	person := model.NewPerson(steamID)
	errAuthor := app.PersonBySID(ctx, steamID, &person)

	if errAuthor != nil {
		return nil, errAuthor
	}

	//
	// person, errAuthor := getDiscordAuthor(ctx, app.db, interaction)
	// if errAuthor != nil {
	//	return nil, errAuthor
	// }

	classStats, errClassStats := app.db.StatsPlayerClass(ctx, person.SteamID)
	if errClassStats != nil {
		return nil, errors.Wrap(errClassStats, "Failed to fetch class stats")
	}

	weaponStats, errWeaponStats := app.db.StatsPlayerWeapons(ctx, person.SteamID)
	if errWeaponStats != nil {
		return nil, errors.Wrap(errWeaponStats, "Failed to fetch weapon stats")
	}

	killstreakStats, errKillstreakStats := app.db.StatsPlayerKillstreaks(ctx, person.SteamID)
	if errKillstreakStats != nil {
		return nil, errors.Wrap(errKillstreakStats, "Failed to fetch killstreak stats")
	}

	medicStats, errMedicStats := app.db.StatsPlayerMedic(ctx, person.SteamID)
	if errMedicStats != nil {
		return nil, errors.Wrap(errMedicStats, "Failed to fetch medic stats")
	}

	conf := app.config()

	emb := discord.NewEmbed(conf)
	emb.Embed().
		SetTitle("Overall Player Stats").
		SetColor(conf.Discord.ColourSuccess)

	emb.Embed().SetDescription(fmt.Sprintf("Class Totals```%s```Healing```%s```Top Weapons```%s```Top Killstreaks```%s```",
		makeClassStatsTable(classStats),
		makeMedicStatsTable(medicStats),
		makeWeaponStatsTable(weaponStats),
		makeKillstreakStatsTable(killstreakStats),
	))

	emb.AddAuthorPersonInfo(person)

	return emb.Embed().MessageEmbed, nil
}

//	func (discord *discord) onStatsServer(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, response *botResponse) error {
//		serverIdStr := interaction.Data.Options[0].Options[0].Value.(string)
//		var (
//			server model.Server
//			stats  model.ServerStats
//		)
//		if errServer := discord.database.GetServerByName(ctx, serverIdStr, &server); errServer != nil {
//			return errServer
//		}
//		if errStats := discord.database.GetServerStats(ctx, server.ServerID, &stats); errStats != nil {
//			return errCommandFailed
//		}
//		acc := 0.0
//		if stats.Hits > 0 && stats.Shots > 0 {
//			acc = float64(stats.Hits) / float64(stats.Shots) * 100
//		}
//		embed := respOk(response, fmt.Sprintf("Server stats for %s ", server.ShortName))
//		addFieldInline(embed, "Kills", fmt.Sprintf("%d", stats.Kills))
//		addFieldInline(embed, "Assists", fmt.Sprintf("%d", stats.Assists))
//		addFieldInline(embed, "Damage", fmt.Sprintf("%d", stats.Damage))
//		addFieldInline(embed, "MedicStats", fmt.Sprintf("%d", stats.MedicStats))
//		addFieldInline(embed, "Shots", fmt.Sprintf("%d", stats.Shots))
//		addFieldInline(embed, "Hits", fmt.Sprintf("%d", stats.Hits))
//		addFieldInline(embed, "Accuracy", fmt.Sprintf("%.2f%%", acc))
//		return nil
//	}
//
//	func (discord *discord) onStatsGlobal(ctx context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate, response *botResponse) error {
//		var stats model.GlobalStats
//		errStats := discord.database.GetGlobalStats(ctx, &stats)
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
//		addFieldInline(embed, "MedicStats", fmt.Sprintf("%d", stats.MedicStats))
//		addFieldInline(embed, "Shots", fmt.Sprintf("%d", stats.Shots))
//		addFieldInline(embed, "Hits", fmt.Sprintf("%d", stats.Hits))
//		addFieldInline(embed, "Accuracy", fmt.Sprintf("%.2f%%", acc))
//		return nil
//	}

func (app *App) genDiscordMatchEmbed(match model.MatchResult) *embed.Embed {
	conf := app.config()

	msgEmbed := discord.NewEmbed(conf, strings.Join([]string{match.Title, match.MapName}, " | "))
	msgEmbed.Embed().
		SetColor(conf.Discord.ColourSuccess).
		SetURL(conf.ExtURLRaw("/log/%s", match.MatchID.String()))

	msgEmbed.Embed().SetDescription(matchASCIITable(match))

	msgEmbed.Embed().AddField("Red Score", fmt.Sprintf("%d", match.TeamScores.Red)).MakeFieldInline()
	msgEmbed.Embed().AddField("Blu Score", fmt.Sprintf("%d", match.TeamScores.Blu)).MakeFieldInline()
	msgEmbed.Embed().AddField("Map", match.MapName).MakeFieldInline()
	msgEmbed.Embed().AddField("Chat Messages", fmt.Sprintf("%d", len(match.Chat))).MakeFieldInline()

	msgCounts := map[steamid.SID64]int{}

	for _, msg := range match.Chat {
		_, found := msgCounts[msg.SteamID]
		if !found {
			msgCounts[msg.SteamID] = 0
		}
		msgCounts[msg.SteamID]++
	}

	var (
		chatSid   steamid.SID64
		count     int
		kathyName string
	)

	for sid, cnt := range msgCounts {
		if cnt > count {
			count = cnt
			chatSid = sid
		}
	}

	for _, player := range match.Players {
		if player.SteamID == chatSid {
			kathyName = player.Name

			break
		}
	}

	msgEmbed.Embed().AddField("Top Chatter", fmt.Sprintf("%s (count: %d)", kathyName, count)).MakeFieldInline()
	msgEmbed.Embed().AddField("Players", fmt.Sprintf("%d", len(match.Players))).MakeFieldInline()
	msgEmbed.Embed().AddField("Duration", match.TimeEnd.Sub(match.TimeStart).String()).MakeFieldInline()

	return msgEmbed.Embed().Truncate()
}

func makeOnLogs(app *App) discord.CommandHandler {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		author, errAuthor := getDiscordAuthor(ctx, app.db, interaction)
		if errAuthor != nil {
			return nil, errAuthor
		}

		matches, count, errMatch := app.db.Matches(ctx, store.MatchesQueryOpts{
			SteamID:     author.SteamID,
			QueryFilter: store.QueryFilter{Limit: 5},
		})

		if errMatch != nil {
			return nil, discord.ErrCommandFailed
		}

		conf := app.config()

		matchesWriter := &strings.Builder{}

		for _, match := range matches {
			status := ":x:"
			if match.IsWinner {
				status = ":white_check_mark:"
			}

			_, _ = matchesWriter.WriteString(fmt.Sprintf("%s [%s](%s) `%s` `%s`\n",
				status, match.Title, conf.ExtURL(match), match.MapName, match.TimeStart.Format(time.DateOnly)))
		}

		msgEmbed := discord.
			NewEmbed(conf, fmt.Sprintf("Your most recent matches [%d total]", count)).Embed().
			SetColor(conf.Discord.ColourSuccess).
			SetDescription(matchesWriter.String())

		return msgEmbed.MessageEmbed, nil
	}
}

func makeOnLog(app *App) discord.CommandHandler {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		opts := discord.OptionMap(interaction.ApplicationCommandData().Options)

		matchIDStr := opts[discord.OptMatchID].StringValue()

		matchID, errMatchID := uuid.FromString(matchIDStr)
		if errMatchID != nil {
			return nil, discord.ErrCommandFailed
		}

		var match model.MatchResult

		if errMatch := app.db.MatchGetByID(ctx, matchID, &match); errMatch != nil {
			return nil, discord.ErrCommandFailed
		}

		return app.genDiscordMatchEmbed(match).MessageEmbed, nil
	}
}

func makeOnFind(app *App) discord.CommandHandler {
	return func(ctx context.Context, _ *discordgo.Session, i *discordgo.InteractionCreate,
	) (*discordgo.MessageEmbed, error) {
		opts := discord.OptionMap(i.ApplicationCommandData().Options)
		userIdentifier := opts[discord.OptUserIdentifier].StringValue()

		var playerFindOpts findOpts

		steamID, errSteamID := steamid.StringToSID64(userIdentifier)
		if errSteamID != nil {
			playerFindOpts = findOpts{Name: userIdentifier}
		} else {
			playerFindOpts = findOpts{SteamID: steamID}
		}

		state := app.state.current()
		players := state.find(playerFindOpts)

		if len(players) == 0 {
			return nil, consts.ErrUnknownID
		}

		conf := app.config()

		msgEmbed := discord.NewEmbed(conf, "Player(s) Found")

		for _, player := range players {
			var server model.Server
			if errServer := app.db.GetServer(ctx, player.ServerID, &server); errServer != nil {
				return nil, errors.Wrapf(errServer, "Failed to get server")
			}

			person := model.NewPerson(player.Player.SID)
			if errPerson := app.PersonBySID(ctx, player.Player.SID, &person); errPerson != nil {
				return nil, errPerson
			}

			msgEmbed.Embed().
				AddField("Name", player.Player.Name).
				AddField("Server", server.ShortName).MakeFieldInline().
				AddField("steam", fmt.Sprintf("https://steamcommunity.com/profiles/%d", player.Player.SID.Int64())).
				AddField("connect", fmt.Sprintf("connect %s", server.Addr()))
		}

		return msgEmbed.Embed().Truncate().MessageEmbed, nil
	}
}

func defaultTable(writer io.Writer) *tablewriter.Table {
	tbl := tablewriter.NewWriter(writer)
	tbl.SetAutoFormatHeaders(true)
	tbl.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	tbl.SetCenterSeparator("")
	tbl.SetColumnSeparator("")
	tbl.SetRowSeparator("")
	tbl.SetHeaderLine(false)
	tbl.SetTablePadding("")
	tbl.SetAutoMergeCells(true)
	tbl.SetAlignment(tablewriter.ALIGN_LEFT)

	return tbl
}

func infString(f float64) string {
	if f == -1 {
		return "∞"
	}

	return fmt.Sprintf("%.1f", f)
}

const tableNameLen = 13

func matchASCIITable(match model.MatchResult) string {
	writerPlayers := &strings.Builder{}
	tablePlayers := defaultTable(writerPlayers)
	tablePlayers.SetHeader([]string{"T", "Name", "K", "A", "D", "KD", "KAD", "DA", "B/H/A/C"})

	players := match.TopPlayers()

	for i, player := range players {
		if i == tableNameLen {
			break
		}

		name := player.SteamID.String()
		if player.Name != "" {
			name = player.Name
		}

		if len(name) > tableNameLen {
			name = name[0:tableNameLen]
		}

		tablePlayers.Append([]string{
			player.Team.String()[0:1],
			name,
			fmt.Sprintf("%d", player.Kills),
			fmt.Sprintf("%d", player.Assists),
			fmt.Sprintf("%d", player.Deaths),
			infString(player.KDRatio()),
			infString(player.KDARatio()),
			fmt.Sprintf("%d", player.Damage),
			fmt.Sprintf("%d/%d/%d/%d", player.Backstabs, player.Headshots, player.Airshots, player.Captures),
		})
	}

	tablePlayers.Render()

	writerHealers := &strings.Builder{}
	tableHealers := defaultTable(writerPlayers)
	tableHealers.SetHeader([]string{" ", "Name", "A", "D", "Heal", "H/M", "Dr", "U/K/Q/V", "AUL"})

	for _, player := range match.Healers() {
		if player.MedicStats.Healing < store.MinMedicHealing {
			continue
		}

		name := player.SteamID.String()
		if player.Name != "" {
			name = player.Name
		}

		if len(name) > tableNameLen {
			name = name[0:tableNameLen]
		}

		tableHealers.Append([]string{
			player.Team.String()[0:1],
			name,
			fmt.Sprintf("%d", player.Assists),
			fmt.Sprintf("%d", player.Deaths),
			fmt.Sprintf("%d", player.MedicStats.Healing),
			fmt.Sprintf("%d", player.MedicStats.HealingPerMin(player.TimeEnd.Sub(player.TimeStart))),
			fmt.Sprintf("%d", player.MedicStats.Drops),
			fmt.Sprintf("%d/%d/%d/%d", player.MedicStats.ChargesUber, player.MedicStats.ChargesKritz,
				player.MedicStats.ChargesQuickfix, player.MedicStats.ChargesVacc),

			fmt.Sprintf("%.1f", player.MedicStats.AvgUberLength),
		})
	}

	tableHealers.Render()

	var topKs string

	topKillstreaks := match.TopKillstreaks(3)

	if len(topKillstreaks) > 0 {
		writerKillstreak := &strings.Builder{}
		tableKillstreaks := defaultTable(writerKillstreak)
		tableKillstreaks.SetHeader([]string{" ", "Name", "Killstreak", "Class", "Duration"})

		for _, player := range topKillstreaks {
			killstreak := player.BiggestKillstreak()

			name := player.SteamID.String()
			if player.Name != "" {
				name = player.Name
			}

			if len(name) > 17 {
				name = name[0:17]
			}

			tableKillstreaks.Append([]string{
				player.Team.String()[0:1],
				name,
				fmt.Sprintf("%d", killstreak.Killstreak),
				killstreak.PlayerClass.String(),
				(time.Duration(killstreak.Duration) * time.Second).String(),
			})
		}

		tableKillstreaks.Render()

		topKs = writerKillstreak.String()
	}

	resp := fmt.Sprintf("```%s\n%s\n%s```",
		strings.Trim(writerPlayers.String(), "\n"),
		strings.Trim(writerHealers.String(), "\n"),
		strings.Trim(topKs, "\n"))

	return resp
}

func makeOnMute(app *App) discord.CommandHandler {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	) (*discordgo.MessageEmbed, error) {
		var (
			opts     = discord.OptionMap(interaction.ApplicationCommandData().Options)
			playerID = model.StringSID(opts.String(discord.OptUserIdentifier))
			reason   model.Reason
		)

		reasonValueOpt, ok := opts[discord.OptBanReason]
		if !ok {
			return nil, errors.New("Invalid mute reason")
		}

		reason = model.Reason(reasonValueOpt.IntValue())

		duration, errDuration := util.ParseDuration(opts[discord.OptDuration].StringValue())
		if errDuration != nil {
			return nil, consts.ErrInvalidDuration
		}

		modNote := opts[discord.OptNote].StringValue()

		author, errAuthor := getDiscordAuthor(ctx, app.db, interaction)
		if errAuthor != nil {
			return nil, errAuthor
		}

		var banSteam model.BanSteam
		if errOpts := model.NewBanSteam(ctx,
			model.StringSID(author.SteamID.String()),
			playerID,
			duration,
			reason,
			reason.String(),
			modNote,
			model.Bot,
			0,
			model.NoComm,
			false,
			&banSteam,
		); errOpts != nil {
			return nil, errors.Wrapf(errOpts, "Failed to parse options")
		}

		if errBan := app.BanSteam(ctx, &banSteam); errBan != nil {
			return nil, errBan
		}

		msgEmbed := discord.NewEmbed(app.config(), "Player muted successfully")
		msgEmbed.AddFieldsSteamID(banSteam.TargetID)

		return msgEmbed.Embed().Truncate().MessageEmbed, nil
	}
}

func onBanASN(ctx context.Context, app *App, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate,
) (*discordgo.MessageEmbed, error) {
	var (
		opts     = discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
		asNumStr = opts[discord.OptASN].StringValue()
		reason   = model.Reason(opts[discord.OptBanReason].IntValue())
		targetID = model.StringSID(opts[discord.OptUserIdentifier].StringValue())
		modNote  = opts[discord.OptNote].StringValue()
		author   = model.NewPerson("")
	)

	duration, errDuration := util.ParseDuration(opts[discord.OptDuration].StringValue())
	if errDuration != nil {
		return nil, consts.ErrInvalidDuration
	}

	if errGetPersonByDiscordID := app.db.GetPersonByDiscordID(ctx, interaction.Interaction.Member.User.ID, &author); errGetPersonByDiscordID != nil {
		if errors.Is(errGetPersonByDiscordID, store.ErrNoResult) {
			return nil, errors.New("Must set steam id. See /set_steam")
		}

		return nil, errors.New("Error fetching author info")
	}

	asNum, errConv := strconv.ParseInt(asNumStr, 10, 64)
	if errConv != nil {
		return nil, errors.New("Invalid ASN")
	}

	asnRecords, errGetASNRecords := app.db.GetASNRecordsByNum(ctx, asNum)
	if errGetASNRecords != nil {
		if errors.Is(errGetASNRecords, store.ErrNoResult) {
			return nil, errors.New("No asnRecords found matching ASN")
		}

		return nil, errors.New("Error fetching asn asnRecords")
	}

	var banASN model.BanASN
	if errOpts := model.NewBanASN(ctx,
		model.StringSID(author.SteamID.String()),
		targetID,
		duration,
		reason,
		reason.String(),
		modNote,
		model.Bot,
		asNum,
		model.Banned,
		&banASN,
	); errOpts != nil {
		return nil, errors.Wrapf(errOpts, "Failed to parse options")
	}

	if errBanASN := app.BanASN(ctx, &banASN); errBanASN != nil {
		if errors.Is(errBanASN, store.ErrDuplicate) {
			return nil, errors.New("Duplicate ASN ban")
		}

		return nil, discord.ErrCommandFailed
	}

	conf := app.config()

	msgEmbed := discord.
		NewEmbed(conf, "ASN BanSteam Created Successfully").Embed().
		SetColor(conf.Discord.ColourSuccess).
		AddField("ASNum", asNumStr).
		AddField("Total IPs Blocked", fmt.Sprintf("%d", asnRecords.Hosts())).
		Truncate()

	return msgEmbed.MessageEmbed, nil
}

func onBanIP(ctx context.Context, app *App, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate,
) (*discordgo.MessageEmbed, error) {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	target := model.StringSID(opts[discord.OptUserIdentifier].StringValue())
	reason := model.Reason(opts[discord.OptBanReason].IntValue())
	cidr := opts[discord.OptCIDR].StringValue()

	_, network, errParseCIDR := net.ParseCIDR(cidr)
	if errParseCIDR != nil {
		return nil, errors.Wrap(errParseCIDR, "Invalid IP")
	}

	duration, errDuration := util.ParseDuration(opts[discord.OptDuration].StringValue())
	if errDuration != nil {
		return nil, errors.New("Invalid duration")
	}

	modNote := opts[discord.OptNote].StringValue()

	author := model.NewPerson("")
	if errGetPerson := app.db.GetPersonByDiscordID(ctx, interaction.Interaction.Member.User.ID, &author); errGetPerson != nil {
		if errors.Is(errGetPerson, store.ErrNoResult) {
			return nil, errors.New("Must set steam id. See /set_steam")
		}

		return nil, errors.New("Error fetching author info")
	}

	var banCIDR model.BanCIDR
	if errOpts := model.NewBanCIDR(ctx,
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
		return nil, errors.Wrapf(errOpts, "Failed to parse options")
	}

	if errBanNet := app.BanCIDR(ctx, &banCIDR); errBanNet != nil {
		return nil, errBanNet
	}

	state := app.state.current()
	players := state.find(findOpts{CIDR: network})

	if len(players) == 0 {
		return nil, consts.ErrPlayerNotFound
	}

	for _, player := range players {
		if errKick := app.Kick(ctx, model.Bot, player.Player.SID, author.SteamID, reason); errKick != nil {
			app.log.Error("Failed to perform kick", zap.Error(errKick))
		}
	}

	conf := app.config()

	msgEmbed := discord.
		NewEmbed(conf, "IP ban created successfully").Embed().
		SetColor(conf.Discord.ColourSuccess).
		Truncate()

	return msgEmbed.MessageEmbed, nil
}

// onBanSteam !ban <id> <duration> [reason].
func onBanSteam(ctx context.Context, app *App, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate,
) (*discordgo.MessageEmbed, error) {
	var (
		opts    = discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
		target  = opts[discord.OptUserIdentifier].StringValue()
		reason  = model.Reason(opts[discord.OptBanReason].IntValue())
		modNote = opts[discord.OptNote].StringValue()
	)

	duration, errDuration := util.ParseDuration(opts[discord.OptDuration].StringValue())
	if errDuration != nil {
		return nil, consts.ErrInvalidDuration
	}

	author, errAuthor := getDiscordAuthor(ctx, app.db, interaction)
	if errAuthor != nil {
		return nil, errAuthor
	}

	var banSteam model.BanSteam
	if errOpts := model.NewBanSteam(ctx,
		model.StringSID(author.SteamID.String()),
		model.StringSID(target),
		duration,
		reason,
		reason.String(),
		modNote,
		model.Bot,
		0,
		model.Banned,
		false,
		&banSteam,
	); errOpts != nil {
		return nil, errors.Wrapf(errOpts, "Failed to parse options")
	}

	if errBan := app.BanSteam(ctx, &banSteam); errBan != nil {
		if errors.Is(errBan, store.ErrDuplicate) {
			return nil, errors.New("Duplicate ban")
		}

		return nil, discord.ErrCommandFailed
	}

	msgEmbed, errCreate := app.createDiscordBanEmbed(ctx, banSteam)
	if errCreate != nil {
		return nil, errCreate
	}

	return msgEmbed.MessageEmbed, nil
}
