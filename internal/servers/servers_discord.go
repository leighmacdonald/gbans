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
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

//go:embed servers_discord.gotmpl
var templateBody []byte

const (
	// maxSelectMenuOptions is discord's limit on the number of options a select menu can hold.
	maxSelectMenuOptions = 25
	seedRolesCustomID    = "seed_roles_select"
)

type DiscordHandler struct {
	persons       person.Provider
	servers       *Servers
	network       network.Networks
	notifier      notification.Notifier
	kickChannelID string
}

func RegisterDiscordCommands(service discord.Connection, persons person.Provider, servers *Servers,
	network network.Networks, notifier notification.Notifier, kickChannelID string,
) {
	discord.MustRegisterTemplate(templateBody)

	handler := DiscordHandler{
		persons:       persons,
		servers:       servers,
		network:       network,
		notifier:      notifier,
		kickChannelID: kickChannelID,
	}

	serverAutoCompleteOption := &discordgo.ApplicationCommandOption{
		Type:         discordgo.ApplicationCommandOptionString,
		Name:         "server_name",
		Description:  "Short server name",
		Required:     true,
		Autocomplete: true,
	}

	service.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
		Name:                     "find",
		Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
		DefaultMemberPermissions: new(discord.ModPerms),
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
		DefaultMemberPermissions: new(discord.UserPerms),
		Description:              "Show a table of the current players on the server",
		Options:                  []*discordgo.ApplicationCommandOption{serverAutoCompleteOption},
	}, handler.onPlayers)
	service.MustRegisterPrefixHandler("server_name", discord.Autocomplete(servers.AutoCompleteServers))

	service.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
		Name:                     "kick",
		Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
		DefaultMemberPermissions: new(discord.ModPerms),
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
		DefaultMemberPermissions: new(discord.ModPerms),
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
		DefaultMemberPermissions: new(discord.ModPerms),
		Options: []*discordgo.ApplicationCommandOption{
			serverAutoCompleteOption,
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
		DefaultMemberPermissions: new(discord.ModPerms),
		Options: []*discordgo.ApplicationCommandOption{
			serverAutoCompleteOption,
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
		DefaultMemberPermissions: new(discord.UserPerms),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "full",
				Description: "Return the full status output including server versions and tags",
			},
		},
	}, handler.onServers)

	service.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
		Name:                     "seed_roles",
		Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
		DefaultMemberPermissions: new(discord.UserPerms),
		Description:              "Select the seed role(s) you would like to receive seed request notifications for",
	}, handler.onSeedRoles)

	service.MustRegisterPrefixHandler(seedRolesCustomID, handler.onSeedRolesSelect)
}

func (d DiscordHandler) onFind(ctx context.Context, session *discordgo.Session, interation *discordgo.InteractionCreate) error {
	opts := discord.OptionMap(interation.ApplicationCommandData().Options)
	userIdentifier := opts[discord.OptUserIdentifier].StringValue()

	steamID, errSteamID := steamid.Resolve(ctx, userIdentifier)
	if errSteamID != nil || !steamID.Valid() {
		return steamid.ErrDecodeSID
	}

	results := d.servers.FindPlayers(FindOpts{SteamID: steamID})
	if len(results) == 0 {
		return steamid.ErrDecodeSID
	}

	return discord.RespondUpdate(session, interation, discordFindMessage(results)...)
}

func (d DiscordHandler) onKick(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options)
	target, errTarget := steamid.Resolve(ctx, opts[discord.OptUserIdentifier].StringValue())
	if errTarget != nil || !target.Valid() {
		return steamid.ErrDecodeSID
	}
	reason := opts["reason"].StringValue()
	if err := discord.AckInteraction(session, interaction); err != nil {
		return err
	}

	results := d.servers.FindPlayers(FindOpts{SteamID: target})
	if len(results) == 0 {
		return ErrPlayerNotFound
	}

	var err error
	for _, result := range results {
		if errKick := result.Server.Kick(ctx, result.Player.SID, reason); errKick != nil {
			err = errors.Join(err, errKick)

			continue
		}
		d.notifier.Send(notification.NewDiscord(d.kickChannelID, discord.NewMessage(
			discord.Heading("User Kicked [%s]", result.Server.ShortName),
			discord.BodyColouredText(discord.ColourInfo, result.Player.Name))))
	}

	if len(results) > 0 {
		return discord.RespondUpdate(session, interaction,
			discord.BodyColouredText(discord.ColourSuccess, fmt.Sprintf("Kick successful [Kicked: %d]", len(results))))
	}

	return discord.RespondUpdate(session, interaction,
		discord.BodyColouredText(discord.ColourError, "No matching users found"))
}

func (d DiscordHandler) onSay(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	opts := discord.OptionMap(interaction.ApplicationCommandData().Options)
	serverName := opts[discord.OptServerIdentifier].StringValue()
	msg := opts[discord.OptMessage].StringValue()

	server, err := d.servers.GetByName(ctx, serverName)
	if err != nil {
		return ErrUnknownServer
	}

	if errSay := server.Say(ctx, SayOpts{Type: Say, Message: msg}); errSay != nil {
		return discord.ErrCommandFailed
	}

	return discord.Respond(session, interaction, discordSayMessage(serverName, msg)...)
}

func (d DiscordHandler) onCSay(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	var (
		opts       = discord.OptionMap(interaction.ApplicationCommandData().Options)
		serverName = opts[discord.OptServerIdentifier].StringValue()
		msg        = opts[discord.OptMessage].StringValue()
	)

	server, err := d.servers.GetByName(ctx, serverName)
	if err != nil {
		return ErrUnknownServer
	}

	if len(msg) == 0 {
		return discord.ErrCommandFailed
	}

	if errCSay := server.Say(ctx, SayOpts{Type: CSay, Message: msg}); errCSay != nil {
		return discord.ErrCommandFailed
	}

	return discord.Respond(session, interaction, discordCSayMessage(server.Name, msg)...)
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

	results := d.servers.FindPlayers(FindOpts{SteamID: playerSid})
	if len(results) == 0 {
		return discord.ErrCommandFailed
	}

	for _, result := range results {
		sayOpts := SayOpts{Type: PSay, Message: msg, Targets: []steamid.SteamID{playerSid}}
		if errPSay := result.Server.Say(ctx, sayOpts); errPSay != nil {
			return discord.ErrCommandFailed
		}
	}

	return discord.Respond(session, interaction, discordPSayMessage(playerSid, msg)...)
}

func (d DiscordHandler) onServers(_ context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	return discord.Respond(session, interaction, discordServersMessage(d.servers.servers.sortRegion())...)
}

// onSeedRoles responds with a select menu of the seed roles configured on enabled servers.
// Roles selected from the menu are added to the requesting user's own account.
func (d DiscordHandler) onSeedRoles(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	options, errOptions := d.seedRoleOptions(ctx, session, interaction)
	if errOptions != nil {
		return errOptions
	}

	if len(options) == 0 {
		return discord.SuccessMsg(session, interaction, "No seed roles are currently configured on enabled servers.")
	}

	minValues := 1

	return discord.RespondPrivate(session, interaction, discordgo.Container{
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{
				Content: "Select the seed role(s) you would like to receive seed request notifications for:",
			},
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						CustomID:    seedRolesCustomID,
						Placeholder: "Select seed role(s)",
						MinValues:   &minValues,
						MaxValues:   len(options),
						MenuType:    discordgo.StringSelectMenu,
						Options:     options,
					},
				},
			},
		},
	})
}

// onSeedRolesSelect handles the seed role select menu callback, adding the selected roles to the
// requesting member's own discord account.
func (d DiscordHandler) onSeedRolesSelect(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	values := interaction.MessageComponentData().Values
	if len(values) == 0 {
		return discord.ErrCommandInvalid
	}

	seedRoleIDs, errSeedRoles := d.servers.SeedRoleIDs(ctx)
	if errSeedRoles != nil {
		return errSeedRoles
	}

	allowed := make(map[string]struct{}, len(seedRoleIDs))
	for _, roleID := range seedRoleIDs {
		allowed[roleID] = struct{}{}
	}

	var (
		errs    []error
		granted []string
	)

	for _, value := range values {
		if _, ok := allowed[value]; !ok {
			// Role is no longer configured on an enabled server, eg: the server was disabled after the menu was rendered.
			errs = append(errs, fmt.Errorf("%w: %s", ErrInvalidSeedRole, value))

			continue
		}

		if errAdd := session.GuildMemberRoleAdd(interaction.GuildID, interaction.Member.User.ID, value); errAdd != nil {
			errs = append(errs, fmt.Errorf("failed to add seed role %s: %w", value, errAdd))

			continue
		}

		granted = append(granted, value)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	mentions := make([]string, 0, len(granted))
	for _, roleID := range granted {
		mentions = append(mentions, "<@&"+roleID+">")
	}

	return discord.SuccessMsg(session, interaction,
		"Seed role(s) added to your account: "+strings.Join(mentions, " "))
}

// seedRoleOptions builds the select menu options for the seed roles configured on enabled servers,
// labelled with their guild role name. Role ids that no longer exist in the guild are skipped.
func (d DiscordHandler) seedRoleOptions(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) ([]discordgo.SelectMenuOption, error) {
	seedRoleIDs, errSeedRoles := d.servers.SeedRoleIDs(ctx)
	if errSeedRoles != nil {
		return nil, errSeedRoles
	}

	if len(seedRoleIDs) == 0 {
		return nil, nil
	}

	guildRoles, errGuildRoles := session.GuildRoles(interaction.GuildID)
	if errGuildRoles != nil {
		return nil, errors.Join(errGuildRoles, discord.ErrRole)
	}

	return seedRoleMenuOptions(seedRoleIDs, guildRoles), nil
}

// seedRoleMenuOptions maps seed role ids to select menu options labelled with their guild role
// name, skipping ids that no longer exist in the guild. The result is sorted by label and capped
// at discord's 25 option limit.
func seedRoleMenuOptions(seedRoleIDs []string, guildRoles []*discordgo.Role) []discordgo.SelectMenuOption {
	byID := make(map[string]*discordgo.Role, len(guildRoles))
	for _, role := range guildRoles {
		byID[role.ID] = role
	}

	var options []discordgo.SelectMenuOption //nolint:prealloc
	for _, roleID := range seedRoleIDs {
		role, found := byID[roleID]
		if !found {
			slog.Warn("Seed role not found in guild", slog.String("role_id", roleID))

			continue
		}
		options = append(options, discordgo.SelectMenuOption{Label: role.Name, Value: role.ID})
	}

	sort.SliceStable(options, func(i, j int) bool {
		return options[i].Label < options[j].Label
	})

	if len(options) > maxSelectMenuOptions {
		options = options[:maxSelectMenuOptions]
	}

	return options
}

func (d DiscordHandler) onPlayers(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	if err := discord.AckInteraction(session, interaction); err != nil {
		return err
	}

	opts := discord.OptionMap(interaction.ApplicationCommandData().Options)
	serverName := opts["server_name"].StringValue()
	server, errServers := d.servers.GetByName(ctx, serverName)
	if errServers != nil {
		return errServers
	}

	var rows []string

	if len(server.state.Players) > 0 {
		sort.SliceStable(server.state.Players, func(i, j int) bool {
			return server.state.Players[i].Name < server.state.Players[j].Name
		})

		for _, player := range server.state.Players {
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

	return discord.RespondUpdate(session, interaction, discordPlayersMessage(rows, server.state.MaxPlayers, server.Name)...)
}

func discordFindMessage(found []FindResult) []discordgo.MessageComponent {
	content, err := discord.RenderTemplate("find", struct {
		Found []FindResult
	}{Found: found})
	if err != nil {
		slog.Error("Failed to render player find", slog.String("error", err.Error()))
	}

	return []discordgo.MessageComponent{
		discord.BodyColouredText(discord.ColourSuccess, content),
	}
}

func discordSayMessage(server string, msg string) []discordgo.MessageComponent {
	content, errContent := discord.RenderTemplate("say", struct {
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
		discord.BodyColouredText(discord.ColourSuccess, content),
	}
}

func discordCSayMessage(server string, msg string) []discordgo.MessageComponent {
	content, errContent := discord.RenderTemplate("csay", struct {
		Server  string
		Message string
	}{Server: server, Message: msg})
	if errContent != nil {
		return nil
	}

	return []discordgo.MessageComponent{
		discord.BodyColouredText(discord.ColourSuccess, content),
	}
}

func discordPSayMessage(player steamid.SteamID, msg string) []discordgo.MessageComponent {
	content, errContent := discord.RenderTemplate("psay", struct {
		Player  string
		Message string
	}{Player: player.String(), Message: msg})
	if errContent != nil {
		return nil
	}

	return []discordgo.MessageComponent{
		discord.BodyColouredText(discord.ColourSuccess, content),
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

func discordServersMessage(currentStateRegion map[string][]*Server) []discordgo.MessageComponent {
	var ( //nolint:prealloc
		stats       = map[string]float64{}
		used, total = int32(0), int32(0)
		regionNames = make([]string, 9) //nolint:prealloc
		rows        []string
	)

	for k := range currentStateRegion {
		regionNames = append(regionNames, k)
	}

	sort.Strings(regionNames)

	for _, region := range regionNames {
		var counts []string

		for _, curState := range currentStateRegion[region] {
			_, ok := stats[region]
			if !ok {
				stats[region] = 0
				stats[region+"total"] = 0
			}

			maxPlayers := curState.state.MaxPlayers - curState.ReservedSlots
			if maxPlayers <= 0 {
				maxPlayers = 32 - curState.ReservedSlots
			}

			stats[region] += float64(curState.state.PlayerCount)
			stats[region+"total"] += float64(maxPlayers)
			used += curState.state.PlayerCount
			total += maxPlayers
			counts = append(counts, fmt.Sprintf("%s:   %2d/%2d  ", curState.ShortName, curState.state.PlayerCount, maxPlayers))
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

	content, errContent := discord.RenderTemplate("servers", struct {
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
		discord.BodyColouredText(colour, content),
		discord.Buttons(discord.Link("View Servers", link.Raw("/servers"))),
	}
}

func discordPlayersMessage(rows []string, maxPlayers int32, serverName string) []discordgo.MessageComponent {
	body := "No players"
	if len(rows) > 0 {
		body = strings.Join(rows, "\n")
	}
	content := fmt.Sprintf(`# %s
### Current Players: %d / %d
%s`, serverName, len(rows), maxPlayers, body)

	return []discordgo.MessageComponent{discord.BodyText(content)}
}

func discordKickMessage(players []FindResult) []discordgo.MessageComponent {
	content, err := discord.RenderTemplate("user_kick", struct {
		Players []FindResult
	}{
		Players: players,
	})
	if err != nil {
		slog.Error("Failed to render template", slog.String("error", err.Error()))
	}

	return []discordgo.MessageComponent{
		discord.BodyColouredText(discord.ColourSuccess, content),
	}
}
