package ban

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type banASNRepository struct {
	db database.Database
}

func NewBanASNRepository(database database.Database) domain.BanASNRepository {
	return &banASNRepository{
		db: database,
	}
}

func (r banASNRepository) Expired(ctx context.Context) ([]domain.BanASN, error) {
	query := r.db.
		Builder().
		Select("ban_asn_id", "as_num", "origin", "source_id", "target_id", "reason_text", "valid_until",
			"created_on", "updated_on", "deleted", "reason", "is_enabled", "unban_reason_text", "appeal_state").
		From("ban_asn").
		Where(sq.And{sq.Lt{"valid_until": time.Now()}, sq.Eq{"deleted": false}})

	var bans []domain.BanASN

	rows, errQuery := r.db.QueryBuilder(ctx, nil, query)
	if errQuery != nil {
		return nil, r.db.DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			banASN   domain.BanASN
			targetID int64
			sourceID int64
		)

		if errScan := rows.
			Scan(&banASN.BanASNId, &banASN.ASNum, &banASN.Origin, &sourceID, &targetID,
				&banASN.ReasonText, &banASN.ValidUntil, &banASN.CreatedOn, &banASN.UpdatedOn, &banASN.Deleted,
				&banASN.Reason, &banASN.IsEnabled, &banASN.UnbanReasonText, &banASN.AppealState); errScan != nil {
			return nil, errors.Join(errScan, domain.ErrScanASN)
		}

		banASN.TargetID = steamid.New(targetID)
		banASN.SourceID = steamid.New(sourceID)

		bans = append(bans, banASN)
	}

	if bans == nil {
		bans = []domain.BanASN{}
	}

	return bans, nil
}

func (r banASNRepository) GetByASN(ctx context.Context, asNum int64, banASN *domain.BanASN) error {
	const query = `
		SELECT ban_asn_id, as_num, origin, source_id, target_id, reason_text, valid_until, created_on, updated_on, 
		       deleted, reason, is_enabled, unban_reason_text, appeal_state, note
		FROM ban_asn 
		WHERE deleted = false AND as_num = $1`

	var (
		targetID int64
		sourceID int64
	)

	if errQuery := r.db.
		QueryRow(ctx, nil, query, asNum).
		Scan(&banASN.BanASNId, &banASN.ASNum, &banASN.Origin,
			&sourceID, &targetID, &banASN.ReasonText, &banASN.ValidUntil, &banASN.CreatedOn,
			&banASN.UpdatedOn, &banASN.Deleted, &banASN.Reason, &banASN.IsEnabled, &banASN.UnbanReasonText,
			&banASN.AppealState, &banASN.Note); errQuery != nil {
		return r.db.DBErr(errQuery)
	}

	banASN.TargetID = steamid.New(targetID)
	banASN.SourceID = steamid.New(sourceID)

	return nil
}

func (r banASNRepository) GetByID(ctx context.Context, banID int64) (domain.BannedASNPerson, error) {
	const query = `
		SELECT b.ban_asn_id, b.as_num, b.origin, b.source_id, b.target_id, b.reason_text, b.valid_until, b.created_on, b.updated_on, 
		       b.deleted, b.reason, b.is_enabled, b.unban_reason_text, b.appeal_state, b.note,
		       s.avatarhash, s.personaname,t.avatarhash, t.personaname
		FROM ban_asn b
		LEFT JOIN person s on s.steam_id = b.source_id
		LEFT JOIN person t on t.steam_id = b.source_id
		WHERE deleted = false AND ban_asn_id = $1`

	var (
		targetID int64
		sourceID int64
		ban      domain.BannedASNPerson
	)

	if errQuery := r.db.
		QueryRow(ctx, nil, query, banID).
		Scan(&ban.BanASNId, &ban.ASNum, &ban.Origin,
			&sourceID, &targetID, &ban.ReasonText, &ban.ValidUntil, &ban.CreatedOn,
			&ban.UpdatedOn, &ban.Deleted, &ban.Reason, &ban.IsEnabled, &ban.UnbanReasonText,
			&ban.AppealState, &ban.Note, &ban.SourceAvatarhash, &ban.SourcePersonaname, &ban.TargetAvatarhash,
			&ban.TargetPersonaname); errQuery != nil {
		return ban, r.db.DBErr(errQuery)
	}

	ban.TargetID = steamid.New(targetID)
	ban.SourceID = steamid.New(sourceID)

	return ban, nil
}

func (r banASNRepository) Get(ctx context.Context, filter domain.ASNBansQueryFilter) ([]domain.BannedASNPerson, error) {
	builder := r.db.
		Builder().
		Select("b.ban_asn_id", "b.as_num", "b.origin", "b.source_id",
			"b.target_id", "b.reason_text", "b.valid_until", "b.created_on", "b.updated_on",
			"b.deleted", "b.reason", "b.is_enabled", "b.unban_reason_text", "b.appeal_state",
			"coalesce(s.personaname, '') as source_personaname", "coalesce(s.avatarhash, '')",
			"coalesce(t.personaname, '') as target_personaname", "coalesce(t.avatarhash, '')",
			"coalesce(t.community_banned, false)", "coalesce(t.vac_bans, 0)", "coalesce(t.game_bans, 0)").
		From("ban_asn b").
		LeftJoin("person s on s.steam_id = b.source_id").
		LeftJoin("person t on t.steam_id = b.target_id")

	var constraints sq.And

	if !filter.Deleted {
		constraints = append(constraints, sq.Eq{"b.deleted": false})
	}

	rows, errRows := r.db.QueryBuilder(ctx, nil, builder.Where(constraints))
	if errRows != nil {
		if errors.Is(errRows, domain.ErrNoResult) {
			return []domain.BannedASNPerson{}, nil
		}

		return nil, r.db.DBErr(errRows)
	}

	defer rows.Close()

	var records []domain.BannedASNPerson

	for rows.Next() {
		var (
			ban      domain.BannedASNPerson
			targetID int64
			sourceID int64
		)

		if errScan := rows.
			Scan(&ban.BanASNId, &ban.ASNum, &ban.Origin, &sourceID, &targetID, &ban.ReasonText, &ban.ValidUntil,
				&ban.CreatedOn, &ban.UpdatedOn, &ban.Deleted, &ban.Reason, &ban.IsEnabled,
				&ban.UnbanReasonText, &ban.AppealState,
				&ban.SourceTarget.SourcePersonaname, &ban.SourceTarget.SourceAvatarhash,
				&ban.SourceTarget.TargetPersonaname, &ban.SourceTarget.TargetAvatarhash,
				&ban.CommunityBanned, &ban.VacBans, &ban.GameBans); errScan != nil {
			return nil, r.db.DBErr(errScan)
		}

		ban.SourceID = steamid.New(sourceID)
		ban.TargetID = steamid.New(targetID)

		records = append(records, ban)
	}

	if records == nil {
		records = []domain.BannedASNPerson{}
	}

	return records, nil
}

func (r banASNRepository) Save(ctx context.Context, banASN *domain.BanASN) (domain.BannedASNPerson, error) {
	var bannedPerson domain.BannedASNPerson

	banASN.UpdatedOn = time.Now()

	if banASN.BanASNId > 0 {
		const queryUpdate = `
			UPDATE ban_asn 
			SET as_num = $2, origin = $3, source_id = $4, target_id = $5, reason = $6,
				valid_until = $7, updated_on = $8, reason_text = $9, is_enabled = $10, deleted = $11, 
				unban_reason_text = $12, appeal_state = $13, note = $14
			WHERE ban_asn_id = $1`
		if err := r.db.Exec(ctx, nil, queryUpdate, banASN.BanASNId, banASN.ASNum, banASN.Origin, banASN.SourceID.Int64(),
			banASN.TargetID.Int64(), banASN.Reason, banASN.ValidUntil, banASN.UpdatedOn, banASN.ReasonText, banASN.IsEnabled,
			banASN.Deleted, banASN.UnbanReasonText, banASN.AppealState, banASN.Note); err != nil {
			return bannedPerson, r.db.DBErr(err)
		}
	} else {
		const queryInsert = `
		INSERT INTO ban_asn (as_num, origin, source_id, target_id, reason, valid_until, updated_on, created_on, 
		                     reason_text, is_enabled, deleted, unban_reason_text, appeal_state, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING ban_asn_id`

		if err := r.db.
			QueryRow(ctx, nil, queryInsert, banASN.ASNum, banASN.Origin, banASN.SourceID.Int64(), banASN.TargetID.Int64(),
				banASN.Reason, banASN.ValidUntil, banASN.UpdatedOn, banASN.CreatedOn, banASN.ReasonText, banASN.IsEnabled,
				banASN.Deleted, banASN.UnbanReasonText, banASN.AppealState, banASN.Note).
			Scan(&banASN.BanASNId); err != nil {
			return bannedPerson, r.db.DBErr(err)
		}
	}

	if banASN.Deleted {
		return bannedPerson, nil
	}

	return r.GetByID(ctx, banASN.BanASNId)
}

func (r banASNRepository) Delete(ctx context.Context, banASN domain.BanASN) error {
	banASN.Deleted = true

	if _, err := r.Save(ctx, &banASN); err != nil {
		return err
	}

	return nil
}
