package ban

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/discord"
)

func (h discordHandler) onAppealReplySubmit(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) error {
	return nil
}

func appealAuthorPath(msg AppealMessage) string {
	return link.Raw(fmt.Sprintf("/profile/%d", msg.AuthorID.Int64()))
}

func appealAuthorMarkdown(msg AppealMessage) string {
	name := msg.Personaname
	if name == "" {
		name = msg.AuthorID.String()
	}

	return fmt.Sprintf("By: **[%s](%s)** Steam ID: **%s**", name, appealAuthorPath(msg), msg.AuthorID.String())
}

func newAppealMessageResponse(msg AppealMessage) *discordgo.MessageSend {
	return discord.NewMessage(
		discord.Heading("Appeal Message Create/Edited"),
		discord.BodyText(discord.HydrateLinks()(fmt.Sprintf("%s\n\n%s", appealAuthorMarkdown(msg), msg.MessageMD))),
		discord.Buttons(
			discord.Button(discordgo.PrimaryButton, "💬 Reply", fmt.Sprintf("appeal_reply_button_resp_%d", msg.BanID)),
			discord.Button(discordgo.DangerButton, "❌️ Delete", fmt.Sprintf("appeal_delete_button_resp_%d", msg.BanMessageID)),
			// discord.Button(discordgo.SecondaryButton, "🚦 Status", fmt.Sprintf("appeal_status_button_resp_%d", msg.BanID)),
			discord.Link("🔎 View", link.Path(msg)),
			discord.Link("👤 Author", appealAuthorPath(msg)),
		))
}

func newAppealMessageDelete(msg AppealMessage) *discordgo.MessageSend {
	return discord.NewMessage(
		discord.Heading("Appeal Message Deleted"),
		discord.BodyColouredText(discord.ColourError, discord.HydrateLinks()(fmt.Sprintf("%s\n\n%s", appealAuthorMarkdown(msg), msg.MessageMD))),
		discord.Buttons(
			discord.Link("🔎 View", link.Path(msg)),
			discord.Link("👤 Author", appealAuthorPath(msg)),
		),
	)
}
