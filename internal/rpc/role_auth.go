package rpc

import (
	"context"
	"net/http"

	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

// RolePermissionsResolver resolves the granular permissions granted to a user
// based on their assigned roles. Implemented by roles.Roles.
type RolePermissionsResolver interface {
	PermissionsBySteamID(ctx context.Context, steamID steamid.SteamID) ([]string, error)
}

// RoleAuth authorizes users based on the granular permissions granted by their
// assigned roles. It resolves the user's permissions at request time via the
// steam_id from the JWT token, so role changes take effect immediately.
type RoleAuth struct {
	resolver RolePermissionsResolver
}

// NewRoleAuth creates a RoleAuth middleware backed by the given resolver.
func NewRoleAuth(resolver RolePermissionsResolver) *RoleAuth {
	return &RoleAuth{resolver: resolver}
}

// WithOneOf returns a UserRouteAuthFn that grants access when the user
// holds any of the required permissions in their role-derived permission set.
// It is fail-closed: users with no role assignments, or when permission
// resolution errors, are denied.
func (m *RoleAuth) WithOneOf(required ...rolesv1.Permission) UserRouteAuthFn {
	return func(ctx context.Context, _ *http.Request, user UserInfo) bool {
		perms, errPerms := m.resolver.PermissionsBySteamID(ctx, user.GetSteamID())
		if errPerms != nil {
			return false
		}

		granted := make(map[string]struct{}, len(perms))
		for _, perm := range perms {
			granted[perm] = struct{}{}
		}

		for _, reqPerm := range required {
			if _, ok := granted[reqPerm.String()]; ok {
				return true
			}
		}

		return false
	}
}
