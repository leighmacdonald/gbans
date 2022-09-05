package store

import (
	"context"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	"strings"
	"time"
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
			updated_on_steam = $24, muted = $25
		WHERE steam_id = $1`
	if _, errExec := database.conn.Exec(ctx, query, person.SteamID, person.UpdatedOn,
		person.PlayerSummary.CommunityVisibilityState, person.PlayerSummary.ProfileState,
		person.PlayerSummary.PersonaName, person.PlayerSummary.ProfileURL, person.PlayerSummary.Avatar,
		person.PlayerSummary.AvatarMedium, person.PlayerSummary.AvatarFull, person.PlayerSummary.AvatarHash,
		person.PlayerSummary.PersonaState, person.PlayerSummary.RealName, person.TimeCreated,
		person.PlayerSummary.LocCountryCode, person.PlayerSummary.LocStateCode, person.PlayerSummary.LocCityID,
		person.PermissionLevel, person.DiscordID, person.CommunityBanned, person.VACBans, person.GameBans,
		person.EconomyBan, person.DaysSinceLastBan, person.UpdatedOnSteam, person.Muted); errExec != nil {
		return Err(errExec)
	}
	return nil
}

func (database *pgStore) insertPerson(ctx context.Context, person *model.Person) error {
	query, args, errQueryArgs := sb.
		Insert("person").
		Columns("created_on", "updated_on", "steam_id", "communityvisibilitystate", "profilestate",
			"personaname", "profileurl", "avatar", "avatarmedium", "avatarfull", "avatarhash", "personastate",
			"realname", "timecreated", "loccountrycode", "locstatecode", "loccityid", "permission_level",
			"discord_id", "community_banned", "vac_bans", "game_bans", "economy_ban", "days_since_last_ban",
			"updated_on_steam", "muted").
		Values(person.CreatedOn, person.UpdatedOn, person.SteamID, person.PlayerSummary.CommunityVisibilityState,
			person.PlayerSummary.ProfileState, person.PlayerSummary.PersonaName, person.PlayerSummary.ProfileURL,
			person.PlayerSummary.Avatar, person.PlayerSummary.AvatarMedium, person.PlayerSummary.AvatarFull,
			person.PlayerSummary.AvatarHash, person.PlayerSummary.PersonaState, person.PlayerSummary.RealName,
			person.PlayerSummary.TimeCreated, person.PlayerSummary.LocCountryCode, person.PlayerSummary.LocStateCode,
			person.PlayerSummary.LocCityID, person.PermissionLevel, person.DiscordID, person.CommunityBanned,
			person.VACBans, person.GameBans, person.EconomyBan, person.DaysSinceLastBan, person.UpdatedOnSteam,
			person.Muted).
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
	"community_banned", "vac_bans", "game_bans", "economy_ban", "days_since_last_ban", "updated_on_steam",
	"muted"}

// GetPersonBySteamID returns a person by their steam_id. ErrNoResult is returned if the steam_id
// is not known.
func (database *pgStore) GetPersonBySteamID(ctx context.Context, sid64 steamid.SID64, person *model.Person) error {
	const query = `
    	SELECT p.steam_id,
			p.created_on,
			p.updated_on,
			p.communityvisibilitystate,
			p.profilestate,
			p.personaname,
			p.profileurl,
			p.avatar,
			p.avatarmedium,
			p.avatarfull,
			p.avatarhash,
			p.personastate,
			p.realname,
			p.timecreated,
			p.loccountrycode,
			p.locstatecode,
			p.loccityid,
			p.permission_level,
			p.discord_id,
			/*		   //(
			//   SELECT (e.meta_data ->> 'address')::inet
			//   FROM server_log e
			//   WHERE e.event_type = 1004
			//	 AND e.source_id = person.steam_id
			//   ORDER BY e.created_on DESC
			//   LIMIT 1
			//),*/
			p.community_banned,
			p.vac_bans,
			p.game_bans,
			p.economy_ban,
			p.days_since_last_ban,
			p.updated_on_steam,
			p.muted
	FROM person p
	WHERE p.steam_id = $1;`
	if !sid64.Valid() {
		return consts.ErrInvalidSID
	}
	person.IsNew = false
	person.PlayerSummary = &steamweb.PlayerSummary{}
	errQuery := database.conn.QueryRow(ctx, query, sid64.Int64()).Scan(&person.SteamID, &person.CreatedOn,
		&person.UpdatedOn, &person.CommunityVisibilityState, &person.ProfileState, &person.PersonaName,
		&person.ProfileURL, &person.Avatar, &person.AvatarMedium, &person.AvatarFull, &person.AvatarHash,
		&person.PersonaState, &person.RealName, &person.TimeCreated, &person.LocCountryCode, &person.LocStateCode,
		&person.LocCityID, &person.PermissionLevel, &person.DiscordID /*&person.IPAddr,*/, &person.CommunityBanned,
		&person.VACBans, &person.GameBans, &person.EconomyBan, &person.DaysSinceLastBan, &person.UpdatedOnSteam,
		&person.Muted)
	if errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

// TODO search cached people first?
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
			&person.ProfileState, &person.PersonaName, &person.ProfileURL, &person.Avatar, &person.AvatarMedium,
			&person.AvatarFull, &person.AvatarHash, &person.PersonaState, &person.RealName, &person.TimeCreated,
			&person.LocCountryCode, &person.LocStateCode, &person.LocCityID, &person.PermissionLevel, &person.DiscordID,
			&person.CommunityBanned, &person.VACBans, &person.GameBans, &person.EconomyBan, &person.DaysSinceLastBan,
			&person.UpdatedOnSteam, &person.Muted); errScan != nil {
			return nil, errScan
		}
		people = append(people, person)
	}
	return people, nil
}

func (database *pgStore) GetPeople(ctx context.Context, queryFilter QueryFilter) (model.People, error) {
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
			&person.ProfileState, &person.PersonaName, &person.ProfileURL, &person.Avatar,
			&person.AvatarMedium, &person.AvatarFull, &person.AvatarHash, &person.PersonaState,
			&person.RealName, &person.TimeCreated, &person.LocCountryCode, &person.LocStateCode,
			&person.LocCityID, &person.PermissionLevel, &person.DiscordID, &person.CommunityBanned,
			&person.VACBans, &person.GameBans, &person.EconomyBan, &person.DaysSinceLastBan,
			&person.UpdatedOnSteam, &person.Muted); errScan != nil {
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
		newPerson := model.NewPerson(sid64)
		person = &newPerson
		return database.SavePerson(ctx, person)
	}
	return errGetPerson
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
	errQuery := database.conn.QueryRow(ctx, query, args...).Scan(&person.SteamID, &person.CreatedOn,
		&person.UpdatedOn, &person.CommunityVisibilityState, &person.ProfileState, &person.PersonaName,
		&person.ProfileURL, &person.Avatar, &person.AvatarMedium, &person.AvatarFull, &person.AvatarHash,
		&person.PersonaState, &person.RealName, &person.TimeCreated, &person.LocCountryCode, &person.LocStateCode,
		&person.LocCityID, &person.PermissionLevel, &person.DiscordID, &person.CommunityBanned, &person.VACBans,
		&person.GameBans, &person.EconomyBan, &person.DaysSinceLastBan, &person.UpdatedOnSteam, &person.Muted)
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
	community_banned, vac_bans, game_bans, economy_ban, days_since_last_ban, updated_on_steam, muted
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
			&person.ProfileState, &person.PersonaName, &person.ProfileURL, &person.Avatar, &person.AvatarMedium,
			&person.AvatarFull, &person.AvatarHash, &person.PersonaState, &person.RealName, &person.TimeCreated,
			&person.LocCountryCode, &person.LocStateCode, &person.LocCityID, &person.PermissionLevel,
			&person.DiscordID, &person.CommunityBanned, &person.VACBans, &person.GameBans,
			&person.EconomyBan, &person.DaysSinceLastBan, &person.UpdatedOnSteam, &person.Muted); errScan != nil {
			return nil, errScan
		}
		people = append(people, person)
	}
	return people, nil
}

func (database *pgStore) AddChatHistory(ctx context.Context, message *model.PersonMessage) error {
	const q = `INSERT INTO person_messages 
    		(steam_id, server_id, body, team, created_on, persona_name) 
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING person_message_id`
	if errScan := database.QueryRow(ctx, q, message.SteamId, message.ServerId, message.Body, message.Team,
		message.CreatedOn, message.PersonaName).
		Scan(&message.PersonMessageId); errScan != nil {
		return Err(errScan)
	}
	return nil
}

func (database *pgStore) GetPersonMessageById(ctx context.Context, personMessageId int64, msg *model.PersonMessage) error {
	query, args, errQuery := sb.Select(
		"m.person_message_id",
		"m.steam_id",
		"m.server_id",
		"m.body",
		"m.team",
		"m.created_on",
		"m.persona_name",
		"s.short_name").
		From("person_messages m").
		LeftJoin("server s on m.server_id = s.server_id").
		Where(sq.Eq{"m.person_message_id": personMessageId}).
		ToSql()
	if errQuery != nil {
		return errors.Wrap(errQuery, "Failed to create query")
	}
	return Err(database.QueryRow(ctx, query, args...).
		Scan(&msg.PersonMessageId,
			&msg.SteamId,
			&msg.ServerId,
			&msg.Body,
			&msg.Team,
			&msg.CreatedOn,
			&msg.PersonaName,
			&msg.ServerName))
}

type ChatHistoryQueryFilter struct {
	QueryFilter
	// TODO Index this string query
	PersonaName string `json:"persona_name,omitempty"`
	SteamId     string `json:"steam_id,omitempty"`
	// TODO Index this body query
	ServerId   int        `json:"server_id,omitempty"`
	SentAfter  *time.Time `json:"sent_after,omitempty"`
	SentBefore *time.Time `json:"sent_before,omitempty"`
}

func (database *pgStore) QueryChatHistory(ctx context.Context, query ChatHistoryQueryFilter) (model.PersonMessages, error) {
	qb := sb.Select(
		"m.person_message_id",
		"m.steam_id",
		"m.server_id",
		"m.body",
		"m.team",
		"m.created_on",
		"m.persona_name",
		"s.short_name").
		From("person_messages m").
		LeftJoin("server s on m.server_id = s.server_id")
	if query.Offset > 0 {
		qb = qb.Offset(query.Offset)
	}
	if query.Limit > 0 {
		qb = qb.Limit(query.Limit)
	}
	if query.OrderBy != "" {
		if query.SortDesc {
			qb = qb.OrderBy(query.OrderBy + " DESC")
		} else {
			qb = qb.OrderBy(query.OrderBy + " ASC")
		}
	}
	if query.ServerId > 0 {
		qb = qb.Where(sq.Eq{"m.server_id": query.ServerId})
	}
	if query.SteamId != "" {
		qb = qb.Where(sq.Eq{"m.steam_id": query.SteamId})
	}
	if query.PersonaName != "" {
		qb = qb.Where(sq.ILike{"m.persona_name": fmt.Sprintf("%%%s%%", strings.ToLower(query.PersonaName))})
	}
	if query.Query != "" {
		qb = qb.Where(sq.ILike{"m.body": fmt.Sprintf("%%%s%%", strings.ToLower(query.Query))})
	}
	if query.SentBefore != nil {
		qb = qb.Where(sq.Lt{"m.created_on": query.SentBefore})
	}
	if query.SentAfter != nil {
		qb = qb.Where(sq.Gt{"m.created_on": query.SentAfter})
	}
	q, a, qErr := qb.ToSql()
	if qErr != nil {
		return nil, errors.Wrap(qErr, "Failed to build query")
	}
	rows, errQuery := database.Query(ctx, q, a...)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	var messages model.PersonMessages
	for rows.Next() {
		var message model.PersonMessage
		if errScan := rows.Scan(
			&message.PersonMessageId,
			&message.SteamId,
			&message.ServerId,
			&message.Body,
			&message.Team,
			&message.CreatedOn,
			&message.PersonaName,
			&message.ServerName,
		); errScan != nil {
			return nil, Err(errScan)
		}
		messages = append(messages, message)
	}
	return messages, nil
}

func (database *pgStore) GetPersonIPHistory(ctx context.Context, sid64 steamid.SID64, limit int) (model.PersonConnections, error) {
	qb := sb.Select("pc.person_connection_id",
		"pc.steam_id",
		"pc.ip_addr",
		"coalesce(pc.persona_name, pc.steam_id::text)",
		"pc.created_on",
		"coalesce(loc.city_name, '')",
		"coalesce(loc.country_name, '')",
		"coalesce(loc.country_code, '')").
		From("person_connections pc").
		LeftJoin("net_location loc ON pc.ip_addr <@ loc.ip_range").
		//Join("LEFT JOIN net_asn asn ON pc.ip_addr <@ asn.ip_range").
		//Join("LEFT JOIN net_proxy proxy ON pc.ip_addr <@ proxy.ip_range").
		OrderBy("pc.person_connection_id DESC").
		Limit(1000)
	qb = qb.Where(sq.Eq{"pc.steam_id": sid64.Int64()})
	query, args, errCreateQuery := qb.ToSql()
	if errCreateQuery != nil {
		return nil, errors.Wrap(errCreateQuery, "Failed to build query")
	}
	rows, errQuery := database.conn.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	var connections model.PersonConnections
	for rows.Next() {
		var c model.PersonConnection
		if errScan := rows.Scan(&c.PersonConnectionId, &c.SteamId, &c.IPAddr, &c.PersonaName, &c.CreatedOn,
			&c.IPInfo.CityName, &c.IPInfo.CountryName, &c.IPInfo.CountryCode,
		); errScan != nil {
			return nil, Err(errScan)
		}
		connections = append(connections, c)
	}
	return connections, nil
}

func (database *pgStore) AddConnectionHistory(ctx context.Context, conn *model.PersonConnection) error {
	const q = `
		INSERT INTO person_connections (steam_id, ip_addr, persona_name, created_on) 
		VALUES ($1, $2, $3, $4) RETURNING person_connection_id`
	if errQuery := database.QueryRow(ctx, q, conn.SteamId, conn.IPAddr, conn.PersonaName, conn.CreatedOn).
		Scan(&conn.PersonConnectionId); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}
