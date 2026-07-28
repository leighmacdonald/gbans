package roles

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type Repository struct {
	database.Database
}

func NewRepository(database database.Database) Repository {
	return Repository{Database: database}
}

func (r Repository) Save(ctx context.Context, role *Role) error {
	now := time.Now()

	role.UpdatedOn = now

	if role.RoleID > 0 {
		if err := database.Err(r.ExecUpdateBuilder(ctx, r.Builder().
			Update("roles").
			SetMap(map[string]any{
				"role_name":  role.RoleName,
				"updated_on": role.UpdatedOn,
			}).
			Where(sq.Eq{"role_id": role.RoleID}))); err != nil {
			return err
		}
	} else {
		role.CreatedOn = now

		if err := database.Err(r.ExecInsertBuilderWithReturnValue(ctx, r.Builder().
			Insert("roles").
			SetMap(map[string]any{
				"role_name":  role.RoleName,
				"created_on": role.CreatedOn,
				"updated_on": role.UpdatedOn,
			}).
			Suffix("RETURNING role_id"), &role.RoleID)); err != nil {
			return err
		}
	}

	if err := r.savePermissions(ctx, role.RoleID, role.Permissions); err != nil {
		return err
	}

	return nil
}

func (r Repository) savePermissions(ctx context.Context, roleID int32, perms []string) error {
	if err := database.Err(r.ExecDeleteBuilder(ctx, r.Builder().
		Delete("role_permissions").
		Where(sq.Eq{"role_id": roleID}))); err != nil {
		return err
	}

	if len(perms) == 0 {
		return nil
	}

	batch := pgx.Batch{}
	for _, perm := range perms {
		batch.Queue("INSERT INTO role_permissions (role_id, permission) VALUES ($1, $2::permission)", roleID, perm)
	}

	if err := r.SendBatch(ctx, &batch).Close(); err != nil {
		return errors.Join(err, database.ErrCloseBatch)
	}

	return nil
}

func (r Repository) GetByID(ctx context.Context, roleID int32) (Role, error) {
	row, errRow := r.QueryRowBuilder(ctx, r.Builder().
		Select("role_id", "role_name", "created_on", "updated_on").
		From("roles").
		Where(sq.Eq{"role_id": roleID}))
	if errRow != nil {
		return Role{}, database.Err(errRow)
	}

	var role Role
	if errScan := row.Scan(&role.RoleID, &role.RoleName, &role.CreatedOn, &role.UpdatedOn); errScan != nil {
		return Role{}, database.Err(errScan)
	}

	perms, errPerms := r.getPermissions(ctx, roleID)
	if errPerms != nil {
		return Role{}, errPerms
	}

	role.Permissions = perms

	return role, nil
}

func (r Repository) GetAll(ctx context.Context) ([]Role, error) {
	roles := make([]Role, 0)

	rows, errRows := r.QueryBuilder(ctx, r.Builder().
		Select("role_id", "role_name", "created_on", "updated_on").
		From("roles"))
	if errRows != nil {
		if errors.Is(errRows, database.ErrNoResult) {
			return roles, nil
		}

		return nil, database.Err(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var role Role
		if errScan := rows.Scan(&role.RoleID, &role.RoleName, &role.CreatedOn, &role.UpdatedOn); errScan != nil {
			return nil, database.Err(errScan)
		}

		roles = append(roles, role)
	}

	for idx := range roles {
		perms, errPerms := r.getPermissions(ctx, roles[idx].RoleID)
		if errPerms != nil {
			return nil, errPerms
		}

		roles[idx].Permissions = perms
	}

	return roles, nil
}

func (r Repository) getPermissions(ctx context.Context, roleID int32) ([]string, error) {
	var perms []string

	rows, errRows := r.QueryBuilder(ctx, r.Builder().
		Select("permission").
		From("role_permissions").
		Where(sq.Eq{"role_id": roleID}))
	if errRows != nil {
		if errors.Is(errRows, database.ErrNoResult) {
			return perms, nil
		}

		return nil, database.Err(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var perm string
		if errScan := rows.Scan(&perm); errScan != nil {
			return nil, database.Err(errScan)
		}

		perms = append(perms, perm)
	}

	return perms, nil
}

func (r Repository) Delete(ctx context.Context, roleID int32) error {
	return database.Err(r.ExecDeleteBuilder(ctx, r.Builder().
		Delete("roles").
		Where(sq.Eq{"role_id": roleID})))
}

func (r Repository) Assign(ctx context.Context, steamID steamid.SteamID, roleID int32) error {
	return database.Err(r.ExecInsertBuilder(ctx, r.Builder().
		Insert("role_assignments").
		SetMap(map[string]any{
			"steam_id":   steamID.Int64(),
			"role_id":    roleID,
			"created_on": time.Now(),
		})))
}

func (r Repository) GetRolesBySteamID(ctx context.Context, steamID steamid.SteamID) ([]Role, error) {
	roles := make([]Role, 0)

	rows, errRows := r.QueryBuilder(ctx, r.Builder().
		Select("r.role_id", "r.role_name", "r.created_on", "r.updated_on").
		From("role_assignments ra").
		Join("roles r USING(role_id)").
		Where(sq.Eq{"ra.steam_id": steamID.Int64()}))
	if errRows != nil {
		if errors.Is(errRows, database.ErrNoResult) {
			return roles, nil
		}

		return nil, database.Err(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var role Role
		if errScan := rows.Scan(&role.RoleID, &role.RoleName, &role.CreatedOn, &role.UpdatedOn); errScan != nil {
			return nil, database.Err(errScan)
		}

		roles = append(roles, role)
	}

	for idx := range roles {
		perms, errPerms := r.getPermissions(ctx, roles[idx].RoleID)
		if errPerms != nil {
			return nil, errPerms
		}

		roles[idx].Permissions = perms
	}

	return roles, nil
}
