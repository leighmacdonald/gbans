package store

import (
	"context"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"time"
)

func (database *pgStore) DropBan(ctx context.Context, ban *model.Ban, hardDelete bool) error {
	if hardDelete {
		const query = `DELETE FROM ban WHERE ban_id = $1`
		if errExec := database.Exec(ctx, query, ban.BanID); errExec != nil {
			return Err(errExec)
		}
		ban.BanID = 0
		return nil
	} else {
		ban.Deleted = true
		return database.updateBan(ctx, ban)
	}
}

func (database *pgStore) getBanByColumn(ctx context.Context, column string, identifier any, full bool, person *model.BannedPerson) error {
	var query = fmt.Sprintf(`
	SELECT
		b.ban_id, b.steam_id, b.author_id, b.ban_type, b.reason,
		b.reason_text, b.note, b.ban_source, b.valid_until, b.created_on, b.updated_on,
		p.steam_id as sid2, p.created_on as created_on2, p.updated_on as updated_on2, p.communityvisibilitystate,
		p.profilestate,
		p.personaname, p.profileurl, p.avatar, p.avatarmedium, p.avatarfull, p.avatarhash,
		p.personastate, p.realname, p.timecreated, p.loccountrycode, p.locstatecode, p.loccityid,
		p.permission_level, p.discord_id, p.community_banned, p.vac_bans, p.game_bans, p.economy_ban,
		p.days_since_last_ban, b.deleted, case WHEN b.report_id is null THEN 0 ELSE b.report_id END,
		b.unban_reason_text
	FROM ban b
	LEFT OUTER JOIN person p on p.steam_id = b.steam_id
	WHERE b.%s = $1 AND b.valid_until > $2 AND b.deleted = false
	GROUP BY b.ban_id, p.steam_id
	ORDER BY b.created_on DESC
	LIMIT 1`, column)
	if errQuery := database.QueryRow(ctx, query, identifier, config.Now()).
		Scan(&person.Ban.BanID, &person.Ban.SteamID, &person.Ban.AuthorID, &person.Ban.BanType, &person.Ban.Reason, &person.Ban.ReasonText,
			&person.Ban.Note, &person.Ban.Source, &person.Ban.ValidUntil, &person.Ban.CreatedOn, &person.Ban.UpdatedOn,
			&person.Person.SteamID, &person.Person.CreatedOn, &person.Person.UpdatedOn,
			&person.Person.CommunityVisibilityState, &person.Person.ProfileState, &person.Person.PersonaName,
			&person.Person.ProfileURL, &person.Person.Avatar, &person.Person.AvatarMedium, &person.Person.AvatarFull,
			&person.Person.AvatarHash, &person.Person.PersonaState, &person.Person.RealName, &person.Person.TimeCreated, &person.Person.LocCountryCode,
			&person.Person.LocStateCode, &person.Person.LocCityID, &person.Person.PermissionLevel, &person.Person.DiscordID, &person.Person.CommunityBanned,
			&person.Person.VACBans, &person.Person.GameBans, &person.Person.EconomyBan, &person.Person.DaysSinceLastBan,
			&person.Ban.Deleted, &person.Ban.ReportId, &person.Ban.UnbanReasonText); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func (database *pgStore) GetBanBySteamID(ctx context.Context, sid64 steamid.SID64, full bool, bannedPerson *model.BannedPerson) error {
	return database.getBanByColumn(ctx, "steam_id", sid64, full, bannedPerson)
}

func (database *pgStore) GetBanByBanID(ctx context.Context, banID int64, full bool, bannedPerson *model.BannedPerson) error {
	return database.getBanByColumn(ctx, "ban_id", banID, full, bannedPerson)
}

// SaveBan will insert or update the ban record
// New records will have the Ban.BanID set automatically
func (database *pgStore) SaveBan(ctx context.Context, ban *model.Ban) error {
	// Ensure the foreign keys are satisfied
	targetPerson := model.NewPerson(ban.SteamID)
	errGetPerson := database.GetOrCreatePersonBySteamID(ctx, ban.SteamID, &targetPerson)
	if errGetPerson != nil {
		return errors.Wrapf(errGetPerson, "Failed to get targetPerson for ban")
	}
	authorPerson := model.NewPerson(ban.AuthorID)
	errGetAuthor := database.GetOrCreatePersonBySteamID(ctx, ban.AuthorID, &authorPerson)
	if errGetAuthor != nil {
		return errors.Wrapf(errGetPerson, "Failed to get author for ban")
	}
	ban.UpdatedOn = config.Now()
	if ban.BanID > 0 {
		return database.updateBan(ctx, ban)
	}
	ban.CreatedOn = config.Now()
	existing := model.NewBannedPerson()
	errGetBan := database.GetBanBySteamID(ctx, ban.SteamID, false, &existing)
	if errGetBan != nil {
		if !errors.Is(errGetBan, ErrNoResult) {
			return errors.Wrapf(errGetPerson, "Failed to check existing ban state")
		}
	} else {
		if ban.BanType <= existing.Ban.BanType {
			return ErrDuplicate
		}
	}
	return database.insertBan(ctx, ban)
}

func (database *pgStore) insertBan(ctx context.Context, ban *model.Ban) error {
	const query = `
		INSERT INTO ban (steam_id, author_id, ban_type, reason, reason_text, note, valid_until, created_on, updated_on, ban_source, report_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, case WHEN $11 = 0 THEN null ELSE $11 END)
		RETURNING ban_id`
	errQuery := database.QueryRow(ctx, query, ban.SteamID, ban.AuthorID, ban.BanType, ban.Reason, ban.ReasonText,
		ban.Note, ban.ValidUntil, ban.CreatedOn, ban.UpdatedOn, ban.Source, ban.ReportId).Scan(&ban.BanID)
	if errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func (database *pgStore) updateBan(ctx context.Context, ban *model.Ban) error {
	const query = `
		UPDATE
		    ban
		SET
		    author_id = $2, reason = $3, reason_text = $4, note = $5, valid_until = $6, updated_on = $7, 
			ban_source = $8, ban_type = $9, deleted = $10, report_id = case WHEN $11 = 0 THEN null ELSE $11 END, unban_reason_text = $12
		WHERE ban_id = $1`
	if errExec := database.Exec(ctx, query, ban.BanID, ban.AuthorID, ban.Reason, ban.ReasonText, ban.Note, ban.ValidUntil,
		ban.UpdatedOn, ban.Source, ban.BanType, ban.Deleted, ban.ReportId, ban.UnbanReasonText); errExec != nil {
		return Err(errExec)
	}
	return nil
}

func (database *pgStore) GetExpiredBans(ctx context.Context) ([]model.Ban, error) {
	const q = `SELECT ban_id, steam_id, author_id, ban_type, reason, reason_text,
       note, valid_until, ban_source, created_on, updated_on, deleted, 
       case WHEN report_id is null THEN 0 ELSE report_id END, unban_reason_text FROM ban
       WHERE valid_until < $1 AND deleted = false`
	var bans []model.Ban
	rows, errQuery := database.Query(ctx, q, config.Now())
	if errQuery != nil {
		return nil, errQuery
	}
	defer rows.Close()
	for rows.Next() {
		var ban model.Ban
		if errScan := rows.Scan(&ban.BanID, &ban.SteamID, &ban.AuthorID, &ban.BanType, &ban.Reason, &ban.ReasonText, &ban.Note,
			&ban.ValidUntil, &ban.Source, &ban.CreatedOn, &ban.UpdatedOn, &ban.Deleted, &ban.ReportId, &ban.UnbanReasonText); errScan != nil {
			return nil, errScan
		}
		bans = append(bans, ban)
	}
	return bans, nil
}

type BansQueryFilter struct {
	QueryFilter
	SteamId steamid.SID64 `json:"steam_id,omitempty"`
}

// GetBans returns all bans that fit the filter criteria passed in
func (database *pgStore) GetBans(ctx context.Context, filter *BansQueryFilter) ([]model.BannedPerson, error) {
	// 	query := fmt.Sprintf(`SELECT
	//		b.ban_id, b.steam_id, b.author_id, b.ban_type, b.reason,
	//		b.reason_text, b.note, b.ban_source, b.valid_until, b.created_on, b.updated_on,
	//		p.steam_id as sid2, p.created_on as created_on2, p.updated_on as updated_on2, p.communityvisibilitystate,
	//		p.profilestate,
	//		p.personaname, p.profileurl, p.avatar, p.avatarmedium, p.avatarfull, p.avatarhash,
	//		p.personastate, p.realname, p.timecreated, p.loccountrycode, p.locstatecode, p.loccityid,
	//		p.permission_level, p.discord_id, p.community_banned, p.vac_bans, p.game_bans, p.economy_ban,
	//		p.days_since_last_ban, b.deleted
	//	FROM ban b
	//	LEFT OUTER JOIN person p on p.steam_id = b.steam_id
	//	WHERE b.deleted = false
	//	ORDER BY b.%s LIMIT %d OFFSET %d`, filter.OrderBy, filter.Limit, filter.Offset)
	qb := sb.Select("b.ban_id as ban_id", "b.steam_id as steam_id", "b.author_id as author_id",
		"b.ban_type as ban_type", "b.reason as reason", "b.reason_text as reason_text",
		"b.note as note", "b.ban_source as ban_source", "b.valid_until as valid_until", "b.created_on as created_on",
		"b.updated_on as updated_on", "p.steam_id as sid2",
		"p.created_on as created_on2", "p.updated_on as updated_on2", "p.communityvisibilitystate",
		"p.profilestate", "p.personaname as personaname", "p.profileurl", "p.avatar", "p.avatarmedium", "p.avatarfull",
		"p.avatarhash", "p.personastate", "p.realname", "p.timecreated", "p.loccountrycode", "p.locstatecode",
		"p.loccityid", "p.permission_level", "p.discord_id as discord_id", "p.community_banned", "p.vac_bans", "p.game_bans",
		"p.economy_ban", "p.days_since_last_ban", "b.deleted as deleted", "case WHEN b.report_id is null THEN 0 ELSE b.report_id END", "b.unban_reason_text").
		From("ban b").
		JoinClause("LEFT OUTER JOIN person p on p.steam_id = b.steam_id")
	if !filter.Deleted {
		qb = qb.Where(sq.Eq{"deleted": false})
	}

	if filter.SteamId.Valid() {
		qb = qb.Where(sq.Eq{"b.steam_id": filter.SteamId.Int64()})
	}
	if filter.OrderBy != "" {
		if filter.SortDesc {
			qb = qb.OrderBy(fmt.Sprintf("b.%s DESC", filter.OrderBy))
		} else {
			qb = qb.OrderBy(fmt.Sprintf("b.%s ASC", filter.OrderBy))
		}
	}
	if filter.Limit > 0 {
		qb = qb.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		qb = qb.Offset(filter.Offset)
	}
	query, args, errQueryBuilder := qb.ToSql()
	if errQueryBuilder != nil {
		return nil, Err(errQueryBuilder)
	}
	var bans []model.BannedPerson
	rows, errQuery := database.conn.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	for rows.Next() {
		bannedPerson := model.NewBannedPerson()
		if errScan := rows.Scan(&bannedPerson.Ban.BanID, &bannedPerson.Ban.SteamID, &bannedPerson.Ban.AuthorID,
			&bannedPerson.Ban.BanType, &bannedPerson.Ban.Reason, &bannedPerson.Ban.ReasonText,
			&bannedPerson.Ban.Note, &bannedPerson.Ban.Source, &bannedPerson.Ban.ValidUntil,
			&bannedPerson.Ban.CreatedOn, &bannedPerson.Ban.UpdatedOn,
			&bannedPerson.Person.SteamID, &bannedPerson.Person.CreatedOn, &bannedPerson.Person.UpdatedOn,
			&bannedPerson.Person.CommunityVisibilityState, &bannedPerson.Person.ProfileState,
			&bannedPerson.Person.PersonaName, &bannedPerson.Person.ProfileURL, &bannedPerson.Person.Avatar,
			&bannedPerson.Person.AvatarMedium, &bannedPerson.Person.AvatarFull, &bannedPerson.Person.AvatarHash,
			&bannedPerson.Person.PersonaState, &bannedPerson.Person.RealName, &bannedPerson.Person.TimeCreated,
			&bannedPerson.Person.LocCountryCode, &bannedPerson.Person.LocStateCode, &bannedPerson.Person.LocCityID,
			&bannedPerson.Person.PermissionLevel, &bannedPerson.Person.DiscordID, &bannedPerson.Person.CommunityBanned,
			&bannedPerson.Person.VACBans, &bannedPerson.Person.GameBans, &bannedPerson.Person.EconomyBan,
			&bannedPerson.Person.DaysSinceLastBan, &bannedPerson.Ban.Deleted, &bannedPerson.Ban.ReportId,
			&bannedPerson.Ban.UnbanReasonText); errScan != nil {
			return nil, Err(errScan)
		}
		bans = append(bans, bannedPerson)
	}
	return bans, nil
}

func (database *pgStore) GetBansOlderThan(ctx context.Context, filter *QueryFilter, since time.Time) ([]model.Ban, error) {
	query := fmt.Sprintf(`
		SELECT
			b.ban_id, b.steam_id, b.author_id, b.ban_type, b.reason, b.reason_text, b.note, 
			b.ban_source, b.valid_until, b.created_on, b.updated_on, b.deleted, 
			case WHEN b.report_id is null THEN 0 ELSE b.report_id END, b.unban_reason_text
		FROM ban b
		WHERE updated_on < $1 AND deleted = false
		LIMIT %d
		OFFSET %d`, filter.Limit, filter.Offset)
	var bans []model.Ban
	rows, errQuery := database.Query(ctx, query, since)
	if errQuery != nil {
		return nil, errQuery
	}
	defer rows.Close()
	for rows.Next() {
		var ban model.Ban
		if errQuery = rows.Scan(&ban.BanID, &ban.SteamID, &ban.AuthorID, &ban.BanType, &ban.Reason, &ban.ReasonText, &ban.Note,
			&ban.Source, &ban.ValidUntil, &ban.CreatedOn, &ban.UpdatedOn, &ban.Deleted, &ban.ReportId, &ban.UnbanReasonText); errQuery != nil {
			return nil, errQuery
		}
		bans = append(bans, ban)
	}
	return bans, nil
}

func (database *pgStore) SaveBanMessage(ctx context.Context, message *model.UserMessage) error {
	if message.MessageId > 0 {
		return database.updateBanMessage(ctx, message)
	}
	return database.insertBanMessage(ctx, message)
}

func (database *pgStore) updateBanMessage(ctx context.Context, message *model.UserMessage) error {
	message.UpdatedOn = config.Now()
	const query = `
		UPDATE ban_appeal 
		SET deleted = $2, author_id = $3, updated_on = $4, message_md = $5
		WHERE ban_message_id = $1
	`
	if errQuery := database.Exec(ctx, query,
		message.MessageId,
		message.Deleted,
		message.AuthorId,
		message.UpdatedOn,
		message.Message,
	); errQuery != nil {
		return Err(errQuery)
	}
	log.WithFields(log.Fields{
		"ban_id":     message.ParentId,
		"message_id": message.Message,
		"author_id":  message.AuthorId,
	}).Infof("Appeal message edited")
	return nil
}

func (database *pgStore) insertBanMessage(ctx context.Context, message *model.UserMessage) error {
	const query = `
		INSERT INTO ban_appeal (
		    ban_id, author_id, message_md, deleted, created_on, updated_on
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING ban_message_id
	`
	if errQuery := database.QueryRow(ctx, query,
		message.ParentId,
		message.AuthorId,
		message.Message,
		message.Deleted,
		message.CreatedOn,
		message.UpdatedOn,
	).Scan(&message.MessageId); errQuery != nil {
		return Err(errQuery)
	}
	log.WithFields(log.Fields{
		"ban_id":     message.ParentId,
		"message_id": message.MessageId,
		"author_id":  message.AuthorId,
	}).Infof("Report message saved")
	return nil
}

func (database *pgStore) GetBanMessages(ctx context.Context, banId int64) ([]model.UserMessage, error) {
	const query = `
		SELECT 
		   ban_message_id, ban_id, author_id, message_md, deleted, created_on, updated_on
		FROM ban_appeal
		WHERE deleted = false AND ban_id = $1 
		ORDER BY created_on`
	rows, errQuery := database.Query(ctx, query, banId)
	if errQuery != nil {
		if Err(errQuery) == ErrNoResult {
			return nil, nil
		}
	}
	defer rows.Close()
	var messages []model.UserMessage
	for rows.Next() {
		var msg model.UserMessage
		if errScan := rows.Scan(
			&msg.MessageId,
			&msg.ParentId,
			&msg.AuthorId,
			&msg.Message,
			&msg.Deleted,
			&msg.CreatedOn,
			&msg.UpdatedOn,
		); errScan != nil {
			return nil, Err(errQuery)
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

func (database *pgStore) GetBanMessageById(ctx context.Context, banMessageId int, message *model.UserMessage) error {
	const query = `
		SELECT 
		   ban_message_id, ban_id, author_id, message_md, deleted, created_on, updated_on
		FROM ban_appeal
		WHERE ban_message_id = $1`
	if errQuery := database.QueryRow(ctx, query, banMessageId).Scan(
		&message.MessageId,
		&message.ParentId,
		&message.AuthorId,
		&message.Message,
		&message.Deleted,
		&message.CreatedOn,
		&message.UpdatedOn,
	); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func (database *pgStore) DropBanMessage(ctx context.Context, message *model.UserMessage) error {
	const q = `UPDATE ban_appeal SET deleted = true WHERE ban_message_id = $1`
	if errExec := database.Exec(ctx, q, message.MessageId); errExec != nil {
		return Err(errExec)
	}
	log.WithFields(log.Fields{
		"ban_message_id": message.MessageId,
		"soft":           true,
	}).Infof("Appeal message deleted")
	message.Deleted = true
	return nil
}
