package store

import (
	"context"
	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	log "github.com/sirupsen/logrus"
)

func (database *pgStore) insertReport(ctx context.Context, report *model.Report) error {
	const query = `INSERT INTO report (
		    author_id, reported_id, report_status, description, deleted, created_on, updated_on, reason, reason_text
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING report_id`
	if errQuery := database.conn.QueryRow(ctx, query,
		report.AuthorId,
		report.ReportedId,
		report.ReportStatus,
		report.Description,
		report.Deleted,
		report.CreatedOn,
		report.UpdatedOn,
		report.Reason,
		report.ReasonText,
	).Scan(&report.ReportId); errQuery != nil {
		return Err(errQuery)
	}
	log.WithFields(log.Fields{
		"report_id": report.ReportId,
		"author_id": report.AuthorId}).
		Debugf("Report saved")
	return nil
}

func (database *pgStore) updateReport(ctx context.Context, report *model.Report) error {
	report.UpdatedOn = config.Now()
	const q = `
		UPDATE report 
		SET author_id = $1, reported_id = $2, report_status = $3, description = $4,
            deleted = $5, updated_on = $6, reason = $7, reason_text = $8
        WHERE report_id = $9`
	return Err(database.Exec(ctx, q, report.AuthorId, report.ReportedId, report.ReportStatus, report.Description,
		report.Deleted, report.UpdatedOn, report.Reason, report.ReasonText, report.ReportId))
}

func (database *pgStore) SaveReport(ctx context.Context, report *model.Report) error {
	if report.ReportId > 0 {
		return database.updateReport(ctx, report)
	}
	return database.insertReport(ctx, report)
}

func (database *pgStore) SaveReportMessage(ctx context.Context, message *model.UserMessage) error {
	if message.MessageId > 0 {
		return database.updateReportMessage(ctx, message)
	}
	return database.insertReportMessage(ctx, message)
}

func (database *pgStore) updateReportMessage(ctx context.Context, message *model.UserMessage) error {
	message.UpdatedOn = config.Now()
	const query = `
		UPDATE report_message 
		SET deleted = $2, author_id = $3, updated_on = $4, message_md = $5
		WHERE report_message_id = $1
	`
	if errQuery := database.Exec(ctx, query,
		message.MessageId,
		message.Deleted,
		message.AuthorId,
		message.UpdatedOn,
		message.Message,
	); errQuery != nil {
		return Err(errQuery)
	}
	log.WithFields(log.Fields{
		"report_id":  message.ParentId,
		"message_id": message.MessageId,
		"author_id":  message.AuthorId,
	}).Infof("Report message saved")
	return nil
}

func (database *pgStore) insertReportMessage(ctx context.Context, message *model.UserMessage) error {
	const query = `
		INSERT INTO report_message (
		    report_id, author_id, message_md, deleted, created_on, updated_on
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING report_message_id
	`
	if errQuery := database.conn.QueryRow(ctx, query,
		message.ParentId,
		message.AuthorId,
		message.Message,
		message.Deleted,
		message.CreatedOn,
		message.UpdatedOn,
	).Scan(&message.MessageId); errQuery != nil {
		return Err(errQuery)
	}
	log.WithFields(log.Fields{
		"report_id":  message.ParentId,
		"message_id": message.MessageId,
		"author_id":  message.AuthorId,
	}).Infof("Report message saved")
	return nil
}

func (database *pgStore) DropReport(ctx context.Context, report *model.Report) error {
	const q = `UPDATE report SET deleted = true WHERE report_id = $1`
	if _, errExec := database.conn.Exec(ctx, q, report.ReportId); errExec != nil {
		return Err(errExec)
	}
	log.WithFields(log.Fields{
		"report_id": report.ReportId,
		"soft":      true,
	}).Infof("Report deleted")
	report.Deleted = true
	return nil
}

func (database *pgStore) DropReportMessage(ctx context.Context, message *model.UserMessage) error {
	const q = `UPDATE report_message SET deleted = true WHERE report_message_id = $1`
	if _, errExec := database.conn.Exec(ctx, q, message.Message); errExec != nil {
		return Err(errExec)
	}
	log.WithFields(log.Fields{
		"report_message_id": message.MessageId,
		"soft":              true,
	}).Infof("Report deleted")
	message.Deleted = true
	return nil
}

type AuthorQueryFilter struct {
	QueryFilter
	AuthorId steamid.SID64 `json:"author_id"`
}

type ReportQueryFilter struct {
	AuthorQueryFilter
	ReportStatus model.ReportStatus `json:"report_status"`
}

func (database *pgStore) GetReports(ctx context.Context, opts AuthorQueryFilter) ([]model.Report, error) {
	var conditions sq.And
	conditions = append(conditions, sq.Eq{"deleted": opts.Deleted})
	if opts.AuthorId > 0 {
		conditions = append(conditions, sq.Eq{"author_id": opts.AuthorId})
	}
	builder := sb.
		Select("report_id", "author_id", "reported_id", "report_status",
			"description", "deleted", "created_on", "updated_on", "reason", "reason_text").
		From("report").
		Where(conditions)
	if opts.Limit > 0 {
		builder = builder.Limit(opts.Limit)
	}
	//if opts.OrderBy != "" {
	//	if opts.SortDesc {
	//		builder = builder.OrderBy(fmt.Sprintf("%s DESC", opts.OrderBy))
	//	} else {
	//		builder = builder.OrderBy(fmt.Sprintf("%s ASC", opts.OrderBy))
	//	}
	//}
	q, a, errSql := builder.ToSql()
	if errSql != nil {
		return nil, Err(errSql)
	}
	rows, errQuery := database.conn.Query(ctx, q, a...)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	var reports []model.Report
	for rows.Next() {
		var report model.Report
		if errScan := rows.Scan(
			&report.ReportId,
			&report.AuthorId,
			&report.ReportedId,
			&report.ReportStatus,
			&report.Description,
			&report.Deleted,
			&report.CreatedOn,
			&report.UpdatedOn,
			&report.Reason,
			&report.ReasonText,
		); errScan != nil {
			return nil, Err(errScan)
		}
		reports = append(reports, report)
	}
	return reports, nil
}

func (database *pgStore) GetReport(ctx context.Context, reportId int64, report *model.Report) error {
	const query = `
		SELECT 
		   report_id, author_id, reported_id, report_status, description, 
		   deleted, created_on, updated_on, reason, reason_text
		FROM report
		WHERE deleted = false AND report_id = $1`
	if errQuery := database.conn.QueryRow(ctx, query, reportId).Scan(
		&report.ReportId,
		&report.AuthorId,
		&report.ReportedId,
		&report.ReportStatus,
		&report.Description,
		&report.Deleted,
		&report.CreatedOn,
		&report.UpdatedOn,
		&report.Reason,
		&report.ReasonText,
	); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func (database *pgStore) GetReportMessages(ctx context.Context, reportId int64) ([]model.UserMessage, error) {
	const query = `
		SELECT 
		   report_message_id, report_id, author_id, message_md, deleted, created_on, updated_on
		FROM report_message
		WHERE deleted = false AND report_id = $1 
		ORDER BY created_on`
	rows, errQuery := database.conn.Query(ctx, query, reportId)
	if errQuery != nil {
		if Err(errQuery) == ErrNoResult {
			return nil, nil
		}
	}
	defer rows.Close()
	var messages []model.UserMessage
	for rows.Next() {
		var msg model.UserMessage
		if errScan := rows.Scan(
			&msg.MessageId,
			&msg.ParentId,
			&msg.AuthorId,
			&msg.Message,
			&msg.Deleted,
			&msg.CreatedOn,
			&msg.UpdatedOn,
		); errScan != nil {
			return nil, Err(errQuery)
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

func (database *pgStore) GetReportMessageById(ctx context.Context, reportMessageId int64, message *model.UserMessage) error {
	const query = `
		SELECT 
		   report_message_id, report_id, author_id, message_md, deleted, created_on, updated_on
		FROM report_message
		WHERE report_message_id = $1`
	if errQuery := database.conn.QueryRow(ctx, query, reportMessageId).Scan(
		&message.MessageId,
		&message.ParentId,
		&message.AuthorId,
		&message.Message,
		&message.Deleted,
		&message.CreatedOn,
		&message.UpdatedOn,
	); errQuery != nil {
		return Err(errQuery)
	}

	return nil
}
