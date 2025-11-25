package chat

import (
	"context"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/datetime"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/ptr"
)

func RegisterDiscordCommands(bot discord.Service, wordFilters WordFilters) {
	handler := &discordHandler{wordFilters: wordFilters}

	bot.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
		Name:                     "filter",
		Description:              "Manage and test global word filters",
		Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
		DefaultMemberPermissions: ptr.To(discord.ModPerms),
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

func (h discordHandler) onFilterCheck(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) error {
	// opts := discord.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	// message := opts[discord.OptMessage].StringValue()
	// return FilterCheckMessage(h.wordFilters.Check(message))
	return nil
}

func (h discordHandler) makeOnFilter(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	switch interaction.ApplicationCommandData().Options[0].Name {
	case "check":
		return h.onFilterCheck(ctx, session, interaction)
	default:
		return discord.ErrCommandFailed
	}
}

const format = ` # Title
{{range .Matches }}
ID: {{ .FilterID }} Pattern: {{ .Pattern }}
{{end}}
`

func FilterCheckMessage(matches []Filter) *discordgo.MessageSend {
	var colour int
	var title string
	if len(matches) == 0 {
		title = "No Matches Found"
		colour = discord.ColourSuccess
	} else {
		title = "Matched Found"
		colour = discord.ColourWarn
	}

	content, err := discord.Render("lang_check", format, struct {
		Title   string
		Matches []Filter
	}{
		Title:   title,
		Matches: matches,
	})
	if err != nil {
		slog.Error("Failed to render check message", slog.String("error", err.Error()))
	}

	return discord.NewMessageSend(discordgo.Container{
		AccentColor: ptr.To(colour),
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: content},
		},
	})
}

func WarningMessage(newWarning NewUserWarning, validUntil time.Time) *discordgo.MessageSend {
	const format = `# Language Warning
FilterID: {{ .FilterID }}
Matched: {{ .Matched }}
Server: {{ .Server }}
Pattern: {{ .Pattern }}
Expires In: {{ .ExpiresIn }}
Expires At: {{ .ExpiresAt }}`

	var (
		expIn = "Permanent"
		expAt = expIn
	)

	if validUntil.Year()-time.Now().Year() < 5 {
		expIn = datetime.FmtDuration(validUntil)
		expAt = datetime.FmtTimeShort(validUntil)
	}

	content, err := discord.Render("lang_warn", format, struct {
		FilterID  int64
		Matched   string
		Server    string
		Pattern   string
		ExpiresIn string
		ExpiresAt string
	}{
		FilterID:  newWarning.MatchedFilter.FilterID,
		Matched:   newWarning.Matched,
		Server:    newWarning.UserMessage.ServerName,
		Pattern:   newWarning.MatchedFilter.Pattern,
		ExpiresIn: expIn,
		ExpiresAt: expAt,
	})
	if err != nil {
		slog.Error("Failed to render warning message", slog.String("error", err.Error()))
	}

	return discord.NewMessageSend(discordgo.Container{
		AccentColor: ptr.To(discord.ColourInfo),
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: content},
		},
	})
}
