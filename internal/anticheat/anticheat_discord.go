package anticheat

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/discordgo-lipstick/bot"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

func RegisterDiscordCommands(bot *bot.Bot, anticheat AntiCheat, config *config.Configuration) {
	handler := discordHandler{anticheat: anticheat, config: config}

	bot.MustRegisterHandler("ac", &discordgo.ApplicationCommand{
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

type discordHandler struct {
	anticheat AntiCheat
	persons   person.Provider
	config    *config.Configuration
}

func (h discordHandler) onAC(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	name := interaction.ApplicationCommandData().Options[0].Name
	switch name {
	case "player":
		return h.onACPlayer(ctx, session, interaction)
	default:
		return nil, discord.ErrCommandFailed
	}
}

func (h discordHandler) onACPlayer(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := bot.OptionMap(interaction.ApplicationCommandData().Options[0].Options)

	steamID, errResolveSID := steamid.Resolve(ctx, opts[discord.OptUserIdentifier].StringValue())
	if errResolveSID != nil || !steamID.Valid() {
		return nil, domain.ErrInvalidSID
	}

	person, errAuthor := h.persons.GetOrCreatePersonBySteamID(ctx, steamID)
	if errAuthor != nil {
		return nil, errAuthor
	}

	logs, errQuery := h.anticheat.Query(ctx, Query{SteamID: steamID.String()})
	if errQuery != nil {
		return nil, errQuery
	}

	return ACPlayerLogs(h.config, person, logs), nil
}

func NewAnticheatTrigger(ban ban.Ban, action config.Action, entry logparse.StacEntry, count int) *discordgo.MessageEmbed {
	embed := discord.NewEmbed("Player triggered anti-cheat response")
	embed.Embed().
		SetColor(discord.ColourSuccess).
		AddField("Detection", string(entry.Detection)).
		AddField("Count", strconv.Itoa(count)).
		AddField("Action", string(action))

	if entry.DemoName != "" {
		embed.Emb.AddField("Demo Name", entry.DemoName)
		embed.Emb.AddField("Demo Tick", strconv.Itoa(entry.DemoTick))
	}

	embed = embed.AddFieldsSteamID(entry.SteamID)

	if ban.Note != "" {
		embed.Emb.Description = "```\n" + entry.RawLog + "\n```"
	}

	return embed.Embed().MessageEmbed
}

func ACPlayerLogs(conf *config.Configuration, person person.Info, entries []Entry) *discordgo.MessageEmbed {
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
		SetAuthor(person.GetName(), person.GetAvatar().Small(), conf.ExtURL(person))

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
