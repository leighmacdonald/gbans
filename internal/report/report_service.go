package report

import (
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
	reportUsecase  domain.ReportUsecase
	configUsecase  domain.ConfigUsecase
	discordUsecase domain.DiscordUsecase
	personUsecase  domain.PersonUsecase
	demoUsecase    domain.DemoUsecase
}

func NewReportHandler(engine *gin.Engine, reportUsecase domain.ReportUsecase, configUsecase domain.ConfigUsecase,
	discordUsecase domain.DiscordUsecase, personUsecase domain.PersonUsecase, authUsecase domain.AuthUsecase, demoUsecase domain.DemoUsecase,
) {
	handler := reportHandler{
		reportUsecase:  reportUsecase,
		configUsecase:  configUsecase,
		discordUsecase: discordUsecase,
		personUsecase:  personUsecase,
		demoUsecase:    demoUsecase,
	}

	// auth
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(authUsecase.AuthMiddleware(domain.PUser))
		authed.POST("/api/report", handler.onAPIPostReportCreate())
		authed.GET("/api/report/:report_id", handler.onAPIGetReport())
		authed.POST("/api/reports", handler.onAPIGetReports())
		authed.POST("/api/report_status/:report_id", handler.onAPISetReportStatus())
		authed.GET("/api/report/:report_id/messages", handler.onAPIGetReportMessages())
		authed.POST("/api/report/:report_id/messages", handler.onAPIPostReportMessage())
		authed.POST("/api/report/message/:report_message_id", handler.onAPIEditReportMessage())
		authed.DELETE("/api/report/message/:report_message_id", handler.onAPIDeleteReportMessage())
	}
	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(authUsecase.AuthMiddleware(domain.PModerator))
		mod.POST("/api/report/:report_id/state", handler.onAPIPostBanState())
	}
}

func (h reportHandler) onAPIPostBanState() gin.HandlerFunc {
	// TODO doesnt do anything
	return func(ctx *gin.Context) {
		reportID, errID := httphelper.GetInt64Param(ctx, "report_id")
		if errID != nil || reportID <= 0 {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		report, errReport := h.reportUsecase.GetReport(ctx, httphelper.CurrentUserProfile(ctx), reportID)
		if errReport != nil {
			httphelper.ErrorHandled(ctx, errReport)

			return
		}

		ctx.JSON(http.StatusOK, report)

		h.discordUsecase.SendPayload(domain.ChannelModLog, discord.EditBanAppealStatusMessage())
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
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrSourceID)
			slog.Error("Invalid steam_id", slog.String("steamid", req.SourceID.String()))

			return
		}

		if !req.TargetID.Valid() {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrTargetID)
			slog.Error("Invalid target_id", slog.String("steamid", req.TargetID.String()))

			return
		}

		if req.SourceID.Int64() == req.TargetID.Int64() {
			httphelper.ResponseErr(ctx, http.StatusConflict, domain.ErrSelfReport)

			return
		}

		personSource, errSource := h.personUsecase.GetPersonBySteamID(ctx, req.SourceID)
		if errSource != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Could not load player profile", log.ErrAttr(errSource))

			return
		}

		personTarget, errTarget := h.personUsecase.GetOrCreatePersonBySteamID(ctx, req.TargetID)
		if errTarget != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Could not load player profile", log.ErrAttr(errTarget))

			return
		}

		if personTarget.Expired() {
			if err := thirdparty.UpdatePlayerSummary(ctx, &personTarget); err != nil {
				slog.Error("Failed to update target player", log.ErrAttr(err))
			} else {
				if errSave := h.personUsecase.SavePerson(ctx, &personTarget); errSave != nil {
					slog.Error("Failed to save target player update", log.ErrAttr(err))
				}
			}
		}

		// Ensure the user doesn't already have an open report against the user
		existing, errReports := h.reportUsecase.GetReportBySteamID(ctx, personSource.SteamID, req.TargetID)
		if errReports != nil {
			if !errors.Is(errReports, domain.ErrNoResult) {
				slog.Error("Failed to query reports by steam id", log.ErrAttr(errReports))
				httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

				return
			}
		}

		if existing.ReportID > 0 {
			httphelper.ResponseErr(ctx, http.StatusConflict, domain.ErrReportExists)

			return
		}

		var demo domain.DemoFile

		if req.DemoID > 0 {
			if errDemo := h.demoUsecase.GetDemoByID(ctx, req.DemoID, &demo); errDemo != nil {
				httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)
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

		if errReportSave := h.reportUsecase.SaveReport(ctx, &report); errReportSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to save report", log.ErrAttr(errReportSave))

			return
		}

		if demo.DemoID > 0 && !demo.Archive {
			demo.Archive = true

			if errMark := h.demoUsecase.MarkArchived(ctx, &demo); errMark != nil {
				slog.Error("Failed to mark demo as archived", log.ErrAttr(errMark))
			}
		}

		ctx.JSON(http.StatusCreated, report)

		slog.Info("New report created successfully", slog.Int64("report_id", report.ReportID))

		conf := h.configUsecase.Config()

		if !conf.Discord.Enabled {
			return
		}

		demoURL := ""

		if report.DemoID > 0 {
			demoURL = conf.ExtURLRaw("/asset/%s", demo.AssetID.String())
		}

		msg := discord.NewInGameReportResponse(report, conf.ExtURL(report), currentUser, conf.ExtURL(currentUser), demoURL)

		h.discordUsecase.SendPayload(domain.ChannelModLog, msg)
	}
}

func (h reportHandler) onAPIGetReport() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportID, errParam := httphelper.GetInt64Param(ctx, "report_id")
		if errParam != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		report, errReport := h.reportUsecase.GetReport(ctx, httphelper.CurrentUserProfile(ctx), reportID)
		if errReport != nil {
			httphelper.ErrorHandled(ctx, errReport)

			return
		}

		ctx.JSON(http.StatusOK, report)
	}
}

func (h reportHandler) onAPIGetReports() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user := httphelper.CurrentUserProfile(ctx)

		var req domain.ReportQueryFilter
		if !httphelper.Bind(ctx, &req) {
			return
		}

		reports, count, errReports := h.reportUsecase.GetReports(ctx, user, req)
		if errReports != nil {
			httphelper.ErrorHandled(ctx, errReports)

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(count, reports))
	}
}

func (h reportHandler) onAPISetReportStatus() gin.HandlerFunc {
	type stateUpdateReq struct {
		Status domain.ReportStatus `json:"status"`
	}

	return func(ctx *gin.Context) {
		reportID, errParam := httphelper.GetInt64Param(ctx, "report_id")
		if errParam != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var req stateUpdateReq
		if !httphelper.Bind(ctx, &req) {
			return
		}

		report, errGet := h.reportUsecase.GetReport(ctx, httphelper.CurrentUserProfile(ctx), reportID)
		if errGet != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to get report to set state", log.ErrAttr(errGet))

			return
		}

		if report.ReportStatus == req.Status {
			ctx.JSON(http.StatusConflict, domain.ErrDuplicate)

			return
		}

		original := report.ReportStatus

		report.ReportStatus = req.Status
		if errSave := h.reportUsecase.SaveReport(ctx, &report.Report); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
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
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		report, errGetReport := h.reportUsecase.GetReport(ctx, httphelper.CurrentUserProfile(ctx), reportID)
		if errGetReport != nil {
			httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

			return
		}

		if !httphelper.HasPrivilege(httphelper.CurrentUserProfile(ctx), steamid.Collection{report.SourceID, report.TargetID}, domain.PModerator) {
			return
		}

		reportMessages, errGetReportMessages := h.reportUsecase.GetReportMessages(ctx, reportID)
		if errGetReportMessages != nil {
			httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrPlayerNotFound)

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
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var req newMessage
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if req.Message == "" {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		report, errReport := h.reportUsecase.GetReport(ctx, httphelper.CurrentUserProfile(ctx), reportID)
		if errReport != nil {
			if errors.Is(errReport, domain.ErrNoResult) {
				httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to load report", log.ErrAttr(errReport))

			return
		}

		person := httphelper.CurrentUserProfile(ctx)
		msg := domain.NewReportMessage(reportID, person.SteamID, req.Message)

		if errSave := h.reportUsecase.SaveReportMessage(ctx, &msg); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to save report message", log.ErrAttr(errSave))

			return
		}

		report.UpdatedOn = time.Now()

		if errSave := h.reportUsecase.SaveReport(ctx, &report.Report); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to update report activity", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, msg)

		conf := h.configUsecase.Config()

		h.discordUsecase.SendPayload(domain.ChannelModLog,
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
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		existing, errExist := h.reportUsecase.GetReportMessageByID(ctx, reportMessageID)
		if errExist != nil {
			if errors.Is(errExist, domain.ErrNoResult) {
				httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrPlayerNotFound)

				return
			}

			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		if req.BodyMD == existing.MessageMD {
			httphelper.ResponseErr(ctx, http.StatusConflict, domain.ErrDuplicate)

			return
		}

		existing.MessageMD = req.BodyMD
		if errSave := h.reportUsecase.SaveReportMessage(ctx, &existing); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to save report message", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, existing)

		conf := h.configUsecase.Config()

		msg := discord.EditReportMessageResponse(req.BodyMD, existing.MessageMD,
			conf.ExtURLRaw("/report/%d", existing.ReportID), curUser, conf.ExtURL(curUser))

		h.discordUsecase.SendPayload(domain.ChannelModLog, msg)
	}
}

func (h reportHandler) onAPIDeleteReportMessage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportMessageID, errID := httphelper.GetInt64Param(ctx, "report_message_id")
		if errID != nil || reportMessageID == 0 {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		existing, errExist := h.reportUsecase.GetReportMessageByID(ctx, reportMessageID)
		if errExist != nil {
			if errors.Is(errExist, domain.ErrNoResult) {
				httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		curUser := httphelper.CurrentUserProfile(ctx)
		if !httphelper.HasPrivilege(curUser, steamid.Collection{existing.AuthorID}, domain.PModerator) {
			return
		}

		existing.Deleted = true
		if errSave := h.reportUsecase.SaveReportMessage(ctx, &existing); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to save report message", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusNoContent, nil)

		conf := h.configUsecase.Config()

		h.discordUsecase.SendPayload(domain.ChannelModLog, discord.DeleteReportMessage(existing, curUser, conf.ExtURL(curUser)))
	}
}
