package chat

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/discordgo-lipstick/bot"
	"github.com/leighmacdonald/gbans/internal/datetime"
	"github.com/leighmacdonald/gbans/internal/discord"
)

func RegisterDiscordCommands(bot discord.Service, wordFilters WordFilters) {
	handler := &discordHandler{wordFilters: wordFilters}

	bot.MustRegisterHandler("filter", &discordgo.ApplicationCommand{
		Name:                     "filter",
		Description:              "Manage and test global word filters",
		DMPermission:             &discord.DmPerms,
		DefaultMemberPermissions: &discord.ModPerms,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "add",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Description: "Add a new filtered word",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionBoolean,
						Name:        discord.OptIsRegex,
						Description: "Is the pattern a regular expression?",
						Required:    true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        discord.OptPattern,
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
						Name:        discord.OptMessage,
						Description: "String to check filters against",
						Required:    true,
					},
				},
			},
		},
	}, handler.makeOnFilter)
}

type discordHandler struct {
	wordFilters WordFilters
}

func (h discordHandler) onFilterCheck(_ context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := bot.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	message := opts[discord.OptMessage].StringValue()

	return FilterCheckMessage(h.wordFilters.Check(message)), nil
}

func (h discordHandler) makeOnFilter(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	switch interaction.ApplicationCommandData().Options[0].Name {
	case "check":
		return h.onFilterCheck(ctx, session, interaction)
	default:
		return nil, discord.ErrCommandFailed
	}
}

func filterAddMessage(filter Filter) *discordgo.MessageEmbed {
	msgEmbed := discord.NewEmbed("Filter Created Successfully").Embed().
		SetColor(discord.ColourSuccess).
		AddField("pattern", filter.Pattern).
		Truncate()

	return msgEmbed.MessageEmbed
}

func filterDelMessage(filter Filter) *discordgo.MessageEmbed {
	return discord.NewEmbed("Filter Deleted Successfully").
		Embed().
		SetColor(discord.ColourSuccess).
		AddField("filter", filter.Pattern).
		Truncate().MessageEmbed
}

func FilterCheckMessage(matches []Filter) *discordgo.MessageEmbed {
	msgEmbed := discord.NewEmbed()
	if len(matches) == 0 {
		msgEmbed.Embed().SetTitle("No Matches Found")
		msgEmbed.Embed().SetColor(discord.ColourSuccess)
	} else {
		msgEmbed.Embed().SetTitle("Matched Found")
		msgEmbed.Embed().SetColor(discord.ColourWarn)

		for _, match := range matches {
			msgEmbed.Embed().AddField(fmt.Sprintf("Matched ID: %d", match.FilterID), match.Pattern)
		}
	}

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func WarningMessage(newWarning NewUserWarning, validUntil time.Time) *discordgo.MessageEmbed {
	msgEmbed := discord.NewEmbed("Language Warning")
	msgEmbed.Embed().
		SetDescription(newWarning.UserWarning.Message).
		SetColor(discord.ColourWarn).
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

	if validUntil.Year()-time.Now().Year() < 5 {
		expIn = datetime.FmtDuration(validUntil)
		expAt = datetime.FmtTimeShort(validUntil)
	}

	return msgEmbed.
		Embed().
		AddField("Expires In", expIn).
		AddField("Expires At", expAt).
		MessageEmbed
}
