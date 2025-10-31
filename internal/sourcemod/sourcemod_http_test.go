package sourcemod_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/sourcemod"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

var fixture *tests.Fixture //nolint:gochecknoglobals

func TestMain(m *testing.M) {
	fixture = tests.NewFixture()
	defer fixture.Close()

	m.Run()
}

func TestSourcemod(t *testing.T) {
	var (
		router = fixture.CreateRouter()
		// br     = ban.NewRepository(fixture.Database, fixture.Persons)
		// bans    = ban.NewBans(br, fixture.Persons, fixture.Config, nil, notification.NewNullNotifications())
		tokens        = &tests.AuthTokens{}
		admin         = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)
		moderator     = fixture.CreateTestPerson(t.Context(), tests.ModSID, permission.Moderator)
		authenticator = &tests.StaticAuthenticator{Profile: moderator}
		sm            = sourcemod.New(sourcemod.NewRepository(fixture.Database), fixture.Config, fixture.Persons)
	)

	sourcemod.NewHandler(router, authenticator, nil, sm)

	// Non-admin should be 403
	tests.Endpoint(t, router, http.MethodGet, "/api/smadmin/admins", nil, http.StatusForbidden, tokens)

	var admins []sourcemod.Admin
	authenticator.Profile = admin
	tests.EndpointReceiver(t, router, http.MethodGet, "/api/smadmin/admins", nil, http.StatusOK, tokens, &admins)
	require.Empty(t, admins)

	randUser := steamid.RandSID64()

	var adminRecv sourcemod.Admin
	tests.EndpointReceiver(t, router, http.MethodPost, "/api/smadmin/admins", sourcemod.CreateAdminRequest{
		AuthType: sourcemod.AuthTypeSteam,
		Identity: randUser.String(),
		Password: "",
		Flags:    "z",
		Name:     "admin test",
		Immunity: 100,
	}, http.StatusOK, tokens, &adminRecv)

	authenticator.Profile = admin
	tests.EndpointReceiver(t, router, http.MethodGet, "/api/smadmin/admins", nil, http.StatusOK, tokens, &admins)
	require.Len(t, admins, 1)

	adminUpdate := adminRecv
	adminUpdate.Name = adminRecv.Name + "xxx"
	tests.EndpointReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/smadmin/admins/%d", adminUpdate.AdminID), nil, http.StatusOK, tokens, &adminUpdate)
	require.Len(t, admins, 1)

	var updatesAdmins []sourcemod.Admin
	tests.EndpointReceiver(t, router, http.MethodGet, "/api/smadmin/admins", nil, http.StatusOK, tokens, &updatesAdmins)

	require.Len(t, updatesAdmins, 1)
	require.Equal(t, adminUpdate, updatesAdmins[0])
}
