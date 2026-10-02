package person_test

import (
	"math/rand/v2"
	"strconv"
	"testing"

	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

var fixture *tests.Fixture //nolint:gochecknoglobals

func TestMain(m *testing.M) {
	fixture = tests.NewFixture()
	defer fixture.Close()

	m.Run()
}

func TestPerson(t *testing.T) {
	personCase := person.NewPersons(person.NewRepository(fixture.Database, true), fixture.TFApi)

	_, err := personCase.BySteamID(t.Context(), steamid.RandSID64())
	require.Error(t, err)

	for _, sid := range []steamid.SteamID{tests.OwnerSID, tests.ModSID, tests.UserSID, tests.GuestSID} {
		user := person.New(sid)
		require.NoError(t, personCase.Save(t.Context(), &user))
	}

	fetched, errFetched := personCase.BySteamID(t.Context(), tests.UserSID)
	require.NoError(t, errFetched)

	fetched.PersonaName = stringutil.SecureRandomString(10)
	fetched.DiscordID = strconv.FormatInt(rand.Int64(), 10) //nolint:gosec

	require.NoError(t, personCase.Save(t.Context(), &fetched))

	_, errSettings := personCase.GetPersonSettings(t.Context(), fetched.SteamID)
	require.NoError(t, errSettings)

	yes := true
	updateValues := person.SettingsUpdate{
		StatsHidden:       true,
		CenterProjectiles: &yes,
	}

	_, errUpdate := personCase.SavePersonSettings(t.Context(), fetched, updateValues)

	require.NoError(t, errUpdate)
	settings, errSettingsUpdate := personCase.GetPersonSettings(t.Context(), fetched.SteamID)
	require.NoError(t, errSettingsUpdate)

	// TODO fix
	// require.EqualValues(t, updateValues.CenterProjectiles, settings.CenterProjectiles)
	require.Equal(t, updateValues.StatsHidden, settings.StatsHidden)

	players, _, errPlayers := personCase.GetPeople(t.Context(), person.Query{})
	require.NoError(t, errPlayers)
	require.GreaterOrEqual(t, len(players), 4)

	discord, errDisc := personCase.GetPersonByDiscordID(t.Context(), fetched.DiscordID)
	require.NoError(t, errDisc)
	require.Equal(t, fetched.SteamID, discord.SteamID)

	steamID, _ := personCase.BySteamID(t.Context(), fetched.SteamID)
	require.Equal(t, fetched.DiscordID, steamID.DiscordID)
}
