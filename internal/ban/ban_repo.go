package ban

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type Repository struct {
	db      database.Database
	persons person.Provider // TODO remove
}

func (r *Repository) Query(ctx context.Context, opts QueryOpts) ([]Ban, error) {
	builder := r.db.
		Builder().
		Select("b.ban_id", "b.target_id", "b.source_id", "b.ban_type", "b.reason",
			"b.reason_text", "b.note", "b.origin", "b.valid_until", "b.created_on", "b.updated_on", "b.evade_ok",
			"b.deleted", "case WHEN b.report_id is null THEN 0 ELSE b.report_id END",
			"b.unban_reason_text", "b.is_enabled", "b.appeal_state", "b.cidr",
			"coalesce(s.personaname, s.steam_id::text)", "coalesce(s.avatarhash, 'fef49e7fa7e1997310d705b2a6158ff8dc1cdfeb')",
			"coalesce(t.personaname, t.steam_id::text)", "coalesce(t.avatarhash, 'fef49e7fa7e1997310d705b2a6158ff8dc1cdfeb')").
		From("ban b").
		LeftJoin("person s ON b.source_id = s.steam_id").
		LeftJoin("person t ON b.target_id = t.steam_id")

	var ands sq.And

	if !opts.Deleted {
		ands = append(ands, sq.Eq{"b.deleted": false})
	}

	if opts.CIDROnly {
		ands = append(ands, sq.NotEq{"b.cidr": nil})
	}

	if opts.CIDR != "" {
		ands = append(ands, sq.Expr("?::inet @> ip_range", opts.CIDR))
	}

	if !opts.ValidUntil.IsZero() {
		ands = append(ands, sq.Lt{"valid_until": time.Now()})
	}

	// if opts.SourceID.Valid() {
	// 	ands = append(ands, sq.Eq{"b.source_id": opts.SourceID.Int64()})
	// }

	// if opts.TargetID.Valid() {
	// 	ands = append(ands, sq.Eq{"b.target_id": opts.TargetID.Int64()})
	// }

	if len(opts.Reasons) > 0 {
		ands = append(ands, sq.Eq{"b.reason": opts.Reasons})
	}

	if opts.GroupsOnly {
		ands = append(ands, sq.GtOrEq{"b.target_id": steamid.BaseGID})
	}

	if !opts.IncludeGroups {
		ands = append(ands, sq.Lt{"b.target_id": steamid.BaseGID})
	}

	if len(ands) > 0 {
		builder = builder.Where(ands)
	}

	var bans []Ban

	rows, errQuery := r.db.QueryBuilder(ctx, builder)
	if errQuery != nil {
		return nil, database.DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			ban      = Ban{}
			sourceID int64
			targetID int64
		)

		if errScan := rows.
			Scan(&ban.BanID, &targetID, &sourceID, &ban.BanType, &ban.Reason,
				&ban.ReasonText, &ban.Note, &ban.Origin, &ban.ValidUntil, &ban.CreatedOn,
				&ban.UpdatedOn, &ban.EvadeOk, &ban.Deleted, &ban.ReportID, &ban.UnbanReasonText,
				&ban.IsEnabled, &ban.AppealState, &ban.CIDR,
				&ban.SourcePersonaname, &ban.SourceAvatarhash,
				&ban.TargetPersonaname, &ban.TargetAvatarhash); errScan != nil {
			return nil, database.DBErr(errScan)
		}

		ban.TargetID = steamid.New(targetID)
		ban.SourceID = steamid.New(sourceID)

		bans = append(bans, ban)
	}

	if bans == nil {
		bans = []Ban{}
	}

	return bans, nil
}

func NewRepository(database database.Database, persons person.Provider) Repository {
	return Repository{db: database, persons: persons}
}

func (r *Repository) TruncateCache(ctx context.Context) error {
	// return r.db.DBErr(r.db.ExecDeleteBuilder(ctx, nil, r.db.Builder().Delete("steam_friends")))
	return database.DBErr(r.db.ExecDeleteBuilder(ctx, r.db.Builder().Delete("steam_group_members")))
}

func (r *Repository) GetMembersList(ctx context.Context, parentID int64, list *MembersList) error {
	row, err := r.db.QueryRowBuilder(ctx, r.db.
		Builder().
		Select("members_id", "parent_id", "members", "created_on", "updated_on").
		From("members").
		Where(sq.Eq{"parent_id": parentID}))
	if err != nil {
		return database.DBErr(err)
	}

	return database.DBErr(row.Scan(&list.MembersID, &list.ParentID, &list.Members, &list.CreatedOn, &list.UpdatedOn))
}

func (r *Repository) SaveMembersList(ctx context.Context, list *MembersList) error {
	if list.MembersID > 0 {
		list.UpdatedOn = time.Now()

		const update = `UPDATE members SET members = $2::jsonb, updated_on = $3 WHERE members_id = $1`

		return database.DBErr(r.db.Exec(ctx, update, list.MembersID, list.Members, list.UpdatedOn))
	}

	const insert = `INSERT INTO members (parent_id, members, created_on, updated_on)
		VALUES ($1, $2::jsonb, $3, $4) RETURNING members_id`

	return database.DBErr(r.db.QueryRow(ctx, insert, list.ParentID, list.Members, list.CreatedOn, list.UpdatedOn).Scan(&list.MembersID))
}

func (r *Repository) InsertCache(ctx context.Context, groupID steamid.SteamID, entries []int64) error {
	const query = "INSERT INTO steam_group_members (steam_id, group_id, created_on) VALUES ($1, $2, $3)"

	batch := pgx.Batch{}
	now := time.Now()

	for _, entrySteamID := range entries {
		batch.Queue(query, entrySteamID, groupID.Int64(), now)
	}

	batchResults := r.db.SendBatch(ctx, &batch)
	if errCloseBatch := batchResults.Close(); errCloseBatch != nil {
		return errors.Join(errCloseBatch, database.ErrCloseBatch)
	}

	return nil
}

func (r *Repository) Stats(ctx context.Context, stats *Stats) error {
	const query = `
	SELECT
		(SELECT COUNT(ban_id) FROM ban) as bans_total,
		(SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '1 DAY')) as bans_day,
	    (SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '1 DAY')) as bans_week,
		(SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '1 MONTH')) as bans_month,
	    (SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '3 MONTH')) as bans_3month,
	    (SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '6 MONTH')) as bans_6month,
	    (SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '1 YEAR')) as bans_year,
		(SELECT COUNT(net_id) FROM ban_net) as bans_cidr,
		(SELECT COUNT(filter_id) FROM filtered_word) as filtered_words,
		(SELECT COUNT(server_id) FROM server) as servers_total`

	if errQuery := r.db.QueryRow(ctx, query).
		Scan(&stats.BansTotal, &stats.BansDay, &stats.BansWeek, &stats.BansMonth, &stats.Bans3Month, &stats.Bans6Month, &stats.BansYear, &stats.BansCIDRTotal, &stats.FilteredWords, &stats.ServersTotal); errQuery != nil {
		return database.DBErr(errQuery)
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, ban *Ban, hardDelete bool) error {
	if hardDelete {
		if errExec := r.db.Exec(ctx, `DELETE FROM ban WHERE ban_id = $1`, ban.BanID); errExec != nil {
			return database.DBErr(errExec)
		}

		ban.BanID = 0

		return nil
	}

	ban.Deleted = true

	return r.updateBan(ctx, ban)
}

// Save will insert or update the ban record
// New records will have the Ban.BanID set automatically.
func (r *Repository) Save(ctx context.Context, ban *Ban) error {
	// Ensure the foreign keys are satisfied
	_, errGetPerson := r.persons.GetOrCreatePersonBySteamID(ctx, ban.TargetID)
	if errGetPerson != nil {
		return errors.Join(errGetPerson, ErrPersonTarget)
	}

	_, errGetAuthor := r.persons.GetOrCreatePersonBySteamID(ctx, ban.SourceID)
	if errGetAuthor != nil {
		return errors.Join(errGetAuthor, ErrPersonSource)
	}

	ban.UpdatedOn = time.Now()
	if ban.BanID > 0 {
		return r.updateBan(ctx, ban)
	}

	ban.CreatedOn = ban.UpdatedOn

	existing, errGetBan := r.Query(ctx, QueryOpts{TargetID: ban.TargetID, EvadeOk: true})
	if errGetBan != nil {
		if !errors.Is(errGetBan, database.ErrNoResult) {
			return errors.Join(errGetBan, ErrGetBan)
		}
	} else if len(existing) > 0 && ban.BanType <= existing[0].BanType {
		return database.ErrDuplicate
	}

	// TODO Use trigger / stored proc
	// if lastIP := r.network.GetPlayerMostRecentIP(ctx, ban.TargetID); lastIP != nil {
	// 	last := lastIP.String()
	// 	ban.LastIP = &last
	// }

	return r.insertBan(ctx, ban)
}

func (r *Repository) insertBan(ctx context.Context, ban *Ban) error {
	const query = `
		INSERT INTO ban (target_id, source_id, ban_type, reason, reason_text, note, valid_until,
		                 created_on, updated_on, origin, report_id, appeal_state, evade_ok, last_ip, cidr)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, case WHEN $11 = 0 THEN null ELSE $11 END, $12, $13, $14, $15)
		RETURNING ban_id`

	errQuery := r.db.
		QueryRow(ctx, query, ban.TargetID.Int64(), ban.SourceID.Int64(), ban.BanType, ban.Reason, ban.ReasonText,
			ban.Note, ban.ValidUntil, ban.CreatedOn, ban.UpdatedOn, ban.Origin, ban.ReportID, ban.AppealState,
			ban.EvadeOk, &ban.LastIP, &ban.CIDR).
		Scan(&ban.BanID)

	if errQuery != nil {
		return database.DBErr(errQuery)
	}

	return nil
}

func (r *Repository) updateBan(ctx context.Context, ban *Ban) error {
	var reportID *int64
	if ban.ReportID > 0 {
		reportID = &ban.ReportID
	}

	query := r.db.
		Builder().
		Update("ban").
		Set("source_id", ban.SourceID.Int64()).
		Set("target_id", ban.TargetID.Int64()).
		Set("reason", ban.Reason).
		Set("reason_text", ban.ReasonText).
		Set("note", ban.Note).
		Set("valid_until", ban.ValidUntil).
		Set("updated_on", ban.UpdatedOn).
		Set("origin", ban.Origin).
		Set("ban_type", ban.BanType).
		Set("deleted", ban.Deleted).
		Set("report_id", reportID).
		Set("unban_reason_text", ban.UnbanReasonText).
		Set("is_enabled", ban.IsEnabled).
		Set("appeal_state", ban.AppealState).
		Set("evade_ok", ban.EvadeOk).
		Set("cidr", ban.CIDR).
		Where(sq.Eq{"ban_id": ban.BanID})

	return database.DBErr(r.db.ExecUpdateBuilder(ctx, query))
}

func (r *Repository) GetOlderThan(ctx context.Context, filter query.Filter, since time.Time) ([]Ban, error) {
	query := r.db.
		Builder().
		Select("b.ban_id", "b.target_id", "b.source_id", "b.ban_type", "b.reason",
			"b.reason_text", "b.note", "b.origin", "b.valid_until", "b.created_on", "b.updated_on", "b.deleted",
			"case WHEN b.report_id is null THEN 0 ELSE s.report_id END", "b.unban_reason_text", "b.is_enabled",
			"b.appeal_state", "b.evade_ok", "b.cidr").
		From("ban b").
		Where(sq.And{sq.Lt{"b.updated_on": since}, sq.Eq{"b.deleted": false}})

	rows, errQuery := r.db.QueryBuilder(ctx, filter.ApplyLimitOffsetDefault(query))
	if errQuery != nil {
		return nil, database.DBErr(errQuery)
	}

	defer rows.Close()

	var bans []Ban

	for rows.Next() {
		var (
			ban      Ban
			sourceID int64
			targetID int64
		)

		if errQuery = rows.Scan(&ban.BanID, &targetID, &sourceID, &ban.BanType, &ban.Reason, &ban.ReasonText, &ban.Note,
			&ban.Origin, &ban.ValidUntil, &ban.CreatedOn, &ban.UpdatedOn, &ban.Deleted, &ban.ReportID, &ban.UnbanReasonText,
			&ban.IsEnabled, &ban.AppealState, &ban.EvadeOk, &ban.CIDR); errQuery != nil {
			return nil, errors.Join(errQuery, database.ErrScanResult)
		}

		ban.SourceID = steamid.New(sourceID)
		ban.TargetID = steamid.New(targetID)

		bans = append(bans, ban)
	}

	if bans == nil {
		return []Ban{}, nil
	}

	return bans, nil
}
