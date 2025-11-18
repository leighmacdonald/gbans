package ban

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type AppealRepository struct {
	database.Database
}

func NewAppealRepository(database database.Database) AppealRepository {
	return AppealRepository{Database: database}
}

func (r *AppealRepository) ByActivity(ctx context.Context, opts AppealQueryFilter) ([]AppealOverview, error) {
	constraints := sq.And{sq.Gt{"m.count": 0}}

	if !opts.Deleted {
		constraints = append(constraints, sq.Eq{"b.deleted": false})
	}

	builder := r.Builder().
		Select("b.ban_id", "b.target_id", "b.source_id", "b.ban_type", "b.reason", "b.reason_text",
			"b.note", "b.valid_until", "b.origin", "b.created_on", "b.updated_on", "b.deleted",
			"CASE WHEN b.report_id IS NULL THEN 0 ELSE b.report_id END",
			"b.unban_reason_text", "b.is_enabled", "b.appeal_state",
			"source.steam_id as source_steam_id", "source.personaname as source_personaname",
			"source.avatarhash as source_avatar",
			"target.steam_id as target_steam_id", "target.personaname as target_personaname",
			"target.avatarhash as target_avatar").
		From("ban b").
		Where(constraints).
		InnerJoin(`
			LATERAL (
				SELECT count(a.ban_message_id) as count
				FROM ban_appeal a
				WHERE b.ban_id = a.ban_id
			) m ON TRUE`).
		LeftJoin("person source on source.steam_id = b.source_id").
		LeftJoin("person target on target.steam_id = b.target_id")

	rows, errQuery := r.QueryBuilder(ctx, builder)
	if errQuery != nil {
		return nil, database.DBErr(errQuery)
	}

	defer rows.Close()

	var overviews []AppealOverview

	for rows.Next() {
		var (
			overview      AppealOverview
			sourceID      int64
			SourceSteamID int64
			targetID      int64
			TargetSteamID int64
		)

		if errScan := rows.Scan(
			&overview.BanID, &targetID, &sourceID, &overview.BanType,
			&overview.Reason, &overview.ReasonText, &overview.Note, &overview.ValidUntil,
			&overview.Origin, &overview.CreatedOn, &overview.UpdatedOn, &overview.Deleted,
			&overview.ReportID, &overview.UnbanReasonText, &overview.IsEnabled, &overview.AppealState,
			&SourceSteamID, &overview.SourcePersonaname, &overview.SourceAvatarhash,
			&TargetSteamID, &overview.TargetPersonaname, &overview.TargetAvatarhash,
		); errScan != nil {
			return nil, errors.Join(errScan, database.ErrScanResult)
		}

		overview.SourceID = steamid.New(SourceSteamID)
		overview.TargetID = steamid.New(TargetSteamID)

		overviews = append(overviews, overview)
	}

	if overviews == nil {
		return []AppealOverview{}, nil
	}

	return overviews, nil
}

func (r *AppealRepository) SaveMessage(ctx context.Context, message *AppealMessage) error {
	var err error
	if message.BanMessageID > 0 {
		err = r.updateBanMessage(ctx, message)
	} else {
		err = r.insertBanMessage(ctx, message)
	}

	return err
}

func (r *AppealRepository) updateBanMessage(ctx context.Context, message *AppealMessage) error {
	message.UpdatedOn = time.Now()

	query := r.Builder().
		Update("ban_appeal").
		Set("deleted", message.Deleted).
		Set("author_id", message.AuthorID.Int64()).
		Set("updated_on", message.UpdatedOn).
		Set("message_md", message.MessageMD).
		Where(sq.Eq{"ban_message_id": message.BanMessageID})

	if errQuery := r.ExecUpdateBuilder(ctx, query); errQuery != nil {
		return database.DBErr(errQuery)
	}

	return nil
}

func (r *AppealRepository) insertBanMessage(ctx context.Context, message *AppealMessage) error {
	const query = `
	INSERT INTO ban_appeal (
		ban_id, author_id, message_md, deleted, created_on, updated_on
	)
	VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING ban_message_id
	`

	if errQuery := r.QueryRow(ctx, query,
		message.BanID,
		message.AuthorID.Int64(),
		message.MessageMD,
		message.Deleted,
		message.CreatedOn,
		message.UpdatedOn,
	).Scan(&message.BanMessageID); errQuery != nil {
		return database.DBErr(errQuery)
	}

	return nil
}

func (r *AppealRepository) Messages(ctx context.Context, banID int64) ([]AppealMessage, error) {
	query := r.Builder().
		Select("a.ban_message_id", "a.ban_id", "a.author_id", "a.message_md", "a.deleted",
			"a.created_on", "a.updated_on", "p.avatarhash", "p.personaname", "p.permission_level").
		From("ban_appeal a").
		LeftJoin("person p ON a.author_id = p.steam_id").
		Where(sq.And{sq.Eq{"a.deleted": false}, sq.Eq{"a.ban_id": banID}}).
		OrderBy("a.created_on")

	rows, errQuery := r.QueryBuilder(ctx, query)
	if errQuery != nil {
		if errors.Is(database.DBErr(errQuery), database.ErrNoResult) {
			return nil, nil
		}
	}

	defer rows.Close()

	var messages []AppealMessage

	for rows.Next() {
		var (
			msg      AppealMessage
			authorID int64
		)

		if errScan := rows.Scan(
			&msg.BanMessageID,
			&msg.BanID,
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

	if messages == nil {
		return []AppealMessage{}, nil
	}

	return messages, nil
}

func (r *AppealRepository) MessageByID(ctx context.Context, banMessageID int64) (AppealMessage, error) {
	query := r.Builder().
		Select("a.ban_message_id", "a.ban_id", "a.author_id", "a.message_md", "a.deleted", "a.created_on",
			"a.updated_on", "p.avatarhash", "p.personaname", "p.permission_level").
		From("ban_appeal a").
		LeftJoin("person p ON a.author_id = p.steam_id").
		Where(sq.Eq{"a.ban_message_id": banMessageID})

	var (
		authorID int64
		message  AppealMessage
	)

	row, errQuery := r.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return message, database.DBErr(errQuery)
	}

	if errScan := row.Scan(
		&message.BanMessageID,
		&message.BanID,
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

func (r *AppealRepository) DropMessage(ctx context.Context, message *AppealMessage) error {
	query := r.Builder().
		Update("ban_appeal").
		Set("deleted", true).
		Where(sq.Eq{"ban_message_id": message.BanMessageID})

	if errExec := r.ExecUpdateBuilder(ctx, query); errExec != nil {
		return database.DBErr(errExec)
	}

	message.Deleted = true

	return nil
}
