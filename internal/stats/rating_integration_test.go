package stats_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/authn"
	"connectrpc.com/connect"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	personDomain "github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/maps"
	"github.com/leighmacdonald/gbans/internal/opengraph"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/stats"
	v1 "github.com/leighmacdonald/gbans/internal/stats/v1"
	"github.com/leighmacdonald/gbans/internal/stats/v1/statsv1connect"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

type ratingPermissionResolver struct {
	perms []rolesv1.Permission
}

func (r ratingPermissionResolver) PermissionsBySteamID(_ context.Context, _ steamid.SteamID) []rolesv1.Permission {
	return r.perms
}

// newRatingRPC serves the stats service behind the real auth middleware with a
// fixed permission set and returns a client factory.
func newRatingRPC(t *testing.T, fixture *tests.Fixture, perms []rolesv1.Permission) (*rpc.Middleware, func(opts ...connect.ClientOption) statsv1connect.StatsServiceClient) {
	t.Helper()

	resolver := ratingPermissionResolver{perms: perms}
	middleware := rpc.NewMiddleware("gbans-test", "test-cookie-secret", resolver)

	serversCase, errServers := servers.New(servers.NewRepository(fixture.Database), nil, "")
	require.NoError(t, errServers)

	svc := stats.NewService(
		stats.New(stats.NewRepository(fixture.Database), maps.New(maps.NewRepository(fixture.Database))),
		serversCase,
		opengraph.NewRepository(opengraph.Options{}),
		rpc.NewRoleAuth(resolver),
		middleware,
		rpc.CreateInterceptors(),
	)

	mux := http.NewServeMux()
	mux.Handle(svc.Pattern, svc.Handler)

	server := httptest.NewServer(authn.NewMiddleware(middleware.Authenticate).Wrap(mux))
	t.Cleanup(server.Close)

	return middleware, func(opts ...connect.ClientOption) statsv1connect.StatsServiceClient {
		return statsv1connect.NewStatsServiceClient(server.Client(), server.URL, opts...)
	}
}

func ratingAuthOptions(t *testing.T, middleware *rpc.Middleware, steamID steamid.SteamID) []connect.ClientOption {
	t.Helper()

	token, fingerprint, errToken := middleware.MakeUserToken(personDomain.Core{SteamID: steamID})
	require.NoError(t, errToken)

	return []connect.ClientOption{connect.WithInterceptors(connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+token)
			req.Header().Set("Cookie", "fingerprint="+fingerprint)

			return next(ctx, req)
		}
	}))}
}

// seedRatingMatch inserts a match with one round and one row per player,
// assigning each player to the given team.
func seedRatingMatch(
	t *testing.T,
	fixture *tests.Fixture,
	serverID, mapID, demoID int32,
	start time.Time,
	scoreRed, scoreBlu int32,
	red, blu []steamid.SteamID,
) uuid.UUID {
	t.Helper()
	ctx := t.Context()

	matchID, errID := uuid.NewV4()
	require.NoError(t, errID)

	require.NoError(t, fixture.Database.Exec(ctx, `
		INSERT INTO match (
			match_id, server_id, map_id, demo_id, stats_bucket_id, hostname,
			score_red, score_blu, start_time, duration_ms, created_on
		) VALUES ($1, $2, $3, $4, 1, 'rating-test', $5, $6, $7, 1800000, now())`,
		matchID, serverID, mapID, demoID, scoreRed, scoreBlu, start))

	var roundID int64
	//nolint:dupword // the SQL repeats the literal false.
	require.NoError(t, fixture.Database.QueryRow(ctx, `
		INSERT INTO match_round (match_id, winner, is_stalemate, is_sudden_death, duration_ms)
		VALUES ($1, 'red', false, false, 1800000) RETURNING round_id`, matchID).Scan(&roundID))

	teams := map[string][]steamid.SteamID{"red": red, "blu": blu}
	for team, players := range teams {
		for _, player := range players {
			insertRatingRoundPlayer(t, fixture, roundID, player, team)
		}
	}

	return matchID
}

func insertRatingRoundPlayer(t *testing.T, fixture *tests.Fixture, roundID int64, player steamid.SteamID, team string) {
	t.Helper()

	require.NoError(t, fixture.Database.Exec(t.Context(), `
		INSERT INTO match_round_player (
			round_id, steam_id, team, tick_start, tick_end,
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
			$1, $2, $3, 1000, 2000,
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
		)`, roundID, player.Int64(), team))
}

// ratingSeed carries the seeded identifiers shared by the rating subtests.
type ratingSeed struct {
	winnerA      steamid.SteamID
	winnerB      steamid.SteamID
	loserC       steamid.SteamID
	loserD       steamid.SteamID
	provisionalE steamid.SteamID
	provisionalF steamid.SteamID
	unassigned   steamid.SteamID
}

// seedRatingMatches creates five wins, one draw, a four-match provisional
// player and the two match shapes that must never be rated.
func seedRatingMatches(t *testing.T, fixture *tests.Fixture) ratingSeed {
	t.Helper()
	ctx := t.Context()

	server := fixture.CreateTestServer(ctx)
	mapID := seedTestMap(t, fixture, "cp_ratings")
	demoID := seedTestDemo(t, fixture, server.ServerID, "rating-demo", "cp_ratings")

	seed := ratingSeed{
		winnerA:      steamid.New("[U:1:7300000001]"),
		winnerB:      steamid.New("[U:1:7300000002]"),
		loserC:       steamid.New("[U:1:7300000003]"),
		loserD:       steamid.New("[U:1:7300000004]"),
		provisionalE: steamid.New("[U:1:7300000005]"),
		provisionalF: steamid.New("[U:1:7300000006]"),
		unassigned:   steamid.New("[U:1:7300000007]"),
	}

	insertTestPersons(ctx, t, fixture.Database, map[steamid.SteamID]string{
		seed.winnerA:      "RatingA",
		seed.winnerB:      "RatingB",
		seed.loserC:       "RatingC",
		seed.loserD:       "RatingD",
		seed.provisionalE: "RatingE",
		seed.provisionalF: "RatingF",
		seed.unassigned:   "RatingG",
	})

	start := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	red := []steamid.SteamID{seed.winnerA, seed.winnerB}
	blu := []steamid.SteamID{seed.loserC, seed.loserD}

	for match := range 5 {
		seedRatingMatch(t, fixture, server.ServerID, mapID, demoID,
			start.Add(time.Duration(match)*time.Hour), 2, 1, red, blu)
	}

	seedRatingMatch(t, fixture, server.ServerID, mapID, demoID,
		start.Add(5*time.Hour), 2, 2, red, blu)

	for match := range 4 {
		seedRatingMatch(t, fixture, server.ServerID, mapID, demoID,
			start.Add(time.Duration(match)*time.Hour), 2, 1,
			[]steamid.SteamID{seed.provisionalE}, []steamid.SteamID{seed.provisionalF})
	}

	// Excluded: no round wins, so the match result is unknown.
	seedRatingMatch(t, fixture, server.ServerID, mapID, demoID,
		start.Add(6*time.Hour), 0, 0, red, blu)

	// Excluded: no player was ever assigned to a team.
	unassignedMatchID := seedRatingMatch(t, fixture, server.ServerID, mapID, demoID,
		start.Add(7*time.Hour), 2, 1, nil, nil)
	require.NoError(t, fixture.Database.Exec(ctx, `
		INSERT INTO match_round_player (round_id, steam_id, team, tick_start, tick_end,
			kills, assists, deaths, postround_kills, postround_assists, postround_deaths,
			preround_healing, healing, postround_healing, drops, near_full_charge_death,
			charges_uber, charges_kritz, charges_vacc, charges_quickfix,
			damage, damage_taken, dominations, dominated, revenges, revenged,
			airshots, headshots, headshot_kills, backstabs, backstab_kills,
			captures, captures_blocked, was_headshot, was_backstabbed,
			shots, hits, objects_built, objects_destroyed,
			points, connection_count, bonus_points,
			scoreboard_kills, scoreboard_assists, scoreboard_healing, scoreboard_deaths, scoreboard_damage,
			suicides, extinguishes, ignites)
		SELECT round_id, $1, 'unassigned', 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0
		FROM match_round WHERE match_id = $2`,
		seed.unassigned.Int64(), unassignedMatchID))

	return seed
}

// TestPlayerRatings exercises the daily rating rebuild end to end: matches are
// replayed, ratings and history are persisted, and the materialized view
// exposes display ratings with provisional handling and ban filtering.
func TestPlayerRatings(t *testing.T) { //nolint:tparallel // subtests share one seeded fixture.
	t.Parallel()

	testFixture := tests.NewFixture()
	defer testFixture.Close()

	ctx := t.Context()
	seed := seedRatingMatches(t, testFixture)

	repository := stats.NewRepository(testFixture.Database)
	service := stats.New(repository, maps.New(maps.NewRepository(testFixture.Database)))

	require.NoError(t, service.RefreshViews(ctx))

	t.Run("ranked player", func(t *testing.T) {
		rating, errRating := repository.PlayerRating(ctx, seed.winnerA, 1)
		require.NoError(t, errRating)

		require.Equal(t, seed.winnerA, rating.SteamID)
		require.Equal(t, int32(1), rating.StatsBucketID)
		require.Equal(t, "RatingA", rating.Personaname)
		require.Equal(t, 6, rating.MatchesPlayed)
		require.Equal(t, 5, rating.Wins)
		require.Equal(t, 0, rating.Losses)
		require.Equal(t, 1, rating.Draws)
		require.False(t, rating.Provisional)
		require.Positive(t, rating.Rank)
		require.Greater(t, rating.Rating, stats.DefaultRating)
		require.Positive(t, rating.RatingSigma)
		require.Less(t, rating.RatingConservative, rating.Rating)
		require.Equal(t, rating.Rating, stats.RatingDisplay(rating.Mu))
		require.Equal(t, rating.RatingSigma, stats.RatingDisplaySigma(rating.Sigma))
		require.False(t, rating.LastMatchAt.IsZero())
	})

	t.Run("provisional player", func(t *testing.T) {
		rating, errRating := repository.PlayerRating(ctx, seed.provisionalE, 1)
		require.NoError(t, errRating)

		require.Equal(t, 4, rating.MatchesPlayed)
		require.True(t, rating.Provisional)
		require.Zero(t, rating.Rank)
		require.Equal(t, stats.DefaultRating, rating.Rating)
		require.Zero(t, rating.RatingSigma)
		require.Zero(t, rating.RatingConservative)
		// The underlying skill still moves so the player is not stuck at the
		// default once they qualify.
		require.Greater(t, rating.Mu, stats.RatingMu)
	})

	t.Run("excluded players", func(t *testing.T) {
		_, errUnassigned := repository.PlayerRating(ctx, seed.unassigned, 1)
		require.ErrorIs(t, errUnassigned, database.ErrNoResult)
	})

	t.Run("leaderboard", func(t *testing.T) {
		entries, total, errLeague := repository.RatingLeaderboard(ctx, stats.RatingLeaderboardOpts{
			StatsBucketID: 1,
			Limit:         10,
		})
		require.NoError(t, errLeague)
		require.EqualValues(t, 4, total)
		require.Len(t, entries, 4)

		for _, entry := range entries {
			require.False(t, entry.Provisional)
			require.Positive(t, entry.Rank)
		}

		require.NotEqual(t, seed.provisionalE, entries[0].SteamID)
		require.Greater(t, entries[0].Mu, entries[len(entries)-1].Mu)

		paged, pagedTotal, errPage := repository.RatingLeaderboard(ctx, stats.RatingLeaderboardOpts{
			StatsBucketID: 1,
			Limit:         2,
			Offset:        2,
		})
		require.NoError(t, errPage)
		require.EqualValues(t, 4, pagedTotal)
		require.Len(t, paged, 2)
		require.Equal(t, entries[2].SteamID, paged[0].SteamID)
	})

	t.Run("history", func(t *testing.T) {
		entries, errHistory := repository.PlayerRatingHistory(ctx, seed.winnerA, 1, 100)
		require.NoError(t, errHistory)
		require.Len(t, entries, 6)

		// Newest first: the drawn match, then the five wins.
		require.Equal(t, stats.RatingDraw, entries[0].Outcome)
		require.Equal(t, 6, entries[0].MatchesPlayed)
		require.Equal(t, stats.RatingWin, entries[1].Outcome)
		require.Equal(t, 5, entries[1].MatchesPlayed)

		for idx := 1; idx < len(entries); idx++ {
			require.False(t, entries[idx].MatchStartTime.After(entries[idx-1].MatchStartTime))
		}
	})

	t.Run("rebuild is idempotent", func(t *testing.T) {
		before, errBefore := repository.PlayerRating(ctx, seed.winnerA, 1)
		require.NoError(t, errBefore)

		require.NoError(t, service.RecalculateRatings(ctx))
		require.NoError(t, testFixture.Database.RefreshMaterializedView(ctx, "stats_player_rating_view"))

		after, errAfter := repository.PlayerRating(ctx, seed.winnerA, 1)
		require.NoError(t, errAfter)
		require.InDelta(t, before.Mu, after.Mu, 1e-12)
		require.InDelta(t, before.Sigma, after.Sigma, 1e-12)
		require.Equal(t, before.MatchesPlayed, after.MatchesPlayed)
		require.Equal(t, before.Rank, after.Rank)

		entries, errHistory := repository.PlayerRatingHistory(ctx, seed.winnerA, 1, 100)
		require.NoError(t, errHistory)
		require.Len(t, entries, 6)
	})

	t.Run("api", func(t *testing.T) {
		requester := steamid.New("[U:1:7300000088]")
		request := &v1.PlayerRatingRequest{SteamId: new(seed.winnerA.Int64())}

		_, anonymousClient := newRatingRPC(t, testFixture, nil)
		_, errAnonymous := anonymousClient().PlayerRating(ctx, request)
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(errAnonymous))

		// The auth middleware intentionally reports a missing permission the
		// same way as a missing token.
		deniedMiddleware, deniedClient := newRatingRPC(t, testFixture, nil)
		_, errDenied := deniedClient(ratingAuthOptions(t, deniedMiddleware, requester)...).PlayerRating(ctx, request)
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(errDenied))

		middleware, clientFor := newRatingRPC(t, testFixture, []rolesv1.Permission{rolesv1.Permission_PERMISSION_STATS_READ})
		client := clientFor(ratingAuthOptions(t, middleware, requester)...)

		// A zero bucket resolves to the default bucket.
		ratingResp, errRating := client.PlayerRating(ctx, request)
		require.NoError(t, errRating)
		require.Equal(t, int32(1), ratingResp.GetRating().GetStatsBucketId())
		require.EqualValues(t, 6, ratingResp.GetRating().GetMatchesPlayed())
		require.False(t, ratingResp.GetRating().GetProvisional())
		require.Equal(t, "RatingA", ratingResp.GetRating().GetPlayer().GetName())

		_, errInvalid := client.PlayerRating(ctx, &v1.PlayerRatingRequest{SteamId: new(int64(0))})
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(errInvalid))

		_, errMissing := client.PlayerRating(ctx, &v1.PlayerRatingRequest{
			SteamId: new(seed.unassigned.Int64()),
		})
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(errMissing))

		provisionalResp, errProvisional := client.PlayerRating(ctx, &v1.PlayerRatingRequest{
			SteamId: new(seed.provisionalE.Int64()),
		})
		require.NoError(t, errProvisional)
		require.True(t, provisionalResp.GetRating().GetProvisional())
		require.EqualValues(t, stats.DefaultRating, provisionalResp.GetRating().GetRating())
		require.Zero(t, provisionalResp.GetRating().GetRank())

		historyResp, errHistory := client.PlayerRatingHistory(ctx, &v1.PlayerRatingHistoryRequest{
			SteamId: new(seed.winnerA.Int64()),
		})
		require.NoError(t, errHistory)
		require.Len(t, historyResp.GetEntries(), 6)
		require.Equal(t, v1.RatingOutcome_RATING_OUTCOME_DRAW_UNSPECIFIED, historyResp.GetEntries()[0].GetOutcome())
		require.Equal(t, v1.RatingOutcome_RATING_OUTCOME_WIN, historyResp.GetEntries()[1].GetOutcome())
		require.EqualValues(t, 6, historyResp.GetEntries()[0].GetMatchesPlayed())

		boardResp, errBoard := client.RatingLeaderboard(ctx, &v1.RatingLeaderboardRequest{
			Limit: new(uint32(2)),
		})
		require.NoError(t, errBoard)
		require.EqualValues(t, 4, boardResp.GetCount())
		require.Len(t, boardResp.GetRatings(), 2)
		require.False(t, boardResp.GetRatings()[0].GetProvisional())
	})

	t.Run("banned players are hidden", func(t *testing.T) {
		insertTestBan(ctx, t, testFixture.Database, seed.winnerA, steamid.New("[U:1:7300000099]"), 2, time.Now().Add(time.Hour))
		require.NoError(t, testFixture.Database.RefreshMaterializedView(ctx, "stats_player_rating_view"))

		_, errHidden := repository.PlayerRating(ctx, seed.winnerA, 1)
		require.ErrorIs(t, errHidden, database.ErrNoResult)

		// The rating itself is kept; only the public view filters.
		count, errCount := repository.RatingCount(ctx)
		require.NoError(t, errCount)
		require.EqualValues(t, 6, count)
	})
}
