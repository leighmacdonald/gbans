package roles_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/roles"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
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

func TestAssignUserRole(t *testing.T) {
	t.Parallel()

	repo := roles.NewRepository(fixture.Database)
	service := roles.NewRoles(repo, tests.OwnerSID)
	ctx := t.Context()

	t.Run("assigns user role when user has no roles", func(t *testing.T) {
		t.Parallel()

		sid := steamid.New(76561198000123457)
		fixture.CreateTestPerson(ctx, sid)

		require.NoError(t, service.AssignUserRole(ctx, sid))

		userRoles, err := service.GetRolesBySteamID(ctx, sid)
		require.NoError(t, err)
		require.Len(t, userRoles, 1)
		require.Equal(t, roles.UserRoleName, userRoles[0].RoleName)
		require.Contains(t, userRoles[0].Permissions, rolesv1.Permission_PERMISSION_LOGIN.String())
	})

	t.Run("no-op when user already has a role", func(t *testing.T) {
		t.Parallel()

		sid := steamid.New(76561198000123458)
		fixture.CreateTestPerson(ctx, sid)

		adminRole, err := repo.GetByName(ctx, "admin")
		require.NoError(t, err)
		require.NoError(t, service.Assign(ctx, sid, adminRole.RoleID))

		require.NoError(t, service.AssignUserRole(ctx, sid))

		userRoles, err := service.GetRolesBySteamID(ctx, sid)
		require.NoError(t, err)
		require.Len(t, userRoles, 1)
		require.Equal(t, "admin", userRoles[0].RoleName)
	})

	t.Run("idempotent on repeated calls", func(t *testing.T) {
		t.Parallel()

		sid := steamid.New(76561198000123459)
		fixture.CreateTestPerson(ctx, sid)

		require.NoError(t, service.AssignUserRole(ctx, sid))
		require.NoError(t, service.AssignUserRole(ctx, sid))

		userRoles, err := service.GetRolesBySteamID(ctx, sid)
		require.NoError(t, err)
		require.Len(t, userRoles, 1)
	})
}

func TestGetAll(t *testing.T) {
	t.Parallel()

	repo := roles.NewRepository(fixture.Database)
	service := roles.NewRoles(repo, tests.OwnerSID)
	ctx := t.Context()

	t.Run("returns default seeded roles", func(t *testing.T) {
		t.Parallel()

		allRoles, err := service.GetAll(ctx)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(allRoles), 5)

		roleNames := make(map[string]bool)
		for _, r := range allRoles {
			roleNames[r.RoleName] = true
		}

		for _, expected := range []string{"admin", "moderator", "streamer", "banned", "user"} {
			require.True(t, roleNames[expected], "expected role %q not found", expected)
		}
	})

	t.Run("includes user count", func(t *testing.T) {
		t.Parallel()

		sid := steamid.New(76561198000124100)
		fixture.CreateTestPerson(ctx, sid)

		adminRole, err := repo.GetByName(ctx, "admin")
		require.NoError(t, err)
		require.NoError(t, service.Assign(ctx, sid, adminRole.RoleID))

		allRoles, err := service.GetAll(ctx)
		require.NoError(t, err)

		for _, r := range allRoles {
			if r.RoleName == "admin" {
				require.GreaterOrEqual(t, r.UserCount, uint64(1))
			}
		}
	})
}

func TestGetByID(t *testing.T) {
	t.Parallel()

	repo := roles.NewRepository(fixture.Database)
	service := roles.NewRoles(repo, tests.OwnerSID)
	ctx := t.Context()

	t.Run("returns existing role", func(t *testing.T) {
		t.Parallel()

		adminRole, err := repo.GetByName(ctx, "admin")
		require.NoError(t, err)

		role, err := service.GetByID(ctx, adminRole.RoleID)
		require.NoError(t, err)
		require.Equal(t, "admin", role.RoleName)
		require.NotZero(t, role.CreatedOn)
		require.NotZero(t, role.UpdatedOn)
	})

	t.Run("returns ErrRoleNotFound for missing role", func(t *testing.T) {
		t.Parallel()

		_, err := service.GetByID(ctx, 999999)
		require.ErrorIs(t, err, roles.ErrRoleNotFound)
	})
}

func TestGetByName(t *testing.T) {
	t.Parallel()

	repo := roles.NewRepository(fixture.Database)
	ctx := t.Context()

	t.Run("returns existing role", func(t *testing.T) {
		t.Parallel()

		role, err := repo.GetByName(ctx, "admin")
		require.NoError(t, err)
		require.Equal(t, "admin", role.RoleName)
		require.NotZero(t, role.RoleID)
		require.NotEmpty(t, role.Permissions)
	})

	t.Run("returns ErrNoResult for missing role", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetByName(ctx, "nonexistent")
		require.ErrorIs(t, err, database.ErrNoResult)
	})
}

func TestCreate(t *testing.T) {
	t.Parallel()

	repo := roles.NewRepository(fixture.Database)
	service := roles.NewRoles(repo, tests.OwnerSID)
	ctx := t.Context()

	t.Run("creates role with permissions", func(t *testing.T) {
		t.Parallel()

		permStrs := []string{
			rolesv1.Permission_PERMISSION_BAN_READ.String(),
			rolesv1.Permission_PERMISSION_BAN_WRITE.String(),
		}
		role, err := service.Create(ctx, "test-role-create", permStrs)
		require.NoError(t, err)
		require.NotZero(t, role.RoleID)
		require.Equal(t, "test-role-create", role.RoleName)
		require.ElementsMatch(t, permStrs, role.Permissions)
		require.False(t, role.CreatedOn.IsZero())
		require.False(t, role.UpdatedOn.IsZero())

		fetched, err := service.GetByID(ctx, role.RoleID)
		require.NoError(t, err)
		require.ElementsMatch(t, permStrs, fetched.Permissions)
	})

	t.Run("creates role with no permissions", func(t *testing.T) {
		t.Parallel()

		role, err := service.Create(ctx, "test-role-noperms", nil)
		require.NoError(t, err)
		require.NotZero(t, role.RoleID)
		require.Empty(t, role.Permissions)
	})

	t.Run("duplicate name returns error", func(t *testing.T) {
		t.Parallel()

		_, err := service.Create(ctx, "test-role-dup", nil)
		require.NoError(t, err)

		_, err = service.Create(ctx, "test-role-dup", nil)
		require.ErrorIs(t, err, database.ErrDuplicate)
	})
}

func TestEdit(t *testing.T) {
	t.Parallel()

	repo := roles.NewRepository(fixture.Database)
	service := roles.NewRoles(repo, tests.OwnerSID)
	ctx := t.Context()

	t.Run("updates role name", func(t *testing.T) {
		t.Parallel()

		role, err := service.Create(ctx, "test-edit-name", nil)
		require.NoError(t, err)

		updated, err := service.Edit(ctx, role.RoleID, "test-edit-name-renamed", nil)
		require.NoError(t, err)
		require.Equal(t, "test-edit-name-renamed", updated.RoleName)
	})

	t.Run("updates permissions", func(t *testing.T) {
		t.Parallel()

		role, err := service.Create(ctx, "test-edit-perms", []string{
			rolesv1.Permission_PERMISSION_BAN_READ.String(),
		})
		require.NoError(t, err)

		updated, err := service.Edit(ctx, role.RoleID, "test-edit-perms", []string{
			rolesv1.Permission_PERMISSION_BAN_READ.String(),
			rolesv1.Permission_PERMISSION_BAN_WRITE.String(),
		})
		require.NoError(t, err)
		require.ElementsMatch(t, []string{
			rolesv1.Permission_PERMISSION_BAN_READ.String(),
			rolesv1.Permission_PERMISSION_BAN_WRITE.String(),
		}, updated.Permissions)
	})

	t.Run("removes permissions", func(t *testing.T) {
		t.Parallel()

		role, err := service.Create(ctx, "test-edit-rm", []string{
			rolesv1.Permission_PERMISSION_BAN_READ.String(),
			rolesv1.Permission_PERMISSION_BAN_WRITE.String(),
		})
		require.NoError(t, err)

		updated, err := service.Edit(ctx, role.RoleID, "test-edit-rm", []string{
			rolesv1.Permission_PERMISSION_BAN_READ.String(),
		})
		require.NoError(t, err)
		require.ElementsMatch(t, []string{
			rolesv1.Permission_PERMISSION_BAN_READ.String(),
		}, updated.Permissions)
	})

	t.Run("replaces all permissions", func(t *testing.T) {
		t.Parallel()

		role, err := service.Create(ctx, "test-edit-replace", []string{
			rolesv1.Permission_PERMISSION_BAN_READ.String(),
		})
		require.NoError(t, err)

		updated, err := service.Edit(ctx, role.RoleID, "test-edit-replace", []string{
			rolesv1.Permission_PERMISSION_FORUM_READ.String(),
			rolesv1.Permission_PERMISSION_FORUM_WRITE.String(),
		})
		require.NoError(t, err)
		require.ElementsMatch(t, []string{
			rolesv1.Permission_PERMISSION_FORUM_READ.String(),
			rolesv1.Permission_PERMISSION_FORUM_WRITE.String(),
		}, updated.Permissions)
	})

	t.Run("edit non-existent returns ErrRoleNotFound", func(t *testing.T) {
		t.Parallel()

		_, err := service.Edit(ctx, 999999, "nonexistent", nil)
		require.ErrorIs(t, err, roles.ErrRoleNotFound)
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()

	repo := roles.NewRepository(fixture.Database)
	service := roles.NewRoles(repo, tests.OwnerSID)
	ctx := t.Context()

	t.Run("deletes non-admin role", func(t *testing.T) {
		t.Parallel()

		role, err := service.Create(ctx, "test-delete", nil)
		require.NoError(t, err)

		require.NoError(t, service.Delete(ctx, role.RoleID))

		_, err = service.GetByID(ctx, role.RoleID)
		require.ErrorIs(t, err, roles.ErrRoleNotFound)
	})

	t.Run("deleting admin role returns ErrAdminRoleProtected", func(t *testing.T) {
		t.Parallel()

		adminRole, err := repo.GetByName(ctx, "admin")
		require.NoError(t, err)

		err = service.Delete(ctx, adminRole.RoleID)
		require.ErrorIs(t, err, roles.ErrAdminRoleProtected)
	})

	t.Run("deleting non-existent role does not return error", func(t *testing.T) {
		t.Parallel()

		err := service.Delete(ctx, 999999)
		require.NoError(t, err)
	})
}

func TestAssign(t *testing.T) {
	t.Parallel()

	repo := roles.NewRepository(fixture.Database)
	service := roles.NewRoles(repo, tests.OwnerSID)
	ctx := t.Context()

	t.Run("assigns role to user", func(t *testing.T) {
		t.Parallel()

		sid := steamid.New(76561198000125100)
		fixture.CreateTestPerson(ctx, sid)

		moderatorRole, err := repo.GetByName(ctx, "moderator")
		require.NoError(t, err)

		require.NoError(t, service.Assign(ctx, sid, moderatorRole.RoleID))

		userRoles, err := service.GetRolesBySteamID(ctx, sid)
		require.NoError(t, err)
		require.Len(t, userRoles, 1)
		require.Equal(t, "moderator", userRoles[0].RoleName)
	})

	t.Run("assigns multiple roles to user", func(t *testing.T) {
		t.Parallel()

		sid := steamid.New(76561198000125101)
		fixture.CreateTestPerson(ctx, sid)

		moderatorRole, err := repo.GetByName(ctx, "moderator")
		require.NoError(t, err)
		streamerRole, err := repo.GetByName(ctx, "streamer")
		require.NoError(t, err)

		require.NoError(t, service.Assign(ctx, sid, moderatorRole.RoleID))
		require.NoError(t, service.Assign(ctx, sid, streamerRole.RoleID))

		userRoles, err := service.GetRolesBySteamID(ctx, sid)
		require.NoError(t, err)
		require.Len(t, userRoles, 2)
	})
}

func TestUnassign(t *testing.T) {
	t.Parallel()

	repo := roles.NewRepository(fixture.Database)
	service := roles.NewRoles(repo, tests.OwnerSID)
	ctx := t.Context()

	t.Run("removes role from user", func(t *testing.T) {
		t.Parallel()

		sid := steamid.New(76561198000126100)
		fixture.CreateTestPerson(ctx, sid)

		moderatorRole, err := repo.GetByName(ctx, "moderator")
		require.NoError(t, err)

		require.NoError(t, service.Assign(ctx, sid, moderatorRole.RoleID))
		require.NoError(t, service.Unassign(ctx, sid, moderatorRole.RoleID))

		userRoles, err := service.GetRolesBySteamID(ctx, sid)
		require.NoError(t, err)
		require.Empty(t, userRoles)
	})

	t.Run("unassigning admin from owner returns ErrAdminRoleProtected", func(t *testing.T) {
		t.Parallel()

		fixture.CreateTestPerson(ctx, tests.OwnerSID)
		err := service.AssignAdminRole(ctx, tests.OwnerSID)
		require.True(t, err == nil || errors.Is(err, database.ErrDuplicate))

		err = service.Unassign(ctx, tests.OwnerSID, 1)
		require.ErrorIs(t, err, roles.ErrAdminRoleProtected)
	})

	t.Run("unassigning admin from non-owner succeeds", func(t *testing.T) {
		t.Parallel()

		sid := steamid.New(76561198000126102)
		fixture.CreateTestPerson(ctx, sid)

		require.NoError(t, service.AssignAdminRole(ctx, sid))

		err := service.Unassign(ctx, sid, 1)
		require.NoError(t, err)

		userRoles, err := service.GetRolesBySteamID(ctx, sid)
		require.NoError(t, err)
		require.Empty(t, userRoles)
	})
}

func TestAssignAdminRole(t *testing.T) {
	t.Parallel()

	repo := roles.NewRepository(fixture.Database)
	service := roles.NewRoles(repo, tests.OwnerSID)
	ctx := t.Context()

	t.Run("grants admin role to steam id", func(t *testing.T) {
		t.Parallel()

		sid := steamid.New(76561198000127100)
		fixture.CreateTestPerson(ctx, sid)

		require.NoError(t, service.AssignAdminRole(ctx, sid))

		userRoles, err := service.GetRolesBySteamID(ctx, sid)
		require.NoError(t, err)
		require.Len(t, userRoles, 1)
		require.Equal(t, "admin", userRoles[0].RoleName)
	})
}

func TestGetRolesBySteamID(t *testing.T) {
	t.Parallel()

	repo := roles.NewRepository(fixture.Database)
	service := roles.NewRoles(repo, tests.OwnerSID)
	ctx := t.Context()

	t.Run("returns roles for user with assignments", func(t *testing.T) {
		t.Parallel()

		sid := steamid.New(76561198000128100)
		fixture.CreateTestPerson(ctx, sid)

		moderatorRole, err := repo.GetByName(ctx, "moderator")
		require.NoError(t, err)
		streamerRole, err := repo.GetByName(ctx, "streamer")
		require.NoError(t, err)

		require.NoError(t, service.Assign(ctx, sid, moderatorRole.RoleID))
		require.NoError(t, service.Assign(ctx, sid, streamerRole.RoleID))

		userRoles, err := service.GetRolesBySteamID(ctx, sid)
		require.NoError(t, err)
		require.Len(t, userRoles, 2)

		roleNames := make(map[string]bool)
		for _, r := range userRoles {
			roleNames[r.RoleName] = true
			require.NotEmpty(t, r.Permissions)
		}
		require.True(t, roleNames["moderator"])
		require.True(t, roleNames["streamer"])
	})

	t.Run("returns empty for user with no assignments", func(t *testing.T) {
		t.Parallel()

		sid := steamid.New(76561198000128101)
		fixture.CreateTestPerson(ctx, sid)

		userRoles, err := service.GetRolesBySteamID(ctx, sid)
		require.NoError(t, err)
		require.Empty(t, userRoles)
	})
}

func TestPermissionsBySteamID(t *testing.T) {
	t.Parallel()

	repo := roles.NewRepository(fixture.Database)
	service := roles.NewRoles(repo, tests.OwnerSID)
	ctx := t.Context()

	t.Run("returns union of permissions from multiple roles", func(t *testing.T) {
		t.Parallel()

		sid := steamid.New(76561198000129100)
		fixture.CreateTestPerson(ctx, sid)

		moderatorRole, err := repo.GetByName(ctx, "moderator")
		require.NoError(t, err)
		streamerRole, err := repo.GetByName(ctx, "streamer")
		require.NoError(t, err)

		require.NoError(t, service.Assign(ctx, sid, moderatorRole.RoleID))
		require.NoError(t, service.Assign(ctx, sid, streamerRole.RoleID))

		perms := service.PermissionsBySteamID(ctx, sid)
		require.NotEmpty(t, perms)

		permSet := make(map[rolesv1.Permission]bool)
		for _, p := range perms {
			permSet[p] = true
		}

		for _, p := range perms {
			require.True(t, permSet[p], "permission %v not found in set", p)
		}
	})

	t.Run("deduplicates overlapping permissions", func(t *testing.T) {
		t.Parallel()

		sid := steamid.New(76561198000129101)
		fixture.CreateTestPerson(ctx, sid)

		_, err := service.Create(ctx, "test-dedup-a", []string{
			rolesv1.Permission_PERMISSION_BAN_READ.String(),
			rolesv1.Permission_PERMISSION_BAN_WRITE.String(),
		})
		require.NoError(t, err)

		roleB, err := service.Create(ctx, "test-dedup-b", []string{
			rolesv1.Permission_PERMISSION_BAN_READ.String(),
			rolesv1.Permission_PERMISSION_FORUM_READ.String(),
		})
		require.NoError(t, err)

		require.NoError(t, service.Assign(ctx, sid, 2))
		require.NoError(t, service.Assign(ctx, sid, roleB.RoleID))

		perms := service.PermissionsBySteamID(ctx, sid)

		permCount := make(map[rolesv1.Permission]int)
		for _, p := range perms {
			permCount[p]++
		}

		for perm, count := range permCount {
			require.Equal(t, 1, count, "permission %v appears %d times (expected 1)", perm, count)
		}
	})

	t.Run("auto-assigns user role for new users", func(t *testing.T) {
		t.Parallel()

		sid := steamid.New(76561198000129102)
		fixture.CreateTestPerson(ctx, sid)

		perms := service.PermissionsBySteamID(ctx, sid)
		require.NotEmpty(t, perms)

		require.True(t, slices.Contains(perms, rolesv1.Permission_PERMISSION_LOGIN),
			"expected PERMISSION_LOGIN in auto-assigned user role")
	})

	t.Run("returns nil for user with no roles assignment error", func(t *testing.T) {
		t.Parallel()

		sid := steamid.New(76561198000129103)

		perms := service.PermissionsBySteamID(ctx, sid)
		require.Nil(t, perms)
	})
}

func TestPermissionsBySteamIDEdgeCases(t *testing.T) {
	t.Parallel()

	repo := roles.NewRepository(fixture.Database)
	service := roles.NewRoles(repo, tests.OwnerSID)
	ctx := t.Context()

	t.Run("returns empty permissions for role with no permissions", func(t *testing.T) {
		t.Parallel()

		sid := steamid.New(76561198000130100)
		fixture.CreateTestPerson(ctx, sid)

		emptyRole, err := service.Create(ctx, "test-empty-perms", nil)
		require.NoError(t, err)

		require.NoError(t, service.Assign(ctx, sid, emptyRole.RoleID))

		perms := service.PermissionsBySteamID(ctx, sid)
		require.Empty(t, perms)
	})

	t.Run("handles role with single permission", func(t *testing.T) {
		t.Parallel()

		sid := steamid.New(76561198000130101)
		fixture.CreateTestPerson(ctx, sid)

		singleRole, err := service.Create(ctx, "test-single-perm", []string{
			rolesv1.Permission_PERMISSION_FORUM_READ.String(),
		})
		require.NoError(t, err)

		require.NoError(t, service.Assign(ctx, sid, singleRole.RoleID))

		perms := service.PermissionsBySteamID(ctx, sid)
		require.Len(t, perms, 1)
		require.Equal(t, rolesv1.Permission_PERMISSION_FORUM_READ, perms[0])
	})
}

func TestSavePermissions(t *testing.T) {
	t.Parallel()

	repo := roles.NewRepository(fixture.Database)
	service := roles.NewRoles(repo, tests.OwnerSID)
	ctx := t.Context()

	t.Run("converges permissions on existing role", func(t *testing.T) {
		t.Parallel()

		role, err := repo.GetByName(ctx, "admin")
		require.NoError(t, err)

		originalPerms := make([]string, len(role.Permissions))
		copy(originalPerms, role.Permissions)

		newPerm := rolesv1.Permission_PERMISSION_WIKI_EDIT.String()
		role.Permissions = append(role.Permissions, newPerm)

		err = repo.Save(ctx, &role)
		require.NoError(t, err)

		fetched, err := repo.GetByName(ctx, "admin")
		require.NoError(t, err)
		require.Contains(t, fetched.Permissions, newPerm)

		role.Permissions = originalPerms
		err = repo.Save(ctx, &role)
		require.NoError(t, err)
	})

	t.Run("removes permissions not in desired set", func(t *testing.T) {
		t.Parallel()

		role, err := service.Create(ctx, "test-rm-perms", []string{
			rolesv1.Permission_PERMISSION_BAN_READ.String(),
			rolesv1.Permission_PERMISSION_FORUM_READ.String(),
		})
		require.NoError(t, err)

		fetched, err := repo.GetByName(ctx, "test-rm-perms")
		require.NoError(t, err)
		require.Contains(t, fetched.Permissions, rolesv1.Permission_PERMISSION_FORUM_READ.String())

		role.Permissions = []string{
			rolesv1.Permission_PERMISSION_BAN_READ.String(),
		}
		err = repo.Save(ctx, &role)
		require.NoError(t, err)

		fetched, err = repo.GetByName(ctx, "test-rm-perms")
		require.NoError(t, err)
		require.NotContains(t, fetched.Permissions, rolesv1.Permission_PERMISSION_FORUM_READ.String())
		require.Contains(t, fetched.Permissions, rolesv1.Permission_PERMISSION_BAN_READ.String())
	})
}

func TestRoleProtection(t *testing.T) {
	t.Parallel()

	repo := roles.NewRepository(fixture.Database)
	service := roles.NewRoles(repo, tests.OwnerSID)
	ctx := t.Context()

	t.Run("admin role ID is 1", func(t *testing.T) {
		t.Parallel()

		adminRole, err := repo.GetByName(ctx, "admin")
		require.NoError(t, err)
		require.Equal(t, int32(1), adminRole.RoleID)
	})

	t.Run("owner cannot have admin unassigned", func(t *testing.T) {
		t.Parallel()

		fixture.CreateTestPerson(ctx, tests.OwnerSID)
		err := service.AssignAdminRole(ctx, tests.OwnerSID)
		require.True(t, err == nil || errors.Is(err, database.ErrDuplicate))

		err = service.Unassign(ctx, tests.OwnerSID, 1)
		require.ErrorIs(t, err, roles.ErrAdminRoleProtected)
	})

	t.Run("admin role cannot be deleted", func(t *testing.T) {
		t.Parallel()

		err := service.Delete(ctx, 1)
		require.ErrorIs(t, err, roles.ErrAdminRoleProtected)
	})
}

func TestRoleAssignmentPermissions(t *testing.T) {
	t.Parallel()

	repo := roles.NewRepository(fixture.Database)
	service := roles.NewRoles(repo, tests.OwnerSID)
	ctx := t.Context()

	t.Run("user with admin role has admin permissions", func(t *testing.T) {
		t.Parallel()

		sid := steamid.New(76561198000131100)
		fixture.CreateTestPerson(ctx, sid)

		require.NoError(t, service.AssignAdminRole(ctx, sid))

		perms := service.PermissionsBySteamID(ctx, sid)
		permSet := make(map[rolesv1.Permission]bool)
		for _, p := range perms {
			permSet[p] = true
		}

		require.True(t, permSet[rolesv1.Permission_PERMISSION_ROLE_READ])
		require.True(t, permSet[rolesv1.Permission_PERMISSION_ROLE_WRITE])
		require.True(t, permSet[rolesv1.Permission_PERMISSION_BAN_READ])
		require.True(t, permSet[rolesv1.Permission_PERMISSION_BAN_WRITE])
	})

	t.Run("user with moderator role has moderator permissions", func(t *testing.T) {
		t.Parallel()

		sid := steamid.New(76561198000131101)
		fixture.CreateTestPerson(ctx, sid)

		modRole, err := repo.GetByName(ctx, "moderator")
		require.NoError(t, err)
		require.NoError(t, service.Assign(ctx, sid, modRole.RoleID))

		perms := service.PermissionsBySteamID(ctx, sid)
		permSet := make(map[rolesv1.Permission]bool)
		for _, p := range perms {
			permSet[p] = true
		}

		require.True(t, permSet[rolesv1.Permission_PERMISSION_BAN_READ])
		require.True(t, permSet[rolesv1.Permission_PERMISSION_BAN_WRITE])
	})

	t.Run("user with user role has baseline permissions", func(t *testing.T) {
		t.Parallel()

		sid := steamid.New(76561198000131102)
		fixture.CreateTestPerson(ctx, sid)

		require.NoError(t, service.AssignUserRole(ctx, sid))

		perms := service.PermissionsBySteamID(ctx, sid)
		permSet := make(map[rolesv1.Permission]bool)
		for _, p := range perms {
			permSet[p] = true
		}

		require.True(t, permSet[rolesv1.Permission_PERMISSION_LOGIN])
	})
}
