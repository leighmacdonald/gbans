package rpc_test

import (
	"context"
	"net/http"
	"testing"

	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

type fakeRoleResolver struct {
	perms []string
	err   error
}

func (f fakeRoleResolver) PermissionsBySteamID(_ context.Context, _ steamid.SteamID) ([]string, error) {
	return f.perms, f.err
}

func TestRoleAuthWithPermissions(t *testing.T) {
	t.Parallel()

	const (
		banRead  = rolesv1.Permission_PERMISSION_BAN_READ
		banWrite = rolesv1.Permission_PERMISSION_BAN_WRITE
		roleRead = rolesv1.Permission_PERMISSION_ROLE_READ
	)

	steamID := steamid.New(76561198000123456)
	user := rpc.UserInfo{SteamID: steamID}

	req, errReq := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com", nil)
	require.NoError(t, errReq)

	tests := []struct {
		name     string
		resolver rpc.RolePermissionsResolver
		required []rolesv1.Permission
		want     bool
	}{
		{
			name:     "granted when any permission matches",
			resolver: fakeRoleResolver{perms: []string{banRead.String()}},
			required: []rolesv1.Permission{roleRead, banRead},
			want:     true,
		},
		{
			name:     "denied when no required permission held",
			resolver: fakeRoleResolver{perms: []string{banWrite.String()}},
			required: []rolesv1.Permission{roleRead, banRead},
			want:     false,
		},
		{
			name:     "denied for user with no role assignments",
			resolver: fakeRoleResolver{perms: nil},
			required: []rolesv1.Permission{banRead},
			want:     false,
		},
		{
			name:     "denied on resolver error",
			resolver: fakeRoleResolver{err: context.DeadlineExceeded},
			required: []rolesv1.Permission{banRead},
			want:     false,
		},
		{
			name:     "denied when no permissions required",
			resolver: fakeRoleResolver{perms: []string{banRead.String()}},
			required: nil,
			want:     false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			auth := rpc.NewRoleAuth(testCase.resolver)
			authFn := auth.WithOneOf(testCase.required...)

			require.Equal(t, testCase.want, authFn(context.Background(), req, user))
		})
	}
}
