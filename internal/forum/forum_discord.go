package forum

import (
	_ "embed"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/ptr"
)

//go:embed forum_discord.tmpl
var templateBody []byte

func discordCategorySave(category Category) *discordgo.MessageSend {
	content, err := discord.Render("forum_cat_save", templateBody, struct {
		Category    string
		ID          int
		Description string
	}{
		Category:    category.Title,
		ID:          category.ForumCategoryID,
		Description: category.Description,
	})
	if err != nil {
		slog.Error("Failed to render forum_cat_save template", slog.String("error", err.Error()))
	}

	return discord.NewMessageSend(discordgo.Container{
		AccentColor: ptr.To(discord.ColourSuccess),
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: content},
		},
	})
}

func discordCategoryDelete(category Category) *discordgo.MessageSend {
	content, err := discord.Render("forum_cat_deleted", templateBody, struct {
		Category    string
		ID          int
		Description string
	}{
		Category:    category.Title,
		ID:          category.ForumCategoryID,
		Description: category.Description,
	})
	if err != nil {
		slog.Error("Failed to render forum_cat_deleted template", slog.String("error", err.Error()))
	}

	return discord.NewMessageSend(discordgo.Container{
		AccentColor: ptr.To(discord.ColourError),
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: content},
		},
	})
}

func discordForumMessageSaved(forumMessage Message) *discordgo.MessageSend {
	content, err := discord.Render("forum_message_created", templateBody, struct {
		Category string
		Body     string
	}{
		Category: forumMessage.Title,
		Body:     forumMessage.BodyMD,
	})
	if err != nil {
		slog.Error("Failed to render forum_message_saved template", slog.String("error", err.Error()))
	}

	return discord.NewMessageSend(discordgo.Container{
		AccentColor: ptr.To(discord.ColourSuccess),
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: content},
		},
	})
}

func discordForumSaved(forumMessage Forum) *discordgo.MessageSend {
	content, err := discord.Render("forum_forum_saved", templateBody, struct {
		Forum       string
		Description string
	}{
		Forum:       forumMessage.Title,
		Description: forumMessage.Description,
	})
	if err != nil {
		slog.Error("Failed to render forum_forum_saved template", slog.String("error", err.Error()))
	}

	return discord.NewMessageSend(discordgo.Container{
		AccentColor: ptr.To(discord.ColourSuccess),
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: content},
		},
	})
}
