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

	rows, errQuery := r.db.QueryBuilder(ctx, query)
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
		       deleted, reason, is_enabled, unban_reason_text, appeal_state
		FROM ban_asn 
		WHERE deleted = false AND as_num = $1`

	var (
		targetID int64
		sourceID int64
	)

	if errQuery := r.db.
		QueryRow(ctx, query, asNum).
		Scan(&banASN.BanASNId, &banASN.ASNum, &banASN.Origin,
			&sourceID, &targetID, &banASN.ReasonText, &banASN.ValidUntil, &banASN.CreatedOn,
			&banASN.UpdatedOn, &banASN.Deleted, &banASN.Reason, &banASN.IsEnabled, &banASN.UnbanReasonText,
			&banASN.AppealState); errQuery != nil {
		return r.db.DBErr(errQuery)
	}

	banASN.TargetID = steamid.New(targetID)
	banASN.SourceID = steamid.New(sourceID)

	return nil
}
func (r banASNRepository) GetByID(ctx context.Context, banID int64, banASN *domain.BanASN) error {
	const query = `
		SELECT ban_asn_id, as_num, origin, source_id, target_id, reason_text, valid_until, created_on, updated_on, 
		       deleted, reason, is_enabled, unban_reason_text, appeal_state
		FROM ban_asn 
		WHERE deleted = false AND ban_asn_id = $1`

	var (
		targetID int64
		sourceID int64
	)

	if errQuery := r.db.
		QueryRow(ctx, query, banID).
		Scan(&banASN.BanASNId, &banASN.ASNum, &banASN.Origin,
			&sourceID, &targetID, &banASN.ReasonText, &banASN.ValidUntil, &banASN.CreatedOn,
			&banASN.UpdatedOn, &banASN.Deleted, &banASN.Reason, &banASN.IsEnabled, &banASN.UnbanReasonText,
			&banASN.AppealState); errQuery != nil {
		return r.db.DBErr(errQuery)
	}

	banASN.TargetID = steamid.New(targetID)
	banASN.SourceID = steamid.New(sourceID)

	return nil
}

func (r banASNRepository) Get(ctx context.Context, filter domain.ASNBansQueryFilter) ([]domain.BannedASNPerson, int64, error) {
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

	if filter.Reason > 0 {
		constraints = append(constraints, sq.Eq{"b.reason": filter.Reason})
	}

	if filter.PermanentOnly {
		constraints = append(constraints, sq.Gt{"b.valid_until": time.Now()})
	}

	if sid, ok := filter.TargetSteamID(ctx); ok {
		constraints = append(constraints, sq.Eq{"b.target_id": sid})
	}

	if sid, ok := filter.SourceSteamID(ctx); ok {
		constraints = append(constraints, sq.Eq{"b.source_id": sid})
	}

	if filter.ASNum > 0 {
		constraints = append(constraints, sq.Eq{"b.as_num": filter.ASNum})
	}

	builder = filter.QueryFilter.ApplySafeOrder(builder, map[string][]string{
		"b.": {
			"ban_asn_id", "as_num", "origin", "source_id", "target_id", "valid_until", "created_on", "updated_on",
			"deleted", "reason", "is_enabled", "appeal_state",
		},
		"s.": {"source_personaname"},
		"t.": {"target_personaname", "community_banned", "vac_bans", "game_bans"},
	}, "ban_asn_id")

	builder = filter.QueryFilter.ApplyLimitOffsetDefault(builder)

	rows, errRows := r.db.QueryBuilder(ctx, builder.Where(constraints))
	if errRows != nil {
		if errors.Is(errRows, domain.ErrNoResult) {
			return []domain.BannedASNPerson{}, 0, nil
		}

		return nil, 0, r.db.DBErr(errRows)
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
			return nil, 0, r.db.DBErr(errScan)
		}

		ban.SourceID = steamid.New(sourceID)
		ban.TargetID = steamid.New(targetID)

		records = append(records, ban)
	}

	count, errCount := r.db.GetCount(ctx, r.db.
		Builder().
		Select("COUNT(b.ban_asn_id)").
		From("ban_asn b").
		Where(constraints))

	if errCount != nil {
		if errors.Is(errCount, domain.ErrNoResult) {
			return []domain.BannedASNPerson{}, 0, nil
		}

		return nil, 0, r.db.DBErr(errCount)
	}

	if records == nil {
		records = []domain.BannedASNPerson{}
	}

	return records, count, nil
}

func (r banASNRepository) Save(ctx context.Context, banASN *domain.BanASN) error {
	banASN.UpdatedOn = time.Now()

	if banASN.BanASNId > 0 {
		const queryUpdate = `
			UPDATE ban_asn 
			SET as_num = $2, origin = $3, source_id = $4, target_id = $5, reason = $6,
				valid_until = $7, updated_on = $8, reason_text = $9, is_enabled = $10, deleted = $11, 
				unban_reason_text = $12, appeal_state = $13
			WHERE ban_asn_id = $1`

		return r.db.DBErr(r.db.
			Exec(ctx, queryUpdate, banASN.BanASNId, banASN.ASNum, banASN.Origin, banASN.SourceID.Int64(),
				banASN.TargetID.Int64(), banASN.Reason, banASN.ValidUntil, banASN.UpdatedOn, banASN.ReasonText, banASN.IsEnabled,
				banASN.Deleted, banASN.UnbanReasonText, banASN.AppealState))
	}

	const queryInsert = `
		INSERT INTO ban_asn (as_num, origin, source_id, target_id, reason, valid_until, updated_on, created_on, 
		                     reason_text, is_enabled, deleted, unban_reason_text, appeal_state)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING ban_asn_id`

	return r.db.DBErr(r.db.
		QueryRow(ctx, queryInsert, banASN.ASNum, banASN.Origin, banASN.SourceID.Int64(), banASN.TargetID.Int64(),
			banASN.Reason, banASN.ValidUntil, banASN.UpdatedOn, banASN.CreatedOn, banASN.ReasonText, banASN.IsEnabled,
			banASN.Deleted, banASN.UnbanReasonText, banASN.AppealState).
		Scan(&banASN.BanASNId))
}

func (r banASNRepository) Delete(ctx context.Context, banASN *domain.BanASN) error {
	banASN.Deleted = true

	return r.Save(ctx, banASN)
}
