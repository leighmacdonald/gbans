package forum

import (
	_ "embed"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/discord"
)

//go:embed forum_discord.tmpl
var templateBody []byte

type catSaveView struct {
	Category    string
	ID          int
	Description string
}

func discordCategorySave(category Category) *discordgo.MessageSend {
	content, err := discord.Render("forum_cat_save", templateBody, catSaveView{
		Category:    category.Title,
		ID:          category.ForumCategoryID,
		Description: category.Description,
	})
	if err != nil {
		slog.Error("Failed to render forum_cat_save template", slog.String("error", err.Error()))
	}

	return discord.NewMessage(discord.BodyColouredText(discord.ColourSuccess, content))
}

func discordCategoryDelete(category Category) *discordgo.MessageSend {
	content, err := discord.Render("forum_cat_deleted", templateBody, catSaveView{
		Category:    category.Title,
		ID:          category.ForumCategoryID,
		Description: category.Description,
	})
	if err != nil {
		slog.Error("Failed to render forum_cat_deleted template", slog.String("error", err.Error()))
	}

	return discord.NewMessage(discord.BodyColouredText(discord.ColourError, content))
}

type messageSaveView struct {
	Category string
	Body     string
}

func discordForumMessageSaved(forumMessage Message) *discordgo.MessageSend {
	content, err := discord.Render("forum_message_created", templateBody, messageSaveView{
		Category: forumMessage.Title,
		Body:     forumMessage.BodyMD,
	})
	if err != nil {
		slog.Error("Failed to render forum_message_saved template", slog.String("error", err.Error()))
	}

	return discord.NewMessage(discord.BodyColouredText(discord.ColourSuccess, content))
}

type forumSaveView struct {
	Forum       string
	Description string
}

func discordForumSaved(forumMessage Forum) *discordgo.MessageSend {
	content, err := discord.Render("forum_forum_saved", templateBody, forumSaveView{
		Forum:       forumMessage.Title,
		Description: forumMessage.Description,
	})
	if err != nil {
		slog.Error("Failed to render forum_forum_saved template", slog.String("error", err.Error()))
	}

	return discord.NewMessage(discord.BodyColouredText(discord.ColourSuccess, content))
}
