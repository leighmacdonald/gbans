package ban

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/auth/session"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type reportHandler struct {
	Reports
}

func NewReportHandler(engine *gin.Engine, authenticator httphelper.Authenticator, reports Reports) {
	handler := reportHandler{reports}

	// auth
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(authenticator.Middleware(permission.User))

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
		mod := modGrp.Use(authenticator.Middleware(permission.Moderator))
		mod.POST("/api/reports", handler.onAPIGetAllReports())
	}
}

func (h reportHandler) onAPIPostReportCreate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser, _ := session.CurrentUserProfile(ctx)
		req, ok := httphelper.BindJSON[RequestReportCreate](ctx)
		if !ok {
			return
		}

		report, errReportSave := h.Save(ctx, currentUser, req)
		if errReportSave != nil {
			if errors.Is(errReportSave, ErrReportExists) {
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusConflict, ErrReportExists,
					"An open report already exists for this player, duplicates are not allowed."))

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errReportSave, httphelper.ErrInternal)))

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

		user, _ := session.CurrentUserProfile(ctx)
		report, errReport := h.Report(ctx, user, reportID)
		if errReport != nil {
			if errors.Is(errReport, database.ErrNoResult) {
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusNotFound, httphelper.ErrNotFound,
					"Could not find a report with the id: %d", reportID))

				return
			}
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(errReport, httphelper.ErrInternal),
				"Could not load report with the id: %d", reportID))

			return
		}

		ctx.JSON(http.StatusOK, report)
	}
}

func (h reportHandler) onAPIGetUserReports() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user, _ := session.CurrentUserProfile(ctx)
		reports, errReports := h.BySteamID(ctx, user.GetSteamID())
		if errReports != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errReports, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, reports)
	}
}

func (h reportHandler) onAPIGetAllReports() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// TODO needed?
		// req, ok := httphelper.BindGJSON[ReportQueryFilter](ctx)
		// if !ok {
		// 	return
		// }

		reports, errReports := h.Reports.Reports(ctx)
		if errReports != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errReports, httphelper.ErrInternal)))

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

		req, ok := httphelper.BindJSON[RequestReportStatusUpdate](ctx)
		if !ok {
			return
		}

		user, _ := session.CurrentUserProfile(ctx)
		_, err := h.SetReportStatus(ctx, reportID, user, req.Status)
		if err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal)))

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

		user, _ := session.CurrentUserProfile(ctx)
		report, errGetReport := h.Report(ctx, user, reportID)
		if errGetReport != nil {
			if errors.Is(errGetReport, database.ErrNoResult) {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, database.ErrNoResult))

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errGetReport, httphelper.ErrInternal)))

			return
		}

		if !httphelper.HasPrivilege(user, steamid.Collection{report.SourceID, report.TargetID}, permission.Moderator) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusForbidden, httphelper.ErrBadRequest))

			return
		}

		reportMessages, errGetReportMessages := h.Messages(ctx, reportID)
		if errGetReportMessages != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errGetReportMessages, httphelper.ErrInternal)))

			return
		}

		if reportMessages == nil {
			reportMessages = []ReportMessage{}
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
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusBadRequest, httphelper.ErrBadRequest))

			return
		}

		req, ok := httphelper.BindJSON[RequestMessageBodyMD](ctx)
		if !ok {
			return
		}

		curUser, _ := session.CurrentUserProfile(ctx)
		msg, errSave := h.CreateMessage(ctx, reportID, curUser, req)
		if errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errSave, httphelper.ErrInternal)))

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

		req, ok := httphelper.BindJSON[RequestMessageBodyMD](ctx)
		if !ok {
			return
		}

		user, _ := session.CurrentUserProfile(ctx)
		msg, errMsg := h.EditMessage(ctx, reportMessageID, user, req)
		if errMsg != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errMsg, httphelper.ErrInternal)))

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
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusBadRequest, httphelper.ErrBadRequest))

			return
		}

		user, _ := session.CurrentUserProfile(ctx)
		if err := h.DropMessage(ctx, user, reportMessageID); err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, err))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}
