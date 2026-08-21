package roles

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/leighmacdonald/gbans/internal/database"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	ErrRoleNotFound       = errors.New("role not found")
	ErrRoleExists         = errors.New("role already exists")
	ErrAdminRoleProtected = errors.New("admin role cannot be deleted")
)

const adminRoleID = 1

// Legacy levels derived from role assignments for the person.permission_level
// column. The column predates the RBAC system and is only maintained for
// reporting/display purposes.
const (
	LegacyLevelBanned int32 = 0
	LegacyLevelUser   int32 = 10
	LegacyLevelMod    int32 = 50
	LegacyLevelAdmin  int32 = 100
)

// UserRoleName is the name of the default role granted to any authenticated
// user without explicit role assignments.
const UserRoleName = "user"

type Role struct {
	RoleID      int32
	RoleName    string
	Permissions []string
	CreatedOn   time.Time
	UpdatedOn   time.Time
	UserCount   uint64
}

type Roles struct {
	repo  Repository
	owner steamid.SteamID
}

func NewRoles(repo Repository, owner steamid.SteamID) *Roles {
	return &Roles{repo: repo, owner: owner}
}

func (r *Roles) GetAll(ctx context.Context) ([]Role, error) {
	return r.repo.GetAll(ctx)
}

func (r *Roles) GetByID(ctx context.Context, roleID int32) (Role, error) {
	role, err := r.repo.GetByID(ctx, roleID)
	if err != nil {
		if errors.Is(err, database.ErrNoResult) {
			return Role{}, ErrRoleNotFound
		}

		return Role{}, err
	}

	return role, nil
}

func (r *Roles) Create(ctx context.Context, roleName string, permissions []string) (Role, error) {
	role := Role{
		RoleName:    roleName,
		Permissions: permissions,
		CreatedOn:   time.Now(),
		UpdatedOn:   time.Now(),
	}

	if err := r.repo.Save(ctx, &role); err != nil {
		return Role{}, err
	}

	return role, nil
}

func (r *Roles) Edit(ctx context.Context, roleID int32, roleName string, permissions []string) (Role, error) {
	role, err := r.GetByID(ctx, roleID)
	if err != nil {
		return Role{}, err
	}

	role.RoleName = roleName
	role.Permissions = permissions
	role.UpdatedOn = time.Now()

	if err := r.repo.Save(ctx, &role); err != nil {
		return Role{}, err
	}

	return role, nil
}

func (r *Roles) Delete(ctx context.Context, roleID int32) error {
	if roleID == adminRoleID {
		return ErrAdminRoleProtected
	}

	if err := r.repo.Delete(ctx, roleID); err != nil {
		return err
	}

	return nil
}

func (r *Roles) Assign(ctx context.Context, steamID steamid.SteamID, roleID int32) error {
	if err := r.repo.Assign(ctx, steamID, roleID); err != nil {
		return err
	}

	return r.updateLegacyLevel(ctx, steamID)
}

// Unassign removes the role from the steam ID. The admin role cannot be
// unassigned from the configured owner to prevent locking out the root user.
func (r *Roles) Unassign(ctx context.Context, steamID steamid.SteamID, roleID int32) error {
	if roleID == adminRoleID && steamID.Equal(r.owner) {
		return ErrAdminRoleProtected
	}

	if err := r.repo.Unassign(ctx, steamID, roleID); err != nil {
		return err
	}

	return r.updateLegacyLevel(ctx, steamID)
}

// EffectivePrivilegeLevel derives the legacy privilege level represented by
// the given roles. It is used to keep the person.permission_level column
// populated for reporting/display; it is not used for authorization.
func EffectivePrivilegeLevel(roles []Role) int32 {
	for _, role := range roles {
		switch role.RoleName {
		case "banned":
			return LegacyLevelBanned
		case "admin":
			return LegacyLevelAdmin
		case "moderator":
			return LegacyLevelMod
		}
	}

	return LegacyLevelUser
}

func (r *Roles) updateLegacyLevel(ctx context.Context, steamID steamid.SteamID) error {
	userRoles, errRoles := r.GetRolesBySteamID(ctx, steamID)
	if errRoles != nil {
		return errRoles
	}

	return r.repo.SetPermissionLevel(ctx, steamID, EffectivePrivilegeLevel(userRoles))
}

// AssignAdminRole grants the admin role to the given steam id. Used to
// bootstrap the owner during first time setup.
func (r *Roles) AssignAdminRole(ctx context.Context, steamID steamid.SteamID) error {
	return r.Assign(ctx, steamID, adminRoleID)
}

// AssignUserRole ensures the default "user" role is assigned to the steam ID.
// It is a no-op when the user already holds any role assignments. Concurrent
// assignments are tolerated via the role_assignments primary key.
func (r *Roles) AssignUserRole(ctx context.Context, steamID steamid.SteamID) error {
	userRoles, err := r.GetRolesBySteamID(ctx, steamID)
	if err != nil {
		return err
	}

	if len(userRoles) > 0 {
		return nil
	}

	userRole, errRole := r.repo.GetByName(ctx, UserRoleName)
	if errRole != nil {
		return errRole
	}

	return r.Assign(ctx, steamID, userRole.RoleID)
}

func (r *Roles) GetRolesBySteamID(ctx context.Context, steamID steamid.SteamID) ([]Role, error) {
	return r.repo.GetRolesBySteamID(ctx, steamID)
}

// PermissionsBySteamID returns the union of all granular permissions granted to
// the user via their assigned roles. Duplicate permissions are de-duplicated.
//
// Users without any role assignments are lazily granted the default "user" role
// so that they always receive the baseline set of user-level permissions.
func (r *Roles) PermissionsBySteamID(ctx context.Context, steamID steamid.SteamID) []rolesv1.Permission {
	if errAssign := r.AssignUserRole(ctx, steamID); errAssign != nil && !errors.Is(errAssign, database.ErrDuplicate) {
		slog.Error("Failed to setup user roles", slog.String("error", errAssign.Error()))

		return nil
	}

	userRoles, err := r.GetRolesBySteamID(ctx, steamID)
	if err != nil {
		slog.Error("Failed to load steamid roles", slog.String("error", err.Error()))

		return nil
	}

	seen := make(map[string]struct{}, len(userRoles))
	var perms []rolesv1.Permission
	for _, role := range userRoles {
		for _, perm := range role.Permissions {
			if _, ok := seen[perm]; ok {
				continue
			}

			seen[perm] = struct{}{}
			perms = append(perms, rolesv1.Permission(rolesv1.Permission_value[perm]))
		}
	}

	return perms
}
