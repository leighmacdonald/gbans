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

// savePermissions converges the role's permission set onto perms: entries no
// longer present are removed and new entries are upserted, while unchanged
// entries are left untouched so concurrent edits cannot wipe the whole set.
func (r Repository) savePermissions(ctx context.Context, roleID int32, perms []string) error {
	existing, errExisting := r.getPermissions(ctx, roleID)
	if errExisting != nil {
		return errExisting
	}

	existingSet := make(map[string]struct{}, len(existing))
	for _, perm := range existing {
		existingSet[perm] = struct{}{}
	}

	desiredSet := make(map[string]struct{}, len(perms))
	for _, perm := range perms {
		desiredSet[perm] = struct{}{}
	}

	for _, perm := range existing {
		if _, ok := desiredSet[perm]; ok {
			continue
		}

		if err := database.Err(r.ExecDeleteBuilder(ctx, r.Builder().
			Delete("role_permissions").
			Where(sq.And{
				sq.Eq{"role_id": roleID},
				sq.Eq{"permission": perm},
			}))); err != nil {
			return err
		}
	}

	if len(perms) == 0 {
		return nil
	}

	now := time.Now()

	seen := make(map[string]struct{}, len(perms))

	for _, perm := range perms {
		if _, ok := existingSet[perm]; ok {
			continue
		}

		if _, dup := seen[perm]; dup {
			continue
		}

		seen[perm] = struct{}{}

		if err := database.Err(r.ExecInsertBuilder(ctx, r.Builder().
			Insert("role_permissions").
			Columns("role_id", "permission", "created_on", "updated_on").
			Values(roleID, perm, now, now).
			Suffix("ON CONFLICT (role_id, permission) DO NOTHING"))); err != nil {
			return err
		}
	}

	return nil
}

func (r Repository) GetByName(ctx context.Context, roleName string) (Role, error) {
	row, errRow := r.QueryRowBuilder(ctx, r.Builder().
		Select("role_id", "role_name", "created_on", "updated_on").
		From("roles").
		Where(sq.Eq{"role_name": roleName}))
	if errRow != nil {
		return Role{}, database.Err(errRow)
	}

	var role Role
	if errScan := row.Scan(&role.RoleID, &role.RoleName, &role.CreatedOn, &role.UpdatedOn); errScan != nil {
		return Role{}, database.Err(errScan)
	}

	perms, errPerms := r.getPermissions(ctx, role.RoleID)
	if errPerms != nil {
		return Role{}, errPerms
	}

	role.Permissions = perms

	return role, nil
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
		Select("r.role_id", "r.role_name", "r.created_on", "r.updated_on", "COUNT(ra.steam_id)").
		From("roles r").
		LeftJoin("role_assignments ra USING(role_id)").
		GroupBy("r.role_id", "r.role_name", "r.created_on", "r.updated_on"))
	if errRows != nil {
		if errors.Is(errRows, database.ErrNoResult) {
			return roles, nil
		}

		return nil, database.Err(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var role Role
		if errScan := rows.Scan(&role.RoleID, &role.RoleName, &role.CreatedOn, &role.UpdatedOn, &role.UserCount); errScan != nil {
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
		}).
		Suffix("ON CONFLICT (steam_id, role_id) DO NOTHING")))
}

func (r Repository) Unassign(ctx context.Context, steamID steamid.SteamID, roleID int32) error {
	return database.Err(r.ExecDeleteBuilder(ctx, r.Builder().
		Delete("role_assignments").
		Where(sq.Eq{"steam_id": steamID.Int64(), "role_id": roleID})))
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

func (r Repository) RoleUsers(ctx context.Context) ([]RoleUser, error) {
	users := make([]RoleUser, 0)

	rows, errRows := r.QueryBuilder(ctx, r.Builder().
		Select("p.steam_id", "p.personaname", "r.role_id", "r.role_name").
		From("role_assignments ra").
		Join("person p ON p.steam_id = ra.steam_id").
		Join("roles r ON r.role_id = ra.role_id").
		OrderBy("p.personaname ASC", "r.role_name ASC"))
	if errRows != nil {
		if errors.Is(errRows, database.ErrNoResult) {
			return users, nil
		}

		return nil, database.Err(errRows)
	}

	defer rows.Close()

	bySteam := make(map[int64]int)

	for rows.Next() {
		var steamID int64
		var personaName string
		var role Role

		if errScan := rows.Scan(&steamID, &personaName, &role.RoleID, &role.RoleName); errScan != nil {
			return nil, database.Err(errScan)
		}

		idx, ok := bySteam[steamID]
		if !ok {
			users = append(users, RoleUser{SteamID: steamID, PersonaName: personaName, Roles: []Role{role}})
			bySteam[steamID] = len(users) - 1
		} else {
			users[idx].Roles = append(users[idx].Roles, role)
		}
	}

	return users, nil
}

// SetUserRoles replaces the user's full set of role assignments in a single
// transaction: all existing assignments are removed, then the provided roles
// are inserted. The caller is responsible for ensuring the person exists first.
func (r Repository) SetUserRoles(ctx context.Context, steamID steamid.SteamID, roleIDs []int32) error {
	return database.Err(r.WrapTx(ctx, func(transaction pgx.Tx) error {
		deleteQuery, deleteArgs, errDelete := r.Builder().
			Delete("role_assignments").
			Where(sq.Eq{"steam_id": steamID.Int64()}).
			ToSql()
		if errDelete != nil {
			return database.Err(errDelete)
		}

		if _, errExec := transaction.Exec(ctx, deleteQuery, deleteArgs...); errExec != nil {
			return database.Err(errExec)
		}

		if len(roleIDs) == 0 {
			return nil
		}

		insert := r.Builder().
			Insert("role_assignments").
			Columns("steam_id", "role_id", "created_on").
			Suffix("ON CONFLICT (steam_id, role_id) DO NOTHING")

		now := time.Now()
		for _, roleID := range roleIDs {
			insert = insert.Values(steamID.Int64(), roleID, now)
		}

		insertQuery, insertArgs, errInsert := insert.ToSql()
		if errInsert != nil {
			return database.Err(errInsert)
		}

		_, errExec := transaction.Exec(ctx, insertQuery, insertArgs...)

		return database.Err(errExec)
	}))
}
