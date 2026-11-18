package stats

import (
	"cmp"
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

// ratingKey identifies a player's rating within a stats bucket.
type ratingKey struct {
	StatsBucketID int32
	SteamID       steamid.SteamID
}

// ratingAccumulator carries the running state of a single player's rating
// during a chronological replay.
type ratingAccumulator struct {
	skill       Skill
	matches     int
	wins        int
	losses      int
	draws       int
	lastMatchID uuid.UUID
	lastMatchAt time.Time
}

// RecalculateRatings rebuilds every player rating from scratch by replaying
// all rated matches in chronological order. Because the replay is complete,
// late-imported demos and deleted matches are handled without checkpoints.
func (s Stats) RecalculateRatings(ctx context.Context) error {
	start := time.Now()

	rows, err := s.repo.ratedMatchOutcomes(ctx)
	if err != nil {
		return err
	}

	ratings, history := replayRatings(rows, s.calculator)

	if err := s.repo.ReplaceRatings(ctx, ratings, history); err != nil {
		return err
	}

	slog.Info("Recalculated player ratings",
		slog.Int("players", len(ratings)),
		slog.Int("history", len(history)),
		slog.Int("player_matches", len(rows)),
		slog.Duration("duration", time.Since(start)))

	return nil
}

// replayRatings runs the rating algorithm over the ordered outcome rows. The
// rows must be sorted by match time and match ID, with each match's players
// grouped together. It returns the final rating of every player and one
// history snapshot per player per rated match.
func replayRatings(rows []ratedOutcomeRow, calculator RatingCalculator) ([]PlayerRating, []RatingHistory) {
	states := make(map[ratingKey]*ratingAccumulator)
	history := make([]RatingHistory, 0, len(rows))
	calculatedOn := time.Now().UTC()
	initial := calculator.NewPlayerSkill()

	for start := 0; start < len(rows); {
		match := rows[start]

		end := start + 1
		for end < len(rows) && rows[end].MatchID == match.MatchID {
			end++
		}

		redIDs, bluIDs := matchTeams(rows[start:end])
		if len(redIDs) > 0 && len(bluIDs) > 0 {
			outcome := ratingOutcomeFor(match)
			newRed, newBlu := calculator.UpdateMatch(
				skillsFor(states, match.StatsBucketID, redIDs, initial),
				skillsFor(states, match.StatsBucketID, bluIDs, initial),
				outcome)

			redOutcome, bluOutcome := teamOutcomes(outcome)
			recordMatch(states, match, redIDs, newRed, redOutcome, initial, &history)
			recordMatch(states, match, bluIDs, newBlu, bluOutcome, initial, &history)
		}

		start = end
	}

	ratings := make([]PlayerRating, 0, len(states))
	for key, state := range states {
		ratings = append(ratings, PlayerRating{
			SteamID:       key.SteamID,
			StatsBucketID: key.StatsBucketID,
			Mu:            state.skill.Mu,
			Sigma:         state.skill.Sigma,
			Display:       calculator.Display(state.skill),
			MatchesPlayed: state.matches,
			Wins:          state.wins,
			Losses:        state.losses,
			Draws:         state.draws,
			LastMatchID:   state.lastMatchID,
			LastMatchAt:   state.lastMatchAt,
			CalculatedOn:  calculatedOn,
		})
	}

	slices.SortFunc(ratings, func(a, b PlayerRating) int {
		if a.StatsBucketID != b.StatsBucketID {
			return int(a.StatsBucketID - b.StatsBucketID)
		}

		return cmp.Compare(a.SteamID.Int64(), b.SteamID.Int64())
	})

	return ratings, history
}

// matchTeams splits one match's rows into the red and blue rosters. A player
// joins the team they played the most rounds for; the last round played breaks
// a tie.
func matchTeams(rows []ratedOutcomeRow) ([]steamid.SteamID, []steamid.SteamID) {
	var redIDs, bluIDs []steamid.SteamID

	for _, row := range rows {
		switch {
		case row.RedRounds > row.BluRounds:
			redIDs = append(redIDs, row.SteamID)
		case row.BluRounds > row.RedRounds:
			bluIDs = append(bluIDs, row.SteamID)
		case row.RedRounds == 0:
			continue
		case row.LastRedRound > row.LastBluRound:
			redIDs = append(redIDs, row.SteamID)
		default:
			bluIDs = append(bluIDs, row.SteamID)
		}
	}

	return redIDs, bluIDs
}

// ratingOutcomeFor compares the round wins of both teams.
func ratingOutcomeFor(match ratedOutcomeRow) RatingOutcome {
	switch {
	case match.ScoreRed > match.ScoreBlu:
		return RatingWin
	case match.ScoreBlu > match.ScoreRed:
		return RatingLoss
	default:
		return RatingDraw
	}
}

// teamOutcomes returns each team's outcome for a given match result.
func teamOutcomes(outcome RatingOutcome) (RatingOutcome, RatingOutcome) {
	switch outcome {
	case RatingWin:
		return RatingWin, RatingLoss
	case RatingLoss:
		return RatingLoss, RatingWin
	default:
		return RatingDraw, RatingDraw
	}
}

// skillsFor returns the current skill of each player, defaulting new players
// to the calculator's initial skill.
func skillsFor(states map[ratingKey]*ratingAccumulator, bucketID int32, ids []steamid.SteamID, initial Skill) []Skill {
	skills := make([]Skill, len(ids))
	for idx, steamID := range ids {
		key := ratingKey{StatsBucketID: bucketID, SteamID: steamID}
		if state, exists := states[key]; exists {
			skills[idx] = state.skill
		} else {
			skills[idx] = initial
		}
	}

	return skills
}

// recordMatch stores the post-match skill of each player and appends their
// history snapshot.
func recordMatch(
	states map[ratingKey]*ratingAccumulator,
	match ratedOutcomeRow,
	ids []steamid.SteamID,
	skills []Skill,
	outcome RatingOutcome,
	initial Skill,
	history *[]RatingHistory,
) {
	for idx, steamID := range ids {
		key := ratingKey{StatsBucketID: match.StatsBucketID, SteamID: steamID}

		state, exists := states[key]
		if !exists {
			state = &ratingAccumulator{skill: initial}
			states[key] = state
		}

		state.skill = skills[idx]
		state.matches++
		state.lastMatchID = match.MatchID
		state.lastMatchAt = match.StartTime

		switch outcome {
		case RatingWin:
			state.wins++
		case RatingLoss:
			state.losses++
		default:
			state.draws++
		}

		*history = append(*history, RatingHistory{
			SteamID:        steamID,
			StatsBucketID:  match.StatsBucketID,
			MatchID:        match.MatchID,
			MatchStartTime: match.StartTime,
			Mu:             state.skill.Mu,
			Sigma:          state.skill.Sigma,
			Outcome:        outcome,
			MatchesPlayed:  state.matches,
		})
	}
}
