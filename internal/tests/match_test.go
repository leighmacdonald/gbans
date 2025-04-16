package tests_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/demoparse"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/rand"
)

func generateDemoDetails(players int) demoparse.Demo {
	demo := demoparse.Demo{
		Header: demoparse.Header{
			DemoType: domain.DemoType,
			Version:  3,
			Protocol: 24,
			Server:   "Test server: " + stringutil.SecureRandomString(5),
			Nick:     "SourceTV Demo",
			Map:      "pl_test",
			Game:     "tf",
			Duration: float64(rand.Int31n(5000)),
			Ticks:    int(rand.Int31n(250000)),
			Frames:   int(rand.Int31n(25000)),
			Signon:   int(rand.Int31n(1000000)),
		},
		State: demoparse.GameState{
			Users:   make(map[int]demoparse.Player),
			Players: make(map[int]demoparse.PlayerSummary),
			Results: demoparse.Results{},
			Rounds:  make([]demoparse.DemoRoundSummary, 0),
			Chat:    make([]demoparse.ChatMessage, 0),
		},
	}
	weaponIdx := 1
	for playerIdx := range players {
		team := logparse.BLU
		if playerIdx%2 == 0 {
			team = logparse.RED
		}

		demo.State.Users[playerIdx+1] = demoparse.Player{
			Classes: nil,
			Name:    stringutil.SecureRandomString(10),
			UserID:  playerIdx,
			SteamID: steamid.RandSID64(),
			Team:    team,
		}

		weaponSum := make(map[demoparse.WeaponID]demoparse.WeaponSummary)

		for range int(rand.Int31n(5) + 1) {
			weaponIdx++
			weaponSum[demoparse.WeaponID(rune(weaponIdx))] = demoparse.WeaponSummary{
				Kills:     int(rand.Int31n(200)),
				Damage:    int(rand.Int31n(30000)),
				Shots:     int(rand.Int31n(500)),
				Hits:      int(rand.Int31n(200)),
				Backstabs: int(rand.Int31n(20)),
				Headshots: int(rand.Int31n(20)),
				Airshots:  int(rand.Int31n(20)),
			}
		}

		demo.State.Players[playerIdx+1] = demoparse.PlayerSummary{
			Points:             int(rand.Int31n(200)),
			Kills:              int(rand.Int31n(200)),
			Assists:            int(rand.Int31n(200)),
			Deaths:             int(rand.Int31n(200)),
			BuildingsDestroyed: int(rand.Int31n(20)),
			Captures:           int(rand.Int31n(20)),
			Defenses:           int(rand.Int31n(20)),
			Dominations:        int(rand.Int31n(20)),
			Revenges:           int(rand.Int31n(20)),
			Ubercharges:        int(rand.Int31n(20)),
			Headshots:          int(rand.Int31n(20)),
			Teleports:          int(rand.Int31n(20)),
			Healing:            int(rand.Int31n(20)),
			Backstabs:          int(rand.Int31n(50000)),
			BonusPoints:        int(rand.Int31n(2000)),
			Support:            int(rand.Int31n(20000)),
			DamageDealt:        int(rand.Int31n(50000)),
			DamageTaken:        int(rand.Int31n(200)),
			HealingTaken:       int(rand.Int31n(200)),
			HealthPacks:        int(rand.Int31n(200)),
			HealingPacks:       int(rand.Int31n(200)),
			Extinguishes:       int(rand.Int31n(200)),
			BuildingBuilt:      int(rand.Int31n(200)),
			BuildingDestroyed:  int(rand.Int31n(200)),
			Airshots:           int(rand.Int31n(200)),
			Shots:              int(rand.Int31n(200)),
			Hits:               int(rand.Int31n(200)),
			WeaponMap:          weaponSum,
		}
	}

	return demo
}

func TestMatchFromDemo(t *testing.T) {
	demoDetails := generateDemoDetails(24)

	match, errMatch := matchUC.CreateFromDemo(t.Context(), testServer.ServerID, demoDetails)
	require.NoError(t, errMatch)

	require.Equal(t, match.MapName, demoDetails.Header.Map)

	require.NoError(t, matchUC.MatchSave(t.Context(), &match))
}
