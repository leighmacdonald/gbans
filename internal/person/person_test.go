package person_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

func TestPerson(t *testing.T) {
	testFixture := tests.NewFixture()
	defer testFixture.Close()

	personCase := person.NewPersons(person.NewRepository(testFixture.Config, testFixture.Database), tests.OwnerSID, nil)

	_, err := personCase.GetPersonBySteamID(t.Context(), nil, tests.UserSID)
	require.Error(t, err)

	user := person.New(tests.UserSID)
	require.NoError(t, personCase.SavePerson(t.Context(), nil, &user))

	fetched, errFetched := personCase.GetPersonBySteamID(t.Context(), nil, tests.UserSID)
	require.NoError(t, errFetched)

	fetched.PermissionLevel = permission.Moderator
	fetched.PersonaName = stringutil.SecureRandomString(10)

	require.NoError(t, personCase.SavePerson(t.Context(), nil, &fetched))

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
	//require.EqualValues(t, updateValues.CenterProjectiles, settings.CenterProjectiles)
	require.Equal(t, updateValues.StatsHidden, settings.StatsHidden)

	players, _, errPlayers := personCase.GetPeople(t.Context(), nil, person.PlayerQuery{})
	require.NoError(t, errPlayers)
	require.True(t, len(players) > 0)
}
