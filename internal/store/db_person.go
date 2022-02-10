package store

import (
	"context"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"strings"
)

func (db *pgStore) DropPerson(ctx context.Context, steamID steamid.SID64) error {
	q, a, e := sb.Delete("person").Where(sq.Eq{"steam_id": steamID}).ToSql()
	if e != nil {
		return e
	}
	if _, err := db.c.Exec(ctx, q, a...); err != nil {
		return Err(err)
	}
	return nil
}

// SavePerson will insert or update the person record
func (db *pgStore) SavePerson(ctx context.Context, person *model.Person) error {
	person.UpdatedOn = config.Now()
	if !person.IsNew {
		return db.updatePerson(ctx, person)
	}
	person.CreatedOn = person.UpdatedOn
	return db.insertPerson(ctx, person)
}

func (db *pgStore) updatePerson(ctx context.Context, p *model.Person) error {
	p.UpdatedOn = config.Now()
	const q = `
		UPDATE person 
		SET 
		    updated_on = $2, communityvisibilitystate = $3, profilestate = $4, personaname = $5, profileurl = $6, avatar = $7,
    		avatarmedium = $8, avatarfull = $9, avatarhash = $10, personastate = $11, realname = $12, timecreated = $13,
		    loccountrycode = $14, locstatecode = $15, loccityid = $16, permission_level = $17, discord_id = $18,
		    community_banned = $19, vac_bans = $20, game_bans = $21, economy_ban = $22, days_since_last_ban = $23,
			updated_on_steam = $24
		WHERE steam_id = $1`
	if _, err := db.c.Exec(ctx, q, p.SteamID, p.UpdatedOn, p.PlayerSummary.CommunityVisibilityState,
		p.PlayerSummary.ProfileState, p.PlayerSummary.PersonaName, p.PlayerSummary.ProfileURL, p.PlayerSummary.Avatar,
		p.PlayerSummary.AvatarMedium, p.PlayerSummary.AvatarFull, p.PlayerSummary.AvatarHash,
		p.PlayerSummary.PersonaState, p.PlayerSummary.RealName, p.TimeCreated, p.PlayerSummary.LocCountryCode,
		p.PlayerSummary.LocStateCode, p.PlayerSummary.LocCityID, p.PermissionLevel, p.DiscordID,
		p.CommunityBanned, p.VACBans, p.GameBans, p.EconomyBan, p.DaysSinceLastBan, p.UpdatedOnSteam); err != nil {
		return Err(err)
	}
	return nil
}

func (db *pgStore) insertPerson(ctx context.Context, p *model.Person) error {
	q, a, e := sb.
		Insert("person").
		Columns(
			"created_on", "updated_on", "steam_id", "communityvisibilitystate",
			"profilestate", "personaname", "profileurl", "avatar", "avatarmedium", "avatarfull",
			"avatarhash", "personastate", "realname", "timecreated", "loccountrycode", "locstatecode",
			"loccityid", "permission_level", "discord_id", "community_banned", "vac_bans", "game_bans",
			"economy_ban", "days_since_last_ban", "updated_on_steam").
		Values(p.CreatedOn, p.UpdatedOn, p.SteamID,
			p.PlayerSummary.CommunityVisibilityState, p.PlayerSummary.ProfileState, p.PlayerSummary.PersonaName,
			p.PlayerSummary.ProfileURL,
			p.PlayerSummary.Avatar, p.PlayerSummary.AvatarMedium, p.PlayerSummary.AvatarFull, p.PlayerSummary.AvatarHash,
			p.PlayerSummary.PersonaState, p.PlayerSummary.RealName, p.PlayerSummary.TimeCreated,
			p.PlayerSummary.LocCountryCode, p.PlayerSummary.LocStateCode, p.PlayerSummary.LocCityID, p.PermissionLevel,
			p.DiscordID, p.CommunityBanned, p.VACBans, p.GameBans, p.EconomyBan, p.DaysSinceLastBan, p.UpdatedOnSteam).
		ToSql()
	if e != nil {
		return e
	}
	_, err := db.c.Exec(ctx, q, a...)
	if err != nil {
		return Err(err)
	}
	p.IsNew = false
	return nil
}

//"community_banned", "vac_bans", "game_bans", "economy_ban", "days_since_last_ban"
var profileColumns = []string{"steam_id", "created_on", "updated_on",
	"communityvisibilitystate", "profilestate", "personaname", "profileurl", "avatar",
	"avatarmedium", "avatarfull", "avatarhash", "personastate", "realname", "timecreated",
	"loccountrycode", "locstatecode", "loccityid", "permission_level", "discord_id",
	"community_banned", "vac_bans", "game_bans", "economy_ban", "days_since_last_ban", "updated_on_steam"}

// GetPersonBySteamID returns a person by their steam_id. ErrNoResult is returned if the steam_id
// is not known.
func (db *pgStore) GetPersonBySteamID(ctx context.Context, sid steamid.SID64, p *model.Person) error {
	const q = `
    	SELECT p.steam_id,
		   created_on,
		   updated_on,
		   communityvisibilitystate,
		   profilestate,
		   personaname,
		   profileurl,
		   avatar,
		   avatarmedium,
		   avatarfull,
		   avatarhash,
		   personastate,
		   realname,
		   timecreated,
		   loccountrycode,
		   locstatecode,
		   loccityid,
		   permission_level,
		   discord_id,
		   (
			   SELECT (e.meta_data ->> 'address')::inet
			   FROM server_log e
			   WHERE e.event_type = 1004
				 AND e.source_id = p.steam_id
			   ORDER BY e.created_on DESC
			   LIMIT 1
		   ),
		   community_banned,
		   vac_bans,
		   game_bans,
		   economy_ban,
		   days_since_last_ban,
		   updated_on_steam
	FROM person p
	WHERE p.steam_id = $1;`

	p.IsNew = false
	p.PlayerSummary = &steamweb.PlayerSummary{}
	err := db.c.QueryRow(ctx, q, sid.Int64()).Scan(&p.SteamID, &p.CreatedOn, &p.UpdatedOn,
		&p.CommunityVisibilityState, &p.ProfileState, &p.PersonaName, &p.ProfileURL, &p.Avatar, &p.AvatarMedium,
		&p.AvatarFull, &p.AvatarHash, &p.PersonaState, &p.RealName, &p.TimeCreated, &p.LocCountryCode,
		&p.LocStateCode, &p.LocCityID, &p.PermissionLevel, &p.DiscordID, &p.IPAddr, &p.CommunityBanned,
		&p.VACBans, &p.GameBans, &p.EconomyBan, &p.DaysSinceLastBan, &p.UpdatedOnSteam)
	if err != nil {
		return Err(err)
	}
	return nil
}

func (db *pgStore) GetPeopleBySteamID(ctx context.Context, steamids steamid.Collection) (model.People, error) {
	qb := sb.Select(profileColumns...).From("person").Where(sq.Eq{"steam_id": fp.Uniq[steamid.SID64](steamids)})
	q, a, e := qb.ToSql()
	if e != nil {
		return nil, e
	}
	var people model.People
	rows, err := db.c.Query(ctx, q, a...)
	if err != nil {
		return nil, Err(err)
	}
	defer rows.Close()
	for rows.Next() {
		p := model.NewPerson(0)
		if err2 := rows.Scan(&p.SteamID, &p.CreatedOn, &p.UpdatedOn, &p.CommunityVisibilityState,
			&p.ProfileState, &p.PersonaName, &p.ProfileURL, &p.Avatar, &p.AvatarMedium, &p.AvatarFull, &p.AvatarHash,
			&p.PersonaState, &p.RealName, &p.TimeCreated, &p.LocCountryCode, &p.LocStateCode, &p.LocCityID,
			&p.PermissionLevel, &p.DiscordID, &p.CommunityBanned, &p.VACBans, &p.GameBans, &p.EconomyBan,
			&p.DaysSinceLastBan, &p.UpdatedOnSteam); err2 != nil {
			return nil, err2
		}
		people = append(people, p)
	}
	return people, nil
}

func (db *pgStore) GetPeople(ctx context.Context, qf *QueryFilter) (model.People, error) {
	qb := sb.Select(profileColumns...).From("person")
	if qf.Query != "" {
		// TODO add lower-cased functional index to avoid tableName scan
		qb = qb.Where(sq.ILike{"personaname": strings.ToLower(qf.Query)})
	}
	if qf.Offset > 0 {
		qb = qb.Offset(qf.Offset)
	}
	if qf.OrderBy != "" {
		qb = qb.OrderBy(qf.orderString())
	}
	if qf.Limit == 0 {
		qb = qb.Limit(100)
	} else {
		qb = qb.Limit(uint64(qf.Limit))
	}
	q, a, e := qb.ToSql()
	if e != nil {
		return nil, e
	}
	var people model.People
	rows, err := db.c.Query(ctx, q, a...)
	if err != nil {
		return nil, Err(err)
	}
	defer rows.Close()
	for rows.Next() {
		p := model.NewPerson(0)
		if errScan := rows.Scan(&p.SteamID, &p.CreatedOn, &p.UpdatedOn, &p.CommunityVisibilityState,
			&p.ProfileState, &p.PersonaName, &p.ProfileURL, &p.Avatar, &p.AvatarMedium, &p.AvatarFull, &p.AvatarHash,
			&p.PersonaState, &p.RealName, &p.TimeCreated, &p.LocCountryCode, &p.LocStateCode, &p.LocCityID,
			&p.PermissionLevel, &p.DiscordID, &p.CommunityBanned, &p.VACBans, &p.GameBans, &p.EconomyBan,
			&p.DaysSinceLastBan, &p.UpdatedOnSteam); errScan != nil {
			return nil, errScan
		}
		people = append(people, p)
	}
	return people, nil
}

// GetOrCreatePersonBySteamID returns a person by their steam_id, creating a new person if the steam_id
// does not exist.
func (db *pgStore) GetOrCreatePersonBySteamID(ctx context.Context, sid steamid.SID64, p *model.Person) error {
	err := db.GetPersonBySteamID(ctx, sid, p)
	if err != nil && Err(err) == ErrNoResult {
		// FIXME
		//p = model.NewPerson(sid)
		p.SteamID = sid
		p.IsNew = true
		return db.SavePerson(ctx, p)
	} else if err != nil {
		return err
	}
	return nil
}

// GetPersonByDiscordID returns a person by their discord_id
func (db *pgStore) GetPersonByDiscordID(ctx context.Context, did string, p *model.Person) error {
	q, a, e := sb.Select(profileColumns...).
		From("person").
		Where(sq.Eq{"discord_id": did}).
		ToSql()
	if e != nil {
		return e
	}
	p.IsNew = false
	p.PlayerSummary = &steamweb.PlayerSummary{}
	err := db.c.QueryRow(ctx, q, a...).Scan(&p.SteamID, &p.CreatedOn, &p.UpdatedOn,
		&p.CommunityVisibilityState, &p.ProfileState, &p.PersonaName, &p.ProfileURL, &p.Avatar, &p.AvatarMedium,
		&p.AvatarFull, &p.AvatarHash, &p.PersonaState, &p.RealName, &p.TimeCreated, &p.LocCountryCode,
		&p.LocStateCode, &p.LocCityID, &p.PermissionLevel, &p.DiscordID, &p.CommunityBanned, &p.VACBans, &p.GameBans,
		&p.EconomyBan, &p.DaysSinceLastBan, &p.UpdatedOnSteam)
	if err != nil {
		return Err(err)
	}
	return nil
}

func (db *pgStore) GetExpiredProfiles(ctx context.Context, limit int) ([]model.Person, error) {
	q := fmt.Sprintf(`SELECT steam_id, created_on, updated_on,
	communityvisibilitystate, profilestate, personaname, profileurl, avatar,
	avatarmedium, avatarfull, avatarhash, personastate, realname, timecreated,
	loccountrycode, locstatecode, loccityid, permission_level, discord_id,
	community_banned, vac_bans, game_bans, economy_ban, days_since_last_ban, updated_on_steam
	FROM person ORDER BY updated_on LIMIT %d`, limit)

	var people []model.Person
	rows, err := db.c.Query(ctx, q)
	if err != nil {
		return nil, Err(err)
	}
	defer rows.Close()
	for rows.Next() {
		p := model.NewPerson(0)
		if errScan := rows.Scan(&p.SteamID, &p.CreatedOn, &p.UpdatedOn, &p.CommunityVisibilityState,
			&p.ProfileState, &p.PersonaName, &p.ProfileURL, &p.Avatar, &p.AvatarMedium, &p.AvatarFull, &p.AvatarHash,
			&p.PersonaState, &p.RealName, &p.TimeCreated, &p.LocCountryCode, &p.LocStateCode, &p.LocCityID,
			&p.PermissionLevel, &p.DiscordID, &p.CommunityBanned, &p.VACBans, &p.GameBans, &p.EconomyBan,
			&p.DaysSinceLastBan, &p.UpdatedOnSteam); errScan != nil {
			return nil, errScan
		}
		people = append(people, p)
	}
	return people, nil
}

func (db *pgStore) GetChatHistory(ctx context.Context, sid64 steamid.SID64, limit int) ([]logparse.SayEvt, error) {
	q := `
		SELECT l.source_id, coalesce(p.personaname, ''), coalesce((l.meta_data->>'msg')::text, '')
	
		FROM server_log l
		LEFT JOIN person p on l.source_id = p.steam_id
		WHERE source_id = $1
		  AND (event_type = 10 OR event_type = 11) 
		ORDER BY l.created_on DESC`
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, errQuery := db.c.Query(ctx, q, sid64.String())
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	var hist []logparse.SayEvt
	for rows.Next() {
		var h logparse.SayEvt
		if errScan := rows.Scan(&h.SourcePlayer.SID, &h.SourcePlayer.Name, &h.Msg); errScan != nil {
			return nil, errScan
		}
		hist = append(hist, h)
	}
	return hist, nil
}

func (db *pgStore) GetPersonIPHistory(ctx context.Context, sid steamid.SID64, limit int) ([]model.PersonIPRecord, error) {
	var q = `
		SELECT
			(log.meta_data->>'address')::inet, 
		    log.created_on,
			loc.city_name,
		    loc.country_name, 
		    loc.country_code,
			asn.as_name, 
		    asn.as_num,
			coalesce(proxy.isp, ''), 
		    coalesce(proxy.usage_type, ''),
			coalesce(proxy.threat, ''),
		    coalesce(proxy.domain_used, '')
		FROM server_log log
		LEFT JOIN net_location loc 
		    ON (log.meta_data->>'address')::inet <@ loc.ip_range
		LEFT JOIN net_asn asn 
		    ON (log.meta_data->>'address')::inet <@ asn.ip_range
		LEFT JOIN net_proxy proxy 
		    ON (log.meta_data->>'address')::inet <@ proxy.ip_range
		WHERE event_type = 1004 AND log.source_id = $1`
	rows, err := db.c.Query(ctx, q, sid.Int64())
	if err != nil {
		return nil, err
	}
	var records []model.PersonIPRecord
	defer rows.Close()
	for rows.Next() {
		var r model.PersonIPRecord
		if errR := rows.Scan(&r.IP, &r.CreatedOn, &r.CityName, &r.CountryName, &r.CountryCode, &r.ASName,
			&r.ASNum, &r.ISP, &r.UsageType, &r.Threat, &r.DomainUsed); errR != nil {
			return nil, errR
		}
		records = append(records, r)
	}
	return records, nil
}
