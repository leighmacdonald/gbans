package ban

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/queue"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/riverqueue/river"
)

type ReportUsecase struct {
	repository ReportRepository
	config     *config.ConfigUsecase
	persons    person.PersonUsecase
	demos      servers.DemoUsecase
	tfAPI      *thirdparty.TFAPI
}

func NewReportUsecase(repository ReportRepository,
	config *config.ConfigUsecase, persons person.PersonUsecase, demos servers.DemoUsecase, tfAPI *thirdparty.TFAPI,
) ReportUsecase {
	return ReportUsecase{
		repository: repository,
		config:     config,
		persons:    persons,
		demos:      demos,
		tfAPI:      tfAPI,
	}
}

func (r ReportUsecase) GenerateMetaStats(ctx context.Context) error {
	reports, errReports := r.GetReports(ctx)
	if errReports != nil {
		slog.Error("failed to fetch reports for report metadata", log.ErrAttr(errReports))

		return errReports
	}

	var (
		now  = time.Now()
		meta ReportMeta
	)

	for _, report := range reports {
		if report.ReportStatus == ClosedWithAction || report.ReportStatus == ClosedWithoutAction {
			meta.TotalClosed++

			continue
		}

		meta.TotalOpen++

		if report.ReportStatus == NeedMoreInfo {
			meta.NeedInfo++
		} else {
			meta.Open++
		}

		switch {
		case now.Sub(report.CreatedOn) > time.Hour*24*7:
			meta.OpenWeek++
		case now.Sub(report.CreatedOn) > time.Hour*24*3:
			meta.Open3Days++
		case now.Sub(report.CreatedOn) > time.Hour*24:
			meta.Open1Day++
		default:
			meta.OpenNew++
		}
	}

	// r.notifications.Enqueue(ctx, notification.NewDiscordNotification(
	// 	r.config.Config().Discord.LogChannelID,
	// 	discord.ReportStatsMessage(meta, r.config.ExtURLRaw("/admin/reports"))))

	return nil
}

func (r ReportUsecase) addAuthorsToReports(ctx context.Context, reports []Report) ([]ReportWithAuthor, error) {
	var peopleIDs steamid.Collection
	for _, report := range reports {
		peopleIDs = append(peopleIDs, report.SourceID, report.TargetID)
	}

	people, errAuthors := r.persons.GetPeopleBySteamID(ctx, nil, fp.Uniq(peopleIDs))
	if errAuthors != nil {
		return nil, errAuthors
	}

	peopleMap := people.AsMap()

	userReports := make([]ReportWithAuthor, len(reports))

	for i, report := range reports {
		userReports[i] = ReportWithAuthor{
			Author:  peopleMap[report.SourceID],
			Report:  report,
			Subject: peopleMap[report.TargetID],
		}
	}

	return userReports, nil
}

func (r ReportUsecase) SetReportStatus(ctx context.Context, reportID int64, user domain.PersonInfo, status ReportStatus) (ReportWithAuthor, error) {
	report, errGet := r.GetReport(ctx, user, reportID)
	if errGet != nil {
		return report, errGet
	}

	if report.ReportStatus == status {
		return report, database.ErrDuplicate // TODO proper specific error
	}

	// fromStatus := report.ReportStatus

	report.ReportStatus = status

	if errSave := r.repository.SaveReport(ctx, &report.Report); errSave != nil {
		return report, errSave
	}

	// r.notifications.Enqueue(ctx, notification.NewDiscordNotification(
	// 	r.config.Config().Discord.LogChannelID,
	// 	discord.ReportStatusChangeMessage(report, fromStatus, r.config.ExtURL(report))))

	// r.notifications.Enqueue(ctx, notification.NewSiteGroupNotificationWithAuthor(
	// 	[]permission.Privilege{permission.PModerator, permission.PAdmin},
	// 	notification.SeverityInfo,
	// 	fmt.Sprintf("A report status has changed: %s -> %s", fromStatus, status),
	// 	report.Path(),
	// 	user,
	// ))

	// r.notifications.Enqueue(ctx, notification.NewSiteUserNotification(
	// 	[]steamid.SteamID{report.Author.SteamID},
	// 	notification.SeverityInfo,
	// 	fmt.Sprintf("Your report status has changed: %s -> %s", fromStatus, status),
	// 	report.Path(),
	// ))

	slog.Info("Report status changed",
		slog.Int64("report_id", report.ReportID),
		slog.String("to_status", report.ReportStatus.String()))

	return report, nil
}

func (r ReportUsecase) GetReportsBySteamID(ctx context.Context, steamID steamid.SteamID) ([]ReportWithAuthor, error) {
	if !steamID.Valid() {
		return nil, domain.ErrInvalidSID
	}

	reports, errReports := r.repository.GetReports(ctx, steamID)
	if errReports != nil {
		if errors.Is(errReports, database.ErrNoResult) {
			return []ReportWithAuthor{}, nil
		}

		return nil, errReports
	}

	return r.addAuthorsToReports(ctx, reports)
}

func (r ReportUsecase) GetReports(ctx context.Context) ([]ReportWithAuthor, error) {
	reports, errReports := r.repository.GetReports(ctx, steamid.SteamID{})
	if errReports != nil {
		if errors.Is(errReports, database.ErrNoResult) {
			return []ReportWithAuthor{}, nil
		}

		return nil, errReports
	}

	return r.addAuthorsToReports(ctx, reports)
}

func (r ReportUsecase) GetReport(ctx context.Context, curUser domain.PersonInfo, reportID int64) (ReportWithAuthor, error) {
	report, err := r.repository.GetReport(ctx, reportID)
	if err != nil {
		return ReportWithAuthor{}, err
	}

	author, errAuthor := r.persons.GetPersonBySteamID(ctx, nil, report.SourceID)
	if errAuthor != nil {
		return ReportWithAuthor{}, errAuthor
	}

	if !httphelper.HasPrivilege(curUser, steamid.Collection{author.SteamID}, permission.PModerator) {
		return ReportWithAuthor{}, permission.ErrPermissionDenied
	}

	target, errTarget := r.persons.GetPersonBySteamID(ctx, nil, report.TargetID)
	if errTarget != nil {
		return ReportWithAuthor{}, errTarget
	}

	var demo servers.DemoFile
	if report.DemoID > 0 {
		if errDemo := r.demos.GetDemoByID(ctx, report.DemoID, &demo); errDemo != nil {
			slog.Error("Failed to load report demo", slog.Int64("report_id", report.ReportID))
		}
	}

	return ReportWithAuthor{
		Author:  author,
		Subject: target,
		Report:  report,
		// TODO FIX Demo:    demo,
	}, nil
}

func (r ReportUsecase) GetReportBySteamID(ctx context.Context, authorID steamid.SteamID, steamID steamid.SteamID) (Report, error) {
	return r.repository.GetReportBySteamID(ctx, authorID, steamID)
}

func (r ReportUsecase) GetReportMessages(ctx context.Context, reportID int64) ([]ReportMessage, error) {
	return r.repository.GetReportMessages(ctx, reportID)
}

func (r ReportUsecase) GetReportMessageByID(ctx context.Context, reportMessageID int64) (ReportMessage, error) {
	return r.repository.GetReportMessageByID(ctx, reportMessageID)
}

func (r ReportUsecase) DropReportMessage(ctx context.Context, curUser domain.PersonInfo, reportMessageID int64) error {
	existing, errExist := r.repository.GetReportMessageByID(ctx, reportMessageID)
	if errExist != nil {
		return errExist
	}

	if !httphelper.HasPrivilege(curUser, steamid.Collection{existing.AuthorID}, permission.PModerator) {
		return permission.ErrPermissionDenied
	}

	if err := r.repository.DropReportMessage(ctx, &existing); err != nil {
		return err
	}

	// r.notifications.Enqueue(ctx, notification.NewDiscordNotification(
	// 	discord.ChannelModAppealLog,
	// 	discord.DeleteReportMessage(existing, curUser, r.config.ExtURL(curUser))))

	slog.Info("Deleted report message", slog.Int64("report_message_id", reportMessageID))

	return nil
}

func (r ReportUsecase) DropReport(ctx context.Context, report *Report) error {
	return r.repository.DropReport(ctx, report)
}

func (r ReportUsecase) SaveReport(ctx context.Context, currentUser domain.PersonInfo, req RequestReportCreate) (ReportWithAuthor, error) {
	if req.Description == "" || len(req.Description) < 10 {
		return ReportWithAuthor{}, fmt.Errorf("%w: description", domain.ErrParamInvalid)
	}

	// ServerStore initiated requests will have a sourceID set by the server
	// Web based reports the source should not be set, the reporter will be taken from the
	// current session information instead
	if !req.SourceID.Valid() {
		req.SourceID = currentUser.GetSteamID()
	}

	if !req.SourceID.Valid() {
		return ReportWithAuthor{}, fmt.Errorf("%w: source_id", domain.ErrParamInvalid)
	}

	if !req.TargetID.Valid() {
		return ReportWithAuthor{}, fmt.Errorf("%w: target_id", domain.ErrParamInvalid)
	}

	if req.SourceID.Int64() == req.TargetID.Int64() {
		return ReportWithAuthor{}, fmt.Errorf("%w: cannot report self", domain.ErrParamInvalid)
	}

	personSource, errSource := r.persons.GetPersonBySteamID(ctx, nil, req.SourceID)
	if errSource != nil {
		return ReportWithAuthor{}, errSource
	}

	personTarget, errTarget := r.persons.GetOrCreatePersonBySteamID(ctx, nil, req.TargetID)
	if errTarget != nil {
		return ReportWithAuthor{}, errTarget
	}

	if personTarget.Expired() {
		if err := person.UpdatePlayerSummary(ctx, &personTarget, r.tfAPI); err != nil {
			slog.Error("Failed to update target player", log.ErrAttr(err))
		} else {
			if errSave := r.persons.SavePerson(ctx, nil, &personTarget); errSave != nil {
				slog.Error("Failed to save target player update", log.ErrAttr(err))
			}
		}
	}

	// Ensure the user doesn't already have an open report against the user
	existing, errReports := r.GetReportBySteamID(ctx, personSource.SteamID, req.TargetID)
	if errReports != nil {
		if !errors.Is(errReports, database.ErrNoResult) {
			return ReportWithAuthor{}, errReports
		}
	}

	if existing.ReportID > 0 {
		return ReportWithAuthor{}, domain.ErrReportExists
	}

	var demo servers.DemoFile

	if req.DemoID > 0 {
		if errDemo := r.demos.GetDemoByID(ctx, req.DemoID, &demo); errDemo != nil {
			return ReportWithAuthor{}, errDemo
		}
	}

	// TODO encapsulate all operations in single tx
	report := NewReport()
	report.SourceID = req.SourceID
	report.ReportStatus = Opened
	report.Description = req.Description
	report.TargetID = req.TargetID
	report.Reason = req.Reason
	report.ReasonText = req.ReasonText
	report.DemoID = req.DemoID
	report.DemoTick = req.DemoTick
	report.PersonMessageID = req.PersonMessageID

	if err := r.repository.SaveReport(ctx, &report); err != nil {
		return ReportWithAuthor{}, err
	}

	slog.Info("New report created", slog.Int64("report_id", report.ReportID))

	if demo.DemoID > 0 && !demo.Archive {
		if errMark := r.demos.MarkArchived(ctx, &demo); errMark != nil {
			slog.Error("Failed to mark demo as archived", log.ErrAttr(errMark))
		}
	}

	newReport, errReport := r.GetReport(ctx, currentUser, report.ReportID)
	if errReport != nil {
		return ReportWithAuthor{}, errReport
	}

	//	conf := r.config.Config()
	// demoURL := ""
	// if report.DemoID > 0 {
	// 	demoURL = conf.ExtURLRaw("/asset/%s", demo.AssetID.String())
	// }
	// r.notifications.Enqueue(ctx, notification.NewDiscordNotification(
	// 	discord.ChannelModAppealLog,
	// 	discord.NewInGameReportResponse(newReport, conf.ExtURL(report), currentUser, conf.ExtURL(currentUser), demoURL)))
	// r.notifications.Enqueue(ctx, notification.NewSiteGroupNotificationWithAuthor(
	// 	[]permission.Privilege{permission.PModerator, permission.PAdmin},
	// 	notification.SeverityInfo,
	// 	fmt.Sprintf("A new report was created. Author: %s, Target: %s", currentUser.GetName(), personTarget.PersonaName),
	// 	newReport.Path(),
	// 	currentUser,
	// ))

	return newReport, nil
}

func (r ReportUsecase) EditReportMessage(ctx context.Context, reportMessageID int64, curUser domain.PersonInfo, req RequestMessageBodyMD) (ReportMessage, error) {
	if reportMessageID <= 0 {
		return ReportMessage{}, domain.ErrParamInvalid
	}

	existing, errExist := r.GetReportMessageByID(ctx, reportMessageID)
	if errExist != nil {
		return ReportMessage{}, errExist
	}

	if !httphelper.HasPrivilege(curUser, steamid.Collection{existing.AuthorID}, permission.PModerator) {
		return ReportMessage{}, permission.ErrPermissionDenied
	}

	req.BodyMD = strings.TrimSpace(req.BodyMD)

	if req.BodyMD == "" {
		return ReportMessage{}, domain.ErrInvalidParameter
	}

	if req.BodyMD == existing.MessageMD {
		return ReportMessage{}, database.ErrDuplicate // TODO replace
	}

	existing.MessageMD = req.BodyMD

	if errSave := r.repository.SaveReportMessage(ctx, &existing); errSave != nil {
		return ReportMessage{}, errSave
	}

	// conf := r.config.Config()

	// r.notifications.Enqueue(ctx, notification.NewDiscordNotification(
	// 	discord.ChannelModAppealLog,
	// 	discord.EditReportMessageResponse(req.BodyMD, existing.MessageMD,
	// 		conf.ExtURLRaw("/report/%d", existing.ReportID), curUser, conf.ExtURL(curUser))))

	slog.Info("Report message edited", slog.Int64("report_message_id", reportMessageID))

	return r.GetReportMessageByID(ctx, reportMessageID)
}

func (r ReportUsecase) CreateReportMessage(ctx context.Context, reportID int64, curUser domain.PersonInfo, req RequestMessageBodyMD) (ReportMessage, error) {
	req.BodyMD = strings.TrimSpace(req.BodyMD)

	if req.BodyMD == "" {
		return ReportMessage{}, domain.ErrParamInvalid
	}

	report, errReport := r.GetReport(ctx, curUser, reportID)
	if errReport != nil {
		return ReportMessage{}, errReport
	}

	msg := NewReportMessage(reportID, curUser.GetSteamID(), req.BodyMD)
	if err := r.repository.SaveReportMessage(ctx, &msg); err != nil {
		return ReportMessage{}, err
	}

	report.UpdatedOn = time.Now()

	if errSave := r.repository.SaveReport(ctx, &report.Report); errSave != nil {
		return ReportMessage{}, errSave
	}

	// conf := r.config.Config()

	// r.notifications.Enqueue(ctx, notification.NewDiscordNotification(
	// 	discord.ChannelModAppealLog,
	// 	discord.NewReportMessageResponse(msg.MessageMD, conf.ExtURL(report), curUser, conf.ExtURL(curUser))))

	// path := fmt.Sprintf("/report/%d", reportID)
	//
	// r.notifications.Enqueue(ctx, notification.NewSiteGroupNotificationWithAuthor(
	// 	[]permission.Privilege{permission.PModerator, permission.PAdmin},
	// 	notification.SeverityInfo,
	// 	"A new report reply has been posted. Author: "+curUser.GetName(),
	// 	path,
	// 	curUser,
	// ))

	// if report.Author.SteamID != curUser.GetSteamID() {
	// 	r.notifications.Enqueue(ctx, notification.NewSiteUserNotification(
	// 		[]steamid.SteamID{report.Author.SteamID},
	// 		notification.SeverityInfo,
	// 		"A new report reply has been posted",
	// 		path,
	// 	))
	// }

	sid := curUser.GetSteamID()
	slog.Info("New report message created",
		slog.Int64("report_id", reportID), slog.String("steam_id", sid.String()))

	return msg, nil
}

type MetaInfoArgs struct{}

func (args MetaInfoArgs) Kind() string {
	return "reports_meta"
}

func (args MetaInfoArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: string(queue.Default), UniqueOpts: river.UniqueOpts{ByPeriod: time.Hour * 24}}
}

func NewMetaInfoWorker(reports ReportUsecase) *MetaInfoWorker {
	return &MetaInfoWorker{reports: reports}
}

type MetaInfoWorker struct {
	river.WorkerDefaults[MetaInfoArgs]
	reports ReportUsecase
}

func (worker *MetaInfoWorker) Work(ctx context.Context, _ *river.Job[MetaInfoArgs]) error {
	return worker.reports.GenerateMetaStats(ctx)
}
