package ban

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/ptr"
)

func newInGameReportResponse(report ReportWithAuthor) *discordgo.MessageSend {
	content, errContent := discord.Render("report_new", templateBody, struct {
		Report ReportWithAuthor
	}{
		Report: report,
	})
	if errContent != nil {
		return nil
	}

	return discord.NewMessage(
		discord.BodyText(content),
		discordgo.Section{
			Components: []discordgo.MessageComponent{
				discordgo.TextDisplay{Content: report.Description},
			},
			Accessory: discordgo.Thumbnail{
				Media:       discordgo.UnfurledMediaItem{URL: report.Subject.GetAvatar().Full()},
				Description: ptr.To(report.Subject.GetName()),
			},
		},
		discord.Buttons(
			discordgo.Button{
				Label:    "🔨 Reply",
				CustomID: fmt.Sprintf("report_reply_button_resp_%d", report.ReportID),
				Style:    discordgo.SecondaryButton,
			},
			discordgo.Button{
				Label:    "✏️ Set State",
				CustomID: fmt.Sprintf("report_state_button_resp_%d", report.ReportID),
				Style:    discordgo.SecondaryButton,
			},
			discordgo.Button{
				Label: "🔎 View",
				URL:   link.Path(report),
				Style: discordgo.LinkButton,
			}))
}

func (h discordHandler) onReportReplyButton(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	reportID, errReportID := discord.CustomIDInt64(interaction.MessageComponentData().CustomID)
	if errReportID != nil {
		return errReportID
	}

	caller, errCaller := h.discord.GetPersonByDiscordID(ctx, interaction.Member.User.ID)
	if errCaller != nil {
		return errCaller
	}

	report, errReport := h.reports.Report(ctx, caller, reportID)
	if errReport != nil {
		return errReport
	}

	return discord.Respond(session, interaction, []discordgo.MessageComponent{
		discordgo.TextDisplay{Content: report.Description},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					ID:          int(discord.IDBody),
					CustomID:    "reply_message_",
					Label:       "Response",
					Style:       discordgo.TextInputParagraph,
					Placeholder: "Finally took a shower",
					Required:    true,
					MaxLength:   2000,
					MinLength:   10,
				},
			},
		},
	})
}

type replyRequestModal struct {
	BodyMD string `id:"6"`
}

func (h discordHandler) onReportReplySubmit(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	reportID, errReportID := discord.CustomIDInt64(interaction.ModalSubmitData().CustomID)
	if errReportID != nil {
		return errReportID
	}

	req, errReq := discord.Bind[replyRequestModal](ctx, interaction.ModalSubmitData().Components)
	if errReq != nil {
		return errReq
	}

	caller, errCaller := h.discord.GetPersonByDiscordID(ctx, interaction.Member.User.ID)
	if errCaller != nil {
		return errCaller
	}

	report, errReport := h.reports.Report(ctx, caller, reportID)
	if errReport != nil {
		return errReport
	}

	_, errMsg := h.reports.CreateMessage(ctx, reportID, caller, RequestMessageBodyMD(req))
	if errMsg != nil {
		return errMsg
	}

	return discord.Respond(session, interaction, []discordgo.MessageComponent{
		discord.BodyColouredText(discord.ColourSuccess,
			fmt.Sprintf("Reply successful [View](%s)", link.Path(report))),
	})
}

func ReportStatusChangeMessage(report ReportWithAuthor, fromStatus ReportStatus) *discordgo.MessageSend {
	content, errContent := discord.Render("report_status_change", templateBody, nil)
	if errContent != nil {
		return nil
	}

	return discord.NewMessage(
		discordgo.TextDisplay{Content: content},
		discordgo.TextDisplay{Content: fmt.Sprintf("Changed from %s to %s", fromStatus.String(), report.ReportStatus.String())},
		discord.Buttons(discordgo.Button{
			Label: "🔎 View Report",
			URL:   link.Path(report),
			Style: discordgo.LinkButton,
		},
		),
	)
}

func NewReportMessageResponse(report ReportWithAuthor, msg ReportMessage) *discordgo.MessageSend {
	content, errContent := discord.Render("report_message_new", templateBody, struct {
		Message string
	}{Message: msg.MessageMD})
	if errContent != nil {
		return nil
	}

	return discord.NewMessage(
		discord.BodyText(content),
		discord.Buttons(discordgo.Button{
			Label:    "💬 Reply",
			CustomID: fmt.Sprintf("report_reply_button_resp_%d", report.ReportID),
			Style:    discordgo.PrimaryButton,
		},
			discordgo.Button{
				Label:    "❌️ Delete",
				CustomID: fmt.Sprintf("report_delete_button_resp_%d", report.ReportID),
				Style:    discordgo.SecondaryButton,
			},
			discordgo.Button{
				Label:    "🚦 Status",
				CustomID: fmt.Sprintf("report_status_button_resp_%d", report.ReportID),
				Style:    discordgo.SecondaryButton,
			},
			discordgo.Button{
				Label: "🔎 View",
				URL:   link.Path(report),
				Style: discordgo.LinkButton,
			}))
}
