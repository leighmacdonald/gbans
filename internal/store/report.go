package store

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (db *Store) insertReport(ctx context.Context, report *model.Report) error {
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

func (db *Store) updateReport(ctx context.Context, report *model.Report) error {
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

func (db *Store) SaveReport(ctx context.Context, report *model.Report) error {
	if report.ReportID > 0 {
		return db.updateReport(ctx, report)
	}

	return db.insertReport(ctx, report)
}

func (db *Store) SaveReportMessage(ctx context.Context, message *model.ReportMessage) error {
	if message.ReportMessageID > 0 {
		return db.updateReportMessage(ctx, message)
	}

	return db.insertReportMessage(ctx, message)
}

func (db *Store) updateReportMessage(ctx context.Context, message *model.ReportMessage) error {
	message.UpdatedOn = time.Now()

	if errQuery := db.ExecUpdateBuilder(ctx, db.sb.Update("report_message").
		Set("deleted", message.Deleted).
		Set("author_id", message.AuthorID).
		Set("updated_on", message.UpdatedOn).
		Set("message_md", message.MessageMD).
		Where(sq.Eq{"report_message_id": message.ReportMessageID})); errQuery != nil {
		return Err(errQuery)
	}

	db.log.Info("Report message updated",
		zap.Int64("report_id", message.ReportID),
		zap.Int64("message_id", message.ReportMessageID),
		zap.Int64("author_id", message.AuthorID.Int64()))

	return nil
}

func (db *Store) insertReportMessage(ctx context.Context, message *model.ReportMessage) error {
	const query = `
		INSERT INTO report_message (
		    report_id, author_id, message_md, deleted, created_on, updated_on
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING report_message_id
	`

	if errQuery := db.QueryRow(ctx, query,
		message.ReportID,
		message.AuthorID,
		message.MessageMD,
		message.Deleted,
		message.CreatedOn,
		message.UpdatedOn,
	).Scan(&message.ReportMessageID); errQuery != nil {
		return Err(errQuery)
	}

	db.log.Info("Report message created",
		zap.Int64("report_id", message.ReportID),
		zap.Int64("message_id", message.ReportMessageID),
		zap.Int64("author_id", message.AuthorID.Int64()))

	return nil
}

func (db *Store) DropReport(ctx context.Context, report *model.Report) error {
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

func (db *Store) DropReportMessage(ctx context.Context, message *model.ReportMessage) error {
	message.Deleted = true

	if errExec := db.ExecUpdateBuilder(ctx, db.sb.
		Update("report_message").
		Set("deleted", message.Deleted).
		Where(sq.Eq{"report_message_id": message.ReportMessageID})); errExec != nil {
		return Err(errExec)
	}

	db.log.Info("Report message deleted", zap.Int64("report_message_id", message.ReportMessageID))

	return nil
}

type ReportQueryFilter struct {
	QueryFilter
	ReportStatus model.ReportStatus `json:"report_status"`
	SourceID     model.StringSID    `json:"source_id"`
	TargetID     model.StringSID    `json:"target_id"`
}

func (db *Store) GetReports(ctx context.Context, opts ReportQueryFilter) ([]model.Report, int64, error) {
	constraints := sq.And{sq.Eq{"deleted": opts.Deleted}}

	if opts.SourceID != "" {
		authorID, errAuthorID := opts.SourceID.SID64(ctx)
		if errAuthorID != nil {
			return nil, 0, errors.Wrap(errAuthorID, "Invalid source id")
		}

		constraints = append(constraints, sq.Eq{"r.author_id": authorID.Int64()})
	}

	if opts.TargetID != "" {
		targetID, errTargetID := opts.TargetID.SID64(ctx)
		if errTargetID != nil {
			return nil, 0, errors.Wrap(errTargetID, "Invalid target id")
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

	var reports []model.Report

	for rows.Next() {
		var (
			report          model.Report
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
func (db *Store) GetReportBySteamID(ctx context.Context, authorID steamid.SID64, steamID steamid.SID64, report *model.Report) error {
	row, errRow := db.QueryRowBuilder(ctx, db.sb.
		Select("r.report_id", "r.author_id", "r.reported_id", "r.report_status", "r.description",
			"r.deleted", "r.created_on", "r.updated_on", "r.reason", "r.reason_text", "r.demo_name", "r.demo_tick",
			"coalesce(d.demo_id, 0)", "coalesce(r.person_message_id, 0)").
		From("report r").
		LeftJoin("demo d on r.demo_name = d.title").
		Where(sq.And{
			sq.Eq{"r.deleted": false},
			sq.Eq{"r.reported_id": steamID},
			sq.LtOrEq{"r.report_status": model.NeedMoreInfo},
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

func (db *Store) GetReport(ctx context.Context, reportID int64, report *model.Report) error {
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

func (db *Store) GetReportMessages(ctx context.Context, reportID int64) ([]model.ReportMessage, error) {
	rows, errQuery := db.QueryBuilder(ctx, db.sb.
		Select("r.report_message_id", "r.report_id", "r.author_id", "r.message_md", "r.deleted",
			"r.created_on", "r.updated_on", "p.avatarhash", "p.personaname", "p.permission_level").
		From("report_message r").
		LeftJoin("person p ON r.author_id = p.steam_id").
		Where(sq.And{sq.And{sq.Eq{"r.deleted": false}, sq.Eq{"r.report_id": reportID}}}).
		OrderBy("r.created_on"))
	if errQuery != nil {
		if errors.Is(Err(errQuery), ErrNoResult) {
			return []model.ReportMessage{}, nil
		}
	}

	defer rows.Close()

	var messages []model.ReportMessage

	for rows.Next() {
		var (
			msg      model.ReportMessage
			authorID int64
		)

		if errScan := rows.Scan(
			&msg.ReportMessageID,
			&msg.ReportID,
			&authorID,
			&msg.MessageMD,
			&msg.Deleted,
			&msg.CreatedOn,
			&msg.UpdatedOn,
			&msg.Avatarhash,
			&msg.Personaname,
			&msg.PermissionLevel,
		); errScan != nil {
			return nil, Err(errQuery)
		}

		msg.AuthorID = steamid.New(authorID)

		messages = append(messages, msg)
	}

	return messages, nil
}

func (db *Store) GetReportMessageByID(ctx context.Context, reportMessageID int64, message *model.ReportMessage) error {
	row, errRow := db.QueryRowBuilder(ctx, db.sb.
		Select("r.report_message_id", "r.report_id", "r.author_id", "r.message_md", "r.deleted",
			"r.created_on", "r.updated_on", "p.avatarhash", "p.personaname", "p.permission_level").
		From("report_message r").
		LeftJoin("person p ON r.author_id = p.steam_id").
		Where(sq.Eq{"r.report_message_id": reportMessageID}))
	if errRow != nil {
		return errRow
	}

	var authorID int64

	if errScan := row.Scan(
		&message.ReportMessageID,
		&message.ReportID,
		&authorID,
		&message.MessageMD,
		&message.Deleted,
		&message.CreatedOn,
		&message.UpdatedOn,
		&message.Avatarhash,
		&message.Personaname,
		&message.PermissionLevel,
	); errScan != nil {
		return Err(errScan)
	}

	message.AuthorID = steamid.New(authorID)

	return nil
}
