package ban

import (
	"fmt"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain/ban"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/person/permission"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type RequestMessageBodyMD struct {
	BodyMD string `json:"body_md"`
}

type RequestReportStatusUpdate struct {
	Status ReportStatus `json:"status"`
}

type RequestReportCreate struct {
	SourceID        steamid.SteamID `json:"source_id"`
	TargetID        steamid.SteamID `json:"target_id"`
	Description     string          `json:"description"`
	Reason          ban.Reason      `json:"reason"`
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
	Reason          ban.Reason      `json:"reason"`
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
	Author  person.Person `json:"author"`
	Subject person.Person `json:"subject"`
	// TODO FIX Demo    demo.DemoFile `json:"demo"`
	Report
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
