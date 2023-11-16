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

func (db *Store) insertReport(ctx context.Context, report *Report) error {
	const query = `INSERT INTO report (
		    author_id, reported_id, report_status, description, deleted, created_on, updated_on, reason, 
            reason_text, demo_name, demo_tick, person_message_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING report_id`

	var msgID *int64
	if report.PersonMessageID > 0 {
		msgID = &report.PersonMessageID
	}

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
		msgID,
	).Scan(&report.ReportID); errQuery != nil {
		return Err(errQuery)
	}

	return nil
}

func (db *Store) updateReport(ctx context.Context, report *Report) error {
	report.UpdatedOn = time.Now()

	var msgID *int64
	if report.PersonMessageID > 0 {
		msgID = &report.PersonMessageID
	}

	return db.ExecUpdateBuilder(ctx, db.sb.Update("report").
		Set("author_id", report.SourceID).
		Set("reported_id", report.TargetID).
		Set("report_status", report.ReportStatus).
		Set("description", report.Description).
		Set("deleted", report.Deleted).
		Set("updated_on", report.UpdatedOn).
		Set("reason", report.Reason).
		Set("reason_text", report.ReasonText).
		Set("demo_name", report.DemoName).
		Set("demo_tick", report.DemoTick).
		Set("person_message_id", msgID).
		Where(sq.Eq{"report_id": report.ReportID}))
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
	message.UpdatedOn = time.Now()

	if errQuery := db.ExecUpdateBuilder(ctx, db.sb.Update("report_message").
		Set("deleted", message.Deleted).
		Set("author_id", message.AuthorID).
		Set("updated_on", message.UpdatedOn).
		Set("message_md", message.Contents).
		Where(sq.Eq{"report_message_id": message.MessageID})); errQuery != nil {
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
	report.Deleted = true

	if errExec := db.ExecUpdateBuilder(ctx, db.sb.
		Update("report").
		Set("deleted", report.Deleted).
		Where(sq.Eq{"report_id": report.ReportID})); errExec != nil {
		return Err(errExec)
	}

	db.log.Info("Report deleted", zap.Int64("report_id", report.ReportID))

	return nil
}

func (db *Store) DropReportMessage(ctx context.Context, message *UserMessage) error {
	message.Deleted = true

	if errExec := db.ExecUpdateBuilder(ctx, db.sb.
		Update("report_message").
		Set("deleted", message.Deleted).
		Where(sq.Eq{"report_message_id": message.MessageID})); errExec != nil {
		return Err(errExec)
	}

	db.log.Info("Report message deleted", zap.Int64("report_message_id", message.MessageID))

	return nil
}

type ReportQueryFilter struct {
	QueryFilter
	ReportStatus ReportStatus `json:"report_status"`
	SourceID     StringSID    `json:"source_id"`
	TargetID     StringSID    `json:"target_id"`
}

func (db *Store) GetReports(ctx context.Context, opts ReportQueryFilter) ([]Report, int64, error) {
	constraints := sq.And{sq.Eq{"deleted": opts.Deleted}}

	if opts.SourceID != "" {
		authorID, errAuthorID := opts.SourceID.SID64(ctx)
		if errAuthorID != nil {
			return nil, 0, errAuthorID
		}

		constraints = append(constraints, sq.Eq{"r.author_id": authorID.Int64()})
	}

	if opts.TargetID != "" {
		targetID, errTargetID := opts.TargetID.SID64(ctx)
		if errTargetID != nil {
			return nil, 0, errTargetID
		}

		constraints = append(constraints, sq.Eq{"r.reported_id": targetID.Int64()})
	}

	if opts.ReportStatus >= 0 {
		constraints = append(constraints, sq.Eq{"r.report_status": opts.ReportStatus})
	}

	counterQuery := db.sb.
		Select("count(r.report_id) as total").
		From("report r").
		Where(constraints)

	builder := db.sb.
		Select("r.report_id", "r.author_id", "r.reported_id", "r.report_status",
			"r.description", "r.deleted", "r.created_on", "r.updated_on", "r.reason", "r.reason_text",
			"r.demo_name", "r.demo_tick", "coalesce(d.demo_id, 0)", "r.person_message_id").
		From("report r").
		Where(constraints).
		LeftJoin("demo d on d.title = r.demo_name")

	builder = opts.applySafeOrder(builder, map[string][]string{
		"r.": {"report_id", "author_id", "reported_id", "report_status", "deleted", "created_on", "updated_on", "reason"},
	}, "report_id")

	builder = opts.applyLimitOffsetDefault(builder)

	count, errCount := db.GetCount(ctx, counterQuery)
	if errCount != nil {
		return nil, 0, Err(errCount)
	}

	rows, errQuery := db.QueryBuilder(ctx, builder)
	if errQuery != nil {
		return nil, 0, Err(errQuery)
	}

	defer rows.Close()

	var reports []Report

	for rows.Next() {
		var (
			report          Report
			sourceID        int64
			targetID        int64
			personMessageID *int64
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
			&personMessageID,
		); errScan != nil {
			return nil, 0, Err(errScan)
		}

		if personMessageID != nil {
			report.PersonMessageID = *personMessageID
		}

		report.SourceID = steamid.New(sourceID)
		report.TargetID = steamid.New(targetID)

		reports = append(reports, report)
	}

	return reports, count, nil
}

// GetReportBySteamID returns any open report for the user by the author.
func (db *Store) GetReportBySteamID(ctx context.Context, authorID steamid.SID64, steamID steamid.SID64, report *Report) error {
	row, errRow := db.QueryRowBuilder(ctx, db.sb.
		Select("r.report_id", "r.author_id", "r.reported_id", "r.report_status", "r.description",
			"r.deleted", "r.created_on", "r.updated_on", "r.reason", "r.reason_text", "r.demo_name", "r.demo_tick",
			"coalesce(d.demo_id, 0)", "coalesce(r.person_message_id, 0)").
		From("report r").
		LeftJoin("demo d on r.demo_name = d.title").
		Where(sq.And{
			sq.Eq{"r.deleted": false},
			sq.Eq{"r.reported_id": steamID},
			sq.LtOrEq{"r.report_status": NeedMoreInfo},
			sq.Eq{"r.author_id": authorID},
		}))

	if errRow != nil {
		return errRow
	}

	var (
		sourceID int64
		targetID int64
	)

	if errScan := row.Scan(
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
		&report.PersonMessageID,
	); errScan != nil {
		return Err(errScan)
	}

	report.SourceID = steamid.New(sourceID)
	report.TargetID = steamid.New(targetID)

	return nil
}

func (db *Store) GetReport(ctx context.Context, reportID int64, report *Report) error {
	row, errRow := db.QueryRowBuilder(ctx, db.sb.Select("r.report_id", "r.author_id", "r.reported_id", "r.report_status", "r.description",
		"r.deleted", "r.created_on", "r.updated_on", "r.reason", "r.reason_text", "r.demo_name", "r.demo_tick",
		"coalesce(d.demo_id, 0)", "coalesce(r.person_message_id, 0)").
		From("report r").LeftJoin("demo d on r.demo_name = d.title").
		Where(sq.And{sq.Eq{"deleted": false}, sq.Eq{"report_id": reportID}}))

	if errRow != nil {
		return errRow
	}

	var (
		sourceID int64
		targetID int64
	)

	if errScan := row.Scan(
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
		&report.PersonMessageID,
	); errScan != nil {
		return Err(errScan)
	}

	report.SourceID = steamid.New(sourceID)
	report.TargetID = steamid.New(targetID)

	return nil
}

func (db *Store) GetReportMessages(ctx context.Context, reportID int64) ([]UserMessage, error) {
	rows, errQuery := db.QueryBuilder(ctx, db.sb.
		Select("report_message_id", "report_id", "author_id", "message_md", "deleted", "created_on", "updated_on").
		From("report_message").
		Where(sq.And{sq.And{sq.Eq{"deleted": false}, sq.Eq{"report_id": reportID}}}).
		OrderBy("created_on"))
	if errQuery != nil {
		if errors.Is(Err(errQuery), ErrNoResult) {
			return []UserMessage{}, nil
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
	row, errRow := db.QueryRowBuilder(ctx, db.sb.
		Select("report_message_id", "report_id", "author_id", "message_md", "deleted", "created_on", "updated_on").
		From("report_message").Where(sq.Eq{"report_message_id": reportMessageID}))
	if errRow != nil {
		return errRow
	}

	var authorID int64

	if errScan := row.Scan(
		&message.MessageID,
		&message.ParentID,
		&authorID,
		&message.Contents,
		&message.Deleted,
		&message.CreatedOn,
		&message.UpdatedOn,
	); errScan != nil {
		return Err(errScan)
	}

	message.AuthorID = steamid.New(authorID)

	return nil
}
