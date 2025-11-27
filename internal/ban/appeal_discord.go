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
			discordgo.Button{
				Label:    "💬 Reply",
				CustomID: fmt.Sprintf("appeal_reply_button_resp_%d", msg.BanID),
				Style:    discordgo.PrimaryButton,
			},
			discordgo.Button{
				Label:    "❌️ Delete",
				CustomID: fmt.Sprintf("appeal_delete_button_resp_%d", msg.BanMessageID),
				Style:    discordgo.DangerButton,
			},
			discordgo.Button{
				Label:    "🚦 Status",
				CustomID: fmt.Sprintf("appeal_status_button_resp_%d", msg.BanID),
				Style:    discordgo.SecondaryButton,
			},
			discordgo.Button{
				Label: "🔎 View",
				URL:   link.Path(msg),
				Style: discordgo.LinkButton,
			}))
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
		discord.Buttons(
			discordgo.Button{
				Label: "🔎 View",
				URL:   link.Path(msg),
				Style: discordgo.LinkButton,
			}),
	)
}
