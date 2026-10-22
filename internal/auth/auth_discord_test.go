package auth_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/auth"
	"github.com/leighmacdonald/gbans/internal/discord"
	personDomain "github.com/leighmacdonald/gbans/internal/domain/person"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

// loginInfoView mirrors the (unexported) view the login_info template renders against.
type loginInfoView struct {
	Person personDomain.Info
}

func TestLoginInfoTemplate_RendersPermissions(t *testing.T) {
	auth.RegisterDiscordCommands(discord.Discard{})

	person := personDomain.Core{
		SteamID: steamid.New(76561198000000001),
		Name:    "Tester",
		Permissions: []rolesv1.Permission{
			rolesv1.Permission_PERMISSION_BAN_READ,
			rolesv1.Permission_PERMISSION_BAN_WRITE,
		},
	}

	out, err := discord.RenderTemplate("login_info", loginInfoView{Person: person})
	require.NoError(t, err)
	require.Contains(t, out, "PERMISSION_BAN_READ")
	require.Contains(t, out, "PERMISSION_BAN_WRITE")
	require.NotContains(t, out, "Failed to render")
}

func TestLoginInfoTemplate_NoPermissions(t *testing.T) {
	auth.RegisterDiscordCommands(discord.Discard{})

	person := personDomain.Core{
		SteamID: steamid.New(76561198000000001),
		Name:    "Tester",
	}

	out, err := discord.RenderTemplate("login_info", loginInfoView{Person: person})
	require.NoError(t, err)
	require.Contains(t, out, "Permissions: **None**")
}
