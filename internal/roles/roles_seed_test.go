package roles_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/roles"
	"github.com/stretchr/testify/require"
)

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
