package notification

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type Repository struct {
	db database.Database
}

func NewRepository(db database.Database) Repository {
	return Repository{db: db}
}

func (r *Repository) SendSite(ctx context.Context, targetIDs steamid.Collection, severity Severity,
	message string, link string, authorID *int64,
) error {
	const query = `
		INSERT INTO person_notification (steam_id, severity, message, link, created_on, author_id)
		VALUES ($1, $2, $3, $4, $5, $6)`

	batch := &pgx.Batch{}
	for _, sid := range targetIDs {
		batch.Queue(query, sid.Int64(), severity, message, link, time.Now(), authorID)
	}

	return database.DBErr(r.db.SendBatch(ctx, nil, batch).Close())
}

func (r *Repository) MarkMessagesRead(ctx context.Context, steamID steamid.SteamID, ids []int) error {
	return database.DBErr(r.db.ExecUpdateBuilder(ctx, nil, r.db.Builder().
		Update("person_notification").
		Set("read", true).
		Where(sq.And{sq.Eq{"steam_id": steamID.Int64()}, sq.Eq{"person_notification_id": ids}})))
}

func (r *Repository) MarkAllRead(ctx context.Context, steamID steamid.SteamID) error {
	return database.DBErr(r.db.ExecUpdateBuilder(ctx, nil, r.db.Builder().
		Update("person_notification").
		Set("read", true).
		Where(sq.Eq{"steam_id": steamID.Int64()})))
}

func (r *Repository) DeleteMessages(ctx context.Context, steamID steamid.SteamID, ids []int) error {
	return database.DBErr(r.db.ExecUpdateBuilder(ctx, nil, r.db.Builder().
		Update("person_notification").
		Set("deleted", true).
		Where(sq.And{sq.Eq{"steam_id": steamID.Int64()}, sq.Eq{"person_notification_id": ids}})))
}

func (r *Repository) DeleteAll(ctx context.Context, steamID steamid.SteamID) error {
	return database.DBErr(r.db.ExecUpdateBuilder(ctx, nil, r.db.Builder().
		Update("person_notification").
		Set("deleted", true).
		Where(sq.Eq{"steam_id": steamID.Int64()})))
}

func (r *Repository) GetPersonNotifications(ctx context.Context, steamID steamid.SteamID) ([]UserNotification, error) {
	builder := r.db.
		Builder().
		Select("r.person_notification_id", "r.steam_id", "r.read", "r.deleted", "r.severity",
			"r.message", "r.link", "r.count", "r.created_on", "r.author_id",
			"p.personaname", "p.permission_level", "p.discord_id", "p.avatarhash", "p.created_on", "p.updated_on").
		From("person_notification r").
		LeftJoin("person p on r.author_id = p.steam_id").
		OrderBy("r.person_notification_id desc")

	constraints := sq.And{sq.Eq{"r.deleted": false}, sq.Eq{"r.steam_id": steamID.Int64()}}

	rows, errRows := r.db.QueryBuilder(ctx, nil, builder.Where(constraints))
	if errRows != nil {
		return nil, database.DBErr(errRows)
	}

	defer rows.Close()

	notifications := []UserNotification{}

	for rows.Next() {
		var (
			notif      UserNotification
			name       *string
			pLevel     *permission.Privilege
			authorID   *int64
			discordID  *string
			avatarHash *string
			createdOn  *time.Time
			updatedOn  *time.Time
			outSteamID int64
		)

		if errScan := rows.Scan(&notif.PersonNotificationID, &outSteamID, &notif.Read, &notif.Deleted,
			&notif.Severity, &notif.Message, &notif.Link, &notif.Count, &notif.CreatedOn,
			&authorID, &name, &pLevel, &discordID, &avatarHash, &createdOn, &updatedOn); errScan != nil {
			return nil, errors.Join(errScan, domain.ErrScanResult)
		}

		notif.SteamID = steamid.New(outSteamID)

		if authorID != nil {
			notif.Author = Author{
				SteamID:         steamid.New(*authorID),
				PermissionLevel: *pLevel,
				Name:            *name,
				Avatarhash:      *avatarHash,
			}
		}

		notifications = append(notifications, notif)
	}

	return notifications, nil
}
