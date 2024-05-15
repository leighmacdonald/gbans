package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

type ReportRepository interface {
	GetReportBySteamID(ctx context.Context, authorID steamid.SteamID, steamID steamid.SteamID) (Report, error)
	GetReports(ctx context.Context, opts ReportQueryFilter) ([]Report, error)
	GetReport(ctx context.Context, reportID int64) (Report, error)
	GetReportMessages(ctx context.Context, reportID int64) ([]ReportMessage, error)
	GetReportMessageByID(ctx context.Context, reportMessageID int64) (ReportMessage, error)
	DropReportMessage(ctx context.Context, message *ReportMessage) error
	DropReport(ctx context.Context, report *Report) error
	SaveReport(ctx context.Context, report *Report) error
	SaveReportMessage(ctx context.Context, message *ReportMessage) error
}

type ReportUsecase interface {
	GetReportBySteamID(ctx context.Context, authorID steamid.SteamID, steamID steamid.SteamID) (Report, error)
	GetReports(ctx context.Context, user PersonInfo, opts ReportQueryFilter) ([]ReportWithAuthor, error)
	GetReport(ctx context.Context, curUser PersonInfo, reportID int64) (ReportWithAuthor, error)
	GetReportMessages(ctx context.Context, reportID int64) ([]ReportMessage, error)
	GetReportMessageByID(ctx context.Context, reportMessageID int64) (ReportMessage, error)
	DropReportMessage(ctx context.Context, message *ReportMessage) error
	DropReport(ctx context.Context, report *Report) error
	SaveReport(ctx context.Context, report *Report) error
	SaveReportMessage(ctx context.Context, message *ReportMessage) error
}

type CreateReportReq struct {
	SourceID        steamid.SteamID `json:"source_id"`
	TargetID        steamid.SteamID `json:"target_id"`
	Description     string          `json:"description"`
	Reason          Reason          `json:"reason"`
	ReasonText      string          `json:"reason_text"`
	DemoID          int64           `json:"demo_id"`
	DemoTick        int             `json:"demo_tick"`
	PersonMessageID int64           `json:"person_message_id"`
}

type ReportStatus int

const (
	AnyStatus                        = -1
	Opened              ReportStatus = 0
	NeedMoreInfo                     = 1
	ClosedWithoutAction              = 2
	ClosedWithAction                 = 3
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
	Reason          Reason          `json:"reason"`
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
	Author  Person   `json:"author"`
	Subject Person   `json:"subject"`
	Demo    DemoFile `json:"demo"`
	Report
}

type ReportMessage struct {
	ReportID        int64           `json:"report_id"`
	ReportMessageID int64           `json:"report_message_id"`
	AuthorID        steamid.SteamID `json:"author_id"`
	MessageMD       string          `json:"message_md"`
	Deleted         bool            `json:"deleted"`
	CreatedOn       time.Time       `json:"created_on"`
	UpdatedOn       time.Time       `json:"updated_on"`
	SimplePerson
}

func NewReportMessage(reportID int64, authorID steamid.SteamID, messageMD string) ReportMessage {
	now := time.Now()

	return ReportMessage{
		ReportID:     reportID,
		AuthorID:     authorID,
		MessageMD:    messageMD,
		Deleted:      false,
		CreatedOn:    now,
		UpdatedOn:    now,
		SimplePerson: SimplePerson{},
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
