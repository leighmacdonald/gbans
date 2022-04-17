package store

import (
	"context"
	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/model"
	log "github.com/sirupsen/logrus"
)

func (database *pgStore) SaveReport(ctx context.Context, report *model.Report) error {
	const q = `INSERT INTO report (
		    author_id, reported_id, report_status, title, description, deleted, created_on, updated_on
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8 )
		RETURNING report_id`
	if errQuery := database.conn.QueryRow(ctx, q,
		report.AuthorId,
		report.ReportedId,
		report.ReportStatus,
		report.Title,
		report.Description,
		report.Deleted,
		report.CreatedOn,
		report.UpdatedOn,
	).Scan(&report.ReportId); errQuery != nil {
		return Err(errQuery)
	}
	log.WithFields(log.Fields{"report_id": report.ReportId, "author_id": report.AuthorId}).
		Infof("Report saved")
	return nil
}

func (database *pgStore) SaveReportMedia(ctx context.Context, reportId int, media *model.ReportMedia) error {
	const q = `
		INSERT INTO report_media (
		    report_id, author_id, mime_type, contents, deleted, created_on, updated_on
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING report_media_id
	`
	if errQuery := database.conn.QueryRow(ctx, q,
		reportId,
		media.AuthorId,
		media.MimeType,
		media.Contents,
		media.Deleted,
		media.CreatedOn,
		media.UpdatedOn,
	).Scan(&media.ReportMediaId); errQuery != nil {
		return Err(errQuery)
	}
	media.ReportId = reportId
	log.WithFields(log.Fields{
		"report_id": reportId, "media_id": media.ReportMediaId, "author_id": media.AuthorId,
	}).Infof("Report media saved")
	return nil
}

func (database *pgStore) SaveReportMessage(ctx context.Context, reportId int, message *model.ReportMessage) error {
	const q = `
		INSERT INTO report_message (
		    report_id, author_id, message_md, deleted, created_on, updated_on
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING report_message_id
	`
	message.ReportId = reportId
	if errQuery := database.conn.QueryRow(ctx, q,
		message.ReportId,
		message.AuthorId,
		message.Message,
		message.Deleted,
		message.CreatedOn,
		message.UpdatedOn,
	).Scan(&message.ReportMessageId); errQuery != nil {
		return Err(errQuery)
	}
	log.WithFields(log.Fields{
		"report_id":  reportId,
		"message_id": message.ReportMessageId,
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

func (database *pgStore) DropReportMessage(ctx context.Context, message *model.ReportMessage) error {
	const q = `UPDATE report_message SET deleted = true WHERE report_message_id = $1`
	if _, errExec := database.conn.Exec(ctx, q, message.ReportMessageId); errExec != nil {
		return Err(errExec)
	}
	log.WithFields(log.Fields{
		"report_message_id": message.ReportMessageId,
		"soft":              true,
	}).Infof("Report deleted")
	message.Deleted = true
	return nil
}

func (database *pgStore) DropReportMedia(ctx context.Context, media *model.ReportMedia) error {
	const q = `UPDATE report_media SET deleted = true WHERE report_media_id = $1`
	if _, errExec := database.conn.Exec(ctx, q, media.ReportMediaId); errExec != nil {
		return Err(errExec)
	}
	log.WithFields(log.Fields{
		"report_media_id": media.ReportMediaId,
		"soft":            true,
	}).Infof("Report deleted")
	media.Deleted = true
	return nil
}

type AuthorQueryFilter struct {
	QueryFilter
	AuthorId int64 `json:"author_id,string"`
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
	qb := sb.
		Select("report_id", "author_id", "reported_id", "report_status",
			"title", "description", "deleted", "created_on", "updated_on").
		From("report").
		Where(conditions)
	if opts.Limit > 0 {
		qb = qb.Limit(uint64(opts.Limit))
	}
	//if opts.OrderBy != "" {
	//	if opts.SortDesc {
	//		qb = qb.OrderBy(fmt.Sprintf("%s DESC", opts.OrderBy))
	//	} else {
	//		qb = qb.OrderBy(fmt.Sprintf("%s ASC", opts.OrderBy))
	//	}
	//}
	q, a, errSql := qb.ToSql()
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
			&report.Title,
			&report.Description,
			&report.Deleted,
			&report.CreatedOn,
			&report.UpdatedOn,
		); errScan != nil {
			return nil, Err(errScan)
		}
		reports = append(reports, report)
	}
	return reports, nil
}

func (database *pgStore) GetReport(ctx context.Context, reportId int, report *model.Report) error {
	const q = `
		SELECT 
		   report_id, author_id, reported_id, report_status, title, description, 
		   deleted, created_on, updated_on 
		FROM report
		WHERE deleted = false AND report_id = $1`
	if errQuery := database.conn.QueryRow(ctx, q, reportId).Scan(
		&report.ReportId,
		&report.AuthorId,
		&report.ReportedId,
		&report.ReportStatus,
		&report.Title,
		&report.Description,
		&report.Deleted,
		&report.CreatedOn,
		&report.UpdatedOn,
	); errQuery != nil {
		return Err(errQuery)
	}
	const q2 = `SELECT report_media_id FROM report_media WHERE deleted = false AND report_id = $1`
	mediaIds, errQueryMedia := database.conn.Query(ctx, q2, reportId)
	if errQueryMedia != nil && Err(errQueryMedia) != ErrNoResult {
		return Err(errQueryMedia)
	}
	defer mediaIds.Close()
	for mediaIds.Next() {
		var mediaId int
		if err := mediaIds.Scan(&mediaId); err != nil {
			return Err(err)
		}
		report.MediaIds = append(report.MediaIds, mediaId)
	}

	return nil
}

func (database *pgStore) GetReportMediaById(ctx context.Context, reportId int, media *model.ReportMedia) error {
	const q = `
		SELECT 
		   report_media_id, report_id, author_id, mime_type, contents, deleted, created_on, updated_on
		FROM report_media
		WHERE deleted = false AND report_media_id = $1`
	if errQuery := database.conn.QueryRow(ctx, q, reportId).Scan(
		&media.ReportMediaId,
		&media.ReportId,
		&media.AuthorId,
		&media.MimeType,
		&media.Contents,
		&media.Deleted,
		&media.CreatedOn,
		&media.UpdatedOn,
	); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func (database *pgStore) GetReportMessages(ctx context.Context, reportId int) ([]model.ReportMessage, error) {
	const q = `
		SELECT 
		   report_message_id, report_id, author_id, message_md, deleted, created_on, updated_on
		FROM report_message
		WHERE deleted = false AND report_id = $1 
		ORDER BY created_on`
	rows, errQuery := database.conn.Query(ctx, q, reportId)
	if errQuery != nil {
		if Err(errQuery) == ErrNoResult {
			return nil, nil
		}
	}
	defer rows.Close()
	var messages []model.ReportMessage
	for rows.Next() {
		var msg model.ReportMessage
		if errScan := rows.Scan(
			&msg.ReportMessageId,
			&msg.ReportId,
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
