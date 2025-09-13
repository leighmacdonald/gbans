package chat

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/discord/helper"
	"github.com/leighmacdonald/gbans/internal/discord/message"
	"github.com/leighmacdonald/gbans/pkg/datetime"
)

var slashCommands = []*discordgo.ApplicationCommand{
	{
		Name:                     "filter",
		Description:              "Manage and test global word filters",
		DMPermission:             &helper.DmPerms,
		DefaultMemberPermissions: &helper.ModPerms,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "add",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Description: "Add a new filtered word",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionBoolean,
						Name:        helper.OptIsRegex,
						Description: "Is the pattern a regular expression?",
						Required:    true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        helper.OptPattern,
						Description: "Regular expression or word for matching",
						Required:    true,
					},
				},
			},
			{
				Name:        "del",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Description: "Remove a filtered word",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "filter",
						Description: "Filter ID",
						Required:    true,
					},
				},
			},
			{
				Name:        "check",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Description: "Check if a string has a matching filter",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        helper.OptMessage,
						Description: "String to check filters against",
						Required:    true,
					},
				},
			},
		},
	},
}

type DiscordHandler struct {
	wordFilters WordFilterUsecase
}

func (h DiscordHandler) onFilterCheck(_ context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := helper.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	message := opts[helper.OptMessage].StringValue()

	return FilterCheckMessage(h.wordFilters.Check(message)), nil
}

func (h DiscordHandler) makeOnFilter(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	switch interaction.ApplicationCommandData().Options[0].Name {
	case "check":
		return h.onFilterCheck(ctx, session, interaction)
	default:
		return nil, helper.ErrCommandFailed
	}
}

func FilterAddMessage(filter Filter) *discordgo.MessageEmbed {
	msgEmbed := message.NewEmbed("Filter Created Successfully").Embed().
		SetColor(message.ColourSuccess).
		AddField("pattern", filter.Pattern).
		Truncate()

	return msgEmbed.MessageEmbed
}

func FilterDelMessage(filter Filter) *discordgo.MessageEmbed {
	return message.NewEmbed("Filter Deleted Successfully").
		Embed().
		SetColor(message.ColourSuccess).
		AddField("filter", filter.Pattern).
		Truncate().MessageEmbed
}

func FilterCheckMessage(matches []Filter) *discordgo.MessageEmbed {
	msgEmbed := message.NewEmbed()
	if len(matches) == 0 {
		msgEmbed.Embed().SetTitle("No Matches Found")
		msgEmbed.Embed().SetColor(message.ColourSuccess)
	} else {
		msgEmbed.Embed().SetTitle("Matched Found")
		msgEmbed.Embed().SetColor(message.ColourWarn)

		for _, match := range matches {
			msgEmbed.Embed().AddField(fmt.Sprintf("Matched ID: %d", match.FilterID), match.Pattern)
		}
	}

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func WarningMessage(newWarning NewUserWarning, banSteam ban.Ban) *discordgo.MessageEmbed {
	msgEmbed := message.NewEmbed("Language Warning")
	msgEmbed.Embed().
		SetDescription(newWarning.UserWarning.Message).
		SetColor(message.ColourWarn).
		AddField("Filter ID", strconv.FormatInt(newWarning.MatchedFilter.FilterID, 10)).
		AddField("Matched", newWarning.Matched).
		AddField("ServerStore", newWarning.UserMessage.ServerName).InlineAllFields().
		AddField("Pattern", newWarning.MatchedFilter.Pattern)

	// TODO
	// msgEmbed.
	// 	AddFieldsSteamID(newWarning.UserMessage.SteamID).
	// 	Embed().
	// 	AddField("Name", banSteam.SourcePersonaname)

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
