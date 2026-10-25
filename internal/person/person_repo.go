package person

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type Repository struct {
	database.Database

	clientPrefsEnabled bool
}

func NewRepository(database database.Database, clientPrefsEnabled bool) Repository {
	return Repository{Database: database, clientPrefsEnabled: clientPrefsEnabled}
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
			realname, timecreated, loccountrycode, locstatecode, loccityid,
			discord_id, community_banned, vac_bans, game_bans, economy_ban, days_since_last_ban,
			updated_on_steam, muted,  created_on, updated_on)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
		ON CONFLICT (steam_id) DO UPDATE SET
			communityvisibilitystate = $2, profilestate = $3,
			personaname = $4, avatarhash = $5, personastate = $6,
			realname = $7, timecreated = $8, loccountrycode = $9, locstatecode = $10, loccityid = $11,
			discord_id = $12, community_banned = $13,  vac_bans = $14, game_bans = $15, economy_ban = $16, days_since_last_ban = $17,
			updated_on_steam = $18, muted = $19, updated_on=$21`

	return database.Err(r.Exec(ctx, query, person.SteamID.Int64(), person.VisibilityState,
		person.ProfileState, person.PersonaName,
		person.AvatarHash, person.PersonaState, person.RealName,
		person.TimeCreated, person.LocCountryCode, person.LocStateCode,
		person.LocCityID, person.DiscordID, person.CommunityBanned,
		person.VACBans, person.GameBans, person.EconomyBan, person.DaysSinceLastBan, person.UpdatedOnSteam,
		person.Muted, person.CreatedOn, person.UpdatedOn))
}

func normalizeStringLikeQuery(input string) string {
	space := regexp.MustCompile(`\s+`)

	return fmt.Sprintf("%%%s%%", strings.ToLower(strings.Trim(space.ReplaceAllString(input, "%"), "%")))
}

func (r *Repository) Query(ctx context.Context, query Query) (People, uint64, error) { //nolint:cyclop
	builder := r.Builder().
		Select("p.steam_id", "p.created_on", "p.updated_on",
			"p.communityvisibilitystate", "p.profilestate", "p.personaname", "p.avatarhash", "p.personastate", "p.realname", "p.timecreated",
			"p.loccountrycode", "p.locstatecode", "p.loccityid", "p.discord_id",
			"p.community_banned", "p.vac_bans", "p.game_bans", "p.economy_ban", "p.days_since_last_ban",
			"p.updated_on_steam", "p.muted", "coalesce(pt.patreon_id, '')",
			// Newest non-deleted, unexpired ban, or 0 when the player has no active ban.
			`COALESCE((SELECT b.ban_id FROM ban b WHERE b.target_id = p.steam_id AND b.deleted = false AND b.valid_until > now() ORDER BY b.ban_id DESC LIMIT 1), 0)::int AS active_ban_id`).
		From("person p").
		LeftJoin("auth_patreon pt USING (steam_id)")

	constraints := sq.And{}

	minDate := time.Date(2007, 0, 0, 0, 0, 0, 0, time.UTC)

	if query.SteamUpdateOlderThan.After(minDate) {
		// builder = builder.OrderBy("p.updated_on_steam ASC")
		constraints = append(constraints, sq.Lt{"p.updated_on_steam": query.SteamUpdateOlderThan})
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
	case query.TimeCreatedAfter != nil && !query.TimeCreatedAfter.IsZero() && !query.TimeCreatedBefore.IsZero() && query.TimeCreatedAfter.After(minDate) && query.TimeCreatedBefore != nil && query.TimeCreatedBefore.After(minDate):
		constraints = append(constraints, sq.Expr("p.timecreated BETWEEN $1 AND $2", query.TimeCreatedAfter.Unix(), query.TimeCreatedBefore.Unix()))
	case query.TimeCreatedAfter != nil && !query.TimeCreatedAfter.IsZero() && !query.TimeCreatedAfter.After(minDate):
		constraints = append(constraints, sq.GtOrEq{"p.timecreated": query.TimeCreatedAfter.Unix()})
	case query.TimeCreatedBefore != nil && !query.TimeCreatedBefore.IsZero() && !query.TimeCreatedBefore.After(minDate):
		constraints = append(constraints, sq.LtOrEq{"p.timecreated": query.TimeCreatedBefore.Unix()})
	}
	builder = query.ApplyLimitOffsetDefault(builder)
	builder = query.ApplySafeOrder(builder, map[string][]string{
		"p.": {
			"steam_id", "created_on", "updated_on",
			"communityvisibilitystate", "profilestate", "personaname", "avatarhash", "personastate",
			"realname", "timecreated", "loccountrycode", "locstatecode", "loccityid",
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
				&person.LocCityID, &person.DiscordID, &person.CommunityBanned,
				&person.VACBans, &person.GameBans, &person.EconomyBan, &person.DaysSinceLastBan,
				&person.UpdatedOnSteam, &person.Muted, &person.PatreonID, &person.BanID); errScan != nil {
			return nil, 0, errors.Join(errScan, database.ErrScanResult)
		}

		// Rows loaded from the database already exist, so they must not be
		// treated as new. Otherwise Expired() is always true and every read
		// triggers a steam refresh.
		person.isNew = false

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
			"stats_hidden", "match_notifications", "created_on", "updated_on").
		From("person_settings").
		Where(sq.Eq{"steam_id": steamID.Int64()}))

	if errRow != nil {
		return settings, database.Err(errRow)
	}

	settings.SteamID = steamID

	if errScan := row.Scan(&settings.PersonSettingsID, &settings.ForumSignature,
		&settings.ForumProfileMessages, &settings.StatsHidden, &settings.MatchNotifications,
		&settings.CreatedOn, &settings.UpdatedOn); errScan != nil {
		if errors.Is(database.Err(errScan), database.ErrNoResult) {
			settings.ForumProfileMessages = true

			return settings, nil
		}

		return settings, database.Err(errScan)
	}

	return settings, nil
}

func (r *Repository) ClientPrefs(ctx context.Context, steamID steamid.SteamID) ([]ClientPref, error) {
	if !r.clientPrefsEnabled {
		return []ClientPref{}, nil
	}

	rows, errQuery := r.QueryBuilder(ctx, r.Builder().
		Select("c.id", "c.name", "COALESCE(c.description, '')", "COALESCE(c.access, 0)", "COALESCE(cc.value, '')").
		From("sm_cookies c").
		LeftJoin("sm_cookie_cache cc ON cc.cookie_id = c.id AND cc.player = ?", steamID.Steam(false)).
		Where("COALESCE(c.access, 0) <> 2").
		OrderBy("c.id"))
	if errQuery != nil {
		return nil, database.Err(errQuery)
	}

	defer rows.Close()

	prefs := make([]ClientPref, 0)

	for rows.Next() {
		var pref ClientPref
		if errScan := rows.Scan(&pref.CookieID, &pref.Name, &pref.Description, &pref.Access, &pref.Value); errScan != nil {
			return nil, database.Err(errScan)
		}

		pref.Type = clientPrefType(pref.Name)
		prefs = append(prefs, pref)
	}

	if errRows := rows.Err(); errRows != nil {
		return nil, database.Err(errRows)
	}

	return prefs, nil
}

func (r *Repository) SaveClientPrefs(ctx context.Context, steamID steamid.SteamID, values map[string]string) error {
	if !r.clientPrefsEnabled || len(values) == 0 {
		return nil
	}

	if !steamID.Valid() {
		return steamid.ErrDecodeSID
	}

	const (
		deleteQuery = `
DELETE FROM sm_cookie_cache cc
USING sm_cookies c
WHERE cc.cookie_id = c.id
  AND cc.player = $1
  AND c.name = $2
  AND COALESCE(c.access, 0) = 0`

		upsertQuery = `
INSERT INTO sm_cookie_cache (player, cookie_id, value, timestamp)
SELECT $1, c.id, $2, CAST(EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) AS integer)
FROM sm_cookies c
WHERE c.name = $3
  AND COALESCE(c.access, 0) = 0
ON CONFLICT (player, cookie_id)
DO UPDATE SET value = EXCLUDED.value, timestamp = EXCLUDED.timestamp`
	)

	return r.WrapTx(ctx, func(tx pgx.Tx) error {
		for name, value := range values {
			if utf8.RuneCountInString(value) > 100 {
				return ErrClientPrefValueTooLong
			}

			if errValue := validateClientPrefValue(name, value); errValue != nil {
				return errValue
			}

			player := steamID.Steam(false)
			if value == "" {
				if _, errDelete := tx.Exec(ctx, deleteQuery, player, name); errDelete != nil {
					return errDelete //nolint:wrapcheck
				}

				continue
			}

			if _, errUpsert := tx.Exec(ctx, upsertQuery, player, value, name); errUpsert != nil {
				return errUpsert //nolint:wrapcheck
			}
		}

		return nil
	})
}

func validateClientPrefValue(name string, value string) error {
	if value == "" {
		return nil
	}

	switch clientPrefType(name) {
	case ClientPrefTypeBoolean:
		if value != "0" && value != "1" {
			return ErrClientPrefInvalidValue
		}
	case ClientPrefTypeNumber:
		if _, errNumber := strconv.Atoi(value); errNumber != nil {
			return ErrClientPrefInvalidValue
		}
	}

	return nil
}

func (r *Repository) SaveSettings(ctx context.Context, settings *Settings) error {
	if !settings.SteamID.Valid() {
		return steamid.ErrDecodeSID
	}

	settings.UpdatedOn = time.Now()

	if settings.PersonSettingsID == 0 {
		settings.CreatedOn = settings.UpdatedOn

		return database.Err(r.ExecInsertBuilderWithReturnValue(ctx, r.Builder().
			Insert("person_settings").
			SetMap(map[string]any{
				"steam_id":               settings.SteamID.Int64(),
				"forum_signature":        settings.ForumSignature,
				"forum_profile_messages": settings.ForumProfileMessages,
				"stats_hidden":           settings.StatsHidden,
				"match_notifications":    settings.MatchNotifications,
				"created_on":             settings.CreatedOn,
				"updated_on":             settings.UpdatedOn,
			}).
			Suffix("RETURNING person_settings_id"),
			&settings.PersonSettingsID))
	}

	return database.Err(r.ExecUpdateBuilder(ctx, r.Builder().
		Update("person_settings").
		SetMap(map[string]any{
			"forum_signature":        settings.ForumSignature,
			"forum_profile_messages": settings.ForumProfileMessages,
			"stats_hidden":           settings.StatsHidden,
			"match_notifications":    settings.MatchNotifications,
			"updated_on":             settings.UpdatedOn,
		}).
		Where(sq.Eq{"steam_id": settings.SteamID.Int64()})))
}
