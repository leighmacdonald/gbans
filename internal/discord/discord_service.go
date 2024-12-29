package discord

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/datetime"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type discordService struct {
	anticheat   domain.AntiCheatUsecase
	discord     domain.DiscordUsecase
	persons     domain.PersonUsecase
	bansSteam   domain.BanSteamUsecase
	bansNet     domain.BanNetUsecase
	bansASN     domain.BanASNUsecase
	state       domain.StateUsecase
	servers     domain.ServersUsecase
	config      domain.ConfigUsecase
	network     domain.NetworkUsecase
	wordFilters domain.WordFilterUsecase
	matches     domain.MatchUsecase
}

func NewDiscordHandler(discordUsecase domain.DiscordUsecase, persons domain.PersonUsecase,
	bansSteam domain.BanSteamUsecase, state domain.StateUsecase, servers domain.ServersUsecase,
	config domain.ConfigUsecase, network domain.NetworkUsecase, wordFilters domain.WordFilterUsecase,
	matches domain.MatchUsecase, bansNet domain.BanNetUsecase, bansASN domain.BanASNUsecase,
	anticheat domain.AntiCheatUsecase,
) domain.ServiceStarter {
	handler := &discordService{
		discord:     discordUsecase,
		persons:     persons,
		state:       state,
		bansSteam:   bansSteam,
		servers:     servers,
		config:      config,
		network:     network,
		matches:     matches,
		wordFilters: wordFilters,
		bansNet:     bansNet,
		bansASN:     bansASN,
		anticheat:   anticheat,
	}

	return handler
}

func (h discordService) Start(_ context.Context) {
	cmdMap := map[domain.Cmd]func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error){
		domain.CmdBan:     h.makeOnBan(),
		domain.CmdCheck:   h.makeOnCheck(),
		domain.CmdCSay:    h.makeOnCSay(),
		domain.CmdFilter:  h.makeOnFilter(),
		domain.CmdFind:    h.makeOnFind(),
		domain.CmdHistory: h.makeOnHistory(),
		domain.CmdKick:    h.makeOnKick(),
		domain.CmdLog:     h.makeOnLog(),
		domain.CmdLogs:    h.makeOnLogs(),
		domain.CmdMute:    h.makeOnMute(),
		// domain.CmdCheckIP:  h.onCheckIp,
		domain.CmdPlayers: h.makeOnPlayers(),
		domain.CmdPSay:    h.makeOnPSay(),
		domain.CmdSay:     h.makeOnSay(),
		domain.CmdServers: h.makeOnServers(),
		domain.CmdUnban:   h.makeOnUnban(),
		domain.CmdStats:   h.makeOnStats(),
		domain.CmdAC:      h.makeOnAC(),
	}

	for k, v := range cmdMap {
		if errRegister := h.discord.RegisterHandler(k, v); errRegister != nil {
			slog.Error("Failed to register handler", log.ErrAttr(errRegister))
		}
	}
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
		sid, errResolveSID := steamid.Resolve(ctx, opts[domain.OptUserIdentifier].StringValue())

		if errResolveSID != nil || !sid.Valid() {
			return nil, domain.ErrInvalidSID
		}

		player, errGetPlayer := h.persons.GetOrCreatePersonBySteamID(ctx, nil, sid)
		if errGetPlayer != nil {
			return nil, domain.ErrCommandFailed
		}

		ban, errGetBanBySID := h.bansSteam.GetBySteamID(ctx, sid, false, true)
		if errGetBanBySID != nil {
			if !errors.Is(errGetBanBySID, domain.ErrNoResult) {
				slog.Error("Failed to get ban by steamid", log.ErrAttr(errGetBanBySID))

				return nil, domain.ErrCommandFailed
			}
		}

		oldBans, errOld := h.bansSteam.Get(ctx, domain.SteamBansQueryFilter{})
		if errOld != nil {
			if !errors.Is(errOld, domain.ErrNoResult) {
				slog.Error("Failed to fetch old bans", log.ErrAttr(errOld))
			}
		}

		bannedNets, errGetBanNet := h.bansNet.GetByAddress(ctx, player.IPAddr)
		if errGetBanNet != nil {
			if !errors.Is(errGetBanNet, domain.ErrNoResult) {
				slog.Error("Failed to get ban nets by addr", log.ErrAttr(errGetBanNet))

				return nil, domain.ErrCommandFailed
			}
		}

		var banURL string

		var (
			conf = h.config.Config()

			authorProfile domain.Person
		)

		// TODO Show the longest remaining ban.
		if ban.BanID > 0 {
			if ban.SourceID.Valid() {
				ap, errGetProfile := h.persons.GetPersonBySteamID(ctx, nil, ban.SourceID)
				if errGetProfile != nil {
					slog.Error("Failed to load author for ban", log.ErrAttr(errGetProfile))
				} else {
					authorProfile = ap
				}
			}

			banURL = conf.ExtURL(ban.BanSteam)
		}

		// TODO move elsewhere
		logData, errLogs := thirdparty.LogsTFOverview(ctx, sid)
		if errLogs != nil {
			slog.Warn("Failed to fetch logTF data", log.ErrAttr(errLogs))
		}

		network, errNetwork := h.network.QueryNetwork(ctx, player.IPAddr)
		if errNetwork != nil {
			slog.Error("Failed to query network details")
		}

		return CheckMessage(player, ban, banURL, authorProfile, oldBans, bannedNets, network.Asn, network.Location, network.Proxy, logData), nil
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

	steamID, errResolve := steamid.Resolve(ctx, opts[domain.OptUserIdentifier].StringValue())
	if errResolve != nil || !steamID.Valid() {
		return nil, domain.ErrInvalidSID
	}

	person, errPersonBySID := h.persons.GetOrCreatePersonBySteamID(ctx, nil, steamID)
	if errPersonBySID != nil {
		return nil, domain.ErrCommandFailed
	}

	// TODO actually show record

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

func (h discordService) onUnbanSteam(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := domain.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	reason := opts[domain.OptUnbanReason].StringValue()

	author, err := h.getDiscordAuthor(ctx, interaction)
	if err != nil {
		return nil, err
	}

	steamID, errResolveSID := steamid.Resolve(ctx, opts[domain.OptUserIdentifier].StringValue())
	if errResolveSID != nil || !steamID.Valid() {
		return nil, domain.ErrInvalidSID
	}

	found, errUnban := h.bansSteam.Unban(ctx, steamID, reason, author.ToUserProfile())
	if errUnban != nil {
		return nil, errUnban
	}

	if !found {
		return nil, domain.ErrBanDoesNotExist
	}

	user, errUser := h.persons.GetPersonBySteamID(ctx, nil, steamID)
	if errUser != nil {
		slog.Warn("Could not fetch unbanned Person", slog.String("steam_id", steamID.String()), log.ErrAttr(errUser))
	}

	return UnbanMessage(h.config, user), nil
}

func (h discordService) onUnbanASN(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := domain.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	asNumStr := opts[domain.OptASN].StringValue()

	banExisted, errUnbanASN := h.bansASN.Unban(ctx, asNumStr, "")
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

	return UnbanASNMessage(asNum), nil
}

func (h discordService) getDiscordAuthor(ctx context.Context, interaction *discordgo.InteractionCreate) (domain.Person, error) {
	author, errPersonByDiscordID := h.persons.GetPersonByDiscordID(ctx, interaction.Member.User.ID)
	if errPersonByDiscordID != nil {
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
			reason = domain.Reason(opts[domain.OptBanReason].IntValue())
		)

		target, errTarget := steamid.Resolve(ctx, opts[domain.OptUserIdentifier].StringValue())
		if errTarget != nil || !target.Valid() {
			return nil, domain.ErrInvalidSID
		}

		players := h.state.FindBySteamID(target)

		if len(players) == 0 {
			return nil, domain.ErrPlayerNotFound
		}

		var err error

		for _, player := range players {
			if errKick := h.state.Kick(ctx, player.Player.SID, reason); errKick != nil {
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
		if err := h.servers.GetByName(ctx, serverName, &server, false, false); err != nil {
			return nil, domain.ErrUnknownServer
		}

		if errSay := h.state.Say(ctx, server.ServerID, msg); errSay != nil {
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
		if err := h.servers.GetByName(ctx, serverName, &server, false, false); err != nil {
			return nil, domain.ErrUnknownServer
		}

		if errCSay := h.state.CSay(ctx, server.ServerID, msg); errCSay != nil {
			return nil, domain.ErrCommandFailed
		}

		return CSayMessage(server.Name, msg), nil
	}
}

func (h discordService) makeOnPSay() func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		opts := domain.OptionMap(interaction.ApplicationCommandData().Options)
		msg := opts[domain.OptMessage].StringValue()

		playerSid, errPlayerSid := steamid.Resolve(ctx, opts[domain.OptUserIdentifier].StringValue())
		if errPlayerSid != nil || playerSid.Valid() {
			return nil, errors.Join(errPlayerSid, domain.ErrInvalidSID)
		}

		if errPSay := h.state.PSay(ctx, playerSid, msg); errPSay != nil {
			return nil, domain.ErrCommandFailed
		}

		return PSayMessage(playerSid, msg), nil
	}
}

func (h discordService) makeOnServers() func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		return ServersMessage(h.state.SortRegion(), h.config.ExtURLRaw("/servers")), nil
	}
}

func (h discordService) makeOnPlayers() func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		opts := domain.OptionMap(interaction.ApplicationCommandData().Options)
		serverName := opts[domain.OptServerIdentifier].StringValue()

		serverStates := h.state.ByName(serverName, false)

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
				address, errIP := netip.ParseAddr(player.IP.String())
				if errIP != nil {
					slog.Error("Failed to parse player ip", log.ErrAttr(errIP))

					continue
				}

				network, errNetwork := h.network.QueryNetwork(ctx, address)
				if errNetwork != nil {
					slog.Error("Failed to get network info", log.ErrAttr(errNetwork))

					continue
				}

				flag := ""
				if network.Location.CountryCode != "" {
					flag = fmt.Sprintf(":flag_%s: ", strings.ToLower(network.Location.CountryCode))
				}

				proxyStr := ""
				if network.Proxy.ProxyType != "" {
					proxyStr = fmt.Sprintf("Threat: %s | %s | %s", network.Proxy.ProxyType, network.Proxy.Threat, network.Proxy.UsageType)
				}

				rows = append(rows, fmt.Sprintf("%s`%s` `%3dms` [%s](https://steamcommunity.com/profiles/%s)%s",
					flag, player.SID.Steam3(), player.Ping, player.Name, player.SID.String(), proxyStr))
			}
		}

		return PlayersMessage(rows, serverState.MaxPlayers, serverState.Name), nil
	}
}

func (h discordService) onFilterCheck(_ context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := domain.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	message := opts[domain.OptMessage].StringValue()

	return FilterCheckMessage(h.wordFilters.Check(message)), nil
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

func (h discordService) makeOnAC() func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		name := interaction.ApplicationCommandData().Options[0].Name
		switch name {
		case "player":
			return h.onACPlayer(ctx, session, interaction)
		// case string(cmdStatsGlobal):
		//	return discord.onStatsGlobal(ctx, session, interaction, response)
		// case string(cmdStatsServer):
		//	return discord.onStatsServer(ctx, session, interaction, response)
		default:
			return nil, domain.ErrCommandFailed
		}
	}
}

func (h discordService) onACPlayer(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := domain.OptionMap(interaction.ApplicationCommandData().Options[0].Options)

	steamID, errResolveSID := steamid.Resolve(ctx, opts[domain.OptUserIdentifier].StringValue())
	if errResolveSID != nil || !steamID.Valid() {
		return nil, domain.ErrInvalidSID
	}

	person, errAuthor := h.persons.GetPersonBySteamID(ctx, nil, steamID)
	if errAuthor != nil {
		return nil, errAuthor
	}

	logs, errQuery := h.anticheat.Query(ctx, domain.AnticheatQuery{SteamID: steamID.String()})
	if errQuery != nil {
		return nil, errQuery
	}

	return ACPlayerLogs(h.config, person, logs), nil
}

func (h discordService) onStatsPlayer(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := domain.OptionMap(interaction.ApplicationCommandData().Options[0].Options)

	steamID, errResolveSID := steamid.Resolve(ctx, opts[domain.OptUserIdentifier].StringValue())
	if errResolveSID != nil || !steamID.Valid() {
		return nil, domain.ErrInvalidSID
	}

	person, errAuthor := h.persons.GetPersonBySteamID(ctx, nil, steamID)
	if errAuthor != nil {
		return nil, errAuthor
	}

	//
	// Person, errAuthor := getDiscordAuthor(ctx, app.db, interaction)
	// if errAuthor != nil {
	//	return nil, errAuthor
	// }

	classStats, errClassStats := h.matches.StatsPlayerClass(ctx, person.SteamID)
	if errClassStats != nil {
		return nil, errors.Join(errClassStats, domain.ErrFetchClassStats)
	}

	weaponStats, errWeaponStats := h.matches.StatsPlayerWeapons(ctx, person.SteamID)
	if errWeaponStats != nil {
		return nil, errors.Join(errWeaponStats, domain.ErrFetchWeaponStats)
	}

	killstreakStats, errKillstreakStats := h.matches.StatsPlayerKillstreaks(ctx, person.SteamID)
	if errKillstreakStats != nil {
		return nil, errors.Join(errKillstreakStats, domain.ErrFetchKillstreakStats)
	}

	medicStats, errMedicStats := h.matches.StatsPlayerMedic(ctx, person.SteamID)
	if errMedicStats != nil {
		return nil, errors.Join(errMedicStats, domain.ErrFetchMedicStats)
	}

	return StatsPlayerMessage(person, h.config.ExtURL(person), classStats, medicStats, weaponStats, killstreakStats), nil
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

		matches, count, errMatch := h.matches.Matches(ctx, domain.MatchesQueryOpts{
			SteamID:     author.SteamID.String(),
			QueryFilter: domain.QueryFilter{Limit: 5},
		})

		if errMatch != nil {
			return nil, domain.ErrCommandFailed
		}

		matchesWriter := &strings.Builder{}

		for _, match := range matches {
			status := ":x:"

			for _, player := range match.Players {
				if player.SteamID == author.SteamID && match.Winner == player.Team {
					status = ":white_check_mark:"
				}
			}

			if _, err := fmt.Fprintf(matchesWriter, "%s [%s](%s) `%s` `%s`\n",
				status, match.Title, h.config.ExtURLRaw("/match/%s", match.MatchID.String()), match.MapName, match.TimeStart.Format(time.DateOnly)); err != nil {
				slog.Error("Failed to write match line", log.ErrAttr(err))

				continue
			}
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

		if errMatch := h.matches.MatchGetByID(ctx, matchID, &match); errMatch != nil {
			return nil, domain.ErrCommandFailed
		}

		return MatchMessage(match, h.config.ExtURLRaw("/log/%s", match.MatchID.String())), nil
	}
}

func (h discordService) makeOnFind() func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, i *discordgo.InteractionCreate,
	) (*discordgo.MessageEmbed, error) {
		opts := domain.OptionMap(i.ApplicationCommandData().Options)
		userIdentifier := opts[domain.OptUserIdentifier].StringValue()

		var name string

		steamID, errSteamID := steamid.Resolve(ctx, userIdentifier)
		if errSteamID != nil || !steamID.Valid() {
			// Search for name instead on error
			name = userIdentifier
		}

		players := h.state.Find(name, steamID, nil, nil)

		if len(players) == 0 {
			return nil, domain.ErrUnknownID
		}

		var found []domain.FoundPlayer

		for _, player := range players {
			server, errServer := h.servers.Server(ctx, player.ServerID)
			if errServer != nil {
				return nil, errors.Join(errServer, domain.ErrGetServer)
			}

			_, errPerson := h.persons.GetOrCreatePersonBySteamID(ctx, nil, player.Player.SID)
			if errPerson != nil {
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
		opts := domain.OptionMap(interaction.ApplicationCommandData().Options)

		playerID, errPlayerID := steamid.Resolve(ctx, opts.String(domain.OptUserIdentifier))
		if errPlayerID != nil || !playerID.Valid() {
			return nil, domain.ErrInvalidSID
		}

		reasonValueOpt, ok := opts[domain.OptBanReason]
		if !ok {
			return nil, domain.ErrReasonInvalid
		}

		author, errAuthor := h.getDiscordAuthor(ctx, interaction)
		if errAuthor != nil {
			return nil, errAuthor
		}

		banSteam, errBan := h.bansSteam.Ban(ctx, author.ToUserProfile(), domain.Bot, domain.RequestBanSteamCreate{
			SourceIDField: domain.SourceIDField{SourceID: author.SteamID.String()},
			TargetIDField: domain.TargetIDField{TargetID: opts.String(domain.OptUserIdentifier)},
			Duration:      opts[domain.OptDuration].StringValue(),
			BanType:       domain.NoComm,
			Reason:        domain.Reason(reasonValueOpt.IntValue()),
			ReasonText:    "",
			Note:          opts[domain.OptNote].StringValue(),
		})
		if errBan != nil {
			return nil, errBan
		}

		return MuteMessage(banSteam), nil
	}
}

func (h discordService) onBanASN(ctx context.Context, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate,
) (*discordgo.MessageEmbed, error) {
	opts := domain.OptionMap(interaction.ApplicationCommandData().Options[0].Options)

	author, errGetPersonByDiscordID := h.persons.GetPersonByDiscordID(ctx, interaction.Member.User.ID)
	if errGetPersonByDiscordID != nil {
		if errors.Is(errGetPersonByDiscordID, domain.ErrNoResult) {
			return nil, domain.ErrSteamUnset
		}

		return nil, errors.Join(errGetPersonByDiscordID, domain.ErrFetchPerson)
	}

	req := domain.RequestBanASNCreate{
		SourceIDField: domain.SourceIDField{SourceID: author.SteamID.String()},
		TargetIDField: domain.TargetIDField{TargetID: opts[domain.OptUserIdentifier].StringValue()},
		Note:          opts[domain.OptNote].StringValue(),
		Reason:        domain.Reason(opts[domain.OptBanReason].IntValue()),
		ReasonText:    "",
		ASNum:         opts[domain.OptASN].IntValue(),
		Duration:      opts[domain.OptDuration].StringValue(),
	}

	bannedPerson, errBanASN := h.bansASN.Ban(ctx, req)
	if errBanASN != nil {
		if errors.Is(errBanASN, domain.ErrDuplicate) {
			return nil, domain.ErrDuplicateBan
		}

		return nil, domain.ErrCommandFailed
	}

	return BanASNMessage(bannedPerson, h.config.Config()), nil
}

func (h discordService) onBanIP(ctx context.Context, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate,
) (*discordgo.MessageEmbed, error) {
	opts := domain.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	reason := domain.Reason(opts[domain.OptBanReason].IntValue())
	cidr := opts[domain.OptCIDR].StringValue()

	targetID, errTargetID := steamid.Resolve(ctx, opts[domain.OptUserIdentifier].StringValue())
	if errTargetID != nil || !targetID.Valid() {
		return nil, domain.ErrInvalidSID
	}

	_, network, errParseCIDR := net.ParseCIDR(cidr)
	if errParseCIDR != nil {
		return nil, errors.Join(errParseCIDR, domain.ErrNetworkInvalidIP)
	}

	duration, errDuration := datetime.ParseDuration(opts[domain.OptDuration].StringValue())
	if errDuration != nil {
		return nil, errDuration
	}

	modNote := opts[domain.OptNote].StringValue()

	author, errGetPerson := h.persons.GetPersonByDiscordID(ctx, interaction.Member.User.ID)
	if errGetPerson != nil {
		if errors.Is(errGetPerson, domain.ErrNoResult) {
			return nil, domain.ErrSteamUnset
		}

		return nil, domain.ErrFetchPerson
	}

	var banCIDR domain.BanCIDR
	if errOpts := domain.NewBanCIDR(author.SteamID, targetID, duration, reason, reason.String(), modNote, domain.Bot,
		cidr, domain.Banned, &banCIDR); errOpts != nil {
		return nil, errOpts
	}

	if errBanNet := h.bansNet.Ban(ctx, &banCIDR); errBanNet != nil {
		return nil, errBanNet
	}

	players := h.state.FindByCIDR(network)

	if len(players) == 0 {
		return nil, domain.ErrPlayerNotFound
	}

	for _, player := range players {
		if errKick := h.state.Kick(ctx, player.Player.SID, reason); errKick != nil {
			slog.Error("Failed to perform kick", log.ErrAttr(errKick))
		}
	}

	return BanIPMessage(), nil
}

// onBanSteam !ban <id> <duration> [reason].
func (h discordService) onBanSteam(ctx context.Context, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate,
) (*discordgo.MessageEmbed, error) {
	opts := domain.OptionMap(interaction.ApplicationCommandData().Options[0].Options)

	author, errAuthor := h.getDiscordAuthor(ctx, interaction)
	if errAuthor != nil {
		return nil, errAuthor
	}

	banSteam, errBan := h.bansSteam.Ban(ctx, author.ToUserProfile(), domain.Bot, domain.RequestBanSteamCreate{
		SourceIDField: domain.SourceIDField{SourceID: author.SteamID.String()},
		TargetIDField: domain.TargetIDField{TargetID: opts[domain.OptUserIdentifier].StringValue()},
		Duration:      opts[domain.OptDuration].StringValue(),
		BanType:       domain.Banned,
		Reason:        domain.Reason(opts[domain.OptBanReason].IntValue()),
		ReasonText:    "",
		Note:          opts[domain.OptNote].StringValue(),
	})
	if errBan != nil {
		if errors.Is(errBan, domain.ErrDuplicate) {
			return nil, domain.ErrDuplicateBan
		}

		return nil, domain.ErrCommandFailed
	}

	return BanSteamResponse(banSteam), nil
}
