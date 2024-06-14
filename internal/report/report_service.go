package report

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type reportHandler struct {
	reports domain.ReportUsecase
	config  domain.ConfigUsecase
	discord domain.DiscordUsecase
	persons domain.PersonUsecase
	demos   domain.DemoUsecase
}

func NewReportHandler(engine *gin.Engine, reports domain.ReportUsecase, config domain.ConfigUsecase,
	discord domain.DiscordUsecase, person domain.PersonUsecase, auth domain.AuthUsecase, demos domain.DemoUsecase,
) {
	handler := reportHandler{
		reports: reports,
		config:  config,
		discord: discord,
		persons: person,
		demos:   demos,
	}

	// auth
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(auth.AuthMiddleware(domain.PUser))
		authed.POST("/api/report", handler.onAPIPostReportCreate())
		authed.GET("/api/report/:report_id", handler.onAPIGetReport())
		authed.POST("/api/report_status/:report_id", handler.onAPISetReportStatus())
		authed.GET("/api/report/:report_id/messages", handler.onAPIGetReportMessages())
		authed.POST("/api/report/:report_id/messages", handler.onAPIPostReportMessage())
		authed.POST("/api/report/message/:report_message_id", handler.onAPIEditReportMessage())
		authed.DELETE("/api/report/message/:report_message_id", handler.onAPIDeleteReportMessage())
		authed.POST("/api/reports/user", handler.onAPIGetUserReports())
	}
	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(auth.AuthMiddleware(domain.PModerator))
		mod.POST("/api/report/:report_id/state", handler.onAPIPostBanState())
		mod.POST("/api/reports", handler.onAPIGetAllReports())
	}
}

func (h reportHandler) onAPIPostBanState() gin.HandlerFunc {
	// TODO doesnt do anything
	return func(ctx *gin.Context) {
		reportID, errID := httphelper.GetInt64Param(ctx, "report_id")
		if errID != nil || reportID <= 0 {
			httphelper.HandleErrBadRequest(ctx)
			if errID != nil {
				slog.Warn("Failed to get report_id", log.ErrAttr(errID))
			}

			return
		}

		report, errReport := h.reports.GetReport(ctx, httphelper.CurrentUserProfile(ctx), reportID)
		if errReport != nil {
			httphelper.ErrorHandled(ctx, errReport)
			slog.Error("Failed to get user report", log.ErrAttr(errReport))

			return
		}

		ctx.JSON(http.StatusOK, report)

		h.discord.SendPayload(domain.ChannelModAppealLog, discord.EditBanAppealStatusMessage())
	}
}

func (h reportHandler) onAPIPostReportCreate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser := httphelper.CurrentUserProfile(ctx)

		var req domain.CreateReportReq
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if req.Description == "" || len(req.Description) < 10 {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, fmt.Errorf("%w: description", domain.ErrParamInvalid))

			return
		}

		// ServerStore initiated requests will have a sourceID set by the server
		// Web based reports the source should not be set, the reporter will be taken from the
		// current session information instead
		if !req.SourceID.Valid() {
			req.SourceID = currentUser.SteamID
		}

		if !req.SourceID.Valid() {
			httphelper.HandleErrBadRequest(ctx)
			slog.Error("Invalid steam_id", slog.String("steamid", req.SourceID.String()))

			return
		}

		if !req.TargetID.Valid() {
			httphelper.HandleErrBadRequest(ctx)
			slog.Error("Invalid target_id", slog.String("steamid", req.TargetID.String()))

			return
		}

		if req.SourceID.Int64() == req.TargetID.Int64() {
			httphelper.ResponseErr(ctx, http.StatusConflict, domain.ErrSelfReport)

			return
		}

		personSource, errSource := h.persons.GetPersonBySteamID(ctx, req.SourceID)
		if errSource != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Could not load player profile", log.ErrAttr(errSource))

			return
		}

		personTarget, errTarget := h.persons.GetOrCreatePersonBySteamID(ctx, req.TargetID)
		if errTarget != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Could not load player profile", log.ErrAttr(errTarget))

			return
		}

		if personTarget.Expired() {
			if err := thirdparty.UpdatePlayerSummary(ctx, &personTarget); err != nil {
				slog.Error("Failed to update target player", log.ErrAttr(err))
			} else {
				if errSave := h.persons.SavePerson(ctx, &personTarget); errSave != nil {
					slog.Error("Failed to save target player update", log.ErrAttr(err))
				}
			}
		}

		// Ensure the user doesn't already have an open report against the user
		existing, errReports := h.reports.GetReportBySteamID(ctx, personSource.SteamID, req.TargetID)
		if errReports != nil {
			if !errors.Is(errReports, domain.ErrNoResult) {
				httphelper.HandleErrInternal(ctx)
				slog.Error("Failed to query reports by steam id", log.ErrAttr(errReports))

				return
			}
		}

		if existing.ReportID > 0 {
			httphelper.ResponseErr(ctx, http.StatusConflict, domain.ErrReportExists)

			return
		}

		var demo domain.DemoFile

		if req.DemoID > 0 {
			if errDemo := h.demos.GetDemoByID(ctx, req.DemoID, &demo); errDemo != nil {
				httphelper.HandleErrBadRequest(ctx)
				slog.Error("Failed to load demo for report", slog.Int64("demo_id", req.DemoID))

				return
			}
		}

		// TODO encapsulate all operations in single tx
		report := domain.NewReport()
		report.SourceID = req.SourceID
		report.ReportStatus = domain.Opened
		report.Description = req.Description
		report.TargetID = req.TargetID
		report.Reason = req.Reason
		report.ReasonText = req.ReasonText
		report.DemoID = req.DemoID
		report.DemoTick = req.DemoTick
		report.PersonMessageID = req.PersonMessageID

		if errReportSave := h.reports.SaveReport(ctx, &report); errReportSave != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to save report", log.ErrAttr(errReportSave))

			return
		}

		ctx.JSON(http.StatusCreated, report)

		if demo.DemoID > 0 && !demo.Archive {
			if errMark := h.demos.MarkArchived(ctx, &demo); errMark != nil {
				slog.Error("Failed to mark demo as archived", log.ErrAttr(errMark))
			}
		}

		conf := h.config.Config()

		demoURL := ""

		if report.DemoID > 0 {
			demoURL = conf.ExtURLRaw("/asset/%s", demo.AssetID.String())
		}

		msg := discord.NewInGameReportResponse(report, conf.ExtURL(report), currentUser, conf.ExtURL(currentUser), demoURL)

		h.discord.SendPayload(domain.ChannelModAppealLog, msg)
	}
}

func (h reportHandler) onAPIGetReport() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportID, errParam := httphelper.GetInt64Param(ctx, "report_id")
		if errParam != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get report_id", log.ErrAttr(errParam))

			return
		}

		report, errReport := h.reports.GetReport(ctx, httphelper.CurrentUserProfile(ctx), reportID)
		if errReport != nil {
			httphelper.ErrorHandled(ctx, errReport)
			slog.Error("failed to get report", log.ErrAttr(errReport))

			return
		}

		ctx.JSON(http.StatusOK, report)
	}
}

func (r reportUsecase) GetReportBySteamID(ctx context.Context, authorID steamid.SteamID, steamID steamid.SteamID) (domain.Report, error) {
	return r.repository.GetReportBySteamID(ctx, authorID, steamID)
}

func (h reportHandler) onAPIGetUserReports() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user := httphelper.CurrentUserProfile(ctx)

		reports, errReports := h.reports.GetReportsBySteamID(ctx, user.SteamID)
		if errReports != nil {
			httphelper.ErrorHandled(ctx, errReports)
			slog.Error("Failed to get reports by steam id", log.ErrAttr(errReports))

			return
		}

		ctx.JSON(http.StatusOK, reports)
	}
}

func (h reportHandler) onAPIGetAllReports() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.ReportQueryFilter
		if !httphelper.Bind(ctx, &req) {
			return
		}

		reports, errReports := h.reports.GetReports(ctx)
		if errReports != nil {
			httphelper.ErrorHandled(ctx, errReports)
			slog.Error("Failed to get reports", log.ErrAttr(errReports))

			return
		}

		ctx.JSON(http.StatusOK, reports)
	}
}

func (h reportHandler) onAPISetReportStatus() gin.HandlerFunc {
	type stateUpdateReq struct {
		Status domain.ReportStatus `json:"status"`
	}

	return func(ctx *gin.Context) {
		reportID, errParam := httphelper.GetInt64Param(ctx, "report_id")
		if errParam != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Error("Failed to get report_id", log.ErrAttr(errParam))

			return
		}

		var req stateUpdateReq
		if !httphelper.Bind(ctx, &req) {
			return
		}

		report, errGet := h.reports.GetReport(ctx, httphelper.CurrentUserProfile(ctx), reportID)
		if errGet != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get report to set state", log.ErrAttr(errGet))

			return
		}

		if report.ReportStatus == req.Status {
			ctx.JSON(http.StatusConflict, domain.ErrDuplicate)

			return
		}

		original := report.ReportStatus

		report.ReportStatus = req.Status
		if errSave := h.reports.SaveReport(ctx, &report.Report); errSave != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to save report state", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusAccepted, nil)
		slog.Info("Report status changed",
			slog.Int64("report_id", report.ReportID),
			slog.String("from_status", original.String()),
			slog.String("to_status", report.ReportStatus.String()))
		// discord.SendDiscord(model.NotificationPayload{
		//	Sids:     steamid.Collection{report.SourceID},
		//	Severity: db.SeverityInfo,
		//	Message:  "Report status updated",
		//	Link:     report.ToURL(),
		// })
	} //nolint:wsl
}

func (h reportHandler) onAPIGetReportMessages() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportID, errParam := httphelper.GetInt64Param(ctx, "report_id")
		if errParam != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Got invalid report_id", log.ErrAttr(errParam))

			return
		}

		report, errGetReport := h.reports.GetReport(ctx, httphelper.CurrentUserProfile(ctx), reportID)
		if errGetReport != nil {
			httphelper.HandleErrNotFound(ctx)
			slog.Error("Failed to get report. Not found.", log.ErrAttr(errGetReport))

			return
		}

		if !httphelper.HasPrivilege(httphelper.CurrentUserProfile(ctx), steamid.Collection{report.SourceID, report.TargetID}, domain.PModerator) {
			return
		}

		reportMessages, errGetReportMessages := h.reports.GetReportMessages(ctx, reportID)
		if errGetReportMessages != nil {
			httphelper.HandleErrNotFound(ctx)
			slog.Error("Failed to get report messages", log.ErrAttr(errGetReportMessages))

			return
		}

		if reportMessages == nil {
			reportMessages = []domain.ReportMessage{}
		}

		ctx.JSON(http.StatusOK, reportMessages)
	}
}

func (h reportHandler) onAPIPostReportMessage() gin.HandlerFunc {
	type newMessage struct {
		Message string `json:"message"`
	}

	return func(ctx *gin.Context) {
		reportID, errID := httphelper.GetInt64Param(ctx, "report_id")
		if errID != nil || reportID == 0 {
			httphelper.HandleErrBadRequest(ctx)

			if errID != nil {
				slog.Warn("Failed to get report_id", log.ErrAttr(errID))
			}

			return
		}

		var req newMessage
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if req.Message == "" {
			httphelper.HandleErrBadRequest(ctx)

			return
		}

		report, errReport := h.reports.GetReport(ctx, httphelper.CurrentUserProfile(ctx), reportID)
		if errReport != nil {
			if errors.Is(errReport, domain.ErrNoResult) {
				httphelper.HandleErrNotFound(ctx)

				return
			}

			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to load report", log.ErrAttr(errReport))

			return
		}

		person := httphelper.CurrentUserProfile(ctx)
		msg := domain.NewReportMessage(reportID, person.SteamID, req.Message)

		if errSave := h.reports.SaveReportMessage(ctx, &msg); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to save report message", log.ErrAttr(errSave))

			return
		}

		report.UpdatedOn = time.Now()

		if errSave := h.reports.SaveReport(ctx, &report.Report); errSave != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to update report activity", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, msg)

		conf := h.config.Config()

		h.discord.SendPayload(domain.ChannelModAppealLog,
			discord.NewReportMessageResponse(msg.MessageMD, conf.ExtURL(report), person, conf.ExtURL(person)))
	}
}

func (h reportHandler) onAPIEditReportMessage() gin.HandlerFunc {
	type editMessage struct {
		BodyMD string `json:"body_md"`
	}

	return func(ctx *gin.Context) {
		reportMessageID, errID := httphelper.GetInt64Param(ctx, "report_message_id")
		if errID != nil || reportMessageID == 0 {
			httphelper.HandleErrBadRequest(ctx)

			if errID != nil {
				slog.Warn("Failed to get report_message_id", log.ErrAttr(errID))
			}

			return
		}

		existing, errExist := h.reports.GetReportMessageByID(ctx, reportMessageID)
		if errExist != nil {
			if errors.Is(errExist, domain.ErrNoResult) {
				httphelper.HandleErrNotFound(ctx)

				return
			}

			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get report message by id", log.ErrAttr(errExist))

			return
		}

		curUser := httphelper.CurrentUserProfile(ctx)
		if !httphelper.HasPrivilege(curUser, steamid.Collection{existing.AuthorID}, domain.PModerator) {
			return
		}

		var req editMessage
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if req.BodyMD == "" {
			httphelper.HandleErrBadRequest(ctx)

			return
		}

		if req.BodyMD == existing.MessageMD {
			httphelper.ResponseErr(ctx, http.StatusConflict, domain.ErrDuplicate)

			return
		}

		existing.MessageMD = req.BodyMD
		if errSave := h.reports.SaveReportMessage(ctx, &existing); errSave != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to save report message", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, existing)

		conf := h.config.Config()

		msg := discord.EditReportMessageResponse(req.BodyMD, existing.MessageMD,
			conf.ExtURLRaw("/report/%d", existing.ReportID), curUser, conf.ExtURL(curUser))

		h.discord.SendPayload(domain.ChannelModAppealLog, msg)
	}
}

func (h reportHandler) onAPIDeleteReportMessage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportMessageID, errID := httphelper.GetInt64Param(ctx, "report_message_id")
		if errID != nil || reportMessageID == 0 {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get report_message_id", log.ErrAttr(errID))

			return
		}

		existing, errExist := h.reports.GetReportMessageByID(ctx, reportMessageID)
		if errExist != nil {
			if errors.Is(errExist, domain.ErrNoResult) {
				httphelper.HandleErrNotFound(ctx)

				return
			}

			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get report message by id", log.ErrAttr(errExist))

			return
		}

		curUser := httphelper.CurrentUserProfile(ctx)
		if !httphelper.HasPrivilege(curUser, steamid.Collection{existing.AuthorID}, domain.PModerator) {
			return
		}

		existing.Deleted = true
		if errSave := h.reports.SaveReportMessage(ctx, &existing); errSave != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to save report message", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusNoContent, nil)

		conf := h.config.Config()

		h.discord.SendPayload(domain.ChannelModAppealLog, discord.DeleteReportMessage(existing, curUser, conf.ExtURL(curUser)))
	}
}
