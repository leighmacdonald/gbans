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
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/quarckster/go-mpris-server/pkg/server"
)

func makeOnFind() func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, i *discordgo.InteractionCreate,
	) (*discordgo.MessageEmbed, error) {
		opts := OptionMap(i.ApplicationCommandData().Options)
		userIdentifier := opts[OptUserIdentifier].StringValue()

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

		var found []message.FoundPlayer

		for _, player := range players {
			server, errServer := h.servers.Server(ctx, player.ServerID)
			if errServer != nil {
				return nil, errors.Join(errServer, domain.ErrGetServer)
			}

			_, errPerson := h.persons.GetOrCreatePersonBySteamID(ctx, nil, player.Player.SID)
			if errPerson != nil {
				return nil, errors.Join(errPerson, domain.ErrFetchPerson)
			}

			found = append(found, message.FoundPlayer{
				Player: player,
				Server: server,
			})
		}

		return message.FindMessage(found), nil
	}
}

func makeOnKick() func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		var (
			opts   = OptionMap(interaction.ApplicationCommandData().Options)
			reason = ban.Reason(opts[OptBanReason].IntValue())
		)

		target, errTarget := steamid.Resolve(ctx, opts[OptUserIdentifier].StringValue())
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

		return message.KickMessage(players), err
	}
}

func makeOnSay() func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		opts := OptionMap(interaction.ApplicationCommandData().Options)
		serverName := opts[OptServerIdentifier].StringValue()
		msg := opts[OptMessage].StringValue()

		var server servers.Server
		if err := h.servers.GetByName(ctx, serverName, &server, false, false); err != nil {
			return nil, servers.ErrUnknownServer
		}

		if errSay := h.state.Say(ctx, server.ServerID, msg); errSay != nil {
			return nil, ErrCommandFailed
		}

		return message.SayMessage(serverName, msg), nil
	}
}

func makeOnCSay() func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	) (*discordgo.MessageEmbed, error) {
		opts := OptionMap(interaction.ApplicationCommandData().Options)
		serverName := opts[OptServerIdentifier].StringValue()
		msg := opts[OptMessage].StringValue()

		var server server.Server
		if err := h.servers.GetByName(ctx, serverName, &server, false, false); err != nil {
			return nil, domain.ErrUnknownServer
		}

		if errCSay := h.state.CSay(ctx, server.ServerID, msg); errCSay != nil {
			return nil, ErrCommandFailed
		}

		return message.CSayMessage(server.Name, msg), nil
	}
}

func makeOnPSay() func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		opts := OptionMap(interaction.ApplicationCommandData().Options)
		msg := opts[OptMessage].StringValue()

		playerSid, errPlayerSid := steamid.Resolve(ctx, opts[OptUserIdentifier].StringValue())
		if errPlayerSid != nil || playerSid.Valid() {
			return nil, errors.Join(errPlayerSid, domain.ErrInvalidSID)
		}

		if errPSay := h.state.PSay(ctx, playerSid, msg); errPSay != nil {
			return nil, ErrCommandFailed
		}

		return message.PSayMessage(playerSid, msg), nil
	}
}

func makeOnServers() func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		return message.ServersMessage(h.state.SortRegion(), h.config.ExtURLRaw("/servers")), nil
	}
}

func makeOnPlayers() func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		opts := OptionMap(interaction.ApplicationCommandData().Options)
		serverName := opts[OptServerIdentifier].StringValue()

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

		return message.PlayersMessage(rows, serverState.MaxPlayers, serverState.Name), nil
	}
}

type FoundPlayer struct {
	Player servers.PlayerServerInfo
	Server servers.Server
}

func FindMessage(found []FoundPlayer) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("Player(s) Found")
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
	return NewEmbed("Sent chat message successfully").Embed().
		SetColor(ColourSuccess).
		AddField("ServerStore", server).
		AddField("Message", msg).
		Truncate().MessageEmbed
}

func CSayMessage(server string, msg string) *discordgo.MessageEmbed {
	return NewEmbed("Sent console message successfully").Embed().
		SetColor(ColourSuccess).
		AddField("ServerStore", server).
		AddField("Message", msg).
		Truncate().MessageEmbed
}

func PSayMessage(player steamid.SteamID, msg string) *discordgo.MessageEmbed {
	return NewEmbed("Sent private message successfully").Embed().
		SetColor(ColourSuccess).
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

func ServersMessage(currentStateRegion map[string][]state.ServerState, serversURL string) *discordgo.MessageEmbed {
	var (
		stats       = map[string]float64{}
		used, total = 0, 0
		regionNames = make([]string, 9)
	)

	msgEmbed := NewEmbed("Current ServerStore Populations")
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
		msgEmbed.Embed().SetColor(ColourError)
		msgEmbed.Embed().SetDescription("No server states available")
	}

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func PlayersMessage(rows []string, maxPlayers int, serverName string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed(fmt.Sprintf("%s Current Players: %d / %d", serverName, len(rows), maxPlayers))
	if len(rows) > 0 {
		msgEmbed.Embed().SetDescription(strings.Join(rows, "\n"))
		msgEmbed.Embed().SetColor(ColourSuccess)
	} else {
		msgEmbed.Embed().SetDescription("No players :(")
		msgEmbed.Embed().SetColor(ColourError)
	}

	return msgEmbed.Embed().MessageEmbed
}

func KickMessage(players []servers.PlayerServerInfo) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("Users Kicked")
	for _, player := range players {
		msgEmbed.Embed().AddField("Name", player.Player.Name)
		msgEmbed.AddFieldsSteamID(player.Player.SID)
	}

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func NewInGameReportResponse(report ban.ReportWithAuthor, reportURL string, author domain.PersonInfo, authorURL string, _ string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("New User Report Created")
	msgEmbed.
		Embed().
		SetDescription(report.Description).
		SetColor(ColourSuccess).
		SetURL(reportURL)

	msgEmbed.AddAuthorPersonInfo(author, authorURL)

	name := author.GetName()

	if name == "" {
		name = report.TargetID.String()
	}

	msgEmbed.
		Embed().
		AddField("Subject", name).
		AddField("Reason", report.Reason.String())

	if report.ReasonText != "" {
		msgEmbed.Embed().AddField("Custom Reason", report.ReasonText)
	}

	return msgEmbed.AddFieldsSteamID(report.TargetID).Embed().Truncate().MessageEmbed
}

func PingModMessage(author domain.PersonInfo, authorURL string, reason string, server servers.Server, roleID string, connect string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("New User In-Game Report")
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
