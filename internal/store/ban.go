package store

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

var (
	ErrPersonSource = errors.New("failed to load source person")
	ErrPersonTarget = errors.New("failed to load target person")
	ErrGetBan       = errors.New("failed to load existing ban")
)

func (s Stores) DropBan(ctx context.Context, ban *domain.BanSteam, hardDelete bool) error {
	if hardDelete {
		if errExec := s.Exec(ctx, `DELETE FROM ban WHERE ban_id = $1`, ban.BanID); errExec != nil {
			return errs.DBErr(errExec)
		}

		ban.BanID = 0

		return nil
	} else {
		ban.Deleted = true

		return s.updateBan(ctx, ban)
	}
}

func (s Stores) getBanByColumn(ctx context.Context, column string, identifier any, person *domain.BannedSteamPerson, deletedOk bool) error {
	whereClauses := sq.And{
		sq.Eq{fmt.Sprintf("s.%s", column): identifier}, // valid columns are immutable
	}

	if !deletedOk {
		whereClauses = append(whereClauses, sq.Eq{"s.deleted": false})
	} else {
		whereClauses = append(whereClauses, sq.Gt{"s.valid_until": time.Now()})
	}

	query := s.
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

	row, errQuery := s.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return errs.DBErr(errQuery)
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
		return errs.DBErr(errScan)
	}

	person.SourceID = steamid.New(sourceID)
	person.TargetID = steamid.New(targetID)

	return nil
}

func (s Stores) GetBanBySteamID(ctx context.Context, sid64 steamid.SID64, bannedPerson *domain.BannedSteamPerson, deletedOk bool) error {
	return s.getBanByColumn(ctx, "target_id", sid64, bannedPerson, deletedOk)
}

func (s Stores) GetBanByBanID(ctx context.Context, banID int64, bannedPerson *domain.BannedSteamPerson, deletedOk bool) error {
	return s.getBanByColumn(ctx, "ban_id", banID, bannedPerson, deletedOk)
}

func (s Stores) GetBanByLastIP(ctx context.Context, lastIP net.IP, bannedPerson *domain.BannedSteamPerson, deletedOk bool) error {
	return s.getBanByColumn(ctx, "last_ip", fmt.Sprintf("::ffff:%s", lastIP.String()), bannedPerson, deletedOk)
}

// SaveBan will insert or update the ban record
// New records will have the Ban.BanID set automatically.
func (s Stores) SaveBan(ctx context.Context, ban *domain.BanSteam) error {
	// Ensure the foreign keys are satisfied
	targetPerson := domain.NewPerson(ban.TargetID)
	if errGetPerson := s.GetOrCreatePersonBySteamID(ctx, ban.TargetID, &targetPerson); errGetPerson != nil {
		return errors.Join(errGetPerson, ErrPersonTarget)
	}

	authorPerson := domain.NewPerson(ban.SourceID)
	if errGetAuthor := s.GetOrCreatePersonBySteamID(ctx, ban.SourceID, &authorPerson); errGetAuthor != nil {
		return errors.Join(errGetAuthor, ErrPersonSource)
	}

	ban.UpdatedOn = time.Now()
	if ban.BanID > 0 {
		return s.updateBan(ctx, ban)
	}

	ban.CreatedOn = ban.UpdatedOn

	existing := domain.NewBannedPerson()

	errGetBan := s.GetBanBySteamID(ctx, ban.TargetID, &existing, false)
	if errGetBan != nil {
		if !errors.Is(errGetBan, errs.ErrNoResult) {
			return errors.Join(errGetBan, ErrGetBan)
		}
	} else {
		if ban.BanType <= existing.BanType {
			return errs.ErrDuplicate
		}
	}

	ban.LastIP = s.GetPlayerMostRecentIP(ctx, ban.TargetID)

	return s.insertBan(ctx, ban)
}

func (s Stores) insertBan(ctx context.Context, ban *domain.BanSteam) error {
	const query = `
		INSERT INTO ban (target_id, source_id, ban_type, reason, reason_text, note, valid_until, 
		                 created_on, updated_on, origin, report_id, appeal_state, include_friends, last_ip)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, case WHEN $11 = 0 THEN null ELSE $11 END, $12, $13, $14)
		RETURNING ban_id`

	errQuery := s.
		QueryRow(ctx, query, ban.TargetID.Int64(), ban.SourceID.Int64(), ban.BanType, ban.Reason, ban.ReasonText,
			ban.Note, ban.ValidUntil, ban.CreatedOn, ban.UpdatedOn, ban.Origin, ban.ReportID, ban.AppealState,
			ban.IncludeFriends, &ban.LastIP).
		Scan(&ban.BanID)

	if errQuery != nil {
		return errs.DBErr(errQuery)
	}

	return nil
}

func (s Stores) updateBan(ctx context.Context, ban *domain.BanSteam) error {
	var reportID *int64
	if ban.ReportID > 0 {
		reportID = &ban.ReportID
	}

	query := s.
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

	return errs.DBErr(s.ExecUpdateBuilder(ctx, query))
}

func (s Stores) GetExpiredBans(ctx context.Context) ([]domain.BanSteam, error) {
	query := s.
		Builder().
		Select("ban_id", "target_id", "source_id", "ban_type", "reason", "reason_text", "note",
			"valid_until", "origin", "created_on", "updated_on", "deleted", "case WHEN report_id is null THEN 0 ELSE report_id END",
			"unban_reason_text", "is_enabled", "appeal_state", "include_friends").
		From("ban").
		Where(sq.And{sq.Lt{"valid_until": time.Now()}, sq.Eq{"deleted": false}})

	rows, errQuery := s.QueryBuilder(ctx, query)
	if errQuery != nil {
		return nil, errs.DBErr(errQuery)
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
			return nil, errors.Join(errScan, ErrScanResult)
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

func (s Stores) GetAppealsByActivity(ctx context.Context, opts domain.AppealQueryFilter) ([]domain.AppealOverview, int64, error) {
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
			return nil, 0, errors.Join(errAuthorID, errs.ErrSourceID)
		}

		constraints = append(constraints, sq.Eq{"s.source_id": authorID.Int64()})
	}

	if opts.TargetID != "" {
		targetID, errTargetID := opts.TargetID.SID64(ctx)
		if errTargetID != nil {
			return nil, 0, errors.Join(errTargetID, errs.ErrTargetID)
		}

		constraints = append(constraints, sq.Eq{"s.target_id": targetID.Int64()})
	}

	counterQuery := s.
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

	count, errCount := getCount(ctx, s, counterQuery)
	if errCount != nil {
		return nil, 0, errs.DBErr(errCount)
	}

	builder := s.
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

	rows, errQuery := s.QueryBuilder(ctx, builder)
	if errQuery != nil {
		return nil, 0, errs.DBErr(errQuery)
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
			return nil, 0, errors.Join(errScan, ErrScanResult)
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

// GetBansSteam returns all bans that fit the filter criteria passed in.
func (s Stores) GetBansSteam(ctx context.Context, filter domain.SteamBansQueryFilter) ([]domain.BannedSteamPerson, int64, error) {
	builder := s.
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
			return nil, 0, errors.Join(errTargetID, errs.ErrTargetID)
		}

		ands = append(ands, sq.Eq{"s.target_id": targetID.Int64()})
	}

	if filter.SourceID != "" {
		sourceID, errSourceID := filter.SourceID.SID64(ctx)
		if errSourceID != nil {
			return nil, 0, errors.Join(errSourceID, errs.ErrSourceID)
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

	count, errCount := getCount(ctx, s, s.
		Builder().
		Select("COUNT(s.ban_id)").
		From("ban s").
		Where(ands))
	if errCount != nil {
		return nil, 0, errs.DBErr(errCount)
	}

	if count == 0 {
		return []domain.BannedSteamPerson{}, 0, nil
	}

	var bans []domain.BannedSteamPerson

	rows, errQuery := s.QueryBuilder(ctx, builder)
	if errQuery != nil {
		return nil, 0, errs.DBErr(errQuery)
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
			return nil, 0, errs.DBErr(errScan)
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

func (s Stores) GetBansOlderThan(ctx context.Context, filter domain.QueryFilter, since time.Time) ([]domain.BanSteam, error) {
	query := s.
		Builder().
		Select("s.ban_id", "s.target_id", "s.source_id", "s.ban_type", "s.reason",
			"s.reason_text", "s.note", "s.origin", "s.valid_until", "s.created_on", "s.updated_on", "s.deleted",
			"case WHEN s.report_id is null THEN 0 ELSE s.report_id END", "s.unban_reason_text", "s.is_enabled",
			"s.appeal_state", "s.include_friends").
		From("ban s").
		Where(sq.And{sq.Lt{"updated_on": since}, sq.Eq{"deleted": false}})

	rows, errQuery := s.QueryBuilder(ctx, filter.ApplyLimitOffsetDefault(query))
	if errQuery != nil {
		return nil, errs.DBErr(errQuery)
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
			return nil, errors.Join(errQuery, ErrScanResult)
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

func (s Stores) SaveBanMessage(ctx context.Context, message *domain.BanAppealMessage) error {
	var err error
	if message.BanMessageID > 0 {
		err = s.updateBanMessage(ctx, message)
	} else {
		err = s.insertBanMessage(ctx, message)
	}

	bannedPerson := domain.NewBannedPerson()
	if errBan := s.GetBanByBanID(ctx, message.BanID, &bannedPerson, true); errBan != nil {
		return errs.ErrNoResult
	}

	bannedPerson.UpdatedOn = time.Now()

	if errUpdate := s.updateBan(ctx, &bannedPerson.BanSteam); errUpdate != nil {
		return errUpdate
	}

	return err
}

func (s Stores) updateBanMessage(ctx context.Context, message *domain.BanAppealMessage) error {
	message.UpdatedOn = time.Now()

	query := s.
		Builder().
		Update("ban_appeal").
		Set("deleted", message.Deleted).
		Set("author_id", message.AuthorID.Int64()).
		Set("updated_on", message.UpdatedOn).
		Set("message_md", message.MessageMD).
		Where(sq.Eq{"ban_message_id": message.BanMessageID})

	if errQuery := s.ExecUpdateBuilder(ctx, query); errQuery != nil {
		return errs.DBErr(errQuery)
	}

	return nil
}

func (s Stores) insertBanMessage(ctx context.Context, message *domain.BanAppealMessage) error {
	const query = `
	INSERT INTO ban_appeal (
		ban_id, author_id, message_md, deleted, created_on, updated_on
	)
	VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING ban_message_id
	`

	if errQuery := s.QueryRow(ctx, query,
		message.BanID,
		message.AuthorID.Int64(),
		message.MessageMD,
		message.Deleted,
		message.CreatedOn,
		message.UpdatedOn,
	).Scan(&message.BanMessageID); errQuery != nil {
		return errs.DBErr(errQuery)
	}

	return nil
}

func (s Stores) GetBanMessages(ctx context.Context, banID int64) ([]domain.BanAppealMessage, error) {
	query := s.
		Builder().
		Select("a.ban_message_id", "a.ban_id", "a.author_id", "a.message_md", "a.deleted",
			"a.created_on", "a.updated_on", "p.avatarhash", "p.personaname", "p.permission_level").
		From("ban_appeal a").
		LeftJoin("person p ON a.author_id = p.steam_id").
		Where(sq.And{sq.Eq{"a.deleted": false}, sq.Eq{"a.ban_id": banID}}).
		OrderBy("a.created_on")

	rows, errQuery := s.QueryBuilder(ctx, query)
	if errQuery != nil {
		if errors.Is(errs.DBErr(errQuery), errs.ErrNoResult) {
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
			return nil, errs.DBErr(errQuery)
		}

		msg.AuthorID = steamid.New(authorID)

		messages = append(messages, msg)
	}

	if messages == nil {
		return []domain.BanAppealMessage{}, nil
	}

	return messages, nil
}

func (s Stores) GetBanMessageByID(ctx context.Context, banMessageID int, message *domain.BanAppealMessage) error {
	query := s.
		Builder().
		Select("a.ban_message_id", "a.ban_id", "a.author_id", "a.message_md", "a.deleted", "a.created_on",
			"a.updated_on", "p.avatarhash", "p.personaname", "p.permission_level").
		From("ban_appeal a").
		LeftJoin("person p ON a.author_id = p.steam_id").
		Where(sq.Eq{"a.ban_message_id": banMessageID})

	var authorID int64

	row, errQuery := s.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return errs.DBErr(errQuery)
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
		return errs.DBErr(errScan)
	}

	message.AuthorID = steamid.New(authorID)

	return nil
}

func (s Stores) DropBanMessage(ctx context.Context, message *domain.BanAppealMessage) error {
	query := s.
		Builder().
		Update("ban_appeal").
		Set("deleted", true).
		Where(sq.Eq{"ban_message_id": message.BanMessageID})

	if errExec := s.ExecUpdateBuilder(ctx, query); errExec != nil {
		return errs.DBErr(errExec)
	}

	message.Deleted = true

	return nil
}

func (s Stores) GetBanGroup(ctx context.Context, groupID steamid.GID, banGroup *domain.BanGroup) error {
	query := s.
		Builder().
		Select("ban_group_id", "source_id", "target_id", "group_name", "is_enabled", "deleted",
			"note", "unban_reason_text", "origin", "created_on", "updated_on", "valid_until", "appeal_state", "group_id").
		From("ban_group").
		Where(sq.And{sq.Eq{"group_id": groupID.Int64()}, sq.Eq{"deleted": false}})

	var (
		sourceID   int64
		targetID   int64
		newGroupID int64
	)

	row, errQuery := s.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return errs.DBErr(errQuery)
	}

	if errScan := row.Scan(&banGroup.BanGroupID, &sourceID, &targetID, &banGroup.GroupName, &banGroup.IsEnabled,
		&banGroup.Deleted, &banGroup.Note, &banGroup.UnbanReasonText, &banGroup.Origin, &banGroup.CreatedOn,
		&banGroup.UpdatedOn, &banGroup.ValidUntil, &banGroup.AppealState, &newGroupID); errScan != nil {
		return errs.DBErr(errScan)
	}

	banGroup.SourceID = steamid.New(sourceID)
	banGroup.TargetID = steamid.New(targetID)
	banGroup.GroupID = steamid.NewGID(newGroupID)

	return nil
}

func (s Stores) GetBanGroupByID(ctx context.Context, banGroupID int64, banGroup *domain.BanGroup) error {
	query := s.
		Builder().
		Select("ban_group_id", "source_id", "target_id", "group_name", "is_enabled", "deleted",
			"note", "unban_reason_text", "origin", "created_on", "updated_on", "valid_until", "appeal_state", "group_id").
		From("ban_group").
		Where(sq.And{sq.Eq{"ban_group_id": banGroupID}, sq.Eq{"is_enabled": true}, sq.Eq{"deleted": false}})

	var (
		groupID  int64
		sourceID int64
		targetID int64
	)

	row, errQuery := s.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return errs.DBErr(errQuery)
	}

	if errScan := row.Scan(
		&banGroup.BanGroupID,
		&sourceID,
		&targetID,
		&banGroup.GroupName,
		&banGroup.IsEnabled,
		&banGroup.Deleted,
		&banGroup.Note,
		&banGroup.UnbanReasonText,
		&banGroup.Origin,
		&banGroup.CreatedOn,
		&banGroup.UpdatedOn,
		&banGroup.ValidUntil,
		&banGroup.AppealState,
		&groupID); errScan != nil {
		return errs.DBErr(errScan)
	}

	banGroup.SourceID = steamid.New(sourceID)
	banGroup.TargetID = steamid.New(targetID)
	banGroup.GroupID = steamid.NewGID(groupID)

	return nil
}

func (s Stores) GetBanGroups(ctx context.Context, filter domain.GroupBansQueryFilter) ([]domain.BannedGroupPerson, int64, error) {
	builder := s.
		Builder().
		Select("s.ban_group_id", "s.source_id", "s.target_id", "s.group_name", "s.is_enabled", "s.deleted",
			"s.note", "s.unban_reason_text", "s.origin", "s.created_on", "s.updated_on", "s.valid_until",
			"s.appeal_state", "s.group_id",
			"s.personaname as source_personaname", "s.avatarhash",
			"t.personaname as target_personaname", "t.avatarhash", "t.community_banned", "t.vac_bans", "t.game_bans").
		From("ban_group s").
		LeftJoin("person s ON s.steam_id = s.source_id").
		LeftJoin("person t ON t.steam_id = s.target_id")

	var constraints sq.And

	if !filter.Deleted {
		constraints = append(constraints, sq.Eq{"s.deleted": false})
	}

	if filter.Reason > 0 {
		constraints = append(constraints, sq.Eq{"s.reason": filter.Reason})
	}

	if filter.PermanentOnly {
		constraints = append(constraints, sq.Gt{"s.valid_until": time.Now()})
	}

	if filter.GroupID != "" {
		gid := steamid.NewGID(filter.GroupID)
		if !gid.Valid() {
			return nil, 0, steamid.ErrInvalidGID
		}

		constraints = append(constraints, sq.Eq{"s.group_id": gid.Int64()})
	}

	if filter.TargetID != "" {
		targetID, errTargetID := filter.TargetID.SID64(ctx)
		if errTargetID != nil {
			return nil, 0, errors.Join(errTargetID, errs.ErrTargetID)
		}

		constraints = append(constraints, sq.Eq{"s.target_id": targetID.Int64()})
	}

	if filter.SourceID != "" {
		sourceID, errSourceID := filter.SourceID.SID64(ctx)
		if errSourceID != nil {
			return nil, 0, errors.Join(errSourceID, errs.ErrSourceID)
		}

		constraints = append(constraints, sq.Eq{"s.source_id": sourceID.Int64()})
	}

	builder = filter.QueryFilter.ApplySafeOrder(builder, map[string][]string{
		"b.": {
			"ban_group_id", "source_id", "target_id", "group_name", "is_enabled", "deleted",
			"origin", "created_on", "updated_on", "valid_until", "appeal_state", "group_id",
		},
		"s.": {"source_personaname"},
		"t.": {"target_personaname", "community_banned", "vac_bans", "game_bans"},
	}, "ban_group_id")

	builder = filter.ApplyLimitOffsetDefault(builder).Where(constraints)

	rows, errRows := s.QueryBuilder(ctx, builder)
	if errRows != nil {
		if errors.Is(errRows, errs.ErrNoResult) {
			return []domain.BannedGroupPerson{}, 0, nil
		}

		return nil, 0, errs.DBErr(errRows)
	}

	defer rows.Close()

	var groups []domain.BannedGroupPerson

	for rows.Next() {
		var (
			group    domain.BannedGroupPerson
			groupID  int64
			sourceID int64
			targetID int64
		)

		if errScan := rows.Scan(
			&group.BanGroupID,
			&sourceID,
			&targetID,
			&group.GroupName,
			&group.IsEnabled,
			&group.Deleted,
			&group.Note,
			&group.UnbanReasonText,
			&group.Origin,
			&group.CreatedOn,
			&group.UpdatedOn,
			&group.ValidUntil,
			&group.AppealState,
			&groupID,
			&group.SourceTarget.SourcePersonaname, &group.SourceTarget.SourceAvatarhash,
			&group.SourceTarget.TargetPersonaname, &group.SourceTarget.TargetAvatarhash,
			&group.CommunityBanned, &group.VacBans, &group.GameBans,
		); errScan != nil {
			return nil, 0, errs.DBErr(errScan)
		}

		group.SourceID = steamid.New(sourceID)
		group.TargetID = steamid.New(targetID)
		group.GroupID = steamid.NewGID(groupID)

		groups = append(groups, group)
	}

	count, errCount := getCount(ctx, s, s.
		Builder().
		Select("s.ban_group_id").
		From("ban_group s").
		Where(constraints))
	if errCount != nil {
		if errors.Is(errCount, errs.ErrNoResult) {
			return []domain.BannedGroupPerson{}, 0, nil
		}

		return nil, 0, errs.DBErr(errCount)
	}

	if groups == nil {
		groups = []domain.BannedGroupPerson{}
	}

	return groups, count, nil
}

func (s Stores) GetMembersList(ctx context.Context, parentID int64, list *domain.MembersList) error {
	row, err := s.QueryRowBuilder(ctx, s.
		Builder().
		Select("members_id", "parent_id", "members", "created_on", "updated_on").
		From("members").
		Where(sq.Eq{"parent_id": parentID}))
	if err != nil {
		return errs.DBErr(err)
	}

	return errs.DBErr(row.Scan(&list.MembersID, &list.ParentID, &list.Members, &list.CreatedOn, &list.UpdatedOn))
}

func (s Stores) SaveMembersList(ctx context.Context, list *domain.MembersList) error {
	if list.MembersID > 0 {
		list.UpdatedOn = time.Now()

		const update = `UPDATE members SET members = $2::jsonb, updated_on = $3 WHERE members_id = $1`

		return errs.DBErr(s.Exec(ctx, update, list.MembersID, list.Members, list.UpdatedOn))
	} else {
		const insert = `INSERT INTO members (parent_id, members, created_on, updated_on) 
		VALUES ($1, $2::jsonb, $3, $4) RETURNING members_id`

		return errs.DBErr(s.QueryRow(ctx, insert, list.ParentID, list.Members, list.CreatedOn, list.UpdatedOn).Scan(&list.MembersID))
	}
}

func (s Stores) SaveBanGroup(ctx context.Context, banGroup *domain.BanGroup) error {
	if banGroup.BanGroupID > 0 {
		return s.updateBanGroup(ctx, banGroup)
	}

	return s.insertBanGroup(ctx, banGroup)
}

func (s Stores) insertBanGroup(ctx context.Context, banGroup *domain.BanGroup) error {
	const query = `
	INSERT INTO ban_group (source_id, target_id, group_id, group_name, is_enabled, deleted, note,
	unban_reason_text, origin, created_on, updated_on, valid_until, appeal_state)
	VALUES ($1, $2, $3, $4, $5, $6,$7, $8, $9, $10, $11, $12, $13)
	RETURNING ban_group_id`

	return errs.DBErr(s.
		QueryRow(ctx, query, banGroup.SourceID.Int64(), banGroup.TargetID.Int64(), banGroup.GroupID.Int64(),
			banGroup.GroupName, banGroup.IsEnabled, banGroup.Deleted, banGroup.Note, banGroup.UnbanReasonText, banGroup.Origin,
			banGroup.CreatedOn, banGroup.UpdatedOn, banGroup.ValidUntil, banGroup.AppealState).
		Scan(&banGroup.BanGroupID))
}

func (s Stores) updateBanGroup(ctx context.Context, banGroup *domain.BanGroup) error {
	banGroup.UpdatedOn = time.Now()

	return errs.DBErr(s.ExecUpdateBuilder(ctx, s.
		Builder().
		Update("ban_group").
		Set("source_id", banGroup.SourceID.Int64()).
		Set("target_id", banGroup.TargetID.Int64()).
		Set("group_name", banGroup.GroupName).
		Set("is_enabled", banGroup.IsEnabled).
		Set("deleted", banGroup.Deleted).
		Set("note", banGroup.Note).
		Set("unban_reason_text", banGroup.UnbanReasonText).
		Set("origin", banGroup.Origin).
		Set("updated_on", banGroup.UpdatedOn).
		Set("group_id", banGroup.GroupID.Int64()).
		Set("valid_until", banGroup.ValidUntil).
		Set("appeal_state", banGroup.AppealState).
		Where(sq.Eq{"ban_group_id": banGroup.BanGroupID})))
}

func (s Stores) DropBanGroup(ctx context.Context, banGroup *domain.BanGroup) error {
	banGroup.IsEnabled = false
	banGroup.Deleted = true

	return s.SaveBanGroup(ctx, banGroup)
}
