package notification

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

type notificationRepository struct {
	db database.Database
}

func NewNotificationRepository(db database.Database) domain.NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) SendNotification(ctx context.Context, targetID steamid.SID64, severity domain.NotificationSeverity, message string, link string) error {
	return r.db.DBErr(r.db.ExecInsertBuilder(ctx, r.db.
		Builder().
		Insert("person_notification").
		Columns("steam_id", "severity", "message", "link", "created_on").
		Values(targetID.Int64(), severity, message, link, time.Now())))
}

func (r *notificationRepository) GetPersonNotifications(ctx context.Context, filters domain.NotificationQuery) ([]domain.UserNotification, int64, error) {
	builder := r.db.
		Builder().
		Select("r.person_notification_id", "r.steam_id", "r.read", "r.deleted", "r.severity",
			"r.message", "r.link", "r.count", "r.created_on").
		From("person_notification r").
		OrderBy("r.person_notification_id desc")

	constraints := sq.And{sq.Eq{"r.deleted": false}, sq.Eq{"r.steam_id": filters.SteamID}}

	builder = filters.ApplySafeOrder(builder, map[string][]string{
		"r.": {"person_notification_id", "steam_id", "read", "deleted", "severity", "message", "link", "count", "created_on"},
	}, "person_notification_id")

	builder = filters.ApplyLimitOffsetDefault(builder).Where(constraints)

	count, errCount := r.db.GetCount(ctx, r.db.
		Builder().
		Select("count(r.person_notification_id)").
		From("person_notification r").
		Where(constraints))
	if errCount != nil {
		return nil, 0, r.db.DBErr(errCount)
	}

	rows, errRows := r.db.QueryBuilder(ctx, builder.Where(constraints))
	if errRows != nil {
		return nil, 0, r.db.DBErr(errRows)
	}

	defer rows.Close()

	var notifications []domain.UserNotification

	for rows.Next() {
		var (
			notif      domain.UserNotification
			outSteamID int64
		)

		if errScan := rows.Scan(&notif.PersonNotificationID, &outSteamID, &notif.Read, &notif.Deleted,
			&notif.Severity, &notif.Message, &notif.Link, &notif.Count, &notif.CreatedOn); errScan != nil {
			return nil, 0, errors.Join(errScan, domain.ErrScanResult)
		}

		notif.SteamID = steamid.New(outSteamID)

		notifications = append(notifications, notif)
	}

	return notifications, count, nil
}
