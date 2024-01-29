package discord

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
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

type discordService struct {
	du  domain.DiscordUsecase
	pu  domain.PersonUsecase
	bu  domain.BanUsecase
	su  domain.StateUsecase
	sv  domain.ServersUsecase
	cu  domain.ConfigUsecase
	nu  domain.NetworkUsecase
	wfu domain.WordFilterUsecase
	ch  domain.ChatUsecase
	mu  domain.MatchUsecase
	log *zap.Logger
}

func DiscordHandler(ctx context.Context, log *zap.Logger, du domain.DiscordUsecase, pu domain.PersonUsecase, bu domain.BanUsecase, su domain.StateUsecase, sv domain.ServersUsecase,
	cu domain.ConfigUsecase, nu domain.NetworkUsecase, wfu domain.WordFilterUsecase, mu domain.MatchUsecase, ch domain.ChatUsecase,
) error {
	handler := discordService{
		du:  du,
		pu:  pu,
		su:  su,
		bu:  bu,
		sv:  sv,
		cu:  cu,
		nu:  nu,
		mu:  mu,
		wfu: wfu,
		log: log.Named("discord"),
	}
	cmdMap := map[domain.Cmd]func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error){
		domain.CmdBan:     handler.makeOnBan(),
		domain.CmdCheck:   handler.makeOnCheck(),
		domain.CmdCSay:    handler.makeOnCSay(),
		domain.CmdFilter:  handler.makeOnFilter(),
		domain.CmdFind:    handler.makeOnFind(),
		domain.CmdHistory: handler.makeOnHistory(),
		domain.CmdKick:    handler.makeOnKick(),
		domain.CmdLog:     handler.makeOnLog(),
		domain.CmdLogs:    handler.makeOnLogs(),
		domain.CmdMute:    handler.makeOnMute(),
		// domain.CmdCheckIP:  handler.onCheckIp,
		domain.CmdPlayers:  handler.makeOnPlayers(),
		domain.CmdPSay:     handler.makeOnPSay(),
		domain.CmdSay:      handler.makeOnSay(),
		domain.CmdServers:  handler.makeOnServers(),
		domain.CmdSetSteam: handler.makeOnSetSteam(),
		domain.CmdUnban:    handler.makeOnUnban(),
		domain.CmdStats:    handler.makeOnStats(),
	}
	for k, v := range cmdMap {
		if errRegister := du.RegisterHandler(k, v); errRegister != nil {
			return errors.Join(errRegister, domain.ErrRegisterCommand)
		}
	}

	return nil
}

func (h discordService) Start() {
	h.du.Start()
}

func (h discordService) makeOnBan() func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		name := interaction.ApplicationCommandData().Options[0].Name
		switch name {
		case "steam":
			return h.onBanSteam(ctx, session, interaction)
		case "ip":
			return h.onBanIP(ctx, session, interaction)
		case "asn":
			return h.onBanASN(ctx, session, interaction)
		default:
			return nil, domain.ErrCommandFailed
		}
	}
}

func (h discordService) makeOnUnban() func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) { //nolint:maintidx
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		switch interaction.ApplicationCommandData().Options[0].Name {
		case "steam":
			return h.onUnbanSteam(ctx, session, interaction)
		case "ip":
			return nil, domain.ErrCommandFailed
			// return discord.onUnbanIP(ctx, session, interaction, response)
		case "asn":
			return h.onUnbanASN(ctx, session, interaction)
		default:
			return nil, domain.ErrCommandFailed
		}
	}
}

func (h discordService) makeOnFilter() func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) { //nolint:maintidx
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		switch interaction.ApplicationCommandData().Options[0].Name {
		case "add":
			return h.onFilterAdd(ctx, session, interaction)
		case "del":
			return h.onFilterDel(ctx, session, interaction)
		case "check":
			return h.onFilterCheck(ctx, session, interaction)
		default:
			return nil, domain.ErrCommandFailed
		}
	}
}

type BanStore interface{}

func (h discordService) makeOnCheck() func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) { //nolint:maintidx
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, //nolint:maintidx
	) (*discordgo.MessageEmbed, error) {
		opts := domain.OptionMap(interaction.ApplicationCommandData().Options)
		sid, errResolveSID := thirdparty.ResolveSID(ctx, opts[domain.OptUserIdentifier].StringValue())

		if errResolveSID != nil {
			return nil, domain.ErrInvalidSID
		}

		player := domain.NewPerson(sid)

		if errGetPlayer := h.pu.GetOrCreatePersonBySteamID(ctx, sid, &player); errGetPlayer != nil {
			return nil, domain.ErrCommandFailed
		}

		ban := domain.NewBannedPerson()
		if errGetBanBySID := h.bu.GetBanBySteamID(ctx, sid, &ban, false); errGetBanBySID != nil {
			if !errors.Is(errGetBanBySID, domain.ErrNoResult) {
				h.log.Error("Failed to get ban by steamid", zap.Error(errGetBanBySID))

				return nil, domain.ErrCommandFailed
			}
		}

		oldBans, _, errOld := h.bu.GetBansSteam(ctx, domain.SteamBansQueryFilter{
			BansQueryFilter: domain.BansQueryFilter{
				QueryFilter: domain.QueryFilter{Deleted: true},
				TargetID:    domain.StringSID(sid),
			},
		})
		if errOld != nil {
			if !errors.Is(errOld, domain.ErrNoResult) {
				h.log.Error("Failed to fetch old bans", zap.Error(errOld))
			}
		}

		bannedNets, errGetBanNet := h.bu.GetBanNetByAddress(ctx, player.IPAddr)
		if errGetBanNet != nil {
			if !errors.Is(errGetBanNet, domain.ErrNoResult) {
				h.log.Error("Failed to get ban nets by addr", zap.Error(errGetBanNet))

				return nil, domain.ErrCommandFailed
			}
		}

		var banURL string

		var (
			conf = h.cu.Config()

			authorProfile = domain.NewPerson(sid)
		)

		// TODO Show the longest remaining ban.
		if ban.BanID > 0 {
			if ban.SourceID.Valid() {
				if errGetProfile := h.pu.GetPersonBySteamID(ctx, ban.SourceID, &authorProfile); errGetProfile != nil {
					h.log.Error("Failed to load author for ban", zap.Error(errGetProfile))
				}
			}

			banURL = conf.ExtURL(ban.BanSteam)
		}

		// TODO move elsewhere
		logData, errLogs := thirdparty.LogsTFOverview(ctx, sid)
		if errLogs != nil {
			h.log.Warn("Failed to fetch logTF data", zap.Error(errLogs))
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
				if errASN := h.nu.GetASNRecordByIP(ctx, player.IPAddr, &asn); errASN != nil {
					h.log.Error("Failed to fetch ASN record", zap.Error(errASN))
				}
			}
		}()

		go func() {
			defer waitGroup.Done()

			if player.IPAddr != nil {
				if errLoc := h.nu.GetLocationRecord(ctx, player.IPAddr, &location); errLoc != nil {
					h.log.Error("Failed to fetch Location record", zap.Error(errLoc))
				}
			}
		}()

		go func() {
			defer waitGroup.Done()

			if player.IPAddr != nil {
				if errProxy := h.nu.GetProxyRecord(ctx, player.IPAddr, &proxy); errProxy != nil && !errors.Is(errProxy, domain.ErrNoResult) {
					h.log.Error("Failed to fetch proxy record", zap.Error(errProxy))
				}
			}
		}()

		waitGroup.Wait()

		return CheckMessage(player, ban, banURL, authorProfile, oldBans, bannedNets, asn, location, proxy, logData), nil
	}
}

func (h discordService) makeOnHistory() func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		switch interaction.ApplicationCommandData().Name {
		case string(domain.CmdHistoryIP):
			return h.onHistoryIP(ctx, session, interaction)
		default:
			// return discord.onHistoryChat(ctx, session, interaction, response)
			return nil, domain.ErrCommandFailed
		}
	}
}

func (h discordService) onHistoryIP(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := domain.OptionMap(interaction.ApplicationCommandData().Options[0].Options)

	steamID, errResolve := thirdparty.ResolveSID(ctx, opts[domain.OptUserIdentifier].StringValue())
	if errResolve != nil {
		return nil, domain.ErrInvalidSID
	}

	person := domain.NewPerson(steamID)
	if errPersonBySID := h.pu.GetOrCreatePersonBySteamID(ctx, steamID, &person); errPersonBySID != nil {
		return nil, domain.ErrCommandFailed
	}

	ipRecords, errGetPersonIPHist := h.nu.GetPersonIPHistory(ctx, steamID, 20)
	if errGetPersonIPHist != nil && !errors.Is(errGetPersonIPHist, domain.ErrNoResult) {
		return nil, domain.ErrCommandFailed
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

	return HistoryMessage(person), nil
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

func (h discordService) makeOnSetSteam() func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session,
		interaction *discordgo.InteractionCreate,
	) (*discordgo.MessageEmbed, error) {
		opts := domain.OptionMap(interaction.ApplicationCommandData().Options)

		steamID, errResolveSID := thirdparty.ResolveSID(ctx, opts[domain.OptUserIdentifier].StringValue())
		if errResolveSID != nil {
			return nil, domain.ErrInvalidSID
		}

		errSetSteam := h.pu.SetSteam(ctx, steamID, interaction.Member.User.ID)
		if errSetSteam != nil {
			return nil, errSetSteam
		}

		return SetSteamMessage(), nil
	}
}

func (h discordService) onUnbanSteam(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := domain.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	reason := opts[domain.OptUnbanReason].StringValue()

	steamID, errResolveSID := thirdparty.ResolveSID(ctx, opts[domain.OptUserIdentifier].StringValue())
	if errResolveSID != nil {
		return nil, domain.ErrInvalidSID
	}

	found, errUnban := h.bu.Unban(ctx, steamID, reason)
	if errUnban != nil {
		return nil, errUnban
	}

	if !found {
		return nil, domain.ErrBanDoesNotExist
	}

	var user domain.Person
	if errUser := h.pu.GetPersonBySteamID(ctx, steamID, &user); errUser != nil {
		h.log.Warn("Could not fetch unbanned Person", zap.String("steam_id", steamID.String()), zap.Error(errUser))
	}

	return UnbanMessage(user), nil
}

func (h discordService) onUnbanASN(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := domain.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	asNumStr := opts[domain.OptASN].StringValue()

	banExisted, errUnbanASN := h.bu.UnbanASN(ctx, asNumStr)
	if errUnbanASN != nil {
		if errors.Is(errUnbanASN, domain.ErrNoResult) {
			return nil, domain.ErrBanDoesNotExist
		}

		return nil, domain.ErrCommandFailed
	}

	if !banExisted {
		return nil, domain.ErrBanDoesNotExist
	}

	asNum, errConv := strconv.ParseInt(asNumStr, 10, 64)
	if errConv != nil {
		return nil, domain.ErrParseASN
	}

	asnNetworks, errGetASNRecords := h.nu.GetASNRecordsByNum(ctx, asNum)
	if errGetASNRecords != nil {
		if errors.Is(errGetASNRecords, domain.ErrNoResult) {
			return nil, domain.ErrFetchASN
		}

		return nil, domain.ErrFetchASN
	}

	return UnbanASNMessage(asNum, asnNetworks), nil
}

func (h discordService) getDiscordAuthor(ctx context.Context, interaction *discordgo.InteractionCreate) (domain.Person, error) {
	author := domain.NewPerson("")
	if errPersonByDiscordID := h.pu.GetPersonByDiscordID(ctx, interaction.Interaction.Member.User.ID, &author); errPersonByDiscordID != nil {
		if errors.Is(errPersonByDiscordID, domain.ErrNoResult) {
			return author, domain.ErrSteamUnset
		}

		return author, domain.ErrFetchSource
	}

	return author, nil
}

func (h discordService) makeOnKick() func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		var (
			opts   = domain.OptionMap(interaction.ApplicationCommandData().Options)
			target = domain.StringSID(opts[domain.OptUserIdentifier].StringValue())
			reason = domain.Reason(opts[domain.OptBanReason].IntValue())
		)

		targetSid64, errTarget := target.SID64(ctx)
		if errTarget != nil {
			return nil, domain.ErrInvalidSID
		}

		players := h.su.FindBySteamID(targetSid64)

		if len(players) == 0 {
			return nil, domain.ErrPlayerNotFound
		}

		var err error

		for _, player := range players {
			if errKick := h.su.Kick(ctx, player.Player.SID, reason); errKick != nil {
				err = errors.Join(err, errKick)

				continue
			}
		}

		return KickMessage(players), err
	}
}

func (h discordService) makeOnSay() func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		opts := domain.OptionMap(interaction.ApplicationCommandData().Options)
		serverName := opts[domain.OptServerIdentifier].StringValue()
		msg := opts[domain.OptMessage].StringValue()

		var server domain.Server
		if err := h.sv.GetServerByName(ctx, serverName, &server, false, false); err != nil {
			return nil, domain.ErrUnknownServer
		}

		if errSay := h.su.Say(ctx, server.ServerID, msg); errSay != nil {
			return nil, domain.ErrCommandFailed
		}

		return SayMessage(serverName, msg), nil
	}
}

func (h discordService) makeOnCSay() func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	) (*discordgo.MessageEmbed, error) {
		opts := domain.OptionMap(interaction.ApplicationCommandData().Options)
		serverName := opts[domain.OptServerIdentifier].StringValue()
		msg := opts[domain.OptMessage].StringValue()

		var server domain.Server
		if err := h.sv.GetServerByName(ctx, serverName, &server, false, false); err != nil {
			return nil, domain.ErrUnknownServer
		}

		if errCSay := h.su.CSay(ctx, server.ServerID, msg); errCSay != nil {
			return nil, domain.ErrCommandFailed
		}

		return CSayMessage(server.Name, msg), nil
	}
}

func (h discordService) makeOnPSay() func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		opts := domain.OptionMap(interaction.ApplicationCommandData().Options)
		player := domain.StringSID(opts[domain.OptUserIdentifier].StringValue())
		msg := opts[domain.OptMessage].StringValue()

		playerSid, errPlayerSid := player.SID64(ctx)
		if errPlayerSid != nil {
			return nil, errors.Join(errPlayerSid, domain.ErrInvalidSID)
		}

		if errPSay := h.su.PSay(ctx, playerSid, msg); errPSay != nil {
			return nil, domain.ErrCommandFailed
		}

		return PSayMessage(string(player), msg), nil
	}
}

func (h discordService) makeOnServers() func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		return ServersMessage(h.su.SortRegion(), h.cu.ExtURLRaw("/servers")), nil
	}
}

func (h discordService) makeOnPlayers() func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		opts := domain.OptionMap(interaction.ApplicationCommandData().Options)
		serverName := opts[domain.OptServerIdentifier].StringValue()

		serverStates := h.su.ByName(serverName, false)

		if len(serverStates) != 1 {
			return nil, domain.ErrUnknownServer
		}

		serverState := serverStates[0]

		var rows []string

		if len(serverState.Players) > 0 {
			sort.SliceStable(serverState.Players, func(i, j int) bool {
				return serverState.Players[i].Name < serverState.Players[j].Name
			})

			for _, player := range serverState.Players {
				var asn ip2location.ASNRecord
				if errASN := h.nu.GetASNRecordByIP(ctx, player.IP, &asn); errASN != nil {
					// Will fail for LAN ips
					h.log.Warn("Failed to get asn record", zap.Error(errASN))
				}

				var loc ip2location.LocationRecord
				if errLoc := h.nu.GetLocationRecord(ctx, player.IP, &loc); errLoc != nil {
					h.log.Warn("Failed to get location record: %v", zap.Error(errLoc))
				}

				proxyStr := ""

				var proxy ip2location.ProxyRecord
				if errGetProxyRecord := h.nu.GetProxyRecord(ctx, player.IP, &proxy); errGetProxyRecord == nil {
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

		return PlayersMessage(rows, serverState.MaxPlayers, serverState.Name), nil
	}
}

func (h discordService) onFilterAdd(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
) (*discordgo.MessageEmbed, error) {
	opts := domain.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	pattern := opts[domain.OptPattern].StringValue()
	isRegex := opts[domain.OptIsRegex].BoolValue()

	if isRegex {
		_, rxErr := regexp.Compile(pattern)
		if rxErr != nil {
			return nil, errors.Join(rxErr, domain.ErrInvalidFilterRegex)
		}
	}

	author, errAuthor := h.getDiscordAuthor(ctx, interaction)
	if errAuthor != nil {
		return nil, errAuthor
	}

	filter := domain.Filter{
		AuthorID:  author.SteamID,
		Pattern:   pattern,
		IsRegex:   isRegex,
		IsEnabled: true,
		CreatedOn: time.Now(),
		UpdatedOn: time.Now(),
	}
	if errFilterAdd := h.wfu.FilterAdd(ctx, &filter); errFilterAdd != nil {
		return nil, domain.ErrCommandFailed
	}

	return FilterAddMessage(filter), nil
}

func (h discordService) onFilterDel(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := domain.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	wordID := opts["filter"].IntValue()

	if wordID <= 0 {
		return nil, domain.ErrInvalidFilterID
	}

	var filter domain.Filter
	if errGetFilter := h.wfu.GetFilterByID(ctx, wordID, &filter); errGetFilter != nil {
		return nil, domain.ErrCommandFailed
	}

	if errDropFilter := h.wfu.DropFilter(ctx, &filter); errDropFilter != nil {
		return nil, domain.ErrCommandFailed
	}

	return FilterDelMessage(filter), nil
}

func (h discordService) onFilterCheck(_ context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := domain.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	message := opts[domain.OptMessage].StringValue()

	return FilterCheckMessage(h.wfu.Check(message)), nil
}

func (h discordService) makeOnStats() func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		name := interaction.ApplicationCommandData().Options[0].Name
		switch name {
		case "player":
			return h.onStatsPlayer(ctx, session, interaction)
		// case string(cmdStatsGlobal):
		//	return discord.onStatsGlobal(ctx, session, interaction, response)
		// case string(cmdStatsServer):
		//	return discord.onStatsServer(ctx, session, interaction, response)
		default:
			return nil, domain.ErrCommandFailed
		}
	}
}

func (h discordService) onStatsPlayer(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := domain.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	steamID, errResolveSID := thirdparty.ResolveSID(ctx, opts[domain.OptUserIdentifier].StringValue())

	if errResolveSID != nil {
		return nil, domain.ErrInvalidSID
	}

	person := domain.NewPerson(steamID)

	if errAuthor := h.pu.GetPersonBySteamID(ctx, steamID, &person); errAuthor != nil {
		return nil, errAuthor
	}

	//
	// Person, errAuthor := getDiscordAuthor(ctx, app.db, interaction)
	// if errAuthor != nil {
	//	return nil, errAuthor
	// }

	classStats, errClassStats := h.mu.StatsPlayerClass(ctx, person.SteamID)
	if errClassStats != nil {
		return nil, errors.Join(errClassStats, domain.ErrFetchClassStats)
	}

	weaponStats, errWeaponStats := h.mu.StatsPlayerWeapons(ctx, person.SteamID)
	if errWeaponStats != nil {
		return nil, errors.Join(errWeaponStats, domain.ErrFetchWeaponStats)
	}

	killstreakStats, errKillstreakStats := h.mu.StatsPlayerKillstreaks(ctx, person.SteamID)
	if errKillstreakStats != nil {
		return nil, errors.Join(errKillstreakStats, domain.ErrFetchKillstreakStats)
	}

	medicStats, errMedicStats := h.mu.StatsPlayerMedic(ctx, person.SteamID)
	if errMedicStats != nil {
		return nil, errors.Join(errMedicStats, domain.ErrFetchMedicStats)
	}

	return StatsPlayerMessage(person, h.cu.ExtURL(person), classStats, medicStats, weaponStats, killstreakStats), nil
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

func (h discordService) makeOnLogs() func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		author, errAuthor := h.getDiscordAuthor(ctx, interaction)
		if errAuthor != nil {
			return nil, errAuthor
		}

		matches, count, errMatch := h.mu.Matches(ctx, domain.MatchesQueryOpts{
			SteamID:     author.SteamID,
			QueryFilter: domain.QueryFilter{Limit: 5},
		})

		if errMatch != nil {
			return nil, domain.ErrCommandFailed
		}

		matchesWriter := &strings.Builder{}

		for _, match := range matches {
			status := ":x:"
			if match.IsWinner {
				status = ":white_check_mark:"
			}

			_, _ = matchesWriter.WriteString(fmt.Sprintf("%s [%s](%s) `%s` `%s`\n",
				status, match.Title, h.cu.ExtURL(match), match.MapName, match.TimeStart.Format(time.DateOnly)))
		}

		return LogsMessage(count, matchesWriter.String()), nil
	}
}

func (h discordService) makeOnLog() func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		opts := domain.OptionMap(interaction.ApplicationCommandData().Options)

		matchIDStr := opts[domain.OptMatchID].StringValue()

		matchID, errMatchID := uuid.FromString(matchIDStr)
		if errMatchID != nil {
			return nil, domain.ErrCommandFailed
		}

		var match domain.MatchResult

		if errMatch := h.mu.MatchGetByID(ctx, matchID, &match); errMatch != nil {
			return nil, domain.ErrCommandFailed
		}

		return MatchMessage(match, h.cu.ExtURLRaw("/log/%s", match.MatchID.String())), nil
	}
}

func (h discordService) makeOnFind() func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, i *discordgo.InteractionCreate,
	) (*discordgo.MessageEmbed, error) {
		opts := domain.OptionMap(i.ApplicationCommandData().Options)
		userIdentifier := opts[domain.OptUserIdentifier].StringValue()

		var name string

		steamID, errSteamID := steamid.StringToSID64(userIdentifier)
		if errSteamID != nil {
			name = userIdentifier
		}

		players := h.su.Find(name, steamID, nil, nil)

		if len(players) == 0 {
			return nil, domain.ErrUnknownID
		}

		var found []domain.FoundPlayer

		for _, player := range players {
			var server domain.Server
			if errServer := h.sv.GetServer(ctx, player.ServerID, &server); errServer != nil {
				return nil, errors.Join(errServer, domain.ErrGetServer)
			}

			person := domain.NewPerson(player.Player.SID)
			if errPerson := h.pu.GetOrCreatePersonBySteamID(ctx, player.Player.SID, &person); errPerson != nil {
				return nil, errors.Join(errPerson, domain.ErrFetchPerson)
			}

			found = append(found, domain.FoundPlayer{
				Player: player,
				Server: server,
			})
		}

		return FindMessage(found), nil
	}
}

func (h discordService) makeOnMute() func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	) (*discordgo.MessageEmbed, error) {
		var (
			opts     = domain.OptionMap(interaction.ApplicationCommandData().Options)
			playerID = domain.StringSID(opts.String(domain.OptUserIdentifier))
			reason   domain.Reason
		)

		reasonValueOpt, ok := opts[domain.OptBanReason]
		if !ok {
			return nil, domain.ErrReasonInvalid
		}

		reason = domain.Reason(reasonValueOpt.IntValue())

		duration, errDuration := util.ParseDuration(opts[domain.OptDuration].StringValue())
		if errDuration != nil {
			return nil, util.ErrInvalidDuration
		}

		modNote := opts[domain.OptNote].StringValue()

		author, errAuthor := h.getDiscordAuthor(ctx, interaction)
		if errAuthor != nil {
			return nil, errAuthor
		}

		var banSteam domain.BanSteam
		if errOpts := domain.NewBanSteam(ctx,
			domain.StringSID(author.SteamID.String()),
			playerID,
			duration,
			reason,
			reason.String(),
			modNote,
			domain.Bot,
			0,
			domain.NoComm,
			false,
			&banSteam,
		); errOpts != nil {
			return nil, errOpts
		}

		if errBan := h.bu.BanSteam(ctx, &banSteam); errBan != nil {
			return nil, errBan
		}

		return MuteMessage(banSteam), nil
	}
}

func (h discordService) onBanASN(ctx context.Context, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate,
) (*discordgo.MessageEmbed, error) {
	var (
		opts     = domain.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
		asNumStr = opts[domain.OptASN].StringValue()
		reason   = domain.Reason(opts[domain.OptBanReason].IntValue())
		targetID = domain.StringSID(opts[domain.OptUserIdentifier].StringValue())
		modNote  = opts[domain.OptNote].StringValue()
		author   = domain.NewPerson("")
	)

	duration, errDuration := util.ParseDuration(opts[domain.OptDuration].StringValue())
	if errDuration != nil {
		return nil, util.ErrInvalidDuration
	}

	if errGetPersonByDiscordID := h.pu.GetPersonByDiscordID(ctx, interaction.Interaction.Member.User.ID, &author); errGetPersonByDiscordID != nil {
		if errors.Is(errGetPersonByDiscordID, domain.ErrNoResult) {
			return nil, domain.ErrSteamUnset
		}

		return nil, errors.Join(errGetPersonByDiscordID, domain.ErrFetchPerson)
	}

	asNum, errConv := strconv.ParseInt(asNumStr, 10, 64)
	if errConv != nil {
		return nil, domain.ErrParseASN
	}

	asnRecords, errGetASNRecords := h.nu.GetASNRecordsByNum(ctx, asNum)
	if errGetASNRecords != nil {
		if errors.Is(errGetASNRecords, domain.ErrNoResult) {
			return nil, domain.ErrASNNoRecords
		}

		return nil, domain.ErrFetchASN
	}

	var banASN domain.BanASN
	if errOpts := domain.NewBanASN(ctx,
		domain.StringSID(author.SteamID.String()),
		targetID,
		duration,
		reason,
		reason.String(),
		modNote,
		domain.Bot,
		asNum,
		domain.Banned,
		&banASN,
	); errOpts != nil {
		return nil, errOpts
	}

	if errBanASN := h.bu.BanASN(ctx, &banASN); errBanASN != nil {
		if errors.Is(errBanASN, domain.ErrDuplicate) {
			return nil, domain.ErrDuplicateBan
		}

		return nil, domain.ErrCommandFailed
	}

	return BanASNMessage(asNum, asnRecords), nil
}

func (h discordService) onBanIP(ctx context.Context, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate,
) (*discordgo.MessageEmbed, error) {
	opts := domain.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	target := domain.StringSID(opts[domain.OptUserIdentifier].StringValue())
	reason := domain.Reason(opts[domain.OptBanReason].IntValue())
	cidr := opts[domain.OptCIDR].StringValue()

	_, network, errParseCIDR := net.ParseCIDR(cidr)
	if errParseCIDR != nil {
		return nil, errors.Join(errParseCIDR, domain.ErrInvalidIP)
	}

	duration, errDuration := util.ParseDuration(opts[domain.OptDuration].StringValue())
	if errDuration != nil {
		return nil, errDuration
	}

	modNote := opts[domain.OptNote].StringValue()

	author := domain.NewPerson("")
	if errGetPerson := h.pu.GetPersonByDiscordID(ctx, interaction.Interaction.Member.User.ID, &author); errGetPerson != nil {
		if errors.Is(errGetPerson, domain.ErrNoResult) {
			return nil, domain.ErrSteamUnset
		}

		return nil, domain.ErrFetchPerson
	}

	var banCIDR domain.BanCIDR
	if errOpts := domain.NewBanCIDR(ctx,
		domain.StringSID(author.SteamID.String()),
		target,
		duration,
		reason,
		reason.String(),
		modNote,
		domain.Bot,
		cidr,
		domain.Banned,
		&banCIDR,
	); errOpts != nil {
		return nil, errOpts
	}

	if errBanNet := h.bu.BanCIDR(ctx, &banCIDR); errBanNet != nil {
		return nil, errBanNet
	}

	players := h.su.FindByCIDR(network)

	if len(players) == 0 {
		return nil, domain.ErrPlayerNotFound
	}

	for _, player := range players {
		if errKick := h.su.Kick(ctx, player.Player.SID, reason); errKick != nil {
			h.log.Error("Failed to perform kick", zap.Error(errKick))
		}
	}

	return BanIPMessage(), nil
}

// onBanSteam !ban <id> <duration> [reason].
func (h discordService) onBanSteam(ctx context.Context, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate,
) (*discordgo.MessageEmbed, error) {
	var (
		opts    = domain.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
		target  = opts[domain.OptUserIdentifier].StringValue()
		reason  = domain.Reason(opts[domain.OptBanReason].IntValue())
		modNote = opts[domain.OptNote].StringValue()
	)

	duration, errDuration := util.ParseDuration(opts[domain.OptDuration].StringValue())
	if errDuration != nil {
		return nil, util.ErrInvalidDuration
	}

	author, errAuthor := h.getDiscordAuthor(ctx, interaction)
	if errAuthor != nil {
		return nil, errAuthor
	}

	var banSteam domain.BanSteam
	if errOpts := domain.NewBanSteam(ctx,
		domain.StringSID(author.SteamID.String()),
		domain.StringSID(target),
		duration,
		reason,
		reason.String(),
		modNote,
		domain.Bot,
		0,
		domain.Banned,
		false,
		&banSteam,
	); errOpts != nil {
		return nil, errOpts
	}

	if errBan := h.bu.BanSteam(ctx, &banSteam); errBan != nil {
		if errors.Is(errBan, domain.ErrDuplicate) {
			return nil, domain.ErrDuplicateBan
		}

		return nil, domain.ErrCommandFailed
	}

	return BanSteamResponse(banSteam), nil
}
