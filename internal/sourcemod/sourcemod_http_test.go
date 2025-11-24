package sourcemod_test

import (
	"fmt"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban/bantype"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/servers"
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
	var (
		authenticator = &tests.UserAuth{}
		router        = fixture.CreateRouter()
		sourcemodUC   = sourcemod.New(sourcemod.NewRepository(fixture.Database), fixture.Persons, notification.NewNullNotifications(), "")
		serversUC     = servers.NewServers(servers.NewRepository(fixture.Database))
	)

	sourcemod.NewHandler(router, authenticator, &tests.ServerAuth{}, sourcemodUC, serversUC, notification.NewNullNotifications())

	t.Run("admins", testAdmins(router, authenticator))
	t.Run("groups", testGroups(router, authenticator))
	t.Run("group_overrides", testGroupOverrides(router, authenticator, sourcemodUC))
	t.Run("global_overrides", testGlobalOverrides(router, authenticator))
	t.Run("group_immunities", testGroupImmunities(router, authenticator, sourcemodUC))
}

func testAdmins(router *gin.Engine, authenticator *tests.UserAuth) func(t *testing.T) {
	return func(t *testing.T) {
		// Non-admin should be 403
		tests.GetForbidden(t, router, "/api/smadmin/admins")

		// Make sure no results exists yet
		authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)
		require.Empty(t, tests.GetGOK[[]sourcemod.Admin](t, router, "/api/smadmin/admins"))

		// Create a admin
		randUser := steamid.RandSID64()
		adminRecv := tests.PostGOK[sourcemod.Admin](t, router, "/api/smadmin/admins", sourcemod.CreateAdminRequest{
			AuthType: sourcemod.AuthTypeSteam,
			Identity: randUser.String(),
			Password: "",
			Flags:    "z",
			Name:     "admin test",
			Immunity: 100,
		})

		// Fetch admins and verify creation
		authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)
		require.Len(t, tests.GetGOK[[]sourcemod.Admin](t, router, "/api/smadmin/admins"), 1)

		// Update admin
		adminUpdate := adminRecv
		adminUpdate.Name = adminRecv.Name + "xxx"
		update := tests.PostGOK[sourcemod.Admin](t, router, fmt.Sprintf("/api/smadmin/admins/%d", adminUpdate.AdminID), adminUpdate)
		require.Equal(t, adminUpdate.Name, update.Name)

		// Verify update
		updatesAdmins := tests.GetGOK[[]sourcemod.Admin](t, router, "/api/smadmin/admins")
		require.Len(t, updatesAdmins, 1)
		require.Equal(t, adminUpdate.Name, updatesAdmins[0].Name)
	}
}

func testGroups(router *gin.Engine, authenticator *tests.UserAuth) func(t *testing.T) {
	return func(t *testing.T) {
		// Non-admin should be 403
		authenticator.Profile = fixture.CreateTestPerson(t.Context(), steamid.RandSID64(), permission.User)
		tests.GetForbidden(t, router, "/api/smadmin/groups")

		authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)
		require.Empty(t, tests.GetGOK[[]sourcemod.Admin](t, router, "/api/smadmin/groups"))

		authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)

		req := sourcemod.CreateGroupRequest{
			Name:     stringutil.SecureRandomString(10),
			Immunity: 100,
			Flags:    "abc",
		}
		group := tests.PostGCreated[sourcemod.Groups](t, router, "/api/smadmin/groups", req)
		require.Equal(t, req.Name, group.Name)
		require.Equal(t, req.Flags, group.Flags)
		require.Equal(t, req.Immunity, group.ImmunityLevel)

		require.NotEmpty(t, tests.GetGOK[[]sourcemod.Admin](t, router, "/api/smadmin/groups"))

		update := group
		update.Flags = "z"
		update.ImmunityLevel = 50
		update.Name = stringutil.SecureRandomString(10)

		group2 := tests.PutGOK[sourcemod.Groups](t, router, fmt.Sprintf("/api/smadmin/groups/%d", update.GroupID), sourcemod.CreateGroupRequest{
			Name:     update.Name,
			Immunity: update.ImmunityLevel,
			Flags:    update.Flags,
		})
		require.Equal(t, update.Name, group2.Name)
		require.Equal(t, update.Flags, group2.Flags)
		require.Equal(t, update.ImmunityLevel, group2.ImmunityLevel)

		// Delete the group
		tests.DeleteOK(t, router, fmt.Sprintf("/api/smadmin/groups/%d", group.GroupID), nil)

		// Make sure its deleted
		authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)
		require.Empty(t, tests.GetGOK[[]sourcemod.Groups](t, router, "/api/smadmin/groups"))
	}
}

func testGroupOverrides(router *gin.Engine, authenticator *tests.UserAuth, sm sourcemod.Sourcemod) func(t *testing.T) {
	return func(t *testing.T) {
		authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)
		group, errGroup := sm.AddGroup(t.Context(), stringutil.SecureRandomString(10), "abc", 100)
		require.NoError(t, errGroup)

		// Make sure none exist
		require.Empty(t, tests.GetGOK[[]sourcemod.GroupOverrides](t, router, fmt.Sprintf("/api/smadmin/groups/%d/overrides", group.GroupID)))

		// Create an override
		req := sourcemod.GroupOverrideRequest{
			Name:   stringutil.SecureRandomString(10),
			Type:   sourcemod.OverrideTypeCommand,
			Access: sourcemod.OverrideAccessAllow,
		}
		override := tests.PostGOK[sourcemod.GroupOverrides](t, router, fmt.Sprintf("/api/smadmin/groups/%d/overrides", group.GroupID), req)
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
		override2 := tests.PostGOK[sourcemod.GroupOverrides](t, router, fmt.Sprintf("/api/smadmin/groups_overrides/%d", origID), update)
		require.Equal(t, update.Name, override2.Name)
		require.Equal(t, update.Type, override2.Type)
		require.Equal(t, update.Access, override2.Access)
		require.Equal(t, origID, override2.GroupOverrideID)

		// Delete it
		tests.DeleteOK(t, router, fmt.Sprintf("/api/smadmin/groups_overrides/%d", origID), update)

		// Make sure it deleted
		require.Empty(t, tests.GetGOK[[]sourcemod.GroupOverrides](t, router, fmt.Sprintf("/api/smadmin/groups/%d/overrides", group.GroupID)))
	}
}

func testGlobalOverrides(router *gin.Engine, authenticator *tests.UserAuth) func(t *testing.T) {
	return func(t *testing.T) {
		authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)
		// group, errGroup := sm.AddGroup(t.Context(), stringutil.SecureRandomString(10), "abc", 100)
		// require.NoError(t, errGroup)

		require.Empty(t, tests.GetGOK[[]sourcemod.Overrides](t, router, "/api/smadmin/overrides"))

		// Create
		req := sourcemod.OverrideRequest{
			Name:  stringutil.SecureRandomString(10),
			Type:  sourcemod.OverrideTypeCommand,
			Flags: "f",
		}
		override := tests.PostGOK[sourcemod.Overrides](t, router, "/api/smadmin/overrides", req)
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
		override2 := tests.PostGOK[sourcemod.Overrides](t, router, fmt.Sprintf("/api/smadmin/overrides/%d", override.OverrideID), req)
		require.Equal(t, req.Name, override2.Name)
		require.Equal(t, req.Type, override2.Type)
		require.Equal(t, req.Flags, override2.Flags)

		// Delete it
		tests.DeleteOK(t, router, fmt.Sprintf("/api/smadmin/overrides/%d", override.OverrideID), nil)

		// Make sure it deleted
		require.Empty(t, tests.GetGOK[[]sourcemod.Overrides](t, router, "/api/smadmin/overrides"))
	}
}

func testGroupImmunities(router *gin.Engine, authenticator *tests.UserAuth, sm sourcemod.Sourcemod) func(t *testing.T) {
	return func(t *testing.T) {
		groupA, _ := sm.AddGroup(t.Context(), stringutil.SecureRandomString(10), "abc", 0)
		groupB, _ := sm.AddGroup(t.Context(), stringutil.SecureRandomString(10), "abc", 0)

		authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)

		// Check none exist
		require.Empty(t, tests.GetGOK[[]sourcemod.GroupImmunity](t, router, "/api/smadmin/group_immunity"))

		// Create
		req := sourcemod.GroupImmunityRequest{
			GroupID: groupA.GroupID,
			OtherID: groupB.GroupID,
		}
		groupImmunity := tests.PostGOK[sourcemod.GroupImmunity](t, router, "/api/smadmin/group_immunity", req)
		require.Equal(t, req.GroupID, groupImmunity.Group.GroupID)
		require.Equal(t, req.OtherID, groupImmunity.Other.GroupID)
		require.Positive(t, groupImmunity.GroupImmunityID)

		// Delete it
		tests.DeleteOK(t, router, fmt.Sprintf("/api/smadmin/group_immunity/%d", groupImmunity.GroupImmunityID), nil)

		// Make sure it deleted
		require.Empty(t, tests.GetGOK[[]sourcemod.GroupImmunity](t, router, "/api/smadmin/overrides"))
	}
}

func TestSRCDS(t *testing.T) {
	var (
		authenticator = &tests.UserAuth{}
		router        = fixture.CreateRouter()
		serversUC     = servers.NewServers(servers.NewRepository(fixture.Database))
		sm            = sourcemod.New(sourcemod.NewRepository(fixture.Database), fixture.Persons, notification.NewNullNotifications(), "")
	)

	sourcemod.NewHandler(router, authenticator, &tests.ServerAuth{}, sm, serversUC, notification.NewNullNotifications())

	t.Run("permissions", testPermissions(router, authenticator, sm))
	t.Run("check", testCheck(router, authenticator))
}

func testPermissions(router *gin.Engine, _ *tests.UserAuth, sourcemodUC sourcemod.Sourcemod) func(t *testing.T) {
	return func(t *testing.T) {
		admin, _ := sourcemodUC.AddAdmin(t.Context(), stringutil.SecureRandomString(10), sourcemod.AuthTypeSteam, tests.ModSID.String(), "abc", 0, "")
		group, _ := sourcemodUC.AddGroup(t.Context(), stringutil.SecureRandomString(10), "abc", 0)
		_, _ = sourcemodUC.AddAdminGroup(t.Context(), admin.AdminID, group.GroupID)
		_, _ = sourcemodUC.AddOverride(t.Context(), stringutil.SecureRandomString(10), sourcemod.OverrideTypeCommand, "g")
		_, _ = sourcemodUC.AddOverride(t.Context(), stringutil.SecureRandomString(10), sourcemod.OverrideTypeGroup, "a")

		users := tests.GetGOK[sourcemod.UsersResponse](t, router, "/api/sm/users")
		require.GreaterOrEqual(t, len(users.Users), 1)
		require.GreaterOrEqual(t, len(users.UserGroups), 1)

		groups := tests.GetGOK[sourcemod.GroupsResp](t, router, "/api/sm/groups")
		require.GreaterOrEqual(t, len(groups.Groups), 1)

		require.GreaterOrEqual(t, len(tests.GetGOK[[]sourcemod.Override](t, router, "/api/sm/overrides")), 2)
	}
}

func testCheck(router *gin.Engine, authenticator *tests.UserAuth) func(t *testing.T) {
	return func(t *testing.T) {
		authenticator.Profile = fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Moderator)

		// Check none exist
		req := sourcemod.CheckRequest{
			SteamID:  tests.UserSID.String(),
			ClientID: 10,
			IP:       "1.2.3.4",
			Name:     stringutil.SecureRandomString(12),
		}
		resp := tests.GetGOK[sourcemod.CheckResponse](t, router, "/api/sm/check", req)
		require.Equal(t, bantype.OK, resp.BanType)
	}
}
