package ban

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type banSteamRepository struct {
	db      database.Database
	persons domain.PersonUsecase
	network domain.NetworkUsecase
}

func NewBanSteamRepository(database database.Database, persons domain.PersonUsecase, network domain.NetworkUsecase) domain.BanSteamRepository {
	return &banSteamRepository{db: database, persons: persons, network: network}
}

func (r banSteamRepository) TruncateCache(ctx context.Context) error {
	return r.db.DBErr(r.db.ExecDeleteBuilder(ctx, r.db.Builder().Delete("steam_friends")))
}

func (r banSteamRepository) InsertCache(ctx context.Context, steamID steamid.SteamID, entries []int64) error {
	const query = "INSERT INTO steam_friends (steam_id, friend_id, created_on) VALUES ($1, $2, $3)"

	batch := pgx.Batch{}
	now := time.Now()

	for _, entrySteamID := range entries {
		_, errPerson := r.persons.GetOrCreatePersonBySteamID(ctx, steamid.New(entrySteamID))
		if errPerson != nil {
			slog.Error("Failed to validate person for friend insertion", log.ErrAttr(errPerson))

			continue
		}

		batch.Queue(query, steamID.Int64(), entrySteamID, now)
	}

	batchResults := r.db.SendBatch(ctx, &batch)
	if errCloseBatch := batchResults.Close(); errCloseBatch != nil {
		return errors.Join(errCloseBatch, domain.ErrCloseBatch)
	}

	return nil
}

func (r banSteamRepository) Stats(ctx context.Context, stats *domain.Stats) error {
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

func (r banSteamRepository) Delete(ctx context.Context, ban *domain.BanSteam, hardDelete bool) error {
	if hardDelete {
		if errExec := r.db.Exec(ctx, `DELETE FROM ban WHERE ban_id = $1`, ban.BanID); errExec != nil {
			return r.db.DBErr(errExec)
		}

		ban.BanID = 0

		return nil
	}

	ban.Deleted = true

	return r.updateBan(ctx, ban)
}

func (r banSteamRepository) getBanByColumn(ctx context.Context, column string, identifier any, deletedOk bool, evadeOK bool) (domain.BannedSteamPerson, error) {
	person := domain.NewBannedPerson()

	whereClauses := sq.And{
		sq.Eq{"b." + column: identifier}, // valid columns are immutable
		sq.Gt{"b.valid_until": time.Now()},
	}

	if !deletedOk {
		whereClauses = append(whereClauses, sq.Eq{"b.deleted": false})
	}

	if !evadeOK {
		whereClauses = append(whereClauses, sq.Eq{"b.evade_ok": false})
	}

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

func (r banSteamRepository) GetBySteamID(ctx context.Context, sid64 steamid.SteamID, deletedOk bool, evadeOK bool) (domain.BannedSteamPerson, error) {
	return r.getBanByColumn(ctx, "target_id", sid64, deletedOk, evadeOK)
}

func (r banSteamRepository) GetByBanID(ctx context.Context, banID int64, deletedOk bool, evadeOK bool) (domain.BannedSteamPerson, error) {
	return r.getBanByColumn(ctx, "ban_id", banID, deletedOk, evadeOK)
}

func (r banSteamRepository) GetByLastIP(ctx context.Context, lastIP netip.Addr, deletedOk bool, evadeOK bool) (domain.BannedSteamPerson, error) {
	// TODO check if works still
	return r.getBanByColumn(ctx, "last_ip", lastIP.String(), deletedOk, evadeOK)
}

// Save will insert or update the ban record
// New records will have the Ban.BanID set automatically.
func (r banSteamRepository) Save(ctx context.Context, ban *domain.BanSteam) error {
	// Ensure the foreign keys are satisfied
	_, errGetPerson := r.persons.GetOrCreatePersonBySteamID(ctx, ban.TargetID)
	if errGetPerson != nil {
		return errors.Join(errGetPerson, domain.ErrPersonTarget)
	}

	_, errGetAuthor := r.persons.GetPersonBySteamID(ctx, ban.SourceID)
	if errGetAuthor != nil {
		return errors.Join(errGetAuthor, domain.ErrPersonSource)
	}

	ban.UpdatedOn = time.Now()
	if ban.BanID > 0 {
		return r.updateBan(ctx, ban)
	}

	ban.CreatedOn = ban.UpdatedOn

	existing, errGetBan := r.GetBySteamID(ctx, ban.TargetID, false, true)
	if errGetBan != nil {
		if !errors.Is(errGetBan, domain.ErrNoResult) {
			return errors.Join(errGetBan, domain.ErrGetBan)
		}
	} else {
		if ban.BanType <= existing.BanType {
			return domain.ErrDuplicate
		}
	}

	ban.LastIP = r.network.GetPlayerMostRecentIP(ctx, ban.TargetID)

	return r.insertBan(ctx, ban)
}

func (r banSteamRepository) insertBan(ctx context.Context, ban *domain.BanSteam) error {
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

func (r banSteamRepository) updateBan(ctx context.Context, ban *domain.BanSteam) error {
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

func (r banSteamRepository) ExpiredBans(ctx context.Context) ([]domain.BanSteam, error) {
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
func (r banSteamRepository) Get(ctx context.Context, filter domain.SteamBansQueryFilter) ([]domain.BannedSteamPerson, error) {
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

	if len(ands) > 0 {
		builder = builder.Where(ands)
	}

	var bans []domain.BannedSteamPerson

	rows, errQuery := r.db.QueryBuilder(ctx, builder)
	if errQuery != nil {
		return nil, r.db.DBErr(errQuery)
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
			return nil, r.db.DBErr(errScan)
		}

		person.TargetID = steamid.New(targetID)
		person.SourceID = steamid.New(sourceID)

		bans = append(bans, person)
	}

	if bans == nil {
		bans = []domain.BannedSteamPerson{}
	}

	return bans, nil
}

func (r banSteamRepository) GetOlderThan(ctx context.Context, filter domain.QueryFilter, since time.Time) ([]domain.BanSteam, error) {
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
