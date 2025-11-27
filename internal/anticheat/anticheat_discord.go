package anticheat

import (
	"context"
	_ "embed"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

//go:embed anticheat_discord.tmpl
var templateBody []byte

type discordHandler struct {
	AntiCheat

	persons person.Provider
}

func RegisterDiscordCommands(bot discord.Service, anticheat AntiCheat) {
	handler := discordHandler{AntiCheat: anticheat}

	bot.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
		Name:                     "anticheat",
		Description:              "Query Anticheat Logs",
		DefaultMemberPermissions: ptr.To(discord.ModPerms),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "player",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Description: "Query a players anticheat logs by steam id",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        discord.OptUserIdentifier,
						Description: "SteamID in any format OR profile url",
						Required:    true,
					},
				},
			},
		},
	}, handler.onAC)
}

func (h discordHandler) onAC(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	name := interaction.ApplicationCommandData().Options[0].Name
	switch name {
	case "player":
		return h.onACPlayer(ctx, session, interaction)
	default:
		return discord.ErrCommandFailed
	}
}

func (h discordHandler) onACPlayer(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	if err := discord.AckInteraction(session, interaction); err != nil {
		return err
	}

	opts := discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)

	uid, found := opts[discord.OptUserIdentifier]
	if !found {
		return steamid.ErrInvalidSID
	}

	steamID, errResolveSID := steamid.Resolve(ctx, uid.StringValue())
	if errResolveSID != nil || !steamID.Valid() {
		return steamid.ErrInvalidSID
	}

	player, errAuthor := h.persons.GetOrCreatePersonBySteamID(ctx, steamID)
	if errAuthor != nil {
		return errAuthor
	}

	logs, errQuery := h.Query(ctx, Query{SteamID: steamID.String()})
	if errQuery != nil {
		return errQuery
	}

	return discord.RespondUpdate(session, interaction, ACPlayerLogs(player, logs)...)
}

func ACPlayerLogs(_ person.Info, entries []Entry) []discordgo.MessageComponent {
	total := 0
	detections := map[logparse.Detection]int{}

	for _, entry := range entries {
		if _, ok := detections[entry.Detection]; !ok {
			detections[entry.Detection] = 0
		}

		detections[entry.Detection]++
		total++
	}

	servers := map[string]int{}
	for _, entry := range entries {
		if _, ok := servers[entry.ServerName]; !ok {
			servers[entry.ServerName] = 0
		}

		servers[entry.ServerName]++
	}

	content, err := discord.Render("ac_logs", templateBody, struct {
		Detections map[logparse.Detection]int
		Servers    map[string]int
	}{
		Detections: detections,
		Servers:    servers,
	})
	if err != nil {
		slog.Error("Failed to render template", slog.String("error", err.Error()))

		return nil
	}

	return []discordgo.MessageComponent{
		discordgo.Container{
			Components: []discordgo.MessageComponent{
				discordgo.TextDisplay{Content: content},
			},
		},
	}
}

func NewAnticheatTrigger(note string, action Action, entry logparse.StacEntry, count int) *discordgo.MessageSend {
	content, errContent := discord.Render("ac_trigger", templateBody, struct {
		Detection string
		Count     int
		Action    string
		Note      string
		Entry     logparse.StacEntry
	}{
		Detection: string(entry.Detection),
		Count:     count,
		Action:    string(action),
		Note:      note,
		Entry:     entry,
	})
	if errContent != nil {
		slog.Error("Failed to render template", slog.String("error", errContent.Error()))
	}

	return discord.NewMessage(
		discordgo.Container{
			AccentColor: ptr.To(discord.ColourSuccess),
			Components: []discordgo.MessageComponent{
				discordgo.TextDisplay{Content: content},
			},
		})
}
