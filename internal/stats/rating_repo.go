package stats

import (
	"context"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

// ratedOutcomeRow is one player's participation in a rated match, aggregated
// across the match's rounds.
type ratedOutcomeRow struct {
	MatchID       uuid.UUID
	StatsBucketID int32
	StartTime     time.Time
	ScoreRed      int
	ScoreBlu      int
	SteamID       steamid.SteamID
	RedRounds     int
	BluRounds     int
	LastRedRound  int
	LastBluRound  int
	Rounds        int
}

// PlayerRating is a player's persisted rating within a stats bucket.
type PlayerRating struct {
	SteamID       steamid.SteamID
	StatsBucketID int32
	Mu            float64
	Sigma         float64
	Display       DisplayRating
	MatchesPlayed int
	RoundsPlayed  int
	Wins          int
	Losses        int
	Draws         int
	LastMatchID   uuid.UUID
	LastMatchAt   time.Time
	CalculatedOn  time.Time
}

// RatingHistory is a post-match rating snapshot for a single player.
type RatingHistory struct {
	SteamID        steamid.SteamID
	StatsBucketID  int32
	MatchID        uuid.UUID
	MatchStartTime time.Time
	Mu             float64
	Sigma          float64
	Expected       float64
	Outcome        RatingOutcome
	MatchesPlayed  int
	Rounds         int
}

// Rating is the public, display-scaled rating of a player.
type Rating struct {
	SteamID            steamid.SteamID
	StatsBucketID      int32
	Personaname        string
	AvatarHash         string
	Rating             int
	RatingSigma        int
	RatingConservative int
	Rank               int
	Mu                 float64
	Sigma              float64
	MatchesPlayed      int
	Wins               int
	Losses             int
	Draws              int
	Provisional        bool
	LastMatchAt        time.Time
	CalculatedOn       time.Time
}

// RatingHistoryEntry is a single point on a player's rating graph.
type RatingHistoryEntry struct {
	MatchID        uuid.UUID
	MatchStartTime time.Time
	Mu             float64
	Sigma          float64
	Expected       float64
	Outcome        RatingOutcome
	MatchesPlayed  int
	Rounds         int
}

// RatingLeaderboardOpts controls the ranked player listing.
type RatingLeaderboardOpts struct {
	StatsBucketID int32
	Limit         int
	Offset        int
}

// ratedMatchOutcomes loads one row per player per rated match, ordered
// chronologically. Matches without any round wins are skipped, as are players
// who never played for a team.
func (r Repository) ratedMatchOutcomes(ctx context.Context) ([]ratedOutcomeRow, error) {
	const query = `
		SELECT
			m.match_id,
			m.stats_bucket_id,
			m.start_time,
			m.score_red,
			m.score_blu,
			rp.steam_id,
			count(*) FILTER (WHERE rp.team::text = 'red') AS red_rounds,
			count(*) FILTER (WHERE rp.team::text = 'blu') AS blu_rounds,
			coalesce(max(rp.round_id) FILTER (WHERE rp.team::text = 'red'), 0) AS last_red_round,
			coalesce(max(rp.round_id) FILTER (WHERE rp.team::text = 'blu'), 0) AS last_blu_round,
			count(*) AS rounds
		FROM match m
		JOIN match_round mr ON mr.match_id = m.match_id
		JOIN match_round_player rp ON rp.round_id = mr.round_id
		WHERE (m.score_red > 0 OR m.score_blu > 0)
			AND rp.team::text IN ('red', 'blu')
		GROUP BY m.match_id, m.stats_bucket_id, m.start_time, m.score_red, m.score_blu, rp.steam_id
		-- Spectators are recorded in every round's player list with no stats.
		-- They must not receive a rating for a match they did not play in.
		HAVING sum(rp.kills + rp.deaths + rp.assists + rp.points + rp.damage + rp.healing + rp.shots) > 0
		ORDER BY m.start_time, m.match_id, rp.steam_id`

	rows, err := r.Database.Query(ctx, query)
	if err != nil {
		return nil, database.Err(err)
	}
	defer rows.Close()

	outcomes := make([]ratedOutcomeRow, 0, 4096)
	for rows.Next() {
		var outcome ratedOutcomeRow
		if errScan := rows.Scan(
			&outcome.MatchID,
			&outcome.StatsBucketID,
			&outcome.StartTime,
			&outcome.ScoreRed,
			&outcome.ScoreBlu,
			&outcome.SteamID,
			&outcome.RedRounds,
			&outcome.BluRounds,
			&outcome.LastRedRound,
			&outcome.LastBluRound,
			&outcome.Rounds,
		); errScan != nil {
			return nil, database.Err(errScan)
		}

		outcomes = append(outcomes, outcome)
	}

	return outcomes, database.Err(rows.Err())
}

// ReplaceRatings swaps in a fully recomputed rating set. Both tables are
// truncated and repopulated in a single transaction so readers never observe a
// partially rebuilt rating.
func (r Repository) ReplaceRatings(ctx context.Context, ratings []PlayerRating, history []RatingHistory) error {
	return r.WrapTx(ctx, func(tx pgx.Tx) error { //nolint:varnamelen // conventional pgx name.
		if _, err := tx.Exec(ctx, `TRUNCATE TABLE player_rating, player_rating_history`); err != nil {
			return database.Err(err)
		}

		if len(ratings) > 0 {
			ratingRows := make([][]any, 0, len(ratings))
			for _, rating := range ratings {
				ratingRows = append(ratingRows, []any{
					rating.SteamID.Int64(),
					rating.StatsBucketID,
					rating.Mu,
					rating.Sigma,
					rating.Display.Rating,
					rating.Display.Sigma,
					rating.Display.Conservative,
					rating.RoundsPlayed,
					rating.Display.Visible,
					rating.Display.Stale,
					rating.MatchesPlayed,
					rating.Wins,
					rating.Losses,
					rating.Draws,
					rating.LastMatchID,
					rating.LastMatchAt,
					rating.CalculatedOn,
				})
			}

			if _, err := tx.CopyFrom(ctx,
				pgx.Identifier{"player_rating"},
				[]string{
					"steam_id", "stats_bucket_id", "mu", "sigma",
					"rating", "rating_sigma", "rating_conservative",
					"rounds_played", "visible", "stale", "matches_played",
					"wins", "losses", "draws", "last_match_id", "last_match_at", "calculated_on",
				},
				pgx.CopyFromRows(ratingRows),
			); err != nil {
				return database.Err(err)
			}
		}

		if len(history) > 0 {
			historyRows := make([][]any, 0, len(history))
			for _, entry := range history {
				historyRows = append(historyRows, []any{
					entry.SteamID.Int64(),
					entry.StatsBucketID,
					entry.MatchID,
					entry.MatchStartTime,
					entry.Mu,
					entry.Sigma,
					entry.Expected,
					int16(entry.Outcome),
					entry.MatchesPlayed,
					entry.Rounds,
				})
			}

			if _, err := tx.CopyFrom(ctx,
				pgx.Identifier{"player_rating_history"},
				[]string{
					"steam_id", "stats_bucket_id", "match_id", "match_start_time",
					"mu", "sigma", "expected", "outcome", "matches_played", "rounds",
				},
				pgx.CopyFromRows(historyRows),
			); err != nil {
				return database.Err(err)
			}
		}

		return nil
	})
}

// RatingCount returns the number of players with a persisted rating.
func (r Repository) RatingCount(ctx context.Context) (int64, error) {
	var count int64
	if err := r.QueryRow(ctx, `SELECT count(*) FROM player_rating`).Scan(&count); err != nil {
		return 0, database.Err(err)
	}

	return count, nil
}

const ratingViewColumns = `stats_bucket_id, steam_id, personaname, avatarhash,
	matches_played, wins, losses, draws, mu, sigma,
	rating, coalesce(rating_sigma, 0), coalesce(rating_conservative, 0), rank, provisional,
	coalesce(last_match_at, to_timestamp(0)), calculated_on`

// PlayerRating returns a single player's current rating in a bucket.
func (r Repository) PlayerRating(ctx context.Context, steamID steamid.SteamID, statsBucketID int32) (Rating, error) {
	var rating Rating

	row, errRow := r.QueryRowBuilder(ctx, r.Builder().
		Select(ratingViewColumns).
		From("stats_player_rating_view").
		Where("stats_bucket_id = ? AND steam_id = ?", statsBucketID, steamID.Int64()))
	if errRow != nil {
		return rating, database.Err(errRow)
	}

	if err := row.Scan(
		&rating.StatsBucketID, &rating.SteamID, &rating.Personaname, &rating.AvatarHash,
		&rating.MatchesPlayed, &rating.Wins, &rating.Losses, &rating.Draws, &rating.Mu, &rating.Sigma,
		&rating.Rating, &rating.RatingSigma, &rating.RatingConservative, &rating.Rank, &rating.Provisional,
		&rating.LastMatchAt, &rating.CalculatedOn,
	); err != nil {
		return rating, database.Err(err)
	}

	return rating, nil
}

// PlayerRatingHistory returns a player's rating snapshots, newest first.
func (r Repository) PlayerRatingHistory(ctx context.Context, steamID steamid.SteamID, statsBucketID int32, limit int) ([]RatingHistoryEntry, error) {
	rows, err := r.Database.Query(ctx, `
		SELECT match_id, match_start_time, mu, sigma, expected, outcome, matches_played, rounds
		FROM player_rating_history
		WHERE steam_id = $1 AND stats_bucket_id = $2
		ORDER BY match_start_time DESC, match_id DESC
		LIMIT $3`, steamID.Int64(), statsBucketID, limit)
	if err != nil {
		return nil, database.Err(err)
	}
	defer rows.Close()

	entries := make([]RatingHistoryEntry, 0, min(limit, 128))
	for rows.Next() {
		var entry RatingHistoryEntry
		if errScan := rows.Scan(
			&entry.MatchID, &entry.MatchStartTime, &entry.Mu, &entry.Sigma,
			&entry.Expected, &entry.Outcome, &entry.MatchesPlayed, &entry.Rounds,
		); errScan != nil {
			return nil, database.Err(errScan)
		}

		entries = append(entries, entry)
	}

	return entries, database.Err(rows.Err())
}

// RatingLeaderboard returns ranked players in a bucket, strongest first.
// Provisional players are excluded because they have no rank yet.
func (r Repository) RatingLeaderboard(ctx context.Context, opts RatingLeaderboardOpts) ([]Rating, uint64, error) {
	row, errRow := r.QueryRowBuilder(ctx, r.Builder().
		Select("count(*)").
		From("stats_player_rating_view").
		Where("stats_bucket_id = ? AND provisional = false", opts.StatsBucketID))
	if errRow != nil {
		return nil, 0, database.Err(errRow)
	}

	var total uint64
	if err := row.Scan(&total); err != nil {
		return nil, 0, database.Err(err)
	}

	rows, err := r.Database.Query(ctx, `
		SELECT `+ratingViewColumns+`
		FROM stats_player_rating_view
		WHERE stats_bucket_id = $1 AND provisional = false
		ORDER BY mu DESC, steam_id
		LIMIT $2 OFFSET $3`,
		opts.StatsBucketID, opts.Limit, opts.Offset)
	if err != nil {
		return nil, 0, database.Err(err)
	}
	defer rows.Close()

	entries := make([]Rating, 0, min(opts.Limit, 128))
	for rows.Next() {
		var rating Rating

		if errScan := rows.Scan(
			&rating.StatsBucketID, &rating.SteamID, &rating.Personaname, &rating.AvatarHash,
			&rating.MatchesPlayed, &rating.Wins, &rating.Losses, &rating.Draws, &rating.Mu, &rating.Sigma,
			&rating.Rating, &rating.RatingSigma, &rating.RatingConservative, &rating.Rank, &rating.Provisional,
			&rating.LastMatchAt, &rating.CalculatedOn,
		); errScan != nil {
			return nil, 0, database.Err(errScan)
		}

		entries = append(entries, rating)
	}

	return entries, total, database.Err(rows.Err())
}

// DefaultBucketID returns the stats bucket marked as the default.
func (r Repository) DefaultBucketID(ctx context.Context) (int32, error) {
	var bucketID int32
	if err := r.QueryRow(ctx, `SELECT stats_bucket_id FROM stats_bucket WHERE is_default LIMIT 1`).Scan(&bucketID); err != nil {
		return 0, database.Err(err)
	}

	return bucketID, nil
}
