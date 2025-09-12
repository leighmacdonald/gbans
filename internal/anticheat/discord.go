package anticheat

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const CmdAC = "ac"

func makeOnAC() func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		name := interaction.ApplicationCommandData().Options[0].Name
		switch name {
		case "player":
			return h.onACPlayer(ctx, session, interaction)
		// case string(cmdStatsGlobal):
		//	return discord.onStatsGlobal(ctx, session, interaction, response)
		// case string(cmdStatsServer):
		//	return discord.onStatsServer(ctx, session, interaction, response)
		default:
			return nil, ErrCommandFailed
		}
	}
}

func onACPlayer(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := OptionMap(interaction.ApplicationCommandData().Options[0].Options)

	steamID, errResolveSID := steamid.Resolve(ctx, opts[OptUserIdentifier].StringValue())
	if errResolveSID != nil || !steamID.Valid() {
		return nil, domain.ErrInvalidSID
	}

	person, errAuthor := h.persons.GetPersonBySteamID(ctx, nil, steamID)
	if errAuthor != nil {
		return nil, errAuthor
	}

	logs, errQuery := h.anticheat.Query(ctx, AnticheatQuery{SteamID: steamID.String()})
	if errQuery != nil {
		return nil, errQuery
	}

	return message.ACPlayerLogs(h.config, person, logs), nil
}

func NewAnticheatTrigger(ban ban.BannedPerson, config *config.Config, entry logparse.StacEntry, count int) *discordgo.MessageEmbed {
	embed := NewEmbed("Player triggered anti-cheat response")
	embed.Embed().
		SetColor(ColourSuccess).
		AddField("Detection", string(entry.Detection)).
		AddField("Count", strconv.Itoa(count)).
		AddField("Action", string(config.Anticheat.Action))

	if entry.DemoName != "" {
		embed.emb.AddField("Demo Name", entry.DemoName)
		embed.emb.AddField("Demo Tick", strconv.Itoa(entry.DemoTick))
	}

	embed = embed.AddFieldsSteamID(entry.SteamID)

	if ban.Note != "" {
		embed.emb.Description = "```\n" + entry.RawLog + "\n```"
	}

	return embed.Embed().MessageEmbed
}

func ACPlayerLogs(conf *config.ConfigUsecase, person domain.PersonInfo, entries []AnticheatEntry) *discordgo.MessageEmbed {
	sid := person.GetSteamID()
	emb := NewEmbed()

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
		SetColor(ColourSuccess).
		SetAuthor(person.GetName(), person.GetAvatar().Small(), conf.ExtURL(person))

	j := 0
	for server, count := range detections {
		emb.Embed().AddField("Detection: "+string(server), strconv.Itoa(count))
		j++
		if j < len(detections) {
			emb.emb.MakeFieldInline()
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
			emb.emb.MakeFieldInline()
		}
	}

	emb.AddFieldsSteamID(sid)

	return emb.Embed().MessageEmbed
}
