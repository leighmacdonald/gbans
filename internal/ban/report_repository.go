package ban

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type ReportRepository struct {
	db database.Database
}

func NewReportRepository(database database.Database) ReportRepository {
	return ReportRepository{db: database}
}

func (r ReportRepository) insertReport(ctx context.Context, report *Report) error {
	const query = `INSERT INTO report (
		    author_id, reported_id, report_status, description, deleted, created_on, updated_on, reason,
            reason_text, demo_id, demo_tick, person_message_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING report_id`

	var msgID *int64
	if report.PersonMessageID > 0 {
		msgID = &report.PersonMessageID
	}

	if errQuery := r.db.QueryRow(ctx, nil, query,
		report.SourceID,
		report.TargetID,
		report.ReportStatus,
		report.Description,
		report.Deleted,
		report.CreatedOn,
		report.UpdatedOn,
		report.Reason,
		report.ReasonText,
		report.DemoID,
		report.DemoTick,
		msgID,
	).
		Scan(&report.ReportID); errQuery != nil {
		return database.DBErr(errQuery)
	}

	return nil
}

func (r ReportRepository) updateReport(ctx context.Context, report *Report) error {
	report.UpdatedOn = time.Now()

	var msgID *int64
	if report.PersonMessageID > 0 {
		msgID = &report.PersonMessageID
	}

	return database.DBErr(r.db.ExecUpdateBuilder(ctx, nil, r.db.
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
		Set("demo_id", report.DemoID).
		Set("demo_tick", report.DemoTick).
		Set("person_message_id", msgID).
		Where(sq.Eq{"report_id": report.ReportID})))
}

func (r ReportRepository) SaveReport(ctx context.Context, report *Report) error {
	if report.ReportID > 0 {
		return r.updateReport(ctx, report)
	}

	return r.insertReport(ctx, report)
}

func (r ReportRepository) SaveReportMessage(ctx context.Context, message *ReportMessage) error {
	if message.ReportMessageID > 0 {
		return r.updateReportMessage(ctx, message)
	}

	return r.insertReportMessage(ctx, message)
}

func (r ReportRepository) updateReportMessage(ctx context.Context, message *ReportMessage) error {
	message.UpdatedOn = time.Now()

	if errQuery := r.db.ExecUpdateBuilder(ctx, nil, r.db.
		Builder().
		Update("report_message").
		Set("deleted", message.Deleted).
		Set("author_id", message.AuthorID).
		Set("updated_on", message.UpdatedOn).
		Set("message_md", message.MessageMD).
		Where(sq.Eq{"report_message_id": message.ReportMessageID})); errQuery != nil {
		return database.DBErr(errQuery)
	}

	return nil
}

func (r ReportRepository) insertReportMessage(ctx context.Context, message *ReportMessage) error {
	const query = `
		INSERT INTO report_message (
		    report_id, author_id, message_md, deleted, created_on, updated_on
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING report_message_id
	`

	if errQuery := r.db.QueryRow(ctx, nil, query,
		message.ReportID,
		message.AuthorID,
		message.MessageMD,
		message.Deleted,
		message.CreatedOn,
		message.UpdatedOn,
	).Scan(&message.ReportMessageID); errQuery != nil {
		return database.DBErr(errQuery)
	}

	return nil
}

func (r ReportRepository) DropReport(ctx context.Context, report *Report) error {
	report.Deleted = true

	if errExec := r.db.ExecUpdateBuilder(ctx, nil, r.db.
		Builder().
		Update("report").
		Set("deleted", report.Deleted).
		Where(sq.Eq{"report_id": report.ReportID})); errExec != nil {
		return database.DBErr(errExec)
	}

	return nil
}

func (r ReportRepository) DropReportMessage(ctx context.Context, message *ReportMessage) error {
	message.Deleted = true

	if errExec := r.db.ExecUpdateBuilder(ctx, nil, r.db.
		Builder().
		Update("report_message").
		Set("deleted", message.Deleted).
		Where(sq.Eq{"report_message_id": message.ReportMessageID})); errExec != nil {
		return database.DBErr(errExec)
	}

	return nil
}

func (r ReportRepository) GetReports(ctx context.Context, steamID steamid.SteamID) ([]Report, error) {
	constraints := sq.And{sq.Eq{"r.deleted": false}}
	if steamID.Valid() {
		constraints = append(constraints, sq.Eq{"r.author_id": steamID.Int64()})
	}

	builder := r.db.
		Builder().
		Select("r.report_id", "r.author_id", "r.reported_id", "r.report_status",
			"r.description", "r.deleted", "r.created_on", "r.updated_on", "r.reason", "r.reason_text",
			"coalesce(d.demo_id, 0)", "r.demo_tick", "r.person_message_id").
		From("report r").
		Where(constraints).
		LeftJoin("demo d on d.demo_id = r.demo_id")

	rows, errQuery := r.db.QueryBuilder(ctx, nil, builder)
	if errQuery != nil {
		return nil, database.DBErr(errQuery)
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
			&report.DemoID,
			&report.DemoTick,
			&personMessageID,
		); errScan != nil {
			return nil, database.DBErr(errScan)
		}

		if personMessageID != nil {
			report.PersonMessageID = *personMessageID
		}

		report.SourceID = steamid.New(sourceID)
		report.TargetID = steamid.New(targetID)

		reports = append(reports, report)
	}

	return reports, nil
}

// GetReportBySteamID returns any open report for the user by the author.
func (r ReportRepository) GetReportBySteamID(ctx context.Context, authorID steamid.SteamID, steamID steamid.SteamID) (Report, error) {
	var report Report

	row, errRow := r.db.QueryRowBuilder(ctx, nil, r.db.
		Builder().
		Select("s.report_id", "s.author_id", "s.reported_id", "s.report_status", "s.description",
			"s.deleted", "s.created_on", "s.updated_on", "s.reason", "s.reason_text", "s.demo_tick",
			"coalesce(d.demo_id, 0)", "coalesce(s.person_message_id, 0)").
		From("report s").
		LeftJoin("demo d on s.demo_id = d.demo_id").
		Where(sq.And{
			sq.Eq{"s.deleted": false},
			sq.Eq{"s.reported_id": steamID},
			sq.LtOrEq{"s.report_status": NeedMoreInfo},
			sq.Eq{"s.author_id": authorID},
		}))

	if errRow != nil {
		return report, database.DBErr(errRow)
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
		&report.DemoTick,
		&report.DemoID,
		&report.PersonMessageID,
	); errScan != nil {
		return report, database.DBErr(errScan)
	}

	report.SourceID = steamid.New(sourceID)
	report.TargetID = steamid.New(targetID)

	return report, nil
}

func (r ReportRepository) GetReport(ctx context.Context, reportID int64) (Report, error) {
	var report Report

	row, errRow := r.db.QueryRowBuilder(ctx, nil, r.db.
		Builder().
		Select("s.report_id", "s.author_id", "s.reported_id", "s.report_status", "s.description",
			"s.deleted", "s.created_on", "s.updated_on", "s.reason", "s.reason_text", "s.demo_tick",
			"coalesce(d.demo_id, 0)", "coalesce(s.person_message_id, 0)").
		From("report s").
		LeftJoin("demo d on s.demo_id = d.demo_id").
		Where(sq.And{sq.Eq{"deleted": false}, sq.Eq{"report_id": reportID}}))

	if errRow != nil {
		return report, database.DBErr(errRow)
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
		&report.DemoTick,
		&report.DemoID,
		&report.PersonMessageID,
	); errScan != nil {
		return report, database.DBErr(errScan)
	}

	report.SourceID = steamid.New(sourceID)
	report.TargetID = steamid.New(targetID)

	return report, nil
}

func (r ReportRepository) GetReportMessages(ctx context.Context, reportID int64) ([]ReportMessage, error) {
	rows, errQuery := r.db.QueryBuilder(ctx, nil, r.db.
		Builder().
		Select("s.report_message_id", "s.report_id", "s.author_id", "s.message_md", "s.deleted",
			"s.created_on", "s.updated_on", "p.avatarhash", "p.personaname", "p.permission_level").
		From("report_message s").
		LeftJoin("person p ON s.author_id = p.steam_id").
		Where(sq.And{sq.And{sq.Eq{"s.deleted": false}, sq.Eq{"s.report_id": reportID}}}).
		OrderBy("s.created_on"))
	if errQuery != nil {
		if errors.Is(database.DBErr(errQuery), database.ErrNoResult) {
			return []ReportMessage{}, nil
		}
	}

	defer rows.Close()

	var messages []ReportMessage

	for rows.Next() {
		var (
			msg      ReportMessage
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
			return nil, database.DBErr(errQuery)
		}

		msg.AuthorID = steamid.New(authorID)

		messages = append(messages, msg)
	}

	return messages, nil
}

func (r ReportRepository) GetReportMessageByID(ctx context.Context, reportMessageID int64) (ReportMessage, error) {
	var message ReportMessage

	row, errRow := r.db.QueryRowBuilder(ctx, nil, r.db.
		Builder().
		Select("s.report_message_id", "s.report_id", "s.author_id", "s.message_md", "s.deleted",
			"s.created_on", "s.updated_on", "p.avatarhash", "p.personaname", "p.permission_level").
		From("report_message s").
		LeftJoin("person p ON s.author_id = p.steam_id").
		Where(sq.Eq{"s.report_message_id": reportMessageID}))

	if errRow != nil {
		return message, errRow
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
		return message, database.DBErr(errScan)
	}

	message.AuthorID = steamid.New(authorID)

	return message, nil
}
