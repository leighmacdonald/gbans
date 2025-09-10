package report

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/queue"
	"github.com/leighmacdonald/gbans/internal/steam"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/riverqueue/river"
)

type reportUsecase struct {
	repository    domain.ReportRepository
	notifications domain.NotificationUsecase
	config        domain.ConfigUsecase
	persons       domain.PersonUsecase
	demos         domain.DemoUsecase
	tfAPI         *thirdparty.TFAPI
}

func NewReportUsecase(repository domain.ReportRepository, notifications domain.NotificationUsecase,
	config domain.ConfigUsecase, persons domain.PersonUsecase, demos domain.DemoUsecase, tfAPI *thirdparty.TFAPI,
) domain.ReportUsecase {
	return &reportUsecase{
		notifications: notifications,
		repository:    repository,
		config:        config,
		persons:       persons,
		demos:         demos,
		tfAPI:         tfAPI,
	}
}

func (r reportUsecase) GenerateMetaStats(ctx context.Context) error {
	reports, errReports := r.GetReports(ctx)
	if errReports != nil {
		slog.Error("failed to fetch reports for report metadata", log.ErrAttr(errReports))

		return errReports
	}

	var (
		now  = time.Now()
		meta domain.ReportMeta
	)

	for _, report := range reports {
		if report.ReportStatus == domain.ClosedWithAction || report.ReportStatus == domain.ClosedWithoutAction {
			meta.TotalClosed++

			continue
		}

		meta.TotalOpen++

		if report.ReportStatus == domain.NeedMoreInfo {
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

	r.notifications.Enqueue(ctx, domain.NewDiscordNotification(
		discord.ChannelMod,
		discord.ReportStatsMessage(meta, r.config.ExtURLRaw("/admin/reports"))))

	return nil
}

func (r reportUsecase) addAuthorsToReports(ctx context.Context, reports []domain.Report) ([]domain.ReportWithAuthor, error) {
	var peopleIDs steamid.Collection
	for _, report := range reports {
		peopleIDs = append(peopleIDs, report.SourceID, report.TargetID)
	}

	people, errAuthors := r.persons.GetPeopleBySteamID(ctx, nil, fp.Uniq(peopleIDs))
	if errAuthors != nil {
		return nil, errAuthors
	}

	peopleMap := people.AsMap()

	userReports := make([]domain.ReportWithAuthor, len(reports))

	for i, report := range reports {
		userReports[i] = domain.ReportWithAuthor{
			Author:  peopleMap[report.SourceID],
			Report:  report,
			Subject: peopleMap[report.TargetID],
		}
	}

	return userReports, nil
}

func (r reportUsecase) SetReportStatus(ctx context.Context, reportID int64, user domain.UserProfile, status domain.ReportStatus) (domain.ReportWithAuthor, error) {
	report, errGet := r.GetReport(ctx, user, reportID)
	if errGet != nil {
		return report, errGet
	}

	if report.ReportStatus == status {
		return report, database.ErrDuplicate // TODO proper specific error
	}

	fromStatus := report.ReportStatus

	report.ReportStatus = status

	if errSave := r.repository.SaveReport(ctx, &report.Report); errSave != nil {
		return report, errSave
	}

	r.notifications.Enqueue(ctx, domain.NewDiscordNotification(
		discord.ChannelMod,
		discord.ReportStatusChangeMessage(report, fromStatus, r.config.ExtURL(report))))

	r.notifications.Enqueue(ctx, domain.NewSiteGroupNotificationWithAuthor(
		[]domain.Privilege{domain.PModerator, domain.PAdmin},
		domain.SeverityInfo,
		fmt.Sprintf("A report status has changed: %s -> %s", fromStatus, status),
		report.Path(),
		user,
	))

	r.notifications.Enqueue(ctx, domain.NewSiteUserNotification(
		[]steamid.SteamID{report.Author.SteamID},
		domain.SeverityInfo,
		fmt.Sprintf("Your report status has changed: %s -> %s", fromStatus, status),
		report.Path(),
	))

	slog.Info("Report status changed",
		slog.Int64("report_id", report.ReportID),
		slog.String("to_status", report.ReportStatus.String()))

	return report, nil
}

func (r reportUsecase) GetReportsBySteamID(ctx context.Context, steamID steamid.SteamID) ([]domain.ReportWithAuthor, error) {
	if !steamID.Valid() {
		return nil, domain.ErrInvalidSID
	}

	reports, errReports := r.repository.GetReports(ctx, steamID)
	if errReports != nil {
		if errors.Is(errReports, database.ErrNoResult) {
			return []domain.ReportWithAuthor{}, nil
		}

		return nil, errReports
	}

	return r.addAuthorsToReports(ctx, reports)
}

func (r reportUsecase) GetReports(ctx context.Context) ([]domain.ReportWithAuthor, error) {
	reports, errReports := r.repository.GetReports(ctx, steamid.SteamID{})
	if errReports != nil {
		if errors.Is(errReports, database.ErrNoResult) {
			return []domain.ReportWithAuthor{}, nil
		}

		return nil, errReports
	}

	return r.addAuthorsToReports(ctx, reports)
}

func (r reportUsecase) GetReport(ctx context.Context, curUser domain.PersonInfo, reportID int64) (domain.ReportWithAuthor, error) {
	report, err := r.repository.GetReport(ctx, reportID)
	if err != nil {
		return domain.ReportWithAuthor{}, err
	}

	author, errAuthor := r.persons.GetPersonBySteamID(ctx, nil, report.SourceID)
	if errAuthor != nil {
		return domain.ReportWithAuthor{}, errAuthor
	}

	if !httphelper.HasPrivilege(curUser, steamid.Collection{author.SteamID}, domain.PModerator) {
		return domain.ReportWithAuthor{}, domain.ErrPermissionDenied
	}

	target, errTarget := r.persons.GetPersonBySteamID(ctx, nil, report.TargetID)
	if errTarget != nil {
		return domain.ReportWithAuthor{}, errTarget
	}

	var demo domain.DemoFile
	if report.DemoID > 0 {
		if errDemo := r.demos.GetDemoByID(ctx, report.DemoID, &demo); errDemo != nil {
			slog.Error("Failed to load report demo", slog.Int64("report_id", report.ReportID))
		}
	}

	return domain.ReportWithAuthor{
		Author:  author,
		Subject: target,
		Report:  report,
		Demo:    demo,
	}, nil
}

func (r reportUsecase) GetReportBySteamID(ctx context.Context, authorID steamid.SteamID, steamID steamid.SteamID) (domain.Report, error) {
	return r.repository.GetReportBySteamID(ctx, authorID, steamID)
}

func (r reportUsecase) GetReportMessages(ctx context.Context, reportID int64) ([]domain.ReportMessage, error) {
	return r.repository.GetReportMessages(ctx, reportID)
}

func (r reportUsecase) GetReportMessageByID(ctx context.Context, reportMessageID int64) (domain.ReportMessage, error) {
	return r.repository.GetReportMessageByID(ctx, reportMessageID)
}

func (r reportUsecase) DropReportMessage(ctx context.Context, curUser domain.PersonInfo, reportMessageID int64) error {
	existing, errExist := r.repository.GetReportMessageByID(ctx, reportMessageID)
	if errExist != nil {
		return errExist
	}

	if !httphelper.HasPrivilege(curUser, steamid.Collection{existing.AuthorID}, domain.PModerator) {
		return domain.ErrPermissionDenied
	}

	if err := r.repository.DropReportMessage(ctx, &existing); err != nil {
		return err
	}

	r.notifications.Enqueue(ctx, domain.NewDiscordNotification(
		discord.ChannelModAppealLog,
		discord.DeleteReportMessage(existing, curUser, r.config.ExtURL(curUser))))

	slog.Info("Deleted report message", slog.Int64("report_message_id", reportMessageID))

	return nil
}

func (r reportUsecase) DropReport(ctx context.Context, report *domain.Report) error {
	return r.repository.DropReport(ctx, report)
}

func (r reportUsecase) SaveReport(ctx context.Context, currentUser domain.UserProfile, req domain.RequestReportCreate) (domain.ReportWithAuthor, error) {
	if req.Description == "" || len(req.Description) < 10 {
		return domain.ReportWithAuthor{}, fmt.Errorf("%w: description", domain.ErrParamInvalid)
	}

	// ServerStore initiated requests will have a sourceID set by the server
	// Web based reports the source should not be set, the reporter will be taken from the
	// current session information instead
	if !req.SourceID.Valid() {
		req.SourceID = currentUser.SteamID
	}

	if !req.SourceID.Valid() {
		return domain.ReportWithAuthor{}, fmt.Errorf("%w: source_id", domain.ErrParamInvalid)
	}

	if !req.TargetID.Valid() {
		return domain.ReportWithAuthor{}, fmt.Errorf("%w: target_id", domain.ErrParamInvalid)
	}

	if req.SourceID.Int64() == req.TargetID.Int64() {
		return domain.ReportWithAuthor{}, fmt.Errorf("%w: cannot report self", domain.ErrParamInvalid)
	}

	personSource, errSource := r.persons.GetPersonBySteamID(ctx, nil, req.SourceID)
	if errSource != nil {
		return domain.ReportWithAuthor{}, errSource
	}

	personTarget, errTarget := r.persons.GetOrCreatePersonBySteamID(ctx, nil, req.TargetID)
	if errTarget != nil {
		return domain.ReportWithAuthor{}, errTarget
	}

	if personTarget.Expired() {
		if err := steam.UpdatePlayerSummary(ctx, &personTarget, r.tfAPI); err != nil {
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
			return domain.ReportWithAuthor{}, errReports
		}
	}

	if existing.ReportID > 0 {
		return domain.ReportWithAuthor{}, domain.ErrReportExists
	}

	var demo domain.DemoFile

	if req.DemoID > 0 {
		if errDemo := r.demos.GetDemoByID(ctx, req.DemoID, &demo); errDemo != nil {
			return domain.ReportWithAuthor{}, errDemo
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

	if err := r.repository.SaveReport(ctx, &report); err != nil {
		return domain.ReportWithAuthor{}, err
	}

	slog.Info("New report created", slog.Int64("report_id", report.ReportID))

	if demo.DemoID > 0 && !demo.Archive {
		if errMark := r.demos.MarkArchived(ctx, &demo); errMark != nil {
			slog.Error("Failed to mark demo as archived", log.ErrAttr(errMark))
		}
	}

	conf := r.config.Config()

	demoURL := ""

	if report.DemoID > 0 {
		demoURL = conf.ExtURLRaw("/asset/%s", demo.AssetID.String())
	}

	newReport, errReport := r.GetReport(ctx, currentUser, report.ReportID)
	if errReport != nil {
		return domain.ReportWithAuthor{}, errReport
	}

	r.notifications.Enqueue(ctx, domain.NewDiscordNotification(
		discord.ChannelModAppealLog,
		discord.NewInGameReportResponse(newReport, conf.ExtURL(report), currentUser, conf.ExtURL(currentUser), demoURL)))

	r.notifications.Enqueue(ctx, domain.NewSiteGroupNotificationWithAuthor(
		[]domain.Privilege{domain.PModerator, domain.PAdmin},
		domain.SeverityInfo,
		fmt.Sprintf("A new report was created. Author: %s, Target: %s", currentUser.GetName(), personTarget.PersonaName),
		newReport.Path(),
		currentUser,
	))

	return newReport, nil
}

func (r reportUsecase) EditReportMessage(ctx context.Context, reportMessageID int64, curUser domain.PersonInfo, req domain.RequestMessageBodyMD) (domain.ReportMessage, error) {
	if reportMessageID <= 0 {
		return domain.ReportMessage{}, domain.ErrParamInvalid
	}

	existing, errExist := r.GetReportMessageByID(ctx, reportMessageID)
	if errExist != nil {
		return domain.ReportMessage{}, errExist
	}

	if !httphelper.HasPrivilege(curUser, steamid.Collection{existing.AuthorID}, domain.PModerator) {
		return domain.ReportMessage{}, domain.ErrPermissionDenied
	}

	req.BodyMD = strings.TrimSpace(req.BodyMD)

	if req.BodyMD == "" {
		return domain.ReportMessage{}, domain.ErrInvalidParameter
	}

	if req.BodyMD == existing.MessageMD {
		return domain.ReportMessage{}, database.ErrDuplicate // TODO replace
	}

	existing.MessageMD = req.BodyMD

	if errSave := r.repository.SaveReportMessage(ctx, &existing); errSave != nil {
		return domain.ReportMessage{}, errSave
	}

	conf := r.config.Config()

	r.notifications.Enqueue(ctx, domain.NewDiscordNotification(
		discord.ChannelModAppealLog,
		discord.EditReportMessageResponse(req.BodyMD, existing.MessageMD,
			conf.ExtURLRaw("/report/%d", existing.ReportID), curUser, conf.ExtURL(curUser))))

	slog.Info("Report message edited", slog.Int64("report_message_id", reportMessageID))

	return r.GetReportMessageByID(ctx, reportMessageID)
}

func (r reportUsecase) CreateReportMessage(ctx context.Context, reportID int64, curUser domain.UserProfile, req domain.RequestMessageBodyMD) (domain.ReportMessage, error) {
	req.BodyMD = strings.TrimSpace(req.BodyMD)

	if req.BodyMD == "" {
		return domain.ReportMessage{}, domain.ErrParamInvalid
	}

	report, errReport := r.GetReport(ctx, curUser, reportID)
	if errReport != nil {
		return domain.ReportMessage{}, errReport
	}

	msg := domain.NewReportMessage(reportID, curUser.GetSteamID(), req.BodyMD)
	if err := r.repository.SaveReportMessage(ctx, &msg); err != nil {
		return domain.ReportMessage{}, err
	}

	report.UpdatedOn = time.Now()

	if errSave := r.repository.SaveReport(ctx, &report.Report); errSave != nil {
		return domain.ReportMessage{}, errSave
	}

	conf := r.config.Config()

	r.notifications.Enqueue(ctx, domain.NewDiscordNotification(
		discord.ChannelModAppealLog,
		discord.NewReportMessageResponse(msg.MessageMD, conf.ExtURL(report), curUser, conf.ExtURL(curUser))))

	path := fmt.Sprintf("/report/%d", reportID)

	r.notifications.Enqueue(ctx, domain.NewSiteGroupNotificationWithAuthor(
		[]domain.Privilege{domain.PModerator, domain.PAdmin},
		domain.SeverityInfo,
		"A new report reply has been posted. Author: "+curUser.GetName(),
		path,
		curUser,
	))

	if report.Author.SteamID != curUser.SteamID {
		r.notifications.Enqueue(ctx, domain.NewSiteUserNotification(
			[]steamid.SteamID{report.Author.SteamID},
			domain.SeverityInfo,
			"A new report reply has been posted",
			path,
		))
	}

	slog.Info("New report message created",
		slog.Int64("report_id", reportID), slog.String("steam_id", curUser.SteamID.String()))

	return msg, nil
}

type MetaInfoArgs struct{}

func (args MetaInfoArgs) Kind() string {
	return "reports_meta"
}

func (args MetaInfoArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: string(queue.Default), UniqueOpts: river.UniqueOpts{ByPeriod: time.Hour * 24}}
}

func NewMetaInfoWorker(reports domain.ReportUsecase) *MetaInfoWorker {
	return &MetaInfoWorker{reports: reports}
}

type MetaInfoWorker struct {
	river.WorkerDefaults[MetaInfoArgs]
	reports domain.ReportUsecase
}

func (worker *MetaInfoWorker) Work(ctx context.Context, _ *river.Job[MetaInfoArgs]) error {
	return worker.reports.GenerateMetaStats(ctx)
}
