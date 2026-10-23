package stats

import (
	"context"
	"errors"
	"sort"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/database/query"
)

// ErrInvalidMapStatsBucket is returned when the history request uses a time
// bucket that cannot be grouped into a time series.
var ErrInvalidMapStatsBucket = errors.New("invalid map stats time bucket")

const mapStatsMaxLimit = 1000

//nolint:gochecknoglobals // static lookup table for safe ordering.
var mapStatValidColumns = map[string][]string{
	// unique_players is an outer query alias in range queries (a correlated
	// subquery), so it must not be prefixed with the aggregate alias.
	"":   {"unique_players"},
	"m.": {"map_name"},
	"v.": {
		"matches_played", "matches_decided", "matches_drawn", "red_wins", "blu_wins",
		"total_duration_ms", "rounds_played", "stalemate_rounds",
		"player_appearances", "kills", "assists", "deaths", "damage", "healing",
		"captures", "captures_blocked", "first_played_at", "last_played_at",
	},
}

func (r Repository) MapStats(ctx context.Context, opts MapStatsOpts) ([]MapStat, uint64, MapStatsSummary, error) {
	rows, errRows := r.queryMapStatsRows(ctx, opts)
	if errRows != nil {
		return nil, 0, MapStatsSummary{}, errRows
	}

	count, errCount := r.GetCount(ctx, r.mapStatsCountBuilder(opts))
	if errCount != nil {
		return nil, 0, MapStatsSummary{}, database.Err(errCount)
	}

	summary, errSummary := r.loadMapStatsSummary(ctx, opts)
	if errSummary != nil {
		return nil, 0, MapStatsSummary{}, errSummary
	}

	return rows, count, summary, nil
}

func (r Repository) MapStatsTop(ctx context.Context, opts MapStatsOpts, limit uint64) ([]MapStat, error) {
	topOpts := opts
	topOpts.Filter = query.Filter{OrderBy: "total_duration_ms", Desc: true, Limit: limit}

	return r.queryMapStatsRows(ctx, topOpts)
}

func (r Repository) MapStatsHistory(ctx context.Context, opts MapStatsHistoryOpts) ([]MapStatsSeries, error) {
	unit, ok := mapStatsUnit(opts.TimeBucket)
	if !ok {
		return nil, ErrInvalidMapStatsBucket
	}

	builder := r.Builder().
		Select(
			"date_trunc('"+unit+"', v.date_bucket) AS date_bucket",
			"v.map_id",
			"m.map_name",
			"sum(v.matches_played)::bigint AS matches_played",
			"sum(v.matches_decided)::bigint AS matches_decided",
			"sum(v.total_duration_ms)::bigint AS duration_ms",
		).
		From("stats_map_daily_view v").
		Join("map m ON m.map_id = v.map_id").
		Where(sq.Eq{"v.stats_bucket_id": opts.StatsBucketID})

	if !opts.Start.IsZero() {
		builder = builder.Where(sq.Expr("v.date_bucket >= date_trunc('"+unit+"', ?::timestamptz)", opts.Start))
	}
	if !opts.End.IsZero() {
		builder = builder.Where(sq.Expr("v.date_bucket <= date_trunc('day', ?::timestamptz)", opts.End))
	}
	if len(opts.MapIDs) > 0 {
		builder = builder.Where(sq.Expr("v.map_id = ANY(?)", opts.MapIDs))
	}

	builder = builder.
		GroupBy("1,2,3").
		OrderBy("1", "2")

	rows, errRows := r.QueryBuilder(ctx, builder)
	if errRows != nil {
		return nil, database.Err(errRows)
	}

	var historyRows []mapStatsHistoryRow
	for rows.Next() {
		var row mapStatsHistoryRow
		if err := rows.Scan(&row.DateBucket, &row.MapID, &row.MapName, &row.MatchesPlayed, &row.MatchesDecided, &row.DurationMS); err != nil {
			return nil, database.Err(err)
		}

		historyRows = append(historyRows, row)
	}

	if rows.Err() != nil {
		return nil, database.Err(rows.Err())
	}

	return buildMapStatsSeries(historyRows, opts), nil
}

func (r Repository) queryMapStatsRows(ctx context.Context, opts MapStatsOpts) ([]MapStat, error) {
	builder := r.mapStatsRowsBuilder(opts)

	rows, errRows := r.QueryBuilder(ctx, builder)
	if errRows != nil {
		return nil, database.Err(errRows)
	}

	var stats []MapStat
	for rows.Next() {
		var stat MapStat
		if err := rows.Scan(
			&stat.MapID, &stat.MapName,
			&stat.MatchesPlayed, &stat.MatchesDecided, &stat.MatchesDrawn,
			&stat.RedWins, &stat.BluWins,
			&stat.TotalDurationMS, &stat.RoundsPlayed, &stat.StalemateRounds,
			&stat.PlayerAppearances, &stat.UniquePlayers,
			&stat.Kills, &stat.Assists, &stat.Deaths, &stat.Damage, &stat.Healing,
			&stat.Captures, &stat.CapturesBlocked,
			&stat.FirstPlayedAt, &stat.LastPlayedAt,
		); err != nil {
			return nil, database.Err(err)
		}

		stats = append(stats, stat)
	}

	if rows.Err() != nil {
		return nil, database.Err(rows.Err())
	}

	return stats, nil
}

func (r Repository) loadMapStatsSummary(ctx context.Context, opts MapStatsOpts) (MapStatsSummary, error) {
	var summary MapStatsSummary

	row, errRow := r.QueryRowBuilder(ctx, r.mapStatsSummaryBuilder(opts))
	if errRow != nil {
		return summary, database.Err(errRow)
	}

	if err := row.Scan(
		&summary.MapsPlayed,
		&summary.MatchesPlayed,
		&summary.MatchesDecided,
		&summary.MatchesDrawn,
		&summary.TotalDurationMS,
		&summary.RoundsPlayed,
	); err != nil {
		return summary, database.Err(err)
	}

	return summary, nil
}

func (r Repository) mapStatsRowsBuilder(opts MapStatsOpts) sq.SelectBuilder {
	if opts.allTime() {
		builder := r.Builder().
			Select(
				"m.map_id", "m.map_name",
				"v.matches_played", "v.matches_decided", "v.matches_drawn",
				"v.red_wins", "v.blu_wins",
				"v.total_duration_ms", "v.rounds_played", "v.stalemate_rounds",
				"v.player_appearances", "v.unique_players",
				"v.kills", "v.assists", "v.deaths", "v.damage", "v.healing",
				"v.captures", "v.captures_blocked",
				"v.first_played_at", "v.last_played_at",
			).
			From("stats_map_alltime_view v"). //nolint:misspell // name must match the database view.
			Join("map m ON m.map_id = v.map_id").
			Where(sq.Eq{"v.stats_bucket_id": opts.StatsBucketID})

		if opts.Search != "" {
			builder = builder.Where(sq.ILike{"m.map_name": "%" + opts.Search + "%"})
		}

		builder = opts.ApplySafeOrder(builder, mapStatValidColumns, "m.map_name")

		return opts.ApplyLimitOffset(builder, mapStatsMaxLimit)
	}

	aggregate := r.mapStatsRangeAggregate(opts)
	uniqueExpr, uniqueArgs := mapStatsUniquePlayersExpr(opts)

	builder := r.Builder().
		Select(
			"m.map_id", "m.map_name",
			"v.matches_played", "v.matches_decided", "v.matches_drawn",
			"v.red_wins", "v.blu_wins",
			"v.total_duration_ms", "v.rounds_played", "v.stalemate_rounds",
			"v.player_appearances",
		).
		FromSelect(aggregate, "v").
		Join("map m ON m.map_id = v.map_id").
		Column(uniqueExpr, uniqueArgs...).
		Columns(
			"v.kills", "v.assists", "v.deaths", "v.damage", "v.healing",
			"v.captures", "v.captures_blocked",
			"v.first_played_at", "v.last_played_at",
		)

	if opts.Search != "" {
		builder = builder.Where(sq.ILike{"m.map_name": "%" + opts.Search + "%"})
	}

	builder = opts.ApplySafeOrder(builder, mapStatValidColumns, "m.map_name")

	return opts.ApplyLimitOffset(builder, mapStatsMaxLimit)
}

func (r Repository) mapStatsRangeAggregate(opts MapStatsOpts) sq.SelectBuilder {
	builder := r.Builder().
		Select(
			"v.map_id",
			"sum(v.matches_played)::bigint AS matches_played",
			"sum(v.matches_decided)::bigint AS matches_decided",
			"sum(v.matches_drawn)::bigint AS matches_drawn",
			"sum(v.red_wins)::bigint AS red_wins",
			"sum(v.blu_wins)::bigint AS blu_wins",
			"sum(v.total_duration_ms)::bigint AS total_duration_ms",
			"sum(v.rounds_played)::bigint AS rounds_played",
			"sum(v.stalemate_rounds)::bigint AS stalemate_rounds",
			"sum(v.player_appearances)::bigint AS player_appearances",
			"sum(v.kills)::bigint AS kills",
			"sum(v.assists)::bigint AS assists",
			"sum(v.deaths)::bigint AS deaths",
			"sum(v.damage)::bigint AS damage",
			"sum(v.healing)::bigint AS healing",
			"sum(v.captures)::bigint AS captures",
			"sum(v.captures_blocked)::bigint AS captures_blocked",
			"min(v.first_played_at) AS first_played_at",
			"max(v.last_played_at) AS last_played_at",
		).
		From("stats_map_daily_view v").
		Where(sq.Eq{"v.stats_bucket_id": opts.StatsBucketID})

	if !opts.Start.IsZero() {
		builder = builder.Where(sq.Expr("v.date_bucket >= date_trunc('day', ?::timestamptz)", opts.Start))
	}
	if !opts.End.IsZero() {
		builder = builder.Where(sq.Expr("v.date_bucket <= date_trunc('day', ?::timestamptz)", opts.End))
	}

	return builder.GroupBy("v.map_id")
}

func (r Repository) mapStatsCountBuilder(opts MapStatsOpts) sq.SelectBuilder {
	if opts.allTime() {
		builder := r.Builder().
			Select("count(v.map_id)::bigint").
			From("stats_map_alltime_view v"). //nolint:misspell // name must match the database view.
			Join("map m ON m.map_id = v.map_id").
			Where(sq.Eq{"v.stats_bucket_id": opts.StatsBucketID})

		if opts.Search != "" {
			builder = builder.Where(sq.ILike{"m.map_name": "%" + opts.Search + "%"})
		}

		return builder
	}

	builder := r.Builder().
		Select("count(distinct v.map_id)::bigint").
		From("stats_map_daily_view v").
		Join("map m ON m.map_id = v.map_id").
		Where(sq.Eq{"v.stats_bucket_id": opts.StatsBucketID})

	if !opts.Start.IsZero() {
		builder = builder.Where(sq.Expr("v.date_bucket >= date_trunc('day', ?::timestamptz)", opts.Start))
	}
	if !opts.End.IsZero() {
		builder = builder.Where(sq.Expr("v.date_bucket <= date_trunc('day', ?::timestamptz)", opts.End))
	}
	if opts.Search != "" {
		builder = builder.Where(sq.ILike{"m.map_name": "%" + opts.Search + "%"})
	}

	return builder
}

func (r Repository) mapStatsSummaryBuilder(opts MapStatsOpts) sq.SelectBuilder {
	metrics := []string{
		"coalesce(sum(v.matches_played), 0)::bigint AS matches_played",
		"coalesce(sum(v.matches_decided), 0)::bigint AS matches_decided",
		"coalesce(sum(v.matches_drawn), 0)::bigint AS matches_drawn",
		"coalesce(sum(v.total_duration_ms), 0)::bigint AS total_duration_ms",
		"coalesce(sum(v.rounds_played), 0)::bigint AS rounds_played",
	}

	if opts.allTime() {
		builder := r.Builder().
			Select("coalesce(count(*), 0)::bigint AS maps_played").
			Columns(metrics...).
			From("stats_map_alltime_view v"). //nolint:misspell // name must match the database view.
			Join("map m ON m.map_id = v.map_id").
			Where(sq.Eq{"v.stats_bucket_id": opts.StatsBucketID})

		if opts.Search != "" {
			builder = builder.Where(sq.ILike{"m.map_name": "%" + opts.Search + "%"})
		}

		return builder
	}

	builder := r.Builder().
		Select("coalesce(count(distinct v.map_id), 0)::bigint AS maps_played").
		Columns(metrics...).
		From("stats_map_daily_view v").
		Join("map m ON m.map_id = v.map_id").
		Where(sq.Eq{"v.stats_bucket_id": opts.StatsBucketID})

	if !opts.Start.IsZero() {
		builder = builder.Where(sq.Expr("v.date_bucket >= date_trunc('day', ?::timestamptz)", opts.Start))
	}
	if !opts.End.IsZero() {
		builder = builder.Where(sq.Expr("v.date_bucket <= date_trunc('day', ?::timestamptz)", opts.End))
	}
	if opts.Search != "" {
		builder = builder.Where(sq.ILike{"m.map_name": "%" + opts.Search + "%"})
	}

	return builder
}

func mapStatsUniquePlayersExpr(opts MapStatsOpts) (string, []any) {
	expr := "(SELECT count(distinct dp.steam_id)::bigint FROM stats_map_daily_player_view dp WHERE dp.stats_bucket_id = ? AND dp.map_id = v.map_id"
	args := []any{opts.StatsBucketID}

	if !opts.Start.IsZero() {
		expr += " AND dp.date_bucket >= date_trunc('day', ?::timestamptz)"
		args = append(args, opts.Start)
	}
	if !opts.End.IsZero() {
		expr += " AND dp.date_bucket <= date_trunc('day', ?::timestamptz)"
		args = append(args, opts.End)
	}

	return expr + ") AS unique_players", args
}

func mapStatsUnit(timeBucket TimeBucket) (string, bool) {
	switch timeBucket {
	case TimeBucketDaily:
		return "day", true
	case TimeBucketWeekly:
		return "week", true
	case TimeBucketMonthly:
		return "month", true
	default:
		return "", false
	}
}

type mapStatsHistoryRow struct {
	DateBucket     time.Time
	MapID          int32
	MapName        string
	MatchesPlayed  uint64
	MatchesDecided uint64
	DurationMS     uint64
}

// buildMapStatsSeries groups raw history rows into one series per named map
// plus a combined "Other" series. The top N maps are selected by total
// duration in the queried range unless MapIDs explicitly names them.
func buildMapStatsSeries(rows []mapStatsHistoryRow, opts MapStatsHistoryOpts) []MapStatsSeries {
	if len(rows) == 0 {
		return nil
	}

	topN := opts.TopN
	if topN == 0 {
		topN = 6
	}

	totals := make(map[int32]uint64, len(rows))
	names := make(map[int32]string, len(rows))
	for _, row := range rows {
		totals[row.MapID] += row.DurationMS
		names[row.MapID] = row.MapName
	}

	selected := make(map[int32]struct{}, len(rows))
	if len(opts.MapIDs) > 0 {
		for _, id := range opts.MapIDs {
			selected[id] = struct{}{}
		}
	} else {
		ids := make([]int32, 0, len(totals))
		for id := range totals {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool {
			if totals[ids[i]] != totals[ids[j]] {
				return totals[ids[i]] > totals[ids[j]]
			}

			return ids[i] < ids[j]
		})

		for i, id := range ids {
			if uint64(i) >= uint64(topN) { //nolint:gosec // topN is bounded by the RPC validation to 20.
				break
			}
			selected[id] = struct{}{}
		}
	}

	seriesByMap := make(map[int32]*MapStatsSeries, len(selected))
	var other *MapStatsSeries

	for _, row := range rows {
		if _, ok := selected[row.MapID]; ok {
			series, ok := seriesByMap[row.MapID]
			if !ok {
				series = &MapStatsSeries{MapID: row.MapID, MapName: names[row.MapID]}
				seriesByMap[row.MapID] = series
			}
			addMapStatsPoint(series, row)

			continue
		}

		// When the caller explicitly names maps, unselected rows are filtered
		// out by the query; "Other" is only produced for top-N queries.
		if len(opts.MapIDs) == 0 {
			if other == nil {
				other = &MapStatsSeries{MapName: "Other"}
			}
			addMapStatsPoint(other, row)
		}
	}

	series := make([]MapStatsSeries, 0, len(seriesByMap)+1)
	for _, s := range seriesByMap {
		series = append(series, *s)
	}
	if other != nil && len(other.Points) > 0 {
		series = append(series, *other)
	}

	sort.Slice(series, func(left, right int) bool {
		if series[left].MapID == 0 || series[right].MapID == 0 {
			return series[left].MapID != 0
		}
		if totals[series[left].MapID] != totals[series[right].MapID] {
			return totals[series[left].MapID] > totals[series[right].MapID]
		}

		return series[left].MapID < series[right].MapID
	})

	return series
}

func addMapStatsPoint(series *MapStatsSeries, row mapStatsHistoryRow) {
	if len(series.Points) > 0 && series.Points[len(series.Points)-1].DateBucket.Equal(row.DateBucket) {
		last := &series.Points[len(series.Points)-1]
		last.MatchesPlayed += row.MatchesPlayed
		last.MatchesDecided += row.MatchesDecided
		last.DurationMS += row.DurationMS

		return
	}

	series.Points = append(series.Points, MapStatsPoint{
		DateBucket:     row.DateBucket,
		MatchesPlayed:  row.MatchesPlayed,
		MatchesDecided: row.MatchesDecided,
		DurationMS:     row.DurationMS,
	})
}
