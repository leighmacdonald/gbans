package servers

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/internal/servers/state"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

//go:embed servers_discord.tmpl
var templateBody []byte

func RegisterDiscordCommands(service discord.Service, state *state.State,
	persons person.Provider, servers Servers, network network.Networks,
) {
	handler := DiscordHandler{state: state, persons: persons, servers: servers, network: network}

	service.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
		Name:                     "find",
		Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
		DefaultMemberPermissions: ptr.To(discord.ModPerms),
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

	service.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
		Name:                     "players",
		Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
		DefaultMemberPermissions: ptr.To(discord.UserPerms),
		Description:              "Show a table of the current players on the server",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "server_name",
				Description:  "Short server name",
				Required:     true,
				Autocomplete: true,
			},
		},
	}, handler.onPlayers)
	service.MustRegisterPrefixHandler("server_name", discord.Autocomplete(servers.AutoCompleteServers))

	service.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
		Name:                     "kick",
		Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
		DefaultMemberPermissions: ptr.To(discord.ModPerms),
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

	service.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
		Name:                     "psay",
		Description:              "Privately message a player",
		Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
		DefaultMemberPermissions: ptr.To(discord.ModPerms),
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

	service.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
		Name:                     "csay",
		Description:              "Send a centered message to the whole server",
		Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
		DefaultMemberPermissions: ptr.To(discord.ModPerms),
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

	service.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
		Name:                     "say",
		Description:              "Send a console message to the whole server",
		Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
		DefaultMemberPermissions: ptr.To(discord.ModPerms),
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

	service.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
		Name:                     "servers",
		Description:              "Show the high level status of all servers",
		DefaultMemberPermissions: ptr.To(discord.UserPerms),
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
	state   *state.State
	persons person.Provider
	servers Servers
	network network.Networks
}

func (d DiscordHandler) onFind(ctx context.Context, session *discordgo.Session, interation *discordgo.InteractionCreate) error {
	opts := discord.OptionMap(interation.ApplicationCommandData().Options)
	userIdentifier := opts[discord.OptUserIdentifier].StringValue()

	steamID, errSteamID := steamid.Resolve(ctx, userIdentifier)
	if errSteamID != nil || !steamID.Valid() {
		return steamid.ErrDecodeSID
	}

	players := d.state.Find(state.FindOpts{SteamID: steamID})
	if len(players) == 0 {
		return steamid.ErrDecodeSID
	}

	found := make([]discordFoundPlayer, len(players))

	for index, player := range players {
		server, errServer := d.servers.Server(ctx, player.ServerID)
		if errServer != nil {
			return errors.Join(errServer, ErrGetServer)
		}

		_, errPerson := d.persons.GetOrCreatePersonBySteamID(ctx, player.Player.SID)
		if errPerson != nil {
			return errPerson
		}

		found[index] = discordFoundPlayer{Player: player, Server: server}
	}

	return discord.RespondUpdate(session, interation, discordFindMessage(found)...)
}

func (d DiscordHandler) onKick(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options)
	// reason = ban.Reason(opts[discord.OptBanReason].IntValue())

	target, errTarget := steamid.Resolve(ctx, opts[discord.OptUserIdentifier].StringValue())
	if errTarget != nil || !target.Valid() {
		return steamid.ErrDecodeSID
	}

	players := d.state.Find(state.FindOpts{SteamID: target})

	if len(players) == 0 {
		return state.ErrPlayerNotFound
	}

	var err error

	for _, player := range players {
		if errKick := d.state.Kick(ctx, player.Player.SID, ""); errKick != nil {
			err = errors.Join(err, errKick)

			continue
		}
	}

	return discord.Respond(session, interaction, discordKickMessage(players))
}

func (d DiscordHandler) onSay(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options)
	serverName := opts[discord.OptServerIdentifier].StringValue()
	msg := opts[discord.OptMessage].StringValue()

	server, err := d.servers.GetByName(ctx, serverName)
	if err != nil {
		return state.ErrUnknownServer
	}

	if errSay := d.state.Say(ctx, server.ServerID, msg); errSay != nil {
		return discord.ErrCommandFailed
	}

	return discord.Respond(session, interaction, discordSayMessage(serverName, msg))
}

func (d DiscordHandler) onCSay(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	var (
		opts       = discord.OptionMap(interaction.ApplicationCommandData().Options)
		serverName = opts[discord.OptServerIdentifier].StringValue()
		msg        = opts[discord.OptMessage].StringValue()
	)

	server, err := d.servers.GetByName(ctx, serverName)
	if err != nil {
		return state.ErrUnknownServer
	}

	if len(msg) == 0 {
		return discord.ErrCommandFailed
	}

	if errCSay := d.state.CSay(ctx, server.ServerID, msg); errCSay != nil {
		return discord.ErrCommandFailed
	}

	return discord.Respond(session, interaction, discordCSayMessage(server.Name, msg))
}

func (d DiscordHandler) onPSay(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	var (
		opts = discord.OptionMap(interaction.ApplicationCommandData().Options)
		msg  = opts[discord.OptMessage].StringValue()
	)

	playerSid, errPlayerSid := steamid.Resolve(ctx, opts[discord.OptUserIdentifier].StringValue())
	if errPlayerSid != nil || playerSid.Valid() {
		return errors.Join(errPlayerSid, steamid.ErrDecodeSID)
	}

	if errPSay := d.state.PSay(ctx, playerSid, msg); errPSay != nil {
		return discord.ErrCommandFailed
	}

	return discord.Respond(session, interaction, discordPSayMessage(playerSid, msg))
}

func (d DiscordHandler) onServers(_ context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	return discord.Respond(session, interaction, discordServersMessage(d.state.SortRegion()))
}

func (d DiscordHandler) onPlayers(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	if err := discord.AckInteraction(session, interaction); err != nil {
		return err
	}

	opts := discord.OptionMap(interaction.ApplicationCommandData().Options)
	serverName := opts["server_name"].StringValue()
	serverStates := d.state.ByName(serverName, false)

	if len(serverStates) != 1 {
		return state.ErrUnknownServer
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
				slog.Error("Failed to parse player ip", slog.String("error", errIP.Error()))

				continue
			}

			foundNetwork, errNetwork := d.network.QueryNetwork(ctx, address)
			if errNetwork != nil {
				slog.Error("Failed to get network info", slog.String("error", errNetwork.Error()))

				continue
			}

			flag := ""
			if foundNetwork.Location.CountryCode != "" {
				flag = fmt.Sprintf(":flag_%s: ", strings.ToLower(foundNetwork.Location.CountryCode))
			}

			proxyStr := ""
			if foundNetwork.Proxy.ProxyType != "" {
				proxyStr = fmt.Sprintf("Threat: %s | %s | %s", foundNetwork.Proxy.ProxyType, foundNetwork.Proxy.Threat, foundNetwork.Proxy.UsageType)
			}

			rows = append(rows, fmt.Sprintf("%s`%s` `%s` `%3dms` [%s](https://steamcommunity.com/profiles/%s)%s",
				flag, player.SID.Steam3(), player.ConnectedTime.String(), player.Ping, player.Name, player.SID.String(), proxyStr))
		}
	}

	return discord.RespondUpdate(session, interaction, discordPlayersMessage(rows, serverState.MaxPlayers, serverState.Name)...)
}

type discordFoundPlayer struct {
	Player state.PlayerServerInfo
	Server Server
}

func discordFindMessage(found []discordFoundPlayer) []discordgo.MessageComponent {
	content, err := discord.Render("find", templateBody, struct {
		Found []discordFoundPlayer
	}{Found: found})
	if err != nil {
		slog.Error("Failed to render player find", slog.String("error", err.Error()))
	}

	return []discordgo.MessageComponent{discordgo.Container{
		AccentColor: ptr.To(discord.ColourSuccess),
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: content},
		},
	}}
}

func discordSayMessage(server string, msg string) []discordgo.MessageComponent {
	content, errContent := discord.Render("say", templateBody, struct {
		Server  string
		Message string
	}{
		Server:  server,
		Message: msg,
	})
	if errContent != nil {
		return nil
	}

	return []discordgo.MessageComponent{
		discordgo.Container{
			AccentColor: ptr.To(discord.ColourSuccess),
			Components: []discordgo.MessageComponent{
				discordgo.TextDisplay{Content: content},
			},
		},
	}
}

func discordCSayMessage(server string, msg string) []discordgo.MessageComponent {
	content, errContent := discord.Render("csay", templateBody, struct {
		Server  string
		Message string
	}{Server: server, Message: msg})
	if errContent != nil {
		return nil
	}

	return []discordgo.MessageComponent{
		discordgo.Container{
			AccentColor: ptr.To(discord.ColourSuccess),
			Components: []discordgo.MessageComponent{
				discordgo.TextDisplay{Content: content},
			},
		},
	}
}

func discordPSayMessage(player steamid.SteamID, msg string) []discordgo.MessageComponent {
	content, errContent := discord.Render("psay", templateBody, struct {
		Player  string
		Message string
	}{Player: player.String(), Message: msg})
	if errContent != nil {
		return nil
	}

	return []discordgo.MessageComponent{
		discordgo.Container{
			AccentColor: ptr.To(discord.ColourSuccess),
			Components: []discordgo.MessageComponent{
				discordgo.TextDisplay{Content: content},
			},
		},
	}
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

func discordServersMessage(currentStateRegion map[string][]state.ServerState) []discordgo.MessageComponent {
	var ( //nolint:prealloc
		stats       = map[string]float64{}
		used, total = 0, 0
		regionNames = make([]string, 9)
		rows        []string
	)

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
			rows = append(rows, mapRegion(region)+fmt.Sprintf("```%s```", msg))
		}
	}

	for statName := range stats {
		if strings.HasSuffix(statName, "total") {
			continue
		}

		rows = append(rows, "**"+mapRegion(statName)+"** "+fmt.Sprintf("%.2f%%", (stats[statName]/stats[statName+"total"])*100))
	}

	rows = append(rows, "Global"+fmt.Sprintf("%d/%d %.2f%%", used, total, float64(used)/float64(total)*100))

	content, errContent := discord.Render("servers", templateBody, struct {
		Rows []string
	}{
		Rows: rows,
	})
	if errContent != nil {
		slog.Error("Failed to render servers message", slog.String("error", errContent.Error()))
	}

	colour := discord.ColourSuccess
	if total == 0 {
		colour = discord.ColourError
	}

	return []discordgo.MessageComponent{
		discordgo.Container{
			AccentColor: ptr.To(colour),
			Components: []discordgo.MessageComponent{
				discordgo.TextDisplay{Content: content},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Style: discordgo.LinkButton,
					Label: "View Servers",
					URL:   link.Raw("/servers"),
				},
			},
		},
	}
}

func discordPlayersMessage(rows []string, maxPlayers int, serverName string) []discordgo.MessageComponent {
	body := "No players"
	if len(rows) > 0 {
		body = strings.Join(rows, "\n")
	}
	content := fmt.Sprintf(`# %s 
### Current Players: %d / %d
%s`, serverName, len(rows), maxPlayers, body)

	return []discordgo.MessageComponent{
		discordgo.TextDisplay{Content: content},
	}
}

func discordKickMessage(players []state.PlayerServerInfo) []discordgo.MessageComponent {
	content, err := discord.Render("user_kick", templateBody, struct {
		Players []state.PlayerServerInfo
	}{
		Players: players,
	})
	if err != nil {
		slog.Error("Failed to render template", slog.String("error", err.Error()))
	}

	return []discordgo.MessageComponent{
		discordgo.Container{
			AccentColor: ptr.To(discord.ColourSuccess),
			Components: []discordgo.MessageComponent{
				discordgo.TextDisplay{Content: content},
			},
		},
	}
}
