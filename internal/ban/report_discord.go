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
	content, errContent := discord.RenderTemplate("report_new", struct {
		Report ReportWithAuthor
	}{
		Report: report,
	})
	if errContent != nil {
		return nil
	}

	return discord.NewMessage(
		discord.Heading("New Report Created"),
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

	return discord.Respond(session, interaction,
		discord.BodyText(report.Description),
		discord.ModalInputRowsRequired(discord.IDBody, "reply_message_", "Response", "Finally took a shower", "", 10, 2000),
	)
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

	return discord.Respond(session, interaction, discord.BodyColouredText(discord.ColourSuccess,
		fmt.Sprintf("Reply successful [View](%s)", link.Path(report))),
	)
}

func ReportStatusChangeMessage(report ReportWithAuthor, fromStatus ReportStatus) *discordgo.MessageSend {
	return discord.NewMessage(
		discord.Heading("Report Status Changed"),
		discord.BodyColouredText(discord.ColourSuccess,
			fmt.Sprintf("Changed from %s to %s", fromStatus.String(), report.ReportStatus.String())),
		discord.Buttons(discord.Link("🔎 View Report", link.Path(report))))
}

func NewReportMessageResponse(report ReportWithAuthor, msg ReportMessage) *discordgo.MessageSend {
	content, errContent := discord.RenderTemplate("report_message_new", struct {
		Message string
	}{Message: msg.MessageMD})
	if errContent != nil {
		return nil
	}

	return discord.NewMessage(
		discord.Heading("New Report Message Posted"),
		discord.BodyText(content),
		discord.Buttons(
			discord.Button(discordgo.PrimaryButton, "💬 Reply", fmt.Sprintf("report_reply_button_resp_%d", report.ReportID)),
			discord.Button(discordgo.SecondaryButton, "❌️ Delete", fmt.Sprintf("report_delete_button_resp_%d", report.ReportID)),
			discord.Button(discordgo.SecondaryButton, "🚦 Status", fmt.Sprintf("report_status_button_resp_%d", report.ReportID)),
			discord.Link("🔎 View", link.Path(report))))
}
