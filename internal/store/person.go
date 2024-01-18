package store

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
)

func DropPerson(ctx context.Context, database Store, steamID steamid.SID64) error {
	return DBErr(database.ExecDeleteBuilder(ctx, database.
		Builder().
		Delete("person").
		Where(sq.Eq{"steam_id": steamID.Int64()})))
}

// SavePerson will insert or update the person record.
func SavePerson(ctx context.Context, database Store, person *model.Person) error {
	person.UpdatedOn = time.Now()
	// FIXME
	if person.PermissionLevel == 0 {
		person.PermissionLevel = 10
	}

	if !person.IsNew {
		return updatePerson(ctx, database, person)
	}

	person.CreatedOn = person.UpdatedOn

	return insertPerson(ctx, database, person)
}

func updatePerson(ctx context.Context, database Store, person *model.Person) error {
	person.UpdatedOn = time.Now()

	return DBErr(database.
		ExecUpdateBuilder(ctx, database.
			Builder().
			Update("person").
			SetMap(map[string]interface{}{
				"updated_on":               person.UpdatedOn,
				"communityvisibilitystate": person.CommunityVisibilityState,
				"profilestate":             person.ProfileState,
				"personaname":              person.PersonaName,
				"profileurl":               person.ProfileURL,
				"avatar":                   person.PlayerSummary.Avatar,
				"avatarmedium":             person.PlayerSummary.AvatarMedium,
				"avatarfull":               person.PlayerSummary.AvatarFull,
				"avatarhash":               person.PlayerSummary.AvatarHash,
				"personastate":             person.PlayerSummary.PersonaState,
				"realname":                 person.PlayerSummary.RealName,
				"timecreated":              person.TimeCreated,
				"loccountrycode":           person.PlayerSummary.LocCountryCode,
				"locstatecode":             person.PlayerSummary.LocStateCode,
				"loccityid":                person.PlayerSummary.LocCityID,
				"permission_level":         person.PermissionLevel,
				"discord_id":               person.DiscordID,
				"community_banned":         person.CommunityBanned,
				"vac_bans":                 person.VACBans,
				"game_bans":                person.GameBans,
				"economy_ban":              person.EconomyBan,
				"days_since_last_ban":      person.DaysSinceLastBan,
				"updated_on_steam":         person.UpdatedOnSteam,
				"muted":                    person.Muted,
			}).
			Where(sq.Eq{"steam_id": person.SteamID.Int64()})))
}

func insertPerson(ctx context.Context, database Store, person *model.Person) error {
	errExec := database.ExecInsertBuilder(ctx, database.
		Builder().
		Insert("person").
		Columns("created_on", "updated_on", "steam_id", "communityvisibilitystate", "profilestate",
			"personaname", "profileurl", "avatar", "avatarmedium", "avatarfull", "avatarhash", "personastate",
			"realname", "timecreated", "loccountrycode", "locstatecode", "loccityid", "permission_level",
			"discord_id", "community_banned", "vac_bans", "game_bans", "economy_ban", "days_since_last_ban",
			"updated_on_steam", "muted").
		Values(person.CreatedOn, person.UpdatedOn, person.SteamID.Int64(), person.PlayerSummary.CommunityVisibilityState,
			person.PlayerSummary.ProfileState, person.PlayerSummary.PersonaName, person.PlayerSummary.ProfileURL,
			person.PlayerSummary.Avatar, person.PlayerSummary.AvatarMedium, person.PlayerSummary.AvatarFull,
			person.PlayerSummary.AvatarHash, person.PlayerSummary.PersonaState, person.PlayerSummary.RealName,
			person.PlayerSummary.TimeCreated, person.PlayerSummary.LocCountryCode, person.PlayerSummary.LocStateCode,
			person.PlayerSummary.LocCityID, person.PermissionLevel, person.DiscordID, person.CommunityBanned,
			person.VACBans, person.GameBans, person.EconomyBan, person.DaysSinceLastBan, person.UpdatedOnSteam,
			person.Muted))
	if errExec != nil {
		return DBErr(errExec)
	}

	person.IsNew = false

	return nil
}

// "community_banned", "vac_bans", "game_bans", "economy_ban", "days_since_last_ban".
var profileColumns = []string{ //nolint:gochecknoglobals
	"steam_id", "created_on", "updated_on",
	"communityvisibilitystate", "profilestate", "personaname", "profileurl", "avatar",
	"avatarmedium", "avatarfull", "avatarhash", "personastate", "realname", "timecreated",
	"loccountrycode", "locstatecode", "loccityid", "permission_level", "discord_id",
	"community_banned", "vac_bans", "game_bans", "economy_ban", "days_since_last_ban", "updated_on_steam",
	"muted",
}

// GetPersonBySteamID returns a person by their steam_id. ErrNoResult is returned if the steam_id
// is not known.
func GetPersonBySteamID(ctx context.Context, database Store, sid64 steamid.SID64, person *model.Person) error {
	if !sid64.Valid() {
		return consts.ErrInvalidSID
	}

	row, errRow := database.QueryRowBuilder(ctx, database.
		Builder().
		Select("p.created_on",
			"p.updated_on",
			"p.communityvisibilitystate",
			"p.profilestate",
			"p.personaname",
			"p.profileurl",
			"p.avatar",
			"p.avatarmedium",
			"p.avatarfull",
			"p.avatarhash",
			"p.personastate",
			"p.realname",
			"p.timecreated",
			"p.loccountrycode",
			"p.locstatecode",
			"p.loccityid",
			"p.permission_level",
			"p.discord_id",
			"p.community_banned",
			"p.vac_bans",
			"p.game_bans",
			"p.economy_ban",
			"p.days_since_last_ban",
			"p.updated_on_steam",
			"p.muted").
		From("person p").
		Where(sq.Eq{"p.steam_id": sid64.Int64()}))

	if errRow != nil {
		return DBErr(errRow)
	}

	person.IsNew = false
	person.PlayerSummary = &steamweb.PlayerSummary{}
	person.SteamID = sid64

	return DBErr(row.Scan(&person.CreatedOn,
		&person.UpdatedOn, &person.CommunityVisibilityState, &person.ProfileState, &person.PersonaName,
		&person.ProfileURL, &person.Avatar, &person.AvatarMedium, &person.AvatarFull, &person.AvatarHash,
		&person.PersonaState, &person.RealName, &person.TimeCreated, &person.LocCountryCode, &person.LocStateCode,
		&person.LocCityID, &person.PermissionLevel, &person.DiscordID, &person.CommunityBanned,
		&person.VACBans, &person.GameBans, &person.EconomyBan, &person.DaysSinceLastBan, &person.UpdatedOnSteam,
		&person.Muted))
}

func GetPeopleBySteamID(ctx context.Context, database Store, steamIds steamid.Collection) (model.People, error) {
	var ids []int64 //nolint:prealloc
	for _, sid := range fp.Uniq[steamid.SID64](steamIds) {
		ids = append(ids, sid.Int64())
	}

	var people model.People

	rows, errQuery := database.QueryBuilder(ctx, database.
		Builder().
		Select(profileColumns...).
		From("person").
		Where(sq.Eq{"steam_id": ids}))
	if errQuery != nil {
		return nil, DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			steamID int64
			person  = model.NewPerson("")
		)

		if errScan := rows.Scan(&steamID, &person.CreatedOn, &person.UpdatedOn, &person.CommunityVisibilityState,
			&person.ProfileState, &person.PersonaName, &person.ProfileURL, &person.Avatar, &person.AvatarMedium,
			&person.AvatarFull, &person.AvatarHash, &person.PersonaState, &person.RealName, &person.TimeCreated,
			&person.LocCountryCode, &person.LocStateCode, &person.LocCityID, &person.PermissionLevel, &person.DiscordID,
			&person.CommunityBanned, &person.VACBans, &person.GameBans, &person.EconomyBan, &person.DaysSinceLastBan,
			&person.UpdatedOnSteam, &person.Muted); errScan != nil {
			return nil, errors.Wrapf(errScan, "Failedto scan person")
		}

		person.SteamID = steamid.New(steamID)

		people = append(people, person)
	}

	return people, nil
}

func normalizeStringLikeQuery(input string) string {
	space := regexp.MustCompile(`\s+`)

	return fmt.Sprintf("%%%s%%", strings.ToLower(strings.Trim(space.ReplaceAllString(input, "%"), "%")))
}

func GetSteamsAtAddress(ctx context.Context, database Store, addr net.IP) (steamid.Collection, error) {
	var ids steamid.Collection

	// TODO
	rows, errRows := database.QueryBuilder(ctx, database.
		Builder().
		Select("DISTINCT steam_id").
		From("person_connections").
		Where(sq.Expr(fmt.Sprintf("ip_addr::inet >>= '::ffff:%s'::CIDR OR ip_addr::inet <<= '::ffff:%s'::CIDR", addr.String(), addr.String()))))
	if errRows != nil {
		return nil, DBErr(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var sid int64
		if errScan := rows.Scan(&sid); errScan != nil {
			return nil, DBErr(errScan)
		}

		ids = append(ids, steamid.New(sid))
	}

	return ids, nil
}

type PlayerQuery struct {
	QueryFilter
	SteamID     model.StringSID `json:"steam_id"`
	Personaname string          `json:"personaname"`
	IP          string          `json:"ip"`
}

func GetPeople(ctx context.Context, database Store, filter PlayerQuery) (model.People, int64, error) {
	builder := database.
		Builder().
		Select("p.steam_id", "p.created_on", "p.updated_on",
			"p.communityvisibilitystate", "p.profilestate", "p.personaname", "p.profileurl", "p.avatar",
			"p.avatarmedium", "p.avatarfull", "p.avatarhash", "p.personastate", "p.realname", "p.timecreated",
			"p.loccountrycode", "p.locstatecode", "p.loccityid", "p.permission_level", "p.discord_id",
			"p.community_banned", "p.vac_bans", "p.game_bans", "p.economy_ban", "p.days_since_last_ban",
			"p.updated_on_steam", "p.muted").
		From("person p")

	conditions := sq.And{}

	if filter.IP != "" {
		addr := net.ParseIP(filter.IP)
		if addr == nil {
			return nil, 0, errors.New("Invalid IP")
		}

		foundIds, errFoundIds := GetSteamsAtAddress(ctx, database, addr)
		if errFoundIds != nil {
			if errors.Is(errFoundIds, ErrNoResult) {
				return model.People{}, 0, nil
			}

			return nil, 0, DBErr(errFoundIds)
		}

		conditions = append(conditions, sq.Eq{"p.steam_id": foundIds})
	}

	if filter.SteamID != "" {
		steamID, errSteamID := filter.SteamID.SID64(ctx)
		if errSteamID != nil {
			return nil, 0, errors.Wrap(errSteamID, "Invalid Steam ID")
		}

		conditions = append(conditions, sq.Eq{"p.steam_id": steamID.Int64()})
	}

	if filter.Personaname != "" {
		// TODO add lower-cased functional index to avoid table scan
		conditions = append(conditions, sq.ILike{"p.personaname": normalizeStringLikeQuery(filter.Personaname)})
	}

	builder = filter.applyLimitOffsetDefault(builder)
	builder = filter.applySafeOrder(builder, map[string][]string{
		"p.": {
			"steam_id", "created_on", "updated_on",
			"communityvisibilitystate", "profilestate", "personaname", "profileurl", "avatar",
			"avatarmedium", "avatarfull", "avatarhash", "personastate", "realname", "timecreated",
			"loccountrycode", "locstatecode", "loccityid", "p.permission_level", "discord_id",
			"community_banned", "vac_bans", "game_bans", "economy_ban", "days_since_last_ban",
			"updated_on_steam", "muted",
		},
	}, "steam_id")

	var people model.People

	rows, errQuery := database.QueryBuilder(ctx, builder.Where(conditions))
	if errQuery != nil {
		return nil, 0, DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			person  = model.NewPerson("")
			steamID int64
		)

		if errScan := rows.
			Scan(&steamID, &person.CreatedOn, &person.UpdatedOn, &person.CommunityVisibilityState,
				&person.ProfileState, &person.PersonaName, &person.ProfileURL, &person.Avatar,
				&person.AvatarMedium, &person.AvatarFull, &person.AvatarHash, &person.PersonaState,
				&person.RealName, &person.TimeCreated, &person.LocCountryCode, &person.LocStateCode,
				&person.LocCityID, &person.PermissionLevel, &person.DiscordID, &person.CommunityBanned,
				&person.VACBans, &person.GameBans, &person.EconomyBan, &person.DaysSinceLastBan,
				&person.UpdatedOnSteam, &person.Muted); errScan != nil {
			return nil, 0, errors.Wrapf(errScan, "Failed to scan person")
		}

		person.SteamID = steamid.New(steamID)

		people = append(people, person)
	}

	count, errCount := getCount(ctx, database, database.
		Builder().
		Select("COUNT(p.steam_id)").
		From("person p").
		Where(conditions))
	if errCount != nil {
		return nil, 0, errors.Wrap(errCount, "Failed to exec count query")
	}

	return people, count, nil
}

// GetOrCreatePersonBySteamID returns a person by their steam_id, creating a new person if the steam_id
// does not exist.
func GetOrCreatePersonBySteamID(ctx context.Context, database Store, sid64 steamid.SID64, person *model.Person) error {
	errGetPerson := GetPersonBySteamID(ctx, database, sid64, person)
	if errGetPerson != nil && errors.Is(DBErr(errGetPerson), ErrNoResult) {
		// FIXME
		newPerson := model.NewPerson(sid64)
		*person = newPerson

		return SavePerson(ctx, database, person)
	}

	return errGetPerson
}

// GetPersonByDiscordID returns a person by their discord_id.
func GetPersonByDiscordID(ctx context.Context, database Store, discordID string, person *model.Person) error {
	var steamID int64

	person.IsNew = false
	person.PlayerSummary = &steamweb.PlayerSummary{}

	row, errRow := database.QueryRowBuilder(ctx, database.
		Builder().
		Select(profileColumns...).
		From("person").
		Where(sq.Eq{"discord_id": discordID}))
	if errRow != nil {
		return DBErr(errRow)
	}

	errQuery := row.Scan(&steamID, &person.CreatedOn,
		&person.UpdatedOn, &person.CommunityVisibilityState, &person.ProfileState, &person.PersonaName,
		&person.ProfileURL, &person.Avatar, &person.AvatarMedium, &person.AvatarFull, &person.AvatarHash,
		&person.PersonaState, &person.RealName, &person.TimeCreated, &person.LocCountryCode, &person.LocStateCode,
		&person.LocCityID, &person.PermissionLevel, &person.DiscordID, &person.CommunityBanned, &person.VACBans,
		&person.GameBans, &person.EconomyBan, &person.DaysSinceLastBan, &person.UpdatedOnSteam, &person.Muted)
	if errQuery != nil {
		return DBErr(errQuery)
	}

	person.SteamID = steamid.New(steamID)

	return nil
}

func GetExpiredProfiles(ctx context.Context, database Store, limit uint64) ([]model.Person, error) {
	var people []model.Person

	rows, errQuery := database.QueryBuilder(ctx, database.
		Builder().
		Select("steam_id", "created_on", "updated_on",
			"communityvisibilitystate", "profilestate", "personaname", "profileurl", "avatar",
			"avatarmedium", "avatarfull", "avatarhash", "personastate", "realname", "timecreated",
			"loccountrycode", "locstatecode", "loccityid", "permission_level", "discord_id",
			"community_banned", "vac_bans", "game_bans", "economy_ban", "days_since_last_ban", "updated_on_steam",
			"muted").
		From("person").
		OrderBy("updated_on_steam ASC").
		Where(sq.Lt{"updated_on_steam": time.Now().AddDate(0, 0, -30)}).
		Limit(limit))
	if errQuery != nil {
		return nil, DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			person  = model.NewPerson("")
			steamID int64
		)

		if errScan := rows.Scan(&steamID, &person.CreatedOn, &person.UpdatedOn, &person.CommunityVisibilityState,
			&person.ProfileState, &person.PersonaName, &person.ProfileURL, &person.Avatar, &person.AvatarMedium,
			&person.AvatarFull, &person.AvatarHash, &person.PersonaState, &person.RealName, &person.TimeCreated,
			&person.LocCountryCode, &person.LocStateCode, &person.LocCityID, &person.PermissionLevel,
			&person.DiscordID, &person.CommunityBanned, &person.VACBans, &person.GameBans,
			&person.EconomyBan, &person.DaysSinceLastBan, &person.UpdatedOnSteam, &person.Muted); errScan != nil {
			return nil, DBErr(errScan)
		}

		person.SteamID = steamid.New(steamID)

		people = append(people, person)
	}

	return people, nil
}

func AddChatHistory(ctx context.Context, database Store, message *model.PersonMessage) error {
	const query = `INSERT INTO person_messages 
    		(steam_id, server_id, body, team, created_on, persona_name, match_id) 
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING person_message_id`

	if errScan := database.
		QueryRow(ctx, query, message.SteamID.Int64(), message.ServerID, message.Body, message.Team,
			message.CreatedOn, message.PersonaName, message.MatchID).
		Scan(&message.PersonMessageID); errScan != nil {
		return DBErr(errScan)
	}

	return nil
}

func GetPersonMessageByID(ctx context.Context, database Store, personMessageID int64, msg *model.PersonMessage) error {
	row, errRow := database.QueryRowBuilder(ctx, database.
		Builder().
		Select(
			"m.person_message_id",
			"m.steam_id",
			"m.server_id",
			"m.body",
			"m.team",
			"m.created_on",
			"m.persona_name",
			"m.match_id",
			"s.short_name").
		From("person_messages m").
		LeftJoin("server s on m.server_id = s.server_id").
		Where(sq.Eq{"m.person_message_id": personMessageID}))

	if errRow != nil {
		return DBErr(errRow)
	}

	var steamID int64

	if errScan := row.Scan(&msg.PersonMessageID,
		&steamID,
		&msg.ServerID,
		&msg.Body,
		&msg.Team,
		&msg.CreatedOn,
		&msg.PersonaName,
		&msg.MatchID,
		&msg.ServerName); errScan != nil {
		return DBErr(errScan)
	}

	msg.SteamID = steamid.New(steamID)

	return nil
}

type ConnectionHistoryQueryFilter struct {
	QueryFilter
	IP       string          `json:"ip"`
	SourceID model.StringSID `json:"source_id"`
}

func QueryConnectionHistory(ctx context.Context, database Store, opts ConnectionHistoryQueryFilter) ([]model.PersonConnection, int64, error) {
	builder := database.
		Builder().
		Select("c.person_connection_id", "c.steam_id",
			"c.ip_addr", "c.persona_name", "c.created_on", "c.server_id", "s.short_name", "s.name").
		From("person_connections c").
		LeftJoin("server s USING(server_id)").
		GroupBy("c.person_connection_id, c.ip_addr, s.short_name", "s.name")

	var constraints sq.And

	if opts.SourceID != "" {
		sid, errSID := opts.SourceID.SID64(ctx)
		if errSID != nil {
			return nil, 0, errors.Wrap(steamid.ErrInvalidSID, "Invalid steam id in query")
		}

		constraints = append(constraints, sq.Eq{"c.steam_id": sid.Int64()})
	}

	builder = opts.applySafeOrder(opts.applyLimitOffsetDefault(builder), map[string][]string{
		"c.": {"person_connection_id", "steam_id", "ip_addr", "persona_name", "created_on"},
		"s.": {"short_name", "name"},
	}, "person_connection_id")

	var messages []model.PersonConnection

	rows, errQuery := database.QueryBuilder(ctx, builder.Where(constraints))
	if errQuery != nil {
		return nil, 0, DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			connHistory model.PersonConnection
			steamID     int64
			serverID    *int
			shortName   *string
			name        *string
		)

		if errScan := rows.Scan(&connHistory.PersonConnectionID,
			&steamID,
			&connHistory.IPAddr,
			&connHistory.PersonaName,
			&connHistory.CreatedOn,
			&serverID, &shortName, &name); errScan != nil {
			return nil, 0, DBErr(errScan)
		}

		// Added later in dev, so can be legacy data w/o a server_id
		if serverID != nil && shortName != nil && name != nil {
			connHistory.ServerID = *serverID
			connHistory.ServerNameShort = *shortName
			connHistory.ServerName = *name
		}

		connHistory.SteamID = steamid.New(steamID)

		messages = append(messages, connHistory)
	}

	if messages == nil {
		return []model.PersonConnection{}, 0, nil
	}

	count, errCount := getCount(ctx, database, database.
		Builder().
		Select("count(c.person_connection_id)").
		From("person_connections c").
		Where(constraints))

	if errCount != nil {
		return nil, 0, DBErr(errCount)
	}

	return messages, count, nil
}

type ChatHistoryQueryFilter struct {
	QueryFilter
	Personaname   string          `json:"personaname,omitempty"`
	SourceID      model.StringSID `json:"source_id,omitempty"`
	ServerID      int             `json:"server_id,omitempty"`
	DateStart     *time.Time      `json:"date_start,omitempty"`
	DateEnd       *time.Time      `json:"date_end,omitempty"`
	Unrestricted  bool            `json:"-"`
	DontCalcTotal bool            `json:"-"`
	FlaggedOnly   bool            `json:"flagged_only"`
}

const minQueryLen = 2

func QueryChatHistory(ctx context.Context, database Store, filters ChatHistoryQueryFilter) ([]model.QueryChatHistoryResult, int64, error) { //nolint:maintidx
	if filters.Query != "" && len(filters.Query) < minQueryLen {
		return nil, 0, errors.New("Query value too short")
	}

	if filters.Personaname != "" && len(filters.Personaname) < minQueryLen {
		return nil, 0, errors.New("Name value too short")
	}

	builder := database.
		Builder().
		Select("m.person_message_id",
			"m.steam_id ",
			"m.server_id",
			"m.body",
			"m.team ",
			"m.created_on",
			"m.persona_name",
			"m.match_id",
			"s.short_name",
			"CASE WHEN mf.person_message_id::int::boolean THEN mf.person_message_filter_id ELSE 0 END as flagged",
			"p.avatarhash",
			"CASE WHEN f.pattern IS NULL THEN '' ELSE f.pattern END").
		From("person_messages m").
		LeftJoin("server s USING(server_id)").
		LeftJoin("person_messages_filter mf USING(person_message_id)").
		LeftJoin("filtered_word f USING(filter_id)").
		LeftJoin("person p USING(steam_id)")

	builder = filters.applySafeOrder(builder, map[string][]string{
		"m.": {"persona_name", "person_message_id"},
	}, "person_message_id")
	builder = filters.applyLimitOffsetDefault(builder)

	var constraints sq.And

	now := time.Now()

	if !filters.Unrestricted {
		unrTime := now.AddDate(0, 0, -14)
		if filters.DateStart != nil && filters.DateStart.Before(unrTime) {
			return nil, 0, consts.ErrInvalidDuration
		}
	}

	switch {
	case filters.DateStart != nil && filters.DateEnd != nil:
		constraints = append(constraints, sq.Expr("m.created_on BETWEEN ? AND ?", filters.DateStart, filters.DateEnd))
	case filters.DateStart != nil:
		constraints = append(constraints, sq.Expr("? > m.created_on", filters.DateStart))
	case filters.DateEnd != nil:
		constraints = append(constraints, sq.Expr("? < m.created_on", filters.DateEnd))
	}

	if filters.ServerID > 0 {
		constraints = append(constraints, sq.Eq{"m.server_id": filters.ServerID})
	}

	if filters.SourceID != "" {
		sid, errSID := filters.SourceID.SID64(ctx)
		if errSID != nil {
			return nil, 0, errors.Wrap(errSID, "Invalid steam id in query")
		}

		constraints = append(constraints, sq.Eq{"m.steam_id": sid.Int64()})
	}

	if filters.Personaname != "" {
		constraints = append(constraints, sq.Expr(`name_search @@ websearch_to_tsquery('simple', ?)`, filters.Personaname))
	}

	if filters.Query != "" {
		constraints = append(constraints, sq.Expr(`message_search @@ websearch_to_tsquery('simple', ?)`, filters.Query))
	}

	if filters.FlaggedOnly {
		constraints = append(constraints, sq.Eq{"flagged": true})
	}

	var messages []model.QueryChatHistoryResult

	rows, errQuery := database.QueryBuilder(ctx, builder.Where(constraints))
	if errQuery != nil {
		return nil, 0, DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			message model.QueryChatHistoryResult
			steamID int64
			matchID []byte
		)

		if errScan := rows.Scan(&message.PersonMessageID,
			&steamID,
			&message.ServerID,
			&message.Body,
			&message.Team,
			&message.CreatedOn,
			&message.PersonaName,
			&matchID,
			&message.ServerName,
			&message.AutoFilterFlagged,
			&message.AvatarHash,
			&message.Pattern); errScan != nil {
			return nil, 0, DBErr(errScan)
		}

		if matchID != nil {
			// Support for old messages which existed before matches
			message.MatchID = uuid.FromBytesOrNil(matchID)
		}

		message.SteamID = steamid.New(steamID)

		messages = append(messages, message)
	}

	if messages == nil {
		// Return empty list instead of null
		messages = []model.QueryChatHistoryResult{}
	}

	count, errCount := getCount(ctx, database, database.
		Builder().
		Select("count(m.created_on) as count").
		From("person_messages m").
		LeftJoin("server s on m.server_id = s.server_id").
		LeftJoin("person_messages_filter f on m.person_message_id = f.person_message_id").
		LeftJoin("person p on p.steam_id = m.steam_id").
		Where(constraints))

	if errCount != nil {
		return nil, 0, DBErr(errCount)
	}

	return messages, count, nil
}

func GetPersonMessage(ctx context.Context, database Store, messageID int64, msg *model.QueryChatHistoryResult) error {
	row, errRow := database.QueryRowBuilder(ctx, database.
		Builder().
		Select("m.person_message_id", "m.steam_id", "m.server_id", "m.body", "m.team", "m.created_on",
			"m.persona_name", "m.match_id", "s.short_name", "COUNT(f.person_message_id)::int::boolean as flagged").
		From("person_messages m").
		LeftJoin("server s USING(server_id)").
		LeftJoin("person_messages_filter f USING(person_message_id)").
		Where(sq.Eq{"m.person_message_id": messageID}).
		GroupBy("m.person_message_id", "s.short_name"))
	if errRow != nil {
		return DBErr(errRow)
	}

	return DBErr(row.Scan(&msg.PersonMessageID, &msg.SteamID, &msg.ServerID, &msg.Body, &msg.Team, &msg.CreatedOn,
		&msg.PersonaName, &msg.MatchID, &msg.ServerName, &msg.AutoFilterFlagged))
}

func GetPersonMessageContext(ctx context.Context, database Store, serverID int, messageID int64, paddedMessageCount int) ([]model.QueryChatHistoryResult, error) {
	const query = `
		(
			SELECT m.person_message_id, m.steam_id,	m.server_id, m.body, m.team, m.created_on, 
			       m.persona_name,  m.match_id, s.short_name, COUNT(f.person_message_id)::int::boolean as flagged
			FROM person_messages m 
			LEFT JOIN server s on m.server_id = s.server_id
			LEFT JOIN person_messages_filter f on m.person_message_id = f.person_message_id
		 	WHERE m.server_id = $3 AND m.person_message_id >= $1 
		 	GROUP BY m.person_message_id, s.short_name 
		 	ORDER BY m.person_message_id ASC
		 	
		 	LIMIT $2+1
		)
		UNION
		(
			SELECT m.person_message_id, m.steam_id, m.server_id, m.body, m.team, m.created_on, 
			       m.persona_name,  m.match_id, s.short_name, COUNT(f.person_message_id)::int::boolean as flagged
		 	FROM person_messages m 
		 	    LEFT JOIN server s on m.server_id = s.server_id 
		 	LEFT JOIN person_messages_filter f on m.person_message_id = f.person_message_id
		 	WHERE m.server_id = $3 AND  m.person_message_id < $1
		 	GROUP BY m.person_message_id, s.short_name
		 	ORDER BY m.person_message_id DESC
		 	LIMIT $2
		)
		ORDER BY person_message_id DESC`

	if paddedMessageCount > 1000 {
		paddedMessageCount = 1000
	}

	if paddedMessageCount <= 0 {
		paddedMessageCount = 5
	}

	rows, errRows := database.Query(ctx, query, messageID, paddedMessageCount, serverID)
	if errRows != nil {
		return nil, errors.Wrap(errRows, "Failed to fetch message context")
	}
	defer rows.Close()

	var messages []model.QueryChatHistoryResult

	for rows.Next() {
		var msg model.QueryChatHistoryResult

		if errScan := rows.Scan(&msg.PersonMessageID, &msg.SteamID, &msg.ServerID, &msg.Body, &msg.Team, &msg.CreatedOn,
			&msg.PersonaName, &msg.MatchID, &msg.ServerName, &msg.AutoFilterFlagged); errScan != nil {
			return nil, errors.Wrap(errRows, "Failed to scan message result")
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

func GetPersonIPHistory(ctx context.Context, database Store, sid64 steamid.SID64, limit uint64) (model.PersonConnections, error) {
	builder := database.
		Builder().
		Select(
			"DISTINCT on (pn, pc.ip_addr) coalesce(pc.persona_name, pc.steam_id::text) as pn",
			"pc.person_connection_id",
			"pc.steam_id",
			"pc.ip_addr",
			"pc.created_on",
			"pc.server_id").
		From("person_connections pc").
		LeftJoin("net_location loc ON pc.ip_addr <@ loc.ip_range").
		// Join("LEFT JOIN net_proxy proxy ON pc.ip_addr <@ proxy.ip_range").
		OrderBy("1").
		Limit(limit)
	builder = builder.Where(sq.Eq{"pc.steam_id": sid64.Int64()})

	rows, errQuery := database.QueryBuilder(ctx, builder)
	if errQuery != nil {
		return nil, DBErr(errQuery)
	}

	defer rows.Close()

	var connections model.PersonConnections

	for rows.Next() {
		var (
			conn    model.PersonConnection
			steamID int64
		)

		if errScan := rows.Scan(&conn.PersonaName, &conn.PersonConnectionID, &steamID,
			&conn.IPAddr, &conn.CreatedOn, &conn.ServerID); errScan != nil {
			return nil, DBErr(errScan)
		}

		conn.SteamID = steamid.New(steamID)

		connections = append(connections, conn)
	}

	return connections, nil
}

func AddConnectionHistory(ctx context.Context, database Store, conn *model.PersonConnection) error {
	const query = `
		INSERT INTO person_connections (steam_id, ip_addr, persona_name, created_on, server_id) 
		VALUES ($1, $2, $3, $4, $5) 
		RETURNING person_connection_id`

	if errQuery := database.
		QueryRow(ctx, query, conn.SteamID.Int64(), conn.IPAddr, conn.PersonaName, conn.CreatedOn, conn.ServerID).
		Scan(&conn.PersonConnectionID); errQuery != nil {
		return DBErr(errQuery)
	}

	return nil
}

func GetPersonAuthByRefreshToken(ctx context.Context, database Store, token string, auth *model.PersonAuth) error {
	row, errRow := database.QueryRowBuilder(ctx, database.
		Builder().
		Select("person_auth_id", "steam_id", "ip_addr", "refresh_token", "created_on").
		From("person_auth").
		Where(sq.And{sq.Eq{"refresh_token": token}}))
	if errRow != nil {
		return DBErr(errRow)
	}

	var steamID int64

	if errScan := row.Scan(&auth.PersonAuthID, &steamID, &auth.IPAddr, &auth.RefreshToken, &auth.CreatedOn); errScan != nil {
		return DBErr(errScan)
	}

	auth.SteamID = steamid.New(steamID)

	return nil
}

func SavePersonAuth(ctx context.Context, database Store, auth *model.PersonAuth) error {
	query, args, errQuery := database.
		Builder().
		Insert("person_auth").
		Columns("steam_id", "ip_addr", "refresh_token", "created_on").
		Values(auth.SteamID.Int64(), auth.IPAddr.String(), auth.RefreshToken, auth.CreatedOn).
		Suffix("RETURNING \"person_auth_id\"").
		ToSql()

	if errQuery != nil {
		return DBErr(errQuery)
	}

	return DBErr(database.QueryRow(ctx, query, args...).Scan(&auth.PersonAuthID))
}

func DeletePersonAuth(ctx context.Context, database Store, authID int64) error {
	return DBErr(database.ExecDeleteBuilder(ctx, database.
		Builder().
		Delete("person_auth").
		Where(sq.Eq{"person_auth_id": authID})))
}

func PrunePersonAuth(ctx context.Context, database Store) error {
	return DBErr(database.ExecDeleteBuilder(ctx, database.
		Builder().
		Delete("person_auth").
		Where(sq.Gt{"created_on + interval '1 month'": time.Now()})))
}

func SendNotification(ctx context.Context, database Store, targetID steamid.SID64, severity consts.NotificationSeverity, message string, link string) error {
	return DBErr(database.ExecInsertBuilder(ctx, database.
		Builder().
		Insert("person_notification").
		Columns("steam_id", "severity", "message", "link", "created_on").
		Values(targetID.Int64(), severity, message, link, time.Now())))
}

type NotificationQuery struct {
	QueryFilter
	SteamID steamid.SID64 `json:"steam_id"`
}

func GetPersonNotifications(ctx context.Context, database Store, filters NotificationQuery) ([]model.UserNotification, int64, error) {
	builder := database.
		Builder().
		Select("p.person_notification_id", "p.steam_id", "p.read", "p.deleted", "p.severity",
			"p.message", "p.link", "p.count", "p.created_on").
		From("person_notification p").
		OrderBy("p.person_notification_id desc")

	constraints := sq.And{sq.Eq{"p.deleted": false}, sq.Eq{"p.steam_id": filters.SteamID}}

	builder = filters.applySafeOrder(builder, map[string][]string{
		"p.": {"person_notification_id", "steam_id", "read", "deleted", "severity", "message", "link", "count", "created_on"},
	}, "person_notification_id")

	builder = filters.applyLimitOffsetDefault(builder).Where(constraints)

	count, errCount := getCount(ctx, database, database.
		Builder().
		Select("count(p.person_notification_id)").
		From("person_notification p").
		Where(constraints))
	if errCount != nil {
		return nil, 0, DBErr(errCount)
	}

	rows, errRows := database.QueryBuilder(ctx, builder.Where(constraints))
	if errRows != nil {
		return nil, 0, DBErr(errRows)
	}

	defer rows.Close()

	var notifications []model.UserNotification

	for rows.Next() {
		var (
			notif      model.UserNotification
			outSteamID int64
		)

		if errScan := rows.Scan(&notif.PersonNotificationID, &outSteamID, &notif.Read, &notif.Deleted,
			&notif.Severity, &notif.Message, &notif.Link, &notif.Count, &notif.CreatedOn); errScan != nil {
			return nil, 0, errors.Wrapf(errScan, "Failed to scan notification")
		}

		notif.SteamID = steamid.New(outSteamID)

		notifications = append(notifications, notif)
	}

	return notifications, count, nil
}

// func SetNotificationsRead(ctx context.Context, database Store, notificationIds []int64) error {
//	return DBErr(database.ExecUpdateBuilder(ctx, database.
//		Builder().
//		Update("person_notification").
//		Set("deleted", true).
//		Where(sq.Eq{"person_notification_id": notificationIds})))
//}

func GetSteamIdsAbove(ctx context.Context, database Store, privilege consts.Privilege) (steamid.Collection, error) {
	rows, errRows := database.QueryBuilder(ctx, database.
		Builder().
		Select("steam_id").
		From("person").
		Where(sq.GtOrEq{"permission_level": privilege}))
	if errRows != nil {
		return nil, DBErr(errRows)
	}

	defer rows.Close()

	var ids steamid.Collection

	for rows.Next() {
		var sid int64
		if errScan := rows.Scan(&sid); errScan != nil {
			return nil, errors.Wrapf(errScan, "Failed to scan steam id")
		}

		ids = append(ids, steamid.New(sid))
	}

	return ids, nil
}

func GetPersonSettings(ctx context.Context, database Store, steamID steamid.SID64, settings *model.PersonSettings) error {
	row, errRow := database.QueryRowBuilder(ctx, database.
		Builder().
		Select("person_settings_id", "forum_signature", "forum_profile_messages",
			"stats_hidden", "created_on", "updated_on").
		From("person_settings").
		Where(sq.Eq{"steam_id": steamID.Int64()}))
	if errRow != nil {
		return DBErr(errRow)
	}

	settings.SteamID = steamID

	if errScan := row.Scan(&settings.PersonSettingsID, &settings.ForumSignature,
		&settings.ForumProfileMessages, &settings.StatsHidden, &settings.CreatedOn, &settings.UpdatedOn); errScan != nil {
		if errors.Is(DBErr(errScan), ErrNoResult) {
			settings.ForumProfileMessages = true

			return nil
		}

		return DBErr(errScan)
	}

	return nil
}

func SavePersonSettings(ctx context.Context, database Store, settings *model.PersonSettings) error {
	if !settings.SteamID.Valid() {
		return consts.ErrInvalidSID
	}

	settings.UpdatedOn = time.Now()

	if settings.PersonSettingsID == 0 {
		settings.CreatedOn = settings.UpdatedOn

		return DBErr(database.ExecInsertBuilderWithReturnValue(ctx, database.
			Builder().
			Insert("person_settings").
			SetMap(map[string]interface{}{
				"steam_id":               settings.SteamID.Int64(),
				"forum_signature":        settings.ForumSignature,
				"forum_profile_messages": settings.ForumProfileMessages,
				"stats_hidden":           settings.StatsHidden,
				"created_on":             settings.CreatedOn,
				"updated_on":             settings.UpdatedOn,
			}).
			Suffix("RETURNING person_settings_id"),
			&settings.PersonSettingsID))
	}

	return DBErr(database.ExecUpdateBuilder(ctx, database.
		Builder().
		Update("person_settings").
		SetMap(map[string]interface{}{
			"forum_signature":        settings.ForumSignature,
			"forum_profile_messages": settings.ForumProfileMessages,
			"stats_hidden":           settings.StatsHidden,
			"updated_on":             settings.UpdatedOn,
		}).
		Where(sq.Eq{"steam_id": settings.SteamID.Int64()})))
}
