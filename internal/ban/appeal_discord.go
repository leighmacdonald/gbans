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

func newAppealMessageResponse(msg AppealMessage, title string) *discordgo.MessageSend {
	return discord.NewMessage(
		discord.Heading(title),
		discord.BodyText(msg.MessageMD),
		discord.Buttons(
			discord.Button(discordgo.PrimaryButton, "💬 Reply", fmt.Sprintf("appeal_reply_button_resp_%d", msg.BanID)),
			discord.Button(discordgo.DangerButton, "❌️ Delete", fmt.Sprintf("appeal_delete_button_resp_%d", msg.BanMessageID)),
			discord.Button(discordgo.SecondaryButton, "🚦 Status", fmt.Sprintf("appeal_status_button_resp_%d", msg.BanID)),
			discord.Link("🔎 View", link.Path(msg)),
		))
}

func newAppealMessageDelete(msg AppealMessage) *discordgo.MessageSend {
	content, errContent := discord.Render("appeal_message_deleted", templateBody, struct {
		Msg AppealMessage
	}{Msg: msg})
	if errContent != nil {
		return nil
	}

	return discord.NewMessage(
		discord.BodyColouredText(discord.ColourError, content),
		discord.Buttons(discord.Link("🔎 View", link.Path(msg))),
	)
}
