package ban

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/database"
	personDomain "github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/sliceutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	ErrReportStateUpdate = errors.New("failed to update report state")
	ErrReportExists      = errors.New("duplicate user report")
)

type RequestMessageBodyMD struct {
	BodyMD string `json:"body_md"`
}

type ReportQueryFilter struct {
	httphelper.SourceIDField

	Deleted bool `json:"deleted"`
}

type RequestReportStatusUpdate struct {
	Status ReportStatus `json:"status"`
}

type RequestReportCreate struct {
	SourceID        steamid.SteamID `json:"source_id"`
	TargetID        steamid.SteamID `json:"target_id"`
	Description     string          `json:"description"`
	Reason          reason.Reason   `json:"reason"`
	ReasonText      string          `json:"reason_text"`
	DemoID          int64           `json:"demo_id"`
	DemoTick        int             `json:"demo_tick"`
	PersonMessageID int64           `json:"person_message_id"`
}

type ReportStatus int

const (
	AnyStatus ReportStatus = iota - 1
	Opened
	NeedMoreInfo
	ClosedWithoutAction
	ClosedWithAction
)

func (status ReportStatus) String() string {
	switch status {
	case ClosedWithoutAction:
		return "Closed without action"
	case ClosedWithAction:
		return "Closed with action"
	case Opened:
		return "Opened"
	default:
		return "Need more information"
	}
}

type Report struct {
	ReportID        int64           `json:"report_id"`
	SourceID        steamid.SteamID `json:"source_id"`
	TargetID        steamid.SteamID `json:"target_id"`
	Description     string          `json:"description"`
	ReportStatus    ReportStatus    `json:"report_status"`
	Reason          reason.Reason   `json:"reason"`
	ReasonText      string          `json:"reason_text"`
	Deleted         bool            `json:"deleted"`
	DemoTick        int             `json:"demo_tick"`
	DemoID          int64           `json:"demo_id"`
	PersonMessageID int64           `json:"person_message_id"`
	CreatedOn       time.Time       `json:"created_on"`
	UpdatedOn       time.Time       `json:"updated_on"`
}

func (report Report) Path() string {
	return fmt.Sprintf("/report/%d", report.ReportID)
}

func NewReport() Report {
	return Report{
		ReportID:     0,
		SourceID:     steamid.SteamID{},
		Description:  "",
		ReportStatus: 0,
		CreatedOn:    time.Now(),
		UpdatedOn:    time.Now(),
		DemoTick:     -1,
		DemoID:       0,
	}
}

type ReportWithAuthor struct {
	Report

	Author  personDomain.Core `json:"author"`
	Subject personDomain.Core `json:"subject"`
	// TODO FIX Demo    demo.DemoFile `json:"demo"`
}

type ReportMessage struct {
	ReportID        int64                `json:"report_id"`
	ReportMessageID int64                `json:"report_message_id"`
	AuthorID        steamid.SteamID      `json:"author_id"`
	MessageMD       string               `json:"message_md"`
	Deleted         bool                 `json:"deleted"`
	CreatedOn       time.Time            `json:"created_on"`
	UpdatedOn       time.Time            `json:"updated_on"`
	Personaname     string               `json:"personaname"`
	Avatarhash      string               `json:"avatarhash"`
	PermissionLevel permission.Privilege `json:"permission_level"`
}

func NewReportMessage(reportID int64, authorID steamid.SteamID, messageMD string) ReportMessage {
	now := time.Now()

	return ReportMessage{
		ReportID:  reportID,
		AuthorID:  authorID,
		MessageMD: messageMD,
		Deleted:   false,
		CreatedOn: now,
		UpdatedOn: now,
	}
}

type ReportMeta struct {
	TotalOpen   int
	TotalClosed int
	Open        int
	NeedInfo    int
	Open1Day    int
	Open3Days   int
	OpenWeek    int
	OpenNew     int
}

type Reports struct {
	repository    ReportRepository
	persons       *person.Persons
	demos         servers.Demos
	tfAPI         *thirdparty.TFAPI
	notif         notification.Notifier
	logChannel    string
	appealChannel string
}

func NewReports(repo ReportRepository, persons *person.Persons, demos servers.Demos, tfAPI *thirdparty.TFAPI,
	notif notification.Notifier, logChannel string, appealChannel string,
) Reports {
	return Reports{
		repository:    repo,
		persons:       persons,
		demos:         demos,
		tfAPI:         tfAPI,
		notif:         notif,
		logChannel:    logChannel,
		appealChannel: appealChannel,
	}
}

func (r Reports) MetaStats(ctx context.Context) error {
	reports, errReports := r.Reports(ctx)
	if errReports != nil {
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

	r.notif.Send(notification.NewDiscord(
		r.logChannel,
		ReportStatsMessage(meta, "/admin/reports")))

	return nil
}

func (r Reports) addAuthorsToReports(ctx context.Context, reports []Report) ([]ReportWithAuthor, error) {
	var peopleIDs steamid.Collection
	for _, report := range reports {
		peopleIDs = append(peopleIDs, report.SourceID, report.TargetID)
	}

	people, errAuthors := r.persons.BySteamIDs(ctx, sliceutil.Uniq(peopleIDs))
	if errAuthors != nil {
		return nil, errAuthors
	}

	peopleMap := people.AsMap()
	userReports := make([]ReportWithAuthor, len(reports))

	for i, report := range reports {
		userReports[i] = ReportWithAuthor{
			Author:  peopleMap[report.SourceID].Core(),
			Report:  report,
			Subject: peopleMap[report.TargetID].Core(),
		}
	}

	return userReports, nil
}

func (r Reports) SetReportStatus(ctx context.Context, reportID int64, user personDomain.Info, status ReportStatus) (ReportWithAuthor, error) {
	report, errGet := r.Report(ctx, user, reportID)
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

	r.notif.Send(notification.NewDiscordNext(
		r.logChannel,
		ReportStatusChangeMessage(report, fromStatus)))

	r.notif.Send(notification.NewSiteGroupNotificationWithAuthor(
		[]permission.Privilege{permission.Moderator, permission.Admin},
		notification.Info,
		fmt.Sprintf("A report status has changed: %s -> %s", fromStatus, status),
		link.Path(report),
		user,
	))

	r.notif.Send(notification.NewSiteUser(
		[]steamid.SteamID{report.Author.SteamID},
		notification.Info,
		fmt.Sprintf("Your report status has changed: %s -> %s", fromStatus, status),
		link.Path(report),
	))

	slog.Info("Report status changed",
		slog.Int64("report_id", report.ReportID),
		slog.String("to_status", report.ReportStatus.String()))

	return report, nil
}

func (r Reports) BySteamID(ctx context.Context, steamID steamid.SteamID) ([]ReportWithAuthor, error) {
	if !steamID.Valid() {
		return nil, steamid.ErrInvalidSID
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

func (r Reports) Reports(ctx context.Context) ([]ReportWithAuthor, error) {
	reports, errReports := r.repository.GetReports(ctx, steamid.SteamID{})
	if errReports != nil {
		if errors.Is(errReports, database.ErrNoResult) {
			return []ReportWithAuthor{}, nil
		}

		return nil, errReports
	}

	return r.addAuthorsToReports(ctx, reports)
}

func (r Reports) Report(ctx context.Context, curUser personDomain.Info, reportID int64) (ReportWithAuthor, error) {
	report, err := r.repository.GetReport(ctx, reportID)
	if err != nil {
		return ReportWithAuthor{}, err
	}

	author, errAuthor := r.persons.BySteamID(ctx, report.SourceID)
	if errAuthor != nil {
		return ReportWithAuthor{}, errAuthor
	}

	if !httphelper.HasPrivilege(curUser, steamid.Collection{author.SteamID}, permission.Moderator) {
		return ReportWithAuthor{}, permission.ErrDenied
	}

	target, errTarget := r.persons.BySteamID(ctx, report.TargetID)
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
		Author:  author.Core(),
		Subject: target.Core(),
		Report:  report,
		// TODO FIX Demo:    demo,
	}, nil
}

func (r Reports) ReportBySteamID(ctx context.Context, authorID steamid.SteamID, steamID steamid.SteamID) (Report, error) {
	return r.repository.GetReportBySteamID(ctx, authorID, steamID)
}

func (r Reports) Messages(ctx context.Context, reportID int64) ([]ReportMessage, error) {
	return r.repository.GetReportMessages(ctx, reportID)
}

func (r Reports) MessageByID(ctx context.Context, reportMessageID int64) (ReportMessage, error) {
	return r.repository.GetReportMessageByID(ctx, reportMessageID)
}

func (r Reports) DropMessage(ctx context.Context, curUser personDomain.Info, reportMessageID int64) error {
	existing, errExist := r.repository.GetReportMessageByID(ctx, reportMessageID)
	if errExist != nil {
		return errExist
	}

	if !httphelper.HasPrivilege(curUser, steamid.Collection{existing.AuthorID}, permission.Moderator) {
		return permission.ErrDenied
	}

	if err := r.repository.DropReportMessage(ctx, &existing); err != nil {
		return err
	}

	r.notif.Send(notification.NewDiscordNext(r.appealChannel,
		DeleteReportMessage(existing, curUser)))

	slog.Info("Deleted report message", slog.Int64("report_message_id", reportMessageID))

	return nil
}

func (r Reports) Drop(ctx context.Context, report *Report) error {
	return r.repository.DropReport(ctx, report)
}

func (r Reports) Save(ctx context.Context, currentUser personDomain.Info, req RequestReportCreate) (ReportWithAuthor, error) {
	if req.Description == "" || len(req.Description) < 10 {
		return ReportWithAuthor{}, fmt.Errorf("%w: description", httphelper.ErrParamInvalid)
	}

	// Server initiated requests will have a sourceID set by the server
	// Web based reports the source should not be set, the reporter will be taken from the
	// current session information instead to avoid forging.
	if !req.SourceID.Valid() {
		req.SourceID = currentUser.GetSteamID()
	}

	if !req.SourceID.Valid() {
		return ReportWithAuthor{}, fmt.Errorf("%w: source_id", httphelper.ErrParamInvalid)
	}

	if !req.TargetID.Valid() {
		return ReportWithAuthor{}, fmt.Errorf("%w: target_id", httphelper.ErrParamInvalid)
	}

	if req.SourceID.Int64() == req.TargetID.Int64() {
		return ReportWithAuthor{}, fmt.Errorf("%w: cannot report self", httphelper.ErrParamInvalid)
	}

	personSource, errSource := r.persons.GetOrCreatePersonBySteamID(ctx, req.SourceID)
	if errSource != nil {
		return ReportWithAuthor{}, errSource
	}

	personTarget, errTarget := r.persons.GetOrCreatePersonBySteamID(ctx, req.TargetID)
	if errTarget != nil {
		return ReportWithAuthor{}, errTarget
	}
	// TODO readd
	// if personTarget.Expired() {
	// 	if err := person.UpdatePlayerSummary(ctx, &personTarget, r.tfAPI); err != nil {
	// 		slog.Error("Failed to update target player", log.ErrAttr(err))
	// 	} else {
	// 		if errSave := r.persons.Save(ctx, &personTarget); errSave != nil {
	// 			slog.Error("Failed to save target player update", log.ErrAttr(err))
	// 		}
	// 	}
	// }

	// Ensure the user doesn't already have an open report against the user
	existing, errReports := r.ReportBySteamID(ctx, personSource.SteamID, req.TargetID)
	if errReports != nil {
		if !errors.Is(errReports, database.ErrNoResult) {
			return ReportWithAuthor{}, errReports
		}
	}

	if existing.ReportID > 0 {
		return ReportWithAuthor{}, ErrReportExists
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
			slog.Error("Failed to mark demo as archived", slog.String("error", errMark.Error()))
		}
	}

	newReport, errReport := r.Report(ctx, currentUser, report.ReportID)
	if errReport != nil {
		return ReportWithAuthor{}, errReport
	}

	go r.notif.Send(notification.NewDiscordNext(r.appealChannel, NewInGameReportResponse(newReport)))
	go r.notif.Send(notification.NewSiteGroupNotificationWithAuthor(
		[]permission.Privilege{permission.Moderator, permission.Admin},
		notification.Info,
		fmt.Sprintf("A new report was created. Author: %s, Target: %s", currentUser.GetName(), personTarget.GetName()),
		link.Path(newReport),
		currentUser,
	))

	return newReport, nil
}

func (r Reports) EditMessage(ctx context.Context, reportMessageID int64, curUser personDomain.Info, req RequestMessageBodyMD) (ReportMessage, error) {
	if reportMessageID <= 0 {
		return ReportMessage{}, httphelper.ErrParamInvalid
	}

	existing, errExist := r.MessageByID(ctx, reportMessageID)
	if errExist != nil {
		return ReportMessage{}, errExist
	}

	if !httphelper.HasPrivilege(curUser, steamid.Collection{existing.AuthorID}, permission.Moderator) {
		return ReportMessage{}, permission.ErrDenied
	}

	req.BodyMD = strings.TrimSpace(req.BodyMD)

	if req.BodyMD == "" {
		return ReportMessage{}, httphelper.ErrInvalidParameter
	}

	if req.BodyMD == existing.MessageMD {
		return ReportMessage{}, database.ErrDuplicate // TODO replace
	}

	existing.MessageMD = req.BodyMD

	if errSave := r.repository.SaveReportMessage(ctx, &existing); errSave != nil {
		return ReportMessage{}, errSave
	}

	r.notif.Send(notification.NewDiscordNext(r.appealChannel,
		EditReportMessageResponse(req.BodyMD, existing.MessageMD,
			fmt.Sprintf("/report/%d", existing.ReportID), curUser, link.Path(curUser))))

	slog.Info("Report message edited", slog.Int64("report_message_id", reportMessageID))

	return r.MessageByID(ctx, reportMessageID)
}

func (r Reports) CreateMessage(ctx context.Context, reportID int64, curUser personDomain.Info, req RequestMessageBodyMD) (ReportMessage, error) {
	req.BodyMD = strings.TrimSpace(req.BodyMD)

	if req.BodyMD == "" {
		return ReportMessage{}, httphelper.ErrParamInvalid
	}

	report, errReport := r.Report(ctx, curUser, reportID)
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

	go r.notif.Send(notification.NewDiscordNext(r.appealChannel, NewReportMessageResponse(report, msg)))

	r.notif.Send(notification.NewSiteGroupNotificationWithAuthor(
		[]permission.Privilege{permission.Moderator, permission.Admin},
		notification.Info,
		"A new report reply has been posted. Author: "+curUser.GetName(),
		link.Path(report),
		curUser,
	))

	if report.Author.SteamID != curUser.GetSteamID() {
		r.notif.Send(notification.NewSiteUser(
			[]steamid.SteamID{report.Author.SteamID},
			notification.Info,
			"A new report reply has been posted",
			link.Path(report),
		))
	}

	sid := curUser.GetSteamID()
	slog.Info("New report message created",
		slog.Int64("report_id", reportID), slog.String("steam_id", sid.String()))

	return msg, nil
}
