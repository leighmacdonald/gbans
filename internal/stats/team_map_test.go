//nolint:testpackage // White-box test for the unexported match team derivation.
package stats

import (
	"fmt"
	"testing"

	"github.com/leighmacdonald/gbans/pkg/demoparse"
	"github.com/stretchr/testify/require"
)

// TestPlayerTeamMapUsesConfirmedRoundRosters covers the fallback behaviour that
// caused losing rosters to balloon: players absent from the round-end winner
// list used to be assumed losers, even though the parser only lists players who
// were alive when the round ended.
func TestPlayerTeamMapUsesConfirmedRoundRosters(t *testing.T) {
	t.Parallel()

	steam3 := func(accountID int) string {
		return fmt.Sprintf("[U:1:%d]", accountID)
	}

	var (
		confirmedWinner = steam3(1)
		switched        = steam3(2)
		confirmedLoser  = steam3(3)
		spectator       = steam3(4)
		lateJoin        = steam3(5)
		redWinner       = steam3(6)
		blueLoser       = steam3(7)
		staleWinner     = steam3(8)
		ghost           = steam3(9)
	)

	players := func(ids ...string) []demoparse.PlayerSummary {
		out := make([]demoparse.PlayerSummary, 0, len(ids))
		for _, id := range ids {
			out = append(out, demoparse.PlayerSummary{SteamID: id})
		}

		return out
	}

	demo := demoparse.Demo{
		Rounds: []demoparse.RoundSummary{
			{
				Winner:  "blue",
				Winners: []string{confirmedWinner, switched, staleWinner},
				Losers:  []string{confirmedLoser},
				Players: players(confirmedWinner, switched, confirmedLoser, spectator, staleWinner),
			},
			{
				Winner:  "red",
				Winners: []string{redWinner, switched},
				Losers:  []string{blueLoser},
				Players: players(confirmedWinner, switched, confirmedLoser, spectator, lateJoin, redWinner, blueLoser, staleWinner),
			},
			{
				// Stub rounds have no winner and no rosters; they must be skipped.
				Players: players(confirmedWinner, switched, confirmedLoser, spectator, lateJoin, redWinner, blueLoser, staleWinner, ghost),
			},
			{
				Winner:  "unassigned",
				Winners: []string{ghost},
				Players: players(ghost),
			},
		},
	}

	teams := playerTeamMap(&demo)

	require.Equal(t, map[string]string{
		confirmedWinner: "blu",
		switched:        "red",
		confirmedLoser:  "red",
		redWinner:       "red",
		blueLoser:       "blu",
		staleWinner:     "blu",
	}, teams)
}
