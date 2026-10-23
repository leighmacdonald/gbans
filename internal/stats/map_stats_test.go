package stats_test

import (
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/stats"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

//nolint:tparallel // subtests share the parent fixture and its seeded matches, so they must stay sequential
func TestMapStats(t *testing.T) {
	t.Parallel()
	testFixture := tests.NewFixture()
	defer testFixture.Close()

	ctx := t.Context()
	seed := seedMapStats(t, testFixture)
	mapA, mapB, mapC := seed.mapA, seed.mapB, seed.mapC
	mapD, mapE, mapF := seed.mapD, seed.mapE, seed.mapF
	day1, day2 := seed.day1, seed.day2

	repo := stats.NewRepository(testFixture.Database)

	t.Run("range", func(t *testing.T) {
		rows, count, summary, err := repo.MapStats(ctx, stats.MapStatsOpts{
			StatsBucketID: 1,
			Start:         day1,
			End:           day2,
			Filter:        query.Filter{OrderBy: "map_name"}, //nolint:modernize // embedded field type is required in a keyed literal.
		})
		require.NoError(t, err)
		require.Equal(t, uint64(6), count)
		require.Equal(t, stats.MapStatsSummary{
			MapsPlayed:      6,
			MatchesPlayed:   7,
			MatchesDecided:  4,
			MatchesDrawn:    3,
			TotalDurationMS: 25_200_000,
			RoundsPlayed:    7,
		}, summary)

		require.Len(t, rows, 6)
		require.Equal(t, mapA, rows[0].MapID)
		require.Equal(t, "cp_map_a", rows[0].MapName)
		require.Equal(t, uint64(2), rows[0].MatchesPlayed)
		require.Equal(t, uint64(2), rows[0].MatchesDecided)
		require.Equal(t, uint64(0), rows[0].MatchesDrawn)
		require.Equal(t, uint64(1), rows[0].RedWins)
		require.Equal(t, uint64(1), rows[0].BluWins)
		require.Equal(t, uint64(10_800_000), rows[0].TotalDurationMS)
		require.Equal(t, uint64(2), rows[0].RoundsPlayed)
		require.Equal(t, uint64(4), rows[0].PlayerAppearances)
		// Unique players across two days cannot be derived by summing the
		// per-day distinct counts.
		require.Equal(t, uint64(3), rows[0].UniquePlayers)
		require.Equal(t, uint64(4), rows[0].Kills)
		require.Equal(t, uint64(400), rows[0].Damage)
		require.True(t, rows[0].FirstPlayedAt.UTC().Equal(day1))
		require.True(t, rows[0].LastPlayedAt.UTC().Equal(day2))

		require.Equal(t, mapB, rows[1].MapID)
		require.Equal(t, uint64(1), rows[1].MatchesDrawn)
		require.Equal(t, uint64(2), rows[1].UniquePlayers)

		require.Equal(t, mapC, rows[2].MapID)
		require.Equal(t, uint64(1), rows[2].MatchesDrawn)
		require.Equal(t, uint64(3), rows[2].UniquePlayers)

		require.Equal(t, mapF, rows[3].MapID)
		require.Equal(t, "custom", rows[3].MapName)
		require.Equal(t, uint64(1), rows[3].MatchesDrawn)

		require.Equal(t, mapD, rows[4].MapID)
		require.Equal(t, "pl_map_d", rows[4].MapName)
		require.Equal(t, uint64(1), rows[4].MatchesDecided)

		require.Equal(t, mapE, rows[5].MapID)
		require.Equal(t, "workshop/koth_map_e.ugc12345", rows[5].MapName)
		require.Equal(t, uint64(1), rows[5].MatchesDecided)
	})

	t.Run("sort and paginate", func(t *testing.T) {
		rows, count, _, err := repo.MapStats(ctx, stats.MapStatsOpts{
			StatsBucketID: 1,
			Start:         day1,
			End:           day2,
			Filter:        query.Filter{OrderBy: "total_duration_ms", Desc: true, Limit: 1, Offset: 1}, //nolint:modernize // embedded field type is required in a keyed literal.
		})
		require.NoError(t, err)
		require.Equal(t, uint64(6), count)
		require.Len(t, rows, 1)
		require.Equal(t, mapB, rows[0].MapID)
	})

	t.Run("search", func(t *testing.T) {
		rows, count, summary, err := repo.MapStats(ctx, stats.MapStatsOpts{
			StatsBucketID: 1,
			Start:         day1,
			End:           day2,
			Search:        "map_a",
		})
		require.NoError(t, err)
		require.Equal(t, uint64(1), count)
		require.Len(t, rows, 1)
		require.Equal(t, mapA, rows[0].MapID)
		require.Equal(t, uint64(1), summary.MapsPlayed)
	})

	t.Run("top maps", func(t *testing.T) {
		top, err := repo.MapStatsTop(ctx, stats.MapStatsOpts{
			StatsBucketID: 1,
			Start:         day1,
			End:           day2,
		}, 10)
		require.NoError(t, err)
		require.Len(t, top, 6)
		require.Equal(t, []int32{mapA, mapB, mapE, mapD, mapC, mapF},
			[]int32{top[0].MapID, top[1].MapID, top[2].MapID, top[3].MapID, top[4].MapID, top[5].MapID})
	})

	t.Run("all time", func(t *testing.T) {
		rows, count, summary, err := repo.MapStats(ctx, stats.MapStatsOpts{StatsBucketID: 1})
		require.NoError(t, err)
		require.Equal(t, uint64(6), count)
		require.Equal(t, uint64(8), summary.MatchesPlayed)
		require.Equal(t, uint64(4), summary.MatchesDecided)
		require.Equal(t, uint64(4), summary.MatchesDrawn)
		require.Equal(t, uint64(27_000_000), summary.TotalDurationMS)

		byMap := make(map[int32]stats.MapStat, len(rows))
		for _, row := range rows {
			byMap[row.MapID] = row
		}

		require.Equal(t, uint64(2), byMap[mapC].MatchesPlayed)
		require.Equal(t, uint64(4), byMap[mapC].UniquePlayers)
	})

	testMapStatsHistory(t, repo, seed)

	t.Run("prefixes", func(t *testing.T) {
		prefixes, err := repo.MapStatsPrefixes(ctx, stats.MapStatsOpts{
			StatsBucketID: 1,
			Start:         day1,
			End:           day2,
		})
		require.NoError(t, err)
		// The workshop map counts under koth_ and the prefix-less map under
		// other, ordered by total playtime descending.
		require.Equal(t, []stats.MapPrefixStat{
			{Prefix: "cp_", MapsPlayed: 3, MatchesPlayed: 4, MatchesDecided: 2, TotalDurationMS: 18_000_000},
			{Prefix: "koth_", MapsPlayed: 1, MatchesPlayed: 1, MatchesDecided: 1, TotalDurationMS: 3_600_000},
			{Prefix: "pl_", MapsPlayed: 1, MatchesPlayed: 1, MatchesDecided: 1, TotalDurationMS: 2_700_000},
			{Prefix: "other", MapsPlayed: 1, MatchesPlayed: 1, TotalDurationMS: 900_000},
		}, prefixes)

		allTime, err := repo.MapStatsPrefixes(ctx, stats.MapStatsOpts{StatsBucketID: 1})
		require.NoError(t, err)
		byPrefix := make(map[string]stats.MapPrefixStat, len(allTime))
		for _, prefix := range allTime {
			byPrefix[prefix.Prefix] = prefix
		}
		require.Equal(t, uint64(5), byPrefix["cp_"].MatchesPlayed)
		require.Equal(t, uint64(1), byPrefix["koth_"].MatchesPlayed)
		require.Equal(t, uint64(1), byPrefix["pl_"].MapsPlayed)
		require.Equal(t, uint64(1), byPrefix["other"].MapsPlayed)
	})
}

// testMapStatsHistory verifies the bucketed time series queries. It runs as
// subtests of TestMapStats so it can share the seeded fixture.
func testMapStatsHistory(t *testing.T, repo stats.Repository, seed mapStatsSeed) {
	t.Helper()
	ctx := t.Context()
	mapA, mapB, mapC := seed.mapA, seed.mapB, seed.mapC
	mapD, mapE, mapF := seed.mapD, seed.mapE, seed.mapF
	day1, day2 := seed.day1, seed.day2

	t.Run("history daily", func(t *testing.T) {
		series, err := repo.MapStatsHistory(ctx, stats.MapStatsHistoryOpts{
			StatsBucketID: 1,
			Start:         day1,
			End:           day2,
			TimeBucket:    stats.TimeBucketDaily,
			TopN:          2,
		})
		require.NoError(t, err)
		require.Len(t, series, 3)

		// Top 2 by duration are named, remaining maps are combined into Other.
		require.Equal(t, mapA, series[0].MapID)
		require.Equal(t, mapB, series[1].MapID)
		require.Equal(t, int32(0), series[2].MapID)
		require.Equal(t, "Other", series[2].MapName)

		day1Start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		day2Start := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

		require.Len(t, series[0].Points, 2)
		require.True(t, series[0].Points[0].DateBucket.UTC().Equal(day1Start))
		require.Equal(t, uint64(1), series[0].Points[0].MatchesPlayed)
		require.Equal(t, uint64(3_600_000), series[0].Points[0].DurationMS)
		require.True(t, series[0].Points[1].DateBucket.UTC().Equal(day2Start))
		require.Equal(t, uint64(7_200_000), series[0].Points[1].DurationMS)

		require.Len(t, series[1].Points, 1)
		require.Equal(t, uint64(5_400_000), series[1].Points[0].DurationMS)

		// Other now covers mapC, mapD, mapE and mapF.
		require.Len(t, series[2].Points, 2)
		require.Equal(t, uint64(2), series[2].Points[0].MatchesPlayed)
		require.Equal(t, uint64(3_600_000), series[2].Points[0].DurationMS)
		require.Equal(t, uint64(2), series[2].Points[1].MatchesPlayed)
		require.Equal(t, uint64(5_400_000), series[2].Points[1].DurationMS)
	})

	t.Run("history weekly and monthly", func(t *testing.T) {
		for _, bucket := range []stats.TimeBucket{stats.TimeBucketWeekly, stats.TimeBucketMonthly} {
			series, err := repo.MapStatsHistory(ctx, stats.MapStatsHistoryOpts{
				StatsBucketID: 1,
				Start:         day1,
				End:           day2,
				TimeBucket:    bucket,
				TopN:          6,
			})
			require.NoError(t, err)
			require.Len(t, series, 6, "topN >= map count produces no Other series for %s", bucket)

			totalByMap := make(map[int32]uint64, len(series))
			for _, s := range series {
				require.Len(t, s.Points, 1, "%s series %s", bucket, s.MapName)
				totalByMap[s.MapID] = s.Points[0].DurationMS
			}

			require.Equal(t, uint64(10_800_000), totalByMap[mapA])
			require.Equal(t, uint64(5_400_000), totalByMap[mapB])
			require.Equal(t, uint64(1_800_000), totalByMap[mapC])
			require.Equal(t, uint64(2_700_000), totalByMap[mapD])
			require.Equal(t, uint64(3_600_000), totalByMap[mapE])
			require.Equal(t, uint64(900_000), totalByMap[mapF])
		}
	})

	t.Run("history explicit maps", func(t *testing.T) {
		series, err := repo.MapStatsHistory(ctx, stats.MapStatsHistoryOpts{
			StatsBucketID: 1,
			Start:         day1,
			End:           day2,
			TimeBucket:    stats.TimeBucketDaily,
			TopN:          2,
			MapIDs:        []int32{mapC},
		})
		require.NoError(t, err)
		require.Len(t, series, 1)
		require.Equal(t, mapC, series[0].MapID)
		require.Equal(t, "cp_map_c", series[0].MapName)
		require.Len(t, series[0].Points, 1)
	})

	t.Run("invalid bucket", func(t *testing.T) {
		_, err := repo.MapStatsHistory(ctx, stats.MapStatsHistoryOpts{
			StatsBucketID: 1,
			TimeBucket:    stats.TimeBucketAlltime,
		})
		require.ErrorIs(t, err, stats.ErrInvalidMapStatsBucket)
	})
}

// mapStatsSeed holds the fixture references the map stats tests assert on.
type mapStatsSeed struct {
	mapA int32
	mapB int32
	mapC int32
	mapD int32
	mapE int32
	mapF int32
	day1 time.Time
	day2 time.Time
}

// seedMapStats seeds matches for two in-range days, one out-of-range day and
// one match in a separate bucket, then refreshes the map stats views.
func seedMapStats(t *testing.T, testFixture *tests.Fixture) mapStatsSeed {
	t.Helper()
	ctx := t.Context()
	server := testFixture.CreateTestServer(ctx)
	mapA := seedTestMap(t, testFixture, "cp_map_a")
	mapB := seedTestMap(t, testFixture, "cp_map_b")
	mapC := seedTestMap(t, testFixture, "cp_map_c")
	// Other map type prefixes, including a workshop map (which stores its
	// name with a workshop/ prefix and .ugc suffix) and a prefix-less map.
	mapD := seedTestMap(t, testFixture, "pl_map_d")
	mapE := seedTestMap(t, testFixture, "workshop/koth_map_e.ugc12345")
	mapF := seedTestMap(t, testFixture, "custom")
	demo := seedTestDemo(t, testFixture, server.ServerID, "map-stats-demo", "cp_map_a")

	day1 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	outsideDay := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	playerA := steamid.New("[U:1:7100000001]")
	playerB := steamid.New("[U:1:7100000002]")
	playerC := steamid.New("[U:1:7100000003]")
	playerD := steamid.New("[U:1:7100000004]")
	insertTestPersons(ctx, t, testFixture.Database, map[steamid.SteamID]string{
		playerA: "MapStatsA",
		playerB: "MapStatsB",
		playerC: "MapStatsC",
		playerD: "MapStatsD",
	})

	// Bucket 1, inside the tested 2026-01-01..2026-01-02 range.
	seedMapStatsMatch(t, testFixture, server.ServerID, mapA, demo, 1, day1, 13, 7, 3_600_000, playerA, playerB)
	seedMapStatsMatch(t, testFixture, server.ServerID, mapB, demo, 1, day1, 13, 13, 5_400_000, playerA, playerC)
	seedMapStatsMatch(t, testFixture, server.ServerID, mapA, demo, 1, day2, 7, 13, 7_200_000, playerB, playerC)
	seedMapStatsMatch(t, testFixture, server.ServerID, mapC, demo, 1, day2, 5, 5, 1_800_000, playerA, playerB, playerC)
	seedMapStatsMatch(t, testFixture, server.ServerID, mapD, demo, 1, day1, 9, 5, 2_700_000, playerA, playerB)
	seedMapStatsMatch(t, testFixture, server.ServerID, mapE, demo, 1, day2, 11, 9, 3_600_000, playerC, playerD)
	seedMapStatsMatch(t, testFixture, server.ServerID, mapF, demo, 1, day1, 1, 1, 900_000, playerA)

	// Outside the tested range, used to prove range and all-time behaviour.
	seedMapStatsMatch(t, testFixture, server.ServerID, mapC, demo, 1, outsideDay, 13, 13, 1_800_000, playerD)

	// Bucket isolation: this match must not leak into bucket 1.
	seedMapStatsMatch(t, testFixture, server.ServerID, mapA, demo, 2, day1, 13, 7, 10_000_000, playerA)

	for _, view := range []string{
		"stats_map_daily_view",
		"stats_map_daily_player_view",
		"stats_map_alltime_view", //nolint:misspell // name must match the database view.
	} {
		require.NoError(t, testFixture.Database.RefreshMaterializedView(ctx, view))
	}

	return mapStatsSeed{mapA: mapA, mapB: mapB, mapC: mapC, mapD: mapD, mapE: mapE, mapF: mapF, day1: day1, day2: day2}
}

// seedMapStatsMatch inserts a bare match row plus one round and one
// match_round_player row per player. Player rows carry small, deterministic
// combat metrics so view sums can be asserted.
func seedMapStatsMatch(
	t *testing.T,
	testFixture *tests.Fixture,
	serverID, mapID, demoID, bucketID int32,
	start time.Time,
	scoreRed, scoreBlu int32,
	durationMS int64,
	players ...steamid.SteamID,
) uuid.UUID {
	t.Helper()
	ctx := t.Context()

	matchID, errMatch := uuid.NewV4()
	require.NoError(t, errMatch)

	require.NoError(t, testFixture.Database.Exec(ctx, `
		INSERT INTO match (
			match_id, server_id, map_id, demo_id, stats_bucket_id, hostname,
			score_red, score_blu, start_time, duration_ms, created_on
		) VALUES ($1, $2, $3, $4, $5, 'map-stats', $6, $7, $8, $9, now())`,
		matchID, serverID, mapID, demoID, bucketID, scoreRed, scoreBlu, start, durationMS))

	if len(players) == 0 {
		return matchID
	}

	var roundID int64
	//nolint:dupword // the SQL repeats the literal false.
	require.NoError(t, testFixture.Database.QueryRow(ctx, `
		INSERT INTO match_round (match_id, winner, is_stalemate, is_sudden_death, duration_ms)
		VALUES ($1, 'red', false, false, $2) RETURNING round_id`,
		matchID, durationMS).Scan(&roundID))

	for _, player := range players {
		require.NoError(t, testFixture.Database.Exec(ctx, `
			INSERT INTO match_round_player (
				round_id, steam_id, team, mvp, tick_start, tick_end,
				kills, assists, deaths, postround_kills, postround_assists, postround_deaths,
				preround_healing, healing, postround_healing, drops, near_full_charge_death,
				charges_uber, charges_kritz, charges_vacc, charges_quickfix,
				damage, damage_taken, dominations, dominated, revenges, revenged,
				airshots, headshots, headshot_kills, backstabs, backstab_kills,
				captures, captures_blocked, was_headshot, was_backstabbed,
				shots, hits, objects_built, objects_destroyed,
				points, connection_count, bonus_points,
				scoreboard_kills, scoreboard_assists, scoreboard_healing, scoreboard_deaths, scoreboard_damage,
				suicides, extinguishes, ignites
			) VALUES (
				$1, $2, 'red', false, 1000, 2000,
				1, 1, 1, 0, 0, 0,
				0, 50, 0, 0, 0,
				0, 0, 0, 0,
				100, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0,
				1, 1, 0, 0,
				0, 0, 0, 0,
				10, 1, 0,
				1, 1, 0, 0, 0,
				0, 0, 0
			)`, roundID, player.Int64()))
	}

	return matchID
}
