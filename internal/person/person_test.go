package person_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func TestPerson(t *testing.T) {
	testFixture := tests.NewFixture()
	defer testFixture.Close()

	personCase := person.NewPersons(person.NewRepository(testFixture.Config, testFixture.Database), tests.OwnerSID, nil)

	_, err := personCase.BySteamID(t.Context(), tests.UserSID)
	require.Error(t, err)

	for idx, sid := range []steamid.SteamID{tests.OwnerSID, tests.ModSID, tests.UserSID, tests.GuestSID} {
		user := person.New(sid)
		switch idx {
		case 0:
			user.PermissionLevel = permission.Admin
		case 1:
			user.PermissionLevel = permission.Moderator
		case 2:
			user.PermissionLevel = permission.User
		case 3:
			user.PermissionLevel = permission.Guest
		}
		require.NoError(t, personCase.Save(t.Context(), &user))
	}

	fetched, errFetched := personCase.BySteamID(t.Context(), tests.UserSID)
	require.NoError(t, errFetched)

	fetched.PermissionLevel = permission.Moderator
	fetched.PersonaName = stringutil.SecureRandomString(10)
	fetched.DiscordID = stringutil.SecureRandomString(10)

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

	players, errPlayers := personCase.GetPeople(t.Context(), person.Query{})
	require.NoError(t, errPlayers)
	require.Len(t, players, 4)

	aboveMod, _ := personCase.GetSteamIDsAbove(t.Context(), permission.Moderator)
	require.Len(t, aboveMod, 3)

	discord, _ := personCase.GetPersonByDiscordID(t.Context(), fetched.DiscordID)
	require.EqualExportedValues(t, fetched, discord)

	steamID, _ := personCase.BySteamID(t.Context(), fetched.SteamID)
	require.EqualExportedValues(t, fetched, steamID)
}
