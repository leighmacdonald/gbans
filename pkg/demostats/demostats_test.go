package demostats_test

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/leighmacdonald/gbans/pkg/demostats"
	"github.com/leighmacdonald/gbans/pkg/fs"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func TestSubmit(t *testing.T) {
	parserURL, found := os.LookupEnv("GBANS_DEMO_PARSER_URL")
	if !found || parserURL == "" {
		t.Skip("Parser url undefined")
	}

	demoPath := fs.FindFile(path.Join("testdata", "test.dem"), "gbans")
	detail, err := demostats.Submit(context.Background(), parserURL, demoPath)
	require.NoError(t, err)
	require.Len(t, detail.PlayerSummaries, 46)
}

func TestParse(t *testing.T) {
	inputResponse, errOpen := os.Open(fs.FindFile(path.Join("testdata", "demostats-good.json"), "gbans"))
	require.NoError(t, errOpen)
	good, errGood := demostats.ParseReader(inputResponse)
	require.NoError(t, errGood)
	require.Len(t, good.PlayerSummaries, 76)

	p, uid, errFind := good.Player(steamid.New("[U:1:46625173]"))
	require.NoError(t, errFind)
	require.Equal(t, "238", uid)
	// TODO Fill in as support is implemented
	require.EqualValues(t, demostats.Player{
		Name:               "Josephology",
		SteamID:            "[U:1:46625173]",
		Team:               demostats.TeamOther,
		TimeStart:          0,
		TimeEnd:            0,
		Points:             13,
		ConnectionCount:    1,
		BonusPoints:        0,
		Kills:              6,
		ScoreboardKills:    6,
		PostroundKills:     0,
		Assists:            1,
		ScoreboardAssists:  0,
		PostroundAssists:   0,
		Suicides:           0,
		Deaths:             7,
		ScoreboardDeaths:   7,
		PostroundDeaths:    0,
		Defenses:           0,
		Dominations:        0,
		Dominated:          0,
		Revenges:           0,
		Damage:             0,
		DamageTaken:        0,
		HealingTaken:       0,
		HealthPacks:        0,
		HealingPacks:       0,
		Captures:           0,
		CapturesBlocked:    0,
		Extinguishes:       0,
		BuildingBuilt:      0,
		BuildingsDestroyed: 0,
		Airshots:           1,
		Ubercharges:        0,
		Headshots:          0,
		Shots:              0,
		Hits:               0,
		Teleports:          0,
		Backstabs:          0,
		Support:            0,
		DamageDealt:        2434,
		Healing:            demostats.Healing{},
		Classes:            struct{}{},
		Killstreaks:        []demostats.Killstreak{},
		Weapons:            demostats.Weapons{},
	}, p)
}
