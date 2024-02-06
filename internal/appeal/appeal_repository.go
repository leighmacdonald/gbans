package appeal

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

type appealRepository struct {
	db database.Database
}

func NewAppealRepository(database database.Database) domain.AppealRepository {
	return &appealRepository{db: database}
}

func (r *appealRepository) GetAppealsByActivity(ctx context.Context, opts domain.AppealQueryFilter) ([]domain.AppealOverview, int64, error) {
	constraints := sq.And{sq.Gt{"m.count": 0}}

	if !opts.Deleted {
		constraints = append(constraints, sq.Eq{"s.deleted": opts.Deleted})
	}

	if opts.AppealState > domain.AnyState {
		constraints = append(constraints, sq.Eq{"s.appeal_state": opts.AppealState})
	}

	if opts.SourceID != "" {
		authorID, errAuthorID := opts.SourceID.SID64(ctx)
		if errAuthorID != nil {
			return nil, 0, errors.Join(errAuthorID, domain.ErrSourceID)
		}

		constraints = append(constraints, sq.Eq{"s.source_id": authorID.Int64()})
	}

	if opts.TargetID != "" {
		targetID, errTargetID := opts.TargetID.SID64(ctx)
		if errTargetID != nil {
			return nil, 0, errors.Join(errTargetID, domain.ErrTargetID)
		}

		constraints = append(constraints, sq.Eq{"s.target_id": targetID.Int64()})
	}

	counterQuery := r.db.
		Builder().
		Select("COUNT(s.ban_id)").
		From("ban s").
		Where(constraints).
		InnerJoin(`
			LATERAL (
				SELECT count(a.ban_message_id) as count 
				FROM ban_appeal a
				WHERE s.ban_id = a.ban_id
			) m ON TRUE`)

	count, errCount := r.db.GetCount(ctx, counterQuery)
	if errCount != nil {
		return nil, 0, r.db.DBErr(errCount)
	}

	builder := r.db.
		Builder().
		Select("s.ban_id", "s.target_id", "s.source_id", "s.ban_type", "s.reason", "s.reason_text",
			"s.note", "s.valid_until", "s.origin", "s.created_on", "s.updated_on", "s.deleted",
			"CASE WHEN s.report_id IS NULL THEN 0 ELSE report_id END",
			"s.unban_reason_text", "s.is_enabled", "s.appeal_state",
			"source.steam_id as source_steam_id", "source.personaname as source_personaname",
			"source.avatarhash as source_avatar",
			"target.steam_id as target_steam_id", "target.personaname as target_personaname",
			"target.avatarhash as target_avatar").
		From("ban s").
		Where(constraints).
		InnerJoin(`
			LATERAL (
				SELECT count(a.ban_message_id) as count 
				FROM ban_appeal a
				WHERE s.ban_id = a.ban_id
			) m ON TRUE`).
		LeftJoin("person source on source.steam_id = s.source_id").
		LeftJoin("person target on target.steam_id = s.target_id")

	builder = opts.QueryFilter.ApplySafeOrder(builder, map[string][]string{
		"s.": {
			"ban_id", "target_id", "source_id", "ban_type", "reason", "valid_until", "origin", "created_on",
			"updated_on", "deleted", "is_enabled", "appeal_state",
		},
	}, "updated_on")

	builder = opts.QueryFilter.ApplyLimitOffsetDefault(builder)

	rows, errQuery := r.db.QueryBuilder(ctx, builder)
	if errQuery != nil {
		return nil, 0, r.db.DBErr(errQuery)
	}

	defer rows.Close()

	var overviews []domain.AppealOverview

	for rows.Next() {
		var (
			overview      domain.AppealOverview
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
			return nil, 0, errors.Join(errScan, domain.ErrScanResult)
		}

		overview.SourceID = steamid.New(SourceSteamID)
		overview.TargetID = steamid.New(TargetSteamID)

		overviews = append(overviews, overview)
	}

	if overviews == nil {
		return []domain.AppealOverview{}, 0, nil
	}

	return overviews, count, nil
}

func (r *appealRepository) SaveBanMessage(ctx context.Context, message *domain.BanAppealMessage) error {
	var err error
	if message.BanMessageID > 0 {
		err = r.updateBanMessage(ctx, message)
	} else {
		err = r.insertBanMessage(ctx, message)
	}

	return err
}

func (r *appealRepository) updateBanMessage(ctx context.Context, message *domain.BanAppealMessage) error {
	message.UpdatedOn = time.Now()

	query := r.db.
		Builder().
		Update("ban_appeal").
		Set("deleted", message.Deleted).
		Set("author_id", message.AuthorID.Int64()).
		Set("updated_on", message.UpdatedOn).
		Set("message_md", message.MessageMD).
		Where(sq.Eq{"ban_message_id": message.BanMessageID})

	if errQuery := r.db.ExecUpdateBuilder(ctx, query); errQuery != nil {
		return r.db.DBErr(errQuery)
	}

	return nil
}

func (r *appealRepository) insertBanMessage(ctx context.Context, message *domain.BanAppealMessage) error {
	const query = `
	INSERT INTO ban_appeal (
		ban_id, author_id, message_md, deleted, created_on, updated_on
	)
	VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING ban_message_id
	`

	if errQuery := r.db.QueryRow(ctx, query,
		message.BanID,
		message.AuthorID.Int64(),
		message.MessageMD,
		message.Deleted,
		message.CreatedOn,
		message.UpdatedOn,
	).Scan(&message.BanMessageID); errQuery != nil {
		return r.db.DBErr(errQuery)
	}

	return nil
}

func (r *appealRepository) GetBanMessages(ctx context.Context, banID int64) ([]domain.BanAppealMessage, error) {
	query := r.db.
		Builder().
		Select("a.ban_message_id", "a.ban_id", "a.author_id", "a.message_md", "a.deleted",
			"a.created_on", "a.updated_on", "p.avatarhash", "p.personaname", "p.permission_level").
		From("ban_appeal a").
		LeftJoin("person p ON a.author_id = p.steam_id").
		Where(sq.And{sq.Eq{"a.deleted": false}, sq.Eq{"a.ban_id": banID}}).
		OrderBy("a.created_on")

	rows, errQuery := r.db.QueryBuilder(ctx, query)
	if errQuery != nil {
		if errors.Is(r.db.DBErr(errQuery), domain.ErrNoResult) {
			return nil, nil
		}
	}

	defer rows.Close()

	var messages []domain.BanAppealMessage

	for rows.Next() {
		var (
			msg      domain.BanAppealMessage
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
			return nil, r.db.DBErr(errQuery)
		}

		msg.AuthorID = steamid.New(authorID)

		messages = append(messages, msg)
	}

	if messages == nil {
		return []domain.BanAppealMessage{}, nil
	}

	return messages, nil
}

func (r *appealRepository) GetBanMessageByID(ctx context.Context, banMessageID int64, message *domain.BanAppealMessage) error {
	query := r.db.
		Builder().
		Select("a.ban_message_id", "a.ban_id", "a.author_id", "a.message_md", "a.deleted", "a.created_on",
			"a.updated_on", "p.avatarhash", "p.personaname", "p.permission_level").
		From("ban_appeal a").
		LeftJoin("person p ON a.author_id = p.steam_id").
		Where(sq.Eq{"a.ban_message_id": banMessageID})

	var authorID int64

	row, errQuery := r.db.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return r.db.DBErr(errQuery)
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
		return r.db.DBErr(errScan)
	}

	message.AuthorID = steamid.New(authorID)

	return nil
}

func (r *appealRepository) DropBanMessage(ctx context.Context, message *domain.BanAppealMessage) error {
	query := r.db.
		Builder().
		Update("ban_appeal").
		Set("deleted", true).
		Where(sq.Eq{"ban_message_id": message.BanMessageID})

	if errExec := r.db.ExecUpdateBuilder(ctx, query); errExec != nil {
		return r.db.DBErr(errExec)
	}

	message.Deleted = true

	return nil
}
