package roles

import (
	"context"
	"errors"
	"time"

	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	ErrRoleNotFound       = errors.New("role not found")
	ErrRoleExists         = errors.New("role already exists")
	ErrAdminRoleProtected = errors.New("admin role cannot be deleted")
)

const adminRoleID = 1

type Role struct {
	RoleID      int32
	RoleName    string
	Permissions []string
	CreatedOn   time.Time
	UpdatedOn   time.Time
	UserCount   uint64
}

type Roles struct {
	repo Repository
}

func NewRoles(repo Repository) Roles {
	return Roles{repo: repo}
}

func (r Roles) GetAll(ctx context.Context) ([]Role, error) {
	return r.repo.GetAll(ctx)
}

func (r Roles) GetByID(ctx context.Context, roleID int32) (Role, error) {
	role, err := r.repo.GetByID(ctx, roleID)
	if err != nil {
		if errors.Is(err, database.ErrNoResult) {
			return Role{}, ErrRoleNotFound
		}

		return Role{}, err
	}

	return role, nil
}

func (r Roles) Create(ctx context.Context, roleName string, permissions []string) (Role, error) {
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

func (r Roles) Edit(ctx context.Context, roleID int32, roleName string, permissions []string) (Role, error) {
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

func (r Roles) Delete(ctx context.Context, roleID int32) error {
	if roleID == adminRoleID {
		return ErrAdminRoleProtected
	}

	if err := r.repo.Delete(ctx, roleID); err != nil {
		return err
	}

	return nil
}

func (r Roles) Assign(ctx context.Context, steamID steamid.SteamID, roleID int32) error {
	return r.repo.Assign(ctx, steamID, roleID)
}

func (r Roles) GetRolesBySteamID(ctx context.Context, steamID steamid.SteamID) ([]Role, error) {
	return r.repo.GetRolesBySteamID(ctx, steamID)
}

// PermissionsBySteamID returns the union of all granular permissions granted to
// the user via their assigned roles. Duplicate permissions are de-duplicated.
func (r Roles) PermissionsBySteamID(ctx context.Context, steamID steamid.SteamID) ([]string, error) {
	userRoles, err := r.GetRolesBySteamID(ctx, steamID)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(userRoles))
	perms := make([]string, 0, len(userRoles))
	for _, role := range userRoles {
		for _, perm := range role.Permissions {
			if _, ok := seen[perm]; ok {
				continue
			}

			seen[perm] = struct{}{}
			perms = append(perms, perm)
		}
	}

	return perms, nil
}
