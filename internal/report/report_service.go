package report

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type reportHandler struct {
	reports       domain.ReportUsecase
	notifications domain.NotificationUsecase
}

func NewHandler(engine *gin.Engine, reports domain.ReportUsecase, auth domain.AuthUsecase, notifications domain.NotificationUsecase) {
	handler := reportHandler{
		reports:       reports,
		notifications: notifications,
	}

	// auth
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(auth.Middleware(domain.PUser))

		// Reports
		authed.POST("/api/report", handler.onAPIPostReportCreate())
		authed.GET("/api/report/:report_id", handler.onAPIGetReport())
		authed.POST("/api/report_status/:report_id", handler.onAPISetReportStatus())
		authed.GET("/api/reports/user", handler.onAPIGetUserReports())

		// Replies
		authed.GET("/api/report/:report_id/messages", handler.onAPIGetReportMessages())
		authed.POST("/api/report/:report_id/messages", handler.onAPIPostReportMessage())
		authed.POST("/api/report/message/:report_message_id", handler.onAPIEditReportMessage())
		authed.DELETE("/api/report/message/:report_message_id", handler.onAPIDeleteReportMessage())
	}
	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(auth.Middleware(domain.PModerator))
		mod.POST("/api/reports", handler.onAPIGetAllReports())
	}
}

func (h reportHandler) onAPIPostReportCreate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser := httphelper.CurrentUserProfile(ctx)

		var req domain.RequestReportCreate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		report, errReportSave := h.reports.SaveReport(ctx, currentUser, req)
		if errReportSave != nil {
			if errors.Is(errReportSave, domain.ErrReportExists) {
				httphelper.ResponseAPIErr(ctx, http.StatusConflict, domain.ErrReportExists)

				return
			}

			httphelper.HandleErrs(ctx, errReportSave)
			slog.Error("Failed to save report", log.ErrAttr(errReportSave))

			return
		}

		ctx.JSON(http.StatusCreated, report)

		slog.Info("New report created", slog.Int64("report_id", report.ReportID))
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
			httphelper.HandleErrs(ctx, errReport)
			slog.Error("failed to get report", log.ErrAttr(errReport))

			return
		}

		ctx.JSON(http.StatusOK, report)
	}
}

func (h reportHandler) onAPIGetUserReports() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user := httphelper.CurrentUserProfile(ctx)

		reports, errReports := h.reports.GetReportsBySteamID(ctx, user.SteamID)
		if errReports != nil {
			httphelper.HandleErrs(ctx, errReports)
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
			httphelper.HandleErrs(ctx, errReports)
			slog.Error("Failed to get reports", log.ErrAttr(errReports))

			return
		}

		ctx.JSON(http.StatusOK, reports)
	}
}

func (h reportHandler) onAPISetReportStatus() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportID, errParam := httphelper.GetInt64Param(ctx, "report_id")
		if errParam != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Error("Failed to get report_id", log.ErrAttr(errParam))

			return
		}

		var req domain.RequestReportStatusUpdate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		report, err := h.reports.SetReportStatus(ctx, reportID, httphelper.CurrentUserProfile(ctx), req.Status)
		if err != nil {
			httphelper.HandleErrs(ctx, err)
			slog.Error("Failed to set report status", log.ErrAttr(err), slog.Int64("report_id", reportID), slog.String("status", req.Status.String()))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
		slog.Info("Report status changed",
			slog.Int64("report_id", report.ReportID),
			slog.String("to_status", report.ReportStatus.String()))
	}
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
	return func(ctx *gin.Context) {
		reportID, errID := httphelper.GetInt64Param(ctx, "report_id")
		if errID != nil || reportID == 0 {
			httphelper.HandleErrBadRequest(ctx)

			if errID != nil {
				slog.Warn("Failed to get report_id", log.ErrAttr(errID))
			}

			return
		}

		var req domain.RequestMessageBodyMD
		if !httphelper.Bind(ctx, &req) {
			return
		}

		curUser := httphelper.CurrentUserProfile(ctx)

		msg, errSave := h.reports.CreateReportMessage(ctx, reportID, curUser, req)
		if errSave != nil {
			httphelper.HandleErrs(ctx, errSave)
			slog.Error("Failed to save report message", log.ErrAttr(errSave), slog.Int64("report_id", reportID))

			return
		}

		ctx.JSON(http.StatusCreated, msg)

		slog.Info("New report message created",
			slog.Int64("report_id", reportID), slog.String("steam_id", curUser.SteamID.String()))
	}
}

func (h reportHandler) onAPIEditReportMessage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportMessageID, errID := httphelper.GetInt64Param(ctx, "report_message_id")
		if errID != nil {
			httphelper.HandleErrs(ctx, errID)
			slog.Warn("Failed to get report_message_id", log.ErrAttr(errID))

			return
		}

		var req domain.RequestMessageBodyMD
		if !httphelper.Bind(ctx, &req) {
			return
		}

		msg, errMsg := h.reports.EditReportMessage(ctx, reportMessageID, httphelper.CurrentUserProfile(ctx), req)
		if errMsg != nil {
			httphelper.HandleErrs(ctx, errMsg)
			slog.Error("Failed to edit report message", log.ErrAttr(errMsg))

			return
		}

		ctx.JSON(http.StatusOK, msg)
		slog.Info("Report message edited", slog.Int64("report_message_id", reportMessageID))
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

		if err := h.reports.DropReportMessage(ctx, httphelper.CurrentUserProfile(ctx), reportMessageID); err != nil {
			httphelper.HandleErrs(ctx, err)
			slog.Error("Failed to drop report message", log.ErrAttr(err))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
		slog.Info("Deleted report message", slog.Int64("report_message_id", reportMessageID))
	}
}
