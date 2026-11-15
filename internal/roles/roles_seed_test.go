package roles_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/roles"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/stretchr/testify/require"
)

func TestSeededAdminRoleHasAllPermissions(t *testing.T) {
	t.Parallel()

	repo := roles.NewRepository(fixture.Database)
	ctx := t.Context()

	role, err := repo.GetByName(ctx, "admin")
	require.NoError(t, err)

	granted := make(map[string]struct{}, len(role.Permissions))
	for _, perm := range role.Permissions {
		granted[perm] = struct{}{}
	}

	for name := range rolesv1.Permission_value {
		if name == "PERMISSION_UNSPECIFIED" {
			continue
		}

		_, ok := granted[name]
		require.Truef(t, ok, "admin role is missing %s", name)
	}
}

func TestSeededRolesOmitWikiRead(t *testing.T) {
	t.Parallel()

	repo := roles.NewRepository(fixture.Database)
	ctx := t.Context()

	for _, name := range []string{"admin", "user"} {
		role, err := repo.GetByName(ctx, name)
		require.NoError(t, err)
		require.NotContains(t, role.Permissions, "PERMISSION_WIKI_READ")
	}
}
