package ban

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type BanRepository struct {
	db      database.Database
	persons domain.PersonProvider  // TODO remove
	network network.NetworkUsecase // TODO remove
}

func (r *BanRepository) Query(ctx context.Context, opts QueryOpts) ([]Ban, error) {
	builder := r.db.
		Builder().
		Select("b.ban_id", "b.target_id", "b.source_id", "b.ban_type", "b.reason",
			"b.reason_text", "b.note", "b.origin", "b.valid_until", "b.created_on", "b.updated_on", "b.include_friends", "b.evade_ok",
			"b.deleted", "case WHEN b.report_id is null THEN 0 ELSE b.report_id END",
			"b.unban_reason_text", "b.is_enabled", "b.appeal_state").
		From("ban b")

	var ands sq.And

	if !opts.Deleted {
		ands = append(ands, sq.Eq{"b.deleted": false})
	}

	if len(ands) > 0 {
		builder = builder.Where(ands)
	}

	var bans []Ban

	rows, errQuery := r.db.QueryBuilder(ctx, nil, builder)
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
				&ban.UpdatedOn, &ban.IncludeFriends, &ban.EvadeOk, &ban.Deleted, &ban.ReportID, &ban.UnbanReasonText,
				&ban.IsEnabled, &ban.AppealState); errScan != nil {
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

func NewBanRepository(database database.Database, persons domain.PersonProvider, network network.NetworkUsecase) BanRepository {
	return BanRepository{db: database, persons: persons, network: network}
}

func (r *BanRepository) TruncateCache(ctx context.Context) error {
	// return r.db.DBErr(r.db.ExecDeleteBuilder(ctx, nil, r.db.Builder().Delete("steam_friends")))
	return database.DBErr(r.db.ExecDeleteBuilder(ctx, nil, r.db.Builder().Delete("steam_group_members")))
}

func (r *BanRepository) GetMembersList(ctx context.Context, parentID int64, list *MembersList) error {
	row, err := r.db.QueryRowBuilder(ctx, nil, r.db.
		Builder().
		Select("members_id", "parent_id", "members", "created_on", "updated_on").
		From("members").
		Where(sq.Eq{"parent_id": parentID}))
	if err != nil {
		return database.DBErr(err)
	}

	return database.DBErr(row.Scan(&list.MembersID, &list.ParentID, &list.Members, &list.CreatedOn, &list.UpdatedOn))
}

func (r *BanRepository) SaveMembersList(ctx context.Context, list *MembersList) error {
	if list.MembersID > 0 {
		list.UpdatedOn = time.Now()

		const update = `UPDATE members SET members = $2::jsonb, updated_on = $3 WHERE members_id = $1`

		return database.DBErr(r.db.Exec(ctx, nil, update, list.MembersID, list.Members, list.UpdatedOn))
	}

	const insert = `INSERT INTO members (parent_id, members, created_on, updated_on)
		VALUES ($1, $2::jsonb, $3, $4) RETURNING members_id`

	return database.DBErr(r.db.QueryRow(ctx, nil, insert, list.ParentID, list.Members, list.CreatedOn, list.UpdatedOn).Scan(&list.MembersID))
}

func (r *BanRepository) InsertCache(ctx context.Context, groupID steamid.SteamID, entries []int64) error {
	const query = "INSERT INTO steam_group_members (steam_id, group_id, created_on) VALUES ($1, $2, $3)"

	batch := pgx.Batch{}
	now := time.Now()

	for _, entrySteamID := range entries {
		batch.Queue(query, entrySteamID, groupID.Int64(), now)
	}

	batchResults := r.db.SendBatch(ctx, nil, &batch)
	if errCloseBatch := batchResults.Close(); errCloseBatch != nil {
		return errors.Join(errCloseBatch, domain.ErrCloseBatch)
	}

	return nil
}

func (r *BanRepository) Stats(ctx context.Context, stats *Stats) error {
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

	if errQuery := r.db.QueryRow(ctx, nil, query).
		Scan(&stats.BansTotal, &stats.BansDay, &stats.BansWeek, &stats.BansMonth, &stats.Bans3Month, &stats.Bans6Month, &stats.BansYear, &stats.BansCIDRTotal, &stats.FilteredWords, &stats.ServersTotal); errQuery != nil {
		return database.DBErr(errQuery)
	}

	return nil
}

func (r *BanRepository) Delete(ctx context.Context, ban *Ban, hardDelete bool) error {
	if hardDelete {
		if errExec := r.db.Exec(ctx, nil, `DELETE FROM ban WHERE ban_id = $1`, ban.BanID); errExec != nil {
			return database.DBErr(errExec)
		}

		ban.BanID = 0

		return nil
	}

	ban.Deleted = true

	return r.updateBan(ctx, ban)
}

// func (r *BanRepository) getBanByColumn(ctx context.Context, column string, identifier any, deletedOk bool, evadeOK bool) (Ban, error) {
// 	person := NewBannedPerson()

// 	whereClauses := sq.And{
// 		sq.Eq{"b." + column: identifier}, // valid columns are immutable
// 	}

// 	if !deletedOk {
// 		whereClauses = append(whereClauses, sq.Eq{"b.deleted": false})
// 	}

// 	if !evadeOK {
// 		whereClauses = append(whereClauses, sq.Eq{"b.evade_ok": false})
// 	}

// 	query := r.db.
// 		Builder().
// 		Select(
// 			"b.ban_id", "b.target_id", "b.source_id", "b.ban_type", "b.reason",
// 			"b.reason_text", "b.note", "b.origin", "b.valid_until", "b.created_on", "b.updated_on", "b.include_friends", "b.evade_ok",
// 			"b.deleted", "case WHEN b.report_id is null THEN 0 ELSE b.report_id END",
// 			"b.unban_reason_text", "b.is_enabled", "b.appeal_state", "b.last_ip",
// 			"s.personaname as source_personaname", "s.avatarhash",
// 			"t.personaname as target_personaname", "t.avatarhash", "t.community_banned", "t.vac_bans", "t.game_bans",
// 		).
// 		From("ban b").
// 		LeftJoin("person s on s.steam_id = b.source_id").
// 		LeftJoin("person t on t.steam_id = b.target_id").
// 		Where(whereClauses).
// 		OrderBy("b.created_on DESC").
// 		Limit(1)

// 	row, errQuery := r.db.QueryRowBuilder(ctx, nil, query)
// 	if errQuery != nil {
// 		return person, r.db.DBErr(errQuery)
// 	}

// 	var (
// 		sourceID int64
// 		targetID int64
// 	)

// 	if errScan := row.
// 		Scan(&person.BanID, &targetID, &sourceID, &person.BanType, &person.Reason,
// 			&person.ReasonText, &person.Note, &person.Origin, &person.ValidUntil, &person.CreatedOn,
// 			&person.UpdatedOn, &person.IncludeFriends, &person.EvadeOk, &person.Deleted, &person.ReportID, &person.UnbanReasonText,
// 			&person.IsEnabled, &person.AppealState, &person.LastIP,
// 			&person.SourcePersonaname, &person.SourceAvatarhash,
// 			&person.TargetPersonaname, &person.TargetAvatarhash,
// 			&person.CommunityBanned, &person.VacBans, &person.GameBans,
// 		); errScan != nil {
// 		return person, r.db.DBErr(errScan)
// 	}

// 	person.SourceID = steamid.New(sourceID)
// 	person.TargetID = steamid.New(targetID)

// 	return person, nil
// }

// func (r *BanRepository) GetBySteamID(ctx context.Context, sid64 steamid.SteamID, deletedOk bool, evadeOK bool) (BannedPerson, error) {
// 	return r.getBanByColumn(ctx, "target_id", sid64, deletedOk, evadeOK)
// }

// func (r *BanRepository) GetByBanID(ctx context.Context, banID int64, deletedOk bool, evadeOK bool) (BannedPerson, error) {
// 	return r.getBanByColumn(ctx, "ban_id", banID, deletedOk, evadeOK)
// }

// func (r *BanRepository) GetByLastIP(ctx context.Context, lastIP netip.Addr, deletedOk bool, evadeOK bool) (BannedPerson, error) {
// 	// TODO check if works still
// 	return r.getBanByColumn(ctx, "last_ip", lastIP.String(), deletedOk, evadeOK)
// }

// Save will insert or update the ban record
// New records will have the Ban.BanID set automatically.
func (r *BanRepository) Save(ctx context.Context, ban *Ban) error {
	// Ensure the foreign keys are satisfied
	_, errGetPerson := r.persons.GetOrCreatePersonBySteamID(ctx, nil, ban.TargetID)
	if errGetPerson != nil {
		return errors.Join(errGetPerson, domain.ErrPersonTarget)
	}

	_, errGetAuthor := r.persons.GetOrCreatePersonBySteamID(ctx, nil, ban.SourceID)
	if errGetAuthor != nil {
		return errors.Join(errGetAuthor, domain.ErrPersonSource)
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

	if lastIp := r.network.GetPlayerMostRecentIP(ctx, ban.TargetID); lastIp != nil {
		ban.LastIP = lastIp.String()
	}

	return r.insertBan(ctx, ban)
}

func (r *BanRepository) insertBan(ctx context.Context, ban *Ban) error {
	const query = `
		INSERT INTO ban (target_id, source_id, ban_type, reason, reason_text, note, valid_until,
		                 created_on, updated_on, origin, report_id, appeal_state, include_friends, evade_ok, last_ip)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, case WHEN $11 = 0 THEN null ELSE $11 END, $12, $13, $14, $15)
		RETURNING ban_id`

	errQuery := r.db.
		QueryRow(ctx, nil, query, ban.TargetID.Int64(), ban.SourceID.Int64(), ban.BanType, ban.Reason, ban.ReasonText,
			ban.Note, ban.ValidUntil, ban.CreatedOn, ban.UpdatedOn, ban.Origin, ban.ReportID, ban.AppealState,
			ban.IncludeFriends, ban.EvadeOk, &ban.LastIP).
		Scan(&ban.BanID)

	if errQuery != nil {
		return database.DBErr(errQuery)
	}

	return nil
}

func (r *BanRepository) updateBan(ctx context.Context, ban *Ban) error {
	var reportID *int64
	if ban.ReportID > 0 {
		reportID = &ban.ReportID
	}

	query := r.db.
		Builder().
		Update("ban").
		Set("source_id", ban.SourceID.Int64()).
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
		Set("target_id", ban.TargetID.Int64()).
		Set("appeal_state", ban.AppealState).
		Set("include_friends", ban.IncludeFriends).
		Set("evade_ok", ban.EvadeOk).
		Where(sq.Eq{"ban_id": ban.BanID})

	return database.DBErr(r.db.ExecUpdateBuilder(ctx, nil, query))
}

func (r *BanRepository) ExpiredBans(ctx context.Context) ([]Ban, error) {
	query := r.db.
		Builder().
		Select("ban_id", "target_id", "source_id", "ban_type", "reason", "reason_text", "note",
			"valid_until", "origin", "created_on", "updated_on", "deleted", "case WHEN report_id is null THEN 0 ELSE report_id END",
			"unban_reason_text", "is_enabled", "appeal_state", "include_friends", "evade_ok").
		From("ban").
		Where(sq.And{sq.Lt{"valid_until": time.Now()}, sq.Eq{"deleted": false}})

	rows, errQuery := r.db.QueryBuilder(ctx, nil, query)
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

		if errScan := rows.Scan(&ban.BanID, &targetID, &sourceID, &ban.BanType, &ban.Reason, &ban.ReasonText, &ban.Note,
			&ban.ValidUntil, &ban.Origin, &ban.CreatedOn, &ban.UpdatedOn, &ban.Deleted, &ban.ReportID, &ban.UnbanReasonText,
			&ban.IsEnabled, &ban.AppealState, &ban.IncludeFriends, &ban.EvadeOk); errScan != nil {
			return nil, errors.Join(errScan, domain.ErrScanResult)
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

func (r *BanRepository) GetOlderThan(ctx context.Context, filter domain.QueryFilter, since time.Time) ([]Ban, error) {
	query := r.db.
		Builder().
		Select("b.ban_id", "b.target_id", "b.source_id", "b.ban_type", "b.reason",
			"b.reason_text", "b.note", "b.origin", "b.valid_until", "b.created_on", "b.updated_on", "b.deleted",
			"case WHEN b.report_id is null THEN 0 ELSE s.report_id END", "b.unban_reason_text", "b.is_enabled",
			"b.appeal_state", "b.include_friends", "b.evade_ok").
		From("ban b").
		Where(sq.And{sq.Lt{"b.updated_on": since}, sq.Eq{"b.deleted": false}})

	rows, errQuery := r.db.QueryBuilder(ctx, nil, filter.ApplyLimitOffsetDefault(query))
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
			&ban.IsEnabled, &ban.AppealState, &ban.IncludeFriends, &ban.EvadeOk); errQuery != nil {
			return nil, errors.Join(errQuery, domain.ErrScanResult)
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
