package report

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
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
				_ = ctx.Error(httphelper.NewAPIErrorf(ctx, http.StatusConflict, domain.ErrReportExists,
					"An open report already exists for this player, duplicates are not allowed."))

				return
			}

			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errReportSave))

			return
		}

		ctx.JSON(http.StatusCreated, report)
	}
}

func (h reportHandler) onAPIGetReport() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportID, idFound := httphelper.GetInt64Param(ctx, "report_id")
		if !idFound {
			return
		}

		report, errReport := h.reports.GetReport(ctx, httphelper.CurrentUserProfile(ctx), reportID)
		if errReport != nil {
			_ = ctx.Error(httphelper.NewAPIErrorf(ctx, http.StatusInternalServerError, errReport,
				"Could not find a report with the id: %d", reportID))

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
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errReports))

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
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errReports))

			return
		}

		ctx.JSON(http.StatusOK, reports)
	}
}

func (h reportHandler) onAPISetReportStatus() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportID, idParam := httphelper.GetInt64Param(ctx, "report_id")
		if !idParam {
			return
		}

		var req domain.RequestReportStatusUpdate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		report, err := h.reports.SetReportStatus(ctx, reportID, httphelper.CurrentUserProfile(ctx), req.Status)
		if err != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, err))

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
		reportID, idFound := httphelper.GetInt64Param(ctx, "report_id")
		if !idFound {
			return
		}

		report, errGetReport := h.reports.GetReport(ctx, httphelper.CurrentUserProfile(ctx), reportID)
		if errGetReport != nil {
			if errors.Is(errGetReport, domain.ErrNoResult) {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusNotFound, domain.ErrNoResult))

				return
			}

			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errGetReport))

			return
		}

		if !httphelper.HasPrivilege(httphelper.CurrentUserProfile(ctx), steamid.Collection{report.SourceID, report.TargetID}, domain.PModerator) {
			return
		}

		reportMessages, errGetReportMessages := h.reports.GetReportMessages(ctx, reportID)
		if errGetReportMessages != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errGetReport))

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
		reportID, idFound := httphelper.GetInt64Param(ctx, "report_id")
		if !idFound {
			return
		}

		if reportID == 0 {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusBadRequest, domain.ErrBadRequest))

			return
		}

		var req domain.RequestMessageBodyMD
		if !httphelper.Bind(ctx, &req) {
			return
		}

		curUser := httphelper.CurrentUserProfile(ctx)

		msg, errSave := h.reports.CreateReportMessage(ctx, reportID, curUser, req)
		if errSave != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errSave))

			return
		}

		ctx.JSON(http.StatusCreated, msg)

		slog.Info("New report message created",
			slog.Int64("report_id", reportID), slog.String("steam_id", curUser.SteamID.String()))
	}
}

func (h reportHandler) onAPIEditReportMessage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportMessageID, idFound := httphelper.GetInt64Param(ctx, "report_message_id")
		if idFound {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusBadRequest, domain.ErrBadRequest))

			return
		}

		var req domain.RequestMessageBodyMD
		if !httphelper.Bind(ctx, &req) {
			return
		}

		msg, errMsg := h.reports.EditReportMessage(ctx, reportMessageID, httphelper.CurrentUserProfile(ctx), req)
		if errMsg != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errMsg))

			return
		}

		ctx.JSON(http.StatusOK, msg)
		slog.Info("Report message edited", slog.Int64("report_message_id", reportMessageID))
	}
}

func (h reportHandler) onAPIDeleteReportMessage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportMessageID, idFound := httphelper.GetInt64Param(ctx, "report_message_id")
		if !idFound {
			return
		}
		if reportMessageID == 0 {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusBadRequest, domain.ErrBadRequest))

			return
		}

		if err := h.reports.DropReportMessage(ctx, httphelper.CurrentUserProfile(ctx), reportMessageID); err != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, err))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
		slog.Info("Deleted report message", slog.Int64("report_message_id", reportMessageID))
	}
}
