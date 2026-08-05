package roles_test

import (
	"testing"

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
	repo := roles.NewRepository(fixture.Database)
	service := roles.NewRoles(repo)
	ctx := t.Context()

	t.Run("assigns user role when user has no roles", func(t *testing.T) {
		sid := steamid.New(76561198000123457)
		fixture.CreateTestPerson(ctx, sid, 0)

		require.NoError(t, service.AssignUserRole(ctx, sid))

		userRoles, err := service.GetRolesBySteamID(ctx, sid)
		require.NoError(t, err)
		require.Len(t, userRoles, 1)
		require.Equal(t, roles.UserRoleName, userRoles[0].RoleName)
		require.Contains(t, userRoles[0].Permissions, rolesv1.Permission_PERMISSION_LOGIN.String())
	})

	t.Run("no-op when user already has a role", func(t *testing.T) {
		sid := steamid.New(76561198000123458)
		fixture.CreateTestPerson(ctx, sid, 0)

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
		sid := steamid.New(76561198000123459)
		fixture.CreateTestPerson(ctx, sid, 0)

		require.NoError(t, service.AssignUserRole(ctx, sid))
		require.NoError(t, service.AssignUserRole(ctx, sid))

		userRoles, err := service.GetRolesBySteamID(ctx, sid)
		require.NoError(t, err)
		require.Len(t, userRoles, 1)
	})
}
