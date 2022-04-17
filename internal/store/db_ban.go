package store

import (
	"context"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"time"
)

func (database *pgStore) DropBan(ctx context.Context, ban *model.Ban) error {
	const q = `DELETE FROM ban WHERE ban_id = $1`
	if _, err := database.conn.Exec(ctx, q, ban.BanID); err != nil {
		return Err(err)
	}
	return nil
}

func (database *pgStore) getBanByColumn(ctx context.Context, column string, identifier any, full bool, b *model.BannedPerson) error {
	var q = fmt.Sprintf(`
	SELECT
		b.ban_id, b.steam_id, b.author_id, b.ban_type, b.reason,
		b.reason_text, b.note, b.ban_source, b.valid_until, b.created_on, b.updated_on,
		p.steam_id as sid2, p.created_on as created_on2, p.updated_on as updated_on2, p.communityvisibilitystate,
		p.profilestate,
		p.personaname, p.profileurl, p.avatar, p.avatarmedium, p.avatarfull, p.avatarhash,
		p.personastate, p.realname, p.timecreated, p.loccountrycode, p.locstatecode, p.loccityid,
		p.permission_level, p.discord_id, p.community_banned, p.vac_bans, p.game_bans, p.economy_ban,
		p.days_since_last_ban
	FROM ban b
	LEFT OUTER JOIN person p on p.steam_id = b.steam_id
	WHERE b.%s = $1 AND b.valid_until > $2
	GROUP BY b.ban_id, p.steam_id
	ORDER BY b.created_on DESC
	LIMIT 1`, column)
	if err := database.conn.QueryRow(ctx, q, identifier, config.Now()).
		Scan(&b.Ban.BanID, &b.Ban.SteamID, &b.Ban.AuthorID, &b.Ban.BanType, &b.Ban.Reason, &b.Ban.ReasonText,
			&b.Ban.Note, &b.Ban.Source, &b.Ban.ValidUntil, &b.Ban.CreatedOn, &b.Ban.UpdatedOn,
			&b.Person.SteamID, &b.Person.CreatedOn, &b.Person.UpdatedOn,
			&b.Person.CommunityVisibilityState, &b.Person.ProfileState, &b.Person.PersonaName,
			&b.Person.ProfileURL, &b.Person.Avatar, &b.Person.AvatarMedium, &b.Person.AvatarFull,
			&b.Person.AvatarHash, &b.Person.PersonaState, &b.Person.RealName, &b.Person.TimeCreated, &b.Person.LocCountryCode,
			&b.Person.LocStateCode, &b.Person.LocCityID, &b.Person.PermissionLevel, &b.Person.DiscordID, &b.Person.CommunityBanned,
			&b.Person.VACBans, &b.Person.GameBans, &b.Person.EconomyBan, &b.Person.DaysSinceLastBan); err != nil {
		return Err(err)
	}
	if full {
		h, err := database.GetChatHistory(ctx, b.Person.SteamID, 25)
		if err == nil {
			b.HistoryChat = h
		}
		b.HistoryConnections = []string{}
		ips, _ := database.GetPersonIPHistory(ctx, b.Person.SteamID, 10000)
		b.HistoryIP = ips
		b.HistoryPersonaName = []string{}
	}
	return nil
}

func (database *pgStore) GetBanBySteamID(ctx context.Context, steamID steamid.SID64, full bool, p *model.BannedPerson) error {
	return database.getBanByColumn(ctx, "steam_id", steamID, full, p)
}

func (database *pgStore) GetBanByBanID(ctx context.Context, banID uint64, full bool, p *model.BannedPerson) error {
	return database.getBanByColumn(ctx, "ban_id", banID, full, p)
}

func (database *pgStore) GetAppeal(ctx context.Context, banID uint64, ap *model.Appeal) error {
	q, a, e := sb.Select("appeal_id", "ban_id", "appeal_text", "appeal_state",
		"email", "created_on", "updated_on").
		From("ban_appeal").
		Where(sq.Eq{"ban_id": banID}).
		ToSql()
	if e != nil {
		return e
	}

	if err := database.conn.QueryRow(ctx, q, a...).
		Scan(&ap.AppealID, &ap.BanID, &ap.AppealText, &ap.AppealState, &ap.Email, &ap.CreatedOn,
			&ap.UpdatedOn); err != nil {
		return err
	}
	return nil
}

func (database *pgStore) updateAppeal(ctx context.Context, appeal *model.Appeal) error {
	q, a, e := sb.Update("ban_appeal").
		Set("appeal_text", appeal.AppealText).
		Set("appeal_state", appeal.AppealState).
		Set("email", appeal.Email).
		Set("updated_on", appeal.UpdatedOn).
		Where(sq.Eq{"appeal_id": appeal.AppealID}).
		ToSql()
	if e != nil {
		return e
	}
	_, err := database.conn.Exec(ctx, q, a...)
	if err != nil {
		return Err(err)
	}
	return nil
}

func (database *pgStore) insertAppeal(ctx context.Context, ap *model.Appeal) error {
	q, a, e := sb.Insert("ban_appeal").
		Columns("ban_id", "appeal_text", "appeal_state", "email", "created_on", "updated_on").
		Values(ap.BanID, ap.AppealText, ap.AppealState, ap.Email, ap.CreatedOn, ap.UpdatedOn).
		Suffix("RETURNING appeal_id").
		ToSql()
	if e != nil {
		return e
	}
	err := database.conn.QueryRow(ctx, q, a...).Scan(&ap.AppealID)
	if err != nil {
		return Err(err)
	}
	return nil
}

func (database *pgStore) SaveAppeal(ctx context.Context, appeal *model.Appeal) error {
	appeal.UpdatedOn = config.Now()
	if appeal.AppealID > 0 {
		return database.updateAppeal(ctx, appeal)
	}
	appeal.CreatedOn = config.Now()
	return database.insertAppeal(ctx, appeal)
}

// SaveBan will insert or update the ban record
// New records will have the Ban.BanID set automatically
func (database *pgStore) SaveBan(ctx context.Context, ban *model.Ban) error {
	// Ensure the foreign keys are satisfied
	p := model.NewPerson(ban.SteamID)
	err := database.GetOrCreatePersonBySteamID(ctx, ban.SteamID, &p)
	if err != nil {
		return errors.Wrapf(err, "Failed to get person for ban")
	}
	a := model.NewPerson(ban.AuthorID)
	err2 := database.GetOrCreatePersonBySteamID(ctx, ban.AuthorID, &a)
	if err2 != nil {
		return errors.Wrapf(err, "Failed to get author for ban")
	}
	ban.UpdatedOn = config.Now()
	if ban.BanID > 0 {
		return database.updateBan(ctx, ban)
	}
	ban.CreatedOn = config.Now()
	existing := model.NewBannedPerson()
	e := database.GetBanBySteamID(ctx, ban.SteamID, false, &existing)
	if e != nil && !errors.Is(e, ErrNoResult) {
		return errors.Wrapf(err, "Failed to check existing ban state")
	}
	if ban.BanType <= existing.Ban.BanType {
		return ErrDuplicate
	}
	return database.insertBan(ctx, ban)
}

func (database *pgStore) insertBan(ctx context.Context, ban *model.Ban) error {
	const q = `
		INSERT INTO ban (steam_id, author_id, ban_type, reason, reason_text, note, valid_until, created_on, updated_on, ban_source)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING ban_id`
	err := database.conn.QueryRow(ctx, q, ban.SteamID, ban.AuthorID, ban.BanType, ban.Reason, ban.ReasonText,
		ban.Note, ban.ValidUntil, ban.CreatedOn, ban.UpdatedOn, ban.Source).Scan(&ban.BanID)
	if err != nil {
		return Err(err)
	}
	return nil
}

func (database *pgStore) updateBan(ctx context.Context, ban *model.Ban) error {
	const q = `
		UPDATE
		    ban
		SET
		    author_id = $2, reason = $3, reason_text = $4, note = $5, valid_until = $6, updated_on = $7, ban_source = $8, ban_type = $9
		WHERE ban_id = $1`
	if _, err := database.conn.Exec(ctx, q, ban.BanID, ban.AuthorID, ban.Reason, ban.ReasonText, ban.Note, ban.ValidUntil,
		ban.UpdatedOn, ban.Source, ban.BanType); err != nil {
		return Err(err)
	}
	return nil
}

func (database *pgStore) GetExpiredBans(ctx context.Context) ([]model.Ban, error) {
	const q = `SELECT ban_id, steam_id, author_id, ban_type, reason, reason_text,
       note, valid_until, ban_source, created_on, updated_on FROM ban
       WHERE valid_until < $1`
	var bans []model.Ban
	rows, err := database.conn.Query(ctx, q, config.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var b model.Ban
		if err2 := rows.Scan(&b.BanID, &b.SteamID, &b.AuthorID, &b.BanType, &b.Reason, &b.ReasonText, &b.Note,
			&b.ValidUntil, &b.Source, &b.CreatedOn, &b.UpdatedOn); err2 != nil {
			return nil, err2
		}
		bans = append(bans, b)
	}
	return bans, nil
}

//func GetBansTotal(o *QueryFilter) (int, error) {
//	q, _, e := sb.Select("count(*) as total_rows").From(string(tableBan)).ToSql()
//	if e != nil {
//		return 0, e
//	}
//	var total int
//	if err := db.QueryRow(context.Background(), q).Scan(&total); err != nil {
//		return 0, err
//	}
//	return total, nil
//}

// GetBans returns all bans that fit the filter criteria passed in
func (database *pgStore) GetBans(ctx context.Context, o *QueryFilter) ([]model.BannedPerson, error) {
	q := fmt.Sprintf(`SELECT
		b.ban_id, b.steam_id, b.author_id, b.ban_type, b.reason,
		b.reason_text, b.note, b.ban_source, b.valid_until, b.created_on, b.updated_on,
		p.steam_id as sid2, p.created_on as created_on2, p.updated_on as updated_on2, p.communityvisibilitystate,
		p.profilestate,
		p.personaname, p.profileurl, p.avatar, p.avatarmedium, p.avatarfull, p.avatarhash,
		p.personastate, p.realname, p.timecreated, p.loccountrycode, p.locstatecode, p.loccityid,
		p.permission_level, p.discord_id, p.community_banned, p.vac_bans, p.game_bans, p.economy_ban,
		p.days_since_last_ban
	FROM ban b
	LEFT OUTER JOIN person p on p.steam_id = b.steam_id
	ORDER BY b.%s LIMIT %d OFFSET %d`, o.OrderBy, o.Limit, o.Offset)
	var bans []model.BannedPerson
	rows, err := database.conn.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		b := model.NewBannedPerson()
		if errS := rows.Scan(&b.Ban.BanID, &b.Ban.SteamID, &b.Ban.AuthorID, &b.Ban.BanType, &b.Ban.Reason, &b.Ban.ReasonText,
			&b.Ban.Note, &b.Ban.Source, &b.Ban.ValidUntil, &b.Ban.CreatedOn, &b.Ban.UpdatedOn,
			&b.Person.SteamID, &b.Person.CreatedOn, &b.Person.UpdatedOn,
			&b.Person.CommunityVisibilityState, &b.Person.ProfileState, &b.Person.PersonaName, &b.Person.ProfileURL,
			&b.Person.Avatar, &b.Person.AvatarMedium, &b.Person.AvatarFull, &b.Person.AvatarHash,
			&b.Person.PersonaState, &b.Person.RealName, &b.Person.TimeCreated, &b.Person.LocCountryCode,
			&b.Person.LocStateCode, &b.Person.LocCityID, &b.Person.PermissionLevel,
			&b.Person.DiscordID, &b.Person.CommunityBanned, &b.Person.VACBans, &b.Person.GameBans, &b.Person.EconomyBan,
			&b.Person.DaysSinceLastBan); errS != nil {
			return nil, errS
		}
		bans = append(bans, b)
	}
	return bans, nil
}

func (database *pgStore) GetBansOlderThan(ctx context.Context, o *QueryFilter, t time.Time) ([]model.Ban, error) {
	q := fmt.Sprintf(`
		SELECT
			b.ban_id, b.steam_id, b.author_id, b.ban_type, b.reason,
			b.reason_text, b.note, b.valid_until, b.created_on, b.updated_on, b.ban_source
		FROM ban b
		WHERE updated_on < $1
		LIMIT %d
		OFFSET %d`, o.Limit, o.Offset)
	var bans []model.Ban
	rows, err := database.conn.Query(ctx, q, t)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var b model.Ban
		if err = rows.Scan(&b.BanID, &b.SteamID, &b.AuthorID, &b.BanType, &b.Reason, &b.ReasonText, &b.Note,
			&b.Source, &b.ValidUntil, &b.CreatedOn, &b.UpdatedOn); err != nil {
			return nil, err
		}
		bans = append(bans, b)
	}
	return bans, nil
}
