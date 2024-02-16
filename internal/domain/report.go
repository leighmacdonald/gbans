package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/leighmacdonald/steamid/v3/steamid"
)

type ReportRepository interface {
	GetReportBySteamID(ctx context.Context, authorID steamid.SID64, steamID steamid.SID64) (Report, error)
	GetReports(ctx context.Context, opts ReportQueryFilter) ([]Report, int64, error)
	GetReport(ctx context.Context, reportID int64) (Report, error)
	GetReportMessages(ctx context.Context, reportID int64) ([]ReportMessage, error)
	GetReportMessageByID(ctx context.Context, reportMessageID int64) (ReportMessage, error)
	DropReportMessage(ctx context.Context, message *ReportMessage) error
	DropReport(ctx context.Context, report *Report) error
	SaveReport(ctx context.Context, report *Report) error
	SaveReportMessage(ctx context.Context, message *ReportMessage) error
}

type ReportUsecase interface {
	GetReportBySteamID(ctx context.Context, authorID steamid.SID64, steamID steamid.SID64) (Report, error)
	GetReports(ctx context.Context, user PersonInfo, opts ReportQueryFilter) ([]ReportWithAuthor, int64, error)
	GetReport(ctx context.Context, curUser PersonInfo, reportID int64) (ReportWithAuthor, error)
	GetReportMessages(ctx context.Context, reportID int64) ([]ReportMessage, error)
	GetReportMessageByID(ctx context.Context, reportMessageID int64) (ReportMessage, error)
	DropReportMessage(ctx context.Context, message *ReportMessage) error
	DropReport(ctx context.Context, report *Report) error
	SaveReport(ctx context.Context, report *Report) error
	SaveReportMessage(ctx context.Context, message *ReportMessage) error
}

type CreateReportReq struct {
	SourceID        StringSID `json:"source_id"`
	TargetID        StringSID `json:"target_id"`
	Description     string    `json:"description"`
	Reason          Reason    `json:"reason"`
	ReasonText      string    `json:"reason_text"`
	DemoName        string    `json:"demo_name"`
	DemoTick        int       `json:"demo_tick"`
	PersonMessageID int64     `json:"person_message_id"`
}

type ReportStatus int

const (
	Opened ReportStatus = iota
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
	ReportID     int64         `json:"report_id"`
	SourceID     steamid.SID64 `json:"source_id"`
	TargetID     steamid.SID64 `json:"target_id"`
	Description  string        `json:"description"`
	ReportStatus ReportStatus  `json:"report_status"`
	Reason       Reason        `json:"reason"`
	ReasonText   string        `json:"reason_text"`
	Deleted      bool          `json:"deleted"`
	// Note that we do not use a foreign key here since the demos are not sent until completion
	// and reports can happen mid-game
	DemoName        string    `json:"demo_name"`
	DemoTick        int       `json:"demo_tick"`
	DemoID          int       `json:"demo_id"`
	PersonMessageID int64     `json:"person_message_id"`
	CreatedOn       time.Time `json:"created_on"`
	UpdatedOn       time.Time `json:"updated_on"`
}

func (report Report) Path() string {
	return fmt.Sprintf("/report/%d", report.ReportID)
}

func NewReport() Report {
	return Report{
		ReportID:     0,
		SourceID:     "",
		Description:  "",
		ReportStatus: 0,
		CreatedOn:    time.Now(),
		UpdatedOn:    time.Now(),
		DemoTick:     -1,
		DemoName:     "",
	}
}

type ReportWithAuthor struct {
	Author  Person `json:"author"`
	Subject Person `json:"subject"`
	Report
}

type ReportMessage struct {
	ReportID        int64         `json:"report_id"`
	ReportMessageID int64         `json:"report_message_id"`
	AuthorID        steamid.SID64 `json:"author_id"`
	MessageMD       string        `json:"message_md"`
	Deleted         bool          `json:"deleted"`
	CreatedOn       time.Time     `json:"created_on"`
	UpdatedOn       time.Time     `json:"updated_on"`
	SimplePerson
}

func NewReportMessage(reportID int64, authorID steamid.SID64, messageMD string) ReportMessage {
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
