package service

import (
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/http_helper"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
	"net/http"
	"runtime"
	"time"
)

func onAPIGetReport() gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportID, errParam := http_helper.GetInt64Param(ctx, "report_id")
		if errParam != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter

			return
		}

		var report reportWithAuthor
		if errReport := env.Store().GetReport(ctx, reportID, &report.Report); errReport != nil {
			if errors.Is(errs.DBErr(errReport), errs.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to load report", zap.Error(errReport))

			return
		}

		if !checkPrivilege(ctx, http_helper.CurrentUserProfile(ctx), steamid.Collection{report.Report.SourceID}, domain.PModerator) {
			http_helper.ResponseErr(ctx, http.StatusUnauthorized, errPermissionDenied)

			return
		}

		if errAuthor := env.Store().GetPersonBySteamID(ctx, report.Report.SourceID, &report.Author); errAuthor != nil {
			if errors.Is(errs.DBErr(errAuthor), errs.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)
			log.Error("Failed to load report author", zap.Error(errAuthor))

			return
		}

		if errSubject := env.Store().GetPersonBySteamID(ctx, report.Report.TargetID, &report.Subject); errSubject != nil {
			if errors.Is(errs.DBErr(errSubject), errs.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

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

func onAPIGetReports() gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

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

		reports, count, errReports := env.Store().GetReports(ctx, req)
		if errReports != nil {
			if errors.Is(errs.DBErr(errReports), errs.ErrNoResult) {
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

		authors, errAuthors := env.Store().GetPeopleBySteamID(ctx, fp.Uniq(authorIds))
		if errAuthors != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		authorMap := authors.AsMap()

		var subjectIds steamid.Collection
		for _, report := range reports {
			subjectIds = append(subjectIds, report.TargetID)
		}

		subjects, errSubjects := env.Store().GetPeopleBySteamID(ctx, fp.Uniq(subjectIds))
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

func onAPISetReportStatus() gin.HandlerFunc {
	type stateUpdateReq struct {
		Status domain.ReportStatus `json:"status"`
	}

	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportID, errParam := http_helper.GetInt64Param(ctx, "report_id")
		if errParam != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter

			return
		}

		var req stateUpdateReq
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		var report domain.Report
		if errGet := env.Store().GetReport(ctx, reportID, &report); errGet != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to get report to set state", zap.Error(errGet))

			return
		}

		if report.ReportStatus == req.Status {
			ctx.JSON(http.StatusConflict, errs.ErrDuplicate)

			return
		}

		original := report.ReportStatus

		report.ReportStatus = req.Status
		if errSave := env.Store().SaveReport(ctx, &report); errSave != nil {
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

func onAPIGetReportMessages() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportID, errParam := http_helper.GetInt64Param(ctx, "report_id")
		if errParam != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter

			return
		}

		var report domain.Report
		if errGetReport := env.Store().GetReport(ctx, reportID, &report); errGetReport != nil {
			http_helper.ResponseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

			return
		}

		if !checkPrivilege(ctx, http_helper.CurrentUserProfile(ctx), steamid.Collection{report.SourceID, report.TargetID}, domain.PModerator) {
			return
		}

		reportMessages, errGetReportMessages := env.Store().GetReportMessages(ctx, reportID)
		if errGetReportMessages != nil {
			http_helper.ResponseErr(ctx, http.StatusNotFound, errs.ErrPlayerNotFound)

			return
		}

		if reportMessages == nil {
			reportMessages = []domain.ReportMessage{}
		}

		ctx.JSON(http.StatusOK, reportMessages)
	}
}

func onAPIPostReportMessage() gin.HandlerFunc {
	type newMessage struct {
		Message string `json:"message"`
	}

	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportID, errID := http_helper.GetInt64Param(ctx, "report_id")
		if errID != nil || reportID == 0 {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter

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
		if errReport := env.Store().GetReport(ctx, reportID, &report); errReport != nil {
			if errors.Is(errs.DBErr(errReport), errs.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to load report", zap.Error(errReport))

			return
		}

		person := http_helper.CurrentUserProfile(ctx)
		msg := domain.NewReportMessage(reportID, person.SteamID, req.Message)

		if errSave := env.Store().SaveReportMessage(ctx, &msg); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to save report message", zap.Error(errSave))

			return
		}

		report.UpdatedOn = time.Now()

		if errSave := env.Store().SaveReport(ctx, &report); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to update report activity", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, msg)

		conf := env.Config()

		env.SendPayload(conf.Discord.LogChannelID,
			discord.NewReportMessageResponse(msg.MessageMD, conf.ExtURL(report), person, conf.ExtURL(person)))
	}
}

func onAPIEditReportMessage() gin.HandlerFunc {
	type editMessage struct {
		BodyMD string `json:"body_md"`
	}

	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportMessageID, errID := http_helper.GetInt64Param(ctx, "report_message_id")
		if errID != nil || reportMessageID == 0 {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter

			return
		}

		var existing domain.ReportMessage
		if errExist := env.Store().GetReportMessageByID(ctx, reportMessageID, &existing); errExist != nil {
			if errors.Is(errExist, errs.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusNotFound, errs.ErrPlayerNotFound)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		curUser := http_helper.CurrentUserProfile(ctx)
		if !checkPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, domain.PModerator) {
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
			http_helper.ResponseErr(ctx, http.StatusConflict, errs.ErrDuplicate)

			return
		}

		existing.MessageMD = req.BodyMD
		if errSave := env.Store().SaveReportMessage(ctx, &existing); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to save report message", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, req)

		conf := env.Config()
		msg := discord.EditReportMessageResponse(req.BodyMD, existing.MessageMD,
			conf.ExtURLRaw("/report/%d", existing.ReportID), curUser, conf.ExtURL(curUser))
		env.SendPayload(env.Config().Discord.LogChannelID, msg)
	}
}

func onAPIDeleteReportMessage() gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportMessageID, errID := http_helper.GetInt64Param(ctx, "report_message_id")
		if errID != nil || reportMessageID == 0 {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter

			return
		}

		var existing domain.ReportMessage
		if errExist := env.Store().GetReportMessageByID(ctx, reportMessageID, &existing); errExist != nil {
			if errors.Is(errExist, errs.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		curUser := http_helper.CurrentUserProfile(ctx)
		if !checkPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, domain.PModerator) {
			return
		}

		existing.Deleted = true
		if errSave := env.Store().SaveReportMessage(ctx, &existing); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to save report message", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusNoContent, nil)

		conf := env.Config()

		env.SendPayload(conf.Discord.LogChannelID, discord.DeleteReportMessage(existing, curUser, conf.ExtURL(curUser)))
	}
}
