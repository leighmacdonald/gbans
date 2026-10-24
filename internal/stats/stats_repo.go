package stats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/maps"
	"github.com/leighmacdonald/gbans/pkg/demoparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type Variant int32

const (
	VariantOverallUnspecified = iota
	VariantKills
	VariantHealing
	VariantWeapons
	VariantClasses
)

func (v Variant) String() string {
	switch v {
	case VariantKills:
		return "overall"
	case VariantHealing:
		return "overall"
	case VariantWeapons:
		return "variants"
	case VariantClasses:
		return "variants"
	case VariantOverallUnspecified:
		fallthrough
	default:
		return "overall"
	}
}

type TimeBucket int32

const (
	TimeBucketUnspecified = 0
	TimeBucketDaily       = 1
	TimeBucketWeekly      = 7
	TimeBucketMonthly     = 31
	TimeBucketYearly      = 365
	TimeBucketAlltime     = 9999
)

func (t TimeBucket) String() string {
	switch t {
	case TimeBucketDaily:
		return "daily"
	case TimeBucketWeekly:
		return "weekly"
	case TimeBucketMonthly:
		return "monthly"
	case TimeBucketYearly:
		return "yearly"
	default:
		return "alltime" //nolint:misspell
	}
}

type Opts struct {
	query.Filter

	StatsBucketID uint32
	Variant       Variant
	VariantKey    string
	TimeBucket    TimeBucket
	TimeStamp     time.Time
}

func (o Opts) view() string {
	return fmt.Sprintf("stats_summary_%s_%s_view", o.TimeBucket, o.Variant)
}

// dateConstraint restricts rows to the time-bucket period containing
// TimeStamp. All-time views carry no date_bucket and are never constrained,
// and a zero TimeStamp disables filtering (used by tests/backfills).
func (o Opts) dateConstraint() sq.Sqlizer { //nolint:ireturn
	if o.TimeStamp.IsZero() {
		return nil
	}

	var trunc string

	switch o.TimeBucket {
	case TimeBucketDaily:
		trunc = "day"
	case TimeBucketWeekly:
		trunc = "week"
	case TimeBucketMonthly:
		trunc = "month"
	case TimeBucketYearly:
		trunc = "year"
	default:
		return nil
	}

	return sq.Expr("date_bucket = date_trunc('"+trunc+"', ?::timestamptz)", o.TimeStamp)
}

type Repository struct{ database.Database }

func (r Repository) Delete(ctx context.Context, demoID int32) error {
	if err := r.Exec(ctx, `DELETE FROM match WHERE demo_id = $1`, demoID); err != nil {
		return database.Err(err)
	}

	return nil
}

func (r Repository) MatchIDByDemoID(ctx context.Context, demoID int32) (uuid.UUID, error) {
	var matchID uuid.UUID
	if err := r.QueryRow(ctx, `SELECT match_id FROM match WHERE demo_id = $1`, demoID).Scan(&matchID); err != nil {
		return matchID, database.Err(err)
	}

	return matchID, nil
}

func (r Repository) Matches(ctx context.Context, opts MatchesOpts) ([]Match, uint64, error) {
	builder := r.Builder().Select("m.match_id", "m.server_id", "m.map_id", "mp.map_name",
		"m.demo_id", "s.stats_bucket_id", "s.bucket_name", "m.hostname",
		"m.score_red", "m.score_blu", "m.duration_ms", "m.created_on",
		"srv.name", "srv.short_name", "m.start_time").
		From("match m").
		LeftJoin("map mp USING(map_id)").
		LeftJoin("stats_bucket s USING(stats_bucket_id)").
		LeftJoin("server srv ON m.server_id = srv.server_id")
	var constraints sq.And
	if opts.mapID > 0 {
		constraints = append(constraints, sq.Eq{"m.map_id": opts.mapID})
	}
	if len(opts.serverIDs) > 0 {
		constraints = append(constraints, sq.Eq{"m.server_id": opts.serverIDs})
	}
	if opts.statsBucketID > 0 {
		constraints = append(constraints, sq.Eq{"s.stats_bucket_id": opts.statsBucketID})
	}
	if !opts.startTimeStart.IsZero() {
		constraints = append(constraints, sq.GtOrEq{"m.start_time": opts.startTimeStart})
	}
	if !opts.startTimeEnd.IsZero() {
		constraints = append(constraints, sq.LtOrEq{"m.start_time": opts.startTimeEnd})
	}
	// The bucket column is identified by its display name in the UI, but is
	// stored and ordered by the bucket id.
	if opts.OrderBy == "stats_bucket_name" {
		opts.OrderBy = "stats_bucket_id"
	}
	builder = opts.ApplySafeOrder(opts.ApplyLimitOffsetDefault(builder), map[string][]string{
		"m.": {"map_id", "stats_bucket_id", "duration_ms", "start_time"},
	}, "m.start_time")
	rows, errRows := r.QueryBuilder(ctx, opts.ApplyLimitOffset(builder, 100).Where(constraints))
	if errRows != nil {
		return nil, 0, database.Err(errRows)
	}

	var matches []Match
	for rows.Next() {
		var match Match
		if err := rows.Scan(&match.MatchID, &match.ServerID, &match.MapID, &match.MapName,
			&match.DemoID, &match.StatsBucketID, &match.StatsBucketName, &match.Hostname,
			&match.ScoreRed, &match.ScoreBlu, &match.DurationMs, &match.CreatedOn,
			&match.ServerName, &match.ServerNameShort, &match.StartTime); err != nil {
			return nil, 0, database.Err(err)
		}

		matches = append(matches, match)
	}

	if rows.Err() != nil {
		return nil, 0, database.Err(rows.Err())
	}

	countQuery := r.Builder().
		Select("count(m.match_id)").
		From("match m").
		LeftJoin("map mp USING(map_id)").
		LeftJoin("stats_bucket s USING(stats_bucket_id)").
		LeftJoin("server srv ON m.server_id = srv.server_id").
		Where(constraints)
	count, errCount := r.GetCount(ctx, countQuery)
	if errCount != nil {
		return nil, 0, database.Err(errCount)
	}

	return matches, count, nil
}

func (r Repository) WeaponList(ctx context.Context) ([]string, error) {
	const query = `SELECT variant FROM stats_weapons_view`

	rows, errRows := r.Database.Query(ctx, query)
	if errRows != nil {
		return nil, database.Err(errRows)
	}

	var weapons []string
	for rows.Next() {
		var weapon string
		if err := rows.Scan(&weapon); err != nil {
			return nil, database.Err(err)
		}

		weapons = append(weapons, weapon)
	}

	return weapons, nil
}

func (r Repository) Buckets(ctx context.Context) ([]Bucket, error) {
	const query = `SELECT stats_bucket_id, bucket_name, is_enabled, is_default FROM stats_bucket`

	var buckets []Bucket
	rows, errRows := r.Database.Query(ctx, query)
	if errRows != nil {
		return nil, database.Err(errRows)
	}

	for rows.Next() {
		var bucket Bucket
		if err := rows.Scan(&bucket.BucketID, &bucket.BucketName, &bucket.IsEnabled, &bucket.IsDefault); err != nil {
			return nil, database.Err(err)
		}
		buckets = append(buckets, bucket)
	}

	if rows.Err() != nil {
		return nil, database.Err(rows.Err())
	}

	return buckets, nil
}

func NewRepository(database database.Database) Repository {
	return Repository{Database: database}
}

// rankColumn returns the summary view column that ranks players for the
// variant: leaderboards rank by the selected stat, not always by points.
func (o Opts) rankColumn() string {
	switch o.Variant {
	case VariantKills:
		return "rank_kills"
	case VariantHealing:
		return "rank_healing"
	default:
		return "rank"
	}
}

// eligibilityConstraint returns the minimum stat requirement for the selected
// leaderboard. Every view except healing requires a minimum kill count; the
// healing view instead requires a minimum amount of healing. This keeps
// low-sample-size players off the boards.
func (o Opts) eligibilityConstraint() sq.Sqlizer {
	if o.Variant == VariantHealing {
		return sq.GtOrEq{"healing": MinHealing}
	}

	return sq.GtOrEq{"kills": MinKills}
}

// MyRank returns steamID's rank in the bucket, variant and time window
// described by opts, or 0 when the player has no stats in that window.
func (r Repository) MyRank(ctx context.Context, opts Opts, steamID steamid.SteamID) (uint64, error) {
	constraints := sq.And{
		sq.Eq{"stats_bucket_id": opts.StatsBucketID},
		sq.Eq{"steam_id": steamID.Int64()},
		opts.eligibilityConstraint(),
	}
	if opts.Variant == VariantWeapons || opts.Variant == VariantClasses {
		constraints = append(constraints, sq.Eq{"variant": opts.VariantKey})
	}
	if dateConstraint := opts.dateConstraint(); dateConstraint != nil {
		constraints = append(constraints, dateConstraint)
	}

	var rank uint64
	row, errRow := r.QueryRowBuilder(ctx, r.Builder().
		Select(opts.rankColumn()+" AS rank").
		From(opts.view()).
		Where(constraints))
	if errRow != nil {
		return 0, database.Err(errRow)
	}
	if err := row.Scan(&rank); errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	} else if err != nil {
		return 0, database.Err(err)
	}

	return rank, nil
}

// MyIndex returns the 0-based position of steamID's row in the full result
// set for the window described by opts, ordered by (rank, steam_id) exactly
// as Query returns rows. It identifies the user's row even when players are
// tied at the same rank. 0 when the user has no stats in the window.
func (r Repository) MyIndex(ctx context.Context, opts Opts, steamID steamid.SteamID, rank uint64) (uint64, error) {
	constraints := sq.And{
		sq.Eq{"stats_bucket_id": opts.StatsBucketID},
		// Only players meeting the leaderboard's minimum sample size can
		// appear ahead of, or tie with, the user.
		opts.eligibilityConstraint(),
		// Rows sorting ahead of the user: strictly better rank, or the same
		// rank with a lower steam_id (the deterministic tiebreaker).
		sq.Or{
			sq.Lt{opts.rankColumn(): rank},
			sq.And{
				sq.Eq{opts.rankColumn(): rank},
				sq.Lt{"steam_id": steamID.Int64()},
			},
		},
	}
	if opts.Variant == VariantWeapons || opts.Variant == VariantClasses {
		constraints = append(constraints, sq.Eq{"variant": opts.VariantKey})
	}
	if dateConstraint := opts.dateConstraint(); dateConstraint != nil {
		constraints = append(constraints, dateConstraint)
	}

	return r.GetCount(ctx, r.Builder().
		Select("count(*) as count").
		From(opts.view()).
		Where(constraints))
}

// PlayerKillsAndRank returns steamID's kills and their kills-leaderboard rank
// for the window described by opts. opts.Variant must be VariantKills. The
// rank is 0 when the player has fewer than MinKills kills in the window, even
// though kills itself is always reported.
func (r Repository) PlayerKillsAndRank(ctx context.Context, opts Opts, steamID steamid.SteamID) (uint64, uint64, error) {
	var (
		rank  uint64
		kills uint64
	)

	constraints := sq.And{
		sq.Eq{"stats_bucket_id": opts.StatsBucketID},
		sq.Eq{"steam_id": steamID.Int64()},
	}
	if opts.Variant == VariantWeapons || opts.Variant == VariantClasses {
		constraints = append(constraints, sq.Eq{"variant": opts.VariantKey})
	}
	if dateConstraint := opts.dateConstraint(); dateConstraint != nil {
		constraints = append(constraints, dateConstraint)
	}

	row, errRow := r.QueryRowBuilder(ctx, r.Builder().
		Select(opts.rankColumn()+" AS rank", "kills").
		From(opts.view()).
		Where(constraints))
	if errRow != nil {
		return 0, 0, database.Err(errRow)
	}
	if err := row.Scan(&rank, &kills); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, nil
		}

		return 0, 0, database.Err(err)
	}

	// The rank only has standing once the player clears the leaderboard's
	// minimum sample size; below it they are unranked though their kills are
	// still reported.
	if kills < MinKills {
		rank = 0
	}

	return rank, kills, nil
}

func (r Repository) loadOverallView(ctx context.Context, statsBucketID uint32, opts Opts) ([]any, uint64, error) {
	constraints := sq.And{
		sq.Eq{"stats_bucket_id": statsBucketID},
		opts.eligibilityConstraint(),
	}

	if dateConstraint := opts.dateConstraint(); dateConstraint != nil {
		constraints = append(constraints, dateConstraint)
	}

	builder := r.Builder().
		Select(opts.rankColumn()+" AS rank", "steam_id", "personaname", "avatarhash", "points", "connection_count", "bonus_points", "kills", "assists", "deaths",
			"postround_kills", "postround_assists", "postround_deaths",
			"preround_healing", "healing", "postround_healing", "drops", "near_full_charge_death",
			"charges_uber", "charges_kritz", "charges_vacc", "charges_quickfix",
			"damage", "damage_taken", "dominations", "dominated", "revenges", "revenged",
			"airshots", "headshots", "headshot_kills", "backstabs", "backstab_kills",
			"was_headshot", "was_backstabbed", "shots", "hits", "objects_built", "objects_destroyed",
			"scoreboard_kills", "scoreboard_assists", "scoreboard_deaths", "scoreboard_healing", "suicides",
			"captures", "captures_blocked", "scoreboard_damage", "extinguishes", "ignites",
			"heals", "healed", "crossbow_heals", "crossbow_healing", "heal_on_hit", "building_healing",
			"dropped_ubers", "reflects", "defenses", "direct_hits", "teleports", "push_distance",
			"environmental_deaths", "environmental_kills",
			"object_placed", "object_upgraded", "object_carried", "object_dropped", "object_removed", "object_detonated",
			"ammo_packs", "health_packs", "health_pack_healing").
		From(opts.view()).
		Where(constraints)
	// steam_id is a deterministic tiebreaker so tied ranks (rank() shares a
	// number across ties) always page in a stable order.
	builder = builder.OrderBy("rank ASC", "steam_id ASC")
	builder = opts.ApplyLimitOffset(builder, 10000)

	rows, errRows := r.QueryBuilder(ctx, builder)
	if errRows != nil {
		return nil, 0, database.Err(errRows)
	}

	var variantStats []any
	for rows.Next() {
		var overallStat OverallStats
		if err := rows.Scan(
			&overallStat.Rank, &overallStat.SteamID, &overallStat.Personaname, &overallStat.AvatarHash, &overallStat.Points, &overallStat.ConnectionCount, &overallStat.BonusPoints, &overallStat.Kills, &overallStat.Assists, &overallStat.Deaths,
			&overallStat.PostroundKills, &overallStat.PostroundAssists, &overallStat.PostroundDeaths,
			&overallStat.PreroundHealing, &overallStat.Healing, &overallStat.PostroundHealing, &overallStat.Drops, &overallStat.NearFullChargeDeath,
			&overallStat.ChargesUber, &overallStat.ChargesKritz, &overallStat.ChargesVacc, &overallStat.ChargesQuickfix,
			&overallStat.Damage, &overallStat.DamageTaken, &overallStat.Dominations, &overallStat.Dominated, &overallStat.Revenges, &overallStat.Revenged,
			&overallStat.Airshots, &overallStat.Headshots, &overallStat.HeadshotKills, &overallStat.Backstabs, &overallStat.BackstabKills,
			&overallStat.WasHeadshot, &overallStat.WasBackstabbed, &overallStat.Shots, &overallStat.Hits, &overallStat.ObjectsBuilt, &overallStat.ObjectsDestroyed,
			&overallStat.ScoreboardKills, &overallStat.ScoreboardAssists, &overallStat.ScoreboardDeaths, &overallStat.ScoreboardHealing, &overallStat.Suicides,
			&overallStat.Captures, &overallStat.CapturesBlocked, &overallStat.ScoreboardDamage, &overallStat.Extinguishes, &overallStat.Ignites,
			&overallStat.Heals, &overallStat.Healed, &overallStat.CrossbowHeals, &overallStat.CrossbowHealing, &overallStat.HealOnHit, &overallStat.BuildingHealing,
			&overallStat.DroppedUbers, &overallStat.Reflects, &overallStat.Defenses, &overallStat.DirectHits, &overallStat.Teleports, &overallStat.PushDistance,
			&overallStat.EnvironmentalDeaths, &overallStat.EnvironmentalKills,
			&overallStat.ObjectPlaced, &overallStat.ObjectUpgraded, &overallStat.ObjectCarried, &overallStat.ObjectDropped, &overallStat.ObjectRemoved, &overallStat.ObjectDetonated,
			&overallStat.AmmoPacks, &overallStat.HealthPacks, &overallStat.HealthPackHealing); err != nil {
			return nil, 0, database.Err(err)
		}

		variantStats = append(variantStats, overallStat)
	}

	if rows.Err() != nil {
		return nil, 0, database.Err(rows.Err())
	}

	count, errCount := r.GetCount(ctx, r.Builder().
		Select("count(*) as count").
		From(opts.view()).
		Where(constraints))
	if errCount != nil {
		return nil, 0, database.Err(errCount)
	}

	return variantStats, count, nil
}

func (r Repository) loadVariantView(ctx context.Context, statsBucketID uint32, opts Opts) ([]any, uint64, error) {
	constraints := sq.And{
		sq.Eq{"stats_bucket_id": statsBucketID},
		sq.Eq{"variant": opts.VariantKey},
		opts.eligibilityConstraint(),
	}

	if dateConstraint := opts.dateConstraint(); dateConstraint != nil {
		constraints = append(constraints, dateConstraint)
	}

	builder := r.Builder().
		Select("rank", "steam_id", "personaname", "avatarhash", "kills", "assists", "deaths", "postround_kills",
			"postround_assists", "postround_deaths",
			"damage", "damage_taken", "dominations", "dominated", "revenges", "revenged",
			"airshots", "headshot_kills", "backstab_kills", "headshots", "backstabs",
			"was_headshot", "was_backstabbed", "captures", "captures_blocked",
			"preround_healing", "healing", "postround_healing", "drops", "near_full_charge_death",
			"charges_uber", "charges_kritz", "charges_vacc", "charges_quickfix",
			"shots", "hits", "objects_built", "objects_destroyed",
			"heals", "healed", "crossbow_heals", "crossbow_healing", "heal_on_hit", "extinguishes", "building_healing",
			"dropped_ubers", "reflects", "defenses", "direct_hits", "teleports", "push_distance",
			"environmental_deaths", "environmental_kills",
			"object_placed", "object_upgraded", "object_carried", "object_dropped", "object_removed", "object_detonated",
			"ammo_packs", "health_packs", "health_pack_healing",
		).
		From(opts.view()).
		Where(constraints)
	// steam_id is a deterministic tiebreaker so tied ranks (rank() shares a
	// number across ties) always page in a stable order.
	builder = builder.OrderBy("rank ASC", "steam_id ASC")
	builder = opts.ApplyLimitOffset(builder, 10000)

	rows, errRows := r.QueryBuilder(ctx, builder)
	if errRows != nil {
		return nil, 0, database.Err(errRows)
	}

	var variantStats []any
	for rows.Next() {
		var variantStat VariantStats
		if err := rows.Scan(
			&variantStat.Rank, &variantStat.SteamID, &variantStat.Personaname, &variantStat.AvatarHash, &variantStat.Kills, &variantStat.Assists, &variantStat.Deaths, &variantStat.PostroundKills,
			&variantStat.PostroundAssists, &variantStat.PostroundDeaths,
			&variantStat.Damage, &variantStat.DamageTaken, &variantStat.Dominations, &variantStat.Dominated, &variantStat.Revenges, &variantStat.Revenged,
			&variantStat.Airshots, &variantStat.HeadshotKills, &variantStat.BackstabKills, &variantStat.Headshots, &variantStat.Backstabs,
			&variantStat.WasHeadshot, &variantStat.WasBackstabbed, &variantStat.Captures, &variantStat.CapturesBlocked,
			&variantStat.PreroundHealing, &variantStat.Healing, &variantStat.PostroundHealing, &variantStat.Drops, &variantStat.NearFullChargeDeath,
			&variantStat.ChargesUber, &variantStat.ChargesKritz, &variantStat.ChargesVacc, &variantStat.ChargesQuickfix,
			&variantStat.Shots, &variantStat.Hits, &variantStat.ObjectsBuilt, &variantStat.ObjectsDestroyed,
			&variantStat.Heals, &variantStat.Healed, &variantStat.CrossbowHeals, &variantStat.CrossbowHealing, &variantStat.HealOnHit, &variantStat.Extinguishes, &variantStat.BuildingHealing,
			&variantStat.DroppedUbers, &variantStat.Reflects, &variantStat.Defenses, &variantStat.DirectHits, &variantStat.Teleports, &variantStat.PushDistance,
			&variantStat.EnvironmentalDeaths, &variantStat.EnvironmentalKills,
			&variantStat.ObjectPlaced, &variantStat.ObjectUpgraded, &variantStat.ObjectCarried, &variantStat.ObjectDropped, &variantStat.ObjectRemoved, &variantStat.ObjectDetonated,
			&variantStat.AmmoPacks, &variantStat.HealthPacks, &variantStat.HealthPackHealing); err != nil {
			return nil, 0, database.Err(err)
		}

		variantStats = append(variantStats, variantStat)
	}

	if rows.Err() != nil {
		return nil, 0, database.Err(rows.Err())
	}

	count, errCount := r.GetCount(ctx, r.Builder().
		Select("count(*) as count").
		From(opts.view()).
		Where(constraints))
	if errCount != nil {
		return nil, 0, database.Err(errCount)
	}

	return variantStats, count, nil
}

func (r Repository) Query(ctx context.Context, opts Opts) ([]any, uint64, error) {
	switch opts.Variant {
	case VariantClasses:
		fallthrough
	case VariantWeapons:
		return r.loadVariantView(ctx, opts.StatsBucketID, opts)
	default:
		return r.loadOverallView(ctx, opts.StatsBucketID, opts)
	}
}

func (r Repository) Match(ctx context.Context, matchID uuid.UUID) (*Match, error) {
	match, errMatch := r.getMatch(ctx, matchID)
	if errMatch != nil {
		return nil, errMatch
	}

	if errRounds := r.getRounds(ctx, match); errRounds != nil {
		return nil, errRounds
	}

	if errPlayers := r.getRoundPlayers(ctx, match); errPlayers != nil {
		return nil, errPlayers
	}

	if errVariants := r.getRoundPlayersVariants(ctx, match); errVariants != nil {
		return nil, errVariants
	}

	if errChat := r.getChat(ctx, match); errChat != nil {
		return nil, errChat
	}

	if errKills := r.getKills(ctx, match); errKills != nil {
		return nil, errKills
	}

	if errEvents := r.getEvents(ctx, match); errEvents != nil {
		return nil, errEvents
	}

	return match, nil
}

func (r Repository) getChat(ctx context.Context, match *Match) error {
	const query = `
		SELECT
			person_message_id, steam_id, body, demo_tick, persona_name
		FROM person_messages
		WHERE match_id = $1`

	rows, errRows := r.Database.Query(ctx, query, match.MatchID)
	if errRows != nil {
		return database.Err(errRows)
	}

	for rows.Next() {
		var chatLog MatchChatLog
		if err := rows.Scan(&chatLog.PersonMessageID, &chatLog.SteamID, &chatLog.Body, &chatLog.DemoTick, &chatLog.Name); err != nil {
			return database.Err(err)
		}

		match.ChatLogs = append(match.ChatLogs, chatLog)
	}

	return nil
}

func (r Repository) getRoundPlayers(ctx context.Context, match *Match) error {
	const query = `
		SELECT
			p.round_id, p.steam_id, p.team, p.mvp, p.tick_start, p.tick_end, p.kills, p.assists,
			p.postround_kills, p.postround_assists, p.postround_deaths, p.preround_healing, p.healing,
			p.postround_healing, p.drops, p.near_full_charge_death, p.charges_uber, p.charges_kritz, p.charges_vacc, p.charges_quickfix,
			p.damage, p.damage_taken, p.dominations, p.dominated, p.revenges, p.revenged,
			p.airshots, p.headshots, p.headshot_kills,
			p.backstabs, p.backstab_kills, p.captures, p.captures_blocked, p.was_headshot, p.was_backstabbed, p.shots, p.hits,
			p.objects_built, p.objects_destroyed, p.points, p.connection_count, p.bonus_points,
			p.scoreboard_kills, p.scoreboard_assists, p.scoreboard_healing, p.scoreboard_deaths, p.scoreboard_damage,
			p.suicides, p.extinguishes, p.ignites,
			p.heals, p.healed, p.crossbow_heals, p.crossbow_healing, p.heal_on_hit, p.building_healing,
			p.dropped_ubers, p.reflects, p.defenses, p.direct_hits, p.teleports, p.push_distance,
			p.environmental_deaths, p.environmental_kills,
			p.object_placed, p.object_upgraded, p.object_carried, p.object_dropped, p.object_removed, p.object_detonated,
			p.ammo_packs, p.health_packs, p.health_pack_healing,
			pe.personaname, pe.avatarhash
		FROM
			match_round_player p
		LEFT JOIN
			match_round r ON r.round_id = p.round_id
		LEFT JOIN
			person pe ON p.steam_id = pe.steam_id
		WHERE
			r.match_id = $1`
	rows, errRows := r.Database.Query(ctx, query, match.MatchID)
	if errRows != nil {
		return database.Err(errRows)
	}

	for rows.Next() {
		var mrp MatchOverallStatsRound
		if err := rows.Scan(&mrp.RoundID, &mrp.SteamID, &mrp.Team, &mrp.MVP, &mrp.TickStart, &mrp.TickEnd, &mrp.Kills, &mrp.Assists,
			&mrp.PostroundKills, &mrp.PostroundAssists, &mrp.PostroundDeaths, &mrp.PreroundHealing, &mrp.Healing,
			&mrp.PostroundHealing, &mrp.Drops, &mrp.NearFullChargeDeath, &mrp.ChargesUber, &mrp.ChargesKritz, &mrp.ChargesVacc, &mrp.ChargesQuickfix,
			&mrp.Damage, &mrp.DamageTaken, &mrp.Dominations, &mrp.Dominated, &mrp.Revenges, &mrp.Revenged,
			&mrp.Airshots, &mrp.Headshots, &mrp.HeadshotKills,
			&mrp.Backstabs, &mrp.BackstabKills, &mrp.Captures, &mrp.CapturesBlocked, &mrp.WasHeadshot, &mrp.WasBackstabbed, &mrp.Shots, &mrp.Hits,
			&mrp.ObjectsBuilt, &mrp.ObjectsDestroyed, &mrp.Points, &mrp.ConnectionCount, &mrp.BonusPoints,
			&mrp.ScoreboardKills, &mrp.ScoreboardAssists, &mrp.ScoreboardHealing, &mrp.ScoreboardDeaths, &mrp.ScoreboardDamage,
			&mrp.Suicides, &mrp.Extinguishes, &mrp.Ignites,
			&mrp.Heals, &mrp.Healed, &mrp.CrossbowHeals, &mrp.CrossbowHealing, &mrp.HealOnHit, &mrp.BuildingHealing,
			&mrp.DroppedUbers, &mrp.Reflects, &mrp.Defenses, &mrp.DirectHits, &mrp.Teleports, &mrp.PushDistance,
			&mrp.EnvironmentalDeaths, &mrp.EnvironmentalKills,
			&mrp.ObjectPlaced, &mrp.ObjectUpgraded, &mrp.ObjectCarried, &mrp.ObjectDropped, &mrp.ObjectRemoved, &mrp.ObjectDetonated,
			&mrp.AmmoPacks, &mrp.HealthPacks, &mrp.HealthPackHealing,
			&mrp.Personaname, &mrp.AvatarHash,
		); err != nil {
			return database.Err(err)
		}

		match.Players = append(match.Players, mrp)
	}

	return nil
}

func (r Repository) getRoundPlayersVariants(ctx context.Context, match *Match) error {
	const query = `
		SELECT
			w.variant, w.round_id, w.steam_id, w.kills, w.assists, w.deaths, w.postround_kills, w.postround_assists,
			w.postround_deaths, w.preround_healing, w.healing, w.postround_healing, w.drops, w.near_full_charge_death, w.charges_uber,
			w.charges_kritz, w.charges_vacc, w.charges_quickfix,
			w.damage, w.damage_taken, w.dominations, w.dominated, w.revenges, w.revenged,
			w.airshots, w.headshots, w.headshot_kills, w.backstabs, w.backstab_kills, w.captures, w.captures_blocked, w.was_headshot, w.was_backstabbed,
			w.shots, w.hits, w.objects_built, w.objects_destroyed,
			w.heals, w.healed, w.crossbow_heals, w.crossbow_healing, w.heal_on_hit, w.extinguishes, w.building_healing,
			w.dropped_ubers, w.reflects, w.defenses, w.direct_hits, w.teleports, w.push_distance,
			w.environmental_deaths, w.environmental_kills,
			w.object_placed, w.object_upgraded, w.object_carried, w.object_dropped, w.object_removed, w.object_detonated,
			w.ammo_packs, w.health_packs, w.health_pack_healing
		FROM
			match_round_player_variants w
		LEFT JOIN
			match_round r ON r.round_id = w.round_id
		WHERE
			r.match_id = $1`
	rows, errRows := r.Database.Query(ctx, query, match.MatchID)
	if errRows != nil {
		return database.Err(errRows)
	}

	for rows.Next() {
		var mrws MatchVariantStatsRound
		if err := rows.Scan(
			&mrws.Variant, &mrws.RoundID, &mrws.SteamID, &mrws.Kills, &mrws.Assists, &mrws.Deaths, &mrws.PostroundKills, &mrws.PostroundAssists,
			&mrws.PostroundDeaths, &mrws.PreroundHealing, &mrws.Healing, &mrws.PostroundHealing, &mrws.Drops, &mrws.NearFullChargeDeath,
			&mrws.ChargesUber, &mrws.ChargesKritz, &mrws.ChargesVacc, &mrws.ChargesQuickfix,
			&mrws.Damage, &mrws.DamageTaken, &mrws.Dominations, &mrws.Dominated, &mrws.Revenges, &mrws.Revenged,
			&mrws.Airshots, &mrws.Headshots, &mrws.HeadshotKills, &mrws.Backstabs, &mrws.BackstabKills,
			&mrws.Captures, &mrws.CapturesBlocked,
			&mrws.WasHeadshot, &mrws.WasBackstabbed, &mrws.Shots, &mrws.Hits, &mrws.ObjectsBuilt, &mrws.ObjectsDestroyed,
			&mrws.Heals, &mrws.Healed, &mrws.CrossbowHeals, &mrws.CrossbowHealing, &mrws.HealOnHit, &mrws.Extinguishes, &mrws.BuildingHealing,
			&mrws.DroppedUbers, &mrws.Reflects, &mrws.Defenses, &mrws.DirectHits, &mrws.Teleports, &mrws.PushDistance,
			&mrws.EnvironmentalDeaths, &mrws.EnvironmentalKills,
			&mrws.ObjectPlaced, &mrws.ObjectUpgraded, &mrws.ObjectCarried, &mrws.ObjectDropped, &mrws.ObjectRemoved, &mrws.ObjectDetonated,
			&mrws.AmmoPacks, &mrws.HealthPacks, &mrws.HealthPackHealing,
		); err != nil {
			return database.Err(err)
		}

		match.Variants = append(match.Variants, mrws)
	}

	return nil
}

func (r Repository) MatchesWithPlayer(ctx context.Context, steamID steamid.SteamID) ([]PlayerMatchHistory, error) {
	const query = `
		SELECT DISTINCT
			m.match_id, m.server_id, m.map_id, mp.map_name, m.demo_id, s.stats_bucket_id,
			s.bucket_name, m.hostname, m.score_red, m.score_blu,
			m.duration_ms, m.created_on, m.start_time, srv.name, srv.short_name
		FROM match m
		LEFT JOIN match_round r ON m.match_id = r.match_id
		LEFT JOIN match_round_player p ON r.round_id = p.round_id
		LEFT JOIN map mp USING(map_id)
		LEFT JOIN stats_bucket s USING(stats_bucket_id)
		LEFT JOIN server srv ON m.server_id = srv.server_id
		WHERE p.steam_id = $1
		ORDER BY m.start_time DESC`
	var matches []PlayerMatchHistory

	rows, errRows := r.Database.Query(ctx, query, steamID.Int64())
	if errRows != nil {
		return nil, database.Err(errRows)
	}

	for rows.Next() {
		var match PlayerMatchHistory
		if err := rows.Scan(&match.MatchID, &match.ServerID, &match.MapID, &match.MapName, &match.DemoID,
			&match.BucketID, &match.BucketName, &match.Hostname, &match.ScoreRed, &match.ScoreBlu,
			&match.DurationMs, &match.CreatedOn, &match.StartTime, &match.ServerName, &match.ServerNameShort); err != nil {
			return nil, database.Err(err)
		}

		matches = append(matches, match)
	}

	return matches, nil
}

func (r Repository) getMatch(ctx context.Context, matchID uuid.UUID) (*Match, error) {
	const query = `
		SELECT
			m.match_id, m.server_id, m.map_id, m.demo_id, m.stats_bucket_id, m.hostname, m.score_red, m.score_blu,
			m.start_time, m.duration_ms, m.created_on, COALESCE(a.asset_id, '00000000-0000-0000-0000-000000000000'),
			COALESCE(mp.map_name, ''), COALESCE(srv.name, ''), COALESCE(srv.short_name, '')
		FROM match m
		LEFT JOIN demo d USING (demo_id)
		LEFT JOIN asset a ON a.asset_id = d.asset_id AND NOT a.deleted
		LEFT JOIN map mp ON mp.map_id = m.map_id
		LEFT JOIN server srv ON srv.server_id = m.server_id
		WHERE match_id = $1`

	var match Match
	if err := r.QueryRow(ctx, query, matchID).Scan(&match.MatchID, &match.ServerID, &match.MapID, &match.DemoID,
		&match.StatsBucketID, &match.Hostname, &match.ScoreRed, &match.ScoreBlu, &match.StartTime, &match.DurationMs,
		&match.CreatedOn, &match.AssetID, &match.MapName, &match.ServerName, &match.ServerNameShort); err != nil {
		return nil, database.Err(err)
	}

	return &match, nil
}

func (r Repository) getRounds(ctx context.Context, match *Match) error {
	const query = `
		SELECT
			round_id, winner, is_stalemate, is_sudden_death, duration_ms
		FROM
			match_round
		WHERE
			match_id = $1`
	rows, errRows := r.Database.Query(ctx, query, match.MatchID)
	if errRows != nil {
		return database.Err(errRows)
	}

	for rows.Next() {
		var round MatchRound
		if err := rows.Scan(&round.RoundID, &round.Winner, &round.IsStalemate, &round.IsSuddenDeath, &round.DurationMs); err != nil {
			return database.Err(err)
		}

		match.Rounds = append(match.Rounds, round)
	}

	return nil
}

type roundTickRange struct {
	roundID int64
	minTick int
	maxTick int
}

// roundTickRanges preserves the association between newly inserted rounds and
// their parsed player tick intervals. Only rounds with a positive duration are
// eligible, matching the match view's exclusion of zero-duration stub rounds.
func roundTickRanges(rounds []demoparse.RoundSummary, roundIDs []int64) []roundTickRange {
	ranges := make([]roundTickRange, 0, len(rounds))

	for idx, round := range rounds {
		if idx >= len(roundIDs) || round.Time <= 0 || len(round.Players) == 0 {
			continue
		}

		minTick := round.Players[0].TickStart
		maxTick := round.Players[0].TickEnd

		for _, player := range round.Players[1:] {
			minTick = min(minTick, player.TickStart)
			maxTick = max(maxTick, player.TickEnd)
		}

		ranges = append(ranges, roundTickRange{roundID: roundIDs[idx], minTick: minTick, maxTick: maxTick})
	}

	return ranges
}

// roundIDForTick returns the first parsed round containing tick. Kills outside
// all known round intervals retain a NULL round rather than a guessed round.
func roundIDForTick(ranges []roundTickRange, tick int) *int64 {
	for _, round := range ranges {
		if tick >= round.minTick && tick <= round.maxTick {
			roundID := round.roundID

			return &roundID
		}
	}

	return nil
}

func (r Repository) CreateMatch(ctx context.Context, serverID int32, demoID int32, demo *demoparse.Demo, timeStart time.Time, mapInfo maps.Map, statsBucketID *int32) (uuid.UUID, error) {
	newID, errID := uuid.NewV4()
	if errID != nil {
		return newID, fmt.Errorf("%w: failed to generate UUID", ErrInvalidState)
	}

	scores := demo.Scores()
	duration := time.Duration(demo.Duration) * time.Second

	transaction, errTx := r.Begin(ctx)
	if errTx != nil {
		return newID, database.Err(errTx)
	}

	if statsBucketID == nil {
		resolved, errBucket := r.bucketForServer(ctx, transaction, serverID)
		if errBucket != nil {
			if err := transaction.Rollback(ctx); err != nil {
				slog.Error("Failed to rollback tx", slog.String("error", err.Error()))
			}

			return newID, errBucket
		}
		statsBucketID = resolved
	}

	if _, errMatch := transaction.Exec(ctx, `
		INSERT INTO match (
			match_id, server_id, map_id, demo_id, stats_bucket_id, hostname,
			score_red, score_blu, start_time, duration_ms, created_on)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)`,
		newID, serverID, mapInfo.MapID, demoID, statsBucketID, demo.Filename, scores.Red,
		scores.Blu, timeStart, duration.Milliseconds(), time.Now()); errMatch != nil {
		if err := transaction.Rollback(ctx); err != nil {
			slog.Error("Failed to rollback tx", slog.String("error", err.Error()))
		}

		return newID, database.Err(errMatch)
	}
	playerTeams := playerTeamMap(demo)
	roundIDs := make([]int64, 0, len(demo.Rounds))

	for _, round := range demo.Rounds {
		roundID, err := r.insertRound(ctx, transaction, newID, playerTeams, round)
		if err != nil {
			if err := transaction.Rollback(ctx); err != nil {
				slog.Error("Failed to rollback tx", slog.String("error", err.Error()))
			}

			return newID, err
		}

		roundIDs = append(roundIDs, roundID)
	}

	if err := r.insertEvents(ctx, transaction, newID, demo.Events, roundTickRanges(demo.Rounds, roundIDs)); err != nil {
		if err := transaction.Rollback(ctx); err != nil {
			slog.Error("Failed to rollback tx", slog.String("error", err.Error()))
		}

		return newID, err
	}

	if err := transaction.Commit(ctx); err != nil {
		if err := transaction.Rollback(ctx); err != nil {
			slog.Error("Failed to rollback tx", slog.String("error", err.Error()))
		}

		return newID, database.Err(err)
	}

	return newID, nil
}

func (r Repository) insertRound(ctx context.Context, transaction pgx.Tx, matchID uuid.UUID, playerTeams map[string]string, round demoparse.RoundSummary) (int64, error) {
	const query = `
		INSERT INTO match_round (
			match_id, winner, is_stalemate, is_sudden_death, duration_ms
		) VALUES (
			$1, $2, $3, $4, $5
		)
		RETURNING round_id`
	duration := time.Duration(round.Time) * time.Second

	var roundID int64
	if errRound := transaction.
		QueryRow(ctx, query, matchID, toTfTeam(round.Winner), round.IsStalemate, round.IsSuddenDeath, duration.Milliseconds()).
		Scan(&roundID); errRound != nil {
		return 0, database.Err(errRound)
	}

	for _, player := range round.Players {
		steamID := steamid.New(player.SteamID)
		if !steamID.Valid() {
			continue
		}

		player.Team = playerTeams[player.SteamID]

		if err := r.insertRoundPlayer(ctx, transaction, roundID, steamID, round, player); err != nil {
			return 0, err
		}
	}

	return roundID, nil
}

func (r Repository) insertRoundPlayer(ctx context.Context, transaction pgx.Tx, roundID int64, steamID steamid.SteamID, round demoparse.RoundSummary, player demoparse.PlayerSummary) error {
	const query = `
		INSERT INTO match_round_player (
			round_id, steam_id, team, mvp, tick_start, tick_end, kills, assists, deaths, postround_kills,
			postround_assists, postround_deaths, preround_healing, healing, postround_healing, drops,
			near_full_charge_death, charges_uber, charges_kritz, charges_vacc, charges_quickfix, damage,
			damage_taken, dominations, dominated, revenges, revenged, airshots, headshots, headshot_kills,
			backstabs, backstab_kills, captures, captures_blocked, was_headshot, was_backstabbed,
			shots, hits, objects_built, objects_destroyed,
			points, connection_count, bonus_points, scoreboard_kills, scoreboard_assists, scoreboard_healing, scoreboard_deaths,
			scoreboard_damage, suicides, extinguishes, ignites,
			heals, healed, crossbow_heals, crossbow_healing, heal_on_hit, building_healing, dropped_ubers,
			reflects, defenses, direct_hits, teleports, push_distance,
			environmental_deaths, environmental_kills,
			object_placed, object_upgraded, object_carried, object_dropped, object_removed, object_detonated,
			ammo_packs, health_packs, health_pack_healing)
		VALUES(
			$1,  $2,  $3,  $4,  $5,  $6,  $7,  $8,  $9,  $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26, $27, $28, $29, $30,
			$31, $32, $33, $34, $35, $36, $37, $38, $39, $40,
			$41, $42, $43, $44, $45, $46, $47, $48, $49, $50,
			$51, $52, $53, $54, $55, $56, $57, $58, $59, $60,
			$61, $62, $63, $64, $65, $66, $67, $68, $69, $70,
			$71, $72, $73, $74
		)`

	isMvp := slices.Contains(round.Mvps, string(steamID.Steam3()))

	if _, err := transaction.Exec(ctx, query,
		roundID, steamID.Int64(), toTfTeam(player.Team), isMvp, player.TickStart, player.TickEnd,
		player.Kills, player.Assists, player.Deaths, player.PostroundKills, player.PostroundAssists, player.PostroundDeaths,
		player.PreroundHealing, player.Healing, player.PostroundHealing, player.Drops, player.NearFullChargeDeath,
		player.ChargesUber, player.ChargesKritz, player.ChargesVacc, player.ChargesQuickfix, player.Damage,
		player.DamageTaken, player.Dominations, player.Dominated, player.Revenges, player.Revenged,
		player.Airshots, player.Headshots, player.HeadshotKills, player.Backstabs, player.BackstabKills,
		player.Captures, player.CapturesBlocked, player.WasHeadshot, player.WasBackstabbed, player.Shots,
		player.Hits, player.ObjectBuilt, player.ObjectDestroyed, player.Points, player.ConnectionCount,
		player.BonusPoints, player.ScoreboardKills, player.ScoreboardAssists, player.ScoreboardHealing,
		player.ScoreboardDeaths, player.ScoreboardDamage, player.Suicides, player.Extinguishes,
		player.Ignites, player.Heals, player.Healed, player.CrossbowHeals, player.CrossbowHealing,
		player.HealOnHit, player.BuildingHealing, player.DroppedUbers, player.Reflects, player.Defenses,
		player.DirectHits, player.Teleports, player.PushDistance, player.EnvironmentalDeaths, player.EnvironmentalKills,
		player.ObjectPlaced, player.ObjectUpgraded, player.ObjectCarried, player.ObjectDropped, player.ObjectRemoved,
		player.ObjectDetonated, player.AmmoPacks, player.HealthPacks, player.HealthPackHealing); err != nil {
		return database.Err(err)
	}

	for weapon, weaponStats := range player.Weapons {
		if err := r.insertRoundPlayerVariants(ctx, transaction, steamID, roundID, weaponStats, weapon); err != nil {
			return database.Err(err)
		}
	}

	for class, weaponStats := range player.Classes {
		if err := r.insertRoundPlayerVariants(ctx, transaction, steamID, roundID, weaponStats, class); err != nil {
			return database.Err(err)
		}
	}

	return nil
}

func (r Repository) insertRoundPlayerVariants(ctx context.Context, transaction pgx.Tx, steamID steamid.SteamID, roundID int64, stats demoparse.Stats, variantKey string) error {
	const query = `
		INSERT INTO match_round_player_variants (
			variant, round_id, steam_id, kills, assists, deaths, postround_kills,
			postround_assists, postround_deaths, preround_healing, healing, postround_healing, drops,
			near_full_charge_death, charges_uber, charges_kritz, charges_vacc, charges_quickfix, damage,
			damage_taken, dominations, dominated, revenges, revenged, airshots, headshots, headshot_kills,
			backstabs, backstab_kills, captures, captures_blocked, was_headshot, was_backstabbed,
			shots, hits, objects_built, objects_destroyed,
			heals, healed, crossbow_heals, crossbow_healing, heal_on_hit, extinguishes, building_healing,
			dropped_ubers, reflects, defenses, direct_hits, teleports, push_distance,
			environmental_deaths, environmental_kills,
			object_placed, object_upgraded, object_carried, object_dropped, object_removed, object_detonated,
			ammo_packs, health_packs, health_pack_healing
		) VALUES (
			LOWER($1),$2,  $3,  $4,  $5,  $6,  $7,  $8,  $9,  $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31,
			$32, $33, $34, $35, $36, $37, $38, $39, $40, $41,
			$42, $43, $44, $45, $46, $47, $48, $49, $50, $51,
			$52, $53, $54, $55, $56, $57, $58, $59, $60, $61
		)`

	if _, err := transaction.Exec(ctx, query,
		variantKey, roundID, steamID.Int64(), stats.Kills, stats.Assists, stats.Deaths, stats.PostroundKills,
		stats.PostroundAssists, stats.PostroundDeaths, stats.PreroundHealing, stats.Healing, stats.PostroundHealing, stats.Drops,
		stats.NearFullChargeDeath, stats.ChargesUber, stats.ChargesKritz, stats.ChargesVacc, stats.ChargesQuickfix, stats.Damage,
		stats.DamageTaken, stats.Dominations, stats.Dominated, stats.Revenges, stats.Revenged, stats.Airshots, stats.Headshots, stats.HeadshotKills,
		stats.Backstabs, stats.BackstabKills, stats.Captures, stats.CapturesBlocked, stats.WasHeadshot, stats.WasBackstabbed,
		stats.Shots, stats.Hits, stats.ObjectBuilt, stats.ObjectDestroyed,
		stats.Heals, stats.Healed, stats.CrossbowHeals, stats.CrossbowHealing, stats.HealOnHit, stats.Extinguishes, stats.BuildingHealing,
		stats.DroppedUbers, stats.Reflects, stats.Defenses, stats.DirectHits, stats.Teleports, stats.PushDistance,
		stats.EnvironmentalDeaths, stats.EnvironmentalKills,
		stats.ObjectPlaced, stats.ObjectUpgraded, stats.ObjectCarried, stats.ObjectDropped, stats.ObjectRemoved, stats.ObjectDetonated,
		stats.AmmoPacks, stats.HealthPacks, stats.HealthPackHealing); err != nil {
		return database.Err(err)
	}

	return nil
}

// ErrUnknownMatchEventType is returned when a parsed match event carries a
// type not recognised by this version of gbans (e.g. a newer parser emits
// additional kinds). Callers should skip such events rather than fail the
// whole import so manual demo uploads don't surface internal errors.
var ErrUnknownMatchEventType = errors.New("unknown match event type")

// matchEventRow is the storable projection of a demoparse.MatchEvent. Actor,
// target, and assister carry the singular participant roles; multi-party
// roles such as capture cappers live in details.
type matchEventRow struct {
	eventType string
	actor     *int64
	target    *int64
	assister  *int64
	weapon    *string
	building  *string
	killerPos *demoparse.Position
	victimPos *demoparse.Position
	eventPos  *demoparse.Position
	details   []byte
}

func eventSteamID(raw string) *int64 {
	sid := steamid.New(raw)
	if !sid.Valid() {
		return nil
	}

	id := sid.Int64()

	return &id
}

func optionalEventString(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}

func optionalEventBuilding(value string) *string {
	switch value {
	case "sentry", "dispenser", "teleporter", "sapper":
		return &value
	default:
		return nil
	}
}

// pointWKT renders a position as WKT for geometry column binding. A nil
// position binds NULL; otherwise Postgres coerces the text to PointZ.
func pointWKT(pos *demoparse.Position) *string {
	if pos == nil {
		return nil
	}

	out := fmt.Sprintf("POINT Z (%v %v %v)", pos.X, pos.Y, pos.Z)

	return &out
}

// matchEventRowFromEvent projects a parsed match event onto its storable row.
// Variant-specific payloads are preserved verbatim in details; positions use
// the geometry columns.
func matchEventRowFromEvent(event demoparse.MatchEvent) (matchEventRow, error) { //nolint:cyclop,maintidx
	row := matchEventRow{eventType: string(event.Type)}

	payload, err := eventRowPayload(event, &row)
	if err != nil {
		return row, err
	}

	details, err := marshalEventPayload(payload)
	if err != nil {
		return row, err
	}

	row.details = details

	return row, nil
}

func marshalEventPayload(payload any) ([]byte, error) {
	if payload == nil {
		return []byte("{}"), nil
	}

	details, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal match event payload: %w", err)
	}

	return details, nil
}

// eventRowPayload selects the storable payload for an event and records its
// indexed columns on row. Unknown types are an error.
func eventRowPayload(event demoparse.MatchEvent, row *matchEventRow) (any, error) {
	switch event.Type {
	case demoparse.MatchEventKill:
		return killEventPayload(event, row), nil
	case demoparse.MatchEventCaptureStarted,
		demoparse.MatchEventCapture,
		demoparse.MatchEventCaptureBlocked,
		demoparse.MatchEventCaptureBroken:
		return captureEventPayload(event, row), nil
	case demoparse.MatchEventBuildingBuilt,
		demoparse.MatchEventBuildingDestroyed,
		demoparse.MatchEventBuildingUpgraded,
		demoparse.MatchEventBuildingCarried,
		demoparse.MatchEventBuildingDropped,
		demoparse.MatchEventBuildingRemoved,
		demoparse.MatchEventBuildingDetonated,
		demoparse.MatchEventSapperPlaced:
		return buildingEventPayload(event, row), nil
	case demoparse.MatchEventRoundStarted,
		demoparse.MatchEventRoundWon,
		demoparse.MatchEventStalemate,
		demoparse.MatchEventGameOver,
		demoparse.MatchEventSuddenDeathBegin,
		demoparse.MatchEventSuddenDeathEnd,
		demoparse.MatchEventOvertimeBegin,
		demoparse.MatchEventOvertimeEnd,
		demoparse.MatchEventSetupFinished:
		return roundEventPayload(event), nil
	case demoparse.MatchEventUberDropped,
		demoparse.MatchEventUberDeployed,
		demoparse.MatchEventFlagEvent,
		demoparse.MatchEventFlagCaptured,
		demoparse.MatchEventKillstreakEnded:
		return uberFlagEventPayload(event, row), nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownMatchEventType, event.Type)
	}
}

func killEventPayload(event demoparse.MatchEvent, row *matchEventRow) *demoparse.KillEvent {
	kill := event.Kill
	if kill == nil {
		kill = &demoparse.KillEvent{}
	}
	row.actor = eventSteamID(kill.Killer)
	row.target = eventSteamID(kill.Victim)
	row.weapon = optionalEventString(kill.Weapon)
	row.killerPos = kill.KillerPos
	row.victimPos = kill.VictimPos

	return kill
}

func captureEventPayload(event demoparse.MatchEvent, row *matchEventRow) any {
	switch event.Type {
	case demoparse.MatchEventCaptureStarted:
		started := event.CaptureStarted
		if started == nil {
			started = &demoparse.CaptureStartedEvent{}
		}

		return started
	case demoparse.MatchEventCapture:
		capture := event.Capture
		if capture == nil {
			capture = &demoparse.CaptureEvent{}
		}

		return capture
	case demoparse.MatchEventCaptureBlocked:
		blocked := event.CaptureBlocked
		if blocked == nil {
			blocked = &demoparse.CaptureBlockedEvent{}
		}
		row.actor = eventSteamID(blocked.Blocker)
		row.target = eventSteamID(blocked.Victim)

		return blocked
	default:
		broken := event.CaptureBroken
		if broken == nil {
			broken = &demoparse.CaptureBrokenEvent{}
		}

		return broken
	}
}

func buildingEventPayload(event demoparse.MatchEvent, row *matchEventRow) any {
	switch event.Type {
	case demoparse.MatchEventBuildingBuilt:
		built := event.BuildingBuilt
		if built == nil {
			built = &demoparse.BuildingBuiltEvent{}
		}
		row.actor = eventSteamID(built.Owner)
		row.building = optionalEventBuilding(built.Building)
		row.eventPos = &built.Pos

		return built
	case demoparse.MatchEventBuildingDestroyed:
		destroyed := event.BuildingDestroyed
		if destroyed == nil {
			destroyed = &demoparse.BuildingDestroyedEvent{}
		}
		row.actor = eventSteamID(destroyed.Attacker)
		row.target = eventSteamID(destroyed.Owner)
		row.assister = eventSteamID(destroyed.Assister)
		row.weapon = optionalEventString(destroyed.Weapon)
		row.building = optionalEventBuilding(destroyed.Building)
		row.eventPos = destroyed.Pos

		return destroyed
	case demoparse.MatchEventSapperPlaced:
		sapper := event.SapperPlaced
		if sapper == nil {
			sapper = &demoparse.SapperPlacedEvent{}
		}
		row.actor = eventSteamID(sapper.Spy)
		row.target = eventSteamID(sapper.Owner)
		row.building = optionalEventBuilding(sapper.Building)

		return sapper
	default:
		return buildingLifecyclePayload(event, row)
	}
}

func buildingLifecyclePayload(event demoparse.MatchEvent, row *matchEventRow) *demoparse.BuildingLifecycleEvent {
	var lifecycle *demoparse.BuildingLifecycleEvent

	switch event.Type {
	case demoparse.MatchEventBuildingUpgraded:
		lifecycle = event.BuildingUpgraded
	case demoparse.MatchEventBuildingCarried:
		lifecycle = event.BuildingCarried
	case demoparse.MatchEventBuildingDropped:
		lifecycle = event.BuildingDropped
	case demoparse.MatchEventBuildingRemoved:
		lifecycle = event.BuildingRemoved
	default:
		lifecycle = event.BuildingDetonated
	}

	return lifecycleOrEmpty(lifecycle, row)
}

func roundEventPayload(event demoparse.MatchEvent) any {
	switch event.Type {
	case demoparse.MatchEventRoundStarted:
		if event.RoundStarted == nil {
			return &demoparse.RoundStartedEvent{}
		}

		return event.RoundStarted
	case demoparse.MatchEventRoundWon:
		if event.RoundWon == nil {
			return &demoparse.RoundWonEvent{}
		}

		return event.RoundWon
	case demoparse.MatchEventStalemate:
		if event.Stalemate == nil {
			return &demoparse.StalemateEvent{}
		}

		return event.Stalemate
	case demoparse.MatchEventGameOver:
		if event.GameOver == nil {
			return &demoparse.GameOverEvent{}
		}

		return event.GameOver
	default:
		return nil
	}
}

func uberFlagEventPayload(event demoparse.MatchEvent, row *matchEventRow) any {
	switch event.Type {
	case demoparse.MatchEventUberDropped:
		dropped := event.UberDropped
		if dropped == nil {
			dropped = &demoparse.UberDroppedEvent{}
		}
		row.actor = eventSteamID(dropped.Attacker)
		row.target = eventSteamID(dropped.Medic)

		return dropped
	case demoparse.MatchEventUberDeployed:
		deployed := event.UberDeployed
		if deployed == nil {
			deployed = &demoparse.UberDeployedEvent{}
		}
		row.actor = eventSteamID(deployed.Medic)
		row.target = eventSteamID(deployed.Target)

		return deployed
	case demoparse.MatchEventFlagEvent:
		flag := event.FlagEvent
		if flag == nil {
			flag = &demoparse.FlagEvent{}
		}
		row.actor = eventSteamID(flag.Player)
		row.target = eventSteamID(flag.Carrier)

		return flag
	case demoparse.MatchEventFlagCaptured:
		if event.FlagCaptured == nil {
			return &demoparse.FlagCapturedEvent{}
		}

		return event.FlagCaptured
	default:
		streak := event.KillstreakEnded
		if streak == nil {
			streak = &demoparse.KillstreakEndedEvent{}
		}
		row.actor = eventSteamID(streak.Killer)
		row.target = eventSteamID(streak.Player)

		return streak
	}
}

func lifecycleOrEmpty(lifecycle *demoparse.BuildingLifecycleEvent, row *matchEventRow) *demoparse.BuildingLifecycleEvent {
	if lifecycle == nil {
		lifecycle = &demoparse.BuildingLifecycleEvent{}
	}

	row.actor = eventSteamID(lifecycle.Player)
	row.building = optionalEventBuilding(lifecycle.Building)

	return lifecycle
}

func (r Repository) insertEvents(ctx context.Context, transaction pgx.Tx, matchID uuid.UUID, events []demoparse.MatchEvent, rounds []roundTickRange) error {
	const query = `
		INSERT INTO match_event (
			match_id, round_id, tick, event_type,
			actor_steam_id, target_steam_id, assister_steam_id,
			weapon, building,
			killer_position, victim_position, event_position,
			details
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)`

	for _, event := range events {
		row, err := matchEventRowFromEvent(event)
		if err != nil {
			if errors.Is(err, ErrUnknownMatchEventType) {
				slog.Warn("Skipping unknown match event type",
					slog.String("event_type", string(event.Type)),
					slog.Int("tick", event.Tick))

				continue
			}

			return err
		}

		if _, err := transaction.Exec(ctx, query,
			matchID, roundIDForTick(rounds, event.Tick), event.Tick, row.eventType,
			row.actor, row.target, row.assister,
			row.weapon, row.building,
			pointWKT(row.killerPos), pointWKT(row.victimPos), pointWKT(row.eventPos),
			row.details); err != nil {
			return database.Err(err)
		}
	}

	return nil
}

func (r Repository) getKills(ctx context.Context, match *Match) error {
	const query = `
		SELECT
			match_event_id, tick, actor_steam_id, target_steam_id, weapon,
			ST_X(killer_position), ST_Y(killer_position), ST_Z(killer_position),
			ST_X(victim_position), ST_Y(victim_position), ST_Z(victim_position),
			details
		FROM
			match_event
		WHERE
			match_id = $1 AND event_type = 'kill'
		ORDER BY
			tick ASC, match_event_id ASC`
	rows, errRows := r.Database.Query(ctx, query, match.MatchID)
	if errRows != nil {
		return database.Err(errRows)
	}

	for rows.Next() {
		var (
			kill     MatchKill
			killerID *int64
			victimID *int64
			killerX  *float64
			killerY  *float64
			killerZ  *float64
			victimX  *float64
			victimY  *float64
			victimZ  *float64
			details  []byte
			weapon   *string
		)

		if err := rows.Scan(&kill.MatchKillID, &kill.Tick, &killerID, &victimID, &weapon,
			&killerX, &killerY, &killerZ,
			&victimX, &victimY, &victimZ,
			&details); err != nil {
			return database.Err(err)
		}

		// The kill feed API requires a victim; events without a resolvable
		// victim remain stored but are not projected.
		if victimID == nil {
			slog.Debug("Skipping kill event without victim steamid", slog.Int("tick", kill.Tick))

			continue
		}

		if killerID != nil {
			kill.KillerSteamID = steamid.New(*killerID)
			kill.HasKiller = true
		}

		kill.VictimSteamID = steamid.New(*victimID)

		if weapon != nil {
			kill.Weapon = *weapon
		}

		if killerX != nil && killerY != nil && killerZ != nil {
			kill.KillerPos = &demoparse.Position{X: *killerX, Y: *killerY, Z: *killerZ}
		}

		if victimX != nil && victimY != nil && victimZ != nil {
			kill.VictimPos = &demoparse.Position{X: *victimX, Y: *victimY, Z: *victimZ}
		}

		var payload demoparse.KillEvent
		if err := json.Unmarshal(details, &payload); err != nil {
			return database.Err(err)
		}

		kill.KillerAngles = payload.KillerAngles
		kill.VictimAngles = payload.VictimAngles

		match.Kills = append(match.Kills, kill)
	}

	return nil
}

func (r Repository) getEvents(ctx context.Context, match *Match) error {
	const query = `
		SELECT
			match_event_id, tick, event_type, actor_steam_id, target_steam_id,
			weapon, building, details
		FROM
			match_event
		WHERE
			match_id = $1
		ORDER BY
			tick ASC, match_event_id ASC`
	rows, errRows := r.Database.Query(ctx, query, match.MatchID)
	if errRows != nil {
		return database.Err(errRows)
	}

	for rows.Next() {
		var (
			event   MatchEvent
			details []byte
		)
		if err := rows.Scan(&event.MatchEventID, &event.Tick, &event.Type,
			&event.ActorSteamID, &event.TargetSteamID,
			&event.Weapon, &event.Building, &details); err != nil {
			return database.Err(err)
		}

		event.Details = string(details)
		match.Events = append(match.Events, event)
	}

	return nil
}

func (r Repository) GetBucket(ctx context.Context, statsBucketID int32) (*Bucket, error) {
	const query = "SELECT stats_bucket_id, bucket_name, is_enabled, is_default FROM stats_bucket WHERE stats_bucket_id = $1"
	var bucket Bucket
	if err := r.QueryRow(ctx, query, statsBucketID).Scan(&bucket.BucketID, &bucket.BucketName, &bucket.IsEnabled, &bucket.IsDefault); err != nil {
		return nil, database.Err(err)
	}

	return &bucket, nil
}

// defaultBucketID returns the id of the bucket marked as default, or
// database.ErrNoResult when no bucket is marked as such.
func (r Repository) defaultBucketID(ctx context.Context, transaction pgx.Tx) (int32, error) {
	var bucketID int32
	if err := transaction.QueryRow(ctx, `SELECT stats_bucket_id FROM stats_bucket WHERE is_default LIMIT 1`).Scan(&bucketID); err != nil {
		return 0, database.Err(err)
	}

	return bucketID, nil
}

// bucketForServer resolves the stats bucket a new match should be filed
// under: the owning server's configured bucket, falling back to the default
// bucket (casual) when the server has none set. Previously every match was
// filed under bucket 1 regardless of server configuration.
func (r Repository) bucketForServer(ctx context.Context, transaction pgx.Tx, serverID int32) (*int32, error) {
	var bucketID pgtype.Int4
	if err := transaction.QueryRow(ctx, `SELECT stats_bucket_id FROM server WHERE server_id = $1`, serverID).Scan(&bucketID); err != nil {
		return nil, database.Err(err)
	}

	if bucketID.Valid {
		id := bucketID.Int32

		return &id, nil
	}

	defaultID, errDefault := r.defaultBucketID(ctx, transaction)
	if errDefault != nil {
		return nil, errDefault
	}

	return &defaultID, nil
}

func (r Repository) CreateBucket(ctx context.Context, bucketName string) (*Bucket, error) {
	const query = "INSERT INTO stats_bucket (bucket_name) VALUES ($1) RETURNING stats_bucket_id"
	bucket := Bucket{BucketName: bucketName}
	if err := r.QueryRow(ctx, query, bucketName).Scan(&bucket.BucketID); err != nil {
		return nil, database.Err(err)
	}

	return &bucket, nil
}

func (r Repository) SaveBucket(ctx context.Context, bucket Bucket) error {
	const query = "UPDATE stats_bucket SET bucket_name = $1 WHERE stats_bucket_id = $2"
	if err := r.Exec(ctx, query, bucket.BucketName, bucket.BucketID); err != nil {
		return database.Err(err)
	}

	return nil
}

func playerTeamMap(demo *demoparse.Demo) map[string]string {
	// Find the last team the player played on and use that as their final team.
	playerTeams := map[string]string{}
	for _, steamID := range demo.SteamIDs() {
		steam3 := string(steamID.Steam3())
		found := false
		for roundIdx := range slices.Backward(demo.Rounds) {
			for _, player := range demo.Rounds[roundIdx].Players {
				if player.SteamID == steam3 {
					if demo.Rounds[roundIdx].Winner == "" {
						continue
					}
					redWinner := toTfTeam(demo.Rounds[roundIdx].Winner) == "red"
					if redWinner {
						if slices.Contains(demo.Rounds[roundIdx].Winners, steam3) {
							playerTeams[steam3] = "red"
						} else {
							playerTeams[steam3] = "blu"
						}
					} else {
						if slices.Contains(demo.Rounds[roundIdx].Winners, steam3) {
							playerTeams[steam3] = "blu"
						} else {
							playerTeams[steam3] = "red"
						}
					}

					found = true

					break
				}
			}
			if found {
				break
			}
		}
	}

	return playerTeams
}

func toTfTeam(team string) string {
	switch strings.ToLower(team) {
	case "blue":
		fallthrough
	case "blu":
		return "blu"
	case "red":
		return "red"
	case "spec":
		return "spec"
	case "unassigned":
		fallthrough
	default:
		return "unassigned"
	}
}

func toTFClass(class string) string {
	switch strings.ToLower(class) {
	case "scout":
		return "scout"
	case "soldier":
		return "soldier"
	case "pyro":
		return "pyro"
	case "demoman":
		fallthrough
	case "demo":
		return "demo"
	case "Heavyweapons":
		fallthrough
	case "heavy":
		return "heavy"
	case "engy":
		fallthrough
	case "engineer":
		return "engineer"
	case "medic":
		return "medic"
	case "sniper":
		return "sniper"
	case "spy":
		return "spy"
	default:
		return ""
	}
}
