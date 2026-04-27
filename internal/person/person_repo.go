package person

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type Repository struct {
	database.Database

	centerProjectiles bool
}

func NewRepository(database database.Database, centerProjectiles bool) Repository {
	return Repository{Database: database, centerProjectiles: centerProjectiles}
}

func (r *Repository) DropPerson(ctx context.Context, steamID steamid.SteamID) error {
	return database.Err(r.ExecDeleteBuilder(ctx, r.Builder().
		Delete("person").
		Where(sq.Eq{"steam_id": steamID.Int64()})))
}

func (r *Repository) Save(ctx context.Context, person *Person) error {
	const query = `
		INSERT INTO person (
			steam_id, communityvisibilitystate, profilestate,
			personaname, avatarhash, personastate,
			realname, timecreated, loccountrycode, locstatecode, loccityid, permission_level,
			discord_id, community_banned, vac_bans, game_bans, economy_ban, days_since_last_ban,
			updated_on_steam, muted,  created_on, updated_on)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
		ON CONFLICT (steam_id) DO UPDATE SET
			communityvisibilitystate = $2, profilestate = $3,
			personaname = $4, avatarhash = $5, personastate = $6,
			realname = $7, timecreated = $8, loccountrycode = $9, locstatecode = $10, loccityid = $11,  permission_level = $12,
			discord_id = $13, community_banned = $14,  vac_bans = $15, game_bans = $16, economy_ban = $17, days_since_last_ban = $18,
			updated_on_steam = $19, muted = $20, updated_on=$22`

	return database.Err(r.Exec(ctx, query, person.SteamID.Int64(), person.VisibilityState,
		person.ProfileState, person.PersonaName,
		person.AvatarHash, person.PersonaState, person.RealName,
		person.TimeCreated, person.LocCountryCode, person.LocStateCode,
		person.LocCityID, person.PermissionLevel, person.DiscordID, person.CommunityBanned,
		person.VACBans, person.GameBans, person.EconomyBan, person.DaysSinceLastBan, person.UpdatedOnSteam,
		person.Muted, person.CreatedOn, person.UpdatedOn))
}

func normalizeStringLikeQuery(input string) string {
	space := regexp.MustCompile(`\s+`)

	return fmt.Sprintf("%%%s%%", strings.ToLower(strings.Trim(space.ReplaceAllString(input, "%"), "%")))
}

func (r *Repository) Query(ctx context.Context, query Query) (People, int64, error) {
	builder := r.Builder().
		Select("p.steam_id", "p.created_on", "p.updated_on",
			"p.communityvisibilitystate", "p.profilestate", "p.personaname", "p.avatarhash", "p.personastate", "p.realname", "p.timecreated",
			"p.loccountrycode", "p.locstatecode", "p.loccityid", "p.permission_level", "p.discord_id",
			"p.community_banned", "p.vac_bans", "p.game_bans", "p.economy_ban", "p.days_since_last_ban",
			"p.updated_on_steam", "p.muted", "coalesce(pt.patreon_id, '')").
		From("person p").
		LeftJoin("auth_patreon pt USING (steam_id)")

	constraints := sq.And{}

	minDate := time.Date(2007, 0, 0, 0, 0, 0, 0, time.UTC)

	if query.SteamUpdateOlderThan.After(minDate) {
		// builder = builder.OrderBy("p.updated_on_steam ASC")
		constraints = append(constraints, sq.Lt{"p.updated_on_steam": query.SteamUpdateOlderThan})
	}
	if len(query.WithPermissions) > 0 {
		constraints = append(constraints, sq.Eq{"p.permission_level": query.WithPermissions})
	}

	if query.DiscordID != "" {
		constraints = append(constraints, sq.Eq{"p.discord_id": query.DiscordID})
	}

	if len(query.SteamIDs) > 0 {
		constraints = append(constraints, sq.Eq{"p.steam_id": query.SteamIDs})
	}

	if query.PersonaName != "" {
		// TODO add lower-cased functional index to avoid table scan
		constraints = append(constraints, sq.ILike{"p.personaname": normalizeStringLikeQuery(query.PersonaName)})
	}

	if query.GameBans > 0 {
		constraints = append(constraints, sq.GtOrEq{"p.game_bans": query.GameBans})
	}

	if query.VacBans > 0 {
		constraints = append(constraints, sq.GtOrEq{"p.vac_bans": query.VacBans})
	}

	if query.CommunityBanned != nil && *query.CommunityBanned {
		constraints = append(constraints, sq.Eq{"p.community_banned": true})
	}

	switch {
	case query.TimeCreatedAfter != nil && query.TimeCreatedAfter.After(minDate) && query.TimeCreatedBefore != nil && query.TimeCreatedBefore.After(minDate):
		constraints = append(constraints, sq.Expr("p.timecreated BETWEEN $1 AND $2", *query.TimeCreatedAfter, *query.TimeCreatedBefore))
	case query.TimeCreatedAfter != nil && !query.TimeCreatedAfter.After(minDate):
		constraints = append(constraints, sq.GtOrEq{"p.timecreated": *query.TimeCreatedAfter})
	case query.TimeCreatedBefore != nil && !query.TimeCreatedBefore.After(minDate):
		constraints = append(constraints, sq.LtOrEq{"p.timecreated": *query.TimeCreatedBefore})
	}
	builder = query.ApplyLimitOffsetDefault(builder)
	builder = query.ApplySafeOrder(builder, map[string][]string{
		"p.": {
			"steam_id", "created_on", "updated_on",
			"communityvisibilitystate", "profilestate", "personaname", "avatarhash", "personastate",
			"realname", "timecreated", "loccountrycode", "locstatecode", "loccityid", "permission_level",
			"discord_id", "community_banned", "vac_bans", "game_bans", "economy_ban", "days_since_last_ban",
			"updated_on_steam", "muted",
		},
		"pt.": {"patreon_id"},
	}, "steam_id")

	var people People

	rows, errQuery := r.QueryBuilder(ctx, builder.Where(constraints))
	if errQuery != nil {
		return nil, 0, database.Err(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		person := New(steamid.SteamID{})
		if errScan := rows.
			Scan(&person.SteamID, &person.CreatedOn, &person.UpdatedOn, &person.VisibilityState,
				&person.ProfileState, &person.PersonaName, &person.AvatarHash, &person.PersonaState,
				&person.RealName, &person.TimeCreated, &person.LocCountryCode, &person.LocStateCode,
				&person.LocCityID, &person.PermissionLevel, &person.DiscordID, &person.CommunityBanned,
				&person.VACBans, &person.GameBans, &person.EconomyBan, &person.DaysSinceLastBan,
				&person.UpdatedOnSteam, &person.Muted, &person.PatreonID); errScan != nil {
			return nil, 0, errors.Join(errScan, database.ErrScanResult)
		}

		people = append(people, person)
	}

	count, errQuery := r.GetCount(ctx, r.Builder().
		Select("count(p.steam_id)").
		From("person p").
		LeftJoin("auth_patreon pt USING (steam_id)").Where(constraints))
	if errQuery != nil {
		return nil, 0, database.Err(errQuery)
	}

	return people, count, nil
}

func (r *Repository) Settings(ctx context.Context, steamID steamid.SteamID) (Settings, error) {
	var settings Settings

	row, errRow := r.QueryRowBuilder(ctx, r.Builder().
		Select("person_settings_id", "forum_signature", "forum_profile_messages",
			"stats_hidden", "created_on", "updated_on").
		From("person_settings").
		Where(sq.Eq{"steam_id": steamID.Int64()}))

	if errRow != nil {
		return settings, database.Err(errRow)
	}

	settings.SteamID = steamID

	if errScan := row.Scan(&settings.PersonSettingsID, &settings.ForumSignature,
		&settings.ForumProfileMessages, &settings.StatsHidden, &settings.CreatedOn, &settings.UpdatedOn); errScan != nil {
		if errors.Is(database.Err(errScan), database.ErrNoResult) {
			settings.ForumProfileMessages = true

			return settings, nil
		}

		return settings, database.Err(errScan)
	}

	if r.centerProjectiles {
		rows, errRow := r.QueryBuilder(ctx, r.Builder().
			Select("name", "value").
			From("sm_cookie_cache").
			Join("sm_cookies ON cookie_id=id").
			Where(sq.And{
				sq.Eq{"player": steamID.Steam(false)},
				sq.Eq{"name": "tf2centerprojectiles"},
			}))
		if errRow != nil {
			return settings, database.Err(errRow)
		}

		defer rows.Close()

		for rows.Next() {
			key := ""
			value := ""

			if errScan := rows.Scan(&key, &value); errScan != nil {
				return settings, database.Err(errScan)
			}

			if key == "tf2centerprojectiles" {
				settings.CenterProjectiles = makeBool(value == "1")
			}
		}
	}

	return settings, nil
}

// Helper to make a bool pointer, useful for optional json fields.
func makeBool(v bool) *bool { return &v }

// Format booleans for storage as a sourcemod Clientpref.
func boolToStringDigit(b bool) string {
	if b {
		return "1"
	}

	return "0"
}

func (r *Repository) SaveSettings(ctx context.Context, settings *Settings) error {
	const (
		query = `
    INSERT INTO sm_cookie_cache (player, cookie_id, value, timestamp)
    VALUES ($1, (select id from sm_cookies where name='tf2centerprojectiles'), $2, cast(extract(epoch from current_timestamp) as integer))
    ON CONFLICT (player, cookie_id)
    DO UPDATE SET value = EXCLUDED.value, timestamp = EXCLUDED.timestamp
    RETURNING value;`
	)

	if !settings.SteamID.Valid() {
		return steamid.ErrDecodeSID
	}

	settings.UpdatedOn = time.Now()

	var errSiteSettings error

	if settings.PersonSettingsID == 0 {
		settings.CreatedOn = settings.UpdatedOn

		errSiteSettings = database.Err(r.ExecInsertBuilderWithReturnValue(ctx, r.Builder().
			Insert("person_settings").
			SetMap(map[string]any{
				"steam_id":               settings.SteamID.Int64(),
				"forum_signature":        settings.ForumSignature,
				"forum_profile_messages": settings.ForumProfileMessages,
				"stats_hidden":           settings.StatsHidden,
				"created_on":             settings.CreatedOn,
				"updated_on":             settings.UpdatedOn,
			}).
			Suffix("RETURNING person_settings_id"),
			&settings.PersonSettingsID))
	} else {
		errSiteSettings = database.Err(r.ExecUpdateBuilder(ctx, r.Builder().
			Update("person_settings").
			SetMap(map[string]any{
				"forum_signature":        settings.ForumSignature,
				"forum_profile_messages": settings.ForumProfileMessages,
				"stats_hidden":           settings.StatsHidden,
				"updated_on":             settings.UpdatedOn,
			}).
			Where(sq.Eq{"steam_id": settings.SteamID.Int64()})))
	}

	value := ""

	var errGameSettings error
	if r.centerProjectiles && settings.CenterProjectiles != nil {
		// TODO test this
		_ = database.Err(r.QueryRow(ctx, query,
			settings.SteamID.Steam(false),
			boolToStringDigit(*settings.CenterProjectiles)).Scan(&value))
	}

	return errors.Join(errSiteSettings, errGameSettings)
}
