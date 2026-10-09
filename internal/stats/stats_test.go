package stats_test

import (
	stdjson "encoding/json"
	"os"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/json"
	"github.com/leighmacdonald/gbans/internal/maps"
	"github.com/leighmacdonald/gbans/internal/stats"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/demoparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func TestImport(t *testing.T) {
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

	st := stats.New(stats.NewRepository(testFixture.Database), maps.New(maps.NewRepository(testFixture.Database)))
	matchID, importErr := st.Import(ctx, server.ServerID, demoID, &demo, time.Now())
	require.NoError(t, importErr)
	require.NotNil(t, matchID)
}

// TestImportNewParserStats ensures the stats added with the tf2_demostats
// v0.3.x parser survive the full import, match read, and summary view paths.
func TestImportNewParserStats(t *testing.T) {
	testFixture := tests.NewFixture()
	defer testFixture.Close()

	ctx := t.Context()
	server := testFixture.CreateTestServer(ctx)

	steamID := steamid.New("[U:1:12345678]")
	require.True(t, steamID.Valid())

	victimID := steamid.New("[U:1:87654321]")
	require.True(t, victimID.Valid())

	for _, person := range []struct {
		id   steamid.SteamID
		name string
	}{
		{steamID, "NewStats"},
		{victimID, "Victim"},
	} {
		require.NoError(t, testFixture.Database.Exec(ctx,
			`INSERT INTO person (steam_id, created_on, updated_on, personaname, avatarhash, profilestate, personastate,
			                    realname, timecreated, loccountrycode, locstatecode, loccityid)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			 ON CONFLICT DO NOTHING`,
			person.id.Int64(), time.Now(), time.Now(), person.name, "", 0, 0, "", 0, "", "", 0))
	}

	var demoID int32
	require.NoError(t, testFixture.Database.QueryRow(ctx,
		`INSERT INTO demo (server_id, title, map_name, created_on) VALUES ($1, $2, $3, $4) RETURNING demo_id`,
		server.ServerID, "new-stats.dem", "cp_process_final", time.Now()).Scan(&demoID))

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
				Players: []demoparse.PlayerSummary{player},
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
	require.Len(t, match.Players, 1)

	got := match.Players[0]
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

	// RefreshMaterializedView rejects all names (validViewNames is never
	// populated), so refresh directly here.
	require.NoError(t, testFixture.Database.Exec(ctx, "REFRESH MATERIALIZED VIEW stats_summary_daily_overall_view"))
	require.NoError(t, testFixture.Database.Exec(ctx, "REFRESH MATERIALIZED VIEW stats_summary_daily_variants_view"))

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
	testFixture := tests.NewFixture()
	defer testFixture.Close()

	ctx := t.Context()
	server := testFixture.CreateTestServer(ctx)

	killerID := steamid.New("[U:1:11111111]")
	victimID := steamid.New("[U:1:22222222]")
	require.True(t, killerID.Valid())
	require.True(t, victimID.Valid())

	for _, person := range []struct {
		id   steamid.SteamID
		name string
	}{
		{killerID, "Killer"},
		{victimID, "Victim"},
	} {
		require.NoError(t, testFixture.Database.Exec(ctx,
			`INSERT INTO person (steam_id, created_on, updated_on, personaname, avatarhash, profilestate, personastate,
			                    realname, timecreated, loccountrycode, locstatecode, loccityid)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			 ON CONFLICT DO NOTHING`,
			person.id.Int64(), time.Now(), time.Now(), person.name, "", 0, 0, "", 0, "", "", 0))
	}

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
				Players: []demoparse.PlayerSummary{
					{
						Name: "Killer", SteamID: string(killerID.Steam3()), Team: "red",
						TickStart: 1000, TickEnd: 1500,
					},
				},
			},
			{
				Winner: "blu", Time: 300,
				Players: []demoparse.PlayerSummary{
					{
						Name: "Killer", SteamID: string(killerID.Steam3()), Team: "blu",
						TickStart: 2000, TickEnd: 2500,
					},
				},
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

	for _, person := range []struct {
		id   steamid.SteamID
		name string
	}{
		{killerID, "Killer"},
		{victimID, "Victim"},
		{engineerID, "Engineer"},
	} {
		require.NoError(t, testFixture.Database.Exec(ctx,
			`INSERT INTO person (steam_id, created_on, updated_on, personaname, avatarhash, profilestate, personastate,
			                    realname, timecreated, loccountrycode, locstatecode, loccityid)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			 ON CONFLICT DO NOTHING`,
			person.id.Int64(), time.Now(), time.Now(), person.name, "", 0, 0, "", 0, "", "", 0))
	}

	var demoID int32
	require.NoError(t, testFixture.Database.QueryRow(ctx,
		`INSERT INTO demo (server_id, title, map_name, created_on) VALUES ($1, $2, $3, $4) RETURNING demo_id`,
		server.ServerID, "match-events.dem", "cp_process_final", time.Now()).Scan(&demoID))

	demo := demoparse.Demo{
		Filename: "match-events.dem", DemoType: demoparse.HL2Demo,
		Server: "test", Map: "cp_process_final", Game: "tf",
		Duration: 600, Ticks: 40000, Frames: 39900, Signon: 1000,
		Rounds: []demoparse.RoundSummary{
			{
				Winner: "red", Time: 300,
				Players: []demoparse.PlayerSummary{
					{
						Name: "Killer", SteamID: string(killerID.Steam3()), Team: "red",
						TickStart: 1000, TickEnd: 5000,
					},
					{
						Name: "Engineer", SteamID: string(engineerID.Steam3()), Team: "red",
						TickStart: 1000, TickEnd: 5000,
					},
				},
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

	rows, err := testFixture.Database.Query(ctx,
		`SELECT event_type, tick, actor_steam_id, target_steam_id, weapon, building,
			ST_X(event_position), ST_Y(event_position), ST_Z(event_position),
			details
		FROM match_event
		WHERE match_id = $1
		ORDER BY tick`,
		*matchID)
	require.NoError(t, err)
	defer rows.Close()

	type eventRow struct {
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

	var events []eventRow
	for rows.Next() {
		var event eventRow
		require.NoError(t, rows.Scan(
			&event.eventType, &event.tick, &event.actor, &event.target,
			&event.weapon, &event.building,
			&event.x, &event.y, &event.z,
			&event.details,
		))
		events = append(events, event)
	}
	require.NoError(t, rows.Err())
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
		IsMini   bool   `json:"is_mini"`
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
		CpName  string   `json:"cp_name"`
		Cappers []string `json:"cappers"`
		CapTime float64  `json:"cap_time"`
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
