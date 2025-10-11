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
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type Repository struct {
	conf config.Config
	db   database.Database
}

func NewRepository(conf config.Config, database database.Database) Repository {
	return Repository{conf: conf, db: database}
}

func (r *Repository) DropPerson(ctx context.Context, steamID steamid.SteamID) error {
	return database.DBErr(r.db.ExecDeleteBuilder(ctx, r.db.
		Builder().
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
			updated_on_steam, muted, playerqueue_chat_status, playerqueue_chat_reason, created_on, updated_on)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
		ON CONFLICT (steam_id) DO UPDATE SET
			communityvisibilitystate = $2, profilestate = $3,
			personaname = $4, avatarhash = $5, personastate = $6,
			realname = $7, timecreated = $8, loccountrycode = $9, locstatecode = $10, loccityid = $11,  permission_level = $12,
			discord_id = $13, community_banned = $14,  vac_bans = $15, game_bans = $16, economy_ban = $17, days_since_last_ban = $18,
			updated_on_steam = $19, muted =$20, playerqueue_chat_status =$21, playerqueue_chat_reason=$22, updated_on=$24`

	return database.DBErr(r.db.Exec(ctx, query, person.SteamID.Int64(), person.VisibilityState,
		person.ProfileState, person.PersonaName,
		person.AvatarHash, person.PersonaState, person.RealName,
		person.TimeCreated, person.LocCountryCode, person.LocStateCode,
		person.LocCityID, person.PermissionLevel, person.DiscordID, person.CommunityBanned,
		person.VACBans, person.GameBans, person.EconomyBan, person.DaysSinceLastBan, person.UpdatedOnSteam,
		person.Muted, person.PlayerqueueChatStatus, person.PlayerqueueChatReason, person.CreatedOn, person.UpdatedOn))
}

func normalizeStringLikeQuery(input string) string {
	space := regexp.MustCompile(`\s+`)

	return fmt.Sprintf("%%%s%%", strings.ToLower(strings.Trim(space.ReplaceAllString(input, "%"), "%")))
}

// TODO move to network or srcds?
func (r *Repository) GetSteamsAtAddress(ctx context.Context, addr net.IP) (steamid.Collection, error) {
	var ids steamid.Collection

	// TODO
	rows, errRows := r.db.QueryBuilder(ctx, r.db.
		Builder().
		Select("DISTINCT steam_id").
		From("person_connections").
		Where(sq.Expr(fmt.Sprintf("ip_addr::inet >>= '::ffff:%s'::CIDR OR ip_addr::inet <<= '::ffff:%s'::CIDR", addr.String(), addr.String()))))
	if errRows != nil {
		return nil, database.DBErr(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var sid int64
		if errScan := rows.Scan(&sid); errScan != nil {
			return nil, database.DBErr(errScan)
		}

		ids = append(ids, steamid.New(sid))
	}

	return ids, nil
}

func (r *Repository) Query(ctx context.Context, query Query) (People, error) {
	builder := r.db.
		Builder().
		Select("p.steam_id", "p.created_on", "p.updated_on",
			"p.communityvisibilitystate", "p.profilestate", "p.personaname", "p.avatarhash", "p.personastate", "p.realname", "p.timecreated",
			"p.loccountrycode", "p.locstatecode", "p.loccityid", "p.permission_level", "p.discord_id",
			"p.community_banned", "p.vac_bans", "p.game_bans", "p.economy_ban", "p.days_since_last_ban",
			"p.updated_on_steam", "p.muted", "coalesce(pt.patreon_id, '')", "p.playerqueue_chat_status",
			"p.playerqueue_chat_reason").
		From("person p").
		LeftJoin("auth_patreon pt USING (steam_id)")

	conditions := sq.And{}

	if query.IP != "" {
		builder = builder.LeftJoin("person_connections pc ON p.steam_id = pc.steam_id")
		// TODO
		conditions = append(conditions, sq.Expr(fmt.Sprintf("ip_addr::inet >>= '::ffff:%s'::CIDR OR ip_addr::inet <<= '::ffff:%s'::CIDR", query.IP, query.IP)))
	}

	if !query.SteamUpdateOlderThan.IsZero() {
		builder = builder.OrderBy("p.updated_on_steam ASC")
		conditions = append(conditions, sq.Lt{"p.updated_on_steam": query.SteamUpdateOlderThan})
	}

	if query.WithPermissions > 0 {
		conditions = append(conditions, sq.GtOrEq{"p.permission_level": query.WithPermissions})
	}

	if query.IP != "" {
		addr := net.ParseIP(query.IP)
		if addr == nil {
			return nil, ErrNetworkInvalidIP
		}

		foundIDs, errFoundIDs := r.GetSteamsAtAddress(ctx, addr)
		if errFoundIDs != nil {
			if errors.Is(errFoundIDs, database.ErrNoResult) {
				return People{}, nil
			}

			return nil, database.DBErr(errFoundIDs)
		}

		conditions = append(conditions, sq.Eq{"p.steam_id": foundIDs})
	}

	if query.DiscordID != "" {
		conditions = append(conditions, sq.Eq{"p.discord_id": query.DiscordID})
	}

	// sq.Expr(fmt.Sprintf("ip_addr::inet >>= '::ffff:%s'::CIDR OR ip_addr::inet <<= '::ffff:%s'::CIDR", addr.String(), addr.String())))
	if len(query.SteamIDs) > 0 {
		conditions = append(conditions, sq.Eq{"p.steam_id": query.SteamIDs.ToInt64Slice()})
	}

	if query.Personaname != "" {
		// TODO add lower-cased functional index to avoid table scan
		conditions = append(conditions, sq.ILike{"p.personaname": normalizeStringLikeQuery(query.Personaname)})
	}

	builder = query.ApplyLimitOffsetDefault(builder)
	builder = query.ApplySafeOrder(builder, map[string][]string{
		"p.": {
			"steam_id", "created_on", "updated_on",
			"communityvisibilitystate", "profilestate", "personaname", "avatarhash", "personastate",
			"realname", "timecreated", "loccountrycode", "locstatecode", "loccityid", "p.permission_level",
			"discord_id", "community_banned", "vac_bans", "game_bans", "economy_ban", "days_since_last_ban",
			"updated_on_steam", "muted", "playerqueue_chat_status", "playerqueue_chat_reason",
		},
		"pt.": {"patreon_id"},
	}, "steam_id")

	var people People

	rows, errQuery := r.db.QueryBuilder(ctx, builder.Where(conditions))
	if errQuery != nil {
		return nil, database.DBErr(errQuery)
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
				&person.UpdatedOnSteam, &person.Muted, &person.PatreonID, &person.PlayerqueueChatStatus,
				&person.PlayerqueueChatReason); errScan != nil {
			return nil, errors.Join(errScan, database.ErrScanResult)
		}

		people = append(people, person)
	}

	return people, nil
}

func (r *Repository) Settings(ctx context.Context, steamID steamid.SteamID) (Settings, error) {
	var settings Settings

	row, errRow := r.db.QueryRowBuilder(ctx, r.db.
		Builder().
		Select("person_settings_id", "forum_signature", "forum_profile_messages",
			"stats_hidden", "created_on", "updated_on").
		From("person_settings").
		Where(sq.Eq{"steam_id": steamID.Int64()}))

	if errRow != nil {
		return settings, database.DBErr(errRow)
	}

	settings.SteamID = steamID

	if errScan := row.Scan(&settings.PersonSettingsID, &settings.ForumSignature,
		&settings.ForumProfileMessages, &settings.StatsHidden, &settings.CreatedOn, &settings.UpdatedOn); errScan != nil {
		if errors.Is(database.DBErr(errScan), database.ErrNoResult) {
			settings.ForumProfileMessages = true

			return settings, nil
		}

		return settings, database.DBErr(errScan)
	}

	if r.conf.Clientprefs.CenterProjectiles {
		rows, errRow := r.db.QueryBuilder(ctx, r.db.
			Builder().
			Select("name", "value").
			From("sm_cookie_cache").
			Join("sm_cookies ON cookie_id=id").
			Where(sq.And{
				sq.Eq{"player": steamID.Steam(false)},
				sq.Eq{"name": "tf2centerprojectiles"},
			}))
		if errRow != nil {
			return settings, database.DBErr(errRow)
		}

		defer rows.Close()

		for rows.Next() {
			key := ""
			value := ""

			if errScan := rows.Scan(&key, &value); errScan != nil {
				return settings, database.DBErr(errScan)
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

		errSiteSettings = database.DBErr(r.db.ExecInsertBuilderWithReturnValue(ctx, r.db.
			Builder().
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
		errSiteSettings = database.DBErr(r.db.ExecUpdateBuilder(ctx, r.db.
			Builder().
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
	if r.conf.Clientprefs.CenterProjectiles && settings.CenterProjectiles != nil {
		errGameSettings = database.DBErr(r.db.QueryRow(ctx, query,
			settings.SteamID.Steam(false),
			boolToStringDigit(*settings.CenterProjectiles)).Scan(&value))
	}

	return errors.Join(errSiteSettings, errGameSettings)
}
