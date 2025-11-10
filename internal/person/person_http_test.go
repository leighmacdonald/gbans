package person_test

import (
	"fmt"
	"testing"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	personDomain "github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/stretchr/testify/require"
)

func TestPersonHTTPQueries(t *testing.T) {
	var (
		authenticator = &tests.UserAuth{Profile: fixture.CreateTestPerson(t.Context(), tests.UserSID, permission.User)}
		persons       = person.NewPersons(person.NewRepository(fixture.Database, true), tests.OwnerSID, fixture.TFApi)
		router        = fixture.CreateRouter()
	)
	person.NewPersonHandler(router, authenticator, persons)

	profile := tests.GetGOK[person.ProfileResponse](t, router, "/api/profile?query=uncledane")
	require.Equal(t, int64(76561198057999536), profile.Player.SteamID.Int64())

	resp1 := tests.GetGOK[person.SteamValidateResponse](t, router, "/api/steam/validate", httphelper.RequestQuery{
		Query: "uncledane",
	})

	require.Equal(t, "76561198057999536", resp1.SteamID)

	tests.GetNotFound(t, router, "/api/steam/validate", httphelper.RequestQuery{Query: "IaishiuPIUHIOASHD"})
}

func TestPersonHTTPProfile(t *testing.T) {
	var (
		authenticator = &tests.UserAuth{Profile: fixture.CreateTestPerson(t.Context(), tests.UserSID, permission.User)}
		persons       = person.NewPersons(person.NewRepository(fixture.Database, true), tests.OwnerSID, fixture.TFApi)
		router        = fixture.CreateRouter()
	)
	person.NewPersonHandler(router, authenticator, persons)

	profile := tests.GetGOK[personDomain.Core](t, router, "/api/current_profile")
	require.Equal(t, authenticator.Profile.SteamID, profile.SteamID)

	settings := tests.GetGOK[person.Settings](t, router, "/api/current_profile/settings")

	upd := person.SettingsUpdate{
		ForumSignature:       settings.ForumSignature,
		ForumProfileMessages: settings.ForumProfileMessages,
		StatsHidden:          !settings.StatsHidden,
		CenterProjectiles:    settings.CenterProjectiles,
	}
	updated := tests.PostGOK[person.Settings](t, router, "/api/current_profile/settings", upd)
	require.Equal(t, upd.StatsHidden, updated.StatsHidden)
}

func TestPersonHTTPPermission(t *testing.T) {
	var (
		user          = fixture.CreateTestPerson(t.Context(), tests.UserSID, permission.User)
		mod           = fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.Moderator)
		admin         = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)
		authenticator = &tests.UserAuth{Profile: user}
		persons       = person.NewPersons(person.NewRepository(fixture.Database, true), tests.OwnerSID, fixture.TFApi)
		router        = fixture.CreateRouter()
	)
	person.NewPersonHandler(router, authenticator, persons)

	tests.PutForbidden(t, router, fmt.Sprintf("/api/player/%s/permissions", tests.UserSID.String()), person.RequestPermissionLevelUpdate{
		PermissionLevel: permission.Admin,
	})

	authenticator.Profile = admin
	person1 := tests.PutGOK[person.Person](t, router, fmt.Sprintf("/api/player/%s/permissions", tests.UserSID.String()), person.RequestPermissionLevelUpdate{
		PermissionLevel: permission.Moderator,
	})
	require.Equal(t, user.SteamID, person1.SteamID)
	require.Equal(t, permission.Moderator, person1.PermissionLevel)

	// Dont allow lower permission levels to override higher ones
	authenticator.Profile = mod
	tests.PutForbidden(t, router, fmt.Sprintf("/api/player/%s/permissions", admin.SteamID.String()), person.RequestPermissionLevelUpdate{
		PermissionLevel: permission.User,
	})

	authenticator.Profile = admin
	tests.PutForbidden(t, router, fmt.Sprintf("/api/player/%s/permissions", admin.SteamID.String()), person.RequestPermissionLevelUpdate{
		PermissionLevel: permission.User,
	})
}
