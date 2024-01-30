package report

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/http_helper"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

type ReportHandler struct {
	log *zap.Logger
	ru  domain.ReportUsecase
	cu  domain.ConfigUsecase
	du  domain.DiscordUsecase
	pu  domain.PersonUsecase
}

func NewReportHandler(log *zap.Logger, engine *gin.Engine, ru domain.ReportUsecase, cu domain.ConfigUsecase, du domain.DiscordUsecase, pu domain.PersonUsecase, ath domain.AuthUsecase) {
	handler := ReportHandler{
		log: log.Named("report"),
		ru:  ru,
		cu:  cu,
		du:  du,
		pu:  pu,
	}

	// auth
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(ath.AuthMiddleware(domain.PUser))
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
		mod := modGrp.Use(ath.AuthMiddleware(domain.PModerator))
		mod.POST("/api/report/:report_id/state", handler.onAPIPostBanState())
	}
}

func (h ReportHandler) onAPIPostBanState() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportID, errID := http_helper.GetInt64Param(ctx, "report_id")
		if errID != nil || reportID <= 0 {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var report domain.Report
		if errReport := h.ru.GetReport(ctx, reportID, &report); errReport != nil {
			if errors.Is(errReport, domain.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		h.du.SendPayload(domain.ChannelModLog, discord.EditBanAppealStatusMessage())
	}
}

func (h ReportHandler) onAPIPostReportCreate() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		currentUser := http_helper.CurrentUserProfile(ctx)

		var req domain.CreateReportReq
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		if req.Description == "" || len(req.Description) < 10 {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, fmt.Errorf("%w: description", domain.ErrParamInvalid))

			return
		}

		// ServerStore initiated requests will have a sourceID set by the server
		// Web based reports the source should not be set, the reporter will be taken from the
		// current session information instead
		if req.SourceID == "" {
			req.SourceID = domain.StringSID(currentUser.SteamID.String())
		}

		sourceID, errSourceID := req.SourceID.SID64(ctx)
		if errSourceID != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrSourceID)
			log.Error("Invalid steam_id", zap.Error(errSourceID))

			return
		}

		targetID, errTargetID := req.TargetID.SID64(ctx)
		if errTargetID != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrTargetID)
			log.Error("Invalid target_id", zap.Error(errTargetID))

			return
		}

		if sourceID == targetID {
			http_helper.ResponseErr(ctx, http.StatusConflict, domain.ErrSelfReport)

			return
		}

		var personSource domain.Person
		if errCreatePerson := h.pu.GetPersonBySteamID(ctx, sourceID, &personSource); errCreatePerson != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Could not load player profile", zap.Error(errCreatePerson))

			return
		}

		var personTarget domain.Person
		if errCreatePerson := h.pu.GetOrCreatePersonBySteamID(ctx, targetID, &personTarget); errCreatePerson != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Could not load player profile", zap.Error(errCreatePerson))

			return
		}

		if personTarget.Expired() {
			if err := thirdparty.UpdatePlayerSummary(ctx, &personTarget); err != nil {
				log.Error("Failed to update target player", zap.Error(err))
			} else {
				if errSave := h.pu.SavePerson(ctx, &personTarget); errSave != nil {
					log.Error("Failed to save target player update", zap.Error(err))
				}
			}
		}

		// Ensure the user doesn't already have an open report against the user
		var existing domain.Report
		if errReports := h.ru.GetReportBySteamID(ctx, personSource.SteamID, targetID, &existing); errReports != nil {
			if !errors.Is(errReports, domain.ErrNoResult) {
				log.Error("Failed to query reports by steam id", zap.Error(errReports))
				http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

				return
			}
		}

		if existing.ReportID > 0 {
			http_helper.ResponseErr(ctx, http.StatusConflict, domain.ErrReportExists)

			return
		}

		// TODO encapsulate all operations in single tx
		report := domain.NewReport()
		report.SourceID = sourceID
		report.ReportStatus = domain.Opened
		report.Description = req.Description
		report.TargetID = targetID
		report.Reason = req.Reason
		report.ReasonText = req.ReasonText
		parts := strings.Split(req.DemoName, "/")
		report.DemoName = parts[len(parts)-1]
		report.DemoTick = req.DemoTick
		report.PersonMessageID = req.PersonMessageID

		if errReportSave := h.ru.SaveReport(ctx, &report); errReportSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to save report", zap.Error(errReportSave))

			return
		}

		ctx.JSON(http.StatusCreated, report)

		log.Info("New report created successfully", zap.Int64("report_id", report.ReportID))

		conf := h.cu.Config()

		if !conf.Discord.Enabled {
			return
		}

		demoURL := ""

		if report.DemoName != "" {
			demoURL = conf.ExtURLRaw("/demos/name/%s", report.DemoName)
		}

		msg := discord.NewInGameReportResponse(report, conf.ExtURL(report), currentUser, conf.ExtURL(currentUser), demoURL)

		h.du.SendPayload(domain.ChannelModLog, msg)
	}
}

func (h ReportHandler) onAPIGetReport() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportID, errParam := http_helper.GetInt64Param(ctx, "report_id")
		if errParam != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var report reportWithAuthor
		if errReport := h.ru.GetReport(ctx, reportID, &report.Report); errReport != nil {
			if errors.Is(errReport, domain.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to load report", zap.Error(errReport))

			return
		}

		if !http_helper.CheckPrivilege(ctx, http_helper.CurrentUserProfile(ctx), steamid.Collection{report.Report.SourceID}, domain.PModerator) {
			http_helper.ResponseErr(ctx, http.StatusUnauthorized, domain.ErrPermissionDenied)

			return
		}

		if errAuthor := h.pu.GetPersonBySteamID(ctx, report.Report.SourceID, &report.Author); errAuthor != nil {
			if errors.Is(errAuthor, domain.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)
			log.Error("Failed to load report author", zap.Error(errAuthor))

			return
		}

		if errSubject := h.pu.GetPersonBySteamID(ctx, report.Report.TargetID, &report.Subject); errSubject != nil {
			if errors.Is(errSubject, domain.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)
			log.Error("Failed to load report subject", zap.Error(errSubject))

			return
		}

		ctx.JSON(http.StatusOK, report)
	}
}

type reportWithAuthor struct {
	Author  domain.Person `json:"author"`
	Subject domain.Person `json:"subject"`
	domain.Report
}

func (h ReportHandler) onAPIGetReports() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		user := http_helper.CurrentUserProfile(ctx)

		var req domain.ReportQueryFilter
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		if req.Limit <= 0 && req.Limit > 100 {
			req.Limit = 25
		}

		// Make sure the person requesting is either a moderator, or a user
		// only able to request their own reports
		var sourceID steamid.SID64

		if user.PermissionLevel < domain.PModerator {
			sourceID = user.SteamID
		} else if req.SourceID != "" {
			sid, errSourceID := req.SourceID.SID64(ctx)
			if errSourceID != nil {
				http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

				return
			}

			sourceID = sid
		}

		if sourceID.Valid() {
			req.SourceID = domain.StringSID(sourceID.String())
		}

		var userReports []reportWithAuthor

		reports, count, errReports := h.ru.GetReports(ctx, req)
		if errReports != nil {
			if errors.Is(errReports, domain.ErrNoResult) {
				ctx.JSON(http.StatusNoContent, nil)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		var authorIds steamid.Collection
		for _, report := range reports {
			authorIds = append(authorIds, report.SourceID)
		}

		authors, errAuthors := h.pu.GetPeopleBySteamID(ctx, fp.Uniq(authorIds))
		if errAuthors != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		authorMap := authors.AsMap()

		var subjectIds steamid.Collection
		for _, report := range reports {
			subjectIds = append(subjectIds, report.TargetID)
		}

		subjects, errSubjects := h.pu.GetPeopleBySteamID(ctx, fp.Uniq(subjectIds))
		if errSubjects != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		subjectMap := subjects.AsMap()

		for _, report := range reports {
			userReports = append(userReports, reportWithAuthor{
				Author:  authorMap[report.SourceID],
				Report:  report,
				Subject: subjectMap[report.TargetID],
			})
		}

		if userReports == nil {
			userReports = []reportWithAuthor{}
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(count, userReports))
	}
}

func (h ReportHandler) onAPISetReportStatus() gin.HandlerFunc {
	type stateUpdateReq struct {
		Status domain.ReportStatus `json:"status"`
	}

	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportID, errParam := http_helper.GetInt64Param(ctx, "report_id")
		if errParam != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var req stateUpdateReq
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		var report domain.Report
		if errGet := h.ru.GetReport(ctx, reportID, &report); errGet != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to get report to set state", zap.Error(errGet))

			return
		}

		if report.ReportStatus == req.Status {
			ctx.JSON(http.StatusConflict, domain.ErrDuplicate)

			return
		}

		original := report.ReportStatus

		report.ReportStatus = req.Status
		if errSave := h.ru.SaveReport(ctx, &report); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to save report state", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusAccepted, nil)
		log.Info("Report status changed",
			zap.Int64("report_id", report.ReportID),
			zap.String("from_status", original.String()),
			zap.String("to_status", report.ReportStatus.String()))
		// discord.SendDiscord(model.NotificationPayload{
		//	Sids:     steamid.Collection{report.SourceID},
		//	Severity: db.SeverityInfo,
		//	Message:  "Report status updated",
		//	Link:     report.ToURL(),
		// })
	} //nolint:wsl
}

func (h ReportHandler) onAPIGetReportMessages() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportID, errParam := http_helper.GetInt64Param(ctx, "report_id")
		if errParam != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var report domain.Report
		if errGetReport := h.ru.GetReport(ctx, reportID, &report); errGetReport != nil {
			http_helper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

			return
		}

		if !http_helper.CheckPrivilege(ctx, http_helper.CurrentUserProfile(ctx), steamid.Collection{report.SourceID, report.TargetID}, domain.PModerator) {
			return
		}

		reportMessages, errGetReportMessages := h.ru.GetReportMessages(ctx, reportID)
		if errGetReportMessages != nil {
			http_helper.ResponseErr(ctx, http.StatusNotFound, domain.ErrPlayerNotFound)

			return
		}

		if reportMessages == nil {
			reportMessages = []domain.ReportMessage{}
		}

		ctx.JSON(http.StatusOK, reportMessages)
	}
}

func (h ReportHandler) onAPIPostReportMessage() gin.HandlerFunc {
	type newMessage struct {
		Message string `json:"message"`
	}

	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportID, errID := http_helper.GetInt64Param(ctx, "report_id")
		if errID != nil || reportID == 0 {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var req newMessage
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		if req.Message == "" {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		var report domain.Report
		if errReport := h.ru.GetReport(ctx, reportID, &report); errReport != nil {
			if errors.Is(errReport, domain.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to load report", zap.Error(errReport))

			return
		}

		person := http_helper.CurrentUserProfile(ctx)
		msg := domain.NewReportMessage(reportID, person.SteamID, req.Message)

		if errSave := h.ru.SaveReportMessage(ctx, &msg); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to save report message", zap.Error(errSave))

			return
		}

		report.UpdatedOn = time.Now()

		if errSave := h.ru.SaveReport(ctx, &report); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to update report activity", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, msg)

		conf := h.cu.Config()

		h.du.SendPayload(domain.ChannelModLog,
			discord.NewReportMessageResponse(msg.MessageMD, conf.ExtURL(report), person, conf.ExtURL(person)))
	}
}

func (h ReportHandler) onAPIEditReportMessage() gin.HandlerFunc {
	type editMessage struct {
		BodyMD string `json:"body_md"`
	}

	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportMessageID, errID := http_helper.GetInt64Param(ctx, "report_message_id")
		if errID != nil || reportMessageID == 0 {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var existing domain.ReportMessage
		if errExist := h.ru.GetReportMessageByID(ctx, reportMessageID, &existing); errExist != nil {
			if errors.Is(errExist, domain.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusNotFound, domain.ErrPlayerNotFound)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		curUser := http_helper.CurrentUserProfile(ctx)
		if !http_helper.CheckPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, domain.PModerator) {
			return
		}

		var req editMessage
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		if req.BodyMD == "" {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		if req.BodyMD == existing.MessageMD {
			http_helper.ResponseErr(ctx, http.StatusConflict, domain.ErrDuplicate)

			return
		}

		existing.MessageMD = req.BodyMD
		if errSave := h.ru.SaveReportMessage(ctx, &existing); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to save report message", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, req)

		conf := h.cu.Config()

		msg := discord.EditReportMessageResponse(req.BodyMD, existing.MessageMD,
			conf.ExtURLRaw("/report/%d", existing.ReportID), curUser, conf.ExtURL(curUser))

		h.du.SendPayload(domain.ChannelModLog, msg)
	}
}

func (h ReportHandler) onAPIDeleteReportMessage() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportMessageID, errID := http_helper.GetInt64Param(ctx, "report_message_id")
		if errID != nil || reportMessageID == 0 {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var existing domain.ReportMessage
		if errExist := h.ru.GetReportMessageByID(ctx, reportMessageID, &existing); errExist != nil {
			if errors.Is(errExist, domain.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		curUser := http_helper.CurrentUserProfile(ctx)
		if !http_helper.CheckPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, domain.PModerator) {
			return
		}

		existing.Deleted = true
		if errSave := h.ru.SaveReportMessage(ctx, &existing); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to save report message", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusNoContent, nil)

		conf := h.cu.Config()

		h.du.SendPayload(domain.ChannelModLog, discord.DeleteReportMessage(existing, curUser, conf.ExtURL(curUser)))
	}
}
