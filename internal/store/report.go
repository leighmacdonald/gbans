package store

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

func (s Stores) insertReport(ctx context.Context, report *model.Report) error {
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

	if errQuery := s.QueryRow(ctx, query,
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
	).
		Scan(&report.ReportID); errQuery != nil {
		return errs.DBErr(errQuery)
	}

	return nil
}

func (s Stores) updateReport(ctx context.Context, report *model.Report) error {
	report.UpdatedOn = time.Now()

	var msgID *int64
	if report.PersonMessageID > 0 {
		msgID = &report.PersonMessageID
	}

	return errs.DBErr(s.ExecUpdateBuilder(ctx, s.
		Builder().
		Update("report").
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
		Where(sq.Eq{"report_id": report.ReportID})))
}

func (s Stores) SaveReport(ctx context.Context, report *model.Report) error {
	if report.ReportID > 0 {
		return s.updateReport(ctx, report)
	}

	return s.insertReport(ctx, report)
}

func (s Stores) SaveReportMessage(ctx context.Context, message *model.ReportMessage) error {
	if message.ReportMessageID > 0 {
		return s.updateReportMessage(ctx, message)
	}

	return s.insertReportMessage(ctx, message)
}

func (s Stores) updateReportMessage(ctx context.Context, message *model.ReportMessage) error {
	message.UpdatedOn = time.Now()

	if errQuery := s.ExecUpdateBuilder(ctx, s.
		Builder().
		Update("report_message").
		Set("deleted", message.Deleted).
		Set("author_id", message.AuthorID).
		Set("updated_on", message.UpdatedOn).
		Set("message_md", message.MessageMD).
		Where(sq.Eq{"report_message_id": message.ReportMessageID})); errQuery != nil {
		return errs.DBErr(errQuery)
	}

	return nil
}

func (s Stores) insertReportMessage(ctx context.Context, message *model.ReportMessage) error {
	const query = `
		INSERT INTO report_message (
		    report_id, author_id, message_md, deleted, created_on, updated_on
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING report_message_id
	`

	if errQuery := s.QueryRow(ctx, query,
		message.ReportID,
		message.AuthorID,
		message.MessageMD,
		message.Deleted,
		message.CreatedOn,
		message.UpdatedOn,
	).Scan(&message.ReportMessageID); errQuery != nil {
		return errs.DBErr(errQuery)
	}

	return nil
}

func (s Stores) DropReport(ctx context.Context, report *model.Report) error {
	report.Deleted = true

	if errExec := s.ExecUpdateBuilder(ctx, s.
		Builder().
		Update("report").
		Set("deleted", report.Deleted).
		Where(sq.Eq{"report_id": report.ReportID})); errExec != nil {
		return errs.DBErr(errExec)
	}

	return nil
}

func (s Stores) DropReportMessage(ctx context.Context, message *model.ReportMessage) error {
	message.Deleted = true

	if errExec := s.ExecUpdateBuilder(ctx, s.
		Builder().
		Update("report_message").
		Set("deleted", message.Deleted).
		Where(sq.Eq{"report_message_id": message.ReportMessageID})); errExec != nil {
		return errs.DBErr(errExec)
	}

	return nil
}

func (s Stores) GetReports(ctx context.Context, opts model.ReportQueryFilter) ([]model.Report, int64, error) {
	constraints := sq.And{sq.Eq{"deleted": opts.Deleted}}

	if opts.SourceID != "" {
		authorID, errAuthorID := opts.SourceID.SID64(ctx)
		if errAuthorID != nil {
			return nil, 0, errors.Join(errAuthorID, errs.ErrSourceID)
		}

		constraints = append(constraints, sq.Eq{"s.author_id": authorID.Int64()})
	}

	if opts.TargetID != "" {
		targetID, errTargetID := opts.TargetID.SID64(ctx)
		if errTargetID != nil {
			return nil, 0, errors.Join(errTargetID, errs.ErrTargetID)
		}

		constraints = append(constraints, sq.Eq{"s.reported_id": targetID.Int64()})
	}

	if opts.ReportStatus >= 0 {
		constraints = append(constraints, sq.Eq{"s.report_status": opts.ReportStatus})
	}

	counterQuery := s.
		Builder().
		Select("count(s.report_id) as total").
		From("report s").
		Where(constraints)

	builder := s.
		Builder().
		Select("s.report_id", "s.author_id", "s.reported_id", "s.report_status",
			"s.description", "s.deleted", "s.created_on", "s.updated_on", "s.reason", "s.reason_text",
			"s.demo_name", "s.demo_tick", "coalesce(d.demo_id, 0)", "s.person_message_id").
		From("report s").
		Where(constraints).
		LeftJoin("demo d on d.title = s.demo_name")

	builder = opts.ApplySafeOrder(builder, map[string][]string{
		"s.": {"report_id", "author_id", "reported_id", "report_status", "deleted", "created_on", "updated_on", "reason"},
	}, "report_id")

	builder = opts.ApplyLimitOffsetDefault(builder)

	count, errCount := getCount(ctx, s, counterQuery)
	if errCount != nil {
		return nil, 0, errs.DBErr(errCount)
	}

	rows, errQuery := s.QueryBuilder(ctx, builder)
	if errQuery != nil {
		return nil, 0, errs.DBErr(errQuery)
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
			return nil, 0, errs.DBErr(errScan)
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
func (s Stores) GetReportBySteamID(ctx context.Context, authorID steamid.SID64, steamID steamid.SID64, report *model.Report) error {
	row, errRow := s.QueryRowBuilder(ctx, s.
		Builder().
		Select("s.report_id", "s.author_id", "s.reported_id", "s.report_status", "s.description",
			"s.deleted", "s.created_on", "s.updated_on", "s.reason", "s.reason_text", "s.demo_name", "s.demo_tick",
			"coalesce(d.demo_id, 0)", "coalesce(s.person_message_id, 0)").
		From("report s").
		LeftJoin("demo d on s.demo_name = d.title").
		Where(sq.And{
			sq.Eq{"s.deleted": false},
			sq.Eq{"s.reported_id": steamID},
			sq.LtOrEq{"s.report_status": model.NeedMoreInfo},
			sq.Eq{"s.author_id": authorID},
		}))

	if errRow != nil {
		return errs.DBErr(errRow)
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
		return errs.DBErr(errScan)
	}

	report.SourceID = steamid.New(sourceID)
	report.TargetID = steamid.New(targetID)

	return nil
}

func (s Stores) GetReport(ctx context.Context, reportID int64, report *model.Report) error {
	row, errRow := s.QueryRowBuilder(ctx, s.
		Builder().
		Select("s.report_id", "s.author_id", "s.reported_id", "s.report_status", "s.description",
			"s.deleted", "s.created_on", "s.updated_on", "s.reason", "s.reason_text", "s.demo_name", "s.demo_tick",
			"coalesce(d.demo_id, 0)", "coalesce(s.person_message_id, 0)").
		From("report s").
		LeftJoin("demo d on s.demo_name = d.title").
		Where(sq.And{sq.Eq{"deleted": false}, sq.Eq{"report_id": reportID}}))

	if errRow != nil {
		return errs.DBErr(errRow)
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
		return errs.DBErr(errScan)
	}

	report.SourceID = steamid.New(sourceID)
	report.TargetID = steamid.New(targetID)

	return nil
}

func (s Stores) GetReportMessages(ctx context.Context, reportID int64) ([]model.ReportMessage, error) {
	rows, errQuery := s.QueryBuilder(ctx, s.
		Builder().
		Select("s.report_message_id", "s.report_id", "s.author_id", "s.message_md", "s.deleted",
			"s.created_on", "s.updated_on", "p.avatarhash", "p.personaname", "p.permission_level").
		From("report_message s").
		LeftJoin("person p ON s.author_id = p.steam_id").
		Where(sq.And{sq.And{sq.Eq{"s.deleted": false}, sq.Eq{"s.report_id": reportID}}}).
		OrderBy("s.created_on"))
	if errQuery != nil {
		if errors.Is(errs.DBErr(errQuery), errs.ErrNoResult) {
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
			return nil, errs.DBErr(errQuery)
		}

		msg.AuthorID = steamid.New(authorID)

		messages = append(messages, msg)
	}

	return messages, nil
}

func (s Stores) GetReportMessageByID(ctx context.Context, reportMessageID int64, message *model.ReportMessage) error {
	row, errRow := s.QueryRowBuilder(ctx, s.
		Builder().
		Select("s.report_message_id", "s.report_id", "s.author_id", "s.message_md", "s.deleted",
			"s.created_on", "s.updated_on", "p.avatarhash", "p.personaname", "p.permission_level").
		From("report_message s").
		LeftJoin("person p ON s.author_id = p.steam_id").
		Where(sq.Eq{"s.report_message_id": reportMessageID}))
	if errRow != nil {
		return errs.DBErr(errRow)
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
		return errs.DBErr(errScan)
	}

	message.AuthorID = steamid.New(authorID)

	return nil
}
