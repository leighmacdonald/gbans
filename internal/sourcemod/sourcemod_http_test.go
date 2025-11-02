package sourcemod_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/sourcemod"
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

func testAdmins(router *gin.Engine, authenticator *tests.StaticAuthenticator) func(t *testing.T) {
	return func(t *testing.T) {
		// Non-admin should be 403
		tests.Endpoint(t, router, http.MethodGet, "/api/smadmin/admins", nil, http.StatusForbidden, nil)

		// Make sure no results exists yet
		var admins []sourcemod.Admin
		authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)
		tests.EndpointReceiver(t, router, http.MethodGet, "/api/smadmin/admins", nil, http.StatusOK, nil, &admins)
		require.Empty(t, admins)

		// Create a admin
		randUser := steamid.RandSID64()
		var adminRecv sourcemod.Admin
		tests.EndpointReceiver(t, router, http.MethodPost, "/api/smadmin/admins", sourcemod.CreateAdminRequest{
			AuthType: sourcemod.AuthTypeSteam,
			Identity: randUser.String(),
			Password: "",
			Flags:    "z",
			Name:     "admin test",
			Immunity: 100,
		}, http.StatusOK, nil, &adminRecv)

		// Fetch admins and verify creation
		authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)
		tests.EndpointReceiver(t, router, http.MethodGet, "/api/smadmin/admins", nil, http.StatusOK, nil, &admins)
		require.Len(t, admins, 1)

		// Update admin
		adminUpdate := adminRecv
		adminUpdate.Name = adminRecv.Name + "xxx"
		tests.EndpointReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/smadmin/admins/%d", adminUpdate.AdminID), adminUpdate, http.StatusOK, nil, &adminUpdate)
		require.Len(t, admins, 1)

		// Verify update
		var updatesAdmins []sourcemod.Admin
		tests.EndpointReceiver(t, router, http.MethodGet, "/api/smadmin/admins", nil, http.StatusOK, nil, &updatesAdmins)
		require.Len(t, updatesAdmins, 1)
		require.Equal(t, adminUpdate.Name, updatesAdmins[0].Name)
	}
}

func testGroups(router *gin.Engine, authenticator *tests.StaticAuthenticator) func(t *testing.T) {
	return func(t *testing.T) {
		// Non-admin should be 403
		authenticator.Profile = fixture.CreateTestPerson(t.Context(), steamid.RandSID64(), permission.User)
		tests.Endpoint(t, router, http.MethodGet, "/api/smadmin/groups", nil, http.StatusForbidden, nil)

		var groups []sourcemod.Admin
		authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)
		tests.EndpointReceiver(t, router, http.MethodGet, "/api/smadmin/groups", nil, http.StatusOK, nil, &groups)
		require.Empty(t, groups)

		authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)
		var group sourcemod.Groups
		req := sourcemod.CreateGroupRequest{
			Name:     stringutil.SecureRandomString(10),
			Immunity: 100,
			Flags:    "abc",
		}
		tests.EndpointReceiver(t, router, http.MethodPost, "/api/smadmin/groups", req, http.StatusCreated, nil, &group)
		require.Equal(t, req.Name, group.Name)
		require.Equal(t, req.Flags, group.Flags)
		require.Equal(t, req.Immunity, group.ImmunityLevel)

		tests.EndpointReceiver(t, router, http.MethodGet, "/api/smadmin/groups", nil, http.StatusOK, nil, &groups)
		require.NotEmpty(t, groups)

		update := group
		update.Flags = "z"
		update.ImmunityLevel = 50
		update.Name = stringutil.SecureRandomString(10)
		tests.EndpointReceiver(t, router, http.MethodPut, fmt.Sprintf("/api/smadmin/groups/%d", update.GroupID), sourcemod.CreateGroupRequest{
			Name:     update.Name,
			Immunity: update.ImmunityLevel,
			Flags:    update.Flags,
		}, http.StatusOK, nil, &group)
		require.Equal(t, update.Name, group.Name)
		require.Equal(t, update.Flags, group.Flags)
		require.Equal(t, update.ImmunityLevel, group.ImmunityLevel)

		// Delete the group
		tests.Endpoint(t, router, http.MethodDelete, fmt.Sprintf("/api/smadmin/groups/%d", group.GroupID), nil, http.StatusOK, nil)

		// Make sure its deleted
		authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)
		tests.EndpointReceiver(t, router, http.MethodGet, "/api/smadmin/groups", nil, http.StatusOK, nil, &groups)
		require.Empty(t, groups)
	}
}

func testGroupOverrides(router *gin.Engine, authenticator *tests.StaticAuthenticator, sm sourcemod.Sourcemod) func(t *testing.T) {
	return func(t *testing.T) {
		authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)
		group, errGroup := sm.AddGroup(t.Context(), stringutil.SecureRandomString(10), "abc", 100)
		require.NoError(t, errGroup)

		// Make sure none exist
		var overrides []sourcemod.GroupOverrides
		tests.EndpointReceiver(t, router, http.MethodGet, fmt.Sprintf("/api/smadmin/groups/%d/overrides", group.GroupID), nil, http.StatusOK, nil, &overrides)
		require.Empty(t, overrides)

		// Create an override
		var override sourcemod.GroupOverrides
		req := sourcemod.GroupOverrideRequest{
			Name:   stringutil.SecureRandomString(10),
			Type:   sourcemod.OverrideTypeCommand,
			Access: sourcemod.OverrideAccessAllow,
		}
		tests.EndpointReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/smadmin/groups/%d/overrides", group.GroupID), req, http.StatusOK, nil, &override)
		require.Equal(t, req.Name, override.Name)
		require.Equal(t, req.Type, override.Type)
		require.Equal(t, req.Access, override.Access)
		require.Positive(t, override.GroupOverrideID)

		// Update it
		update := sourcemod.GroupOverrideRequest{
			Name:   stringutil.SecureRandomString(10),
			Type:   sourcemod.OverrideTypeGroup,
			Access: sourcemod.OverrideAccessDeny,
		}
		origID := override.GroupOverrideID
		tests.EndpointReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/smadmin/groups_overrides/%d", origID), update, http.StatusOK, nil, &override)
		require.Equal(t, update.Name, override.Name)
		require.Equal(t, update.Type, override.Type)
		require.Equal(t, update.Access, override.Access)
		require.Equal(t, origID, override.GroupOverrideID)

		// Delete it
		tests.Endpoint(t, router, http.MethodDelete, fmt.Sprintf("/api/smadmin/groups_overrides/%d", origID), update, http.StatusOK, nil)

		// Make sure it deleted
		tests.EndpointReceiver(t, router, http.MethodGet, fmt.Sprintf("/api/smadmin/groups/%d/overrides", group.GroupID), nil, http.StatusOK, nil, &overrides)
		require.Empty(t, overrides)
	}
}

func testGlobalOverrides(router *gin.Engine, authenticator *tests.StaticAuthenticator, _ sourcemod.Sourcemod) func(t *testing.T) {
	return func(t *testing.T) {
		authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)
		// group, errGroup := sm.AddGroup(t.Context(), stringutil.SecureRandomString(10), "abc", 100)
		// require.NoError(t, errGroup)

		var overrides []sourcemod.Overrides
		tests.EndpointReceiver(t, router, http.MethodGet, "/api/smadmin/overrides", nil, http.StatusOK, nil, &overrides)
		require.Empty(t, overrides)

		// Create
		var override sourcemod.Overrides
		req := sourcemod.OverrideRequest{
			Name:  stringutil.SecureRandomString(10),
			Type:  sourcemod.OverrideTypeCommand,
			Flags: "f",
		}
		tests.EndpointReceiver(t, router, http.MethodPost, "/api/smadmin/overrides", req, http.StatusOK, nil, &override)
		require.Equal(t, req.Name, override.Name)
		require.Equal(t, req.Type, override.Type)
		require.Equal(t, req.Flags, override.Flags)
		require.Positive(t, override.OverrideID)

		// Update
		req = sourcemod.OverrideRequest{
			Name:  stringutil.SecureRandomString(10),
			Type:  sourcemod.OverrideTypeGroup,
			Flags: "g",
		}
		tests.EndpointReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/smadmin/overrides/%d", override.OverrideID), req, http.StatusOK, nil, &override)
		require.Equal(t, req.Name, override.Name)
		require.Equal(t, req.Type, override.Type)
		require.Equal(t, req.Flags, override.Flags)

		// Delete it
		tests.Endpoint(t, router, http.MethodDelete, fmt.Sprintf("/api/smadmin/overrides/%d", override.OverrideID), nil, http.StatusOK, nil)

		// Make sure it deleted
		tests.EndpointReceiver(t, router, http.MethodGet, "/api/smadmin/overrides", nil, http.StatusOK, nil, &overrides)
		require.Empty(t, overrides)
	}
}

func testGroupImmunities(router *gin.Engine, authenticator *tests.StaticAuthenticator, sm sourcemod.Sourcemod) func(t *testing.T) {
	return func(t *testing.T) {
		groupA, _ := sm.AddGroup(t.Context(), stringutil.SecureRandomString(10), "abc", 0)
		groupB, _ := sm.AddGroup(t.Context(), stringutil.SecureRandomString(10), "abc", 0)

		authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)

		// Check none exist
		var immunities []sourcemod.GroupImmunity
		tests.EndpointReceiver(t, router, http.MethodGet, "/api/smadmin/group_immunity", nil, http.StatusOK, nil, &immunities)
		require.Empty(t, immunities)

		// Create
		var groupImmunity sourcemod.GroupImmunity
		req := sourcemod.GroupImmunityRequest{
			GroupID: groupA.GroupID,
			OtherID: groupB.GroupID,
		}
		tests.EndpointReceiver(t, router, http.MethodPost, "/api/smadmin/group_immunity", req, http.StatusOK, nil, &groupImmunity)
		require.Equal(t, req.GroupID, groupImmunity.Group.GroupID)
		require.Equal(t, req.OtherID, groupImmunity.Other.GroupID)
		require.Positive(t, groupImmunity.GroupImmunityID)

		// Delete it
		tests.Endpoint(t, router, http.MethodDelete, fmt.Sprintf("/api/smadmin/group_immunity/%d", groupImmunity.GroupImmunityID), nil, http.StatusOK, nil)

		// Make sure it deleted
		tests.EndpointReceiver(t, router, http.MethodGet, "/api/smadmin/overrides", nil, http.StatusOK, nil, &immunities)
		require.Empty(t, immunities)
	}
}

func TestSourcemod(t *testing.T) {
	authenticator := &tests.StaticAuthenticator{}
	router := fixture.CreateRouter()
	sm := sourcemod.New(sourcemod.NewRepository(fixture.Database), fixture.Config, fixture.Persons)
	sourcemod.NewHandler(router, authenticator, nil, sm)

	t.Run("admins", testAdmins(router, authenticator))
	t.Run("groups", testGroups(router, authenticator))
	t.Run("group_overrides", testGroupOverrides(router, authenticator, sm))
	t.Run("global_overrides", testGlobalOverrides(router, authenticator, sm))
	t.Run("group_immunities", testGroupImmunities(router, authenticator, sm))
}
