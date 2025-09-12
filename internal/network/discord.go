package network

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

func makeOnHistory() func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		switch interaction.ApplicationCommandData().Name {
		case string(CmdHistoryIP):
			return onHistoryIP(ctx, session, interaction)
		default:
			// return discord.onHistoryChat(ctx, session, interaction, response)
			return nil, ErrCommandFailed
		}
	}
}

func onHistoryIP(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := OptionMap(interaction.ApplicationCommandData().Options[0].Options)

	steamID, errResolve := steamid.Resolve(ctx, opts[OptUserIdentifier].StringValue())
	if errResolve != nil || !steamID.Valid() {
		return nil, domain.ErrInvalidSID
	}

	person, errPersonBySID := h.persons.GetOrCreatePersonBySteamID(ctx, nil, steamID)
	if errPersonBySID != nil {
		return nil, ErrCommandFailed
	}

	// TODO actually show record

	return HistoryMessage(person), nil
}

func HistoryMessage(person domain.PersonInfo) *discordgo.MessageEmbed {
	return NewEmbed("IP History of: " + person.GetName()).Embed().
		SetDescription("IP history (20 max)").
		Truncate().MessageEmbed
}
