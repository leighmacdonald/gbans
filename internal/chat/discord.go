package chat

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/gbans/pkg/datetime"
)

func onFilterCheck(_ context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	message := opts[OptMessage].StringValue()

	return message.FilterCheckMessage(h.wordFilters.Check(message)), nil
}

func makeOnFilter() func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) { //nolint:maintidx
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		switch interaction.ApplicationCommandData().Options[0].Name {
		case "check":
			return h.onFilterCheck(ctx, session, interaction)
		default:
			return nil, ErrCommandFailed
		}
	}
}

func FilterAddMessage(filter Filter) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("Filter Created Successfully").Embed().
		SetColor(ColourSuccess).
		AddField("pattern", filter.Pattern).
		Truncate()

	return msgEmbed.MessageEmbed
}

func FilterDelMessage(filter Filter) *discordgo.MessageEmbed {
	return NewEmbed("Filter Deleted Successfully").
		Embed().
		SetColor(ColourSuccess).
		AddField("filter", filter.Pattern).
		Truncate().MessageEmbed
}

func FilterCheckMessage(matches []Filter) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed()
	if len(matches) == 0 {
		msgEmbed.Embed().SetTitle("No Matches Found")
		msgEmbed.Embed().SetColor(ColourSuccess)
	} else {
		msgEmbed.Embed().SetTitle("Matched Found")
		msgEmbed.Embed().SetColor(ColourWarn)

		for _, match := range matches {
			msgEmbed.Embed().AddField(fmt.Sprintf("Matched ID: %d", match.FilterID), match.Pattern)
		}
	}

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func WarningMessage(newWarning chat.NewUserWarning, banSteam ban.BannedPerson) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("Language Warning")
	msgEmbed.Embed().
		SetDescription(newWarning.UserWarning.Message).
		SetColor(ColourWarn).
		AddField("Filter ID", strconv.FormatInt(newWarning.MatchedFilter.FilterID, 10)).
		AddField("Matched", newWarning.Matched).
		AddField("ServerStore", newWarning.UserMessage.ServerName).InlineAllFields().
		AddField("Pattern", newWarning.MatchedFilter.Pattern)

	msgEmbed.
		AddFieldsSteamID(newWarning.UserMessage.SteamID).
		Embed().
		AddField("Name", banSteam.SourcePersonaname)

	var (
		expIn = "Permanent"
		expAt = expIn
	)

	if banSteam.ValidUntil.Year()-time.Now().Year() < 5 {
		expIn = datetime.FmtDuration(banSteam.ValidUntil)
		expAt = datetime.FmtTimeShort(banSteam.ValidUntil)
	}

	return msgEmbed.
		Embed().
		AddField("Expires In", expIn).
		AddField("Expires At", expAt).
		MessageEmbed
}
