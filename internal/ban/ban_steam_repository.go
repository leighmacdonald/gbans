package ban

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type banSteamRepository struct {
	db database.Database
	pu domain.PersonUsecase
	nu domain.NetworkUsecase
}

func NewBanSteamRepository(database database.Database, pu domain.PersonUsecase, nu domain.NetworkUsecase) domain.BanSteamRepository {
	return &banSteamRepository{db: database, pu: pu, nu: nu}
}

func (r *banSteamRepository) Stats(ctx context.Context, stats *domain.Stats) error {
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
		return r.db.DBErr(errQuery)
	}

	return nil
}

func (r *banSteamRepository) Delete(ctx context.Context, ban *domain.BanSteam, hardDelete bool) error {
	if hardDelete {
		if errExec := r.db.Exec(ctx, `DELETE FROM ban WHERE ban_id = $1`, ban.BanID); errExec != nil {
			return r.db.DBErr(errExec)
		}

		ban.BanID = 0

		return nil
	} else {
		ban.Deleted = true

		return r.updateBan(ctx, ban)
	}
}

func (r *banSteamRepository) getBanByColumn(ctx context.Context, column string, identifier any, deletedOk bool) (domain.BannedSteamPerson, error) {
	person := domain.NewBannedPerson()

	whereClauses := sq.And{
		sq.Eq{fmt.Sprintf("b.%s", column): identifier}, // valid columns are immutable
	}

	if !deletedOk {
		whereClauses = append(whereClauses, sq.Eq{"b.deleted": false})
	}
	// else {
	//	whereClauses = append(whereClauses, sq.Gt{"b.valid_until": time.Now()})
	// }

	query := r.db.
		Builder().
		Select(
			"b.ban_id", "b.target_id", "b.source_id", "b.ban_type", "b.reason",
			"b.reason_text", "b.note", "b.origin", "b.valid_until", "b.created_on", "b.updated_on", "b.include_friends", "b.evade_ok",
			"b.deleted", "case WHEN b.report_id is null THEN 0 ELSE b.report_id END",
			"b.unban_reason_text", "b.is_enabled", "b.appeal_state", "b.last_ip",
			"s.personaname as source_personaname", "s.avatarhash",
			"t.personaname as target_personaname", "t.avatarhash", "t.community_banned", "t.vac_bans", "t.game_bans",
		).
		From("ban b").
		LeftJoin("person s on s.steam_id = b.source_id").
		LeftJoin("person t on t.steam_id = b.target_id").
		Where(whereClauses).
		OrderBy("b.created_on DESC").
		Limit(1)

	row, errQuery := r.db.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return person, r.db.DBErr(errQuery)
	}

	var (
		sourceID int64
		targetID int64
	)

	if errScan := row.
		Scan(&person.BanID, &targetID, &sourceID, &person.BanType, &person.Reason,
			&person.ReasonText, &person.Note, &person.Origin, &person.ValidUntil, &person.CreatedOn,
			&person.UpdatedOn, &person.IncludeFriends, &person.EvadeOk, &person.Deleted, &person.ReportID, &person.UnbanReasonText,
			&person.IsEnabled, &person.AppealState, &person.LastIP,
			&person.SourceTarget.SourcePersonaname, &person.SourceTarget.SourceAvatarhash,
			&person.SourceTarget.TargetPersonaname, &person.SourceTarget.TargetAvatarhash,
			&person.CommunityBanned, &person.VacBans, &person.GameBans,
		); errScan != nil {
		return person, r.db.DBErr(errScan)
	}

	person.SourceID = steamid.New(sourceID)
	person.TargetID = steamid.New(targetID)

	return person, nil
}

func (r *banSteamRepository) GetBySteamID(ctx context.Context, sid64 steamid.SteamID, deletedOk bool) (domain.BannedSteamPerson, error) {
	return r.getBanByColumn(ctx, "target_id", sid64, deletedOk)
}

func (r *banSteamRepository) GetByBanID(ctx context.Context, banID int64, deletedOk bool) (domain.BannedSteamPerson, error) {
	return r.getBanByColumn(ctx, "ban_id", banID, deletedOk)
}

func (r *banSteamRepository) GetByLastIP(ctx context.Context, lastIP netip.Addr, deletedOk bool) (domain.BannedSteamPerson, error) {
	// TODO check if works still
	return r.getBanByColumn(ctx, "last_ip", lastIP.String(), deletedOk)
}

// Save will insert or update the ban record
// New records will have the Ban.BanID set automatically.
func (r *banSteamRepository) Save(ctx context.Context, ban *domain.BanSteam) error {
	// Ensure the foreign keys are satisfied
	_, errGetPerson := r.pu.GetOrCreatePersonBySteamID(ctx, ban.TargetID)
	if errGetPerson != nil {
		return errors.Join(errGetPerson, domain.ErrPersonTarget)
	}

	_, errGetAuthor := r.pu.GetPersonBySteamID(ctx, ban.SourceID)
	if errGetAuthor != nil {
		return errors.Join(errGetAuthor, domain.ErrPersonSource)
	}

	ban.UpdatedOn = time.Now()
	if ban.BanID > 0 {
		return r.updateBan(ctx, ban)
	}

	ban.CreatedOn = ban.UpdatedOn

	existing, errGetBan := r.GetBySteamID(ctx, ban.TargetID, false)
	if errGetBan != nil {
		if !errors.Is(errGetBan, domain.ErrNoResult) {
			return errors.Join(errGetBan, domain.ErrGetBan)
		}
	} else {
		if ban.BanType <= existing.BanType {
			return domain.ErrDuplicate
		}
	}

	ban.LastIP = r.nu.GetPlayerMostRecentIP(ctx, ban.TargetID)

	return r.insertBan(ctx, ban)
}

func (r *banSteamRepository) insertBan(ctx context.Context, ban *domain.BanSteam) error {
	const query = `
		INSERT INTO ban (target_id, source_id, ban_type, reason, reason_text, note, valid_until, 
		                 created_on, updated_on, origin, report_id, appeal_state, include_friends, evade_ok, last_ip)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, case WHEN $11 = 0 THEN null ELSE $11 END, $12, $13, $14, $15)
		RETURNING ban_id`

	errQuery := r.db.
		QueryRow(ctx, query, ban.TargetID.Int64(), ban.SourceID.Int64(), ban.BanType, ban.Reason, ban.ReasonText,
			ban.Note, ban.ValidUntil, ban.CreatedOn, ban.UpdatedOn, ban.Origin, ban.ReportID, ban.AppealState,
			ban.IncludeFriends, ban.EvadeOk, &ban.LastIP).
		Scan(&ban.BanID)

	if errQuery != nil {
		return r.db.DBErr(errQuery)
	}

	return nil
}

func (r *banSteamRepository) updateBan(ctx context.Context, ban *domain.BanSteam) error {
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

	return r.db.DBErr(r.db.ExecUpdateBuilder(ctx, query))
}

func (r *banSteamRepository) ExpiredBans(ctx context.Context) ([]domain.BanSteam, error) {
	query := r.db.
		Builder().
		Select("ban_id", "target_id", "source_id", "ban_type", "reason", "reason_text", "note",
			"valid_until", "origin", "created_on", "updated_on", "deleted", "case WHEN report_id is null THEN 0 ELSE report_id END",
			"unban_reason_text", "is_enabled", "appeal_state", "include_friends", "evade_ok").
		From("ban").
		Where(sq.And{sq.Lt{"valid_until": time.Now()}, sq.Eq{"deleted": false}})

	rows, errQuery := r.db.QueryBuilder(ctx, query)
	if errQuery != nil {
		return nil, r.db.DBErr(errQuery)
	}

	defer rows.Close()

	var bans []domain.BanSteam

	for rows.Next() {
		var (
			ban      domain.BanSteam
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
		return []domain.BanSteam{}, nil
	}

	return bans, nil
}

// Get returns all bans that fit the filter criteria passed in.
func (r *banSteamRepository) Get(ctx context.Context, filter domain.SteamBansQueryFilter) ([]domain.BannedSteamPerson, int64, error) {
	builder := r.db.
		Builder().
		Select("b.ban_id", "b.target_id", "b.source_id", "b.ban_type", "b.reason",
			"b.reason_text", "b.note", "b.origin", "b.valid_until", "b.created_on", "b.updated_on", "b.include_friends", "b.evade_ok",
			"b.deleted", "case WHEN b.report_id is null THEN 0 ELSE b.report_id END",
			"b.unban_reason_text", "b.is_enabled", "b.appeal_state",
			"s.personaname as source_personaname", "s.avatarhash",
			"t.personaname as target_personaname", "t.avatarhash", "t.community_banned", "t.vac_bans", "t.game_bans").
		From("ban b").
		JoinClause("LEFT JOIN person s on s.steam_id = b.source_id").
		JoinClause("LEFT JOIN person t on t.steam_id = b.target_id")

	var ands sq.And

	if !filter.Deleted {
		ands = append(ands, sq.Eq{"b.deleted": false})
	}

	if filter.Reason > 0 {
		ands = append(ands, sq.Eq{"b.reason": filter.Reason})
	}

	if filter.PermanentOnly {
		ands = append(ands, sq.Gt{"b.valid_until": time.Now()})
	}

	if sid, ok := filter.TargetSteamID(ctx); ok {
		ands = append(ands, sq.Eq{"b.target_id": sid})
	}

	if sid, ok := filter.SourceSteamID(ctx); ok {
		ands = append(ands, sq.Eq{"b.source_id": sid})
	}

	if filter.IncludeFriendsOnly {
		ands = append(ands, sq.Eq{"b.include_friends": true})
	}

	if filter.AppealState > domain.AnyState {
		ands = append(ands, sq.Eq{"b.appeal_state": filter.AppealState})
	}

	if len(ands) > 0 {
		builder = builder.Where(ands)
	}

	builder = filter.QueryFilter.ApplySafeOrder(builder, map[string][]string{
		"b.": {
			"ban_id", "target_id", "source_id", "ban_type", "reason",
			"origin", "valid_until", "created_on", "updated_on", "include_friends", "evade_ok",
			"deleted", "report_id", "is_enabled", "appeal_state",
		},
		"s.": {"source_personaname"},
		"t.": {"target_personaname", "community_banned", "vac_bans", "game_bans"},
	}, "ban_id")

	builder = filter.QueryFilter.ApplyLimitOffsetDefault(builder)

	count, errCount := r.db.GetCount(ctx, r.db.
		Builder().
		Select("COUNT(b.ban_id)").
		From("ban b").
		Where(ands))
	if errCount != nil {
		return nil, 0, r.db.DBErr(errCount)
	}

	if count == 0 {
		return []domain.BannedSteamPerson{}, 0, nil
	}

	var bans []domain.BannedSteamPerson

	rows, errQuery := r.db.QueryBuilder(ctx, builder)
	if errQuery != nil {
		return nil, 0, r.db.DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			person   = domain.NewBannedPerson()
			sourceID int64
			targetID int64
		)

		if errScan := rows.
			Scan(&person.BanID, &targetID, &sourceID, &person.BanType, &person.Reason,
				&person.ReasonText, &person.Note, &person.Origin, &person.ValidUntil, &person.CreatedOn,
				&person.UpdatedOn, &person.IncludeFriends, &person.EvadeOk, &person.Deleted, &person.ReportID, &person.UnbanReasonText,
				&person.IsEnabled, &person.AppealState,
				&person.SourceTarget.SourcePersonaname, &person.SourceTarget.SourceAvatarhash,
				&person.SourceTarget.TargetPersonaname, &person.SourceTarget.TargetAvatarhash,
				&person.CommunityBanned, &person.VacBans, &person.GameBans); errScan != nil {
			return nil, 0, r.db.DBErr(errScan)
		}

		person.TargetID = steamid.New(targetID)
		person.SourceID = steamid.New(sourceID)

		bans = append(bans, person)
	}

	if bans == nil {
		bans = []domain.BannedSteamPerson{}
	}

	return bans, count, nil
}

func (r *banSteamRepository) GetOlderThan(ctx context.Context, filter domain.QueryFilter, since time.Time) ([]domain.BanSteam, error) {
	query := r.db.
		Builder().
		Select("b.ban_id", "b.target_id", "b.source_id", "b.ban_type", "b.reason",
			"b.reason_text", "b.note", "b.origin", "b.valid_until", "b.created_on", "b.updated_on", "b.deleted",
			"case WHEN b.report_id is null THEN 0 ELSE s.report_id END", "b.unban_reason_text", "b.is_enabled",
			"b.appeal_state", "b.include_friends", "b.evade_ok").
		From("ban b").
		Where(sq.And{sq.Lt{"b.updated_on": since}, sq.Eq{"b.deleted": false}})

	rows, errQuery := r.db.QueryBuilder(ctx, filter.ApplyLimitOffsetDefault(query))
	if errQuery != nil {
		return nil, r.db.DBErr(errQuery)
	}

	defer rows.Close()

	var bans []domain.BanSteam

	for rows.Next() {
		var (
			ban      domain.BanSteam
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
		return []domain.BanSteam{}, nil
	}

	return bans, nil
}
