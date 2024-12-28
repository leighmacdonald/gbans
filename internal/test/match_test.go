package test

import (
	"fmt"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"golang.org/x/exp/rand"
)

func genMatch(players int) domain.DemoDetails {
	s := domain.DemoState{
		DemoPlayerSummaries: make(map[int]domain.DemoPlayerSummaries),
		Users:               make(map[int]domain.DemoPlayer),
	}
	weaponIdx := 1
	for i := range players {
		team := logparse.BLU
		if i%2 == 0 {
			team = logparse.RED
		}

		s.Users[i+1] = domain.DemoPlayer{
			Classes: nil,
			Name:    stringutil.SecureRandomString(10),
			UserID:  i,
			SteamID: steamid.RandSID64(),
			Team:    team,
		}

		w := make(map[logparse.Weapon]domain.DemoWeaponDetail)

		for range int(rand.Int31n(5) + 1) {
			weaponIdx++
			w[logparse.Weapon(rune(weaponIdx))] = domain.DemoWeaponDetail{
				Kills:  int(rand.Int31n(200)),
				Hits:   int(rand.Int31n(200)),
				Damage: int(rand.Int31n(30000)),
				Shots:  int(rand.Int31n(500)),
			}
		}

		s.DemoPlayerSummaries[i+1] = domain.DemoPlayerSummaries{
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
			DamgageDealt:       int(rand.Int31n(50000)),
			WeaponMap:          w,
		}
	}

	d := domain.DemoDetails{
		State: s,
		Header: domain.DemoHeader{
			DemoType: domain.DemoType,
			Version:  3,
			Protocol: 24,
			Server:   fmt.Sprintf("Test server: %s", stringutil.SecureRandomString(5)),
			Nick:     "SourceTV Demo",
			Map:      "pl_test",
			Game:     "tf",
			Duration: float64(rand.Int31n(5000)),
			Ticks:    int(rand.Int31n(250000)),
			Frames:   int(rand.Int31n(25000)),
			Signon:   int(rand.Int31n(1000000)),
		},
	}

	return d
}
