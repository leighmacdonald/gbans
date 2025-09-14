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
	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/discord/helper"
	"github.com/leighmacdonald/gbans/internal/discord/message"
	"github.com/leighmacdonald/gbans/internal/domain"
	banDomain "github.com/leighmacdonald/gbans/internal/domain/ban"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type PersonProvider interface {
	// FIXME Retuning a interface for now.
	GetOrCreatePersonBySteamID(ctx context.Context, transaction pgx.Tx, sid64 steamid.SteamID) (domain.PersonCore, error)
}

var slashCommands = []*discordgo.ApplicationCommand{
	{
		Name:                     "find",
		DMPermission:             &helper.DmPerms,
		DefaultMemberPermissions: &helper.ModPerms,
		Description:              "Find a user on any of the servers",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        helper.OptUserIdentifier,
				Description: "SteamID in any format OR profile url",
				Required:    true,
			},
		},
	},
	{
		Name:                     "players",
		DMPermission:             &helper.DmPerms,
		DefaultMemberPermissions: &helper.ModPerms,
		Description:              "Show a table of the current players on the server",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        helper.OptServerIdentifier,
				Description: "Short server name",
				Required:    true,
			},
		},
	},
	{
		Name:                     "kick",
		DMPermission:             &helper.DmPerms,
		DefaultMemberPermissions: &helper.ModPerms,
		Description:              "Kick a user from any server they are playing on",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        helper.OptUserIdentifier,
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
	},

	{
		Name:                     "psay",
		Description:              "Privately message a player",
		DMPermission:             &helper.DmPerms,
		DefaultMemberPermissions: &helper.ModPerms,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        helper.OptUserIdentifier,
				Description: "SteamID in any format OR profile url",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        helper.OptMessage,
				Description: "Message to send",
				Required:    true,
			},
		},
	},
	{
		Name:                     "csay",
		Description:              "Send a centered message to the whole server",
		DMPermission:             &helper.DmPerms,
		DefaultMemberPermissions: &helper.ModPerms,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        helper.OptServerIdentifier,
				Description: "Short server name or `*` for all",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        helper.OptMessage,
				Description: "Message to send",
				Required:    true,
			},
		},
	},
	{
		Name:                     "say",
		Description:              "Send a console message to the whole server",
		DMPermission:             &helper.DmPerms,
		DefaultMemberPermissions: &helper.ModPerms,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        helper.OptServerIdentifier,
				Description: "Short server name",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        helper.OptMessage,
				Description: "Message to send",
				Required:    true,
			},
		},
	},
	{
		Name:                     "servers",
		Description:              "Show the high level status of all servers",
		DefaultMemberPermissions: &helper.UserPerms,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "full",
				Description: "Return the full status output including server versions and tags",
			},
		},
	},
}

type DiscordHandler struct {
	state   StateUsecase
	persons PersonProvider
	servers ServersUsecase
	network network.NetworkUsecase
	config  config.Config
}

func NewDiscordHandler(state StateUsecase) *DiscordHandler {
	return &DiscordHandler{
		state: state,
	}
}

func (d DiscordHandler) onFind(ctx context.Context, sessiin *discordgo.Session, interation *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := helper.OptionMap(interation.ApplicationCommandData().Options)
	userIdentifier := opts[helper.OptUserIdentifier].StringValue()

	var name string

	steamID, errSteamID := steamid.Resolve(ctx, userIdentifier)
	if errSteamID != nil || !steamID.Valid() {
		// Search for name instead on error
		name = userIdentifier
	}

	players := d.state.Find(name, steamID, nil, nil)

	if len(players) == 0 {
		return nil, domain.ErrUnknownID
	}

	var found []FoundPlayer

	for _, player := range players {
		server, errServer := d.servers.Server(ctx, player.ServerID)
		if errServer != nil {
			return nil, errors.Join(errServer, domain.ErrGetServer)
		}

		_, errPerson := d.persons.GetOrCreatePersonBySteamID(ctx, nil, player.Player.SID)
		if errPerson != nil {
			return nil, errPerson
		}

		found = append(found, FoundPlayer{Player: player, Server: server})
	}

	return FindMessage(found), nil
}

func (d DiscordHandler) onKick(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	var (
		opts   = helper.OptionMap(interaction.ApplicationCommandData().Options)
		reason = banDomain.Reason(opts[helper.OptBanReason].IntValue())
	)

	target, errTarget := steamid.Resolve(ctx, opts[helper.OptUserIdentifier].StringValue())
	if errTarget != nil || !target.Valid() {
		return nil, domain.ErrInvalidSID
	}

	players := d.state.FindBySteamID(target)

	if len(players) == 0 {
		return nil, domain.ErrPlayerNotFound
	}

	var err error

	for _, player := range players {
		if errKick := d.state.Kick(ctx, player.Player.SID, reason.String()); errKick != nil {
			err = errors.Join(err, errKick)

			continue
		}
	}

	return KickMessage(players), err
}

func (d DiscordHandler) onSay(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := helper.OptionMap(interaction.ApplicationCommandData().Options)
	serverName := opts[helper.OptServerIdentifier].StringValue()
	msg := opts[helper.OptMessage].StringValue()

	var server Server
	if err := d.servers.GetByName(ctx, serverName, &server, false, false); err != nil {
		return nil, ErrUnknownServer
	}

	if errSay := d.state.Say(ctx, server.ServerID, msg); errSay != nil {
		return nil, helper.ErrCommandFailed
	}

	return SayMessage(serverName, msg), nil
}

func (d DiscordHandler) onCSay(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := helper.OptionMap(interaction.ApplicationCommandData().Options)
	serverName := opts[helper.OptServerIdentifier].StringValue()
	msg := opts[helper.OptMessage].StringValue()

	var server Server
	if err := d.servers.GetByName(ctx, serverName, &server, false, false); err != nil {
		return nil, ErrUnknownServer
	}

	if errCSay := d.state.CSay(ctx, server.ServerID, msg); errCSay != nil {
		return nil, helper.ErrCommandFailed
	}

	return CSayMessage(server.Name, msg), nil
}

func (d DiscordHandler) onPSay(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := helper.OptionMap(interaction.ApplicationCommandData().Options)
	msg := opts[helper.OptMessage].StringValue()

	playerSid, errPlayerSid := steamid.Resolve(ctx, opts[helper.OptUserIdentifier].StringValue())
	if errPlayerSid != nil || playerSid.Valid() {
		return nil, errors.Join(errPlayerSid, domain.ErrInvalidSID)
	}

	if errPSay := d.state.PSay(ctx, playerSid, msg); errPSay != nil {
		return nil, helper.ErrCommandFailed
	}

	return PSayMessage(playerSid, msg), nil
}

func (d DiscordHandler) makeOnServers(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return ServersMessage(d.state.SortRegion(), d.config.ExtURLRaw("/servers")), nil
}

func (d DiscordHandler) makeOnPlayers(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := helper.OptionMap(interaction.ApplicationCommandData().Options)
	serverName := opts[helper.OptServerIdentifier].StringValue()
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

	return PlayersMessage(rows, serverState.MaxPlayers, serverState.Name), nil
}

type FoundPlayer struct {
	Player PlayerServerInfo
	Server Server
}

func FindMessage(found []FoundPlayer) *discordgo.MessageEmbed {
	msgEmbed := message.NewEmbed("Player(s) Found")
	for _, info := range found {
		msgEmbed.Embed().
			AddField("Name", info.Player.Player.Name).
			AddField("Server", info.Server.ShortName).MakeFieldInline().
			AddField("steam", fmt.Sprintf("https://steamcommunity.com/profiles/%d", info.Player.Player.SID.Int64())).
			AddField("connect", "connect "+info.Server.Addr())
	}

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func SayMessage(server string, msg string) *discordgo.MessageEmbed {
	return message.NewEmbed("Sent chat message successfully").Embed().
		SetColor(message.ColourSuccess).
		AddField("ServerStore", server).
		AddField("Message", msg).
		Truncate().MessageEmbed
}

func CSayMessage(server string, msg string) *discordgo.MessageEmbed {
	return message.NewEmbed("Sent console message successfully").Embed().
		SetColor(message.ColourSuccess).
		AddField("ServerStore", server).
		AddField("Message", msg).
		Truncate().MessageEmbed
}

func PSayMessage(player steamid.SteamID, msg string) *discordgo.MessageEmbed {
	return message.NewEmbed("Sent private message successfully").Embed().
		SetColor(message.ColourSuccess).
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

func ServersMessage(currentStateRegion map[string][]ServerState, serversURL string) *discordgo.MessageEmbed {
	var (
		stats       = map[string]float64{}
		used, total = 0, 0
		regionNames = make([]string, 9)
	)

	msgEmbed := message.NewEmbed("Current ServerStore Populations")
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
		msgEmbed.Embed().SetColor(message.ColourError)
		msgEmbed.Embed().SetDescription("No server states available")
	}

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func PlayersMessage(rows []string, maxPlayers int, serverName string) *discordgo.MessageEmbed {
	msgEmbed := message.NewEmbed(fmt.Sprintf("%s Current Players: %d / %d", serverName, len(rows), maxPlayers))
	if len(rows) > 0 {
		msgEmbed.Embed().SetDescription(strings.Join(rows, "\n"))
		msgEmbed.Embed().SetColor(message.ColourSuccess)
	} else {
		msgEmbed.Embed().SetDescription("No players :(")
		msgEmbed.Embed().SetColor(message.ColourError)
	}

	return msgEmbed.Embed().MessageEmbed
}

func KickMessage(players []PlayerServerInfo) *discordgo.MessageEmbed {
	msgEmbed := message.NewEmbed("Users Kicked")
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

func PingModMessage(author domain.PersonInfo, authorURL string, reason string, server Server, roleID string, connect string) *discordgo.MessageEmbed {
	msgEmbed := message.NewEmbed("New User In-Game Report")
	msgEmbed.
		Embed().
		SetDescription(fmt.Sprintf("%s | <@&%s>", reason, roleID)).
		AddField("server", server.Name)

	if connect != "" {
		msgEmbed.Embed().AddField("connect", connect)
	}

	msgEmbed.AddAuthorPersonInfo(author, authorURL).Embed().Truncate()

	return msgEmbed.Embed().MessageEmbed
}
