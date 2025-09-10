package report

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/database"
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
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusConflict, domain.ErrReportExists,
					"An open report already exists for this player, duplicates are not allowed."))

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errReportSave, domain.ErrInternal)))

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
			if errors.Is(errReport, database.ErrNoResult) {
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusNotFound, domain.ErrNotFound,
					"Could not find a report with the id: %d", reportID))

				return
			}
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(errReport, domain.ErrInternal),
				"Could not load report with the id: %d", reportID))

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
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errReports, domain.ErrInternal)))

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
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errReports, domain.ErrInternal)))

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

		_, err := h.reports.SetReportStatus(ctx, reportID, httphelper.CurrentUserProfile(ctx), req.Status)
		if err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, domain.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
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
			if errors.Is(errGetReport, database.ErrNoResult) {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, database.ErrNoResult))

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errGetReport, domain.ErrInternal)))

			return
		}

		if !httphelper.HasPrivilege(httphelper.CurrentUserProfile(ctx), steamid.Collection{report.SourceID, report.TargetID}, domain.PModerator) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusForbidden, domain.ErrPermissionDenied))

			return
		}

		reportMessages, errGetReportMessages := h.reports.GetReportMessages(ctx, reportID)
		if errGetReportMessages != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errGetReportMessages, domain.ErrInternal)))

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
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusBadRequest, domain.ErrBadRequest))

			return
		}

		var req domain.RequestMessageBodyMD
		if !httphelper.Bind(ctx, &req) {
			return
		}

		curUser := httphelper.CurrentUserProfile(ctx)

		msg, errSave := h.reports.CreateReportMessage(ctx, reportID, curUser, req)
		if errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errSave, domain.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusCreated, msg)
	}
}

func (h reportHandler) onAPIEditReportMessage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportMessageID, idFound := httphelper.GetInt64Param(ctx, "report_message_id")
		if !idFound {
			return
		}

		var req domain.RequestMessageBodyMD
		if !httphelper.Bind(ctx, &req) {
			return
		}

		msg, errMsg := h.reports.EditReportMessage(ctx, reportMessageID, httphelper.CurrentUserProfile(ctx), req)
		if errMsg != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errMsg, domain.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, msg)
	}
}

func (h reportHandler) onAPIDeleteReportMessage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportMessageID, idFound := httphelper.GetInt64Param(ctx, "report_message_id")
		if !idFound {
			return
		}
		if reportMessageID == 0 {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusBadRequest, domain.ErrBadRequest))

			return
		}

		if err := h.reports.DropReportMessage(ctx, httphelper.CurrentUserProfile(ctx), reportMessageID); err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, err))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}
