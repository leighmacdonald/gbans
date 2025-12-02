package sourcemod

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

//go:embed sourcemod_discord.tmpl
var templateContent []byte

type discordHandler struct {
	sourcemod Sourcemod
	servers   *servers.Servers
}

func RegisterDiscordCommands(service discord.Service, sourcemod Sourcemod, servers *servers.Servers) {
	handler := discordHandler{sourcemod: sourcemod, servers: servers}

	service.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
		Name:                     "rcon",
		Description:              "Send an rcon command",
		DefaultMemberPermissions: ptr.To(discord.AdminPerms),
		Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "target_server",
				Description:  "Short server name",
				Required:     true,
				Autocomplete: true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "command",
				Description: "Command to run",
				Required:    true,
				MinLength:   ptr.To(1),
			},
		},
	}, handler.onRCON)

	service.MustRegisterTemplate("sourcemod", templateContent)
	service.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
		Name:                     "seed",
		Description:              "Request a server seed ping",
		DefaultMemberPermissions: ptr.To(discord.UserPerms),
		Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "target_server",
				Description:  "Short server name",
				Required:     true,
				Autocomplete: true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "message",
				Description: "Message shown",
			},
		},
	}, handler.onSeed)
	service.MustRegisterPrefixHandler("target_server", discord.Autocomplete(servers.AutoCompleteServers))

	service.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
		Name:                     "sourcemod",
		Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
		DefaultMemberPermissions: ptr.To(discord.ModPerms),
		Description:              "Update sourcemod configurations",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "admins",
				Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
				Description: "SteamID in any format OR profile url",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "edit",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Description: "Edit admin",
						Options: []*discordgo.ApplicationCommandOption{
							{
								Name:         "steamid",
								Description:  "SteamID or Profile URL",
								Type:         discordgo.ApplicationCommandOptionString,
								Autocomplete: true,
							},
						},
					},
				},
			},
			{
				Name:        "groups",
				Description: "Sourcemod Groups",
				Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "edit",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
						Description: "Edit Group",
						Options: []*discordgo.ApplicationCommandOption{
							{
								Name:         "group",
								Description:  "Sourcemod Permission Group",
								Type:         discordgo.ApplicationCommandOptionString,
								Autocomplete: true,
							},
						},
					},
				},
			},
		},
	}, handler.onSourcemod)
	service.MustRegisterPrefixHandler("sourcemod_admins_edit_modal", handler.onSourcemodAdminsEditModal)
	service.MustRegisterPrefixHandler("admins", discord.Autocomplete(handler.adminCompleter()))
	service.MustRegisterPrefixHandler("groups", discord.Autocomplete(handler.groupCompleter()))
}

func (h discordHandler) adminCompleter() func(ctx context.Context, query string) ([]discord.AutoCompleteValuer, error) {
	var (
		admins     []Admin
		lastUpdate time.Time
		mutex      sync.RWMutex
	)

	return func(ctx context.Context, query string) ([]discord.AutoCompleteValuer, error) {
		mutex.RLock()
		curAdmins := admins
		expired := time.Since(lastUpdate) > time.Minute || len(admins) == 0
		mutex.RUnlock()

		if expired {
			update, errUpdate := h.sourcemod.Admins(ctx)
			if errUpdate != nil {
				return nil, errUpdate
			}
			mutex.Lock()
			admins = update
			lastUpdate = time.Now()
			mutex.Unlock()
		}

		query = strings.ToLower(query)
		var values []discord.AutoCompleteValuer
		for _, admin := range curAdmins {
			if query == "" ||
				admin.SteamID.Equal(steamid.New(query)) ||
				strings.Contains(strings.ToLower(admin.Name), query) {
				values = append(values, discord.NewAutoCompleteValue(admin.Name+" "+admin.SteamID.String(), admin.SteamID.String()))
			}
		}

		return values, nil
	}
}

func (h discordHandler) groupCompleter() func(ctx context.Context, query string) ([]discord.AutoCompleteValuer, error) {
	var (
		groups     []Groups
		lastUpdate time.Time
		mutex      sync.RWMutex
	)

	return func(ctx context.Context, query string) ([]discord.AutoCompleteValuer, error) {
		mutex.RLock()
		curGroups := groups
		expired := time.Since(lastUpdate) > time.Minute || len(groups) == 0
		mutex.RUnlock()

		if expired {
			mutex.Lock()
			update, errUpdate := h.sourcemod.Groups(ctx)
			if errUpdate != nil {
				return nil, errUpdate
			}
			groups = update
			lastUpdate = time.Now()
			mutex.Unlock()
		}

		query = strings.ToLower(query)
		var values []discord.AutoCompleteValuer
		for _, group := range curGroups {
			if query == "" ||
				strconv.Itoa(group.GroupID) == query ||
				strings.Contains(strings.ToLower(group.Name), query) {
				values = append(values, discord.NewAutoCompleteValue(
					fmt.Sprintf("%s [#%d]", group.Name, group.GroupID),
					strconv.Itoa(group.GroupID)))
			}
		}

		return values, nil
	}
}

func (h discordHandler) onSourcemod(ctx context.Context, session *discordgo.Session, interation *discordgo.InteractionCreate) error {
	data := interation.ApplicationCommandData()
	if len(data.Options) == 0 || len(data.Options[0].Options) == 0 {
		return fmt.Errorf("%w: invalid options", discord.ErrCommandFailed)
	}
	group := data.Options[0]
	subCmd := group.Options[0]
	opts := discord.OptionMap(subCmd.Options)
	switch group.Name { //nolint:gocritic
	case "admins":
		switch subCmd.Name { //nolint:gocritic
		case "edit":
			return h.onAdminsEdit(ctx, session, interation, opts)
		}
	case "groups":
		switch subCmd.Name { //nolint:gocritic
		case "edit":
			return h.onGroupsEdit(ctx, session, interation, opts)
		}
	}

	return fmt.Errorf("%w: unknown command", discord.ErrCommandFailed)
}

func (h discordHandler) onAdminsEdit(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, opts discord.CommandOptions) error {
	var (
		requestedSID string
		alias        string
		flags        string
		immunity     string
	)

	sidOption, found := opts["steamid"]
	if found {
		sid, errResolveSID := steamid.Resolve(ctx, sidOption.StringValue())
		if errResolveSID != nil || !sid.Valid() {
			return steamid.ErrInvalidSID
		}
		requestedSID = sid.String()

		if admin, errAdmin := h.sourcemod.AdminBySteamID(ctx, sid); errAdmin == nil {
			alias = admin.Name
			flags = admin.Flags
			immunity = strconv.Itoa(admin.Immunity)
		}
	}

	groupEdit := "sourcemod_admins_edit_modal"

	return discord.RespondModal(session, interaction, groupEdit,
		"Edit sourcemod admin settings",
		discord.ModalInputRowRequired(discord.IDSteamID, "steamid", "SteamID or Profile URL", "76561197960542812", requestedSID, 0, 64),
		discord.ModalInputRowRequired(discord.IDAlias, "alias", "Player Alias", "Bob Bobbins", alias, 0, 0),
		discord.ModalInputRowRequired(discord.IDFlags, "flags", "Flag Set", "abcdef", flags, 1, 10),
		discord.ModalInputRowRequired(discord.IDImmunityLevel, "immunity", "Immunity Level", "100", immunity, 0, 3))
}

type adminEditModal struct {
	SteamID  steamid.SteamID `id:"1"`
	Alias    string          `id:"8"`
	Flags    string          `id:"9"`
	Immunity int             `id:"7"`
}

func (h discordHandler) onSourcemodAdminsEditModal(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	data := interaction.ModalSubmitData()
	request, errReqiest := discord.Bind[adminEditModal](ctx, data.Components)
	if errReqiest != nil {
		return errReqiest
	}

	admins, errAdmins := h.sourcemod.Admins(ctx)
	if errAdmins != nil && !errors.Is(errAdmins, database.ErrNoResult) {
		return errAdmins
	}

	var admin Admin
	for _, curAdmin := range admins {
		if curAdmin.SteamID.Equal(request.SteamID) {
			admin = curAdmin

			break
		}
	}

	admin.SteamID = request.SteamID
	admin.Name = request.Alias
	admin.Flags = request.Flags
	admin.Immunity = request.Immunity
	admin.AuthType = AuthTypeSteam

	_, errSave := h.sourcemod.SaveAdmin(ctx, admin)
	if errSave != nil {
		return errSave
	}

	return discord.Success(session, interaction)
}

func (h discordHandler) onGroupsEdit(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, opts discord.CommandOptions) error {
	var (
		alias    string
		flags    string
		immunity string
		customID = "sourcemod_group_edit_modal"
	)

	gidOption, found := opts["groupid"]
	if found && gidOption.StringValue() != "" {
		gid, errGID := strconv.Atoi(gidOption.StringValue())
		if errGID != nil {
			return errors.Join(errGID, discord.ErrCommandInvalid)
		}

		group, errGroup := h.sourcemod.GetGroupByID(ctx, gid)
		if errGroup != nil {
			return errGroup
		}

		alias = group.Name
		flags = group.Flags
		immunity = strconv.Itoa(group.ImmunityLevel)
		customID += "_" + strconv.Itoa(gid)
	}

	return discord.RespondModal(session, interaction, customID,
		"Edit sourcemod admin settings",
		discord.ModalInputRowRequired(discord.IDAlias, "alias", "Group Alias/Name", "Bob Bobbins", alias, 0, 0),
		discord.ModalInputRowRequired(discord.IDFlags, "flags", "Flag Set", "abcdef", flags, 1, 10),
		discord.ModalInputRow(discord.IDImmunityLevel, "immunity", "Immunity Level", "100", immunity, 0, 3),
	)
}

func (h discordHandler) onSeed(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	data := discord.OptionMap(interaction.ApplicationCommandData().Options)
	server, errServer := h.servers.GetByName(ctx, data["target_server"].StringValue())
	if errServer != nil {
		return errServer
	}

	if !h.sourcemod.seedRequest(ctx, server, interaction.Member.User.ID) {
		discord.Error(session, interaction, ErrReqTooSoon)

		return nil
	}

	return discord.Success(session, interaction)
}

func (h discordHandler) onRCON(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	if err := discord.AckInteraction(session, interaction); err != nil {
		return err
	}
	data := discord.OptionMap(interaction.ApplicationCommandData().Options)
	server, errServer := h.servers.GetByName(ctx, data["target_server"].StringValue())
	if errServer != nil {
		return errServer
	}
	command := data["command"].StringValue()

	resp, errResp := server.Exec(ctx, command)
	if errResp != nil {
		return errResp
	}

	return discord.RespondUpdate(session, interaction,
		discord.Heading("RCON Command: %s", command),
		discord.BodyColouredText(discord.ColourInfo, "```"+resp+"```"))
}
