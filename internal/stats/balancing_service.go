package stats

import (
	"context"
	"errors"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

const (
	// MaxBalancePlayers caps the size of a scramble request so a malformed
	// caller cannot make the exact solver enumerate an unbounded pool.
	MaxBalancePlayers = 64
	// DefaultMaxExact is the exact-solver cap used when a caller sends zero.
	DefaultMaxExact = 32
)

// ErrTooManyBalancePlayers is returned when a scramble request exceeds the
// supported pool size.
var ErrTooManyBalancePlayers = errors.New("too many players to balance")

// BalanceResult is the team assignment produced for a scramble request.
type BalanceResult struct {
	Red       []steamid.SteamID
	Blu       []steamid.SteamID
	Waiting   []steamid.SteamID
	Imbalance float64
	Algorithm string
	Balancer  string
}

// BalanceTeams assigns every submitted player to a team using the configured
// balancer and the internal skill rating (mu) from the default stats bucket.
// Players without a rating use the configured calculator's starting skill.
// Any player the balancer leaves waiting is reassigned to the smaller team so
// nobody in the submitted list is left out.
func (s Stats) BalanceTeams(ctx context.Context, ids []steamid.SteamID, maxExact int) (BalanceResult, error) {
	result := BalanceResult{
		Algorithm: s.calculator.Name(),
		Balancer:  s.balancer.Name(),
		Red:       make([]steamid.SteamID, 0, len(ids)/2),
		Blu:       make([]steamid.SteamID, 0, len(ids)/2),
	}

	players := uniqueValidSteamIDs(ids)
	if len(players) == 0 {
		return result, nil
	}

	if len(players) > MaxBalancePlayers {
		return result, ErrTooManyBalancePlayers
	}

	bucketID, errBucket := s.repo.DefaultBucketID(ctx)
	if errBucket != nil {
		return result, errBucket
	}

	ratings, errRatings := s.repo.RatingsForSteamIDs(ctx, bucketID, players)
	if errRatings != nil {
		return result, errRatings
	}

	initial := s.calculator.NewPlayerSkill()
	skills := make([]Skill, len(players))
	for idx, id := range players {
		if rating, exists := ratings[id]; exists {
			skills[idx] = Skill{Mu: rating.Mu, Sigma: rating.Sigma}
		} else {
			skills[idx] = initial
		}
	}

	if maxExact <= 0 {
		maxExact = DefaultMaxExact
	}

	split, errSplit := s.balancer.BestSplit(skills, maxExact)
	if errSplit != nil {
		return result, errSplit
	}

	for _, idx := range split.Red {
		if idx >= 0 && idx < len(players) {
			result.Red = append(result.Red, players[idx])
		}
	}

	for _, idx := range split.Blu {
		if idx >= 0 && idx < len(players) {
			result.Blu = append(result.Blu, players[idx])
		}
	}

	for _, idx := range split.Waiting {
		if idx < 0 || idx >= len(players) {
			continue
		}

		if len(result.Red) <= len(result.Blu) {
			result.Red = append(result.Red, players[idx])
		} else {
			result.Blu = append(result.Blu, players[idx])
		}
	}

	result.Imbalance = split.Imbalance

	return result, nil
}

func uniqueValidSteamIDs(ids []steamid.SteamID) []steamid.SteamID {
	seen := make(map[steamid.SteamID]bool, len(ids))
	result := make([]steamid.SteamID, 0, len(ids))
	for _, steamID := range ids {
		if !steamID.Valid() || seen[steamID] {
			continue
		}

		seen[steamID] = true
		result = append(result, steamID)
	}

	return result
}
