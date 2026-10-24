package stats_test

import (
	"context"
	stdjson "encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"connectrpc.com/authn"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/database"
	queryv1 "github.com/leighmacdonald/gbans/internal/database/query/v1"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/json"
	"github.com/leighmacdonald/gbans/internal/maps"
	"github.com/leighmacdonald/gbans/internal/opengraph"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/stats"
	statsv1 "github.com/leighmacdonald/gbans/internal/stats/v1"
	statsv1connect "github.com/leighmacdonald/gbans/internal/stats/v1/statsv1connect"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/demoparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestImport(t *testing.T) {
	t.Parallel()
	testFixture := tests.NewFixture()
	defer testFixture.Close()

	server := testFixture.CreateTestServer(t.Context())

	demoJSON, err := os.Open("testdata/demo-1427611.json")
	require.NoError(t, err)
	demo, errDemo := json.Decode[demoparse.Demo](demoJSON)
	require.NoError(t, errDemo)

	ctx := t.Context()

	seen := map[steamid.SteamID]struct{}{}
	for _, round := range demo.Rounds {
		for _, player := range round.Players {
			sid := steamid.New(player.SteamID)
			if !sid.Valid() {
				continue
			}
			if _, ok := seen[sid]; ok {
				continue
			}
			seen[sid] = struct{}{}
			err := testFixture.Database.Exec(ctx,
				`INSERT INTO person (steam_id, created_on, updated_on, personaname, avatarhash, profilestate, personastate,
				                    realname, timecreated, loccountrycode, locstatecode, loccityid)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
				 ON CONFLICT DO NOTHING`,
				sid.Int64(), time.Now(), time.Now(), player.Name, "", 0, 0, "", 0, "", "", 0)
			require.NoError(t, err)
		}
	}

	var demoID int32
	errDemo = testFixture.Database.QueryRow(ctx,
		`INSERT INTO demo (server_id, title, map_name, created_on) VALUES ($1, $2, $3, $4) RETURNING demo_id`,
		server.ServerID, demo.Filename, demo.Map, time.Now()).Scan(&demoID)
	require.NoError(t, errDemo)

	stats.RegisterDiscordCommands(discord.Discard{})

	notifier := &captureNotifier{}
	statSvc := stats.New(stats.NewRepository(testFixture.Database), maps.New(maps.NewRepository(testFixture.Database)),
		stats.WithMatchLogNotifications(notifier, "match-log-channel"))
	matchID, importErr := statSvc.Import(ctx, server.ServerID, demoID, &demo, time.Now())
	require.NoError(t, importErr)
	require.NotNil(t, matchID)

	match, errMatch := statSvc.Match(ctx, *matchID)
	require.NoError(t, errMatch)
	require.EqualValues(t, demo.Scores().Red, match.ScoreRed)
	require.EqualValues(t, demo.Scores().Blu, match.ScoreBlu)
	require.NotEmpty(t, match.MapName)

	require.Len(t, notifier.payloads, 1)
	require.Equal(t, []string{"match-log-channel"}, notifier.payloads[0].DiscordChannels)
	require.NotNil(t, notifier.payloads[0].MessageSend)
}

// demoPlayerSums aggregates per-player totals across all rounds of a parsed
// demo, mirroring how the summary views group by steam_id.
type demoPlayerSums struct {
	name              string
	avatarHash        string
	kills             uint64
	deaths            uint64
	points            uint64
	healing           uint64
	postroundHealing  uint64
	scoreboardHealing uint64
}

func sumDemoPlayers(demo *demoparse.Demo) map[steamid.SteamID]*demoPlayerSums {
	sums := map[steamid.SteamID]*demoPlayerSums{}
	for _, round := range demo.Rounds {
		for _, player := range round.Players {
			sid := steamid.New(player.SteamID)
			if !sid.Valid() {
				continue
			}
			agg, ok := sums[sid]
			if !ok {
				agg = &demoPlayerSums{
					name:       player.Name,
					avatarHash: fmt.Sprintf("avatar-%d", sid.Int64()),
				}
				sums[sid] = agg
			}
			agg.kills += uint64(player.Kills)                         //nolint:gosec // parser counts are non-negative.
			agg.deaths += uint64(player.Deaths)                       //nolint:gosec // parser counts are non-negative.
			agg.points += uint64(player.Points)                       //nolint:gosec // parser counts are non-negative.
			agg.healing += uint64(player.Healing)                     //nolint:gosec // parser counts are non-negative.
			agg.postroundHealing += uint64(player.PostroundHealing)   //nolint:gosec // parser counts are non-negative.
			agg.scoreboardHealing += uint64(player.ScoreboardHealing) //nolint:gosec // parser counts are non-negative.
		}
	}

	return sums
}

func importDemoAt(t *testing.T, testFixture *tests.Fixture, statSvc stats.Stats, serverID int32, demo *demoparse.Demo, start time.Time) {
	t.Helper()
	ctx := t.Context()

	var demoID int32
	require.NoError(t, testFixture.Database.QueryRow(ctx,
		`INSERT INTO demo (server_id, title, map_name, created_on) VALUES ($1, $2, $3, $4) RETURNING demo_id`,
		serverID, demo.Filename, demo.Map, start).Scan(&demoID))

	_, errImport := statSvc.Import(ctx, serverID, demoID, demo, start)
	require.NoError(t, errImport)
}

// fillerTestPlayers returns num deterministic valid filler players in a
// dedicated steamid range, so synthetic test demos satisfy the MinPlayers
// match-creation threshold without colliding with other tests.
func fillerTestPlayers(num int, tickStart int, tickEnd int) ([]steamid.SteamID, []demoparse.PlayerSummary) {
	ids := make([]steamid.SteamID, 0, num)
	players := make([]demoparse.PlayerSummary, 0, num)

	for i := range num {
		sid := steamid.New(fmt.Sprintf("[U:1:%d]", 90000001+i))
		ids = append(ids, sid)
		players = append(players, demoparse.PlayerSummary{
			Name: fmt.Sprintf("Filler%d", i), SteamID: string(sid.Steam3()),
			Team: "red", TickStart: tickStart, TickEnd: tickEnd,
		})
	}

	return ids, players
}

// insertTestPersons ensures person rows exist for every id (match tables
// reference person via foreign key).
func insertTestPersons(ctx context.Context, t *testing.T, db database.Database, persons map[steamid.SteamID]string) {
	t.Helper()

	for sid, name := range persons {
		require.NoError(t, db.Exec(ctx,
			`INSERT INTO person (steam_id, created_on, updated_on, personaname, avatarhash, profilestate, personastate,
			                    realname, timecreated, loccountrycode, locstatecode, loccityid)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			 ON CONFLICT DO NOTHING`,
			sid.Int64(), time.Now(), time.Now(), name, "", 0, 0, "", 0, "", "", 0))
	}
}

func queryOverall(t *testing.T, repo stats.Repository, bucket uint32, timeBucket stats.TimeBucket, stamp time.Time) map[steamid.SteamID]stats.OverallStats {
	t.Helper()

	rows, _, errQuery := repo.Query(t.Context(), stats.Opts{
		StatsBucketID: bucket,
		TimeBucket:    timeBucket,
		TimeStamp:     stamp,
	})
	require.NoError(t, errQuery)

	out := map[steamid.SteamID]stats.OverallStats{}
	for _, row := range rows {
		stat, ok := row.(stats.OverallStats)
		require.True(t, ok)
		out[stat.SteamID] = stat
	}

	return out
}

// testPerson is a seeded person row for demo imports.
type testPerson struct {
	id   steamid.SteamID
	name string
}

// seedTestPersons creates the person rows a demo import requires.
func seedTestPersons(t *testing.T, testFixture *tests.Fixture, persons ...testPerson) {
	t.Helper()
	ctx := t.Context()

	for _, person := range persons {
		require.NoError(t, testFixture.Database.Exec(ctx,
			`INSERT INTO person (steam_id, created_on, updated_on, personaname, avatarhash, profilestate, personastate,
			                    realname, timecreated, loccountrycode, locstatecode, loccityid)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			 ON CONFLICT DO NOTHING`,
			person.id.Int64(), time.Now(), time.Now(), person.name, "", 0, 0, "", 0, "", "", 0))
	}
}

// TestGlobalStatsAcrossBucketsAndTime imports the same match several times on
// different servers (stats buckets) and match dates, then verifies the
// summary views aggregate correctly per stats bucket and time bucket.
func TestGlobalStatsAcrossBucketsAndTime(t *testing.T) {
	t.Parallel()
	testFixture := tests.NewFixture()
	defer testFixture.Close()

	ctx := t.Context()

	serverA := testFixture.CreateTestServer(ctx)
	serverB := testFixture.CreateTestServer(ctx)
	require.NoError(t, testFixture.Database.Exec(ctx,
		`UPDATE server SET stats_bucket_id = 2 WHERE server_id = $1`, serverB.ServerID))

	demoJSON, err := os.Open("testdata/demo-1427611.json")
	require.NoError(t, err)
	demo, errDemo := json.Decode[demoparse.Demo](demoJSON)
	require.NoError(t, errDemo)

	seen := map[steamid.SteamID]struct{}{}
	for _, round := range demo.Rounds {
		for _, player := range round.Players {
			sid := steamid.New(player.SteamID)
			if !sid.Valid() {
				continue
			}
			if _, ok := seen[sid]; ok {
				continue
			}
			seen[sid] = struct{}{}
			require.NoError(t, testFixture.Database.Exec(ctx,
				`INSERT INTO person (steam_id, created_on, updated_on, personaname, avatarhash, profilestate, personastate,
				                    realname, timecreated, loccountrycode, locstatecode, loccityid)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
				 ON CONFLICT DO NOTHING`,
				sid.Int64(), time.Now(), time.Now(), player.Name, fmt.Sprintf("avatar-%d", sid.Int64()), 0, 0, "", 0, "", "", 0))
		}
	}

	repo := stats.NewRepository(testFixture.Database)
	statSvc := stats.New(repo, maps.New(maps.NewRepository(testFixture.Database)))

	// Mon 2026-09-07, Tue 2026-09-08 (same ISO week, same month) and Sat
	// 2026-08-15 (different week and month).
	day1 := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	day3 := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	importDemoAt(t, testFixture, statSvc, serverA.ServerID, &demo, day1)
	importDemoAt(t, testFixture, statSvc, serverA.ServerID, &demo, day2)
	importDemoAt(t, testFixture, statSvc, serverA.ServerID, &demo, day3)
	importDemoAt(t, testFixture, statSvc, serverB.ServerID, &demo, day1)

	// Exercise the same path as the daily scheduler and admin RefreshStats RPC.
	require.NoError(t, statSvc.RefreshViews(ctx))

	expected := sumDemoPlayers(&demo)
	require.NotEmpty(t, expected)

	topPlayer := steamid.SteamID{}
	var topPoints uint64
	for sid, sums := range expected {
		if sums.points > topPoints {
			topPoints = sums.points
			topPlayer = sid
		}
	}

	topBy := func(metric func(*demoPlayerSums) uint64) map[steamid.SteamID]struct{} {
		var best uint64
		first := true
		out := map[steamid.SteamID]struct{}{}
		for sid, sums := range expected {
			value := metric(sums)
			if first || value > best {
				best = value
				out = map[steamid.SteamID]struct{}{sid: {}}
				first = false
			} else if value == best {
				out[sid] = struct{}{}
			}
		}

		return out
	}
	topKills := topBy(func(s *demoPlayerSums) uint64 { return s.kills })
	topHealing := topBy(func(s *demoPlayerSums) uint64 { return s.healing })

	assertBucket := func(rows map[steamid.SteamID]stats.OverallStats, multiplier uint64, msg string) {
		var eligible int
		for _, sums := range expected {
			if sums.kills >= stats.MinKills {
				eligible++
			}
		}
		require.Len(t, rows, eligible, msg)
		for sid, sums := range expected {
			got, found := rows[sid]
			if sums.kills < stats.MinKills {
				require.False(t, found, "%s: player below the minimum kills must be excluded %s", msg, sid)

				continue
			}
			require.True(t, found, "%s: missing player %s", msg, sid)
			require.Equal(t, sums.name, got.Personaname, "%s: personaname %s", msg, sid)
			require.Equal(t, sums.avatarHash, got.AvatarHash, "%s: avatarhash %s", msg, sid)
			require.Equal(t, sums.kills*multiplier, got.Kills, "%s: kills %s", msg, sid)
			require.Equal(t, sums.deaths*multiplier, got.Deaths, "%s: deaths %s", msg, sid)
			require.Equal(t, sums.points*multiplier, got.Points, "%s: points %s", msg, sid)
			require.Equal(t, sums.healing*multiplier, got.Healing, "%s: healing %s", msg, sid)
			require.Equal(t, sums.postroundHealing*multiplier, got.PostroundHealing, "%s: postround_healing %s", msg, sid)
			require.Equal(t, sums.scoreboardHealing*multiplier, got.ScoreboardHealing, "%s: scoreboard_healing %s", msg, sid)
		}
	}

	// Daily: each import day holds exactly one copy in bucket 1.
	assertBucket(queryOverall(t, repo, 1, stats.TimeBucketDaily, day1), 1, "daily bucket1 day1")
	assertBucket(queryOverall(t, repo, 1, stats.TimeBucketDaily, day2), 1, "daily bucket1 day2")
	assertBucket(queryOverall(t, repo, 1, stats.TimeBucketDaily, day3), 1, "daily bucket1 day3")

	// Weekly: day1+day2 share the week of 2026-09-07; day3 stands alone.
	assertBucket(queryOverall(t, repo, 1, stats.TimeBucketWeekly, day2), 2, "weekly bucket1")
	assertBucket(queryOverall(t, repo, 1, stats.TimeBucketWeekly, day3), 1, "weekly bucket1 day3")

	// Monthly: September holds day1+day2, August holds day3.
	assertBucket(queryOverall(t, repo, 1, stats.TimeBucketMonthly, day1), 2, "monthly bucket1 september")
	assertBucket(queryOverall(t, repo, 1, stats.TimeBucketMonthly, day3), 1, "monthly bucket1 august")

	// Yearly + all-time aggregate everything in the bucket.
	assertBucket(queryOverall(t, repo, 1, stats.TimeBucketYearly, day1), 3, "yearly bucket1")
	assertBucket(queryOverall(t, repo, 1, stats.TimeBucketAlltime, day1), 3, "all-time bucket1")

	// Bucket isolation: serverB's import lives in bucket 2 only.
	assertBucket(queryOverall(t, repo, 2, stats.TimeBucketDaily, day1), 1, "daily bucket2 day1")
	assertBucket(queryOverall(t, repo, 2, stats.TimeBucketAlltime, day1), 1, "all-time bucket2")

	// Ranks are partitioned per (bucket, period): each bucket crowns its own
	// rank 1 for the same top player.
	for _, rows := range []map[steamid.SteamID]stats.OverallStats{
		queryOverall(t, repo, 1, stats.TimeBucketDaily, day1),
		queryOverall(t, repo, 1, stats.TimeBucketAlltime, day1),
		queryOverall(t, repo, 2, stats.TimeBucketDaily, day1),
		queryOverall(t, repo, 2, stats.TimeBucketAlltime, day1),
	} {
		require.Equal(t, uint64(1), rows[topPlayer].Rank, "top player must rank 1")
	}

	// Kills and Healing leaderboards rank by their own metric, not points.
	for _, testCase := range []struct {
		name    string
		variant stats.Variant
		tops    map[steamid.SteamID]struct{}
	}{
		{"kills", stats.VariantKills, topKills},
		{"healing", stats.VariantHealing, topHealing},
	} {
		rows, _, errRank := repo.Query(ctx, stats.Opts{
			StatsBucketID: 1, Variant: testCase.variant, TimeBucket: stats.TimeBucketAlltime,
		})
		require.NoError(t, errRank)
		require.NotEmpty(t, rows)
		first, ok := rows[0].(stats.OverallStats)
		require.True(t, ok)
		require.Contains(t, testCase.tops, first.SteamID, "%s leaderboard must crown a top player", testCase.name)
		require.Equal(t, uint64(1), first.Rank)

		// Every ranked player must meet the view's minimum stat threshold.
		for _, row := range rows {
			overall, okRow := row.(stats.OverallStats)
			require.True(t, okRow)
			if testCase.variant == stats.VariantHealing {
				require.GreaterOrEqual(t, overall.Healing, uint64(stats.MinHealing),
					"healing leaderboard must only include players above the healing minimum")
			} else {
				require.GreaterOrEqual(t, overall.Kills, uint64(stats.MinKills),
					"kills leaderboard must only include players above the kills minimum")
			}
		}
	}

	// Variant rows carry the same display info.
	variantRows, _, errVariants := repo.Query(ctx, stats.Opts{
		StatsBucketID: 1, Variant: stats.VariantClasses, VariantKey: "scout",
		TimeBucket: stats.TimeBucketAlltime,
	})
	require.NoError(t, errVariants)
	require.NotEmpty(t, variantRows)

	for _, row := range variantRows {
		variantStat, ok := row.(stats.VariantStats)
		require.True(t, ok)
		require.NotEmpty(t, variantStat.Personaname)
		require.NotEmpty(t, variantStat.AvatarHash)
	}
}

// TestImportNewParserStats ensures the stats added with the tf2_demostats
// v0.3.x parser survive the full import, match read, and summary view paths.
func TestImportNewParserStats(t *testing.T) {
	t.Parallel()
	testFixture := tests.NewFixture()
	defer testFixture.Close()

	ctx := t.Context()
	server := testFixture.CreateTestServer(ctx)

	steamID := steamid.New("[U:1:12345678]")
	require.True(t, steamID.Valid())

	victimID := steamid.New("[U:1:87654321]")
	require.True(t, victimID.Valid())

	fillerIDs, fillerPlayers := fillerTestPlayers(5, 1000, 5000)

	persons := map[steamid.SteamID]string{
		steamID:  "NewStats",
		victimID: "Victim",
	}
	for i, id := range fillerIDs {
		persons[id] = fmt.Sprintf("Filler%d", i)
	}
	insertTestPersons(ctx, t, testFixture.Database, persons)

	demoID := seedTestDemo(t, testFixture, server.ServerID, "new-stats.dem", "cp_process_final")

	player := demoparse.PlayerSummary{
		Name: "NewStats", SteamID: string(steamID.Steam3()),
		Kills: 10, Healing: 500,
		Heals: 101, Healed: 102, CrossbowHeals: 103, CrossbowHealing: 104,
		HealOnHit: 105, BuildingHealing: 106, DroppedUbers: 107,
		Reflects: 108, Defenses: 109, DirectHits: 110, Teleports: 111,
		PushDistance: 112, EnvironmentalDeaths: 113, EnvironmentalKills: 114,
		ObjectPlaced: 115, ObjectUpgraded: 116, ObjectCarried: 117,
		ObjectDropped: 118, ObjectRemoved: 119, ObjectDetonated: 120,
		AmmoPacks: 121, HealthPacks: 122, HealthPackHealing: 123,
		Classes: map[string]demoparse.Stats{
			"soldier": {
				Kills: 201, Heals: 202, Healed: 203, CrossbowHeals: 204,
				CrossbowHealing: 205, HealOnHit: 206, Extinguishes: 207,
				BuildingHealing: 208, DroppedUbers: 209, Reflects: 210,
				Defenses: 211, DirectHits: 212, Teleports: 213, PushDistance: 214,
				EnvironmentalDeaths: 215, EnvironmentalKills: 216,
				ObjectPlaced: 217, ObjectUpgraded: 218, ObjectCarried: 219,
				ObjectDropped: 220, ObjectRemoved: 221, ObjectDetonated: 222,
				AmmoPacks: 223, HealthPacks: 224, HealthPackHealing: 225,
			},
		},
		Weapons: map[string]demoparse.Stats{
			"tf_projectile_rocket": {
				Kills: 301, Heals: 302, Healed: 303, CrossbowHeals: 304,
				CrossbowHealing: 305, HealOnHit: 306, Extinguishes: 307,
				BuildingHealing: 308, DroppedUbers: 309, Reflects: 310,
				Defenses: 311, DirectHits: 312, Teleports: 313, PushDistance: 314,
				EnvironmentalDeaths: 315, EnvironmentalKills: 316,
				ObjectPlaced: 317, ObjectUpgraded: 318, ObjectCarried: 319,
				ObjectDropped: 320, ObjectRemoved: 321, ObjectDetonated: 322,
				AmmoPacks: 323, HealthPacks: 324, HealthPackHealing: 325,
			},
		},
	}

	demo := demoparse.Demo{
		Filename: "new-stats.dem", DemoType: demoparse.HL2Demo,
		Server: "test", Map: "cp_process_final", Game: "tf",
		Duration: 600, Ticks: 40000, Frames: 39900, Signon: 1000,
		Rounds: []demoparse.RoundSummary{
			{
				Winner: "red", Time: 300,
				Mvps:    []string{string(steamID.Steam3())},
				Winners: []string{string(steamID.Steam3())},
				Players: append([]demoparse.PlayerSummary{player}, fillerPlayers...),
			},
		},
		Events: []demoparse.MatchEvent{
			{
				Tick: 1000, Type: demoparse.MatchEventKill,
				Kill: &demoparse.KillEvent{
					Tick: 1000, Killer: string(steamID.Steam3()), Victim: string(victimID.Steam3()),
					Weapon:       "tf_projectile_rocket",
					KillerPos:    &demoparse.Position{X: -5496, Y: 5393.625, Z: 348},
					VictimPos:    &demoparse.Position{X: -5441.75, Y: 5269.125, Z: 363.25},
					KillerAngles: &demoparse.EyeAngles{Pitch: 26.47, Yaw: 268.85},
					VictimAngles: &demoparse.EyeAngles{Pitch: 8.82, Yaw: 137.24},
				},
			},
			{
				// Environmental kill has no killer or positions.
				Tick: 2000, Type: demoparse.MatchEventKill,
				Kill: &demoparse.KillEvent{
					Tick: 2000, Victim: string(steamID.Steam3()), Weapon: "world",
				},
			},
		},
	}

	repo := stats.NewRepository(testFixture.Database)
	st := stats.New(repo, maps.New(maps.NewRepository(testFixture.Database)))

	matchID, errImport := st.Import(ctx, server.ServerID, demoID, &demo, time.Now())
	require.NoError(t, errImport)
	require.NotNil(t, matchID)

	match, errMatch := repo.Match(ctx, *matchID)
	require.NoError(t, errMatch)
	require.Len(t, match.Players, 6)

	var got stats.MatchOverallStatsRound
	for _, player := range match.Players {
		if player.SteamID.Equal(steamID) {
			got = player

			break
		}
	}
	require.True(t, got.SteamID.Equal(steamID))
	require.Equal(t, "NewStats", got.Personaname)
	require.Equal(t, uint64(10), got.Kills)
	require.Equal(t, uint64(500), got.Healing)
	require.Equal(t, uint64(101), got.Heals)
	require.Equal(t, uint64(102), got.Healed)
	require.Equal(t, uint64(103), got.CrossbowHeals)
	require.Equal(t, uint64(104), got.CrossbowHealing)
	require.Equal(t, uint64(105), got.HealOnHit)
	require.Equal(t, uint64(106), got.BuildingHealing)
	require.Equal(t, uint64(107), got.DroppedUbers)
	require.Equal(t, uint64(108), got.Reflects)
	require.Equal(t, uint64(109), got.Defenses)
	require.Equal(t, uint64(110), got.DirectHits)
	require.Equal(t, uint64(111), got.Teleports)
	require.Equal(t, uint64(112), got.PushDistance)
	require.Equal(t, uint64(113), got.EnvironmentalDeaths)
	require.Equal(t, uint64(114), got.EnvironmentalKills)
	require.Equal(t, uint64(115), got.ObjectPlaced)
	require.Equal(t, uint64(116), got.ObjectUpgraded)
	require.Equal(t, uint64(117), got.ObjectCarried)
	require.Equal(t, uint64(118), got.ObjectDropped)
	require.Equal(t, uint64(119), got.ObjectRemoved)
	require.Equal(t, uint64(120), got.ObjectDetonated)
	require.Equal(t, uint64(121), got.AmmoPacks)
	require.Equal(t, uint64(122), got.HealthPacks)
	require.Equal(t, uint64(123), got.HealthPackHealing)

	byVariant := map[string]stats.MatchVariantStatsRound{}
	for _, variant := range match.Variants {
		byVariant[variant.Variant] = variant
	}

	require.Contains(t, byVariant, "soldier")
	require.Contains(t, byVariant, "tf_projectile_rocket")
	require.Equal(t, uint64(202), byVariant["soldier"].Heals)
	require.Equal(t, uint64(207), byVariant["soldier"].Extinguishes)
	require.Equal(t, uint64(225), byVariant["soldier"].HealthPackHealing)
	require.Equal(t, uint64(302), byVariant["tf_projectile_rocket"].Heals)
	require.Equal(t, uint64(307), byVariant["tf_projectile_rocket"].Extinguishes)
	require.Equal(t, uint64(325), byVariant["tf_projectile_rocket"].HealthPackHealing)

	require.Len(t, match.Kills, 2)
	requireMatchKills(t, match, steamID, victimID)
	requireNewParserStatsViews(ctx, t, testFixture.Database, repo, steamID)
}

func requireNewParserStatsViews(ctx context.Context, t *testing.T, db database.Database, repo stats.Repository, steamID steamid.SteamID) {
	t.Helper()

	// Exercise the allow-listed concurrent refresh path used by the daily job.
	require.NoError(t, db.RefreshMaterializedView(ctx, "stats_summary_daily_overall_view"))
	require.NoError(t, db.RefreshMaterializedView(ctx, "stats_summary_daily_variants_view"))

	overallRows, _, errOverall := repo.Query(ctx, stats.Opts{
		StatsBucketID: 1, TimeBucket: stats.TimeBucketDaily,
	})
	require.NoError(t, errOverall)

	var overall *stats.OverallStats

	for _, row := range overallRows {
		if stat, ok := row.(stats.OverallStats); ok && stat.SteamID.Equal(steamID) {
			stat := stat
			overall = &stat

			break
		}
	}

	require.NotNil(t, overall)
	require.Equal(t, uint64(101), overall.Heals)
	require.Equal(t, uint64(108), overall.Reflects)
	require.Equal(t, uint64(123), overall.HealthPackHealing)

	variantRows, _, errVariants := repo.Query(ctx, stats.Opts{
		StatsBucketID: 1, Variant: stats.VariantWeapons, VariantKey: "soldier",
		TimeBucket: stats.TimeBucketDaily,
	})
	require.NoError(t, errVariants)
	require.NotEmpty(t, variantRows)

	found := false

	for _, row := range variantRows {
		if stat, ok := row.(stats.VariantStats); ok && stat.SteamID.Equal(steamID) {
			require.Equal(t, uint64(202), stat.Heals)
			require.Equal(t, uint64(210), stat.Reflects)
			require.Equal(t, uint64(225), stat.HealthPackHealing)

			found = true
		}
	}

	require.True(t, found)
}

func requireMatchKills(t *testing.T, match *stats.Match, steamID steamid.SteamID, victimID steamid.SteamID) {
	t.Helper()

	pvpKill := match.Kills[0]
	require.Equal(t, 1000, pvpKill.Tick)
	require.True(t, pvpKill.HasKiller)
	require.True(t, pvpKill.KillerSteamID.Equal(steamID))
	require.True(t, pvpKill.VictimSteamID.Equal(victimID))
	require.Equal(t, "tf_projectile_rocket", pvpKill.Weapon)
	require.NotNil(t, pvpKill.KillerPos)
	require.InDelta(t, -5496, pvpKill.KillerPos.X, 0.001)
	require.InDelta(t, 5393.625, pvpKill.KillerPos.Y, 0.001)
	require.InDelta(t, 348, pvpKill.KillerPos.Z, 0.001)
	require.NotNil(t, pvpKill.VictimPos)
	require.InDelta(t, -5441.75, pvpKill.VictimPos.X, 0.001)
	require.NotNil(t, pvpKill.KillerAngles)
	require.InDelta(t, 26.47, pvpKill.KillerAngles.Pitch, 0.001)
	require.InDelta(t, 268.85, pvpKill.KillerAngles.Yaw, 0.001)
	require.NotNil(t, pvpKill.VictimAngles)
	require.InDelta(t, 8.82, pvpKill.VictimAngles.Pitch, 0.001)

	worldKill := match.Kills[1]
	require.Equal(t, 2000, worldKill.Tick)
	require.False(t, worldKill.HasKiller)
	require.True(t, worldKill.VictimSteamID.Equal(steamID))
	require.Equal(t, "world", worldKill.Weapon)
	require.Nil(t, worldKill.KillerPos)
	require.Nil(t, worldKill.VictimPos)
	require.Nil(t, worldKill.KillerAngles)
	require.Nil(t, worldKill.VictimAngles)
}

func TestImportKillRoundAndPositions(t *testing.T) {
	t.Parallel()
	testFixture := tests.NewFixture()
	defer testFixture.Close()

	ctx := t.Context()
	server := testFixture.CreateTestServer(ctx)

	killerID := steamid.New("[U:1:11111111]")
	victimID := steamid.New("[U:1:22222222]")
	require.True(t, killerID.Valid())
	require.True(t, victimID.Valid())

	_, killerFillersRound1 := fillerTestPlayers(5, 1000, 1500)
	_, killerFillersRound2 := fillerTestPlayers(5, 2000, 2500)

	persons := map[steamid.SteamID]string{
		killerID: "Killer",
		victimID: "Victim",
	}
	for i, filler := range killerFillersRound1 {
		persons[steamid.New(filler.SteamID)] = fmt.Sprintf("Filler%d", i)
	}
	insertTestPersons(ctx, t, testFixture.Database, persons)

	var demoID int32
	require.NoError(t, testFixture.Database.QueryRow(ctx,
		`INSERT INTO demo (server_id, title, map_name, created_on) VALUES ($1, $2, $3, $4) RETURNING demo_id`,
		server.ServerID, "kill-geometry.dem", "cp_process_final", time.Now()).Scan(&demoID))

	demo := demoparse.Demo{
		Filename: "kill-geometry.dem", DemoType: demoparse.HL2Demo,
		Server: "test", Map: "cp_process_final", Game: "tf",
		Duration: 600, Ticks: 40000, Frames: 39900, Signon: 1000,
		Rounds: []demoparse.RoundSummary{
			{
				Winner: "red", Time: 300,
				Players: append([]demoparse.PlayerSummary{
					{
						Name: "Killer", SteamID: string(killerID.Steam3()), Team: "red",
						TickStart: 1000, TickEnd: 1500,
					},
				}, killerFillersRound1...),
			},
			{
				Winner: "blu", Time: 300,
				Players: append([]demoparse.PlayerSummary{
					{
						Name: "Killer", SteamID: string(killerID.Steam3()), Team: "blu",
						TickStart: 2000, TickEnd: 2500,
					},
				}, killerFillersRound2...),
			},
		},
		Events: []demoparse.MatchEvent{
			{
				Tick: 1200, Type: demoparse.MatchEventKill,
				Kill: &demoparse.KillEvent{
					Tick: 1200, Killer: string(killerID.Steam3()), Victim: string(victimID.Steam3()),
					Weapon:       "tf_projectile_rocket",
					KillerPos:    &demoparse.Position{X: -5496, Y: 5393.625, Z: 348},
					VictimPos:    &demoparse.Position{X: -5441.75, Y: 5269.125, Z: 363.25},
					KillerAngles: &demoparse.EyeAngles{Pitch: 26.47, Yaw: 268.85},
					VictimAngles: &demoparse.EyeAngles{Pitch: 8.82, Yaw: 137.24},
				},
			},
			{
				Tick: 2200, Type: demoparse.MatchEventKill,
				Kill: &demoparse.KillEvent{
					Tick: 2200, Killer: string(killerID.Steam3()), Victim: string(victimID.Steam3()),
					Weapon:       "scattergun",
					KillerPos:    &demoparse.Position{X: 100.5, Y: -200.25, Z: 10},
					VictimPos:    &demoparse.Position{X: 150.75, Y: -250.5, Z: 20},
					KillerAngles: &demoparse.EyeAngles{Pitch: 0, Yaw: 90},
					VictimAngles: &demoparse.EyeAngles{Pitch: 0, Yaw: 270},
				},
			},
			{
				// A world kill outside every parsed round interval retains a NULL round.
				Tick: 99999, Type: demoparse.MatchEventKill,
				Kill: &demoparse.KillEvent{
					Tick: 99999, Victim: string(victimID.Steam3()), Weapon: "world",
				},
			},
		},
	}

	repo := stats.NewRepository(testFixture.Database)
	st := stats.New(repo, maps.New(maps.NewRepository(testFixture.Database)))
	matchID, errImport := st.Import(ctx, server.ServerID, demoID, &demo, time.Now())
	require.NoError(t, errImport)
	require.NotNil(t, matchID)

	roundRows, errRounds := testFixture.Database.Query(ctx,
		`SELECT round_id FROM match_round WHERE match_id = $1 ORDER BY round_id`,
		*matchID)
	require.NoError(t, errRounds)
	defer roundRows.Close()

	var roundIDs []int64
	for roundRows.Next() {
		var roundID int64
		require.NoError(t, roundRows.Scan(&roundID))
		roundIDs = append(roundIDs, roundID)
	}
	require.NoError(t, roundRows.Err())
	require.Len(t, roundIDs, 2)

	killRows, errKills := testFixture.Database.Query(ctx,
		`SELECT
			tick, round_id,
			ST_NDims(killer_position), ST_Zmflag(killer_position), ST_SRID(killer_position),
			ST_X(killer_position), ST_Y(killer_position), ST_Z(killer_position),
			ST_NDims(victim_position), ST_Zmflag(victim_position), ST_SRID(victim_position),
			ST_X(victim_position), ST_Y(victim_position), ST_Z(victim_position)
		FROM match_event
		WHERE match_id = $1 AND event_type = 'kill'
		ORDER BY tick`,
		*matchID)
	require.NoError(t, errKills)
	defer killRows.Close()

	type killPosition struct {
		dimensions *int32
		zmFlag     *int32
		srid       *int32
		x          *float64
		y          *float64
		z          *float64
	}

	type killRow struct {
		tick   int
		round  *int64
		killer killPosition
		victim killPosition
	}

	var kills []killRow
	for killRows.Next() {
		var kill killRow
		require.NoError(t, killRows.Scan(
			&kill.tick, &kill.round,
			&kill.killer.dimensions, &kill.killer.zmFlag, &kill.killer.srid,
			&kill.killer.x, &kill.killer.y, &kill.killer.z,
			&kill.victim.dimensions, &kill.victim.zmFlag, &kill.victim.srid,
			&kill.victim.x, &kill.victim.y, &kill.victim.z,
		))
		kills = append(kills, kill)
	}
	require.NoError(t, killRows.Err())
	require.Len(t, kills, 3)

	require.Equal(t, 1200, kills[0].tick)
	require.NotNil(t, kills[0].round)
	require.Equal(t, roundIDs[0], *kills[0].round)
	for _, position := range []killPosition{kills[0].killer, kills[0].victim} {
		require.NotNil(t, position.dimensions)
		require.Equal(t, int32(3), *position.dimensions)
		require.NotNil(t, position.zmFlag)
		require.Equal(t, int32(2), *position.zmFlag)
		require.NotNil(t, position.srid)
		require.Equal(t, int32(0), *position.srid)
	}
	require.InDelta(t, -5496, *kills[0].killer.x, 0.001)
	require.InDelta(t, 5393.625, *kills[0].killer.y, 0.001)
	require.InDelta(t, 348, *kills[0].killer.z, 0.001)
	require.InDelta(t, -5441.75, *kills[0].victim.x, 0.001)
	require.InDelta(t, 5269.125, *kills[0].victim.y, 0.001)
	require.InDelta(t, 363.25, *kills[0].victim.z, 0.001)

	require.Equal(t, 2200, kills[1].tick)
	require.NotNil(t, kills[1].round)
	require.Equal(t, roundIDs[1], *kills[1].round)
	require.NotNil(t, kills[1].killer.dimensions)
	require.Equal(t, int32(3), *kills[1].killer.dimensions)
	require.InDelta(t, 100.5, *kills[1].killer.x, 0.001)
	require.InDelta(t, -200.25, *kills[1].killer.y, 0.001)
	require.InDelta(t, 10, *kills[1].killer.z, 0.001)
	require.NotNil(t, kills[1].victim.dimensions)
	require.Equal(t, int32(3), *kills[1].victim.dimensions)
	require.InDelta(t, 150.75, *kills[1].victim.x, 0.001)
	require.InDelta(t, -250.5, *kills[1].victim.y, 0.001)
	require.InDelta(t, 20, *kills[1].victim.z, 0.001)

	require.Equal(t, 99999, kills[2].tick)
	require.Nil(t, kills[2].round)
	require.Nil(t, kills[2].killer.dimensions)
	require.Nil(t, kills[2].killer.x)
	require.Nil(t, kills[2].victim.dimensions)
	require.Nil(t, kills[2].victim.x)
}

func TestImportMatchEvents(t *testing.T) {
	t.Parallel()
	testFixture := tests.NewFixture()
	defer testFixture.Close()

	ctx := t.Context()
	server := testFixture.CreateTestServer(ctx)

	killerID := steamid.New("[U:1:33333333]")
	victimID := steamid.New("[U:1:44444444]")
	engineerID := steamid.New("[U:1:55555555]")
	require.True(t, killerID.Valid())
	require.True(t, victimID.Valid())
	require.True(t, engineerID.Valid())

	fillerIDs, fillerPlayers := fillerTestPlayers(4, 1000, 5000)

	persons := map[steamid.SteamID]string{
		killerID:   "Killer",
		victimID:   "Victim",
		engineerID: "Engineer",
	}
	for i, id := range fillerIDs {
		persons[id] = fmt.Sprintf("Filler%d", i)
	}
	insertTestPersons(ctx, t, testFixture.Database, persons)

	demoID := seedTestDemo(t, testFixture, server.ServerID, "match-events.dem", "cp_process_final")

	demo := demoparse.Demo{
		Filename: "match-events.dem", DemoType: demoparse.HL2Demo,
		Server: "test", Map: "cp_process_final", Game: "tf",
		Duration: 600, Ticks: 40000, Frames: 39900, Signon: 1000,
		Rounds: []demoparse.RoundSummary{
			{
				Winner: "red", Time: 300,
				Players: append([]demoparse.PlayerSummary{
					{
						Name: "Killer", SteamID: string(killerID.Steam3()), Team: "red",
						TickStart: 1000, TickEnd: 5000,
					},
					{
						Name: "Engineer", SteamID: string(engineerID.Steam3()), Team: "red",
						TickStart: 1000, TickEnd: 5000,
					},
				}, fillerPlayers...),
			},
		},
		Events: []demoparse.MatchEvent{
			{
				Tick: 1100, Type: demoparse.MatchEventKill,
				Kill: &demoparse.KillEvent{
					Tick: 1100, Killer: string(killerID.Steam3()), Victim: string(victimID.Steam3()),
					Weapon:    "scattergun",
					VictimPos: &demoparse.Position{X: 10, Y: 20, Z: 30},
				},
			},
			{
				Tick: 1200, Type: demoparse.MatchEventBuildingBuilt,
				BuildingBuilt: &demoparse.BuildingBuiltEvent{
					Owner: string(engineerID.Steam3()), Building: "sentry", Level: 3,
					Pos: demoparse.Position{X: 1, Y: 2, Z: 3},
				},
			},
			{
				Tick: 1300, Type: demoparse.MatchEventCaptureStarted,
				CaptureStarted: &demoparse.CaptureStartedEvent{
					Cp: 2, CpName: "cp_2", Team: 3, CapTeam: 3,
					Cappers: []string{string(killerID.Steam3()), string(engineerID.Steam3())},
					CapTime: 4.5,
				},
			},
			{
				Tick: 1400, Type: demoparse.MatchEventKillstreakEnded,
				KillstreakEnded: &demoparse.KillstreakEndedEvent{
					Player: string(killerID.Steam3()), Streak: 7, Killer: string(victimID.Steam3()),
				},
			},
			{
				Tick: 1500, Type: demoparse.MatchEventSetupFinished,
			},
			{
				// Invalid victims are stored but excluded from the kill projection.
				Tick: 1600, Type: demoparse.MatchEventKill,
				Kill: &demoparse.KillEvent{Tick: 1600, Victim: "BOT", Weapon: "world"},
			},
		},
	}

	repo := stats.NewRepository(testFixture.Database)
	st := stats.New(repo, maps.New(maps.NewRepository(testFixture.Database)))
	matchID, errImport := st.Import(ctx, server.ServerID, demoID, &demo, time.Now())
	require.NoError(t, errImport)
	require.NotNil(t, matchID)

	events := queryMatchEventTestRows(ctx, t, testFixture.Database, *matchID)
	requireMatchEventRows(t, events, killerID, victimID, engineerID)

	match, errMatch := repo.Match(ctx, *matchID)
	require.NoError(t, errMatch)
	require.Len(t, match.Kills, 1)
	require.True(t, match.Kills[0].VictimSteamID.Equal(victimID))

	byType := map[string]int{}
	for _, event := range match.Events {
		byType[event.Type]++
	}
	require.Equal(t, map[string]int{
		"kill": 2, "building_built": 1, "capture_started": 1,
		"killstreak_ended": 1, "setup_finished": 1,
	}, byType)
}

// matchEventTestRow mirrors the match_event projection asserted below.
type matchEventTestRow struct {
	eventType string
	tick      int
	actor     *int64
	target    *int64
	weapon    *string
	building  *string
	x         *float64
	y         *float64
	z         *float64
	details   string
}

func queryMatchEventTestRows(ctx context.Context, t *testing.T, db database.Database, matchID uuid.UUID) []matchEventTestRow {
	t.Helper()

	rows, err := db.Query(ctx,
		`SELECT event_type, tick, actor_steam_id, target_steam_id, weapon, building,
			ST_X(event_position), ST_Y(event_position), ST_Z(event_position),
			details
		FROM match_event
		WHERE match_id = $1
		ORDER BY tick`,
		matchID)
	require.NoError(t, err)
	defer rows.Close()

	var events []matchEventTestRow
	for rows.Next() {
		var event matchEventTestRow
		require.NoError(t, rows.Scan(
			&event.eventType, &event.tick, &event.actor, &event.target,
			&event.weapon, &event.building,
			&event.x, &event.y, &event.z,
			&event.details,
		))
		events = append(events, event)
	}
	require.NoError(t, rows.Err())

	return events
}

func requireMatchEventRows(t *testing.T, events []matchEventTestRow, killerID steamid.SteamID, victimID steamid.SteamID, engineerID steamid.SteamID) {
	t.Helper()

	require.Len(t, events, 6)

	require.Equal(t, "kill", events[0].eventType)
	require.NotNil(t, events[0].actor)
	require.Equal(t, killerID.Int64(), *events[0].actor)
	require.NotNil(t, events[0].target)
	require.Equal(t, victimID.Int64(), *events[0].target)
	require.NotNil(t, events[0].weapon)
	require.Equal(t, "scattergun", *events[0].weapon)

	require.Equal(t, "building_built", events[1].eventType)
	require.NotNil(t, events[1].actor)
	require.Equal(t, engineerID.Int64(), *events[1].actor)
	require.NotNil(t, events[1].building)
	require.Equal(t, "sentry", *events[1].building)
	require.NotNil(t, events[1].x)
	require.InDelta(t, 1, *events[1].x, 0.001)
	require.InDelta(t, 2, *events[1].y, 0.001)
	require.InDelta(t, 3, *events[1].z, 0.001)

	var builtDetails struct {
		Owner    string `json:"owner"`
		Building string `json:"building"`
		Level    int    `json:"level"`
		IsMini   bool   `json:"is_mini"` //nolint:tagliatelle // mirrors parser snake_case JSON.
		Pos      struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
			Z float64 `json:"z"`
		} `json:"pos"`
	}
	require.NoError(t, stdjson.Unmarshal([]byte(events[1].details), &builtDetails))
	require.Equal(t, string(engineerID.Steam3()), builtDetails.Owner)
	require.Equal(t, "sentry", builtDetails.Building)
	require.Equal(t, 3, builtDetails.Level)
	require.False(t, builtDetails.IsMini)
	require.InDelta(t, 1, builtDetails.Pos.X, 0.001)

	require.Equal(t, "capture_started", events[2].eventType)
	require.Nil(t, events[2].actor)

	var captureDetails struct {
		Cp      int      `json:"cp"`
		CpName  string   `json:"cp_name"` //nolint:tagliatelle // mirrors parser snake_case JSON.
		Cappers []string `json:"cappers"`
		CapTime float64  `json:"cap_time"` //nolint:tagliatelle // mirrors parser snake_case JSON.
	}
	require.NoError(t, stdjson.Unmarshal([]byte(events[2].details), &captureDetails))
	require.Equal(t, 2, captureDetails.Cp)
	require.Equal(t, "cp_2", captureDetails.CpName)
	require.Contains(t, captureDetails.Cappers, string(killerID.Steam3()))
	require.Contains(t, captureDetails.Cappers, string(engineerID.Steam3()))
	require.InDelta(t, 4.5, captureDetails.CapTime, 0.001)

	require.Equal(t, "killstreak_ended", events[3].eventType)
	require.NotNil(t, events[3].actor)
	require.Equal(t, victimID.Int64(), *events[3].actor)
	require.NotNil(t, events[3].target)
	require.Equal(t, killerID.Int64(), *events[3].target)

	require.Equal(t, "setup_finished", events[4].eventType)
	require.Equal(t, "{}", events[4].details)

	require.Equal(t, "kill", events[5].eventType)
	require.Nil(t, events[5].target)
}

// TestImportSkipsUnknownMatchEvents ensures demos containing event types from
// a newer parser version don't fail the whole import (previously surfaced as
// an internal server error on manual demo upload). Unknown events are
// skipped while known events are still stored.
func TestImportSkipsUnknownMatchEvents(t *testing.T) {
	t.Parallel()
	testFixture := tests.NewFixture()
	defer testFixture.Close()

	ctx := t.Context()
	server := testFixture.CreateTestServer(ctx)

	killerID := steamid.New("[U:1:33333333]")
	victimID := steamid.New("[U:1:44444444]")
	require.True(t, killerID.Valid())
	require.True(t, victimID.Valid())

	fillerIDs, fillerPlayers := fillerTestPlayers(5, 1000, 5000)

	persons := map[steamid.SteamID]string{
		killerID: "Killer",
		victimID: "Victim",
	}
	for i, id := range fillerIDs {
		persons[id] = fmt.Sprintf("Filler%d", i)
	}
	insertTestPersons(ctx, t, testFixture.Database, persons)

	var demoID int32
	require.NoError(t, testFixture.Database.QueryRow(ctx,
		`INSERT INTO demo (server_id, title, map_name, created_on) VALUES ($1, $2, $3, $4) RETURNING demo_id`,
		server.ServerID, "unknown-events.dem", "cp_process_final", time.Now()).Scan(&demoID))

	demo := demoparse.Demo{
		Filename: "unknown-events.dem", DemoType: demoparse.HL2Demo,
		Server: "test", Map: "cp_process_final", Game: "tf",
		Duration: 600, Ticks: 40000, Frames: 39900, Signon: 1000,
		Rounds: []demoparse.RoundSummary{
			{
				Winner: "red", Time: 300,
				Players: append([]demoparse.PlayerSummary{
					{
						Name: "Killer", SteamID: string(killerID.Steam3()), Team: "red",
						TickStart: 1000, TickEnd: 5000,
					},
				}, fillerPlayers...),
			},
		},
		Events: []demoparse.MatchEvent{
			{
				Tick: 1100, Type: demoparse.MatchEventKill,
				Kill: &demoparse.KillEvent{
					Tick: 1100, Killer: string(killerID.Steam3()), Victim: string(victimID.Steam3()),
					Weapon: "scattergun",
				},
			},
			{Tick: 1200, Type: demoparse.MatchEventType("some_future_event")},
			{Tick: 1300},
		},
	}

	repo := stats.NewRepository(testFixture.Database)
	st := stats.New(repo, maps.New(maps.NewRepository(testFixture.Database)))
	matchID, errImport := st.Import(ctx, server.ServerID, demoID, &demo, time.Now())
	require.NoError(t, errImport)
	require.NotNil(t, matchID)

	var count int
	require.NoError(t, testFixture.Database.QueryRow(ctx,
		`SELECT count(*) FROM match_event WHERE match_id = $1`, *matchID).Scan(&count))
	require.Equal(t, 1, count)
}

func validTestDemo(duration float64, numPlayers int) *demoparse.Demo {
	_, players := fillerTestPlayers(numPlayers, 1000, 5000)

	return &demoparse.Demo{
		Filename: "validate.dem", DemoType: demoparse.HL2Demo,
		Server: "test", Map: "cp_process_final", Game: "tf",
		Duration: duration,
		Rounds: []demoparse.RoundSummary{
			{Winner: "red", Time: 300, Players: players},
		},
	}
}

func TestValidateDemo(t *testing.T) {
	require.NoError(t, stats.ValidateDemo(validTestDemo(3600, 6)))
	require.NoError(t, stats.ValidateDemo(validTestDemo(7200, 6)))
	require.Equal(t, 6, stats.ValidPlayerCount(validTestDemo(3600, 6)))

	tooFew := validTestDemo(3600, 5)
	require.ErrorIs(t, stats.ValidateDemo(tooFew), stats.ErrMatchFiltered)
	require.ErrorIs(t, stats.ValidateDemo(tooFew), stats.ErrNotEnoughPlayers)

	tooLong := validTestDemo(7201, 6)
	require.ErrorIs(t, stats.ValidateDemo(tooLong), stats.ErrMatchFiltered)
	require.ErrorIs(t, stats.ValidateDemo(tooLong), stats.ErrMatchTooLong)

	fake := validTestDemo(3600, 5)
	fake.Rounds[0].Players = append(fake.Rounds[0].Players,
		demoparse.PlayerSummary{Name: "BOT", SteamID: "BOT", Team: "red"},
		demoparse.PlayerSummary{Name: "TV", SteamID: "[U:1:90000999]", Team: "red", IsHlTv: true},
		demoparse.PlayerSummary{Name: "Replay", SteamID: "[U:1:90000998]", Team: "red", IsReplay: true},
		demoparse.PlayerSummary{Name: "Fake", SteamID: "[U:1:90000997]", Team: "red", IsFakePlayer: true},
	)
	require.Equal(t, 5, stats.ValidPlayerCount(fake))
	require.ErrorIs(t, stats.ValidateDemo(fake), stats.ErrNotEnoughPlayers)
}

func TestImportRejectsFilteredDemos(t *testing.T) {
	t.Parallel()
	testFixture := tests.NewFixture()
	defer testFixture.Close()

	ctx := t.Context()
	server := testFixture.CreateTestServer(ctx)

	repo := stats.NewRepository(testFixture.Database)
	statSvc := stats.New(repo, maps.New(maps.NewRepository(testFixture.Database)))

	newDemoID := func() int32 {
		var demoID int32
		require.NoError(t, testFixture.Database.QueryRow(ctx,
			`INSERT INTO demo (server_id, title, map_name, created_on) VALUES ($1, $2, $3, $4) RETURNING demo_id`,
			server.ServerID, "filtered.dem", "cp_process_final", time.Now()).Scan(&demoID))

		return demoID
	}

	_, errFew := statSvc.Import(ctx, server.ServerID, newDemoID(), validTestDemo(3600, 5), time.Now())
	require.ErrorIs(t, errFew, stats.ErrMatchFiltered)
	require.ErrorIs(t, errFew, stats.ErrNotEnoughPlayers)

	_, errLong := statSvc.Import(ctx, server.ServerID, newDemoID(), validTestDemo(7201, 6), time.Now())
	require.ErrorIs(t, errLong, stats.ErrMatchFiltered)
	require.ErrorIs(t, errLong, stats.ErrMatchTooLong)

	var count int
	require.NoError(t, testFixture.Database.QueryRow(ctx,
		`SELECT count(*) FROM match`).Scan(&count))
	require.Equal(t, 0, count)
}

type stubPermissionResolver struct{}

func (stubPermissionResolver) PermissionsBySteamID(_ context.Context, _ steamid.SteamID) []rolesv1.Permission {
	return nil
}

func newTestStatsClient(t *testing.T, fixture *tests.Fixture, user steamid.SteamID) (statsv1connect.StatsServiceClient, func()) {
	t.Helper()

	middleware := rpc.NewMiddleware("gbans-test", "test-cookie-secret", stubPermissionResolver{})
	serversCase, errServers := servers.New(servers.NewRepository(fixture.Database), nil, "")
	require.NoError(t, errServers)

	svc := stats.NewService(
		stats.New(stats.NewRepository(fixture.Database), maps.New(maps.NewRepository(fixture.Database))),
		serversCase,
		opengraph.NewRepository(opengraph.Options{}),
		rpc.NewRoleAuth(stubPermissionResolver{}),
		middleware,
	)

	mux := http.NewServeMux()
	mux.Handle(svc.Pattern, svc.Handler)
	server := httptest.NewServer(authn.NewMiddleware(func(_ context.Context, _ *http.Request) (any, error) {
		return rpc.UserInfo{SteamID: user}, nil
	}).Wrap(mux))

	return statsv1connect.NewStatsServiceClient(server.Client(), server.URL), server.Close
}

func seedTestDemo(t *testing.T, fixture *tests.Fixture, serverID int32, title, mapName string) int32 {
	t.Helper()

	var demoID int32
	require.NoError(t, fixture.Database.QueryRow(t.Context(),
		`INSERT INTO demo (server_id, title, map_name, created_on) VALUES ($1, $2, $3, now()) RETURNING demo_id`,
		serverID, title, mapName).Scan(&demoID))

	return demoID
}

func seedTestMap(t *testing.T, fixture *tests.Fixture, name string) int32 {
	t.Helper()

	var mapID int32
	require.NoError(t, fixture.Database.QueryRow(t.Context(),
		`INSERT INTO map (map_name, updated_on, created_on) VALUES ($1, now(), now()) RETURNING map_id`,
		name).Scan(&mapID))

	return mapID
}

func seedTestAsset(t *testing.T, fixture *tests.Fixture, authorID steamid.SteamID) uuid.UUID {
	t.Helper()

	assetID, errID := uuid.NewV4()
	require.NoError(t, errID)

	require.NoError(t, fixture.Database.Exec(t.Context(), `
		INSERT INTO asset (asset_id, bucket, mime_type, size, name, hash, author_id, created_on, updated_on)
		VALUES ($1, $2, $3, 1, $4, $5, $6, now(), now())`,
		assetID, string(asset.BucketDemo), "application/octet-stream", "test.dem.zst", []byte{0}, authorID.Int64()))

	return assetID
}

func seedTestMatch(t *testing.T, fixture *tests.Fixture, serverID, mapID, demoID, bucketID int32, start time.Time) uuid.UUID {
	t.Helper()

	matchID, errMatch := uuid.NewV4()
	require.NoError(t, errMatch)

	require.NoError(t, fixture.Database.Exec(t.Context(), `
		INSERT INTO match (
			match_id, server_id, map_id, demo_id, stats_bucket_id, hostname,
			score_red, score_blu, start_time, duration_ms, created_on
		) VALUES ($1, $2, $3, $4, $5, $6, 1, 2, $7, 120000, now())`,
		matchID, serverID, mapID, demoID, bucketID, "test-host", start))

	return matchID
}

// TestQueryMatches exercises the match history query end to end, covering the
// server-side column filters and the bucket ordering fallback.
//
//nolint:tparallel // subtests share the parent fixture and one prunes a demo asset the earlier subtests read
func TestQueryMatches(t *testing.T) {
	t.Parallel()
	fixture := tests.NewFixture()
	defer fixture.Close()

	client, cleanup := newTestStatsClient(t, fixture, steamid.RandSID64())
	defer cleanup()

	ctx := t.Context()
	serverA := fixture.CreateTestServer(ctx)
	serverB := fixture.CreateTestServer(ctx)
	mapA := seedTestMap(t, fixture, "cp_filter_a")
	mapB := seedTestMap(t, fixture, "cp_filter_b")
	demoA := seedTestDemo(t, fixture, serverA.ServerID, "filter-demo-a", "cp_filter_demo")
	demoB := seedTestDemo(t, fixture, serverB.ServerID, "filter-demo-b", "cp_filter_demo")

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	// Buckets 1 (casual) and 2 (1ku) are seeded by the migrations.
	firstMatchID := seedTestMatch(t, fixture, serverA.ServerID, mapA, demoA, 1, base)
	seedTestMatch(t, fixture, serverB.ServerID, mapA, demoA, 1, base.Add(time.Hour))
	seedTestMatch(t, fixture, serverA.ServerID, mapB, demoA, 2, base.Add(2*time.Hour))
	seedTestMatch(t, fixture, serverB.ServerID, mapB, demoB, 2, base.Add(3*time.Hour))

	queryMatches := func(t *testing.T, request *statsv1.QueryMatchesRequest) *statsv1.QueryMatchesResponse {
		t.Helper()

		resp, err := client.QueryMatches(ctx, request)
		require.NoError(t, err)

		return resp
	}

	t.Run("no filters", func(t *testing.T) {
		resp := queryMatches(t, &statsv1.QueryMatchesRequest{Filter: &queryv1.Filter{Limit: proto.Uint64(100)}})
		require.Equal(t, uint64(4), resp.GetCount())
		require.Len(t, resp.GetMatches(), 4)
	})

	t.Run("single server", func(t *testing.T) {
		resp := queryMatches(t, &statsv1.QueryMatchesRequest{
			ServerId: []int32{serverA.ServerID},
			Filter:   &queryv1.Filter{Limit: proto.Uint64(100)},
		})
		require.Equal(t, uint64(2), resp.GetCount())
		for _, match := range resp.GetMatches() {
			require.Equal(t, serverA.ServerID, match.GetServerId())
		}
	})

	t.Run("multiple servers", func(t *testing.T) {
		resp := queryMatches(t, &statsv1.QueryMatchesRequest{
			ServerId: []int32{serverA.ServerID, serverB.ServerID},
			Filter:   &queryv1.Filter{Limit: proto.Uint64(100)},
		})
		require.Equal(t, uint64(4), resp.GetCount())
	})

	t.Run("map", func(t *testing.T) {
		resp := queryMatches(t, &statsv1.QueryMatchesRequest{
			MapId:  new(mapB),
			Filter: &queryv1.Filter{Limit: proto.Uint64(100)},
		})
		require.Equal(t, uint64(2), resp.GetCount())
		for _, match := range resp.GetMatches() {
			require.Equal(t, mapB, match.GetMap().GetMapId())
		}
	})

	t.Run("bucket", func(t *testing.T) {
		resp := queryMatches(t, &statsv1.QueryMatchesRequest{
			StatsBucketId: proto.Uint32(2),
			Filter:        &queryv1.Filter{Limit: proto.Uint64(100)},
		})
		require.Equal(t, uint64(2), resp.GetCount())
		for _, match := range resp.GetMatches() {
			require.Equal(t, int32(2), match.GetStatsBucketId())
		}
	})

	t.Run("start time after", func(t *testing.T) {
		resp := queryMatches(t, &statsv1.QueryMatchesRequest{
			StartTimeStart: timestamppb.New(base.Add(90 * time.Minute)),
			Filter:         &queryv1.Filter{Limit: proto.Uint64(100)},
		})
		require.Equal(t, uint64(2), resp.GetCount())
		for _, match := range resp.GetMatches() {
			require.True(t, match.GetStartTime().AsTime().After(base.Add(90*time.Minute)))
		}
	})

	t.Run("start time before", func(t *testing.T) {
		resp := queryMatches(t, &statsv1.QueryMatchesRequest{
			StartTimeEnd: timestamppb.New(base.Add(90 * time.Minute)),
			Filter:       &queryv1.Filter{Limit: proto.Uint64(100)},
		})
		require.Equal(t, uint64(2), resp.GetCount())
		for _, match := range resp.GetMatches() {
			require.True(t, match.GetStartTime().AsTime().Before(base.Add(90*time.Minute)))
		}
	})

	t.Run("start time range", func(t *testing.T) {
		resp := queryMatches(t, &statsv1.QueryMatchesRequest{
			StartTimeStart: timestamppb.New(base.Add(90 * time.Minute)),
			StartTimeEnd:   timestamppb.New(base.Add(150 * time.Minute)),
			Filter:         &queryv1.Filter{Limit: proto.Uint64(100)},
		})
		require.Equal(t, uint64(1), resp.GetCount())
		require.Equal(t, mapB, resp.GetMatches()[0].GetMap().GetMapId())
	})

	t.Run("combined server and map", func(t *testing.T) {
		resp := queryMatches(t, &statsv1.QueryMatchesRequest{
			ServerId: []int32{serverA.ServerID},
			MapId:    new(mapB),
			Filter:   &queryv1.Filter{Limit: proto.Uint64(100)},
		})
		require.Equal(t, uint64(1), resp.GetCount())
		require.Equal(t, mapB, resp.GetMatches()[0].GetMap().GetMapId())
	})

	t.Run("order by bucket name", func(t *testing.T) {
		resp := queryMatches(t, &statsv1.QueryMatchesRequest{
			Filter: &queryv1.Filter{
				Limit:   proto.Uint64(100),
				OrderBy: new("stats_bucket_name"),
			},
		})
		require.Equal(t, uint64(4), resp.GetCount())
		require.Equal(t, int32(1), resp.GetMatches()[0].GetStatsBucketId())
		require.Equal(t, int32(2), resp.GetMatches()[3].GetStatsBucketId())
	})

	t.Run("pruned demo asset", func(t *testing.T) {
		ownerSID := steamid.New(fixture.Config.Config().Owner)
		require.NoError(t, fixture.Persons.EnsurePerson(ctx, ownerSID))

		assetID := seedTestAsset(t, fixture, ownerSID)
		require.NoError(t, fixture.Database.Exec(ctx,
			`UPDATE demo SET asset_id = $1 WHERE demo_id = $2`, assetID, demoA))

		getOverview := func(t *testing.T) *statsv1.MatchOverview {
			t.Helper()

			resp, err := client.Match(ctx, &statsv1.MatchRequest{MatchId: new(firstMatchID.String())})
			require.NoError(t, err)

			return resp.GetMatch().GetOverview()
		}

		require.Equal(t, assetID.String(), getOverview(t).GetAssetId())

		require.NoError(t, fixture.Database.Exec(ctx,
			`UPDATE asset SET deleted = true WHERE asset_id = $1`, assetID))

		require.Empty(t, getOverview(t).GetAssetId())
	})
}

// TestQueryStatsMyRank ensures the requesting user's rank is resolved against
// the full bucket window, not just the returned page, and stays 0 when the
// user has no stats in the window.
//
//nolint:tparallel // subtests share the parent fixture and its seeded matches, so they must stay sequential
func TestQueryStatsMyRank(t *testing.T) {
	t.Parallel()
	fixture := tests.NewFixture()
	defer fixture.Close()

	ctx := t.Context()

	demoJSON, err := os.Open("testdata/demo-1427611.json")
	require.NoError(t, err)
	demo, errDemo := json.Decode[demoparse.Demo](demoJSON)
	require.NoError(t, errDemo)

	server := fixture.CreateTestServer(ctx)

	repo := stats.NewRepository(fixture.Database)
	statsCase := stats.New(repo, maps.New(maps.NewRepository(fixture.Database)))

	for _, round := range demo.Rounds {
		for _, player := range round.Players {
			sid := steamid.New(player.SteamID)
			if !sid.Valid() {
				continue
			}
			require.NoError(t, fixture.Database.Exec(ctx,
				`INSERT INTO person (steam_id, created_on, updated_on, personaname, avatarhash, profilestate, personastate,
				                    realname, timecreated, loccountrycode, locstatecode, loccityid)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
				 ON CONFLICT DO NOTHING`,
				sid.Int64(), time.Now(), time.Now(), player.Name, fmt.Sprintf("avatar-%d", sid.Int64()), 0, 0, "", 0, "", "", 0))
		}
	}

	importDemoAt(t, fixture, statsCase, server.ServerID, &demo, time.Now())
	require.NoError(t, statsCase.RefreshViews(ctx))

	playerSums := sumDemoPlayers(&demo)
	require.NotEmpty(t, playerSums)
	// The leaderboard only ranks players meeting the minimum kill count, so
	// pick a user that is eligible for the default (overall) view.
	userSID := steamid.SteamID{}
	for sid, sums := range playerSums {
		if sums.kills < stats.MinKills {
			continue
		}
		userSID = sid

		break
	}
	require.True(t, userSID.Valid())

	opts := stats.Opts{StatsBucketID: 1}
	expectedRows, _, errQuery := repo.Query(ctx, opts)
	require.NoError(t, errQuery)

	// Rows must be ordered by (rank, steam_id) so my_index is a stable
	// position in the paged result set.
	expectedRank := uint64(0)
	expectedIndex := uint64(0)
	var prev stats.OverallStats
	for idx, row := range expectedRows {
		overall, ok := row.(stats.OverallStats)
		require.True(t, ok)
		if idx > 0 {
			require.GreaterOrEqual(t, overall.Rank, prev.Rank)
			if overall.Rank == prev.Rank {
				require.Greater(t, overall.SteamID.Int64(), prev.SteamID.Int64(),
					"tied ranks must be ordered by steam_id")
			}
		}
		prev = overall

		if overall.SteamID != userSID {
			continue
		}
		expectedRank = overall.Rank
		expectedIndex = uint64(idx)
	}
	require.Positive(t, expectedRank)

	queryStats := func(t *testing.T, user steamid.SteamID) *statsv1.QueryStatsResponse {
		t.Helper()

		client, cleanup := newTestStatsClient(t, fixture, user)
		t.Cleanup(cleanup)

		resp, errResp := client.QueryStats(t.Context(), &statsv1.QueryStatsRequest{
			StatsBucketId: new(uint32(1)),
			// A single-row page: myRank must still be the full-window rank.
			Filter: &queryv1.Filter{Limit: proto.Uint64(1)},
		})
		require.NoError(t, errResp)

		return resp
	}

	t.Run("user with stats", func(t *testing.T) {
		resp := queryStats(t, userSID)
		require.Positive(t, resp.GetCount())
		require.Equal(t, expectedRank, resp.GetMyRank())
		require.Equal(t, expectedIndex, resp.GetMyIndex())
	})

	t.Run("user without stats", func(t *testing.T) {
		resp := queryStats(t, steamid.RandSID64())
		require.Zero(t, resp.GetMyRank())
		require.Zero(t, resp.GetMyIndex())
	})
}

// TestQueryStatsPlayerRank ensures a player requested via steam_id gets
// player_rank/player_index resolved against the full bucket window,
// independently of the requesting user, and stays 0 when the player has no
// stats in the window.
//
//nolint:tparallel // subtests share the parent fixture and its seeded matches, so they must stay sequential
func TestQueryStatsPlayerRank(t *testing.T) {
	t.Parallel()
	fixture := tests.NewFixture()
	defer fixture.Close()

	ctx := t.Context()

	demoJSON, err := os.Open("testdata/demo-1427611.json")
	require.NoError(t, err)
	demo, errDemo := json.Decode[demoparse.Demo](demoJSON)
	require.NoError(t, errDemo)

	server := fixture.CreateTestServer(ctx)

	repo := stats.NewRepository(fixture.Database)
	statsCase := stats.New(repo, maps.New(maps.NewRepository(fixture.Database)))

	for _, round := range demo.Rounds {
		for _, player := range round.Players {
			sid := steamid.New(player.SteamID)
			if !sid.Valid() {
				continue
			}
			require.NoError(t, fixture.Database.Exec(ctx,
				`INSERT INTO person (steam_id, created_on, updated_on, personaname, avatarhash, profilestate, personastate,
				                    realname, timecreated, loccountrycode, locstatecode, loccityid)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
				 ON CONFLICT DO NOTHING`,
				sid.Int64(), time.Now(), time.Now(), player.Name, fmt.Sprintf("avatar-%d", sid.Int64()), 0, 0, "", 0, "", "", 0))
		}
	}

	importDemoAt(t, fixture, statsCase, server.ServerID, &demo, time.Now())
	require.NoError(t, statsCase.RefreshViews(ctx))

	playerSums := sumDemoPlayers(&demo)
	require.NotEmpty(t, playerSums)

	// Two distinct eligible players: one signs the request, the other is
	// searched. Players below the leaderboard's minimum kills have no rank.
	var userSID, searchedSID steamid.SteamID
	found := 0
	for sid, sums := range playerSums {
		if sums.kills < stats.MinKills {
			continue
		}
		switch found {
		case 0:
			userSID = sid
		case 1:
			searchedSID = sid
		}
		found++
	}
	require.GreaterOrEqual(t, found, 2)

	opts := stats.Opts{StatsBucketID: 1}
	expectedRows, _, errQuery := repo.Query(ctx, opts)
	require.NoError(t, errQuery)

	rankAt := func(want steamid.SteamID) (uint64, uint64) {
		for idx, row := range expectedRows {
			overall, ok := row.(stats.OverallStats)
			require.True(t, ok)
			if overall.SteamID != want {
				continue
			}

			return overall.Rank, uint64(idx)
		}

		return 0, 0
	}

	userRank, userIndex := rankAt(userSID)
	searchedRank, searchedIndex := rankAt(searchedSID)
	require.Positive(t, userRank)
	require.Positive(t, searchedRank)

	queryStats := func(t *testing.T, user steamid.SteamID, steamID *int64) *statsv1.QueryStatsResponse {
		t.Helper()

		client, cleanup := newTestStatsClient(t, fixture, user)
		t.Cleanup(cleanup)

		resp, errResp := client.QueryStats(t.Context(), &statsv1.QueryStatsRequest{
			StatsBucketId: new(uint32(1)),
			SteamId:       steamID,
			// A single-row page: both ranks must be full-window values.
			Filter: &queryv1.Filter{Limit: proto.Uint64(1)},
		})
		require.NoError(t, errResp)

		return resp
	}

	searchedID := searchedSID.Int64()
	userID := userSID.Int64()

	t.Run("requested player independent of user", func(t *testing.T) {
		resp := queryStats(t, userSID, &searchedID)
		require.Equal(t, searchedRank, resp.GetPlayerRank())
		require.Equal(t, searchedIndex, resp.GetPlayerIndex())
		require.Equal(t, userRank, resp.GetMyRank())
		require.Equal(t, userIndex, resp.GetMyIndex())
	})

	t.Run("requested player is the user", func(t *testing.T) {
		resp := queryStats(t, userSID, &userID)
		require.Equal(t, userRank, resp.GetPlayerRank())
		require.Equal(t, userIndex, resp.GetPlayerIndex())
	})

	t.Run("no steam_id requested", func(t *testing.T) {
		resp := queryStats(t, userSID, nil)
		require.Zero(t, resp.GetPlayerRank())
		require.Zero(t, resp.GetPlayerIndex())
		require.Positive(t, resp.GetMyRank())
	})

	t.Run("player without stats", func(t *testing.T) {
		unknown := steamid.RandSID64()
		unknownID := unknown.Int64()
		resp := queryStats(t, userSID, &unknownID)
		require.Zero(t, resp.GetPlayerRank())
		require.Zero(t, resp.GetPlayerIndex())
		require.Equal(t, userRank, resp.GetMyRank())
	})
}

// TestQueryStatsMyIndexTies ensures players tied at the same rank page
// deterministically (rank, then steam_id) and that my_index identifies the
// exact position of the requesting user's row within the tied group.
//
//nolint:tparallel // subtests share the parent fixture and its imported demo, so they must stay sequential
func TestQueryStatsMyIndexTies(t *testing.T) {
	t.Parallel()
	fixture := tests.NewFixture()
	defer fixture.Close()

	ctx := t.Context()
	repo := stats.NewRepository(fixture.Database)
	statsCase := stats.New(repo, maps.New(maps.NewRepository(fixture.Database)))

	server := fixture.CreateTestServer(ctx)

	// Three tied players in a dedicated steamid range, one player below them,
	// and fillers to satisfy the MinPlayers import threshold. Everyone gets
	// the minimum kill count so they are eligible for the leaderboard.
	aSID := steamid.New("[U:1:92000001]")
	bSID := steamid.New("[U:1:92000002]")
	cSID := steamid.New("[U:1:92000003]")
	dSID := steamid.New("[U:1:92000004]")
	fillerIDs, fillerPlayers := fillerTestPlayers(2, 1000, 5000)
	for idx := range fillerPlayers {
		fillerPlayers[idx].Kills = stats.MinKills
	}

	persons := map[steamid.SteamID]string{
		aSID: "Tie A", bSID: "Tie B", cSID: "Tie C", dSID: "Fourth",
	}
	for _, fillerID := range fillerIDs {
		persons[fillerID] = "Filler"
	}
	insertTestPersons(ctx, t, fixture.Database, persons)

	var demoID int32
	require.NoError(t, fixture.Database.QueryRow(ctx,
		`INSERT INTO demo (server_id, title, map_name, created_on) VALUES ($1, $2, $3, now()) RETURNING demo_id`,
		server.ServerID, "ties.dem", "cp_ties").Scan(&demoID))

	demo := &demoparse.Demo{
		Filename: "ties.dem", DemoType: demoparse.HL2Demo,
		Server: "test", Map: "cp_ties", Game: "tf",
		Duration: 600, Ticks: 40000, Frames: 39900, Signon: 1000,
		Rounds: []demoparse.RoundSummary{
			{
				Winner: "red", Time: 300,
				Players: append([]demoparse.PlayerSummary{
					{Name: "Tie A", SteamID: string(aSID.Steam3()), Team: "red", TickStart: 1000, TickEnd: 5000, Kills: stats.MinKills, Points: 10},
					{Name: "Tie B", SteamID: string(bSID.Steam3()), Team: "red", TickStart: 1000, TickEnd: 5000, Kills: stats.MinKills, Points: 10},
					{Name: "Tie C", SteamID: string(cSID.Steam3()), Team: "red", TickStart: 1000, TickEnd: 5000, Kills: stats.MinKills, Points: 10},
					{Name: "Fourth", SteamID: string(dSID.Steam3()), Team: "blue", TickStart: 1000, TickEnd: 5000, Kills: stats.MinKills, Points: 5},
				}, fillerPlayers...),
			},
		},
	}

	_, errImport := statsCase.Import(ctx, server.ServerID, demoID, demo, time.Now())
	require.NoError(t, errImport)
	require.NoError(t, statsCase.RefreshViews(ctx))

	var bucketID uint32
	buckets, errBuckets := statsCase.Buckets(ctx)
	require.NoError(t, errBuckets)
	for _, b := range buckets {
		if b.IsDefault {
			bucketID = uint32(b.BucketID) //nolint:gosec // bucket ids are positive.
		}
	}
	require.NotZero(t, bucketID)

	rows, _, errQuery := repo.Query(ctx, stats.Opts{StatsBucketID: bucketID})
	require.NoError(t, errQuery)

	// Tied players page by steam_id: A(0), B(1), C(2), D(3), fillers(4, 5).
	require.Len(t, rows, 6)
	expectedOrder := []steamid.SteamID{aSID, bSID, cSID, dSID}
	for idx, want := range expectedOrder {
		overall, ok := rows[idx].(stats.OverallStats)
		require.True(t, ok)
		require.Equal(t, want, overall.SteamID, "row %d must be ordered by (rank, steam_id)", idx)
	}
	topRow, okTop := rows[0].(stats.OverallStats)
	require.True(t, okTop)
	require.Equal(t, uint64(1), topRow.Rank)

	fourthRow, okFourth := rows[3].(stats.OverallStats)
	require.True(t, okFourth)
	require.Equal(t, uint64(4), fourthRow.Rank)

	queryStats := func(t *testing.T, user steamid.SteamID, offset uint64) *statsv1.QueryStatsResponse {
		t.Helper()

		client, cleanup := newTestStatsClient(t, fixture, user)
		t.Cleanup(cleanup)

		resp, errResp := client.QueryStats(t.Context(), &statsv1.QueryStatsRequest{
			StatsBucketId: new(bucketID),
			Filter:        &queryv1.Filter{Limit: proto.Uint64(1), Offset: new(offset)},
		})
		require.NoError(t, errResp)

		return resp
	}

	t.Run("tied user index", func(t *testing.T) {
		resp := queryStats(t, bSID, 0)
		require.Equal(t, uint64(1), resp.GetMyRank())
		require.Equal(t, uint64(1), resp.GetMyIndex())
	})

	t.Run("index matches paged row", func(t *testing.T) {
		resp := queryStats(t, bSID, 1)
		require.Equal(t, uint64(1), resp.GetMyIndex())

		// The page at my_index must actually be the user's row.
		stats := resp.GetStatsOverall().GetStats()
		require.Len(t, stats, 1)
		require.Equal(t, bSID.Int64(), stats[0].GetPlayer().GetSteamId())
	})

	t.Run("user below the tie", func(t *testing.T) {
		resp := queryStats(t, dSID, 0)
		require.Equal(t, uint64(4), resp.GetMyRank())
		require.Equal(t, uint64(3), resp.GetMyIndex())
	})

	t.Run("user without stats", func(t *testing.T) {
		resp := queryStats(t, steamid.RandSID64(), 0)
		require.Zero(t, resp.GetMyRank())
		require.Zero(t, resp.GetMyIndex())
	})
}

// TestDefaultBucket ensures exactly one bucket is marked as default and that
// matches imported for a server without a bucket of its own are filed under
// that bucket.
func TestDefaultBucket(t *testing.T) {
	t.Parallel()
	testFixture := tests.NewFixture()
	defer testFixture.Close()

	repo := stats.NewRepository(testFixture.Database)
	statsCase := stats.New(repo, maps.New(maps.NewRepository(testFixture.Database)))
	ctx := t.Context()

	buckets, errBuckets := statsCase.Buckets(ctx)
	require.NoError(t, errBuckets)

	var defaultBucket *stats.Bucket
	for _, bucket := range buckets {
		if !bucket.IsDefault {
			continue
		}
		require.Nil(t, defaultBucket, "expected at most one default bucket")
		defaultBucket = &bucket
	}
	require.NotNil(t, defaultBucket, "expected a default bucket to be configured")
	require.Equal(t, "casual", defaultBucket.BucketName)

	// Servers saved through the domain always get a bucket, so clear it to
	// exercise the no-bucket fallback.
	server := testFixture.CreateTestServer(ctx)
	require.NoError(t, testFixture.Database.Exec(ctx,
		`UPDATE server SET stats_bucket_id = NULL WHERE server_id = $1`, server.ServerID))

	var demoID int32
	require.NoError(t, testFixture.Database.QueryRow(ctx,
		`INSERT INTO demo (server_id, title, map_name, created_on) VALUES ($1, $2, $3, now()) RETURNING demo_id`,
		server.ServerID, "default-bucket.dem", "cp_badlands").Scan(&demoID))

	playerID := steamid.RandSID64()
	fillerIDs, fillerPlayers := fillerTestPlayers(5, 1000, 5000)

	persons := map[steamid.SteamID]string{playerID: "Player"}
	for _, fillerID := range fillerIDs {
		persons[fillerID] = "Filler"
	}
	insertTestPersons(ctx, t, testFixture.Database, persons)

	demo := &demoparse.Demo{
		Filename: "default-bucket.dem", DemoType: demoparse.HL2Demo,
		Server: "test", Map: "cp_badlands", Game: "tf",
		Duration: 600, Ticks: 40000, Frames: 39900, Signon: 1000,
		Rounds: []demoparse.RoundSummary{
			{
				Winner: "red", Time: 300,
				Players: append([]demoparse.PlayerSummary{
					{
						Name: "Player", SteamID: string(playerID.Steam3()), Team: "red",
						TickStart: 1000, TickEnd: 5000,
					},
				}, fillerPlayers...),
			},
		},
	}

	matchID, errImport := statsCase.Import(ctx, server.ServerID, demoID, demo, time.Now())
	require.NoError(t, errImport)

	var matchBucket int32
	require.NoError(t, testFixture.Database.QueryRow(ctx,
		`SELECT stats_bucket_id FROM match WHERE match_id = $1`, *matchID).Scan(&matchBucket))
	require.Equal(t, defaultBucket.BucketID, matchBucket)
}

// TestMatchesWithPlayerOrdering ensures a player's match history is returned
// ordered by match start time, newest first.
func TestMatchesWithPlayerOrdering(t *testing.T) {
	t.Parallel()
	testFixture := tests.NewFixture()
	defer testFixture.Close()

	repo := stats.NewRepository(testFixture.Database)
	statsCase := stats.New(repo, maps.New(maps.NewRepository(testFixture.Database)))
	ctx := t.Context()

	server := testFixture.CreateTestServer(ctx)

	playerID := steamid.RandSID64()
	fillerIDs, fillerPlayers := fillerTestPlayers(5, 1000, 5000)

	persons := map[steamid.SteamID]string{playerID: "Player"}
	for _, fillerID := range fillerIDs {
		persons[fillerID] = "Filler"
	}
	insertTestPersons(ctx, t, testFixture.Database, persons)

	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	for idx, start := range []time.Time{base, base.Add(24 * time.Hour)} {
		title := fmt.Sprintf("player-order-%d.dem", idx)
		var demoID int32
		require.NoError(t, testFixture.Database.QueryRow(ctx,
			`INSERT INTO demo (server_id, title, map_name, created_on) VALUES ($1, $2, $3, now()) RETURNING demo_id`,
			server.ServerID, title, "cp_order").Scan(&demoID))

		demo := &demoparse.Demo{
			Filename: title, DemoType: demoparse.HL2Demo,
			Server: "test", Map: "cp_order", Game: "tf",
			Duration: 600, Ticks: 40000, Frames: 39900, Signon: 1000,
			Rounds: []demoparse.RoundSummary{
				{
					Winner: "red", Time: 300,
					Players: append([]demoparse.PlayerSummary{
						{
							Name: "Player", SteamID: string(playerID.Steam3()), Team: "red",
							TickStart: 1000, TickEnd: 5000,
						},
					}, fillerPlayers...),
				},
			},
		}

		_, errImport := statsCase.Import(ctx, server.ServerID, demoID, demo, start)
		require.NoError(t, errImport)
	}

	matches, errMatches := statsCase.MatchesWithPlayer(ctx, playerID)
	require.NoError(t, errMatches)
	require.Len(t, matches, 2)
	require.True(t, matches[0].StartTime.After(matches[1].StartTime),
		"expected newest match first, got %s then %s", matches[0].StartTime, matches[1].StartTime)
}

// insertTestBan creates a ban row directly. banType 2 is a steam ban, 1 is a
// mute; validUntil in the past means expired.
func insertTestBan(ctx context.Context, t *testing.T, db database.Database, target, source steamid.SteamID, banType int, validUntil time.Time) {
	t.Helper()
	require.NoError(t, db.Exec(ctx,
		`INSERT INTO ban (target_id, source_id, ban_type, reason, reason_text, note, valid_until,
		                 created_on, updated_on, origin, appeal_state, evade_ok)
		 VALUES ($1, $2, $3, 3, 'cheating', 'test ban', $4, $5, $5, 0, 0, false)`,
		target.Int64(), source.Int64(), banType, validUntil, time.Now()))
}

// TestBannedPlayersExcludedFromSummaryViews ensures players with an active
// steam ban are filtered out of the stats summary views (and get rank 0),
// while expired, unbanned (deleted) and muted players remain visible.
func TestBannedPlayersExcludedFromSummaryViews(t *testing.T) {
	t.Parallel()
	testFixture := tests.NewFixture()
	defer testFixture.Close()

	ctx := t.Context()
	server := testFixture.CreateTestServer(ctx)

	bannedID := steamid.New("[U:1:60000001]")
	expiredID := steamid.New("[U:1:60000002]")
	unbannedID := steamid.New("[U:1:60000003]")
	mutedID := steamid.New("[U:1:60000004]")
	require.True(t, bannedID.Valid())
	require.True(t, expiredID.Valid())
	require.True(t, unbannedID.Valid())
	require.True(t, mutedID.Valid())

	fillerIDs, fillerPlayers := fillerTestPlayers(2, 1000, 5000)
	for idx := range fillerPlayers {
		fillerPlayers[idx].Kills = stats.MinKills
	}

	persons := map[steamid.SteamID]string{
		bannedID:   "Banned",
		expiredID:  "Expired",
		unbannedID: "Unbanned",
		mutedID:    "Muted",
	}
	for i, id := range fillerIDs {
		persons[id] = fmt.Sprintf("Filler%d", i)
	}
	insertTestPersons(ctx, t, testFixture.Database, persons)

	demoID := seedTestDemo(t, testFixture, server.ServerID, "banned-filter.dem", "cp_process_final")

	players := make([]demoparse.PlayerSummary, 0, 4+len(fillerPlayers))
	players = append(players,
		demoparse.PlayerSummary{Name: "Banned", SteamID: string(bannedID.Steam3()), Team: "red", TickStart: 1000, TickEnd: 5000, Kills: 30},
		demoparse.PlayerSummary{Name: "Expired", SteamID: string(expiredID.Steam3()), Team: "red", TickStart: 1000, TickEnd: 5000, Kills: 5},
		demoparse.PlayerSummary{Name: "Unbanned", SteamID: string(unbannedID.Steam3()), Team: "blu", TickStart: 1000, TickEnd: 5000, Kills: 5},
		demoparse.PlayerSummary{Name: "Muted", SteamID: string(mutedID.Steam3()), Team: "blu", TickStart: 1000, TickEnd: 5000, Kills: 5},
	)
	players = append(players, fillerPlayers...)

	demo := &demoparse.Demo{
		Filename: "banned-filter.dem", DemoType: demoparse.HL2Demo,
		Server: "test", Map: "cp_process_final", Game: "tf",
		Duration: 600, Ticks: 40000, Frames: 39900, Signon: 1000,
		Rounds: []demoparse.RoundSummary{
			{Winner: "red", Time: 300, Players: players},
		},
	}

	repo := stats.NewRepository(testFixture.Database)
	statsCase := stats.New(repo, maps.New(maps.NewRepository(testFixture.Database)))

	start := time.Now()
	_, errImport := statsCase.Import(ctx, server.ServerID, demoID, demo, start)
	require.NoError(t, errImport)
	require.NoError(t, statsCase.RefreshViews(ctx))

	// Sanity: all six players are visible before any bans exist.
	require.Len(t, queryOverall(t, repo, 1, stats.TimeBucketAlltime, time.Time{}), 6)

	source := fillerIDs[0]
	insertTestBan(ctx, t, testFixture.Database, bannedID, source, 2, time.Now().Add(24*time.Hour))
	insertTestBan(ctx, t, testFixture.Database, expiredID, source, 2, time.Now().Add(-24*time.Hour))
	insertTestBan(ctx, t, testFixture.Database, unbannedID, source, 2, time.Now().Add(24*time.Hour))
	require.NoError(t, testFixture.Database.Exec(ctx,
		`UPDATE ban SET deleted = true WHERE target_id = $1`, unbannedID.Int64()))
	insertTestBan(ctx, t, testFixture.Database, mutedID, source, 1, time.Now().Add(24*time.Hour))

	require.NoError(t, statsCase.RefreshViews(ctx))

	rows := queryOverall(t, repo, 1, stats.TimeBucketAlltime, time.Time{})
	require.Len(t, rows, 5)
	require.NotContains(t, rows, bannedID)
	for _, sid := range []steamid.SteamID{expiredID, unbannedID, mutedID, fillerIDs[0], fillerIDs[1]} {
		require.Contains(t, rows, sid)
	}

	rank, errRank := repo.MyRank(ctx, stats.Opts{StatsBucketID: 1, TimeBucket: stats.TimeBucketAlltime}, bannedID)
	require.NoError(t, errRank)
	require.Zero(t, rank)

	cleanRank, errCleanRank := repo.MyRank(ctx, stats.Opts{StatsBucketID: 1, TimeBucket: stats.TimeBucketAlltime}, mutedID)
	require.NoError(t, errCleanRank)
	require.Positive(t, cleanRank)

	// Expiring the active ban makes the player reappear on the next refresh.
	require.NoError(t, testFixture.Database.Exec(ctx,
		`UPDATE ban SET valid_until = now() - interval '1 hour' WHERE target_id = $1`, bannedID.Int64()))
	require.NoError(t, statsCase.RefreshViews(ctx))

	rows = queryOverall(t, repo, 1, stats.TimeBucketAlltime, time.Time{})
	require.Len(t, rows, 6)
	require.Contains(t, rows, bannedID)
}
