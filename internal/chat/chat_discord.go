package chat

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/datetime"
	"github.com/leighmacdonald/gbans/internal/discord"
)

//go:embed chat_discord.gotmpl
var templateBody []byte

func RegisterDiscordCommands(bot discord.Connection, wordFilters WordFilters) {
	discord.MustRegisterTemplate(templateBody)

	handler := &discordHandler{wordFilters: wordFilters}

	bot.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
		Name:                     "filter",
		Description:              "Manage and test global word filters",
		Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
		DefaultMemberPermissions: new(discord.ModPerms),
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
						Description: "filter ID",
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
	// return filterCheckMessage(h.wordFilters.Check(message))
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

func filterCheckMessage(matches []Filter) *discordgo.MessageSend {
	var colour int
	if len(matches) == 0 {
		colour = discord.ColourSuccess
	} else {
		colour = discord.ColourWarn
	}

	content, err := discord.RenderTemplate("filter_check", struct {
		Matches []Filter
	}{Matches: matches})
	if err != nil {
		slog.Error("Failed to render check message", slog.String("error", err.Error()))
	}

	return discord.NewMessage(discord.BodyColouredText(colour, content))
}

// SafePingDiscord reports whether filter events should be announced in Discord.
func (c *Config) SafePingDiscord() bool {
	c.RLock()
	defer c.RUnlock()

	return c.PingDiscord
}

// WarningStatus describes the player's current word-filter warning standing.
type WarningStatus struct {
	WarningCount   int
	CurrentWeight  int32
	MaxWeight      int32
	WindowResetsIn string
}

// WarningFlaggedMessage announces a chat message that matched a filter,
// including the player's current warning status.
func WarningFlaggedMessage(warning NewUserWarning, status WarningStatus) *discordgo.MessageSend {
	profileURL := link.Raw(fmt.Sprintf("/profile/%d", warning.SteamID))
	content, err := discord.RenderTemplate("filter_warning_flagged", struct {
		Player         string
		Profile        string
		SteamID        int64
		Server         string
		Pattern        string
		Matched        string
		WarningCount   int
		CurrentWeight  int32
		MaxWeight      int32
		WindowResetsIn string
	}{
		Player:         warning.Personaname,
		Profile:        profileURL,
		SteamID:        warning.SteamID,
		Server:         warning.UserMessage.ServerName,
		Pattern:        warning.MatchedFilter.Pattern,
		Matched:        warning.Matched,
		WarningCount:   status.WarningCount,
		CurrentWeight:  status.CurrentWeight,
		MaxWeight:      status.MaxWeight,
		WindowResetsIn: status.WindowResetsIn,
	})
	if err != nil {
		slog.Error("Failed to render flagged warning message", slog.String("error", err.Error()))
	}

	return discord.NewMessage(
		discord.BodyColouredText(discord.ColourWarn, content),
		discord.Buttons(
			discord.Link("👤 Player", profileURL),
			discord.Link("💬 Chat Logs", link.Raw("/chatlogs"))))
}

func WarningMessage(newWarning NewUserWarning, validUntil time.Time) *discordgo.MessageSend {
	var (
		expIn = "Permanent"
		expAt = expIn
	)

	if validUntil.Year()-time.Now().Year() < 5 {
		expIn = datetime.FmtDuration(validUntil)
		expAt = datetime.FmtTimeShort(validUntil)
	}

	profileURL := link.Raw(fmt.Sprintf("/profile/%d", newWarning.SteamID))
	content, err := discord.RenderTemplate("filter_warning", struct {
		FilterID  int64
		Matched   string
		Server    string
		Pattern   string
		ExpiresIn string
		ExpiresAt string
		Player    string
		Profile   string
		SteamID   int64
	}{
		FilterID:  newWarning.MatchedFilter.FilterID,
		Matched:   newWarning.Matched,
		Server:    newWarning.UserMessage.ServerName,
		Pattern:   newWarning.MatchedFilter.Pattern,
		ExpiresIn: expIn,
		ExpiresAt: expAt,
		Player:    newWarning.Personaname,
		Profile:   profileURL,
		SteamID:   newWarning.SteamID,
	})
	if err != nil {
		slog.Error("Failed to render warning message", slog.String("error", err.Error()))
	}

	return discord.NewMessage(
		discord.BodyColouredText(discord.ColourInfo, content),
		discord.Buttons(
			discord.Link("👤 Player", profileURL),
			discord.Link("💬 Chat Logs", link.Raw("/chatlogs"))))
}
