package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

func RegisterDiscordHandlers(env *App) error {
	cmdMap := map[Cmd]func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error){
		CmdBan:     makeOnBan(env),
		CmdCheck:   makeOnCheck(env),
		CmdCSay:    makeOnCSay(env),
		CmdFilter:  makeOnFilter(env),
		CmdFind:    makeOnFind(env),
		CmdHistory: makeOnHistory(env),
		CmdKick:    makeOnKick(env),
		CmdLog:     makeOnLog(env),
		CmdLogs:    makeOnLogs(env),
		CmdMute:    makeOnMute(env),
		// discord.CmdCheckIP:  onCheckIp,
		CmdPlayers:  makeOnPlayers(env),
		CmdPSay:     makeOnPSay(env),
		CmdSay:      makeOnSay(env),
		CmdServers:  makeOnServers(env),
		CmdSetSteam: makeOnSetSteam(env),
		CmdUnban:    makeOnUnban(env),
		CmdStats:    makeOnStats(env),
	}
	for k, v := range cmdMap {
		if errRegister := env.discord.RegisterHandler(k, v); errRegister != nil {
			return errors.Join(errRegister, errors.New("Failed to register discord command"))
		}
	}

	return nil
}

func makeOnBan(env *App) func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		name := interaction.ApplicationCommandData().Options[0].Name
		switch name {
		case "steam":
			return onBanSteam(ctx, env, session, interaction)
		case "ip":
			return onBanIP(ctx, env, session, interaction)
		case "asn":
			return onBanASN(ctx, env, session, interaction)
		default:
			return nil, ErrCommandFailed
		}
	}
}

func makeOnUnban(env *App) func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		switch interaction.ApplicationCommandData().Options[0].Name {
		case "steam":
			return onUnbanSteam(ctx, env, session, interaction)
		case "ip":
			return nil, ErrCommandFailed
			// return discord.onUnbanIP(ctx, session, interaction, response)
		case "asn":
			return onUnbanASN(ctx, env, session, interaction)
		default:
			return nil, ErrCommandFailed
		}
	}
}

func makeOnFilter(env *App) func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		switch interaction.ApplicationCommandData().Options[0].Name {
		case "add":
			return onFilterAdd(ctx, env, session, interaction)
		case "del":
			return onFilterDel(ctx, env, session, interaction)
		case "check":
			return onFilterCheck(ctx, env, session, interaction)
		default:
			return nil, ErrCommandFailed
		}
	}
}

type BanStore interface{}

func makeOnCheck(env *App) func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) { //nolint:maintidx
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, //nolint:maintidx
	) (*discordgo.MessageEmbed, error) {
		opts := OptionMap(interaction.ApplicationCommandData().Options)
		sid, errResolveSID := thirdparty.ResolveSID(ctx, opts[OptUserIdentifier].StringValue())

		if errResolveSID != nil {
			return nil, errs.ErrInvalidSID
		}

		player := model.NewPerson(sid)

		if errGetPlayer := env.Store().GetOrCreatePersonBySteamID(ctx, sid, &player); errGetPlayer != nil {
			return nil, ErrCommandFailed
		}

		ban := model.NewBannedPerson()
		if errGetBanBySID := env.Store().GetBanBySteamID(ctx, sid, &ban, false); errGetBanBySID != nil {
			if !errors.Is(errGetBanBySID, errs.ErrNoResult) {
				env.Log().Error("Failed to get ban by steamid", zap.Error(errGetBanBySID))

				return nil, ErrCommandFailed
			}
		}

		oldBans, _, errOld := env.Store().GetBansSteam(ctx, model.SteamBansQueryFilter{
			BansQueryFilter: model.BansQueryFilter{
				QueryFilter: model.QueryFilter{Deleted: true},
				TargetID:    model.StringSID(sid),
			},
		})
		if errOld != nil {
			if !errors.Is(errOld, errs.ErrNoResult) {
				env.Log().Error("Failed to fetch old bans", zap.Error(errOld))
			}
		}

		bannedNets, errGetBanNet := env.Store().GetBanNetByAddress(ctx, player.IPAddr)
		if errGetBanNet != nil {
			if !errors.Is(errGetBanNet, errs.ErrNoResult) {
				env.Log().Error("Failed to get ban nets by addr", zap.Error(errGetBanNet))

				return nil, ErrCommandFailed
			}
		}

		var banUrl string

		var (
			conf = env.Config()

			authorProfile = model.NewPerson(sid)
		)

		// TODO Show the longest remaining ban.
		if ban.BanID > 0 {
			if ban.SourceID.Valid() {
				if errGetProfile := env.Store().GetPersonBySteamID(ctx, ban.SourceID, &authorProfile); errGetProfile != nil {
					env.Log().Error("Failed to load author for ban", zap.Error(errGetProfile))
				} else {
				}
			}
			banUrl = conf.ExtURL(ban.BanSteam)
		}

		// TODO move elsewhere
		logData, errLogs := thirdparty.LogsTFOverview(ctx, sid)
		if errLogs != nil {
			env.Log().Warn("Failed to fetch logTF data", zap.Error(errLogs))
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
				if errASN := env.Store().GetASNRecordByIP(ctx, player.IPAddr, &asn); errASN != nil {
					env.Log().Error("Failed to fetch ASN record", zap.Error(errASN))
				}
			}
		}()

		go func() {
			defer waitGroup.Done()

			if player.IPAddr != nil {
				if errLoc := env.Store().GetLocationRecord(ctx, player.IPAddr, &location); errLoc != nil {
					env.Log().Error("Failed to fetch Location record", zap.Error(errLoc))
				}
			}
		}()

		go func() {
			defer waitGroup.Done()

			if player.IPAddr != nil {
				if errProxy := env.Store().GetProxyRecord(ctx, player.IPAddr, &proxy); errProxy != nil && !errors.Is(errProxy, errs.ErrNoResult) {
					env.Log().Error("Failed to fetch proxy record", zap.Error(errProxy))
				}
			}
		}()

		waitGroup.Wait()

		return discord.CheckMessage(player, ban, banUrl, authorProfile, oldBans, bannedNets, asn, location, proxy, logData), nil
	}
}

func makeOnHistory(env *App) func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		switch interaction.ApplicationCommandData().Name {
		case string(CmdHistoryIP):
			return onHistoryIP(ctx, env, session, interaction)
		default:
			// return discord.onHistoryChat(ctx, session, interaction, response)
			return nil, ErrCommandFailed
		}
	}
}

func onHistoryIP(ctx context.Context, env *App, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := OptionMap(interaction.ApplicationCommandData().Options[0].Options)

	steamID, errResolve := thirdparty.ResolveSID(ctx, opts[OptUserIdentifier].StringValue())
	if errResolve != nil {
		return nil, errs.ErrInvalidSID
	}

	person := model.NewPerson(steamID)
	if errPersonBySID := env.Store().GetOrCreatePersonBySteamID(ctx, steamID, &person); errPersonBySID != nil {
		return nil, ErrCommandFailed
	}

	ipRecords, errGetPersonIPHist := env.Store().GetPersonIPHistory(ctx, steamID, 20)
	if errGetPersonIPHist != nil && !errors.Is(errGetPersonIPHist, errs.ErrNoResult) {
		return nil, ErrCommandFailed
	}

	lastIP := net.IP{}

	for _, ipRecord := range ipRecords {
		// TODO Join query for connections and geoip lookup data
		// addField(embed, ipRecord.IPAddr.String(), fmt.Sprintf("%s %s %s %s %s %s %s %s", Config.FmtTimeShort(ipRecord.CreatedOn), ipRecord.CC,
		//	ipRecord.CityName, ipRecord.ASName, ipRecord.ISP, ipRecord.UsageType, ipRecord.Threat, ipRecord.DomainUsed))
		// lastIP = ipRecord.IPAddr
		if ipRecord.IPAddr.Equal(lastIP) {
			continue
		}
	}

	return discord.HistoryMessage(person), nil
}

//
// func (discord *Discord) onHistoryChat(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, response *botResponse) error {
//	steamId, errResolveSID := resolveSID(ctx, interaction.Data.Options[0].Options[0].Value.(string))
//	if errResolveSID != nil {
//		return consts.ErrInvalidSID
//	}
//	Person := model.NewPerson(steamId)
//	if errPersonBySID := PersonBySID(ctx, discord.database, steamId, "", &Person); errPersonBySID != nil {
//		return errCommandFailed
//	}
//	chatHistory, errChatHistory := discord.database.GetChatHistory(ctx, steamId, 25)
//	if errChatHistory != nil && !errors.Is(errChatHistory, db.ErrNoResult) {
//		return errCommandFailed
//	}
//	if errors.Is(errChatHistory, db.ErrNoResult) {
//		return errors.New("No chat history found")
//	}
//	var lines []string
//	for _, sayEvent := range chatHistory {
//		lines = append(lines, fmt.Sprintf("%s: %s", Config.FmtTimeShort(sayEvent.CreatedOn), sayEvent.Msg))
//	}
//	embed := respOk(response, fmt.Sprintf("Chat History of: %s", Person.PersonaName))
//	embed.Description = strings.Join(lines, "\n")
//	return nil
// }

func makeOnSetSteam(env *App) func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session,
		interaction *discordgo.InteractionCreate,
	) (*discordgo.MessageEmbed, error) {
		opts := OptionMap(interaction.ApplicationCommandData().Options)

		steamID, errResolveSID := thirdparty.ResolveSID(ctx, opts[OptUserIdentifier].StringValue())
		if errResolveSID != nil {
			return nil, errs.ErrInvalidSID
		}

		errSetSteam := env.SetSteam(ctx, steamID, interaction.Member.User.ID)
		if errSetSteam != nil {
			return nil, errSetSteam
		}

		return discord.SetSteamMessage(), nil
	}
}

func onUnbanSteam(ctx context.Context, env *App, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	reason := opts[OptUnbanReason].StringValue()

	steamID, errResolveSID := thirdparty.ResolveSID(ctx, opts[OptUserIdentifier].StringValue())
	if errResolveSID != nil {
		return nil, errs.ErrInvalidSID
	}

	found, errUnban := env.Unban(ctx, steamID, reason)
	if errUnban != nil {
		return nil, errUnban
	}

	if !found {
		return nil, errors.New("No ban found")
	}

	var user model.Person
	if errUser := env.Store().GetPersonBySteamID(ctx, steamID, &user); errUser != nil {
		env.Log().Warn("Could not fetch unbanned Person", zap.String("steam_id", steamID.String()), zap.Error(errUser))
	} else {
	}
	return discord.UnbanMessage(user), nil
}

func onUnbanASN(ctx context.Context, env *App, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	asNumStr := opts[OptASN].StringValue()

	banExisted, errUnbanASN := env.UnbanASN(ctx, asNumStr)
	if errUnbanASN != nil {
		if errors.Is(errUnbanASN, errs.ErrNoResult) {
			return nil, errors.New("Ban for ASN does not exist")
		}

		return nil, ErrCommandFailed
	}

	if !banExisted {
		return nil, errors.New("Ban for ASN does not exist")
	}

	asNum, errConv := strconv.ParseInt(asNumStr, 10, 64)
	if errConv != nil {
		return nil, errors.New("Invalid ASN")
	}

	asnNetworks, errGetASNRecords := env.Store().GetASNRecordsByNum(ctx, asNum)
	if errGetASNRecords != nil {
		if errors.Is(errGetASNRecords, errs.ErrNoResult) {
			return nil, errors.New("No asnNetworks found matching ASN")
		}

		return nil, errors.New("Error fetching asn asnNetworks")
	}

	return discord.UnbanASNMessage(asNum, asnNetworks), nil
}

func getDiscordAuthor(ctx context.Context, db store.Stores, interaction *discordgo.InteractionCreate) (model.Person, error) {
	author := model.NewPerson("")
	if errPersonByDiscordID := db.GetPersonByDiscordID(ctx, interaction.Interaction.Member.User.ID, &author); errPersonByDiscordID != nil {
		if errors.Is(errPersonByDiscordID, errs.ErrNoResult) {
			return author, errors.New("Must set steam id. See /set_steam")
		}

		return author, errors.New("Error fetching author info")
	}

	return author, nil
}

func makeOnKick(env *App) func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		var (
			opts   = OptionMap(interaction.ApplicationCommandData().Options)
			target = model.StringSID(opts[OptUserIdentifier].StringValue())
			reason = model.Reason(opts[OptBanReason].IntValue())
		)

		targetSid64, errTarget := target.SID64(ctx)
		if errTarget != nil {
			return nil, errs.ErrInvalidSID
		}

		currentState := env.State()
		players := currentState.Find("", targetSid64, nil, nil)

		if len(players) == 0 {
			return nil, errs.ErrPlayerNotFound
		}

		var err error

		for _, player := range players {
			if errKick := state.Kick(ctx, env.state, player.Player.SID, reason); errKick != nil {
				err = errors.Join(err, errKick)

				continue
			}
		}

		return discord.KickMessage(players), err
	}
}

func makeOnSay(env *App) func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		opts := OptionMap(interaction.ApplicationCommandData().Options)
		serverName := opts[OptServerIdentifier].StringValue()
		msg := opts[OptMessage].StringValue()

		var server model.Server
		if err := env.Store().GetServerByName(ctx, serverName, &server, false, false); err != nil {
			return nil, errs.ErrUnknownServer
		}

		if errSay := state.Say(ctx, env.State(), server.ServerID, msg); errSay != nil {
			return nil, ErrCommandFailed
		}

		return discord.SayMessage(serverName, msg), nil
	}
}

func makeOnCSay(env *App) func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	) (*discordgo.MessageEmbed, error) {
		opts := OptionMap(interaction.ApplicationCommandData().Options)
		serverName := opts[OptServerIdentifier].StringValue()
		msg := opts[OptMessage].StringValue()

		var server model.Server
		if err := env.Store().GetServerByName(ctx, serverName, &server, false, false); err != nil {
			return nil, errs.ErrUnknownServer
		}

		if errCSay := state.CSay(ctx, env.State(), server.ServerID, msg); errCSay != nil {
			return nil, ErrCommandFailed
		}

		return discord.CSayMessage(server.Name, msg), nil
	}
}

func makeOnPSay(env *App) func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		opts := OptionMap(interaction.ApplicationCommandData().Options)
		player := model.StringSID(opts[OptUserIdentifier].StringValue())
		msg := opts[OptMessage].StringValue()

		playerSid, errPlayerSid := player.SID64(ctx)
		if errPlayerSid != nil {
			return nil, errors.Join(errPlayerSid, errors.New("Failed to get player sid"))
		}

		if errPSay := state.PSay(ctx, env.State(), playerSid, msg); errPSay != nil {
			return nil, ErrCommandFailed
		}

		return discord.PSayMessage(string(player), msg), nil
	}
}

func makeOnServers(env *App) func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		return discord.ServersMessage(env.State().SortRegion(), env.Config().ExtURLRaw("/servers")), nil
	}
}

func makeOnPlayers(env *App) func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		opts := OptionMap(interaction.ApplicationCommandData().Options)
		serverName := opts[OptServerIdentifier].StringValue()
		currentState := env.State()
		serverStates := currentState.ByName(serverName, false)

		if len(serverStates) != 1 {
			return nil, errs.ErrUnknownServer
		}

		serverState := serverStates[0]

		var rows []string

		if len(serverState.Players) > 0 {
			sort.SliceStable(serverState.Players, func(i, j int) bool {
				return serverState.Players[i].Name < serverState.Players[j].Name
			})

			for _, player := range serverState.Players {
				var asn ip2location.ASNRecord
				if errASN := env.Store().GetASNRecordByIP(ctx, player.IP, &asn); errASN != nil {
					// Will fail for LAN ips
					env.Log().Warn("Failed to get asn record", zap.Error(errASN))
				}

				var loc ip2location.LocationRecord
				if errLoc := env.Store().GetLocationRecord(ctx, player.IP, &loc); errLoc != nil {
					env.Log().Warn("Failed to get location record: %v", zap.Error(errLoc))
				}

				proxyStr := ""

				var proxy ip2location.ProxyRecord
				if errGetProxyRecord := env.Store().GetProxyRecord(ctx, player.IP, &proxy); errGetProxyRecord == nil {
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
		}
		return discord.PlayersMessage(rows, serverState.MaxPlayers, serverState.Name), nil
	}
}

func onFilterAdd(ctx context.Context, env *App, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
) (*discordgo.MessageEmbed, error) {
	opts := OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	pattern := opts[OptPattern].StringValue()
	isRegex := opts[OptIsRegex].BoolValue()

	if isRegex {
		_, rxErr := regexp.Compile(pattern)
		if rxErr != nil {
			return nil, errors.Join(fmt.Errorf("Invalid regular expression: %v", rxErr))
		}
	}

	author, errAuthor := getDiscordAuthor(ctx, env.Store(), interaction)
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
	if errFilterAdd := env.FilterAdd(ctx, &filter); errFilterAdd != nil {
		return nil, ErrCommandFailed
	}

	return discord.FilterAddMessage(filter), nil
}

func onFilterDel(ctx context.Context, env *App, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	wordID := opts["filter"].IntValue()

	if wordID <= 0 {
		return nil, errors.New("Invalid filter id")
	}

	var filter model.Filter
	if errGetFilter := env.Store().GetFilterByID(ctx, wordID, &filter); errGetFilter != nil {
		return nil, ErrCommandFailed
	}

	if errDropFilter := env.Store().DropFilter(ctx, &filter); errDropFilter != nil {
		return nil, ErrCommandFailed
	}

	return discord.FilterDelMessage(filter), nil
}

func onFilterCheck(_ context.Context, env *App, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	message := opts[OptMessage].StringValue()

	return discord.FilterCheckMessage(env.WordFilters().Check(message)), nil
}

func makeOnStats(env *App) func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		name := interaction.ApplicationCommandData().Options[0].Name
		switch name {
		case "player":
			return onStatsPlayer(ctx, env, session, interaction)
		// case string(cmdStatsGlobal):
		//	return discord.onStatsGlobal(ctx, session, interaction, response)
		// case string(cmdStatsServer):
		//	return discord.onStatsServer(ctx, session, interaction, response)
		default:
			return nil, ErrCommandFailed
		}
	}
}

func onStatsPlayer(ctx context.Context, env *App, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	steamID, errResolveSID := thirdparty.ResolveSID(ctx, opts[OptUserIdentifier].StringValue())

	if errResolveSID != nil {
		return nil, errs.ErrInvalidSID
	}

	person := model.NewPerson(steamID)

	if errAuthor := env.Store().GetPersonBySteamID(ctx, steamID, &person); errAuthor != nil {
		return nil, errAuthor
	}

	//
	// Person, errAuthor := getDiscordAuthor(ctx, app.db, interaction)
	// if errAuthor != nil {
	//	return nil, errAuthor
	// }

	classStats, errClassStats := env.Store().StatsPlayerClass(ctx, person.SteamID)
	if errClassStats != nil {
		return nil, errors.Join(errClassStats, errors.New("Failed to fetch class stats"))
	}

	weaponStats, errWeaponStats := env.Store().StatsPlayerWeapons(ctx, person.SteamID)
	if errWeaponStats != nil {
		return nil, errors.Join(errWeaponStats, errors.New("Failed to fetch weapon stats"))
	}

	killstreakStats, errKillstreakStats := env.Store().StatsPlayerKillstreaks(ctx, person.SteamID)
	if errKillstreakStats != nil {
		return nil, errors.Join(errKillstreakStats, errors.New("Failed to fetch killstreak stats"))
	}

	medicStats, errMedicStats := env.Store().StatsPlayerMedic(ctx, person.SteamID)
	if errMedicStats != nil {
		return nil, errors.Join(errMedicStats, errors.New("Failed to fetch medic stats"))
	}

	return discord.StatsPlayerMessage(person, env.Config().ExtURL(person), classStats, medicStats, weaponStats, killstreakStats), nil
}

//	func (discord *discord) onStatsServer(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, response *botResponse) error {
//		serverIdStr := interaction.Data.Options[0].Options[0].Value.(string)
//		var (
//			server model.ServerStore
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
//		embed := respOk(response, fmt.Sprintf("ServerStore stats for %s ", server.ShortName))
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

func makeOnLogs(env *App) func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		author, errAuthor := getDiscordAuthor(ctx, env.Store(), interaction)
		if errAuthor != nil {
			return nil, errAuthor
		}

		matches, count, errMatch := env.Store().Matches(ctx, model.MatchesQueryOpts{
			SteamID:     author.SteamID,
			QueryFilter: model.QueryFilter{Limit: 5},
		})

		if errMatch != nil {
			return nil, ErrCommandFailed
		}

		conf := env.Config()

		matchesWriter := &strings.Builder{}

		for _, match := range matches {
			status := ":x:"
			if match.IsWinner {
				status = ":white_check_mark:"
			}

			_, _ = matchesWriter.WriteString(fmt.Sprintf("%s [%s](%s) `%s` `%s`\n",
				status, match.Title, conf.ExtURL(match), match.MapName, match.TimeStart.Format(time.DateOnly)))
		}

		return discord.LogsMessage(count, matchesWriter.String()), nil
	}
}

func makeOnLog(env *App) func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		opts := OptionMap(interaction.ApplicationCommandData().Options)

		matchIDStr := opts[OptMatchID].StringValue()

		matchID, errMatchID := uuid.FromString(matchIDStr)
		if errMatchID != nil {
			return nil, ErrCommandFailed
		}

		var match model.MatchResult

		if errMatch := env.Store().MatchGetByID(ctx, matchID, &match); errMatch != nil {
			return nil, ErrCommandFailed
		}

		return discord.MatchMessage(match, env.Config().ExtURLRaw("/log/%s", match.MatchID.String())), nil
	}
}

func makeOnFind(env *App) func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, i *discordgo.InteractionCreate,
	) (*discordgo.MessageEmbed, error) {
		opts := OptionMap(i.ApplicationCommandData().Options)
		userIdentifier := opts[OptUserIdentifier].StringValue()

		var name string

		steamID, errSteamID := steamid.StringToSID64(userIdentifier)
		if errSteamID != nil {
			name = userIdentifier
		}

		currentState := env.State()
		players := currentState.Find(name, steamID, nil, nil)

		if len(players) == 0 {
			return nil, errs.ErrUnknownID
		}

		var found []discord.FoundPlayer
		for _, player := range players {
			var server model.Server
			if errServer := env.Store().GetServer(ctx, player.ServerID, &server); errServer != nil {
				return nil, errors.Join(errServer, errors.New("Failed to get server"))
			}

			person := model.NewPerson(player.Player.SID)
			if errPerson := env.Store().GetOrCreatePersonBySteamID(ctx, player.Player.SID, &person); errPerson != nil {
				return nil, errors.Join(errPerson, errors.New("Failed to get player instance"))
			}

			found = append(found, discord.FoundPlayer{
				Player: player,
				Server: server,
			})
		}

		return discord.FindMessage(found), nil
	}
}

func makeOnMute(env *App) func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	) (*discordgo.MessageEmbed, error) {
		var (
			opts     = OptionMap(interaction.ApplicationCommandData().Options)
			playerID = model.StringSID(opts.String(OptUserIdentifier))
			reason   model.Reason
		)

		reasonValueOpt, ok := opts[OptBanReason]
		if !ok {
			return nil, errors.New("Invalid mute reason")
		}

		reason = model.Reason(reasonValueOpt.IntValue())

		duration, errDuration := util.ParseDuration(opts[OptDuration].StringValue())
		if errDuration != nil {
			return nil, util.ErrInvalidDuration
		}

		modNote := opts[OptNote].StringValue()

		author, errAuthor := getDiscordAuthor(ctx, env.Store(), interaction)
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
			return nil, errors.Join(errOpts, errors.New("Failed to parse options"))
		}

		if errBan := env.BanSteam(ctx, &banSteam); errBan != nil {
			return nil, errBan
		}

		return discord.MuteMessage(banSteam), nil
	}
}

func onBanASN(ctx context.Context, env *App, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate,
) (*discordgo.MessageEmbed, error) {
	var (
		opts     = OptionMap(interaction.ApplicationCommandData().Options[0].Options)
		asNumStr = opts[OptASN].StringValue()
		reason   = model.Reason(opts[OptBanReason].IntValue())
		targetID = model.StringSID(opts[OptUserIdentifier].StringValue())
		modNote  = opts[OptNote].StringValue()
		author   = model.NewPerson("")
	)

	duration, errDuration := util.ParseDuration(opts[OptDuration].StringValue())
	if errDuration != nil {
		return nil, util.ErrInvalidDuration
	}

	if errGetPersonByDiscordID := env.Store().GetPersonByDiscordID(ctx, interaction.Interaction.Member.User.ID, &author); errGetPersonByDiscordID != nil {
		if errors.Is(errGetPersonByDiscordID, errs.ErrNoResult) {
			return nil, errors.New("Must set steam id. See /set_steam")
		}

		return nil, errors.New("Error fetching author info")
	}

	asNum, errConv := strconv.ParseInt(asNumStr, 10, 64)
	if errConv != nil {
		return nil, errors.New("Invalid ASN")
	}

	asnRecords, errGetASNRecords := env.Store().GetASNRecordsByNum(ctx, asNum)
	if errGetASNRecords != nil {
		if errors.Is(errGetASNRecords, errs.ErrNoResult) {
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
		return nil, errors.Join(errOpts, errors.New("Failed to parse options"))
	}

	if errBanASN := env.BanASN(ctx, &banASN); errBanASN != nil {
		if errors.Is(errBanASN, errs.ErrDuplicate) {
			return nil, errors.New("Duplicate ASN ban")
		}

		return nil, ErrCommandFailed
	}

	return discord.BanASNMessage(asNum, asnRecords), nil
}

func onBanIP(ctx context.Context, env *App, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate,
) (*discordgo.MessageEmbed, error) {
	opts := OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	target := model.StringSID(opts[OptUserIdentifier].StringValue())
	reason := model.Reason(opts[OptBanReason].IntValue())
	cidr := opts[OptCIDR].StringValue()

	_, network, errParseCIDR := net.ParseCIDR(cidr)
	if errParseCIDR != nil {
		return nil, errors.Join(errParseCIDR, errors.New("Invalid IP"))
	}

	duration, errDuration := util.ParseDuration(opts[OptDuration].StringValue())
	if errDuration != nil {
		return nil, errors.New("Invalid duration")
	}

	modNote := opts[OptNote].StringValue()

	author := model.NewPerson("")
	if errGetPerson := env.Store().GetPersonByDiscordID(ctx, interaction.Interaction.Member.User.ID, &author); errGetPerson != nil {
		if errors.Is(errGetPerson, errs.ErrNoResult) {
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
		return nil, errors.Join(errOpts, errors.New("Failed to parse options"))
	}

	if errBanNet := env.BanCIDR(ctx, &banCIDR); errBanNet != nil {
		return nil, errBanNet
	}

	currentState := env.State()
	players := currentState.Find("", "", nil, network)

	if len(players) == 0 {
		return nil, errs.ErrPlayerNotFound
	}

	for _, player := range players {
		if errKick := state.Kick(ctx, env.State(), player.Player.SID, reason); errKick != nil {
			env.Log().Error("Failed to perform kick", zap.Error(errKick))
		}
	}

	return discord.BanIPMessage(), nil
}

// onBanSteam !ban <id> <duration> [reason].
func onBanSteam(ctx context.Context, env *App, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate,
) (*discordgo.MessageEmbed, error) {
	var (
		opts    = OptionMap(interaction.ApplicationCommandData().Options[0].Options)
		target  = opts[OptUserIdentifier].StringValue()
		reason  = model.Reason(opts[OptBanReason].IntValue())
		modNote = opts[OptNote].StringValue()
	)

	duration, errDuration := util.ParseDuration(opts[OptDuration].StringValue())
	if errDuration != nil {
		return nil, util.ErrInvalidDuration
	}

	author, errAuthor := getDiscordAuthor(ctx, env.Store(), interaction)
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
		return nil, errors.Join(errOpts, errors.New("Failed to parse options"))
	}

	if errBan := env.BanSteam(ctx, &banSteam); errBan != nil {
		if errors.Is(errBan, errs.ErrDuplicate) {
			return nil, errors.New("Duplicate ban")
		}

		return nil, ErrCommandFailed
	}

	return discord.BanSteamResponse(banSteam), nil
}
