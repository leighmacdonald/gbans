package stats

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
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
		return "weeky"
	case TimeBucketMonthly:
		return "monthly"
	case TimeBucketYearly:
		return "yearly"
	default:
		return "alltime"
	}
}

type Opts struct {
	query.Filter

	Variant    Variant
	VariantKey string
	TimeBucket TimeBucket
	TimeStamp  time.Time
}

func (o Opts) view() string {
	return fmt.Sprintf("stats_summary_%s_%s_view", o.TimeBucket, o.Variant)
}

type Repository struct{ database.Database }

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
	const query = `SELECT stats_bucket_id, bucket_name, is_enabled FROM stats_bucket`

	var buckets []Bucket
	rows, errRows := r.Database.Query(ctx, query)
	if errRows != nil {
		return nil, database.Err(errRows)
	}

	for rows.Next() {
		var bucket Bucket
		if err := rows.Scan(&bucket.BucketID, &bucket.BucketName, &bucket.IsEnabled); err != nil {
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

func (r Repository) loadOverallView(ctx context.Context, statsBucketID uint32, opts Opts) ([]any, uint64, error) {
	constraints := sq.And{
		sq.Eq{"stats_bucket_id": statsBucketID},
		// sq.Expr("date_trunc('day', ?::date) = date_bucket", opts.TimeStamp),
	}

	builder := r.Builder().
		Select("rank", "steam_id", "points", "connection_count", "bonus_points", "kills", "assists", "deaths",
			"postround_kills", "postround_assists", "preround_healing", "healing", "drops", "near_full_charge_death",
			"charges_uber", "charges_kritz", "charges_vacc", "charges_quickfix",
			"damage", "damage_taken", "dominations", "dominated", "revenges", "revenged",
			"airshots", "headshots", "headshot_kills", "backstabs", "backstab_kills",
			"was_headshot", "was_backstabbed", "shots", "hits", "objects_built", "objects_destroyed",
			"scoreboard_kills", "scoreboard_assists", "scoreboard_deaths", "suicides", "postround_deaths",
			"captures", "captures_blocked", "scoreboard_damage", "extinguishes", "ignites").
		From(opts.view()).
		Where(constraints)
	builder = builder.OrderBy("rank ASC")
	builder = opts.ApplyLimitOffset(builder, 10000)

	rows, errRows := r.QueryBuilder(ctx, builder)
	if errRows != nil {
		return nil, 0, database.Err(errRows)
	}

	var variantStats []any
	for rows.Next() {
		var overallStat OverallStats
		if err := rows.Scan(
			&overallStat.Rank, &overallStat.SteamID, &overallStat.Points, &overallStat.ConnectionCount, &overallStat.BonusPoints, &overallStat.Kills, &overallStat.Assists, &overallStat.Deaths,
			&overallStat.PostroundKills, &overallStat.PostroundAssists, &overallStat.PreroundHealing, &overallStat.Healing, &overallStat.Drops, &overallStat.NearFullChargeDeath,
			&overallStat.ChargesUber, &overallStat.ChargesKritz, &overallStat.ChargesVacc, &overallStat.ChargesQuickfix,
			&overallStat.Damage, &overallStat.DamageTaken, &overallStat.Dominations, &overallStat.Dominated, &overallStat.Revenges, &overallStat.Revenged,
			&overallStat.Airshots, &overallStat.Headshots, &overallStat.HeadshotKills, &overallStat.Backstabs, &overallStat.BackstabKills,
			&overallStat.WasHeadshot, &overallStat.WasBackstabbed, &overallStat.Shots, &overallStat.Hits, &overallStat.ObjectsBuilt, &overallStat.ObjectsDestroyed,
			&overallStat.ScoreboardKills, &overallStat.ScoreboardAssists, &overallStat.ScoreboardDeaths, &overallStat.Suicides, &overallStat.PostroundDeaths,
			&overallStat.Captures, &overallStat.CapturesBlocked, &overallStat.ScoreboardDamage, &overallStat.Extinguishes, &overallStat.Ignites); err != nil {
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
		// sq.Expr("date_trunc('day', ?::date) = date_bucket", opts.TimeStamp),
	}

	builder := r.Builder().
		Select("rank", "steam_id", "kills", "assists", "deaths", "postround_kills",
			"postround_assists", "preround_healing", "healing", "drops", "near_full_charge_death",
			"charges_uber", "charges_kritz", "charges_vacc", "charges_quickfix",
			"damage", "damage_taken", "dominations", "dominated", "revenges", "revenged",
			"airshots", "headshots", "headshot_kills", "backstabs", "backstab_kills",
			"was_headshot", "was_backstabbed", "shots", "hits", "objects_built", "objects_destroyed",
			"postround_deaths", "preround_healing", "postround_healing",
		).
		From(opts.view()).
		Where(constraints)
	builder = builder.OrderBy("rank ASC")
	builder = opts.ApplyLimitOffset(builder, 10000)

	rows, errRows := r.QueryBuilder(ctx, builder)
	if errRows != nil {
		return nil, 0, database.Err(errRows)
	}

	var variantStats []any
	for rows.Next() {
		var variantStat VariantStats
		if err := rows.Scan(
			&variantStat.Rank, &variantStat.SteamID, &variantStat.Kills, &variantStat.Assists, &variantStat.Deaths, &variantStat.PostroundKills,
			&variantStat.PostroundAssists, &variantStat.PreroundHealing, &variantStat.Healing, &variantStat.Drops, &variantStat.NearFullChargeDeath,
			&variantStat.ChargesUber, &variantStat.ChargesKritz, &variantStat.ChargesVacc, &variantStat.ChargesQuickfix,
			&variantStat.Damage, &variantStat.DamageTaken, &variantStat.Dominations, &variantStat.Dominated, &variantStat.Revenges, &variantStat.Revenged,
			&variantStat.Airshots, &variantStat.Headshots, &variantStat.HeadshotKills, &variantStat.Backstabs, &variantStat.BackstabKills,
			&variantStat.WasHeadshot, &variantStat.WasBackstabbed, &variantStat.Shots, &variantStat.Hits, &variantStat.ObjectsBuilt, &variantStat.ObjectsDestroyed,
			&variantStat.PostroundDeaths, &variantStat.PreroundHealing, &variantStat.PostroundHealing); err != nil {
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

func (r Repository) Query(ctx context.Context, statsBucketID uint32, opts Opts) ([]any, uint64, error) {
	switch opts.Variant {
	case VariantClasses:
		fallthrough
	case VariantWeapons:
		return r.loadVariantView(ctx, statsBucketID, opts)
	default:
		return r.loadOverallView(ctx, statsBucketID, opts)
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

	return match, nil
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
			p.suicides, p.extinguishes, p.ignites
		FROM
			match_round_player p
		LEFT JOIN
			match_round r ON r.round_id = p.round_id
		WHERE
			r.match_id = $1`
	rows, errRows := r.Database.Query(ctx, query, match.MatchID)
	if errRows != nil {
		return database.Err(errRows)
	}

	for rows.Next() {
		var mrp OverallStatsRound
		if err := rows.Scan(&mrp.RoundID, &mrp.SteamID, &mrp.Team, &mrp.MVP, &mrp.TickStart, &mrp.TickEnd, &mrp.Kills, &mrp.Assists,
			&mrp.PostroundKills, &mrp.PostroundAssists, &mrp.PostroundDeaths, &mrp.PreroundHealing, &mrp.Healing,
			&mrp.PostroundHealing, &mrp.Drops, &mrp.NearFullChargeDeath, &mrp.ChargesUber, &mrp.ChargesKritz, &mrp.ChargesVacc, &mrp.ChargesQuickfix,
			&mrp.Damage, &mrp.DamageTaken, &mrp.Dominations, &mrp.Dominated, &mrp.Revenges, &mrp.Revenged,
			&mrp.Airshots, &mrp.Headshots, &mrp.HeadshotKills,
			&mrp.Backstabs, &mrp.BackstabKills, &mrp.Captures, &mrp.CapturesBlocked, &mrp.WasHeadshot, &mrp.WasBackstabbed, &mrp.Shots, &mrp.Hits,
			&mrp.ObjectsBuilt, &mrp.ObjectsDestroyed, &mrp.Points, &mrp.ConnectionCount, &mrp.BonusPoints,
			&mrp.ScoreboardKills, &mrp.ScoreboardAssists, &mrp.ScoreboardHealing, &mrp.ScoreboardDeaths, &mrp.ScoreboardDamage,
			&mrp.Suicides, &mrp.Extinguishes, &mrp.Ignites,
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
			w.shots, w.hits, w.objects_built, w.objects_destroyed
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
		var mrws VariantStatsRound
		if err := rows.Scan(
			&mrws.Variant, &mrws.RoundID, &mrws.SteamID, &mrws.Kills, &mrws.Assists, &mrws.Deaths, &mrws.PostroundKills, &mrws.PostroundAssists,
			&mrws.PostroundDeaths, &mrws.PreroundHealing, &mrws.Healing, &mrws.PostroundHealing, &mrws.Drops, &mrws.NearFullChargeDeath,
			&mrws.ChargesUber, &mrws.ChargesKritz, &mrws.ChargesVacc, &mrws.ChargesQuickfix,
			&mrws.Damage, &mrws.DamageTaken, &mrws.Dominations, &mrws.Dominated, &mrws.Revenges, &mrws.Revenged,
			&mrws.Airshots, &mrws.Headshots, &mrws.HeadshotKills, &mrws.Backstabs, &mrws.BackstabKills,
			&mrws.Captures, &mrws.CapturesBlocked,
			&mrws.WasHeadshot, &mrws.WasBackstabbed, &mrws.Shots, &mrws.Hits, &mrws.ObjectsBuilt, &mrws.ObjectsDestroyed,
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
		m.duration_ms, m.created_on, srv.name, srv.short_name
		FROM match m
		LEFT JOIN match_round r ON m.match_id = r.match_id
		LEFT JOIN match_round_player p ON r.round_id = p.round_id
		LEFT JOIN map mp USING(map_id)
		LEFT JOIN stats_bucket s USING(stats_bucket_id)
		LEFT JOIN server srv ON m.server_id = srv.server_id
		WHERE p.steam_id = $1`
	var matches []PlayerMatchHistory

	rows, errRows := r.Database.Query(ctx, query, steamID.Int64())
	if errRows != nil {
		return nil, database.Err(errRows)
	}

	for rows.Next() {
		var match PlayerMatchHistory
		if err := rows.Scan(&match.MatchID, &match.ServerID, &match.MapID, &match.MapName, &match.DemoID,
			&match.BucketID, &match.BucketName, &match.Hostname, &match.ScoreRed, &match.ScoreBlu,
			&match.DurationMs, &match.CreatedOn, &match.ServerName, &match.ServerNameShort); err != nil {
			return nil, database.Err(err)
		}

		matches = append(matches, match)
	}

	return matches, nil
}

func (r Repository) getMatch(ctx context.Context, matchID uuid.UUID) (*Match, error) {
	const query = `
		SELECT
			match_id, server_id, map_id, demo_id, stats_bucket_id, hostname, score_red, score_blu,
			start_time, duration_ms, created_on
		FROM
			match
		WHERE
			match_id = $1`

	var match Match
	if err := r.QueryRow(ctx, query, matchID).Scan(&match.MatchID, &match.ServerID, &match.MapID, &match.DemoID,
		&match.StatsBucketID, &match.Hostname, &match.ScoreRed, &match.ScoreBlu, &match.StartTime, &match.DurationMs,
		&match.CreatedOn); err != nil {
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

		if round.Winner != "" {
			switch round.Winner {
			case "red":
				match.ScoreRed++
			case "blu":
				match.ScoreBlu++
			}
		}

		match.Rounds = append(match.Rounds, round)
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
		bucket, errBucket := r.GetBucket(ctx, 1)
		if errBucket != nil {
			return newID, errBucket
		}
		statsBucketID = &bucket.BucketID
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
	for _, round := range demo.Rounds {
		if err := r.insertRound(ctx, transaction, newID, playerTeams, round); err != nil {
			if err := transaction.Rollback(ctx); err != nil {
				slog.Error("Failed to rollback tx", slog.String("error", err.Error()))
			}

			return newID, err
		}
	}

	if err := transaction.Commit(ctx); err != nil {
		if err := transaction.Rollback(ctx); err != nil {
			slog.Error("Failed to rollback tx", slog.String("error", err.Error()))
		}

		return newID, database.Err(err)
	}

	return newID, nil
}

func (r Repository) insertRound(ctx context.Context, transaction pgx.Tx, matchID uuid.UUID, playerTeams map[string]string, round demoparse.RoundSummary) error {
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
		return database.Err(errRound)
	}

	for _, player := range round.Players {
		steamID := steamid.New(player.SteamID)
		if !steamID.Valid() {
			continue
		}

		player.Team = playerTeams[player.SteamID]

		if err := r.insertRoundPlayer(ctx, transaction, roundID, steamID, round, player); err != nil {
			return err
		}
	}

	return nil
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
			scoreboard_damage, suicides, extinguishes, ignites)
		VALUES(
			$1,  $2,  $3,  $4,  $5,  $6,  $7,  $8,  $9,  $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26, $27, $28, $29, $30,
			$31, $32, $33, $34, $35, $36, $37, $38, $39, $40,
			$41, $42, $43, $44, $45, $46, $47, $48, $49, $50,
			$51
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
		player.Ignites); err != nil {
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
			shots, hits, objects_built, objects_destroyed
		) VALUES (
			LOWER($1),$2,  $3,  $4,  $5,  $6,  $7,  $8,  $9,  $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31,
			$32, $33, $34, $35, $36, $37
		)`

	if _, err := transaction.Exec(ctx, query,
		variantKey, roundID, steamID.Int64(), stats.Kills, stats.Assists, stats.Deaths, stats.PostroundKills,
		stats.PostroundAssists, stats.PostroundDeaths, stats.PreroundHealing, stats.Healing, stats.PostroundHealing, stats.Drops,
		stats.NearFullChargeDeath, stats.ChargesUber, stats.ChargesKritz, stats.ChargesVacc, stats.ChargesQuickfix, stats.Damage,
		stats.DamageTaken, stats.Dominations, stats.Dominated, stats.Revenges, stats.Revenged, stats.Airshots, stats.Headshots, stats.HeadshotKills,
		stats.Backstabs, stats.BackstabKills, stats.Captures, stats.CapturesBlocked, stats.WasHeadshot, stats.WasBackstabbed,
		stats.Shots, stats.Hits, stats.ObjectBuilt, stats.ObjectDestroyed); err != nil {
		return database.Err(err)
	}

	return nil
}

func (r Repository) GetBucket(ctx context.Context, statsBucketID int32) (*Bucket, error) {
	const query = "SELECT stats_bucket_id, bucket_name FROM stats_bucket WHERE stats_bucket_id = $1"
	var bucket Bucket
	if err := r.QueryRow(ctx, query, statsBucketID).Scan(&bucket.BucketID, &bucket.BucketName); err != nil {
		return nil, database.Err(err)
	}

	return &bucket, nil
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
