package servers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/discordgo-lipstick/bot"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

func RegisterDiscordCommands(bot *bot.Bot, state State, persons person.Provider, servers Servers, network network.Networks, config config.Config) {
	handler := DiscordHandler{state: state, persons: persons, servers: servers, network: network, config: config}

	bot.MustRegisterHandler("find", &discordgo.ApplicationCommand{
		Name:                     "find",
		DMPermission:             &discord.DmPerms,
		DefaultMemberPermissions: &discord.ModPerms,
		Description:              "Find a user on any of the servers",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        discord.OptUserIdentifier,
				Description: "SteamID in any format OR profile url",
				Required:    true,
			},
		},
	}, handler.onFind)

	bot.MustRegisterHandler("players", &discordgo.ApplicationCommand{
		Name:                     "players",
		DMPermission:             &discord.DmPerms,
		DefaultMemberPermissions: &discord.ModPerms,
		Description:              "Show a table of the current players on the server",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        discord.OptServerIdentifier,
				Description: "Short server name",
				Required:    true,
			},
		},
	}, handler.onPlayers)

	bot.MustRegisterHandler("kick", &discordgo.ApplicationCommand{
		Name:                     "kick",
		DMPermission:             &discord.DmPerms,
		DefaultMemberPermissions: &discord.ModPerms,
		Description:              "Kick a user from any server they are playing on",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        discord.OptUserIdentifier,
				Description: "SteamID in any format OR profile url",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "reason",
				Description: "Reason for the kick (shown to users on kick)",
				Required:    true,
			},
		},
	}, handler.onKick)

	bot.MustRegisterHandler("psay", &discordgo.ApplicationCommand{
		Name:                     "psay",
		Description:              "Privately message a player",
		DMPermission:             &discord.DmPerms,
		DefaultMemberPermissions: &discord.ModPerms,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        discord.OptUserIdentifier,
				Description: "SteamID in any format OR profile url",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        discord.OptMessage,
				Description: "Message to send",
				Required:    true,
			},
		},
	}, handler.onPSay)

	bot.MustRegisterHandler("csay", &discordgo.ApplicationCommand{
		Name:                     "csay",
		Description:              "Send a centered message to the whole server",
		DMPermission:             &discord.DmPerms,
		DefaultMemberPermissions: &discord.ModPerms,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        discord.OptServerIdentifier,
				Description: "Short server name or `*` for all",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        discord.OptMessage,
				Description: "Message to send",
				Required:    true,
			},
		},
	}, handler.onCSay)

	bot.MustRegisterHandler("say", &discordgo.ApplicationCommand{
		Name:                     "say",
		Description:              "Send a console message to the whole server",
		DMPermission:             &discord.DmPerms,
		DefaultMemberPermissions: &discord.ModPerms,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        discord.OptServerIdentifier,
				Description: "Short server name",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        discord.OptMessage,
				Description: "Message to send",
				Required:    true,
			},
		},
	}, handler.onSay)

	bot.MustRegisterHandler("servers", &discordgo.ApplicationCommand{
		Name:                     "servers",
		Description:              "Show the high level status of all servers",
		DefaultMemberPermissions: &discord.UserPerms,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "full",
				Description: "Return the full status output including server versions and tags",
			},
		},
	}, handler.onServers)
}

type DiscordHandler struct {
	state   State
	persons person.Provider
	servers Servers
	network network.Networks
	config  config.Config
}

func NewDiscordHandler(state State) *DiscordHandler {
	return &DiscordHandler{
		state: state,
	}
}

func (d DiscordHandler) onFind(ctx context.Context, _ *discordgo.Session, interation *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := bot.OptionMap(interation.ApplicationCommandData().Options)
	userIdentifier := opts[discord.OptUserIdentifier].StringValue()

	var name string

	steamID, errSteamID := steamid.Resolve(ctx, userIdentifier)
	if errSteamID != nil || !steamID.Valid() {
		// Search for name instead on error
		name = userIdentifier
	}

	players := d.state.Find(FindOpts{SteamID: steamID, Name: name})
	if len(players) == 0 {
		return nil, steamid.ErrDecodeSID
	}

	found := make([]discordFoundPlayer, len(players))

	for index, player := range players {
		server, errServer := d.servers.Server(ctx, player.ServerID)
		if errServer != nil {
			return nil, errors.Join(errServer, ErrGetServer)
		}

		_, errPerson := d.persons.GetOrCreatePersonBySteamID(ctx, player.Player.SID)
		if errPerson != nil {
			return nil, errPerson
		}

		found[index] = discordFoundPlayer{Player: player, Server: server}
	}

	return discordFindMessage(found), nil
}

func (d DiscordHandler) onKick(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := bot.OptionMap(interaction.ApplicationCommandData().Options)
	// reason = ban.Reason(opts[discord.OptBanReason].IntValue())

	target, errTarget := steamid.Resolve(ctx, opts[discord.OptUserIdentifier].StringValue())
	if errTarget != nil || !target.Valid() {
		return nil, steamid.ErrDecodeSID
	}

	players := d.state.Find(FindOpts{SteamID: target})

	if len(players) == 0 {
		return nil, ErrPlayerNotFound
	}

	var err error

	for _, player := range players {
		if errKick := d.state.Kick(ctx, player.Player.SID, ""); errKick != nil {
			err = errors.Join(err, errKick)

			continue
		}
	}

	return discordKickMessage(players), err
}

func (d DiscordHandler) onSay(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := bot.OptionMap(interaction.ApplicationCommandData().Options)
	serverName := opts[discord.OptServerIdentifier].StringValue()
	msg := opts[discord.OptMessage].StringValue()

	server, err := d.servers.GetByName(ctx, serverName)
	if err != nil {
		return nil, ErrUnknownServer
	}

	if errSay := d.state.Say(ctx, server.ServerID, msg); errSay != nil {
		return nil, discord.ErrCommandFailed
	}

	return discordSayMessage(serverName, msg), nil
}

func (d DiscordHandler) onCSay(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := bot.OptionMap(interaction.ApplicationCommandData().Options)
	serverName := opts[discord.OptServerIdentifier].StringValue()
	msg := opts[discord.OptMessage].StringValue()

	server, err := d.servers.GetByName(ctx, serverName)
	if err != nil {
		return nil, ErrUnknownServer
	}

	if errCSay := d.state.CSay(ctx, server.ServerID, msg); errCSay != nil {
		return nil, discord.ErrCommandFailed
	}

	return discordCSayMessage(server.Name, msg), nil
}

func (d DiscordHandler) onPSay(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := bot.OptionMap(interaction.ApplicationCommandData().Options)
	msg := opts[discord.OptMessage].StringValue()

	playerSid, errPlayerSid := steamid.Resolve(ctx, opts[discord.OptUserIdentifier].StringValue())
	if errPlayerSid != nil || playerSid.Valid() {
		return nil, errors.Join(errPlayerSid, steamid.ErrDecodeSID)
	}

	if errPSay := d.state.PSay(ctx, playerSid, msg); errPSay != nil {
		return nil, discord.ErrCommandFailed
	}

	return discordPSayMessage(playerSid, msg), nil
}

func (d DiscordHandler) onServers(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return discordServersMessage(d.state.SortRegion(), d.config.ExtURLRaw("/servers")), nil
}

func (d DiscordHandler) onPlayers(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := bot.OptionMap(interaction.ApplicationCommandData().Options)
	serverName := opts[discord.OptServerIdentifier].StringValue()
	serverStates := d.state.ByName(serverName, false)

	if len(serverStates) != 1 {
		return nil, ErrUnknownServer
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

			network, errNetwork := d.network.QueryNetwork(ctx, address)
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

	return discordPlayersMessage(rows, serverState.MaxPlayers, serverState.Name), nil
}

type discordFoundPlayer struct {
	Player PlayerServerInfo
	Server Server
}

func discordFindMessage(found []discordFoundPlayer) *discordgo.MessageEmbed {
	msgEmbed := discord.NewEmbed("Player(s) Found")
	for _, info := range found {
		msgEmbed.Embed().
			AddField("Name", info.Player.Player.Name).
			AddField("Server", info.Server.ShortName).MakeFieldInline().
			AddField("steam", fmt.Sprintf("https://steamcommunity.com/profiles/%d", info.Player.Player.SID.Int64())).
			AddField("connect", "connect "+info.Server.Addr())
	}

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func discordSayMessage(server string, msg string) *discordgo.MessageEmbed {
	return discord.NewEmbed("Sent chat message successfully").Embed().
		SetColor(discord.ColourSuccess).
		AddField("ServerStore", server).
		AddField("Message", msg).
		Truncate().MessageEmbed
}

func discordCSayMessage(server string, msg string) *discordgo.MessageEmbed {
	return discord.NewEmbed("Sent console message successfully").Embed().
		SetColor(discord.ColourSuccess).
		AddField("ServerStore", server).
		AddField("Message", msg).
		Truncate().MessageEmbed
}

func discordPSayMessage(player steamid.SteamID, msg string) *discordgo.MessageEmbed {
	return discord.NewEmbed("Sent private message successfully").Embed().
		SetColor(discord.ColourSuccess).
		AddField("Player", string(player.Steam(false))).
		AddField("Message", msg).
		Truncate().MessageEmbed
}

// TODO dont hardcode.
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

func discordServersMessage(currentStateRegion map[string][]ServerState, serversURL string) *discordgo.MessageEmbed {
	var (
		stats       = map[string]float64{}
		used, total = 0, 0
		regionNames = make([]string, 9)
	)

	msgEmbed := discord.NewEmbed("Current ServerStore Populations")
	msgEmbed.Embed().SetURL(serversURL)

	for k := range currentStateRegion {
		regionNames = append(regionNames, k)
	}

	sort.Strings(regionNames)

	for _, region := range regionNames {
		var counts []string

		for _, curState := range currentStateRegion[region] {
			if !curState.Enabled {
				continue
			}

			_, ok := stats[region]
			if !ok {
				stats[region] = 0
				stats[region+"total"] = 0
			}

			maxPlayers := curState.MaxPlayers - curState.ReservedSlots
			if maxPlayers <= 0 {
				maxPlayers = 32 - curState.ReservedSlots
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
		msgEmbed.Embed().SetColor(discord.ColourError)
		msgEmbed.Embed().SetDescription("No server states available")
	}

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func discordPlayersMessage(rows []string, maxPlayers int, serverName string) *discordgo.MessageEmbed {
	msgEmbed := discord.NewEmbed(fmt.Sprintf("%s Current Players: %d / %d", serverName, len(rows), maxPlayers))
	if len(rows) > 0 {
		msgEmbed.Embed().SetDescription(strings.Join(rows, "\n"))
		msgEmbed.Embed().SetColor(discord.ColourSuccess)
	} else {
		msgEmbed.Embed().SetDescription("No players :(")
		msgEmbed.Embed().SetColor(discord.ColourError)
	}

	return msgEmbed.Embed().MessageEmbed
}

func discordKickMessage(players []PlayerServerInfo) *discordgo.MessageEmbed {
	msgEmbed := discord.NewEmbed("Users Kicked")
	for _, player := range players {
		msgEmbed.Embed().AddField("Name", player.Player.Name)
		msgEmbed.AddFieldsSteamID(player.Player.SID)
	}

	return msgEmbed.Embed().Truncate().MessageEmbed
}

// func NewInGameReportResponse(report ban.ReportWithAuthor, reportURL string, author domain.PersonInfo, authorURL string, _ string) *discordgo.MessageEmbed {
// 	msgEmbed := message.NewEmbed("New User Report Created")
// 	msgEmbed.
// 		Embed().
// 		SetDescription(report.Description).
// 		SetColor(message.ColourSuccess).
// 		SetURL(reportURL)

// 	msgEmbed.AddAuthorPersonInfo(author, authorURL)

// 	name := author.GetName()

// 	if name == "" {
// 		name = report.TargetID.String()
// 	}

// 	msgEmbed.
// 		Embed().
// 		AddField("Subject", name).
// 		AddField("Reason", report.Reason.String())

// 	if report.ReasonText != "" {
// 		msgEmbed.Embed().AddField("Custom Reason", report.ReasonText)
// 	}

// 	return msgEmbed.AddFieldsSteamID(report.TargetID).Embed().Truncate().MessageEmbed
// }

// func discordPingModMessage(author domain.PersonInfo, authorURL string, reason string, server Server, roleID string, connect string) *discordgo.MessageEmbed {
// 	msgEmbed := message.NewEmbed("New User In-Game Report")
// 	msgEmbed.
// 		Embed().
// 		SetDescription(fmt.Sprintf("%s | <@&%s>", reason, roleID)).
// 		AddField("server", server.Name)

// 	if connect != "" {
// 		msgEmbed.Embed().AddField("connect", connect)
// 	}

// 	msgEmbed.AddAuthorPersonInfo(author, authorURL).Embed().Truncate()

// 	return msgEmbed.Embed().MessageEmbed
// }
