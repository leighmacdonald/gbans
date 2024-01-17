package store

import (
	"context"
	"fmt"
	"net"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (db *Store) DropBan(ctx context.Context, ban *model.BanSteam, hardDelete bool) error {
	if hardDelete {
		if errExec := db.Exec(ctx, `DELETE FROM ban WHERE ban_id = $1`, ban.BanID); errExec != nil {
			return Err(errExec)
		}

		ban.BanID = 0

		return nil
	} else {
		ban.Deleted = true

		return db.updateBan(ctx, ban)
	}
}

func (db *Store) getBanByColumn(ctx context.Context, column string, identifier any, person *model.BannedSteamPerson, deletedOk bool) error {
	whereClauses := sq.And{
		sq.Eq{fmt.Sprintf("b.%s", column): identifier}, // valid columns are immutable
	}

	if !deletedOk {
		whereClauses = append(whereClauses, sq.Eq{"b.deleted": false})
	} else {
		whereClauses = append(whereClauses, sq.Gt{"b.valid_until": time.Now()})
	}

	query := db.sb.Select(
		"b.ban_id", "b.target_id", "b.source_id", "b.ban_type", "b.reason",
		"b.reason_text", "b.note", "b.origin", "b.valid_until", "b.created_on", "b.updated_on", "b.include_friends",
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

	row, errQuery := db.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return errQuery
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
		return Err(errScan)
	}

	person.SourceID = steamid.New(sourceID)
	person.TargetID = steamid.New(targetID)

	return nil
}

func (db *Store) GetBanBySteamID(ctx context.Context, sid64 steamid.SID64, bannedPerson *model.BannedSteamPerson, deletedOk bool) error {
	return db.getBanByColumn(ctx, "target_id", sid64, bannedPerson, deletedOk)
}

func (db *Store) GetBanByBanID(ctx context.Context, banID int64, bannedPerson *model.BannedSteamPerson, deletedOk bool) error {
	return db.getBanByColumn(ctx, "ban_id", banID, bannedPerson, deletedOk)
}

func (db *Store) GetBanByLastIP(ctx context.Context, lastIP net.IP, bannedPerson *model.BannedSteamPerson, deletedOk bool) error {
	return db.getBanByColumn(ctx, "last_ip", fmt.Sprintf("::ffff:%s", lastIP.String()), bannedPerson, deletedOk)
}

// SaveBan will insert or update the ban record
// New records will have the Ban.BanID set automatically.
func (db *Store) SaveBan(ctx context.Context, ban *model.BanSteam) error {
	// Ensure the foreign keys are satisfied
	targetPerson := model.NewPerson(ban.TargetID)
	if errGetPerson := db.GetOrCreatePersonBySteamID(ctx, ban.TargetID, &targetPerson); errGetPerson != nil {
		return errors.Wrapf(errGetPerson, "Failed to get targetPerson for ban")
	}

	authorPerson := model.NewPerson(ban.SourceID)
	if errGetAuthor := db.GetOrCreatePersonBySteamID(ctx, ban.SourceID, &authorPerson); errGetAuthor != nil {
		return errors.Wrapf(errGetAuthor, "Failed to get author for ban")
	}

	ban.UpdatedOn = time.Now()
	if ban.BanID > 0 {
		return db.updateBan(ctx, ban)
	}

	ban.CreatedOn = ban.UpdatedOn

	existing := model.NewBannedPerson()

	errGetBan := db.GetBanBySteamID(ctx, ban.TargetID, &existing, false)
	if errGetBan != nil {
		if !errors.Is(errGetBan, ErrNoResult) {
			return errors.Wrapf(errGetBan, "Failed to check existing ban state")
		}
	} else {
		if ban.BanType <= existing.BanType {
			return ErrDuplicate
		}
	}

	ban.LastIP = db.GetPlayerMostRecentIP(ctx, ban.TargetID)

	return db.insertBan(ctx, ban)
}

func (db *Store) insertBan(ctx context.Context, ban *model.BanSteam) error {
	const query = `
		INSERT INTO ban (target_id, source_id, ban_type, reason, reason_text, note, valid_until, 
		                 created_on, updated_on, origin, report_id, appeal_state, include_friends, last_ip)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, case WHEN $11 = 0 THEN null ELSE $11 END, $12, $13, $14)
		RETURNING ban_id`

	errQuery := db.
		QueryRow(ctx, query, ban.TargetID.Int64(), ban.SourceID.Int64(), ban.BanType, ban.Reason, ban.ReasonText,
			ban.Note, ban.ValidUntil, ban.CreatedOn, ban.UpdatedOn, ban.Origin, ban.ReportID, ban.AppealState,
			ban.IncludeFriends, &ban.LastIP).
		Scan(&ban.BanID)

	if errQuery != nil {
		return Err(errQuery)
	}

	return nil
}

func (db *Store) updateBan(ctx context.Context, ban *model.BanSteam) error {
	var reportID *int64
	if ban.ReportID > 0 {
		reportID = &ban.ReportID
	}

	query := db.sb.Update("ban").
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

	return Err(db.ExecUpdateBuilder(ctx, query))
}

func (db *Store) GetExpiredBans(ctx context.Context) ([]model.BanSteam, error) {
	query := db.sb.
		Select("ban_id", "target_id", "source_id", "ban_type", "reason", "reason_text", "note",
			"valid_until", "origin", "created_on", "updated_on", "deleted", "case WHEN report_id is null THEN 0 ELSE report_id END",
			"unban_reason_text", "is_enabled", "appeal_state", "include_friends").
		From("ban").
		Where(sq.And{sq.Lt{"valid_until": time.Now()}, sq.Eq{"deleted": false}})

	rows, errQuery := db.QueryBuilder(ctx, query)
	if errQuery != nil {
		return nil, errQuery
	}

	defer rows.Close()

	bans := []model.BanSteam{}

	for rows.Next() {
		var (
			ban      model.BanSteam
			sourceID int64
			targetID int64
		)

		if errScan := rows.Scan(&ban.BanID, &targetID, &sourceID, &ban.BanType, &ban.Reason, &ban.ReasonText, &ban.Note,
			&ban.ValidUntil, &ban.Origin, &ban.CreatedOn, &ban.UpdatedOn, &ban.Deleted, &ban.ReportID, &ban.UnbanReasonText,
			&ban.IsEnabled, &ban.AppealState, &ban.IncludeFriends); errScan != nil {
			return nil, errors.Wrap(errScan, "Failed to load ban")
		}

		ban.SourceID = steamid.New(sourceID)
		ban.TargetID = steamid.New(targetID)

		bans = append(bans, ban)
	}

	return bans, nil
}

type AppealQueryFilter struct {
	QueryFilter
	AppealState model.AppealState `json:"appeal_state"`
	SourceID    model.StringSID   `json:"source_id"`
	TargetID    model.StringSID   `json:"target_id"`
}

func (db *Store) GetAppealsByActivity(ctx context.Context, opts AppealQueryFilter) ([]model.AppealOverview, int64, error) {
	constraints := sq.And{sq.Gt{"m.count": 0}}

	if !opts.Deleted {
		constraints = append(constraints, sq.Eq{"b.deleted": opts.Deleted})
	}

	if opts.AppealState > model.AnyState {
		constraints = append(constraints, sq.Eq{"b.appeal_state": opts.AppealState})
	}

	if opts.SourceID != "" {
		authorID, errAuthorID := opts.SourceID.SID64(ctx)
		if errAuthorID != nil {
			return nil, 0, errors.Wrap(errAuthorID, "Invalid source id")
		}

		constraints = append(constraints, sq.Eq{"b.source_id": authorID.Int64()})
	}

	if opts.TargetID != "" {
		targetID, errTargetID := opts.TargetID.SID64(ctx)
		if errTargetID != nil {
			return nil, 0, errors.Wrap(errTargetID, "Invalid target id")
		}

		constraints = append(constraints, sq.Eq{"b.target_id": targetID.Int64()})
	}

	counterQuery := db.sb.
		Select("COUNT(b.ban_id)").
		From("ban b").
		Where(constraints).
		InnerJoin(`
			LATERAL (
				SELECT count(a.ban_message_id) as count 
				FROM ban_appeal a
				WHERE b.ban_id = a.ban_id
			) m ON TRUE`)

	count, errCount := db.GetCount(ctx, counterQuery)
	if errCount != nil {
		return nil, 0, Err(errCount)
	}

	builder := db.sb.
		Select("b.ban_id", "b.target_id", "b.source_id", "b.ban_type", "b.reason", "b.reason_text",
			"b.note", "b.valid_until", "b.origin", "b.created_on", "b.updated_on", "b.deleted",
			"CASE WHEN b.report_id IS NULL THEN 0 ELSE report_id END",
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

	builder = opts.QueryFilter.applySafeOrder(builder, map[string][]string{
		"b.": {
			"ban_id", "target_id", "source_id", "ban_type", "reason", "valid_until", "origin", "created_on",
			"updated_on", "deleted", "is_enabled", "appeal_state",
		},
	}, "updated_on")

	builder = opts.QueryFilter.applyLimitOffsetDefault(builder)

	rows, errQuery := db.QueryBuilder(ctx, builder)
	if errQuery != nil {
		return nil, 0, Err(errQuery)
	}

	defer rows.Close()

	overviews := []model.AppealOverview{}

	for rows.Next() {
		var (
			overview      model.AppealOverview
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
			return nil, 0, errors.Wrap(errScan, "Failed to scan appeal overview")
		}

		overview.SourceID = steamid.New(SourceSteamID)
		overview.TargetID = steamid.New(TargetSteamID)

		overviews = append(overviews, overview)
	}

	return overviews, count, nil
}

type BansQueryFilter struct {
	QueryFilter
	SourceID      model.StringSID `json:"source_id,omitempty"`
	TargetID      model.StringSID `json:"target_id,omitempty"`
	Reason        model.Reason    `json:"reason,omitempty"`
	PermanentOnly bool            `json:"permanent_only,omitempty"`
}

type CIDRBansQueryFilter struct {
	BansQueryFilter
	IP string `json:"ip,omitempty"`
}

type ASNBansQueryFilter struct {
	BansQueryFilter
	ASNum int64 `json:"as_num,omitempty"`
}

type GroupBansQueryFilter struct {
	BansQueryFilter
	GroupID string `json:"group_id"`
}

type SteamBansQueryFilter struct {
	BansQueryFilter
	// IncludeFriendsOnly Return results that have "deep" bans where players friends list is
	// also banned while the primary targets ban has not expired.
	IncludeFriendsOnly bool              `json:"include_friends_only"`
	AppealState        model.AppealState `json:"appeal_state"`
}

// GetBansSteam returns all bans that fit the filter criteria passed in.
func (db *Store) GetBansSteam(ctx context.Context, filter SteamBansQueryFilter) ([]model.BannedSteamPerson, int64, error) {
	builder := db.sb.Select("b.ban_id", "b.target_id", "b.source_id", "b.ban_type", "b.reason",
		"b.reason_text", "b.note", "b.origin", "b.valid_until", "b.created_on", "b.updated_on", "b.include_friends",
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

	if filter.TargetID != "" {
		targetID, errTargetID := filter.TargetID.SID64(ctx)
		if errTargetID != nil {
			return nil, 0, errors.Wrap(errTargetID, "Invalid target id")
		}

		ands = append(ands, sq.Eq{"b.target_id": targetID.Int64()})
	}

	if filter.SourceID != "" {
		sourceID, errSourceID := filter.SourceID.SID64(ctx)
		if errSourceID != nil {
			return nil, 0, errors.Wrap(errSourceID, "Invalid source id")
		}

		ands = append(ands, sq.Eq{"b.source_id": sourceID.Int64()})
	}

	if filter.IncludeFriendsOnly {
		ands = append(ands, sq.Eq{"b.include_friends": true})
	}

	if filter.AppealState > model.AnyState {
		ands = append(ands, sq.Eq{"b.appeal_state": filter.AppealState})
	}

	if len(ands) > 0 {
		builder = builder.Where(ands)
	}

	builder = filter.QueryFilter.applySafeOrder(builder, map[string][]string{
		"b.": {
			"ban_id", "target_id", "source_id", "ban_type", "reason",
			"origin", "valid_until", "created_on", "updated_on", "include_friends",
			"deleted", "report_id", "is_enabled", "appeal_state",
		},
		"s.": {"source_personaname"},
		"t.": {"target_personaname", "community_banned", "vac_bans", "game_bans"},
	}, "ban_id")

	builder = filter.QueryFilter.applyLimitOffsetDefault(builder)

	count, errCount := db.GetCount(ctx, db.sb.
		Select("COUNT(b.ban_id)").
		From("ban b").
		Where(ands))
	if errCount != nil {
		return nil, 0, Err(errCount)
	}

	if count == 0 {
		return []model.BannedSteamPerson{}, 0, nil
	}

	var bans []model.BannedSteamPerson

	rows, errQuery := db.QueryBuilder(ctx, builder)
	if errQuery != nil {
		return nil, 0, Err(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			person   = model.NewBannedPerson()
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
			return nil, 0, Err(errScan)
		}

		person.TargetID = steamid.New(targetID)
		person.SourceID = steamid.New(sourceID)

		bans = append(bans, person)
	}

	if bans == nil {
		bans = []model.BannedSteamPerson{}
	}

	return bans, count, nil
}

func (db *Store) GetBansOlderThan(ctx context.Context, filter QueryFilter, since time.Time) ([]model.BanSteam, error) {
	query := db.sb.
		Select("b.ban_id", "b.target_id", "b.source_id", "b.ban_type", "b.reason",
			"b.reason_text", "b.note", "b.origin", "b.valid_until", "b.created_on", "b.updated_on", "b.deleted",
			"case WHEN b.report_id is null THEN 0 ELSE b.report_id END", "b.unban_reason_text", "b.is_enabled",
			"b.appeal_state", "b.include_friends").
		From("ban b").
		Where(sq.And{sq.Lt{"updated_on": since}, sq.Eq{"deleted": false}})

	rows, errQuery := db.QueryBuilder(ctx, filter.applyLimitOffsetDefault(query))
	if errQuery != nil {
		return nil, errQuery
	}

	defer rows.Close()

	bans := []model.BanSteam{}

	for rows.Next() {
		var (
			ban      model.BanSteam
			sourceID int64
			targetID int64
		)

		if errQuery = rows.Scan(&ban.BanID, &targetID, &sourceID, &ban.BanType, &ban.Reason, &ban.ReasonText, &ban.Note,
			&ban.Origin, &ban.ValidUntil, &ban.CreatedOn, &ban.UpdatedOn, &ban.Deleted, &ban.ReportID, &ban.UnbanReasonText,
			&ban.IsEnabled, &ban.AppealState, &ban.AppealState); errQuery != nil {
			return nil, errors.Wrap(errQuery, "Failed to scan ban")
		}

		ban.SourceID = steamid.New(sourceID)
		ban.TargetID = steamid.New(targetID)

		bans = append(bans, ban)
	}

	return bans, nil
}

func (db *Store) SaveBanMessage(ctx context.Context, message *model.BanAppealMessage) error {
	var err error
	if message.BanMessageID > 0 {
		err = db.updateBanMessage(ctx, message)
	} else {
		err = db.insertBanMessage(ctx, message)
	}

	bannedPerson := model.NewBannedPerson()
	if errBan := db.GetBanByBanID(ctx, message.BanID, &bannedPerson, true); errBan != nil {
		return ErrNoResult
	}

	bannedPerson.UpdatedOn = time.Now()

	if errUpdate := db.updateBan(ctx, &bannedPerson.BanSteam); errUpdate != nil {
		return errUpdate
	}

	return err
}

func (db *Store) updateBanMessage(ctx context.Context, message *model.BanAppealMessage) error {
	message.UpdatedOn = time.Now()

	query := db.sb.Update("ban_appeal").
		Set("deleted", message.Deleted).
		Set("author_id", message.AuthorID.Int64()).
		Set("updated_on", message.UpdatedOn).
		Set("message_md", message.MessageMD).
		Where(sq.Eq{"ban_message_id": message.BanMessageID})

	if errQuery := db.ExecUpdateBuilder(ctx, query); errQuery != nil {
		return Err(errQuery)
	}

	db.log.Info("Ban appeal message updated",
		zap.Int64("ban_id", message.BanID),
		zap.Int64("message_id", message.BanMessageID),
		zap.Int64("author_id", message.AuthorID.Int64()))

	return nil
}

func (db *Store) insertBanMessage(ctx context.Context, message *model.BanAppealMessage) error {
	const query = `
	INSERT INTO ban_appeal (
		ban_id, author_id, message_md, deleted, created_on, updated_on
	)
	VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING ban_message_id
	`

	if errQuery := db.QueryRow(ctx, query,
		message.BanID,
		message.AuthorID.Int64(),
		message.MessageMD,
		message.Deleted,
		message.CreatedOn,
		message.UpdatedOn,
	).Scan(&message.BanMessageID); errQuery != nil {
		return Err(errQuery)
	}

	db.log.Info("Ban appeal message created",
		zap.Int64("ban_id", message.BanID),
		zap.Int64("message_id", message.BanMessageID),
		zap.Int64("author_id", message.AuthorID.Int64()))

	return nil
}

func (db *Store) GetBanMessages(ctx context.Context, banID int64) ([]model.BanAppealMessage, error) {
	query := db.sb.
		Select("a.ban_message_id", "a.ban_id", "a.author_id", "a.message_md", "a.deleted",
			"a.created_on", "a.updated_on", "p.avatarhash", "p.personaname", "p.permission_level").
		From("ban_appeal a").
		LeftJoin("person p ON a.author_id = p.steam_id").
		Where(sq.And{sq.Eq{"a.deleted": false}, sq.Eq{"a.ban_id": banID}}).
		OrderBy("a.created_on")

	rows, errQuery := db.QueryBuilder(ctx, query)
	if errQuery != nil {
		if errors.Is(Err(errQuery), ErrNoResult) {
			return nil, nil
		}
	}

	defer rows.Close()

	messages := []model.BanAppealMessage{}

	for rows.Next() {
		var (
			msg      model.BanAppealMessage
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
			return nil, Err(errQuery)
		}

		msg.AuthorID = steamid.New(authorID)

		messages = append(messages, msg)
	}

	return messages, nil
}

func (db *Store) GetBanMessageByID(ctx context.Context, banMessageID int, message *model.BanAppealMessage) error {
	query := db.sb.
		Select("a.ban_message_id", "a.ban_id", "a.author_id", "a.message_md", "a.deleted", "a.created_on",
			"a.updated_on", "p.avatarhash", "p.personaname", "p.permission_level").
		From("ban_appeal a").
		LeftJoin("person p ON a.author_id = p.steam_id").
		Where(sq.Eq{"a.ban_message_id": banMessageID})

	var authorID int64

	row, errQuery := db.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return errQuery
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
		return Err(errScan)
	}

	message.AuthorID = steamid.New(authorID)

	return nil
}

func (db *Store) DropBanMessage(ctx context.Context, message *model.BanAppealMessage) error {
	query := db.sb.
		Update("ban_appeal").
		Set("deleted", true).
		Where(sq.Eq{"ban_message_id": message.BanMessageID})

	if errExec := db.ExecUpdateBuilder(ctx, query); errExec != nil {
		return Err(errExec)
	}

	db.log.Info("Appeal message deleted", zap.Int64("ban_message_id", message.BanMessageID))
	message.Deleted = true

	return nil
}

func (db *Store) GetBanGroup(ctx context.Context, groupID steamid.GID, banGroup *model.BanGroup) error {
	query := db.sb.
		Select("ban_group_id", "source_id", "target_id", "group_name", "is_enabled", "deleted",
			"note", "unban_reason_text", "origin", "created_on", "updated_on", "valid_until", "appeal_state", "group_id").
		From("ban_group").
		Where(sq.And{sq.Eq{"group_id": groupID.Int64()}, sq.Eq{"deleted": false}})

	var (
		sourceID   int64
		targetID   int64
		newGroupID int64
	)

	row, errQuery := db.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return errQuery
	}

	if errScan := row.Scan(&banGroup.BanGroupID, &sourceID, &targetID, &banGroup.GroupName, &banGroup.IsEnabled,
		&banGroup.Deleted, &banGroup.Note, &banGroup.UnbanReasonText, &banGroup.Origin, &banGroup.CreatedOn,
		&banGroup.UpdatedOn, &banGroup.ValidUntil, &banGroup.AppealState, &newGroupID); errScan != nil {
		return Err(errScan)
	}

	banGroup.SourceID = steamid.New(sourceID)
	banGroup.TargetID = steamid.New(targetID)
	banGroup.GroupID = steamid.NewGID(newGroupID)

	return nil
}

func (db *Store) GetBanGroupByID(ctx context.Context, banGroupID int64, banGroup *model.BanGroup) error {
	query := db.sb.
		Select("ban_group_id", "source_id", "target_id", "group_name", "is_enabled", "deleted",
			"note", "unban_reason_text", "origin", "created_on", "updated_on", "valid_until", "appeal_state", "group_id").
		From("ban_group").
		Where(sq.And{sq.Eq{"ban_group_id": banGroupID}, sq.Eq{"is_enabled": true}, sq.Eq{"deleted": false}})

	var (
		groupID  int64
		sourceID int64
		targetID int64
	)

	row, errQuery := db.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return errQuery
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
		return Err(errScan)
	}

	banGroup.SourceID = steamid.New(sourceID)
	banGroup.TargetID = steamid.New(targetID)
	banGroup.GroupID = steamid.NewGID(groupID)

	return nil
}

func (db *Store) GetBanGroups(ctx context.Context, filter GroupBansQueryFilter) ([]model.BannedGroupPerson, int64, error) {
	builder := db.sb.
		Select("b.ban_group_id", "b.source_id", "b.target_id", "b.group_name", "b.is_enabled", "b.deleted",
			"b.note", "b.unban_reason_text", "b.origin", "b.created_on", "b.updated_on", "b.valid_until",
			"b.appeal_state", "b.group_id",
			"s.personaname as source_personaname", "s.avatarhash",
			"t.personaname as target_personaname", "t.avatarhash", "t.community_banned", "t.vac_bans", "t.game_bans").
		From("ban_group b").
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

	if filter.GroupID != "" {
		gid := steamid.NewGID(filter.GroupID)
		if !gid.Valid() {
			return nil, 0, steamid.ErrInvalidGID
		}

		constraints = append(constraints, sq.Eq{"b.group_id": gid.Int64()})
	}

	if filter.TargetID != "" {
		targetID, errTargetID := filter.TargetID.SID64(ctx)
		if errTargetID != nil {
			return nil, 0, errors.Wrap(errTargetID, "Invalid target id")
		}

		constraints = append(constraints, sq.Eq{"b.target_id": targetID.Int64()})
	}

	if filter.SourceID != "" {
		sourceID, errSourceID := filter.SourceID.SID64(ctx)
		if errSourceID != nil {
			return nil, 0, errors.Wrap(errSourceID, "Invalid source id")
		}

		constraints = append(constraints, sq.Eq{"b.source_id": sourceID.Int64()})
	}

	builder = filter.QueryFilter.applySafeOrder(builder, map[string][]string{
		"b.": {
			"ban_group_id", "source_id", "target_id", "group_name", "is_enabled", "deleted",
			"origin", "created_on", "updated_on", "valid_until", "appeal_state", "group_id",
		},
		"s.": {"source_personaname"},
		"t.": {"target_personaname", "community_banned", "vac_bans", "game_bans"},
	}, "ban_group_id")

	builder = filter.applyLimitOffsetDefault(builder).Where(constraints)

	rows, errRows := db.QueryBuilder(ctx, builder)
	if errRows != nil {
		if errors.Is(errRows, ErrNoResult) {
			return []model.BannedGroupPerson{}, 0, nil
		}

		return nil, 0, Err(errRows)
	}

	defer rows.Close()

	var groups []model.BannedGroupPerson

	for rows.Next() {
		var (
			group    model.BannedGroupPerson
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
			return nil, 0, Err(errScan)
		}

		group.SourceID = steamid.New(sourceID)
		group.TargetID = steamid.New(targetID)
		group.GroupID = steamid.NewGID(groupID)

		groups = append(groups, group)
	}

	count, errCount := db.GetCount(ctx, db.sb.
		Select("b.ban_group_id").
		From("ban_group b").
		Where(constraints))
	if errCount != nil {
		if errors.Is(errCount, ErrNoResult) {
			return []model.BannedGroupPerson{}, 0, nil
		}

		return nil, 0, errCount
	}

	if groups == nil {
		groups = []model.BannedGroupPerson{}
	}

	return groups, count, nil
}

func (db *Store) GetMembersList(ctx context.Context, parentID int64, list *model.MembersList) error {
	row, err := db.QueryRowBuilder(ctx, db.sb.
		Select("members_id", "parent_id", "members", "created_on", "updated_on").
		From("members").
		Where(sq.Eq{"parent_id": parentID}))
	if err != nil {
		return err
	}

	return Err(row.Scan(&list.MembersID, &list.ParentID, &list.Members, &list.CreatedOn, &list.UpdatedOn))
}

func (db *Store) SaveMembersList(ctx context.Context, list *model.MembersList) error {
	if list.MembersID > 0 {
		list.UpdatedOn = time.Now()

		const update = `UPDATE members SET members = $2::jsonb, updated_on = $3 WHERE members_id = $1`

		return Err(db.Exec(ctx, update, list.MembersID, list.Members, list.UpdatedOn))
	} else {
		const insert = `INSERT INTO members (parent_id, members, created_on, updated_on) 
		VALUES ($1, $2::jsonb, $3, $4) RETURNING members_id`

		return Err(db.QueryRow(ctx, insert, list.ParentID, list.Members, list.CreatedOn, list.UpdatedOn).Scan(&list.MembersID))
	}
}

func (db *Store) SaveBanGroup(ctx context.Context, banGroup *model.BanGroup) error {
	if banGroup.BanGroupID > 0 {
		return db.updateBanGroup(ctx, banGroup)
	}

	return db.insertBanGroup(ctx, banGroup)
}

func (db *Store) insertBanGroup(ctx context.Context, banGroup *model.BanGroup) error {
	const query = `
	INSERT INTO ban_group (source_id, target_id, group_id, group_name, is_enabled, deleted, note,
	unban_reason_text, origin, created_on, updated_on, valid_until, appeal_state)
	VALUES ($1, $2, $3, $4, $5, $6,$7, $8, $9, $10, $11, $12, $13)
	RETURNING ban_group_id`

	return Err(db.
		QueryRow(ctx, query, banGroup.SourceID.Int64(), banGroup.TargetID.Int64(), banGroup.GroupID.Int64(),
			banGroup.GroupName, banGroup.IsEnabled, banGroup.Deleted, banGroup.Note, banGroup.UnbanReasonText, banGroup.Origin,
			banGroup.CreatedOn, banGroup.UpdatedOn, banGroup.ValidUntil, banGroup.AppealState).
		Scan(&banGroup.BanGroupID))
}

func (db *Store) updateBanGroup(ctx context.Context, banGroup *model.BanGroup) error {
	banGroup.UpdatedOn = time.Now()

	return Err(db.ExecUpdateBuilder(ctx, db.sb.
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

func (db *Store) DropBanGroup(ctx context.Context, banGroup *model.BanGroup) error {
	banGroup.IsEnabled = false
	banGroup.Deleted = true

	return db.SaveBanGroup(ctx, banGroup)
}
