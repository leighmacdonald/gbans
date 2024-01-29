package ban

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgtype"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

type banRepository struct {
	db database.Database
	pu domain.PersonUsecase
	nu domain.NetworkUsecase
}

func NewBanRepository(database database.Database, pu domain.PersonUsecase, nu domain.NetworkUsecase) domain.BanRepository {
	return &banRepository{db: database, pu: pu, nu: nu}
}

func (r *banRepository) GetStats(ctx context.Context, stats *domain.Stats) error {
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

// GetBanNetByAddress returns the BanCIDR matching intersecting the supplied ip.
//
// Note that this function does not currently limit results returned. This may change in the future, do not
// rely on this functionality.
func (r *banRepository) GetBanNetByAddress(ctx context.Context, ipAddr net.IP) ([]domain.BanCIDR, error) {
	const query = `
		SELECT net_id, cidr, origin, created_on, updated_on, reason, reason_text, valid_until, deleted, 
		       note, unban_reason_text, is_enabled, target_id, source_id, appeal_state
		FROM ban_net
		WHERE $1 <<= cidr AND deleted = false AND is_enabled = true`

	var nets []domain.BanCIDR

	rows, errQuery := r.db.Query(ctx, query, ipAddr.String())
	if errQuery != nil {
		return nil, r.db.DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			banNet   domain.BanCIDR
			sourceID int64
			targetID int64
			cidr     *net.IPNet
		)

		if errScan := rows.
			Scan(&banNet.NetID, &cidr, &banNet.Origin,
				&banNet.CreatedOn, &banNet.UpdatedOn, &banNet.Reason, &banNet.ReasonText,
				&banNet.ValidUntil, &banNet.Deleted, &banNet.Note, &banNet.UnbanReasonText,
				&banNet.IsEnabled, &targetID, &sourceID, &banNet.AppealState); errScan != nil {
			return nil, r.db.DBErr(errScan)
		}

		banNet.CIDR = cidr.String()
		banNet.SourceID = steamid.New(sourceID)
		banNet.TargetID = steamid.New(targetID)

		nets = append(nets, banNet)
	}

	if nets == nil {
		return []domain.BanCIDR{}, nil
	}

	return nets, nil
}

// GetBansNet returns the BanCIDR matching intersecting the supplied ip.
func (r *banRepository) GetBansNet(ctx context.Context, filter domain.CIDRBansQueryFilter) ([]domain.BannedCIDRPerson, int64, error) {
	validColumns := map[string][]string{
		"b.": {
			"net_id", "cidr", "origin", "created_on", "updated_on",
			"reason", "valid_until", "deleted", "is_enabled", "target_id", "source_id", "appeal_state",
		},
		"s.": {"source_personaname"},
		"t.": {"target_personaname", "community_banned", "vac_bans", "game_bans"},
	}

	builder := r.db.
		Builder().
		Select("b.net_id", "b.cidr", "b.origin", "b.created_on", "b.updated_on",
			"b.reason", "b.reason_text", "b.valid_until", "b.deleted", "b.note", "b.unban_reason_text",
			"b.is_enabled", "b.target_id", "b.source_id", "b.appeal_state",
			"s.personaname as source_personaname", "s.avatarhash",
			"t.personaname as target_personaname", "t.avatarhash", "t.community_banned", "t.vac_bans", "t.game_bans",
		).
		From("ban_net b").
		LeftJoin("person s ON s.steam_id = b.source_id").
		LeftJoin("person t ON t.steam_id = b.target_id")

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

	if filter.TargetID != "" {
		targetID, errTargetID := filter.TargetID.SID64(ctx)
		if errTargetID != nil {
			return nil, 0, errors.Join(errTargetID, domain.ErrTargetID)
		}

		constraints = append(constraints, sq.Eq{"b.target_id": targetID.Int64()})
	}

	if filter.SourceID != "" {
		sourceID, errSourceID := filter.SourceID.SID64(ctx)
		if errSourceID != nil {
			return nil, 0, errors.Join(errSourceID, domain.ErrSourceID)
		}

		constraints = append(constraints, sq.Eq{"b.source_id": sourceID.Int64()})
	}

	if filter.IP != "" {
		var addr string

		_, cidr, errCidr := net.ParseCIDR(filter.IP)

		if errCidr != nil {
			ip := net.ParseIP(filter.IP)
			if ip == nil {
				return nil, 0, errors.Join(errCidr, domain.ErrInvalidIP)
			}

			addr = ip.String()
		} else {
			addr = cidr.String()
		}

		constraints = append(constraints, sq.Expr("? <<= cidr", addr))
	}

	builder = filter.QueryFilter.ApplySafeOrder(builder, validColumns, "net_id")
	builder = filter.QueryFilter.ApplyLimitOffsetDefault(builder)

	var nets []domain.BannedCIDRPerson

	rows, errRows := r.db.QueryBuilder(ctx, builder.Where(constraints))
	if errRows != nil {
		return nil, 0, r.db.DBErr(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			banNet   domain.BannedCIDRPerson
			sourceID int64
			targetID int64
			cidr     *net.IPNet
		)

		if errScan := rows.
			Scan(&banNet.NetID, &cidr, &banNet.Origin,
				&banNet.CreatedOn, &banNet.UpdatedOn, &banNet.Reason, &banNet.ReasonText,
				&banNet.ValidUntil, &banNet.Deleted, &banNet.Note, &banNet.UnbanReasonText,
				&banNet.IsEnabled, &targetID, &sourceID, &banNet.AppealState,
				&banNet.SourceTarget.SourcePersonaname, &banNet.SourceTarget.SourceAvatarhash,
				&banNet.SourceTarget.TargetPersonaname, &banNet.SourceTarget.TargetAvatarhash,
				&banNet.CommunityBanned, &banNet.VacBans, &banNet.GameBans); errScan != nil {
			return nil, 0, r.db.DBErr(errScan)
		}

		banNet.CIDR = cidr.String()
		banNet.SourceID = steamid.New(sourceID)
		banNet.TargetID = steamid.New(targetID)

		nets = append(nets, banNet)
	}

	count, errCount := r.db.GetCount(ctx, r.db.
		Builder().
		Select("COUNT(b.net_id)").
		From("ban_net b").
		Where(constraints))

	if errCount != nil {
		if errors.Is(errCount, domain.ErrNoResult) {
			return []domain.BannedCIDRPerson{}, 0, nil
		}

		return nil, count, r.db.DBErr(errCount)
	}

	if nets == nil {
		return []domain.BannedCIDRPerson{}, 0, nil
	}

	return nets, count, nil
}

func (r *banRepository) updateBanNet(ctx context.Context, banNet *domain.BanCIDR) error {
	banNet.UpdatedOn = time.Now()

	query := r.db.
		Builder().
		Update("ban_net").
		Set("cidr", banNet.CIDR).
		Set("origin", banNet.Origin).
		Set("updated_on", banNet.UpdatedOn).
		Set("reason", banNet.Reason).
		Set("reason_text", banNet.ReasonText).
		Set("valid_until", banNet.ValidUntil).
		Set("deleted", banNet.Deleted).
		Set("note", banNet.Note).
		Set("unban_reason_text", banNet.UnbanReasonText).
		Set("is_enabled", banNet.IsEnabled).
		Set("target_id", banNet.TargetID.Int64()).
		Set("source_id", banNet.SourceID.Int64()).
		Set("appeal_state", banNet.AppealState).
		Where(sq.Eq{"net_id": banNet.NetID})

	return r.db.DBErr(r.db.ExecUpdateBuilder(ctx, query))
}

func (r *banRepository) insertBanNet(ctx context.Context, banNet *domain.BanCIDR) error {
	query, args, errQueryArgs := r.db.
		Builder().
		Insert("ban_net").
		Columns("cidr", "origin", "created_on", "updated_on", "reason", "reason_text", "valid_until",
			"deleted", "note", "unban_reason_text", "is_enabled", "target_id", "source_id", "appeal_state").
		Values(banNet.CIDR, banNet.Origin, banNet.CreatedOn, banNet.UpdatedOn, banNet.Reason, banNet.ReasonText,
			banNet.ValidUntil, banNet.Deleted, banNet.Note, banNet.UnbanReasonText, banNet.IsEnabled,
			banNet.TargetID.Int64(), banNet.SourceID.Int64(), banNet.AppealState).
		Suffix("RETURNING net_id").
		ToSql()
	if errQueryArgs != nil {
		return r.db.DBErr(errQueryArgs)
	}

	return r.db.DBErr(r.db.QueryRow(ctx, query, args...).Scan(&banNet.NetID))
}

func (r *banRepository) SaveBanNet(ctx context.Context, banNet *domain.BanCIDR) error {
	if banNet.NetID > 0 {
		return r.updateBanNet(ctx, banNet)
	}

	return r.insertBanNet(ctx, banNet)
}

func (r *banRepository) DropBanNet(ctx context.Context, banNet *domain.BanCIDR) error {
	query := r.db.
		Builder().
		Delete("ban_net").
		Where(sq.Eq{"net_id": banNet.NetID})

	if errExec := r.db.ExecDeleteBuilder(ctx, query); errExec != nil {
		return r.db.DBErr(errExec)
	}

	banNet.NetID = 0

	return nil
}

func (r *banRepository) GetExpiredNetBans(ctx context.Context) ([]domain.BanCIDR, error) {
	query := r.db.
		Builder().
		Select("net_id", "cidr", "origin", "created_on", "updated_on", "reason_text", "valid_until",
			"deleted", "note", "unban_reason_text", "is_enabled", "target_id", "source_id", "reason", "appeal_state").
		From("ban_net").
		Where(sq.Lt{"valid_until": time.Now()})

	var bans []domain.BanCIDR

	rows, errQuery := r.db.QueryBuilder(ctx, query)
	if errQuery != nil {
		return nil, r.db.DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			banNet   domain.BanCIDR
			targetID int64
			sourceID int64
			cidr     pgtype.CIDR
		)

		if errScan := rows.
			Scan(&banNet.NetID, &cidr, &banNet.Origin, &banNet.CreatedOn,
				&banNet.UpdatedOn, &banNet.ReasonText, &banNet.ValidUntil, &banNet.Deleted, &banNet.Note,
				&banNet.UnbanReasonText, &banNet.IsEnabled, &targetID, &sourceID,
				&banNet.Reason, &banNet.AppealState); errScan != nil {
			return nil, r.db.DBErr(errScan)
		}

		banNet.CIDR = cidr.IPNet.String()
		banNet.TargetID = steamid.New(targetID)
		banNet.SourceID = steamid.New(sourceID)

		bans = append(bans, banNet)
	}

	if bans == nil {
		return []domain.BanCIDR{}, nil
	}

	return bans, nil
}

func (r *banRepository) GetExpiredASNBans(ctx context.Context) ([]domain.BanASN, error) {
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

func (r *banRepository) GetBanNetByID(ctx context.Context, netID int64, banNet *domain.BanCIDR) error {
	const query = `
		SELECT net_id, cidr, origin, created_on, updated_on, reason, reason_text, valid_until, deleted, 
		       note, unban_reason_text, is_enabled, target_id, source_id, appeal_state
		FROM ban_net
		WHERE deleted = false AND net_id = $1`

	var (
		sourceID int64
		targetID int64
		cidr     *net.IPNet
	)

	errQuery := r.db.
		QueryRow(ctx, query, netID).
		Scan(&banNet.NetID, &cidr, &banNet.Origin,
			&banNet.CreatedOn, &banNet.UpdatedOn, &banNet.Reason, &banNet.ReasonText,
			&banNet.ValidUntil, &banNet.Deleted, &banNet.Note, &banNet.UnbanReasonText,
			&banNet.IsEnabled, &targetID, &sourceID, &banNet.AppealState)
	if errQuery != nil {
		return r.db.DBErr(errQuery)
	}

	banNet.CIDR = cidr.String()
	banNet.SourceID = steamid.New(sourceID)
	banNet.TargetID = steamid.New(targetID)

	return nil
}

func (r *banRepository) DropBan(ctx context.Context, ban *domain.BanSteam, hardDelete bool) error {
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

func (r *banRepository) getBanByColumn(ctx context.Context, column string, identifier any, person *domain.BannedSteamPerson, deletedOk bool) error {
	whereClauses := sq.And{
		sq.Eq{fmt.Sprintf("s.%s", column): identifier}, // valid columns are immutable
	}

	if !deletedOk {
		whereClauses = append(whereClauses, sq.Eq{"s.deleted": false})
	} else {
		whereClauses = append(whereClauses, sq.Gt{"s.valid_until": time.Now()})
	}

	query := r.db.
		Builder().
		Select(
			"s.ban_id", "s.target_id", "s.source_id", "s.ban_type", "s.reason",
			"s.reason_text", "s.note", "s.origin", "s.valid_until", "s.created_on", "s.updated_on", "s.include_friends",
			"s.deleted", "case WHEN s.report_id is null THEN 0 ELSE s.report_id END",
			"s.unban_reason_text", "s.is_enabled", "s.appeal_state", "s.last_ip",
			"s.personaname as source_personaname", "s.avatarhash",
			"t.personaname as target_personaname", "t.avatarhash", "t.community_banned", "t.vac_bans", "t.game_bans",
		).
		From("ban s").
		LeftJoin("person s on s.steam_id = s.source_id").
		LeftJoin("person t on t.steam_id = s.target_id").
		Where(whereClauses).
		OrderBy("s.created_on DESC").
		Limit(1)

	row, errQuery := r.db.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return r.db.DBErr(errQuery)
	}

	var (
		sourceID int64
		targetID int64
	)

	if errScan := row.
		Scan(&person.BanID, &targetID, &sourceID, &person.BanType, &person.Reason,
			&person.ReasonText, &person.Note, &person.Origin, &person.ValidUntil, &person.CreatedOn,
			&person.UpdatedOn, &person.IncludeFriends, &person.Deleted, &person.ReportID, &person.UnbanReasonText,
			&person.IsEnabled, &person.AppealState, &person.LastIP,
			&person.SourceTarget.SourcePersonaname, &person.SourceTarget.SourceAvatarhash,
			&person.SourceTarget.TargetPersonaname, &person.SourceTarget.TargetAvatarhash,
			&person.CommunityBanned, &person.VacBans, &person.GameBans,
		); errScan != nil {
		return r.db.DBErr(errScan)
	}

	person.SourceID = steamid.New(sourceID)
	person.TargetID = steamid.New(targetID)

	return nil
}

func (r *banRepository) GetBanBySteamID(ctx context.Context, sid64 steamid.SID64, bannedPerson *domain.BannedSteamPerson, deletedOk bool) error {
	return r.getBanByColumn(ctx, "target_id", sid64, bannedPerson, deletedOk)
}

func (r *banRepository) GetBanByBanID(ctx context.Context, banID int64, bannedPerson *domain.BannedSteamPerson, deletedOk bool) error {
	return r.getBanByColumn(ctx, "ban_id", banID, bannedPerson, deletedOk)
}

func (r *banRepository) GetBanByLastIP(ctx context.Context, lastIP net.IP, bannedPerson *domain.BannedSteamPerson, deletedOk bool) error {
	return r.getBanByColumn(ctx, "last_ip", fmt.Sprintf("::ffff:%s", lastIP.String()), bannedPerson, deletedOk)
}

// SaveBan will insert or update the ban record
// New records will have the Ban.BanID set automatically.
func (r *banRepository) SaveBan(ctx context.Context, ban *domain.BanSteam) error {
	// Ensure the foreign keys are satisfied
	targetPerson := domain.NewPerson(ban.TargetID)
	if errGetPerson := r.pu.GetOrCreatePersonBySteamID(ctx, ban.TargetID, &targetPerson); errGetPerson != nil {
		return errors.Join(errGetPerson, domain.ErrPersonTarget)
	}

	authorPerson := domain.NewPerson(ban.SourceID)
	if errGetAuthor := r.pu.GetOrCreatePersonBySteamID(ctx, ban.SourceID, &authorPerson); errGetAuthor != nil {
		return errors.Join(errGetAuthor, domain.ErrPersonSource)
	}

	ban.UpdatedOn = time.Now()
	if ban.BanID > 0 {
		return r.updateBan(ctx, ban)
	}

	ban.CreatedOn = ban.UpdatedOn

	existing := domain.NewBannedPerson()

	errGetBan := r.GetBanBySteamID(ctx, ban.TargetID, &existing, false)
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

func (r *banRepository) insertBan(ctx context.Context, ban *domain.BanSteam) error {
	const query = `
		INSERT INTO ban (target_id, source_id, ban_type, reason, reason_text, note, valid_until, 
		                 created_on, updated_on, origin, report_id, appeal_state, include_friends, last_ip)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, case WHEN $11 = 0 THEN null ELSE $11 END, $12, $13, $14)
		RETURNING ban_id`

	errQuery := r.db.
		QueryRow(ctx, query, ban.TargetID.Int64(), ban.SourceID.Int64(), ban.BanType, ban.Reason, ban.ReasonText,
			ban.Note, ban.ValidUntil, ban.CreatedOn, ban.UpdatedOn, ban.Origin, ban.ReportID, ban.AppealState,
			ban.IncludeFriends, &ban.LastIP).
		Scan(&ban.BanID)

	if errQuery != nil {
		return r.db.DBErr(errQuery)
	}

	return nil
}

func (r *banRepository) updateBan(ctx context.Context, ban *domain.BanSteam) error {
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
		Where(sq.Eq{"ban_id": ban.BanID})

	return r.db.DBErr(r.db.ExecUpdateBuilder(ctx, query))
}

func (r *banRepository) GetExpiredBans(ctx context.Context) ([]domain.BanSteam, error) {
	query := r.db.
		Builder().
		Select("ban_id", "target_id", "source_id", "ban_type", "reason", "reason_text", "note",
			"valid_until", "origin", "created_on", "updated_on", "deleted", "case WHEN report_id is null THEN 0 ELSE report_id END",
			"unban_reason_text", "is_enabled", "appeal_state", "include_friends").
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
			&ban.IsEnabled, &ban.AppealState, &ban.IncludeFriends); errScan != nil {
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

// GetBansSteam returns all bans that fit the filter criteria passed in.
func (r *banRepository) GetBansSteam(ctx context.Context, filter domain.SteamBansQueryFilter) ([]domain.BannedSteamPerson, int64, error) {
	builder := r.db.
		Builder().
		Select("s.ban_id", "s.target_id", "s.source_id", "s.ban_type", "s.reason",
			"s.reason_text", "s.note", "s.origin", "s.valid_until", "s.created_on", "s.updated_on", "s.include_friends",
			"s.deleted", "case WHEN s.report_id is null THEN 0 ELSE s.report_id END",
			"s.unban_reason_text", "s.is_enabled", "s.appeal_state",
			"s.personaname as source_personaname", "s.avatarhash",
			"t.personaname as target_personaname", "t.avatarhash", "t.community_banned", "t.vac_bans", "t.game_bans").
		From("ban s").
		JoinClause("LEFT JOIN person s on s.steam_id = s.source_id").
		JoinClause("LEFT JOIN person t on t.steam_id = s.target_id")

	var ands sq.And

	if !filter.Deleted {
		ands = append(ands, sq.Eq{"s.deleted": false})
	}

	if filter.Reason > 0 {
		ands = append(ands, sq.Eq{"s.reason": filter.Reason})
	}

	if filter.PermanentOnly {
		ands = append(ands, sq.Gt{"s.valid_until": time.Now()})
	}

	if filter.TargetID != "" {
		targetID, errTargetID := filter.TargetID.SID64(ctx)
		if errTargetID != nil {
			return nil, 0, errors.Join(errTargetID, domain.ErrTargetID)
		}

		ands = append(ands, sq.Eq{"s.target_id": targetID.Int64()})
	}

	if filter.SourceID != "" {
		sourceID, errSourceID := filter.SourceID.SID64(ctx)
		if errSourceID != nil {
			return nil, 0, errors.Join(errSourceID, domain.ErrSourceID)
		}

		ands = append(ands, sq.Eq{"s.source_id": sourceID.Int64()})
	}

	if filter.IncludeFriendsOnly {
		ands = append(ands, sq.Eq{"s.include_friends": true})
	}

	if filter.AppealState > domain.AnyState {
		ands = append(ands, sq.Eq{"s.appeal_state": filter.AppealState})
	}

	if len(ands) > 0 {
		builder = builder.Where(ands)
	}

	builder = filter.QueryFilter.ApplySafeOrder(builder, map[string][]string{
		"b.": {
			"ban_id", "target_id", "source_id", "ban_type", "reason",
			"origin", "valid_until", "created_on", "updated_on", "include_friends",
			"deleted", "report_id", "is_enabled", "appeal_state",
		},
		"s.": {"source_personaname"},
		"t.": {"target_personaname", "community_banned", "vac_bans", "game_bans"},
	}, "ban_id")

	builder = filter.QueryFilter.ApplyLimitOffsetDefault(builder)

	count, errCount := r.db.GetCount(ctx, r.db.
		Builder().
		Select("COUNT(s.ban_id)").
		From("ban s").
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
				&person.UpdatedOn, &person.IncludeFriends, &person.Deleted, &person.ReportID, &person.UnbanReasonText,
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

func (r *banRepository) GetBansOlderThan(ctx context.Context, filter domain.QueryFilter, since time.Time) ([]domain.BanSteam, error) {
	query := r.db.
		Builder().
		Select("s.ban_id", "s.target_id", "s.source_id", "s.ban_type", "s.reason",
			"s.reason_text", "s.note", "s.origin", "s.valid_until", "s.created_on", "s.updated_on", "s.deleted",
			"case WHEN s.report_id is null THEN 0 ELSE s.report_id END", "s.unban_reason_text", "s.is_enabled",
			"s.appeal_state", "s.include_friends").
		From("ban s").
		Where(sq.And{sq.Lt{"updated_on": since}, sq.Eq{"deleted": false}})

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
			&ban.IsEnabled, &ban.AppealState, &ban.AppealState); errQuery != nil {
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

func (r *banRepository) GetBanASN(ctx context.Context, asNum int64, banASN *domain.BanASN) error {
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

func (r *banRepository) GetBansASN(ctx context.Context, filter domain.ASNBansQueryFilter) ([]domain.BannedASNPerson, int64, error) {
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

	if filter.TargetID != "" {
		targetID, errTargetID := filter.TargetID.SID64(ctx)
		if errTargetID != nil {
			return nil, 0, errors.Join(errTargetID, domain.ErrTargetID)
		}

		constraints = append(constraints, sq.Eq{"b.target_id": targetID.Int64()})
	}

	if filter.SourceID != "" {
		sourceID, errSourceID := filter.SourceID.SID64(ctx)
		if errSourceID != nil {
			return nil, 0, errors.Join(errSourceID, domain.ErrSourceID)
		}

		constraints = append(constraints, sq.Eq{"b.source_id": sourceID.Int64()})
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

func (r *banRepository) SaveBanASN(ctx context.Context, banASN *domain.BanASN) error {
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

func (r *banRepository) DropBanASN(ctx context.Context, banASN *domain.BanASN) error {
	banASN.Deleted = true

	return r.SaveBanASN(ctx, banASN)
}
