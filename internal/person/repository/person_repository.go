package repository

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

type personRepository struct {
	store.Database
}

func NewPersonRepository(database store.Database) domain.PersonRepository {
	return &personRepository{Database: database}
}

var (
	ErrScanPerson     = errors.New("failed to scan person result")
	ErrMessageContext = errors.New("could not fetch message context")
)

func (s *personRepository) DropPerson(ctx context.Context, steamID steamid.SID64) error {
	return errs.DBErr(s.ExecDeleteBuilder(ctx, s.
		Builder().
		Delete("person").
		Where(sq.Eq{"steam_id": steamID.Int64()})))
}

// SavePerson will insert or update the person record.
func (s *personRepository) SavePerson(ctx context.Context, person *domain.Person) error {
	person.UpdatedOn = time.Now()
	// FIXME
	if person.PermissionLevel == 0 {
		person.PermissionLevel = 10
	}

	if !person.IsNew {
		return s.updatePerson(ctx, person)
	}

	person.CreatedOn = person.UpdatedOn

	return s.insertPerson(ctx, person)
}

func (s *personRepository) updatePerson(ctx context.Context, person *domain.Person) error {
	person.UpdatedOn = time.Now()

	return errs.DBErr(s.
		ExecUpdateBuilder(ctx, s.
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

func (s *personRepository) insertPerson(ctx context.Context, person *domain.Person) error {
	errExec := s.ExecInsertBuilder(ctx, s.
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
		return errs.DBErr(errExec)
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
func (s *personRepository) GetPersonBySteamID(ctx context.Context, sid64 steamid.SID64, person *domain.Person) error {
	if !sid64.Valid() {
		return errs.ErrInvalidSID
	}

	row, errRow := s.QueryRowBuilder(ctx, s.
		Builder().
		Select("s.created_on",
			"s.updated_on",
			"s.communityvisibilitystate",
			"s.profilestate",
			"s.personaname",
			"s.profileurl",
			"s.avatar",
			"s.avatarmedium",
			"s.avatarfull",
			"s.avatarhash",
			"s.personastate",
			"s.realname",
			"s.timecreated",
			"s.loccountrycode",
			"s.locstatecode",
			"s.loccityid",
			"s.permission_level",
			"s.discord_id",
			"s.community_banned",
			"s.vac_bans",
			"s.game_bans",
			"s.economy_ban",
			"s.days_since_last_ban",
			"s.updated_on_steam",
			"s.muted").
		From("person s").
		Where(sq.Eq{"s.steam_id": sid64.Int64()}))

	if errRow != nil {
		return errs.DBErr(errRow)
	}

	person.IsNew = false
	person.PlayerSummary = &steamweb.PlayerSummary{}
	person.SteamID = sid64

	return errs.DBErr(row.Scan(&person.CreatedOn,
		&person.UpdatedOn, &person.CommunityVisibilityState, &person.ProfileState, &person.PersonaName,
		&person.ProfileURL, &person.Avatar, &person.AvatarMedium, &person.AvatarFull, &person.AvatarHash,
		&person.PersonaState, &person.RealName, &person.TimeCreated, &person.LocCountryCode, &person.LocStateCode,
		&person.LocCityID, &person.PermissionLevel, &person.DiscordID, &person.CommunityBanned,
		&person.VACBans, &person.GameBans, &person.EconomyBan, &person.DaysSinceLastBan, &person.UpdatedOnSteam,
		&person.Muted))
}

func (s *personRepository) GetPeopleBySteamID(ctx context.Context, steamIds steamid.Collection) (domain.People, error) {
	var ids []int64 //nolint:prealloc
	for _, sid := range fp.Uniq[steamid.SID64](steamIds) {
		ids = append(ids, sid.Int64())
	}

	var people domain.People

	rows, errQuery := s.QueryBuilder(ctx, s.
		Builder().
		Select(profileColumns...).
		From("person").
		Where(sq.Eq{"steam_id": ids}))
	if errQuery != nil {
		return nil, errs.DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			steamID int64
			person  = domain.NewPerson("")
		)

		if errScan := rows.Scan(&steamID, &person.CreatedOn, &person.UpdatedOn, &person.CommunityVisibilityState,
			&person.ProfileState, &person.PersonaName, &person.ProfileURL, &person.Avatar, &person.AvatarMedium,
			&person.AvatarFull, &person.AvatarHash, &person.PersonaState, &person.RealName, &person.TimeCreated,
			&person.LocCountryCode, &person.LocStateCode, &person.LocCityID, &person.PermissionLevel, &person.DiscordID,
			&person.CommunityBanned, &person.VACBans, &person.GameBans, &person.EconomyBan, &person.DaysSinceLastBan,
			&person.UpdatedOnSteam, &person.Muted); errScan != nil {
			return nil, errors.Join(errScan, ErrScanPerson)
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

func (s *personRepository) GetSteamsAtAddress(ctx context.Context, addr net.IP) (steamid.Collection, error) {
	var ids steamid.Collection

	// TODO
	rows, errRows := s.QueryBuilder(ctx, s.
		Builder().
		Select("DISTINCT steam_id").
		From("person_connections").
		Where(sq.Expr(fmt.Sprintf("ip_addr::inet >>= '::ffff:%s'::CIDR OR ip_addr::inet <<= '::ffff:%s'::CIDR", addr.String(), addr.String()))))
	if errRows != nil {
		return nil, errs.DBErr(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var sid int64
		if errScan := rows.Scan(&sid); errScan != nil {
			return nil, errs.DBErr(errScan)
		}

		ids = append(ids, steamid.New(sid))
	}

	return ids, nil
}

func (s *personRepository) GetPeople(ctx context.Context, filter domain.PlayerQuery) (domain.People, int64, error) {
	builder := s.
		Builder().
		Select("s.steam_id", "s.created_on", "s.updated_on",
			"s.communityvisibilitystate", "s.profilestate", "s.personaname", "s.profileurl", "s.avatar",
			"s.avatarmedium", "s.avatarfull", "s.avatarhash", "s.personastate", "s.realname", "s.timecreated",
			"s.loccountrycode", "s.locstatecode", "s.loccityid", "s.permission_level", "s.discord_id",
			"s.community_banned", "s.vac_bans", "s.game_bans", "s.economy_ban", "s.days_since_last_ban",
			"s.updated_on_steam", "s.muted").
		From("person s")

	conditions := sq.And{}

	if filter.IP != "" {
		addr := net.ParseIP(filter.IP)
		if addr == nil {
			return nil, 0, errs.ErrInvalidIP
		}

		foundIds, errFoundIds := s.GetSteamsAtAddress(ctx, addr)
		if errFoundIds != nil {
			if errors.Is(errFoundIds, errs.ErrNoResult) {
				return domain.People{}, 0, nil
			}

			return nil, 0, errs.DBErr(errFoundIds)
		}

		conditions = append(conditions, sq.Eq{"s.steam_id": foundIds})
	}

	if filter.SteamID != "" {
		steamID, errSteamID := filter.SteamID.SID64(ctx)
		if errSteamID != nil {
			return nil, 0, errors.Join(errSteamID, errs.ErrSourceID)
		}

		conditions = append(conditions, sq.Eq{"s.steam_id": steamID.Int64()})
	}

	if filter.Personaname != "" {
		// TODO add lower-cased functional index to avoid table scan
		conditions = append(conditions, sq.ILike{"s.personaname": normalizeStringLikeQuery(filter.Personaname)})
	}

	builder = filter.ApplyLimitOffsetDefault(builder)
	builder = filter.ApplySafeOrder(builder, map[string][]string{
		"s.": {
			"steam_id", "created_on", "updated_on",
			"communityvisibilitystate", "profilestate", "personaname", "profileurl", "avatar",
			"avatarmedium", "avatarfull", "avatarhash", "personastate", "realname", "timecreated",
			"loccountrycode", "locstatecode", "loccityid", "s.permission_level", "discord_id",
			"community_banned", "vac_bans", "game_bans", "economy_ban", "days_since_last_ban",
			"updated_on_steam", "muted",
		},
	}, "steam_id")

	var people domain.People

	rows, errQuery := s.QueryBuilder(ctx, builder.Where(conditions))
	if errQuery != nil {
		return nil, 0, errs.DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			person  = domain.NewPerson("")
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
			return nil, 0, errors.Join(errScan, ErrScanPerson)
		}

		person.SteamID = steamid.New(steamID)

		people = append(people, person)
	}

	count, errCount := s.GetCount(ctx, s, s.
		Builder().
		Select("COUNT(s.steam_id)").
		From("person s").
		Where(conditions))
	if errCount != nil {
		return nil, 0, errors.Join(errCount, domain.ErrCountQuery)
	}

	return people, count, nil
}

// GetOrCreatePersonBySteamID returns a person by their steam_id, creating a new person if the steam_id
// does not exist.
func (s *personRepository) GetOrCreatePersonBySteamID(ctx context.Context, sid64 steamid.SID64, person *domain.Person) error {
	errGetPerson := s.GetPersonBySteamID(ctx, sid64, person)
	if errGetPerson != nil && errors.Is(errs.DBErr(errGetPerson), errs.ErrNoResult) {
		// FIXME
		newPerson := domain.NewPerson(sid64)
		*person = newPerson

		return s.SavePerson(ctx, person)
	}

	return errGetPerson
}

// GetPersonByDiscordID returns a person by their discord_id.
func (s *personRepository) GetPersonByDiscordID(ctx context.Context, discordID string, person *domain.Person) error {
	var steamID int64

	person.IsNew = false
	person.PlayerSummary = &steamweb.PlayerSummary{}

	row, errRow := s.QueryRowBuilder(ctx, s.
		Builder().
		Select(profileColumns...).
		From("person").
		Where(sq.Eq{"discord_id": discordID}))
	if errRow != nil {
		return errs.DBErr(errRow)
	}

	errQuery := row.Scan(&steamID, &person.CreatedOn,
		&person.UpdatedOn, &person.CommunityVisibilityState, &person.ProfileState, &person.PersonaName,
		&person.ProfileURL, &person.Avatar, &person.AvatarMedium, &person.AvatarFull, &person.AvatarHash,
		&person.PersonaState, &person.RealName, &person.TimeCreated, &person.LocCountryCode, &person.LocStateCode,
		&person.LocCityID, &person.PermissionLevel, &person.DiscordID, &person.CommunityBanned, &person.VACBans,
		&person.GameBans, &person.EconomyBan, &person.DaysSinceLastBan, &person.UpdatedOnSteam, &person.Muted)
	if errQuery != nil {
		return errs.DBErr(errQuery)
	}

	person.SteamID = steamid.New(steamID)

	return nil
}

func (s *personRepository) GetExpiredProfiles(ctx context.Context, limit uint64) ([]domain.Person, error) {
	var people []domain.Person

	rows, errQuery := s.QueryBuilder(ctx, s.
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
		return nil, errs.DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			person  = domain.NewPerson("")
			steamID int64
		)

		if errScan := rows.Scan(&steamID, &person.CreatedOn, &person.UpdatedOn, &person.CommunityVisibilityState,
			&person.ProfileState, &person.PersonaName, &person.ProfileURL, &person.Avatar, &person.AvatarMedium,
			&person.AvatarFull, &person.AvatarHash, &person.PersonaState, &person.RealName, &person.TimeCreated,
			&person.LocCountryCode, &person.LocStateCode, &person.LocCityID, &person.PermissionLevel,
			&person.DiscordID, &person.CommunityBanned, &person.VACBans, &person.GameBans,
			&person.EconomyBan, &person.DaysSinceLastBan, &person.UpdatedOnSteam, &person.Muted); errScan != nil {
			return nil, errs.DBErr(errScan)
		}

		person.SteamID = steamid.New(steamID)

		people = append(people, person)
	}

	return people, nil
}

func (s *personRepository) AddChatHistory(ctx context.Context, message *domain.PersonMessage) error {
	const query = `INSERT INTO person_messages 
    		(steam_id, server_id, body, team, created_on, persona_name, match_id) 
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING person_message_id`

	if errScan := s.
		QueryRow(ctx, query, message.SteamID.Int64(), message.ServerID, message.Body, message.Team,
			message.CreatedOn, message.PersonaName, message.MatchID).
		Scan(&message.PersonMessageID); errScan != nil {
		return errs.DBErr(errScan)
	}

	return nil
}

func (s *personRepository) GetPersonMessageByID(ctx context.Context, personMessageID int64, msg *domain.PersonMessage) error {
	row, errRow := s.QueryRowBuilder(ctx, s.
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
		return errs.DBErr(errRow)
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
		return errs.DBErr(errScan)
	}

	msg.SteamID = steamid.New(steamID)

	return nil
}

func (s *personRepository) QueryConnectionHistory(ctx context.Context, opts domain.ConnectionHistoryQueryFilter) ([]domain.PersonConnection, int64, error) {
	builder := s.
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
			return nil, 0, errors.Join(steamid.ErrInvalidSID, errs.ErrSourceID)
		}

		constraints = append(constraints, sq.Eq{"c.steam_id": sid.Int64()})
	}

	builder = opts.ApplySafeOrder(opts.ApplyLimitOffsetDefault(builder), map[string][]string{
		"c.": {"person_connection_id", "steam_id", "ip_addr", "persona_name", "created_on"},
		"s.": {"short_name", "name"},
	}, "person_connection_id")

	var messages []domain.PersonConnection

	rows, errQuery := s.QueryBuilder(ctx, builder.Where(constraints))
	if errQuery != nil {
		return nil, 0, errs.DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			connHistory domain.PersonConnection
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
			return nil, 0, errs.DBErr(errScan)
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
		return []domain.PersonConnection{}, 0, nil
	}

	count, errCount := s.GetCount(ctx, s, s.
		Builder().
		Select("count(c.person_connection_id)").
		From("person_connections c").
		Where(constraints))

	if errCount != nil {
		return nil, 0, errs.DBErr(errCount)
	}

	return messages, count, nil
}

const minQueryLen = 2

func (s *personRepository) QueryChatHistory(ctx context.Context, filters domain.ChatHistoryQueryFilter) ([]domain.QueryChatHistoryResult, int64, error) { //nolint:maintidx
	if filters.Query != "" && len(filters.Query) < minQueryLen {
		return nil, 0, fmt.Errorf("%w: query", domain.ErrTooShort)
	}

	if filters.Personaname != "" && len(filters.Personaname) < minQueryLen {
		return nil, 0, fmt.Errorf("%w: name", domain.ErrTooShort)
	}

	builder := s.
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
			"s.avatarhash",
			"CASE WHEN f.pattern IS NULL THEN '' ELSE f.pattern END").
		From("person_messages m").
		LeftJoin("server s USING(server_id)").
		LeftJoin("person_messages_filter mf USING(person_message_id)").
		LeftJoin("filtered_word f USING(filter_id)").
		LeftJoin("person s USING(steam_id)")

	builder = filters.ApplySafeOrder(builder, map[string][]string{
		"m.": {"persona_name", "person_message_id"},
	}, "person_message_id")
	builder = filters.ApplyLimitOffsetDefault(builder)

	var constraints sq.And

	now := time.Now()

	if !filters.Unrestricted {
		unrTime := now.AddDate(0, 0, -14)
		if filters.DateStart != nil && filters.DateStart.Before(unrTime) {
			return nil, 0, util.ErrInvalidDuration
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
			return nil, 0, errors.Join(errSID, errs.ErrSourceID)
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

	var messages []domain.QueryChatHistoryResult

	rows, errQuery := s.QueryBuilder(ctx, builder.Where(constraints))
	if errQuery != nil {
		return nil, 0, errs.DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			message domain.QueryChatHistoryResult
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
			return nil, 0, errs.DBErr(errScan)
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
		messages = []domain.QueryChatHistoryResult{}
	}

	count, errCount := s.GetCount(ctx, s, s.
		Builder().
		Select("count(m.created_on) as count").
		From("person_messages m").
		LeftJoin("server s on m.server_id = s.server_id").
		LeftJoin("person_messages_filter f on m.person_message_id = f.person_message_id").
		LeftJoin("person s on s.steam_id = m.steam_id").
		Where(constraints))

	if errCount != nil {
		return nil, 0, errs.DBErr(errCount)
	}

	return messages, count, nil
}

func (s *personRepository) GetPersonMessage(ctx context.Context, messageID int64, msg *domain.QueryChatHistoryResult) error {
	row, errRow := s.QueryRowBuilder(ctx, s.
		Builder().
		Select("m.person_message_id", "m.steam_id", "m.server_id", "m.body", "m.team", "m.created_on",
			"m.persona_name", "m.match_id", "s.short_name", "COUNT(f.person_message_id)::int::boolean as flagged").
		From("person_messages m").
		LeftJoin("server s USING(server_id)").
		LeftJoin("person_messages_filter f USING(person_message_id)").
		Where(sq.Eq{"m.person_message_id": messageID}).
		GroupBy("m.person_message_id", "s.short_name"))
	if errRow != nil {
		return errs.DBErr(errRow)
	}

	return errs.DBErr(row.Scan(&msg.PersonMessageID, &msg.SteamID, &msg.ServerID, &msg.Body, &msg.Team, &msg.CreatedOn,
		&msg.PersonaName, &msg.MatchID, &msg.ServerName, &msg.AutoFilterFlagged))
}

func (s *personRepository) GetPersonMessageContext(ctx context.Context, serverID int, messageID int64, paddedMessageCount int) ([]domain.QueryChatHistoryResult, error) {
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

	rows, errRows := s.Query(ctx, query, messageID, paddedMessageCount, serverID)
	if errRows != nil {
		return nil, errors.Join(errRows, ErrMessageContext)
	}
	defer rows.Close()

	var messages []domain.QueryChatHistoryResult

	for rows.Next() {
		var msg domain.QueryChatHistoryResult

		if errScan := rows.Scan(&msg.PersonMessageID, &msg.SteamID, &msg.ServerID, &msg.Body, &msg.Team, &msg.CreatedOn,
			&msg.PersonaName, &msg.MatchID, &msg.ServerName, &msg.AutoFilterFlagged); errScan != nil {
			return nil, errors.Join(errRows, ErrScanResult)
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

func (s *personRepository) GetPersonIPHistory(ctx context.Context, sid64 steamid.SID64, limit uint64) (domain.PersonConnections, error) {
	builder := s.
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

	rows, errQuery := s.QueryBuilder(ctx, builder)
	if errQuery != nil {
		return nil, errs.DBErr(errQuery)
	}

	defer rows.Close()

	var connections domain.PersonConnections

	for rows.Next() {
		var (
			conn    domain.PersonConnection
			steamID int64
		)

		if errScan := rows.Scan(&conn.PersonaName, &conn.PersonConnectionID, &steamID,
			&conn.IPAddr, &conn.CreatedOn, &conn.ServerID); errScan != nil {
			return nil, errs.DBErr(errScan)
		}

		conn.SteamID = steamid.New(steamID)

		connections = append(connections, conn)
	}

	return connections, nil
}

func (s *personRepository) AddConnectionHistory(ctx context.Context, conn *domain.PersonConnection) error {
	const query = `
		INSERT INTO person_connections (steam_id, ip_addr, persona_name, created_on, server_id) 
		VALUES ($1, $2, $3, $4, $5) 
		RETURNING person_connection_id`

	if errQuery := s.
		QueryRow(ctx, query, conn.SteamID.Int64(), conn.IPAddr, conn.PersonaName, conn.CreatedOn, conn.ServerID).
		Scan(&conn.PersonConnectionID); errQuery != nil {
		return errs.DBErr(errQuery)
	}

	return nil
}

func (s *personRepository) GetPersonAuthByRefreshToken(ctx context.Context, token string, auth *domain.PersonAuth) error {
	row, errRow := s.QueryRowBuilder(ctx, s.
		Builder().
		Select("person_auth_id", "steam_id", "ip_addr", "refresh_token", "created_on").
		From("person_auth").
		Where(sq.And{sq.Eq{"refresh_token": token}}))
	if errRow != nil {
		return errs.DBErr(errRow)
	}

	var steamID int64

	if errScan := row.Scan(&auth.PersonAuthID, &steamID, &auth.IPAddr, &auth.RefreshToken, &auth.CreatedOn); errScan != nil {
		return errs.DBErr(errScan)
	}

	auth.SteamID = steamid.New(steamID)

	return nil
}

func (s *personRepository) SavePersonAuth(ctx context.Context, auth *domain.PersonAuth) error {
	query, args, errQuery := s.
		Builder().
		Insert("person_auth").
		Columns("steam_id", "ip_addr", "refresh_token", "created_on").
		Values(auth.SteamID.Int64(), auth.IPAddr.String(), auth.RefreshToken, auth.CreatedOn).
		Suffix("RETURNING \"person_auth_id\"").
		ToSql()

	if errQuery != nil {
		return errs.DBErr(errQuery)
	}

	return errs.DBErr(s.QueryRow(ctx, query, args...).Scan(&auth.PersonAuthID))
}

func (s *personRepository) DeletePersonAuth(ctx context.Context, authID int64) error {
	return errs.DBErr(s.ExecDeleteBuilder(ctx, s.
		Builder().
		Delete("person_auth").
		Where(sq.Eq{"person_auth_id": authID})))
}

func (s *personRepository) PrunePersonAuth(ctx context.Context) error {
	return errs.DBErr(s.ExecDeleteBuilder(ctx, s.
		Builder().
		Delete("person_auth").
		Where(sq.Gt{"created_on + interval '1 month'": time.Now()})))
}

func (s *personRepository) SendNotification(ctx context.Context, targetID steamid.SID64, severity domain.NotificationSeverity, message string, link string) error {
	return errs.DBErr(s.ExecInsertBuilder(ctx, s.
		Builder().
		Insert("person_notification").
		Columns("steam_id", "severity", "message", "link", "created_on").
		Values(targetID.Int64(), severity, message, link, time.Now())))
}

func (s *personRepository) GetPersonNotifications(ctx context.Context, filters domain.NotificationQuery) ([]domain.UserNotification, int64, error) {
	builder := s.
		Builder().
		Select("s.person_notification_id", "s.steam_id", "s.read", "s.deleted", "s.severity",
			"s.message", "s.link", "s.count", "s.created_on").
		From("person_notification s").
		OrderBy("s.person_notification_id desc")

	constraints := sq.And{sq.Eq{"s.deleted": false}, sq.Eq{"s.steam_id": filters.SteamID}}

	builder = filters.ApplySafeOrder(builder, map[string][]string{
		"s.": {"person_notification_id", "steam_id", "read", "deleted", "severity", "message", "link", "count", "created_on"},
	}, "person_notification_id")

	builder = filters.ApplyLimitOffsetDefault(builder).Where(constraints)

	count, errCount := s.GetCount(ctx, s, s.
		Builder().
		Select("count(s.person_notification_id)").
		From("person_notification s").
		Where(constraints))
	if errCount != nil {
		return nil, 0, errs.DBErr(errCount)
	}

	rows, errRows := s.QueryBuilder(ctx, builder.Where(constraints))
	if errRows != nil {
		return nil, 0, errs.DBErr(errRows)
	}

	defer rows.Close()

	var notifications []domain.UserNotification

	for rows.Next() {
		var (
			notif      domain.UserNotification
			outSteamID int64
		)

		if errScan := rows.Scan(&notif.PersonNotificationID, &outSteamID, &notif.Read, &notif.Deleted,
			&notif.Severity, &notif.Message, &notif.Link, &notif.Count, &notif.CreatedOn); errScan != nil {
			return nil, 0, errors.Join(errScan, ErrScanResult)
		}

		notif.SteamID = steamid.New(outSteamID)

		notifications = append(notifications, notif)
	}

	return notifications, count, nil
}

// func SetNotificationsRead(ctx context.Context,  notificationIds []int64) error {
//	return errs.DBErr(database.ExecUpdateBuilder(ctx, database.
//		Builder().
//		Update("person_notification").
//		Set("deleted", true).
//		Where(sq.Eq{"person_notification_id": notificationIds})))
//}

func (s *personRepository) GetSteamIdsAbove(ctx context.Context, privilege domain.Privilege) (steamid.Collection, error) {
	rows, errRows := s.QueryBuilder(ctx, s.
		Builder().
		Select("steam_id").
		From("person").
		Where(sq.GtOrEq{"permission_level": privilege}))
	if errRows != nil {
		return nil, errs.DBErr(errRows)
	}

	defer rows.Close()

	var ids steamid.Collection

	for rows.Next() {
		var sid int64
		if errScan := rows.Scan(&sid); errScan != nil {
			return nil, errors.Join(errScan, ErrScanResult)
		}

		ids = append(ids, steamid.New(sid))
	}

	return ids, nil
}

func (s *personRepository) GetPersonSettings(ctx context.Context, steamID steamid.SID64, settings *domain.PersonSettings) error {
	row, errRow := s.QueryRowBuilder(ctx, s.
		Builder().
		Select("person_settings_id", "forum_signature", "forum_profile_messages",
			"stats_hidden", "created_on", "updated_on").
		From("person_settings").
		Where(sq.Eq{"steam_id": steamID.Int64()}))
	if errRow != nil {
		return errs.DBErr(errRow)
	}

	settings.SteamID = steamID

	if errScan := row.Scan(&settings.PersonSettingsID, &settings.ForumSignature,
		&settings.ForumProfileMessages, &settings.StatsHidden, &settings.CreatedOn, &settings.UpdatedOn); errScan != nil {
		if errors.Is(errs.DBErr(errScan), errs.ErrNoResult) {
			settings.ForumProfileMessages = true

			return nil
		}

		return errs.DBErr(errScan)
	}

	return nil
}

func (s *personRepository) SavePersonSettings(ctx context.Context, settings *domain.PersonSettings) error {
	if !settings.SteamID.Valid() {
		return errs.ErrInvalidSID
	}

	settings.UpdatedOn = time.Now()

	if settings.PersonSettingsID == 0 {
		settings.CreatedOn = settings.UpdatedOn

		return errs.DBErr(s.ExecInsertBuilderWithReturnValue(ctx, s.
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

	return errs.DBErr(s.ExecUpdateBuilder(ctx, s.
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
