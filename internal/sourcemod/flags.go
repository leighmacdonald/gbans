package sourcemod

import (
	"fmt"
	"slices"
	"strings"

	sq "github.com/Masterminds/squirrel"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
)

// smRolePrefix marks roles created by the sourcemod system. Roles in this
// namespace are sourcemod roles even before they carry any sourcemod
// permission; the prefix is stripped when the group name is served to the
// web ui and game servers.
const smRolePrefix = "sm-"

// smRoleCondition matches every role owned by the sourcemod system: roles in
// the sm- namespace, and any role granted a SOURCEMOD_* permission. The
// permission arm means web roles carrying sourcemod flags are synced to game
// servers as groups without needing the name prefix.
func smRoleCondition(table string) sq.Or {
	table += "."

	return sq.Or{
		sq.Like{table + "role_name": smRolePrefix + "%"},
		sq.Expr(fmt.Sprintf(`EXISTS (
			SELECT 1
			FROM role_permissions AS rp
			WHERE rp.role_id = %srole_id
			  AND rp.permission::text LIKE 'PERMISSION\_SOURCEMOD\_%%'
		)`, table)),
	}
}

// normalizePermissions deduplicates and sorts a permission set so stored and
// returned sets are deterministic.
func normalizePermissions(perms []rolesv1.Permission) []rolesv1.Permission {
	if perms == nil {
		return []rolesv1.Permission{}
	}

	out := make([]rolesv1.Permission, 0, len(perms))
	seen := make(map[rolesv1.Permission]struct{}, len(perms))
	for _, perm := range perms {
		if _, ok := seen[perm]; ok {
			continue
		}

		seen[perm] = struct{}{}
		out = append(out, perm)
	}

	slices.Sort(out)

	return out
}

// deriveImmunity maps a permission set to an SM immunity level. Only the root
// permission implies immunity, and it is always stored as a full 100.
func deriveImmunity(perms []rolesv1.Permission) int32 {
	if slices.Contains(perms, rolesv1.Permission_PERMISSION_SOURCEMOD_ROOT) {
		return 100
	}

	return 0
}

// permissionNames joins the given permission set into a comma separated list
// of enum names.
func permissionNames(perms []rolesv1.Permission) string {
	names := make([]string, 0, len(perms))
	for _, perm := range perms {
		names = append(names, perm.String())
	}

	return strings.Join(names, ", ")
}

// parsePermissions parses a comma or whitespace separated list of permission
// enum names.
func parsePermissions(input string) ([]rolesv1.Permission, error) {
	fields := strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})

	perms := make([]rolesv1.Permission, 0, len(fields))
	for _, field := range fields {
		value, ok := rolesv1.Permission_value[field]
		if !ok {
			return nil, fmt.Errorf("parse permission %q: %w", field, ErrUnknownPermission)
		}

		perms = append(perms, rolesv1.Permission(value))
	}

	return perms, nil
}

func smRoleName(name string) string {
	return smRolePrefix + name
}

func nameFromSMRole(roleName string) string {
	return strings.TrimPrefix(roleName, smRolePrefix)
}
