package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

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
	DemoName  string    `json:"demo_name"`
	DemoTick  int       `json:"demo_tick"`
	DemoID    int       `json:"demo_id"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
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

func (db *Store) insertReport(ctx context.Context, report *Report) error {
	const query = `INSERT INTO report (
		    author_id, reported_id, report_status, description, deleted, created_on, updated_on, reason, 
            reason_text, demo_name, demo_tick
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING report_id`

	if errQuery := db.QueryRow(ctx, query,
		report.SourceID,
		report.TargetID,
		report.ReportStatus,
		report.Description,
		report.Deleted,
		report.CreatedOn,
		report.UpdatedOn,
		report.Reason,
		report.ReasonText,
		report.DemoName,
		report.DemoTick,
	).Scan(&report.ReportID); errQuery != nil {
		return Err(errQuery)
	}

	return nil
}

func (db *Store) updateReport(ctx context.Context, report *Report) error {
	const query = `
		UPDATE report 
		SET author_id = $1, reported_id = $2, report_status = $3, description = $4,
            deleted = $5, updated_on = $6, reason = $7, reason_text = $8, demo_name = $9, demo_tick = $10
        WHERE report_id = $11`

	report.UpdatedOn = time.Now()

	return Err(db.Exec(ctx, query, report.SourceID, report.TargetID, report.ReportStatus, report.Description,
		report.Deleted, report.UpdatedOn, report.Reason, report.ReasonText,
		report.DemoName, report.DemoTick, report.ReportID))
}

func (db *Store) SaveReport(ctx context.Context, report *Report) error {
	if report.ReportID > 0 {
		return db.updateReport(ctx, report)
	}

	return db.insertReport(ctx, report)
}

func (db *Store) SaveReportMessage(ctx context.Context, message *UserMessage) error {
	if message.MessageID > 0 {
		return db.updateReportMessage(ctx, message)
	}

	return db.insertReportMessage(ctx, message)
}

func (db *Store) updateReportMessage(ctx context.Context, message *UserMessage) error {
	const query = `
		UPDATE report_message 
		SET deleted = $2, author_id = $3, updated_on = $4, message_md = $5
		WHERE report_message_id = $1
	`

	message.UpdatedOn = time.Now()

	if errQuery := db.Exec(ctx, query,
		message.MessageID,
		message.Deleted,
		message.AuthorID,
		message.UpdatedOn,
		message.Contents,
	); errQuery != nil {
		return Err(errQuery)
	}

	db.log.Info("Report message updated",
		zap.Int64("report_id", message.ParentID),
		zap.Int64("message_id", message.MessageID),
		zap.Int64("author_id", message.AuthorID.Int64()))

	return nil
}

func (db *Store) insertReportMessage(ctx context.Context, message *UserMessage) error {
	const query = `
		INSERT INTO report_message (
		    report_id, author_id, message_md, deleted, created_on, updated_on
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING report_message_id
	`

	if errQuery := db.QueryRow(ctx, query,
		message.ParentID,
		message.AuthorID,
		message.Contents,
		message.Deleted,
		message.CreatedOn,
		message.UpdatedOn,
	).Scan(&message.MessageID); errQuery != nil {
		return Err(errQuery)
	}

	db.log.Info("Report message created",
		zap.Int64("report_id", message.ParentID),
		zap.Int64("message_id", message.MessageID),
		zap.Int64("author_id", message.AuthorID.Int64()))

	return nil
}

func (db *Store) DropReport(ctx context.Context, report *Report) error {
	const q = `UPDATE report SET deleted = true WHERE report_id = $1`

	if errExec := db.Exec(ctx, q, report.ReportID); errExec != nil {
		return Err(errExec)
	}

	report.Deleted = true

	db.log.Info("Report deleted", zap.Int64("report_id", report.ReportID))

	return nil
}

func (db *Store) DropReportMessage(ctx context.Context, message *UserMessage) error {
	const query = `UPDATE report_message SET deleted = true WHERE report_message_id = $1`

	if errExec := db.Exec(ctx, query, message.Contents); errExec != nil {
		return Err(errExec)
	}

	message.Deleted = true

	db.log.Info("Report message deleted", zap.Int64("report_message_id", message.MessageID))

	return nil
}

type AuthorQueryFilter struct {
	QueryFilter
	AuthorID steamid.SID64 `json:"author_id"`
}

type ReportQueryFilter struct {
	AuthorQueryFilter
	ReportStatus ReportStatus `json:"report_status"`
}

func (db *Store) GetReports(ctx context.Context, opts AuthorQueryFilter) ([]Report, error) {
	conditions := sq.And{sq.Eq{"deleted": opts.Deleted}}

	if opts.AuthorID.Valid() {
		conditions = append(conditions, sq.Eq{"author_id": opts.AuthorID})
	}

	builder := db.sb.
		Select("r.report_id", "r.author_id", "r.reported_id", "r.report_status",
			"r.description", "r.deleted", "r.created_on", "r.updated_on", "r.reason", "r.reason_text",
			"r.demo_name", "r.demo_tick", "coalesce(d.demo_id, 0)").
		From("report r").
		Where(conditions).
		LeftJoin("demo d on d.title = r.demo_name")

	if opts.Limit > 0 {
		builder = builder.Limit(opts.Limit)
	}
	// if opts.OrderBy != "" {
	//	if opts.Desc {
	//		builder = builder.OrderBy(fmt.Sprintf("%s DESC", opts.OrderBy))
	//	} else {
	//		builder = builder.OrderBy(fmt.Sprintf("%s ASC", opts.OrderBy))
	//	}
	//}
	q, a, errSQL := builder.ToSql()
	if errSQL != nil {
		return nil, Err(errSQL)
	}

	rows, errQuery := db.Query(ctx, q, a...)
	if errQuery != nil {
		return nil, Err(errQuery)
	}

	defer rows.Close()

	var reports []Report

	for rows.Next() {
		var (
			report   Report
			sourceID int64
			targetID int64
		)

		if errScan := rows.Scan(
			&report.ReportID,
			&sourceID,
			&targetID,
			&report.ReportStatus,
			&report.Description,
			&report.Deleted,
			&report.CreatedOn,
			&report.UpdatedOn,
			&report.Reason,
			&report.ReasonText,
			&report.DemoName,
			&report.DemoTick,
			&report.DemoID,
		); errScan != nil {
			return nil, Err(errScan)
		}

		report.SourceID = steamid.New(sourceID)
		report.TargetID = steamid.New(targetID)

		reports = append(reports, report)
	}

	return reports, nil
}

// GetReportBySteamID returns any open report for the user by the author.
func (db *Store) GetReportBySteamID(ctx context.Context, authorID steamid.SID64, steamID steamid.SID64, report *Report) error {
	const query = `
		SELECT 
		   r.report_id, r.author_id, r.reported_id, r.report_status, r.description, 
		   r.deleted, r.created_on, r.updated_on, r.reason, r.reason_text, r.demo_name, r.demo_tick, coalesce(d.demo_id, 0)
		FROM report r
		LEFT JOIN demo d on r.demo_name = d.title
		WHERE deleted = false AND reported_id = $1 AND report_status <= $2 AND author_id = $3`

	var (
		sourceID int64
		targetID int64
	)

	if errQuery := db.QueryRow(ctx, query, steamID, NeedMoreInfo, authorID).
		Scan(
			&report.ReportID,
			&sourceID,
			&targetID,
			&report.ReportStatus,
			&report.Description,
			&report.Deleted,
			&report.CreatedOn,
			&report.UpdatedOn,
			&report.Reason,
			&report.ReasonText,
			&report.DemoName,
			&report.DemoTick,
			&report.DemoID,
		); errQuery != nil {
		return Err(errQuery)
	}

	report.SourceID = steamid.New(sourceID)
	report.TargetID = steamid.New(targetID)

	return nil
}

func (db *Store) GetReport(ctx context.Context, reportID int64, report *Report) error {
	const query = `
		SELECT 
		   r.report_id, r.author_id, r.reported_id, r.report_status, r.description, 
		   r.deleted, r.created_on, r.updated_on, r.reason, r.reason_text, r.demo_name, r.demo_tick, 
		   coalesce(d.demo_id, 0)
		FROM report r
		LEFT JOIN demo d on r.demo_name = d.title
		WHERE deleted = false AND report_id = $1`

	var (
		sourceID int64
		targetID int64
	)

	if errQuery := db.QueryRow(ctx, query, reportID).
		Scan(
			&report.ReportID,
			&sourceID,
			&targetID,
			&report.ReportStatus,
			&report.Description,
			&report.Deleted,
			&report.CreatedOn,
			&report.UpdatedOn,
			&report.Reason,
			&report.ReasonText,
			&report.DemoName,
			&report.DemoTick,
			&report.DemoID,
		); errQuery != nil {
		return Err(errQuery)
	}

	report.SourceID = steamid.New(sourceID)
	report.TargetID = steamid.New(targetID)

	return nil
}

func (db *Store) GetReportMessages(ctx context.Context, reportID int64) ([]UserMessage, error) {
	const query = `
		SELECT 
		   report_message_id, report_id, author_id, message_md, deleted, created_on, updated_on
		FROM report_message
		WHERE deleted = false AND report_id = $1 
		ORDER BY created_on`

	rows, errQuery := db.Query(ctx, query, reportID)
	if errQuery != nil {
		if errors.Is(Err(errQuery), ErrNoResult) {
			return nil, nil
		}
	}

	defer rows.Close()

	var messages []UserMessage

	for rows.Next() {
		var (
			msg      UserMessage
			authorID int64
		)

		if errScan := rows.Scan(
			&msg.MessageID,
			&msg.ParentID,
			&authorID,
			&msg.Contents,
			&msg.Deleted,
			&msg.CreatedOn,
			&msg.UpdatedOn,
		); errScan != nil {
			return nil, Err(errQuery)
		}

		msg.AuthorID = steamid.New(authorID)

		messages = append(messages, msg)
	}

	return messages, nil
}

func (db *Store) GetReportMessageByID(ctx context.Context, reportMessageID int64, message *UserMessage) error {
	const query = `
		SELECT 
		   report_message_id, report_id, author_id, message_md, deleted, created_on, updated_on
		FROM report_message
		WHERE report_message_id = $1`

	var authorID int64

	if errQuery := db.QueryRow(ctx, query, reportMessageID).
		Scan(
			&message.MessageID,
			&message.ParentID,
			&authorID,
			&message.Contents,
			&message.Deleted,
			&message.CreatedOn,
			&message.UpdatedOn,
		); errQuery != nil {
		return Err(errQuery)
	}

	message.AuthorID = steamid.New(authorID)

	return nil
}
