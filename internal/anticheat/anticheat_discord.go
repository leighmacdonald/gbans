package anticheat

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/pkg/logparse"
)

type discordHandler struct {
	AntiCheat

	persons person.Provider
}

func RegisterDiscordCommands(bot discord.Service, anticheat AntiCheat) {
	handler := discordHandler{AntiCheat: anticheat}

	bot.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
		Name:                     "anticheat",
		Description:              "Query Anticheat Logs",
		DefaultMemberPermissions: &discord.ModPerms,
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

func (h discordHandler) onACPlayer(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	//opts := discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	//
	//uid, found := opts[discord.OptUserIdentifier]
	//if !found {
	//	return steamid.ErrInvalidSID
	//}
	//
	//steamID, errResolveSID := steamid.Resolve(ctx, uid.StringValue())
	//if errResolveSID != nil || !steamID.Valid() {
	//	return steamid.ErrInvalidSID
	//}
	//
	//player, errAuthor := h.persons.GetOrCreatePersonBySteamID(ctx, steamID)
	//if errAuthor != nil {
	//	return errAuthor
	//}
	//
	//logs, errQuery := h.Query(ctx, Query{SteamID: steamID.String()})
	//if errQuery != nil {
	//	return errQuery
	//}
	//
	//return ACPlayerLogs(player, logs)
	return nil
}

func NewAnticheatTrigger(note string, action Action, entry logparse.StacEntry, count int) *discordgo.MessageSend {
	const acTemplate = `# Player triggered anti-cheat response

Dection {{ .Detection }}
Count: {{ .Count }}
Action {{ .Action }}
{{- if ne .Entry.DemoName "" }}
Demo Name: {{ .Entry.DemoName }}
Demo Tick: {{ .Entry.Tick }}
{{ end }}
{{ .Note  }}
`
	content, errContent := discord.Render("ac", acTemplate, struct {
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

	return discord.NewMessageSend(
		discordgo.Container{
			AccentColor: ptr.To(discord.ColourSuccess),
			Components: []discordgo.MessageComponent{
				discordgo.TextDisplay{Content: content},
			},
		},
	)
}

func ACPlayerLogs(person person.Info, entries []Entry) *discordgo.MessageEmbed {
	sid := person.GetSteamID()
	emb := discord.NewEmbed()

	total := 0
	detections := map[logparse.Detection]int{}

	for _, entry := range entries {
		if _, ok := detections[entry.Detection]; !ok {
			detections[entry.Detection] = 0
		}

		detections[entry.Detection]++
		total++
	}

	emb.Embed().
		SetTitle(fmt.Sprintf("Anticheat Detections (count: %d)", total)).
		SetColor(discord.ColourSuccess).
		SetAuthor(person.GetName(), person.GetAvatar().Small(), link.Path(person))

	j := 0
	for server, count := range detections {
		emb.Embed().AddField("Detection: "+string(server), strconv.Itoa(count))
		j++
		if j < len(detections) {
			emb.Emb.MakeFieldInline()
		}
	}

	servers := map[string]int{}

	for _, entry := range entries {
		if _, ok := servers[entry.ServerName]; !ok {
			servers[entry.ServerName] = 0
		}

		servers[entry.ServerName]++
	}

	i := 0
	for server, count := range servers {
		emb.Embed().AddField("Server: "+server, strconv.Itoa(count))
		i++
		if i < len(servers) {
			emb.Emb.MakeFieldInline()
		}
	}

	emb.AddFieldsSteamID(sid)

	return emb.Embed().MessageEmbed
}
