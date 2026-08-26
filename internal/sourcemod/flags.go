package sourcemod

import (
	"slices"
	"strings"

	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
)

// smRolePrefix marks roles owned by the sourcemod package. Roles without the
// prefix belong to the web role system and are never touched here.
const smRolePrefix = "sm-"

// flagPermissions returns the permissions granted by a single flag character.
// Flag o has no permission equivalent and returns nil.
func flagPermissions(flag rune) []rolesv1.Permission {
	switch flag {
	case 'a':
		return []rolesv1.Permission{
			rolesv1.Permission_PERMISSION_SOURCEMOD_RESERVED,
			rolesv1.Permission_PERMISSION_SOURCEMOD_GENERIC,
		}
	case 'b':
		return []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_KICK}
	case 'c':
		return []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_BAN}
	case 'd':
		return []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_UNBAN}
	case 'e':
		return []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_SLAY}
	case 'f':
		return []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_CHANGEMAP}
	case 'g':
		return []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_PASSWORD}
	case 'h':
		return []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_CVAR}
	case 'i':
		return []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_CFG}
	case 'j':
		return []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_CHAT}
	case 'k':
		return []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_VOTE}
	case 'l', 'm':
		return []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_RCON}
	case 'n':
		return []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_CHEATS}
	case 'p':
		return []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_CUSTOM_1}
	case 'q':
		return []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_CUSTOM_2}
	case 'r':
		return []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_CUSTOM_3}
	case 's':
		return []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_CUSTOM_4}
	case 't':
		return []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_CUSTOM_5}
	case 'u':
		return []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_CUSTOM_6}
	case 'z':
		return []rolesv1.Permission{rolesv1.Permission_PERMISSION_SOURCEMOD_ROOT}
	default:
		return nil
	}
}

// permissionFlag returns the canonical flag character that grants the given
// permission, or 0 when the permission has no flag equivalent.
func permissionFlag(perm rolesv1.Permission) rune {
	switch perm {
	case rolesv1.Permission_PERMISSION_SOURCEMOD_RESERVED,
		rolesv1.Permission_PERMISSION_SOURCEMOD_GENERIC:
		return 'a'
	case rolesv1.Permission_PERMISSION_SOURCEMOD_KICK:
		return 'b'
	case rolesv1.Permission_PERMISSION_SOURCEMOD_BAN:
		return 'c'
	case rolesv1.Permission_PERMISSION_SOURCEMOD_UNBAN:
		return 'd'
	case rolesv1.Permission_PERMISSION_SOURCEMOD_SLAY:
		return 'e'
	case rolesv1.Permission_PERMISSION_SOURCEMOD_CHANGEMAP:
		return 'f'
	case rolesv1.Permission_PERMISSION_SOURCEMOD_PASSWORD:
		return 'g'
	case rolesv1.Permission_PERMISSION_SOURCEMOD_CVAR:
		return 'h'
	case rolesv1.Permission_PERMISSION_SOURCEMOD_CFG:
		return 'i'
	case rolesv1.Permission_PERMISSION_SOURCEMOD_CHAT:
		return 'j'
	case rolesv1.Permission_PERMISSION_SOURCEMOD_VOTE:
		return 'k'
	case rolesv1.Permission_PERMISSION_SOURCEMOD_RCON:
		return 'l'
	case rolesv1.Permission_PERMISSION_SOURCEMOD_CHEATS:
		return 'n'
	case rolesv1.Permission_PERMISSION_SOURCEMOD_ROOT:
		return 'z'
	case rolesv1.Permission_PERMISSION_SOURCEMOD_CUSTOM_1:
		return 'p'
	case rolesv1.Permission_PERMISSION_SOURCEMOD_CUSTOM_2:
		return 'q'
	case rolesv1.Permission_PERMISSION_SOURCEMOD_CUSTOM_3:
		return 'r'
	case rolesv1.Permission_PERMISSION_SOURCEMOD_CUSTOM_4:
		return 's'
	case rolesv1.Permission_PERMISSION_SOURCEMOD_CUSTOM_5:
		return 't'
	case rolesv1.Permission_PERMISSION_SOURCEMOD_CUSTOM_6:
		return 'u'
	default:
		return 0
	}
}

// flagToPermissions expands a set of flag characters into the union of the
// permissions they grant. Unknown characters are ignored.
func flagToPermissions(flags string) []rolesv1.Permission {
	seen := make(map[rolesv1.Permission]struct{})

	var perms []rolesv1.Permission
	for _, flag := range flags {
		for _, perm := range flagPermissions(flag) {
			if _, ok := seen[perm]; ok {
				continue
			}

			seen[perm] = struct{}{}
			perms = append(perms, perm)
		}
	}

	return perms
}

// permissionsToFlags collapses a set of permissions into the flag characters
// that grant them, ordered by validFlags.
func permissionsToFlags(perms []rolesv1.Permission) string {
	var flags strings.Builder

	for _, flag := range validFlags {
		for _, perm := range perms {
			if permissionFlag(perm) == flag {
				flags.WriteRune(flag)

				break
			}
		}
	}

	return flags.String()
}

// groupPermissions expands a group's flags into its permission set, adding
// the root permission for fully immune groups.
func groupPermissions(flags string, immunity int32) []rolesv1.Permission {
	perms := flagToPermissions(flags)
	if immunity > 0 && !slices.Contains(perms, rolesv1.Permission_PERMISSION_SOURCEMOD_ROOT) {
		perms = append(perms, rolesv1.Permission_PERMISSION_SOURCEMOD_ROOT)
	}

	return perms
}

// deriveImmunity inverts groupPermissions. Only the root permission implies
// immunity, and it is always stored as a full 100.
func deriveImmunity(perms []rolesv1.Permission) int32 {
	if slices.Contains(perms, rolesv1.Permission_PERMISSION_SOURCEMOD_ROOT) {
		return 100
	}

	return 0
}

func smRoleName(name string) string {
	return smRolePrefix + name
}

func nameFromSMRole(roleName string) string {
	return strings.TrimPrefix(roleName, smRolePrefix)
}
