package forum

import (
	_ "embed"
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain/person"
)

//go:embed forum_discord.tmpl
var templateBody []byte

type catSaveView struct {
	Category    string
	ID          int
	Description string
}

func RegisterDiscordCommands(_ discord.Service) {
	discord.MustRegisterTemplate(templateBody)
}

func discordCategorySave(category Category) *discordgo.MessageSend {
	content, err := discord.RenderTemplate("forum_cat_save", catSaveView{
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
	content, err := discord.RenderTemplate("forum_cat_deleted", catSaveView{
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
	Msg     *Message
	Author  person.Info
	Parents parents
}

func discordForumMessageSaved(parent parents, author person.Info, forumMessage *Message) *discordgo.MessageSend {
	return discord.NewMessage(
		discord.Heading("Forum Message Created/Edited"),
		discord.BodyTextWithThumbnailT(discord.ColourSuccess,
			discord.PlayerThumbnail(author), "forum_message_created", messageSaveView{
				Parents: parent, Author: author, Msg: forumMessage,
			}),
		discord.Buttons(
			discord.Link("🔎 View", link.Path(forumMessage)),
			discord.Button(discordgo.SecondaryButton, "Reply",
				fmt.Sprintf("reply_forum_message_%d", forumMessage.ForumMessageID)),
		),
	)
}

type forumSaveView struct {
	Forum       string
	Description string
}

func discordForumSaved(forumMessage Forum) *discordgo.MessageSend {
	content, err := discord.RenderTemplate("forum_forum_saved", forumSaveView{
		Forum:       forumMessage.Title,
		Description: forumMessage.Description,
	})
	if err != nil {
		slog.Error("Failed to render forum_forum_saved template", slog.String("error", err.Error()))
	}

	return discord.NewMessage(discord.BodyColouredText(discord.ColourSuccess, content))
}
