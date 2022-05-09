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

func (database *pgStore) DropPerson(ctx context.Context, steamID steamid.SID64) error {
	query, args, errQueryArgs := sb.Delete("person").Where(sq.Eq{"steam_id": steamID}).ToSql()
	if errQueryArgs != nil {
		return errQueryArgs
	}
	if _, errExec := database.conn.Exec(ctx, query, args...); errExec != nil {
		return Err(errExec)
	}
	return nil
}

// SavePerson will insert or update the person record
func (database *pgStore) SavePerson(ctx context.Context, person *model.Person) error {
	person.UpdatedOn = config.Now()
	if !person.IsNew {
		return database.updatePerson(ctx, person)
	}
	person.CreatedOn = person.UpdatedOn
	return database.insertPerson(ctx, person)
}

func (database *pgStore) updatePerson(ctx context.Context, person *model.Person) error {
	person.UpdatedOn = config.Now()
	const query = `
		UPDATE person 
		SET 
		    updated_on = $2, communityvisibilitystate = $3, profilestate = $4, personaname = $5, profileurl = $6, avatar = $7,
    		avatarmedium = $8, avatarfull = $9, avatarhash = $10, personastate = $11, realname = $12, timecreated = $13,
		    loccountrycode = $14, locstatecode = $15, loccityid = $16, permission_level = $17, discord_id = $18,
		    community_banned = $19, vac_bans = $20, game_bans = $21, economy_ban = $22, days_since_last_ban = $23,
			updated_on_steam = $24
		WHERE steam_id = $1`
	if _, errExec := database.conn.Exec(ctx, query, person.SteamID, person.UpdatedOn, person.PlayerSummary.CommunityVisibilityState,
		person.PlayerSummary.ProfileState, person.PlayerSummary.PersonaName, person.PlayerSummary.ProfileURL, person.PlayerSummary.Avatar,
		person.PlayerSummary.AvatarMedium, person.PlayerSummary.AvatarFull, person.PlayerSummary.AvatarHash,
		person.PlayerSummary.PersonaState, person.PlayerSummary.RealName, person.TimeCreated, person.PlayerSummary.LocCountryCode,
		person.PlayerSummary.LocStateCode, person.PlayerSummary.LocCityID, person.PermissionLevel, person.DiscordID,
		person.CommunityBanned, person.VACBans, person.GameBans, person.EconomyBan, person.DaysSinceLastBan, person.UpdatedOnSteam); errExec != nil {
		return Err(errExec)
	}
	return nil
}

func (database *pgStore) insertPerson(ctx context.Context, person *model.Person) error {
	query, args, errQueryArgs := sb.
		Insert("person").
		Columns(
			"created_on", "updated_on", "steam_id", "communityvisibilitystate",
			"profilestate", "personaname", "profileurl", "avatar", "avatarmedium", "avatarfull",
			"avatarhash", "personastate", "realname", "timecreated", "loccountrycode", "locstatecode",
			"loccityid", "permission_level", "discord_id", "community_banned", "vac_bans", "game_bans",
			"economy_ban", "days_since_last_ban", "updated_on_steam").
		Values(person.CreatedOn, person.UpdatedOn, person.SteamID,
			person.PlayerSummary.CommunityVisibilityState, person.PlayerSummary.ProfileState, person.PlayerSummary.PersonaName,
			person.PlayerSummary.ProfileURL,
			person.PlayerSummary.Avatar, person.PlayerSummary.AvatarMedium, person.PlayerSummary.AvatarFull, person.PlayerSummary.AvatarHash,
			person.PlayerSummary.PersonaState, person.PlayerSummary.RealName, person.PlayerSummary.TimeCreated,
			person.PlayerSummary.LocCountryCode, person.PlayerSummary.LocStateCode, person.PlayerSummary.LocCityID, person.PermissionLevel,
			person.DiscordID, person.CommunityBanned, person.VACBans, person.GameBans, person.EconomyBan, person.DaysSinceLastBan, person.UpdatedOnSteam).
		ToSql()
	if errQueryArgs != nil {
		return errQueryArgs
	}
	_, errExec := database.conn.Exec(ctx, query, args...)
	if errExec != nil {
		return Err(errExec)
	}
	person.IsNew = false
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
func (database *pgStore) GetPersonBySteamID(ctx context.Context, sid64 steamid.SID64, person *model.Person) error {
	const query = `
    	SELECT person.steam_id,
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
				 AND e.source_id = person.steam_id
			   ORDER BY e.created_on DESC
			   LIMIT 1
		   ),
		   community_banned,
		   vac_bans,
		   game_bans,
		   economy_ban,
		   days_since_last_ban,
		   updated_on_steam
	FROM person person
	WHERE person.steam_id = $1;`

	person.IsNew = false
	person.PlayerSummary = &steamweb.PlayerSummary{}
	errQuery := database.conn.QueryRow(ctx, query, sid64.Int64()).Scan(&person.SteamID, &person.CreatedOn, &person.UpdatedOn,
		&person.CommunityVisibilityState, &person.ProfileState, &person.PersonaName, &person.ProfileURL, &person.Avatar, &person.AvatarMedium,
		&person.AvatarFull, &person.AvatarHash, &person.PersonaState, &person.RealName, &person.TimeCreated, &person.LocCountryCode,
		&person.LocStateCode, &person.LocCityID, &person.PermissionLevel, &person.DiscordID, &person.IPAddr, &person.CommunityBanned,
		&person.VACBans, &person.GameBans, &person.EconomyBan, &person.DaysSinceLastBan, &person.UpdatedOnSteam)
	if errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func (database *pgStore) GetPeopleBySteamID(ctx context.Context, steamIds steamid.Collection) (model.People, error) {
	queryBuilder := sb.Select(profileColumns...).From("person").Where(sq.Eq{"steam_id": fp.Uniq[steamid.SID64](steamIds)})
	query, args, errQueryArgs := queryBuilder.ToSql()
	if errQueryArgs != nil {
		return nil, errQueryArgs
	}
	var people model.People
	rows, errQuery := database.conn.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	for rows.Next() {
		person := model.NewPerson(0)
		if errScan := rows.Scan(&person.SteamID, &person.CreatedOn, &person.UpdatedOn, &person.CommunityVisibilityState,
			&person.ProfileState, &person.PersonaName, &person.ProfileURL, &person.Avatar, &person.AvatarMedium, &person.AvatarFull, &person.AvatarHash,
			&person.PersonaState, &person.RealName, &person.TimeCreated, &person.LocCountryCode, &person.LocStateCode, &person.LocCityID,
			&person.PermissionLevel, &person.DiscordID, &person.CommunityBanned, &person.VACBans, &person.GameBans, &person.EconomyBan,
			&person.DaysSinceLastBan, &person.UpdatedOnSteam); errScan != nil {
			return nil, errScan
		}
		people = append(people, person)
	}
	return people, nil
}

func (database *pgStore) GetPeople(ctx context.Context, queryFilter *QueryFilter) (model.People, error) {
	queryBuilder := sb.Select(profileColumns...).From("person")
	if queryFilter.Query != "" {
		// TODO add lower-cased functional index to avoid tableName scan
		queryBuilder = queryBuilder.Where(sq.ILike{"personaname": strings.ToLower(queryFilter.Query)})
	}
	if queryFilter.Offset > 0 {
		queryBuilder = queryBuilder.Offset(queryFilter.Offset)
	}
	if queryFilter.OrderBy != "" {
		queryBuilder = queryBuilder.OrderBy(queryFilter.orderString())
	}
	if queryFilter.Limit == 0 {
		queryBuilder = queryBuilder.Limit(100)
	} else {
		queryBuilder = queryBuilder.Limit(uint64(queryFilter.Limit))
	}
	query, args, errQueryArgs := queryBuilder.ToSql()
	if errQueryArgs != nil {
		return nil, errQueryArgs
	}
	var people model.People
	rows, errQuery := database.conn.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	for rows.Next() {
		person := model.NewPerson(0)
		if errScan := rows.Scan(&person.SteamID, &person.CreatedOn, &person.UpdatedOn, &person.CommunityVisibilityState,
			&person.ProfileState, &person.PersonaName, &person.ProfileURL, &person.Avatar, &person.AvatarMedium, &person.AvatarFull, &person.AvatarHash,
			&person.PersonaState, &person.RealName, &person.TimeCreated, &person.LocCountryCode, &person.LocStateCode, &person.LocCityID,
			&person.PermissionLevel, &person.DiscordID, &person.CommunityBanned, &person.VACBans, &person.GameBans, &person.EconomyBan,
			&person.DaysSinceLastBan, &person.UpdatedOnSteam); errScan != nil {
			return nil, errScan
		}
		people = append(people, person)
	}
	return people, nil
}

// GetOrCreatePersonBySteamID returns a person by their steam_id, creating a new person if the steam_id
// does not exist.
func (database *pgStore) GetOrCreatePersonBySteamID(ctx context.Context, sid64 steamid.SID64, person *model.Person) error {
	errGetPerson := database.GetPersonBySteamID(ctx, sid64, person)
	if errGetPerson != nil && Err(errGetPerson) == ErrNoResult {
		// FIXME
		//person = model.NewPerson(sid64)
		person.SteamID = sid64
		person.IsNew = true
		return database.SavePerson(ctx, person)
	} else if errGetPerson != nil {
		return errGetPerson
	}
	return nil
}

// GetPersonByDiscordID returns a person by their discord_id
func (database *pgStore) GetPersonByDiscordID(ctx context.Context, discordId string, person *model.Person) error {
	query, args, errQueryArgs := sb.Select(profileColumns...).
		From("person").
		Where(sq.Eq{"discord_id": discordId}).
		ToSql()
	if errQueryArgs != nil {
		return errQueryArgs
	}
	person.IsNew = false
	person.PlayerSummary = &steamweb.PlayerSummary{}
	errQuery := database.conn.QueryRow(ctx, query, args...).Scan(&person.SteamID, &person.CreatedOn, &person.UpdatedOn,
		&person.CommunityVisibilityState, &person.ProfileState, &person.PersonaName, &person.ProfileURL, &person.Avatar, &person.AvatarMedium,
		&person.AvatarFull, &person.AvatarHash, &person.PersonaState, &person.RealName, &person.TimeCreated, &person.LocCountryCode,
		&person.LocStateCode, &person.LocCityID, &person.PermissionLevel, &person.DiscordID, &person.CommunityBanned, &person.VACBans, &person.GameBans,
		&person.EconomyBan, &person.DaysSinceLastBan, &person.UpdatedOnSteam)
	if errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func (database *pgStore) GetExpiredProfiles(ctx context.Context, limit int) ([]model.Person, error) {
	query := fmt.Sprintf(`SELECT steam_id, created_on, updated_on,
	communityvisibilitystate, profilestate, personaname, profileurl, avatar,
	avatarmedium, avatarfull, avatarhash, personastate, realname, timecreated,
	loccountrycode, locstatecode, loccityid, permission_level, discord_id,
	community_banned, vac_bans, game_bans, economy_ban, days_since_last_ban, updated_on_steam
	FROM person ORDER BY updated_on LIMIT %d`, limit)

	var people []model.Person
	rows, errQuery := database.conn.Query(ctx, query)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	for rows.Next() {
		person := model.NewPerson(0)
		if errScan := rows.Scan(&person.SteamID, &person.CreatedOn, &person.UpdatedOn, &person.CommunityVisibilityState,
			&person.ProfileState, &person.PersonaName, &person.ProfileURL, &person.Avatar, &person.AvatarMedium, &person.AvatarFull, &person.AvatarHash,
			&person.PersonaState, &person.RealName, &person.TimeCreated, &person.LocCountryCode, &person.LocStateCode, &person.LocCityID,
			&person.PermissionLevel, &person.DiscordID, &person.CommunityBanned, &person.VACBans, &person.GameBans, &person.EconomyBan,
			&person.DaysSinceLastBan, &person.UpdatedOnSteam); errScan != nil {
			return nil, errScan
		}
		people = append(people, person)
	}
	return people, nil
}

func (database *pgStore) GetChatHistory(ctx context.Context, sid64 steamid.SID64, limit int) ([]logparse.SayEvt, error) {
	query := `
		SELECT l.source_id, coalesce(p.personaname, ''), coalesce((l.meta_data->>'msg')::text, '')
		FROM server_log l
		LEFT JOIN person p on l.source_id = p.steam_id
		WHERE source_id = $1
		  AND (event_type = 10 OR event_type = 11) 
		ORDER BY l.created_on DESC`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, errQuery := database.conn.Query(ctx, query, sid64.String())
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	var hist []logparse.SayEvt
	for rows.Next() {
		var event logparse.SayEvt
		if errScan := rows.Scan(&event.SourcePlayer.SID, &event.SourcePlayer.Name, &event.Msg); errScan != nil {
			return nil, errScan
		}
		hist = append(hist, event)
	}
	return hist, nil
}

func (database *pgStore) GetPersonIPHistory(ctx context.Context, sid64 steamid.SID64, limit int) ([]model.PersonIPRecord, error) {
	var query = `
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
	rows, errQuery := database.conn.Query(ctx, query, sid64.Int64())
	if errQuery != nil {
		return nil, errQuery
	}
	var records []model.PersonIPRecord
	defer rows.Close()
	for rows.Next() {
		var ipRecord model.PersonIPRecord
		if errScan := rows.Scan(&ipRecord.IP, &ipRecord.CreatedOn, &ipRecord.CityName, &ipRecord.CountryName, &ipRecord.CountryCode, &ipRecord.ASName,
			&ipRecord.ASNum, &ipRecord.ISP, &ipRecord.UsageType, &ipRecord.Threat, &ipRecord.DomainUsed); errScan != nil {
			return nil, errScan
		}
		records = append(records, ipRecord)
	}
	return records, nil
}
