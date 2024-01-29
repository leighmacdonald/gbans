package person

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

type personRepository struct {
	db database.Database
}

func NewPersonRepository(database database.Database) domain.PersonRepository {
	return &personRepository{db: database}
}

var (
	ErrScanPerson     = errors.New("failed to scan person result")
	ErrMessageContext = errors.New("could not fetch message context")
)

func (r *personRepository) DropPerson(ctx context.Context, steamID steamid.SID64) error {
	return r.db.DBErr(r.db.ExecDeleteBuilder(ctx, r.db.
		Builder().
		Delete("person").
		Where(sq.Eq{"steam_id": steamID.Int64()})))
}

// SavePerson will insert or update the person record.
func (r *personRepository) SavePerson(ctx context.Context, person *domain.Person) error {
	person.UpdatedOn = time.Now()
	// FIXME
	if person.PermissionLevel == 0 {
		person.PermissionLevel = 10
	}

	if !person.IsNew {
		return r.updatePerson(ctx, person)
	}

	person.CreatedOn = person.UpdatedOn

	return r.insertPerson(ctx, person)
}

func (r *personRepository) updatePerson(ctx context.Context, person *domain.Person) error {
	person.UpdatedOn = time.Now()

	return r.db.DBErr(r.db.
		ExecUpdateBuilder(ctx, r.db.
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

func (r *personRepository) insertPerson(ctx context.Context, person *domain.Person) error {
	errExec := r.db.ExecInsertBuilder(ctx, r.db.
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
		return r.db.DBErr(errExec)
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
func (r *personRepository) GetPersonBySteamID(ctx context.Context, sid64 steamid.SID64, person *domain.Person) error {
	if !sid64.Valid() {
		return domain.ErrInvalidSID
	}

	row, errRow := r.db.QueryRowBuilder(ctx, r.db.
		Builder().
		Select("r.created_on",
			"r.updated_on",
			"r.communityvisibilitystate",
			"r.profilestate",
			"r.personaname",
			"r.profileurl",
			"r.avatar",
			"r.avatarmedium",
			"r.avatarfull",
			"r.avatarhash",
			"r.personastate",
			"r.realname",
			"r.timecreated",
			"r.loccountrycode",
			"r.locstatecode",
			"r.loccityid",
			"r.permission_level",
			"r.discord_id",
			"r.community_banned",
			"r.vac_bans",
			"r.game_bans",
			"r.economy_ban",
			"r.days_since_last_ban",
			"r.updated_on_steam",
			"r.muted").
		From("person r").
		Where(sq.Eq{"r.steam_id": sid64.Int64()}))

	if errRow != nil {
		return r.db.DBErr(errRow)
	}

	person.IsNew = false
	person.PlayerSummary = &steamweb.PlayerSummary{}
	person.SteamID = sid64

	return r.db.DBErr(row.Scan(&person.CreatedOn,
		&person.UpdatedOn, &person.CommunityVisibilityState, &person.ProfileState, &person.PersonaName,
		&person.ProfileURL, &person.Avatar, &person.AvatarMedium, &person.AvatarFull, &person.AvatarHash,
		&person.PersonaState, &person.RealName, &person.TimeCreated, &person.LocCountryCode, &person.LocStateCode,
		&person.LocCityID, &person.PermissionLevel, &person.DiscordID, &person.CommunityBanned,
		&person.VACBans, &person.GameBans, &person.EconomyBan, &person.DaysSinceLastBan, &person.UpdatedOnSteam,
		&person.Muted))
}

func (r *personRepository) GetPeopleBySteamID(ctx context.Context, steamIds steamid.Collection) (domain.People, error) {
	var ids []int64 //nolint:prealloc
	for _, sid := range fp.Uniq[steamid.SID64](steamIds) {
		ids = append(ids, sid.Int64())
	}

	var people domain.People

	rows, errQuery := r.db.QueryBuilder(ctx, r.db.
		Builder().
		Select(profileColumns...).
		From("person").
		Where(sq.Eq{"steam_id": ids}))
	if errQuery != nil {
		return nil, r.db.DBErr(errQuery)
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

func (r *personRepository) GetSteamsAtAddress(ctx context.Context, addr net.IP) (steamid.Collection, error) {
	var ids steamid.Collection

	// TODO
	rows, errRows := r.db.QueryBuilder(ctx, r.db.
		Builder().
		Select("DISTINCT steam_id").
		From("person_connections").
		Where(sq.Expr(fmt.Sprintf("ip_addr::inet >>= '::ffff:%r'::CIDR OR ip_addr::inet <<= '::ffff:%r'::CIDR", addr.String(), addr.String()))))
	if errRows != nil {
		return nil, r.db.DBErr(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var sid int64
		if errScan := rows.Scan(&sid); errScan != nil {
			return nil, r.db.DBErr(errScan)
		}

		ids = append(ids, steamid.New(sid))
	}

	return ids, nil
}

func (r *personRepository) GetPeople(ctx context.Context, filter domain.PlayerQuery) (domain.People, int64, error) {
	builder := r.db.
		Builder().
		Select("r.steam_id", "r.created_on", "r.updated_on",
			"r.communityvisibilitystate", "r.profilestate", "r.personaname", "r.profileurl", "r.avatar",
			"r.avatarmedium", "r.avatarfull", "r.avatarhash", "r.personastate", "r.realname", "r.timecreated",
			"r.loccountrycode", "r.locstatecode", "r.loccityid", "r.permission_level", "r.discord_id",
			"r.community_banned", "r.vac_bans", "r.game_bans", "r.economy_ban", "r.days_since_last_ban",
			"r.updated_on_steam", "r.muted").
		From("person r")

	conditions := sq.And{}

	if filter.IP != "" {
		addr := net.ParseIP(filter.IP)
		if addr == nil {
			return nil, 0, domain.ErrInvalidIP
		}

		foundIds, errFoundIds := r.GetSteamsAtAddress(ctx, addr)
		if errFoundIds != nil {
			if errors.Is(errFoundIds, domain.ErrNoResult) {
				return domain.People{}, 0, nil
			}

			return nil, 0, r.db.DBErr(errFoundIds)
		}

		conditions = append(conditions, sq.Eq{"r.steam_id": foundIds})
	}

	if filter.SteamID != "" {
		steamID, errSteamID := filter.SteamID.SID64(ctx)
		if errSteamID != nil {
			return nil, 0, errors.Join(errSteamID, domain.ErrSourceID)
		}

		conditions = append(conditions, sq.Eq{"r.steam_id": steamID.Int64()})
	}

	if filter.Personaname != "" {
		// TODO add lower-cased functional index to avoid table scan
		conditions = append(conditions, sq.ILike{"r.personaname": normalizeStringLikeQuery(filter.Personaname)})
	}

	builder = filter.ApplyLimitOffsetDefault(builder)
	builder = filter.ApplySafeOrder(builder, map[string][]string{
		"r.": {
			"steam_id", "created_on", "updated_on",
			"communityvisibilitystate", "profilestate", "personaname", "profileurl", "avatar",
			"avatarmedium", "avatarfull", "avatarhash", "personastate", "realname", "timecreated",
			"loccountrycode", "locstatecode", "loccityid", "r.permission_level", "discord_id",
			"community_banned", "vac_bans", "game_bans", "economy_ban", "days_since_last_ban",
			"updated_on_steam", "muted",
		},
	}, "steam_id")

	var people domain.People

	rows, errQuery := r.db.QueryBuilder(ctx, builder.Where(conditions))
	if errQuery != nil {
		return nil, 0, r.db.DBErr(errQuery)
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

	count, errCount := r.db.GetCount(ctx, r.db.
		Builder().
		Select("COUNT(r.steam_id)").
		From("person r").
		Where(conditions))
	if errCount != nil {
		return nil, 0, errors.Join(errCount, domain.ErrCountQuery)
	}

	return people, count, nil
}

// GetPersonByDiscordID returns a person by their discord_id.
func (r *personRepository) GetPersonByDiscordID(ctx context.Context, discordID string, person *domain.Person) error {
	var steamID int64

	person.IsNew = false
	person.PlayerSummary = &steamweb.PlayerSummary{}

	row, errRow := r.db.QueryRowBuilder(ctx, r.db.
		Builder().
		Select(profileColumns...).
		From("person").
		Where(sq.Eq{"discord_id": discordID}))
	if errRow != nil {
		return r.db.DBErr(errRow)
	}

	errQuery := row.Scan(&steamID, &person.CreatedOn,
		&person.UpdatedOn, &person.CommunityVisibilityState, &person.ProfileState, &person.PersonaName,
		&person.ProfileURL, &person.Avatar, &person.AvatarMedium, &person.AvatarFull, &person.AvatarHash,
		&person.PersonaState, &person.RealName, &person.TimeCreated, &person.LocCountryCode, &person.LocStateCode,
		&person.LocCityID, &person.PermissionLevel, &person.DiscordID, &person.CommunityBanned, &person.VACBans,
		&person.GameBans, &person.EconomyBan, &person.DaysSinceLastBan, &person.UpdatedOnSteam, &person.Muted)
	if errQuery != nil {
		return r.db.DBErr(errQuery)
	}

	person.SteamID = steamid.New(steamID)

	return nil
}

func (r *personRepository) GetExpiredProfiles(ctx context.Context, limit uint64) ([]domain.Person, error) {
	var people []domain.Person

	rows, errQuery := r.db.QueryBuilder(ctx, r.db.
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
		return nil, r.db.DBErr(errQuery)
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
			return nil, r.db.DBErr(errScan)
		}

		person.SteamID = steamid.New(steamID)

		people = append(people, person)
	}

	return people, nil
}

func (r *personRepository) GetPersonMessageByID(ctx context.Context, personMessageID int64, msg *domain.PersonMessage) error {
	row, errRow := r.db.QueryRowBuilder(ctx, r.db.
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
			"r.short_name").
		From("person_messages m").
		LeftJoin("server r on m.server_id = r.server_id").
		Where(sq.Eq{"m.person_message_id": personMessageID}))

	if errRow != nil {
		return r.db.DBErr(errRow)
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
		return r.db.DBErr(errScan)
	}

	msg.SteamID = steamid.New(steamID)

	return nil
}

// func SetNotificationsRead(ctx context.Context,  notificationIds []int64) error {
//	return errs.DBErr(database.ExecUpdateBuilder(ctx, database.
//		Builder().
//		Update("person_notification").
//		Set("deleted", true).
//		Where(sq.Eq{"person_notification_id": notificationIds})))
//}

func (r *personRepository) GetSteamIdsAbove(ctx context.Context, privilege domain.Privilege) (steamid.Collection, error) {
	rows, errRows := r.db.QueryBuilder(ctx, r.db.
		Builder().
		Select("steam_id").
		From("person").
		Where(sq.GtOrEq{"permission_level": privilege}))
	if errRows != nil {
		return nil, r.db.DBErr(errRows)
	}

	defer rows.Close()

	var ids steamid.Collection

	for rows.Next() {
		var sid int64
		if errScan := rows.Scan(&sid); errScan != nil {
			return nil, errors.Join(errScan, domain.ErrScanResult)
		}

		ids = append(ids, steamid.New(sid))
	}

	return ids, nil
}

func (r *personRepository) GetPersonSettings(ctx context.Context, steamID steamid.SID64, settings *domain.PersonSettings) error {
	row, errRow := r.db.QueryRowBuilder(ctx, r.db.
		Builder().
		Select("person_settings_id", "forum_signature", "forum_profile_messages",
			"stats_hidden", "created_on", "updated_on").
		From("person_settings").
		Where(sq.Eq{"steam_id": steamID.Int64()}))
	if errRow != nil {
		return r.db.DBErr(errRow)
	}

	settings.SteamID = steamID

	if errScan := row.Scan(&settings.PersonSettingsID, &settings.ForumSignature,
		&settings.ForumProfileMessages, &settings.StatsHidden, &settings.CreatedOn, &settings.UpdatedOn); errScan != nil {
		if errors.Is(r.db.DBErr(errScan), domain.ErrNoResult) {
			settings.ForumProfileMessages = true

			return nil
		}

		return r.db.DBErr(errScan)
	}

	return nil
}

func (r *personRepository) SavePersonSettings(ctx context.Context, settings *domain.PersonSettings) error {
	if !settings.SteamID.Valid() {
		return domain.ErrInvalidSID
	}

	settings.UpdatedOn = time.Now()

	if settings.PersonSettingsID == 0 {
		settings.CreatedOn = settings.UpdatedOn

		return r.db.DBErr(r.db.ExecInsertBuilderWithReturnValue(ctx, r.db.
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

	return r.db.DBErr(r.db.ExecUpdateBuilder(ctx, r.db.
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
