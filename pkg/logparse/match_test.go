package logparse_test

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestMatch(t *testing.T) {
	testFilePath := golib.FindFile(path.Join("testdata", "log_3124689.log"), "gbans")
	if testFilePath == "" {
		t.Skipf("Cant find test file: log_3124689.log")

		return
	}

	body, errRead := os.ReadFile(testFilePath)
	require.NoError(t, errRead)

	var (
		parser   = logparse.New()
		newMatch = logparse.NewMatch(1, "test server")
		rows     = strings.Split(string(body), "\n")
	)

	for _, line := range rows {
		if line == "" {
			continue
		}

		result, errResult := parser.Parse(line)
		require.NoError(t, errResult)

		if err := newMatch.Apply(result); err != nil && !errors.Is(err, logparse.ErrIgnored) {
			t.Errorf("Failed to Apply: %v [%d] %v", err, result.EventType, line)
		}
	}

	require.Equal(t, 19, newMatch.PlayerCount())
	require.Equal(t, 2, newMatch.MedicCount())
	require.Equal(t, 43, newMatch.ChatCount())

	playerVar := newMatch.PlayerBySteamID(steamid.New(76561198164892406))
	require.Equal(t, 12, playerVar.KillCount())
	require.Equal(t, 3, playerVar.CaptureCount())
	require.Equal(t, 0.86, playerVar.KDRatio())
	require.Equal(t, 1.58, playerVar.KDARatio())
	require.Equal(t, 16, playerVar.HealthPacks())

	require.Equal(t, 10, playerVar.Assists)
	require.Equal(t, 14, playerVar.Deaths())
	require.Equal(t, 4796, playerVar.Damage())
	require.Equal(t, 277, playerVar.DamagePerMin())
	require.Equal(t, 260, playerVar.DamageTakenPerMin())

	playerTuna := newMatch.PlayerBySteamID(steamid.New(76561198809011070))
	require.Equal(t, 2, playerTuna.CaptureCount())
	require.Equal(t, 1, playerTuna.AirShots())
	require.Equal(t, 3709, playerTuna.DamageTaken())
	require.Equal(t, 3, newMatch.RoundCount())
	// require.Equal(t, 40.68, playerTuna.Accuracy(logparse.ProjectileRocket))
	require.Equal(t, 43.48, playerTuna.AccuracyOverall())

	playerDoctrine := newMatch.PlayerBySteamID(steamid.New(76561199050447792))
	require.Equal(t, 18, playerDoctrine.HeadShots())

	playerNomo := newMatch.PlayerBySteamID(steamid.New(76561198051884373))
	require.Equal(t, 9, playerNomo.BackStabs())
	require.Equal(t, []logparse.PlayerClass{logparse.Spy, logparse.Pyro}, playerNomo.Classes)

	playerAvgIQ := newMatch.PlayerBySteamID(steamid.New(76561198113244106))
	require.Equal(t, 17368, playerAvgIQ.HealingStats.Healing)
	require.Equal(t, 4, playerAvgIQ.HealingStats.ChargesTotal())
	require.Equal(t, 2, playerAvgIQ.HealingStats.DropsTotal())
	require.Equal(t, 2850, playerAvgIQ.TargetInfo[playerVar.SteamID].HealingTaken)
	require.Equal(t, 1005, playerAvgIQ.HealingStats.HealingPerMin())

	require.Equal(t, []int{3, 0}, []int{newMatch.TeamScores.Red, newMatch.TeamScores.Blu})

	require.Equal(t, 3, newMatch.RoundCount())
	round := newMatch.Rounds[1]
	require.Equal(t, float64(377), round.Length.Seconds())
	require.Equal(t, 2, round.Score.Red)
	require.Equal(t, 0, round.Score.Blu)
	require.Equal(t, float64(3), round.UbersRed)
	require.Equal(t, float64(2), round.UbersBlu)
	require.Equal(t, 14605, round.DamageRed)
	require.Equal(t, 13801, round.DamageBlu)
}
