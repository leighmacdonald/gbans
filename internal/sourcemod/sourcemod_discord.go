package sourcemod

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type discordHandler struct {
	sourcemod Sourcemod
	servers   servers.Servers
}

func RegisterDiscordCommands(service discord.Service, sourcemod Sourcemod, servers servers.Servers) {
	handler := discordHandler{sourcemod: sourcemod, servers: servers}

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

	// service.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
	//	Name:                     "sourcemod",
	//	Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
	//	DefaultMemberPermissions: &discord.ModPerms,
	//	Description:              "Update sourcemod configurations",
	//	Options: []*discordgo.ApplicationCommandOption{
	//		{
	//			Name:        "admins",
	//			Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
	//			Description: "SteamID in any format OR profile url",
	//			Options: []*discordgo.ApplicationCommandOption{
	//				{
	//					Name:        "edit",
	//					Type:        discordgo.ApplicationCommandOptionSubCommand,
	//					Description: "Edit admin",
	//					Options: []*discordgo.ApplicationCommandOption{
	//						{Name: "Steamid"},
	//					},
	//				},
	//			},
	//		},
	//		{
	//			Name:        "groups",
	//			Description: "Sourcemod Groups",
	//			Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
	//			Options: []*discordgo.ApplicationCommandOption{
	//				{
	//					Name:        "edit",
	//					Type:        discordgo.ApplicationCommandOptionSubCommand,
	//					Description: "Edit Group",
	//				},
	//			},
	//		},
	//	},
	// }, handler.onSourcemod)
}

func (h discordHandler) onSourcemod(ctx context.Context, session *discordgo.Session, interation *discordgo.InteractionCreate) error {
	slog.Info("Got sm command")
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

func (h discordHandler) onAdminsEdit(ctx context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate, opts discord.CommandOptions) error {
	sid, errResolveSID := steamid.Resolve(ctx, opts[discord.OptUserIdentifier].StringValue())
	if errResolveSID != nil || !sid.Valid() {
		return steamid.ErrInvalidSID
	}

	return nil
}

func (h discordHandler) onGroupsEdit(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate, _ discord.CommandOptions) error {
	return nil
}

var ErrReqTooSoon = errors.New("⏱️ request is not available yet")

func (h discordHandler) onSeed(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	data := discord.OptionMap(interaction.ApplicationCommandData().Options)
	server, errServer := h.servers.GetByName(ctx, data["target_server"].StringValue())
	if errServer != nil {
		return errServer
	}

	if !h.sourcemod.seedRequest(server, interaction.Member.User.ID) {
		discord.Error(session, interaction, ErrReqTooSoon)

		return nil
	}

	return discord.RespondPrivate(session, interaction, []discordgo.MessageComponent{
		discordgo.TextDisplay{Content: "Success"},
	})
}
