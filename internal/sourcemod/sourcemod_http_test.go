package sourcemod_test

import (
	"fmt"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban"
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

func TestSourcemod(t *testing.T) {
	authenticator := &tests.StaticAuthenticator{}
	router := fixture.CreateRouter()
	sourcemodUC := sourcemod.New(sourcemod.NewRepository(fixture.Database), fixture.Config, fixture.Persons)
	sourcemod.NewHandler(router, authenticator, nil, sourcemodUC)

	t.Run("admins", testAdmins(router, authenticator))
	t.Run("groups", testGroups(router, authenticator))
	t.Run("group_overrides", testGroupOverrides(router, authenticator, sourcemodUC))
	t.Run("global_overrides", testGlobalOverrides(router, authenticator))
	t.Run("group_immunities", testGroupImmunities(router, authenticator, sourcemodUC))
}

func testAdmins(router *gin.Engine, authenticator *tests.StaticAuthenticator) func(t *testing.T) {
	return func(t *testing.T) {
		// Non-admin should be 403
		tests.GetForbidden(t, router, "/api/smadmin/admins")

		// Make sure no results exists yet
		var admins []sourcemod.Admin
		authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)
		tests.GetOK(t, router, "/api/smadmin/admins", &admins)
		require.Empty(t, admins)

		// Create a admin
		randUser := steamid.RandSID64()
		var adminRecv sourcemod.Admin
		tests.PostOK(t, router, "/api/smadmin/admins", sourcemod.CreateAdminRequest{
			AuthType: sourcemod.AuthTypeSteam,
			Identity: randUser.String(),
			Password: "",
			Flags:    "z",
			Name:     "admin test",
			Immunity: 100,
		}, &adminRecv)

		// Fetch admins and verify creation
		authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)
		tests.GetOK(t, router, "/api/smadmin/admins", &admins)
		require.Len(t, admins, 1)

		// Update admin
		adminUpdate := adminRecv
		adminUpdate.Name = adminRecv.Name + "xxx"
		tests.PostOK(t, router, fmt.Sprintf("/api/smadmin/admins/%d", adminUpdate.AdminID), adminUpdate, &adminUpdate)
		require.Len(t, admins, 1)

		// Verify update
		var updatesAdmins []sourcemod.Admin
		tests.GetOK(t, router, "/api/smadmin/admins", &updatesAdmins)
		require.Len(t, updatesAdmins, 1)
		require.Equal(t, adminUpdate.Name, updatesAdmins[0].Name)
	}
}

func testGroups(router *gin.Engine, authenticator *tests.StaticAuthenticator) func(t *testing.T) {
	return func(t *testing.T) {
		// Non-admin should be 403
		authenticator.Profile = fixture.CreateTestPerson(t.Context(), steamid.RandSID64(), permission.User)
		tests.GetForbidden(t, router, "/api/smadmin/groups")

		var groups []sourcemod.Admin
		authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)
		tests.GetOK(t, router, "/api/smadmin/groups", &groups)
		require.Empty(t, groups)

		authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)
		var group sourcemod.Groups
		req := sourcemod.CreateGroupRequest{
			Name:     stringutil.SecureRandomString(10),
			Immunity: 100,
			Flags:    "abc",
		}
		tests.PostCreated(t, router, "/api/smadmin/groups", req, &group)
		require.Equal(t, req.Name, group.Name)
		require.Equal(t, req.Flags, group.Flags)
		require.Equal(t, req.Immunity, group.ImmunityLevel)

		tests.GetOK(t, router, "/api/smadmin/groups", &groups)
		require.NotEmpty(t, groups)

		update := group
		update.Flags = "z"
		update.ImmunityLevel = 50
		update.Name = stringutil.SecureRandomString(10)
		tests.PutOK(t, router, fmt.Sprintf("/api/smadmin/groups/%d", update.GroupID), sourcemod.CreateGroupRequest{
			Name:     update.Name,
			Immunity: update.ImmunityLevel,
			Flags:    update.Flags,
		}, &group)
		require.Equal(t, update.Name, group.Name)
		require.Equal(t, update.Flags, group.Flags)
		require.Equal(t, update.ImmunityLevel, group.ImmunityLevel)

		// Delete the group
		tests.DeleteOK(t, router, fmt.Sprintf("/api/smadmin/groups/%d", group.GroupID), nil)

		// Make sure its deleted
		authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)
		tests.GetOK(t, router, "/api/smadmin/groups", &groups)
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
		tests.GetOK(t, router, fmt.Sprintf("/api/smadmin/groups/%d/overrides", group.GroupID), &overrides)
		require.Empty(t, overrides)

		// Create an override
		var override sourcemod.GroupOverrides
		req := sourcemod.GroupOverrideRequest{
			Name:   stringutil.SecureRandomString(10),
			Type:   sourcemod.OverrideTypeCommand,
			Access: sourcemod.OverrideAccessAllow,
		}
		tests.PostOK(t, router, fmt.Sprintf("/api/smadmin/groups/%d/overrides", group.GroupID), req, &override)
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
		tests.PostOK(t, router, fmt.Sprintf("/api/smadmin/groups_overrides/%d", origID), update, &override)
		require.Equal(t, update.Name, override.Name)
		require.Equal(t, update.Type, override.Type)
		require.Equal(t, update.Access, override.Access)
		require.Equal(t, origID, override.GroupOverrideID)

		// Delete it
		tests.DeleteOK(t, router, fmt.Sprintf("/api/smadmin/groups_overrides/%d", origID), update, nil)

		// Make sure it deleted
		tests.GetOK(t, router, fmt.Sprintf("/api/smadmin/groups/%d/overrides", group.GroupID), &overrides)
		require.Empty(t, overrides)
	}
}

func testGlobalOverrides(router *gin.Engine, authenticator *tests.StaticAuthenticator) func(t *testing.T) {
	return func(t *testing.T) {
		authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)
		// group, errGroup := sm.AddGroup(t.Context(), stringutil.SecureRandomString(10), "abc", 100)
		// require.NoError(t, errGroup)

		var overrides []sourcemod.Overrides
		tests.GetOK(t, router, "/api/smadmin/overrides", &overrides)
		require.Empty(t, overrides)

		// Create
		var override sourcemod.Overrides
		req := sourcemod.OverrideRequest{
			Name:  stringutil.SecureRandomString(10),
			Type:  sourcemod.OverrideTypeCommand,
			Flags: "f",
		}
		tests.PostOK(t, router, "/api/smadmin/overrides", req, &override)
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
		tests.PostOK(t, router, fmt.Sprintf("/api/smadmin/overrides/%d", override.OverrideID), req, &override)
		require.Equal(t, req.Name, override.Name)
		require.Equal(t, req.Type, override.Type)
		require.Equal(t, req.Flags, override.Flags)

		// Delete it
		tests.DeleteOK(t, router, fmt.Sprintf("/api/smadmin/overrides/%d", override.OverrideID), nil)

		// Make sure it deleted
		tests.GetOK(t, router, "/api/smadmin/overrides", &overrides)
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
		tests.GetOK(t, router, "/api/smadmin/group_immunity", &immunities)
		require.Empty(t, immunities)

		// Create
		var groupImmunity sourcemod.GroupImmunity
		req := sourcemod.GroupImmunityRequest{
			GroupID: groupA.GroupID,
			OtherID: groupB.GroupID,
		}
		tests.PostOK(t, router, "/api/smadmin/group_immunity", req, &groupImmunity)
		require.Equal(t, req.GroupID, groupImmunity.Group.GroupID)
		require.Equal(t, req.OtherID, groupImmunity.Other.GroupID)
		require.Positive(t, groupImmunity.GroupImmunityID)

		// Delete it
		tests.DeleteOK(t, router, fmt.Sprintf("/api/smadmin/group_immunity/%d", groupImmunity.GroupImmunityID), nil)

		// Make sure it deleted
		tests.GetOK(t, router, "/api/smadmin/overrides", &immunities)
		require.Empty(t, immunities)
	}
}

func TestSRCDS(t *testing.T) {
	authenticator := &tests.StaticAuthenticator{}
	router := fixture.CreateRouter()
	sm := sourcemod.New(sourcemod.NewRepository(fixture.Database), fixture.Config, fixture.Persons)
	sourcemod.NewHandler(router, authenticator, func(ctx *gin.Context) {
		// Dummy server auth
		ctx.Next()
	}, sm)
	t.Run("permissions", testPermissions(router, authenticator, sm))
	t.Run("check", testCheck(router, authenticator))
}

func testPermissions(router *gin.Engine, _ *tests.StaticAuthenticator, sourcemodUC sourcemod.Sourcemod) func(t *testing.T) {
	return func(t *testing.T) {
		admin, _ := sourcemodUC.AddAdmin(t.Context(), stringutil.SecureRandomString(10), sourcemod.AuthTypeSteam, tests.ModSID.String(), "abc", 0, "")
		group, _ := sourcemodUC.AddGroup(t.Context(), stringutil.SecureRandomString(10), "abc", 0)
		_, _ = sourcemodUC.AddAdminGroup(t.Context(), admin.AdminID, group.GroupID)
		_, _ = sourcemodUC.AddOverride(t.Context(), stringutil.SecureRandomString(10), sourcemod.OverrideTypeCommand, "g")
		_, _ = sourcemodUC.AddOverride(t.Context(), stringutil.SecureRandomString(10), sourcemod.OverrideTypeGroup, "a")

		var users sourcemod.UsersResponse
		tests.GetOK(t, router, "/api/sm/users", &users)
		require.Len(t, users.Users, 1)
		require.Len(t, users.UserGroups, 1)

		var groups sourcemod.GroupsResp
		tests.GetOK(t, router, "/api/sm/groups", &groups)
		require.Len(t, users.Users, 1)

		var overrides []sourcemod.Override
		tests.GetOK(t, router, "/api/sm/overrides", &overrides)
		require.Len(t, overrides, 2)
	}
}

func testCheck(router *gin.Engine, authenticator *tests.StaticAuthenticator) func(t *testing.T) {
	return func(t *testing.T) {
		authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Moderator)

		// Check none exist
		var (
			resp sourcemod.CheckResponse
			req  = sourcemod.CheckRequest{
				SteamID:  tests.UserSID.String(),
				ClientID: 10,
				IP:       "1.2.3.4",
				Name:     stringutil.SecureRandomString(12),
			}
		)
		tests.GetOK(t, router, "/api/sm/check", req, &resp)
		require.Equal(t, ban.OK, resp.BanType)
	}
}
