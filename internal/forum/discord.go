package forum

import (
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/forum"
)

func ForumCategorySave(category forum.ForumCategory) *discordgo.MessageEmbed {
	embed := NewEmbed("Forum Category Saved")
	embed.Embed().AddField("Category", category.Title)
	embed.Embed().AddField("ID", strconv.Itoa(category.ForumCategoryID))

	if category.Description != "" {
		embed.Embed().AddField("Description", category.Description)
	}

	return embed.Embed().MessageEmbed
}

func ForumCategoryDelete(category forum.ForumCategory) *discordgo.MessageEmbed {
	embed := NewEmbed("Forum Category Deleted")
	embed.Embed().AddField("Category", category.Title)
	embed.Embed().AddField("ID", strconv.Itoa(category.ForumCategoryID))

	if category.Description != "" {
		embed.Embed().AddField("Description", category.Description)
	}

	return embed.Embed().MessageEmbed
}

func ForumMessageSaved(message forum.ForumMessage) *discordgo.MessageEmbed {
	embed := NewEmbed("Forum Message Created/Edited", message.BodyMD)
	embed.Embed().
		AddField("Category", message.Title)

	if message.Personaname != "" {
		embed.Embed().Author = &discordgo.MessageEmbedAuthor{
			IconURL: domain.NewAvatar(message.Avatarhash).Medium(),
			Name:    message.Personaname,
		}
	}

	return embed.Embed().MessageEmbed
}

func ForumSaved(message forum.Forum) *discordgo.MessageEmbed {
	embed := NewEmbed("Forum Created/Edited")
	embed.Embed().
		AddField("Forum", message.Title)

	if message.Description != "" {
		embed.Embed().AddField("Description", message.Description)
	}

	return embed.Embed().MessageEmbed
}
