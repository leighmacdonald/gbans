package test_test

import (
	"context"
	"testing"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/demoparse"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/rand"
)

func generateDemoDetails(players int) demoparse.Demo {
	demo := demoparse.Demo{
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
			Players: make([]demoparse.PlayerSummary, 0),
			Rounds:  make([]demoparse.RoundSummary, 0),
			Chat:    make([]demoparse.ChatMessage, 0),
	}
	weaponIdx := 1
	for playerIdx := range players {
		team := logparse.BLU
		if playerIdx%2 == 0 {
			team = logparse.RED
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

		append(demo.Players, demoparse.PlayerSummary{
			Team:               playerIdx%2 ? "blue" : "red",
			Points:             int(rand.Int31n(200)),
			Kills:              int(rand.Int31n(200)),
			Assists:            int(rand.Int31n(200)),
			Deaths:             int(rand.Int31n(200)),
			BuildingDestroyed: int(rand.Int31n(20)),
			Captures:           int(rand.Int31n(20)),
			CapturesBlocked:           int(rand.Int31n(20)),
			Dominations:        int(rand.Int31n(20)),
			Revenges:           int(rand.Int31n(20)),
			ChargesUber:        int(rand.Int31n(20)),
			Headshots:          int(rand.Int31n(20)),
// 			Teleports:          int(rand.Int31n(20)),
			Healing:            int(rand.Int31n(20)),
			Backstabs:          int(rand.Int31n(50000)),
			BonusPoints:        int(rand.Int31n(2000)),
			Damage:        int(rand.Int31n(50000)),
			DamageTaken:        int(rand.Int31n(200)),
			HealingTaken:       int(rand.Int31n(200)),
			HealthPacksCount:        int(rand.Int31n(200)),
			HealingFromPacks:       int(rand.Int31n(200)),
			Extinguishes:       int(rand.Int31n(200)),
			BuildingBuilt:      int(rand.Int31n(200)),
			Airshots:           int(rand.Int31n(200)),
			Shots:              int(rand.Int31n(200)),
			Hits:               int(rand.Int31n(200)),
// 			WeaponMap:          weaponSum,
		})
	}

	return demo
}

func TestMatchFromDemo(t *testing.T) {
	demoDetails := generateDemoDetails(24)

	match, errMatch := matchUC.CreateFromDemo(context.Background(), testServer.ServerID, demoDetails)
	require.NoError(t, errMatch)

	require.Equal(t, match.MapName, demoDetails.Header.Map)
}
