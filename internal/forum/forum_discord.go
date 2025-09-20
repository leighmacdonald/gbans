package forum

import (
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/discord/message"
	"github.com/leighmacdonald/gbans/internal/domain"
)

type discordHandler struct{}

func discordCategorySave(category Category) *discordgo.MessageEmbed {
	embed := message.NewEmbed("Forum Category Saved")
	embed.Embed().AddField("Category", category.Title)
	embed.Embed().AddField("ID", strconv.Itoa(category.ForumCategoryID))

	if category.Description != "" {
		embed.Embed().AddField("Description", category.Description)
	}

	return embed.Embed().MessageEmbed
}

func discordCategoryDelete(category Category) *discordgo.MessageEmbed {
	embed := message.NewEmbed("Forum Category Deleted")
	embed.Embed().AddField("Category", category.Title)
	embed.Embed().AddField("ID", strconv.Itoa(category.ForumCategoryID))

	if category.Description != "" {
		embed.Embed().AddField("Description", category.Description)
	}

	return embed.Embed().MessageEmbed
}

func discordForunMessageSaved(forumMessage Message) *discordgo.MessageEmbed {
	embed := message.NewEmbed("Forum Message Created/Edited", forumMessage.BodyMD)
	embed.Embed().
		AddField("Category", forumMessage.Title)

	if forumMessage.Personaname != "" {
		embed.Embed().Author = &discordgo.MessageEmbedAuthor{
			IconURL: domain.NewAvatar(forumMessage.Avatarhash).Medium(),
			Name:    forumMessage.Personaname,
		}
	}

	return embed.Embed().MessageEmbed
}

func discordForumSaved(forumMessage Forum) *discordgo.MessageEmbed {
	embed := message.NewEmbed("Forum Created/Edited")
	embed.Embed().
		AddField("Forum", forumMessage.Title)

	if forumMessage.Description != "" {
		embed.Embed().AddField("Description", forumMessage.Description)
	}

	return embed.Embed().MessageEmbed
}
