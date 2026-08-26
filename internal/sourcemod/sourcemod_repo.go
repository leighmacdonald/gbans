package sourcemod

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"strconv"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/ban/bantype"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/roles"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type Repository struct {
	database.Database
}

func NewRepository(database database.Database) Repository {
	return Repository{Database: database}
}

// QueryBanState is used for querying a players current active ban status.
// FIXME check_ban no longer functional.
func (r Repository) QueryBanState(ctx context.Context, steamID steamid.SteamID, ipAddr netip.Addr) (PlayerBanState, error) {
	const query = `
		SELECT b.out_ban_source, b.out_ban_id, b.out_ban_type, b.out_reason, b.out_evade_ok, b.out_valid_until, p.steam_id
		FROM check_ban($1, $2::text) b
		LEFT JOIN ban sb ON sb.ban_id = b.out_ban_id
		LEFT JOIN person p on p.steam_id = sb.target_id`

	var banState PlayerBanState

	// If there is no matches, a row of NULL values are returned from the stored proc
	var (
		banSource  *BanSource
		banID      *int
		banType    *bantype.Type
		banReason  *reason.Reason
		evadeOK    *bool
		validUntil *time.Time
		banSteamID *int64
	)

	row := r.QueryRow(ctx, query, steamID.String(), ipAddr.String())
	if errScan := row.Scan(&banSource, &banID, &banType, &banReason, &evadeOK, &validUntil, &banSteamID); errScan != nil {
		return banState, errors.Join(errScan, database.ErrScanResult)
	}

	if banSource != nil && banType != nil && banReason != nil && validUntil != nil && banID != nil && evadeOK != nil {
		banState.BanSource = *banSource
		banState.BanID = *banID
		banState.BanType = *banType
		banState.Reason = *banReason
		banState.EvadeOK = *evadeOK
		banState.ValidUntil = *validUntil

		// TODO ensure the person record exists, this will panic otherwise.
		if banSteamID != nil {
			banState.SteamID = steamid.New(*banSteamID)
		}
	}

	return banState, nil
}

type roleRow struct {
	RoleID    int32
	RoleName  string
	CreatedOn time.Time
	UpdatedOn time.Time
}

func (r Repository) smRoles(ctx context.Context) ([]roleRow, error) {
	rows, errRows := r.QueryBuilder(ctx, r.Builder().
		Select("role_id", "role_name", "created_on", "updated_on").
		From("roles").
		Where(sq.Like{"role_name": smRolePrefix + "%"}).
		OrderBy("role_id"))
	if errRows != nil {
		return nil, database.Err(errRows)
	}

	var roleRows []roleRow

	for rows.Next() {
		var row roleRow
		if errScan := rows.Scan(&row.RoleID, &row.RoleName, &row.CreatedOn, &row.UpdatedOn); errScan != nil {
			return nil, database.Err(errScan)
		}

		roleRows = append(roleRows, row)
	}

	return roleRows, nil
}

func (r Repository) smRoleByID(ctx context.Context, roleID int32) (roleRow, error) {
	var row roleRow

	dbRow, errRow := r.QueryRowBuilder(ctx, r.Builder().
		Select("role_id", "role_name", "created_on", "updated_on").
		From("roles").
		Where(sq.And{sq.Eq{"role_id": roleID}, sq.Like{"role_name": smRolePrefix + "%"}}))
	if errRow != nil {
		return roleRow{}, database.Err(errRow)
	}

	if errScan := dbRow.Scan(&row.RoleID, &row.RoleName, &row.CreatedOn, &row.UpdatedOn); errScan != nil {
		return roleRow{}, database.Err(errScan)
	}

	return row, nil
}

// roleIDByName returns the role id for the exact role name, or false when no
// such role exists.
func (r Repository) roleIDByName(ctx context.Context, roleName string) (int32, bool, error) {
	var roleID int32

	dbRow, errRow := r.QueryRowBuilder(ctx, r.Builder().
		Select("role_id").
		From("roles").
		Where(sq.Eq{"role_name": roleName}))
	if errRow != nil {
		return 0, false, database.Err(errRow)
	}

	if errScan := dbRow.Scan(&roleID); errScan != nil {
		if errors.Is(errScan, pgx.ErrNoRows) {
			return 0, false, nil
		}

		return 0, false, database.Err(errScan)
	}

	return roleID, true, nil
}

func (r Repository) rolePermissions(ctx context.Context, roleIDs []int32) (map[int32][]rolesv1.Permission, error) {
	perms := make(map[int32][]rolesv1.Permission, len(roleIDs))
	if len(roleIDs) == 0 {
		return perms, nil
	}

	rows, errRows := r.QueryBuilder(ctx, r.Builder().
		Select("role_id", "permission").
		From("role_permissions").
		Where(sq.Eq{"role_id": roleIDs}))
	if errRows != nil {
		return nil, database.Err(errRows)
	}

	for rows.Next() {
		var (
			roleID int32
			perm   rolesv1.Permission
		)
		if errScan := rows.Scan(&roleID, &perm); errScan != nil {
			return nil, database.Err(errScan)
		}

		perms[roleID] = append(perms[roleID], perm)
	}

	return perms, nil
}

// setRolePermissions replaces the permission set of the given role.
func (r Repository) setRolePermissions(ctx context.Context, roleID int32, perms []rolesv1.Permission) error {
	if err := r.ExecDeleteBuilder(ctx, r.Builder().
		Delete("role_permissions").
		Where(sq.Eq{"role_id": roleID})); err != nil {
		return database.Err(err)
	}

	if len(perms) == 0 {
		return nil
	}

	now := time.Now()
	builder := r.Builder().
		Insert("role_permissions").
		Columns("role_id", "permission", "created_on", "updated_on")

	for _, perm := range perms {
		builder = builder.Values(roleID, perm, now, now)
	}

	if err := r.ExecInsertBuilder(ctx, builder); err != nil {
		return database.Err(err)
	}

	return nil
}

// roleMemberCounts returns the number of assignments for every sourcemod role.
func (r Repository) roleMemberCounts(ctx context.Context) (map[int32]int, error) {
	members := make(map[int32]int)

	rows, errRows := r.QueryBuilder(ctx, r.Builder().
		Select("r.role_id", "count(ra.steam_id)").
		From("roles r").
		LeftJoin("role_assignments ra USING (role_id)").
		Where(sq.Like{"r.role_name": smRolePrefix + "%"}).
		GroupBy("r.role_id"))
	if errRows != nil {
		return nil, database.Err(errRows)
	}

	for rows.Next() {
		var (
			roleID int32
			count  int
		)
		if errScan := rows.Scan(&roleID, &count); errScan != nil {
			return nil, database.Err(errScan)
		}

		members[roleID] = count
	}

	return members, nil
}

// smRoleIDs returns the ids of every sourcemod role.
func (r Repository) smRoleIDs(ctx context.Context) ([]int32, error) {
	roleRows, errRows := r.smRoles(ctx)
	if errRows != nil {
		return nil, errRows
	}

	roleIDs := make([]int32, 0, len(roleRows))
	for _, row := range roleRows {
		roleIDs = append(roleIDs, row.RoleID)
	}

	return roleIDs, nil
}

// roleAssignments maps steam ids to the given sourcemod roles they are assigned to.
func (r Repository) roleAssignments(ctx context.Context, steamIDs []int64, roleIDs []int32) (map[int64][]int32, error) {
	assignments := make(map[int64][]int32, len(steamIDs))
	if len(steamIDs) == 0 || len(roleIDs) == 0 {
		return assignments, nil
	}

	rows, errRows := r.QueryBuilder(ctx, r.Builder().
		Select("steam_id", "role_id").
		From("role_assignments").
		Where(sq.And{
			sq.Eq{"steam_id": steamIDs},
			sq.Eq{"role_id": roleIDs},
		}).
		OrderBy("steam_id", "role_id"))
	if errRows != nil {
		return nil, database.Err(errRows)
	}

	for rows.Next() {
		var (
			steamID int64
			roleID  int32
		)
		if errScan := rows.Scan(&steamID, &roleID); errScan != nil {
			return nil, database.Err(errScan)
		}

		assignments[steamID] = append(assignments[steamID], roleID)
	}

	return assignments, nil
}

func (r Repository) assignRole(ctx context.Context, steamID int64, roleID int32) error {
	if err := r.ExecInsertBuilder(ctx, r.Builder().
		Insert("role_assignments").
		Columns("steam_id", "role_id", "created_on").
		Values(steamID, roleID, time.Now()).
		Suffix("ON CONFLICT DO NOTHING")); err != nil {
		return database.Err(err)
	}

	return nil
}

func (r Repository) unassignRole(ctx context.Context, steamID int64, roleID int32) error {
	return database.Err(r.ExecDeleteBuilder(ctx, r.Builder().
		Delete("role_assignments").
		Where(sq.And{sq.Eq{"steam_id": steamID}, sq.Eq{"role_id": roleID}})))
}

// moveAdminRoles moves every sourcemod role assignment from one steam id to
// another, used when an admin's identity changes.
func (r Repository) moveAdminRoles(ctx context.Context, fromSteamID, toSteamID int64) error {
	roleIDs, errRoleIDs := r.smRoleIDs(ctx)
	if errRoleIDs != nil {
		return errRoleIDs
	}
	if len(roleIDs) == 0 {
		return nil
	}

	if err := r.ExecUpdateBuilder(ctx, r.Builder().
		Update("role_assignments").
		Set("steam_id", toSteamID).
		Where(sq.And{
			sq.Eq{"steam_id": fromSteamID},
			sq.Eq{"role_id": roleIDs},
		})); err != nil {
		return database.Err(err)
	}

	return nil
}

// setPermissionLevel updates the legacy person.permission_level column, kept
// in sync with role assignments for reporting only.
func (r Repository) setPermissionLevel(ctx context.Context, steamID int64, level int32) error {
	if err := r.ExecUpdateBuilder(ctx, r.Builder().
		Update("person").
		Set("permission_level", level).
		Where(sq.Eq{"steam_id": steamID})); err != nil {
		return database.Err(err)
	}

	return nil
}

// refreshPermissionLevel recomputes the legacy permission level from all of
// the steam id's role assignments.
func (r Repository) refreshPermissionLevel(ctx context.Context, steamID int64) error {
	rows, errRows := r.QueryBuilder(ctx, r.Builder().
		Select("r.role_name").
		From("role_assignments ra").
		Join("roles r USING (role_id)").
		Where(sq.Eq{"ra.steam_id": steamID}))
	if errRows != nil {
		return database.Err(errRows)
	}

	var userRoles []roles.Role

	for rows.Next() {
		var roleName string
		if errScan := rows.Scan(&roleName); errScan != nil {
			return database.Err(errScan)
		}

		userRoles = append(userRoles, roles.Role{RoleName: roleName}) //nolint:exhaustruct_v5
	}

	return r.setPermissionLevel(ctx, steamID, roles.EffectivePrivilegeLevel(userRoles))
}

func toGroupFromRole(row roleRow, perms []rolesv1.Permission) Groups {
	return Groups{
		GroupID:       row.RoleID,
		Flags:         permissionsToFlags(perms),
		Name:          nameFromSMRole(row.RoleName),
		ImmunityLevel: deriveImmunity(perms),
		CreatedOn:     row.CreatedOn,
		UpdatedOn:     row.UpdatedOn,
	}
}

// groupsByIDs loads the given sourcemod role ids as groups.
func (r Repository) groupsByIDs(ctx context.Context, roleIDs []int32) (map[int32]Groups, error) {
	groups := make(map[int32]Groups, len(roleIDs))
	if len(roleIDs) == 0 {
		return groups, nil
	}

	perms, errPerms := r.rolePermissions(ctx, roleIDs)
	if errPerms != nil {
		return nil, errPerms
	}

	for _, roleID := range roleIDs {
		row, errRow := r.smRoleByID(ctx, roleID)
		if errRow != nil {
			return nil, errRow
		}

		groups[roleID] = toGroupFromRole(row, perms[roleID])
	}

	return groups, nil
}

func (r Repository) Groups(ctx context.Context) ([]Groups, error) {
	roleRows, errRoles := r.smRoles(ctx)
	if errRoles != nil {
		return nil, errRoles
	}

	var roleIDs []int32
	for _, row := range roleRows {
		roleIDs = append(roleIDs, row.RoleID)
	}

	perms, errPerms := r.rolePermissions(ctx, roleIDs)
	if errPerms != nil {
		return nil, errPerms
	}

	groups := make([]Groups, 0, len(roleRows))
	for _, row := range roleRows {
		groups = append(groups, toGroupFromRole(row, perms[row.RoleID]))
	}

	return groups, nil
}

func (r Repository) GetGroupByID(ctx context.Context, groupID int32) (Groups, error) {
	row, errRow := r.smRoleByID(ctx, groupID)
	if errRow != nil {
		return Groups{}, errRow
	}

	perms, errPerms := r.rolePermissions(ctx, []int32{groupID})
	if errPerms != nil {
		return Groups{}, errPerms
	}

	return toGroupFromRole(row, perms[groupID]), nil
}

func (r Repository) GetGroupByName(ctx context.Context, groupName string) (Groups, error) {
	roleID, exists, errExists := r.roleIDByName(ctx, smRoleName(groupName))
	if errExists != nil {
		return Groups{}, errExists
	}
	if !exists {
		return Groups{}, database.ErrNoResult
	}

	return r.GetGroupByID(ctx, roleID)
}

func (r Repository) AddGroup(ctx context.Context, group Groups) (Groups, error) {
	now := time.Now()
	group.CreatedOn = now
	group.UpdatedOn = now

	if err := r.ExecInsertBuilderWithReturnValue(ctx, r.Builder().
		Insert("roles").
		Columns("role_name").
		Values(smRoleName(group.Name)).
		Suffix("RETURNING role_id"), &group.GroupID); err != nil {
		return Groups{}, database.Err(err)
	}

	if err := r.setRolePermissions(ctx, group.GroupID, groupPermissions(group.Flags, group.ImmunityLevel)); err != nil {
		return Groups{}, err
	}

	slog.Info("Created SM Group", slog.Int("group_id", int(group.GroupID)), slog.String("name", group.Name))

	return group, nil
}

func (r Repository) SaveGroup(ctx context.Context, group Groups) (Groups, error) {
	group.UpdatedOn = time.Now()

	if err := r.ExecUpdateBuilder(ctx, r.Builder().
		Update("roles").
		Set("role_name", smRoleName(group.Name)).
		Set("updated_on", group.UpdatedOn).
		Where(sq.And{sq.Eq{"role_id": group.GroupID}, sq.Like{"role_name": smRolePrefix + "%"}})); err != nil {
		return Groups{}, database.Err(err)
	}

	if err := r.setRolePermissions(ctx, group.GroupID, groupPermissions(group.Flags, group.ImmunityLevel)); err != nil {
		return Groups{}, err
	}

	return group, nil
}

func (r Repository) DeleteGroup(ctx context.Context, group Groups) error {
	if err := r.ExecDeleteBuilder(ctx, r.Builder().
		Delete("roles").
		Where(sq.And{sq.Eq{"role_id": group.GroupID}, sq.Like{"role_name": smRolePrefix + "%"}})); err != nil {
		return database.Err(err)
	}

	slog.Info("Deleted SM Group", slog.Int("group_id", int(group.GroupID)), slog.String("name", group.Name))

	return nil
}

type adminBase struct {
	SteamID    int64
	PersonName *string
	CreatedOn  time.Time
	UpdatedOn  time.Time
}

// adminBases loads the identity of every steam id holding a sourcemod role.
func (r Repository) adminBases(ctx context.Context, steamIDs []int64) ([]adminBase, error) {
	builder := r.Builder().
		Select("ra.steam_id", "p.personaname", "min(ra.created_on)", "max(r.updated_on)").
		From("role_assignments ra").
		Join("roles r USING (role_id)").
		Join("person p ON p.steam_id = ra.steam_id").
		Where(sq.Like{"r.role_name": smRolePrefix + "%"}).
		GroupBy("ra.steam_id", "p.personaname").
		OrderBy("ra.steam_id")

	if len(steamIDs) > 0 {
		builder = builder.Where(sq.Eq{"ra.steam_id": steamIDs})
	}

	rows, errRows := r.QueryBuilder(ctx, builder)
	if errRows != nil {
		return nil, database.Err(errRows)
	}

	var bases []adminBase

	for rows.Next() {
		var base adminBase
		if errScan := rows.Scan(&base.SteamID, &base.PersonName, &base.CreatedOn, &base.UpdatedOn); errScan != nil {
			return nil, database.Err(errScan)
		}

		bases = append(bases, base)
	}

	return bases, nil
}

// buildAdmins assembles the full admin records for the given steam ids (all of
// them when the slice is empty). The admin identity, flags and immunity are
// derived from the admin's personal role; the display name falls back to the
// person name, then the steam id.
func (r Repository) buildAdmins(ctx context.Context, steamIDs []int64) ([]Admin, error) {
	bases, errBases := r.adminBases(ctx, steamIDs)
	if errBases != nil {
		return nil, errBases
	}

	if len(bases) == 0 {
		return []Admin{}, nil
	}

	steamIDList := make([]int64, 0, len(bases))
	for _, base := range bases {
		steamIDList = append(steamIDList, base.SteamID)
	}

	roleRows, errRoles := r.smRoles(ctx)
	if errRoles != nil {
		return nil, errRoles
	}

	roleMap := make(map[int32]roleRow, len(roleRows))
	roleIDs := make([]int32, 0, len(roleRows))
	for _, row := range roleRows {
		roleMap[row.RoleID] = row
		roleIDs = append(roleIDs, row.RoleID)
	}

	perms, errPerms := r.rolePermissions(ctx, roleIDs)
	if errPerms != nil {
		return nil, errPerms
	}

	members, errMembers := r.roleMemberCounts(ctx)
	if errMembers != nil {
		return nil, errMembers
	}

	assignments, errAssignments := r.roleAssignments(ctx, steamIDList, roleIDs)
	if errAssignments != nil {
		return nil, errAssignments
	}

	admins := make([]Admin, 0, len(bases))
	for _, base := range bases {
		sid := steamid.New(base.SteamID)
		admin := Admin{
			AdminID:   base.SteamID,
			SteamID:   sid,
			AuthType:  AuthTypeSteam,
			Identity:  string(sid.Steam3()),
			Password:  "",
			Flags:     "",
			Name:      "",
			Immunity:  0,
			CreatedOn: base.CreatedOn,
			UpdatedOn: base.UpdatedOn,
			Groups:    []Groups{},
		}

		// The personal role is the only sourcemod role assigned to this single
		// steam id; it carries the admin's direct flags and immunity.
		var personal *roleRow
		for _, roleID := range assignments[base.SteamID] {
			row, ok := roleMap[roleID]
			if !ok {
				continue
			}

			admin.Groups = append(admin.Groups, toGroupFromRole(row, perms[roleID]))
			if members[roleID] == 1 {
				captured := row
				personal = &captured
			}
		}

		if personal != nil {
			admin.Name = nameFromSMRole(personal.RoleName)
			admin.Flags = permissionsToFlags(perms[personal.RoleID])
			admin.Immunity = deriveImmunity(perms[personal.RoleID])
		}

		if admin.Name == "" {
			if base.PersonName != nil && *base.PersonName != "" {
				admin.Name = *base.PersonName
			} else {
				admin.Name = admin.SteamID.String()
			}
		}

		admins = append(admins, admin)
	}

	return admins, nil
}

func (r Repository) Admins(ctx context.Context) ([]Admin, error) {
	return r.buildAdmins(ctx, nil)
}

func (r Repository) GetAdminByID(ctx context.Context, adminID int64) (Admin, error) {
	admins, errAdmins := r.buildAdmins(ctx, []int64{adminID})
	if errAdmins != nil {
		return Admin{}, errAdmins
	}

	if len(admins) == 0 {
		return Admin{}, database.ErrNoResult
	}

	return admins[0], nil
}

// createPersonalRole creates, or adopts, a role dedicated to a single admin.
// The role is named after the admin's alias when that name is free; when it is
// already taken by a role with members the role falls back to the admin's
// 64-bit steam id.
func (r Repository) createPersonalRole(ctx context.Context, steamID int64, alias string) (int32, error) {
	names := make([]string, 0, 2)
	if alias != "" {
		names = append(names, smRoleName(alias))
	}
	names = append(names, smRoleName(strconv.FormatInt(steamID, 10)))

	for idx, name := range names {
		existingID, exists, errExists := r.roleIDByName(ctx, name)
		if errExists != nil {
			return 0, errExists
		}

		if exists {
			count, errCount := r.roleMemberCounts(ctx)
			if errCount != nil {
				return 0, errCount
			}

			if count[existingID] == 0 {
				return existingID, nil
			}

			if idx == len(names)-1 {
				return 0, ErrAdminNameExists
			}

			continue
		}

		var roleID int32
		if err := r.ExecInsertBuilderWithReturnValue(ctx, r.Builder().
			Insert("roles").
			Columns("role_name").
			Values(name).
			Suffix("RETURNING role_id"), &roleID); err != nil {
			return 0, database.Err(err)
		}

		return roleID, nil
	}

	return 0, ErrAdminNameExists
}

// renamePersonalRole renames the admin's personal role to the given alias, or
// to the steam id when the alias is empty.
func (r Repository) renamePersonalRole(ctx context.Context, roleID int32, steamID int64, alias string) error {
	desired := smRoleName(strconv.FormatInt(steamID, 10))
	if alias != "" {
		desired = smRoleName(alias)
	}

	var current string
	dbRow, errRow := r.QueryRowBuilder(ctx, r.Builder().
		Select("role_name").
		From("roles").
		Where(sq.Eq{"role_id": roleID}))
	if errRow != nil {
		return database.Err(errRow)
	}

	if errScan := dbRow.Scan(&current); errScan != nil {
		return database.Err(errScan)
	}

	if current == desired {
		return nil
	}

	_, exists, errExists := r.roleIDByName(ctx, desired)
	if errExists != nil {
		return errExists
	}
	if exists {
		return ErrAdminNameExists
	}

	if err := r.ExecUpdateBuilder(ctx, r.Builder().
		Update("roles").
		Set("role_name", desired).
		Set("updated_on", time.Now()).
		Where(sq.Eq{"role_id": roleID})); err != nil {
		return database.Err(err)
	}

	return nil
}

// upsertAdmin writes the admin's personal role (name, flags, immunity) and
// refreshes the legacy permission level. Existing group assignments are
// preserved; when the admin's identity changed, every sourcemod assignment is
// moved to the new steam id first.
func (r Repository) upsertAdmin(ctx context.Context, admin Admin) (Admin, error) {
	target := admin.SteamID.Int64()

	if admin.AdminID != 0 && admin.AdminID != target {
		if err := r.moveAdminRoles(ctx, admin.AdminID, target); err != nil {
			return Admin{}, err
		}

		if err := r.refreshPermissionLevel(ctx, admin.AdminID); err != nil {
			return Admin{}, err
		}
	}

	members, errMembers := r.roleMemberCounts(ctx)
	if errMembers != nil {
		return Admin{}, errMembers
	}

	roleIDs, errRoleIDs := r.smRoleIDs(ctx)
	if errRoleIDs != nil {
		return Admin{}, errRoleIDs
	}

	assignments, errAssignments := r.roleAssignments(ctx, []int64{target}, roleIDs)
	if errAssignments != nil {
		return Admin{}, errAssignments
	}

	var personalID int32
	for _, roleID := range assignments[target] {
		if members[roleID] == 1 {
			personalID = roleID

			break
		}
	}

	if personalID == 0 {
		roleID, errRole := r.createPersonalRole(ctx, target, admin.Name)
		if errRole != nil {
			return Admin{}, errRole
		}

		if errAssign := r.assignRole(ctx, target, roleID); errAssign != nil {
			return Admin{}, errAssign
		}

		personalID = roleID
	}

	if err := r.renamePersonalRole(ctx, personalID, target, admin.Name); err != nil {
		return Admin{}, err
	}

	if err := r.setRolePermissions(ctx, personalID, groupPermissions(admin.Flags, admin.Immunity)); err != nil {
		return Admin{}, err
	}

	if err := r.refreshPermissionLevel(ctx, target); err != nil {
		return Admin{}, err
	}

	slog.Info("Saved SM Admin", slog.String("steam_id", admin.SteamID.String()), slog.String("name", admin.Name))

	return r.GetAdminByID(ctx, target)
}

func (r Repository) AddAdmin(ctx context.Context, admin Admin) (Admin, error) {
	return r.upsertAdmin(ctx, admin)
}

func (r Repository) SaveAdmin(ctx context.Context, admin Admin) (Admin, error) {
	return r.upsertAdmin(ctx, admin)
}

func (r Repository) DelAdmin(ctx context.Context, admin Admin) error {
	steamID := admin.SteamID.Int64()

	members, errMembers := r.roleMemberCounts(ctx)
	if errMembers != nil {
		return errMembers
	}

	roleIDs, errRoleIDs := r.smRoleIDs(ctx)
	if errRoleIDs != nil {
		return errRoleIDs
	}

	assignments, errAssignments := r.roleAssignments(ctx, []int64{steamID}, roleIDs)
	if errAssignments != nil {
		return errAssignments
	}

	for _, roleID := range assignments[steamID] {
		if errUnassign := r.unassignRole(ctx, steamID, roleID); errUnassign != nil {
			return errUnassign
		}

		// Roles that were personal to this admin (assigned to exactly one steam
		// id) are removed entirely rather than left orphaned.
		if members[roleID] == 1 {
			if errDelete := r.ExecDeleteBuilder(ctx, r.Builder().
				Delete("roles").
				Where(sq.Eq{"role_id": roleID})); errDelete != nil {
				return database.Err(errDelete)
			}
		}
	}

	if err := r.refreshPermissionLevel(ctx, steamID); err != nil {
		return err
	}

	slog.Info("Deleted SM Admin", slog.String("steam_id", admin.SteamID.String()))

	return nil
}

func (r Repository) InsertAdminGroup(ctx context.Context, admin Admin, group Groups) error {
	return r.assignRole(ctx, admin.SteamID.Int64(), group.GroupID)
}

func (r Repository) DeleteAdminGroup(ctx context.Context, admin Admin, group Groups) error {
	return r.unassignRole(ctx, admin.SteamID.Int64(), group.GroupID)
}

func (r Repository) GetGroupImmunities(ctx context.Context) ([]GroupImmunity, error) {
	return r.groupImmunities(ctx, nil)
}

func (r Repository) GetGroupImmunityByID(ctx context.Context, groupImmunityID int32) (GroupImmunity, error) {
	immunities, errImmunities := r.groupImmunities(ctx, &groupImmunityID)
	if errImmunities != nil {
		return GroupImmunity{}, errImmunities
	}

	if len(immunities) == 0 {
		return GroupImmunity{}, database.ErrNoResult
	}

	return immunities[0], nil
}

func (r Repository) groupImmunities(ctx context.Context, immunityID *int32) ([]GroupImmunity, error) {
	builder := r.Builder().
		Select("ri.role_immunity_id", "ri.created_on", "ri.role_id", "ri.other_id").
		From("role_immunity ri").
		OrderBy("ri.role_immunity_id")
	if immunityID != nil {
		builder = builder.Where(sq.Eq{"ri.role_immunity_id": *immunityID})
	}

	rows, errRows := r.QueryBuilder(ctx, builder)
	if errRows != nil {
		return nil, database.Err(errRows)
	}

	type immunityRow struct {
		ID        int32
		CreatedOn time.Time
		RoleID    int32
		OtherID   int32
	}

	var rowsData []immunityRow
	ids := make(map[int32]struct{})

	for rows.Next() {
		var row immunityRow
		if errScan := rows.Scan(&row.ID, &row.CreatedOn, &row.RoleID, &row.OtherID); errScan != nil {
			return nil, database.Err(errScan)
		}

		rowsData = append(rowsData, row)
		ids[row.RoleID] = struct{}{}
		ids[row.OtherID] = struct{}{}
	}

	var roleIDs []int32
	for id := range ids {
		roleIDs = append(roleIDs, id)
	}

	groups, errGroups := r.groupsByIDs(ctx, roleIDs)
	if errGroups != nil {
		return nil, errGroups
	}

	immunities := make([]GroupImmunity, 0, len(rowsData))
	for _, row := range rowsData {
		immunities = append(immunities, GroupImmunity{
			GroupImmunityID: row.ID,
			Group:           groups[row.RoleID],
			Other:           groups[row.OtherID],
			CreatedOn:       row.CreatedOn,
		})
	}

	return immunities, nil
}

func (r Repository) AddGroupImmunity(ctx context.Context, group Groups, other Groups) (GroupImmunity, error) {
	immunity := GroupImmunity{
		GroupImmunityID: 0,
		Group:           group,
		Other:           other,
		CreatedOn:       time.Now(),
	}

	if err := r.ExecInsertBuilderWithReturnValue(ctx, r.Builder().
		Insert("role_immunity").
		Columns("role_id", "other_id", "created_on").
		Values(group.GroupID, other.GroupID, immunity.CreatedOn).
		Suffix("RETURNING role_immunity_id"), &immunity.GroupImmunityID); err != nil {
		return GroupImmunity{}, database.Err(err)
	}

	return immunity, nil
}

func (r Repository) DelGroupImmunity(ctx context.Context, groupImmunity GroupImmunity) error {
	return database.Err(r.ExecDeleteBuilder(ctx, r.Builder().
		Delete("role_immunity").
		Where(sq.Eq{"role_immunity_id": groupImmunity.GroupImmunityID})))
}

func (r Repository) AddGroupOverride(ctx context.Context, override GroupOverrides) (GroupOverrides, error) {
	if err := r.ExecInsertBuilderWithReturnValue(ctx, r.Builder().
		Insert("role_overrides").
		SetMap(map[string]any{
			"role_id":    override.GroupID,
			"type":       override.Type,
			"name":       override.Name,
			"access":     override.Access,
			"created_on": override.CreatedOn,
			"updated_on": override.UpdatedOn,
		}).
		Suffix("RETURNING override_id"), &override.GroupOverrideID); err != nil {
		return override, database.Err(err)
	}

	return override, nil
}

func (r Repository) DelGroupOverride(ctx context.Context, override GroupOverrides) error {
	return database.Err(r.ExecDeleteBuilder(ctx, r.Builder().
		Delete("role_overrides").
		Where(sq.Eq{"override_id": override.GroupOverrideID})))
}

func (r Repository) GetGroupOverride(ctx context.Context, overrideID int32) (GroupOverrides, error) {
	var override GroupOverrides

	row, errRow := r.QueryRowBuilder(ctx, r.Builder().
		Select("override_id", "role_id", "type", "name", "access", "created_on", "updated_on").
		From("role_overrides").
		Where(sq.Eq{"override_id": overrideID}))
	if errRow != nil {
		return GroupOverrides{}, database.Err(errRow)
	}

	if errScan := row.Scan(&override.GroupOverrideID, &override.GroupID, &override.Type, &override.Name,
		&override.Access, &override.CreatedOn, &override.UpdatedOn); errScan != nil {
		return override, database.Err(errScan)
	}

	return override, nil
}

func (r Repository) SaveGroupOverride(ctx context.Context, override GroupOverrides) (GroupOverrides, error) {
	override.UpdatedOn = time.Now()

	if err := r.ExecUpdateBuilder(ctx, r.Builder().
		Update("role_overrides").
		SetMap(map[string]any{
			"role_id":    override.GroupID,
			"type":       override.Type,
			"name":       override.Name,
			"access":     override.Access,
			"updated_on": override.UpdatedOn,
		}).
		Where(sq.Eq{"override_id": override.GroupOverrideID})); err != nil {
		return GroupOverrides{}, database.Err(err)
	}

	return override, nil
}

func (r Repository) GroupOverrides(ctx context.Context, group Groups) ([]GroupOverrides, error) {
	rows, errRows := r.QueryBuilder(ctx, r.Builder().
		Select("override_id", "role_id", "type", "name", "access", "created_on", "updated_on").
		From("role_overrides").
		Where(sq.Eq{"role_id": group.GroupID}))
	if errRows != nil {
		return nil, database.Err(errRows)
	}

	var overrides []GroupOverrides

	for rows.Next() {
		var override GroupOverrides
		if errScan := rows.Scan(&override.GroupOverrideID, &override.GroupID, &override.Type, &override.Name, &override.Access,
			&override.CreatedOn, &override.UpdatedOn); errScan != nil {
			return nil, database.Err(errScan)
		}

		overrides = append(overrides, override)
	}

	return overrides, nil
}

func (r Repository) GetOverride(ctx context.Context, overrideID int32) (Overrides, error) {
	var override Overrides

	row, errRow := r.QueryRowBuilder(ctx, r.Builder().
		Select("override_id", "type", "name", "flags", "created_on", "updated_on").
		From("command_overrides").
		Where(sq.Eq{"override_id": overrideID}))
	if errRow != nil {
		return override, database.Err(errRow)
	}

	if errScan := row.Scan(&override.OverrideID, &override.Type, &override.Name,
		&override.Flags, &override.CreatedOn, &override.UpdatedOn); errScan != nil {
		return override, database.Err(errScan)
	}

	return override, nil
}

func (r Repository) AddOverride(ctx context.Context, overrides Overrides) (Overrides, error) {
	if err := r.ExecInsertBuilderWithReturnValue(ctx, r.Builder().
		Insert("command_overrides").
		SetMap(map[string]any{
			"type":       overrides.Type,
			"name":       overrides.Name,
			"flags":      overrides.Flags,
			"created_on": overrides.CreatedOn,
			"updated_on": overrides.UpdatedOn,
		}).
		Suffix("RETURNING override_id"), &overrides.OverrideID); err != nil {
		return overrides, database.Err(err)
	}

	return overrides, nil
}

func (r Repository) DelOverride(ctx context.Context, override Overrides) error {
	return database.Err(r.ExecDeleteBuilder(ctx, r.Builder().
		Delete("command_overrides").
		Where(sq.Eq{"override_id": override.OverrideID}),
	))
}

func (r Repository) SaveOverride(ctx context.Context, override Overrides) (Overrides, error) {
	override.UpdatedOn = time.Now()

	if err := r.ExecUpdateBuilder(ctx, r.Builder().
		Update("command_overrides").
		SetMap(map[string]any{
			"type":       override.Type,
			"name":       override.Name,
			"flags":      override.Flags,
			"updated_on": override.UpdatedOn,
		}).
		Where(sq.Eq{"override_id": override.OverrideID})); err != nil {
		return Overrides{}, database.Err(err)
	}

	return override, nil
}

func (r Repository) Overrides(ctx context.Context) ([]Overrides, error) {
	rows, errRows := r.QueryBuilder(ctx, r.Builder().
		Select("override_id", "type", "name", "flags", "created_on", "updated_on").
		From("command_overrides"))
	if errRows != nil {
		return nil, database.Err(errRows)
	}

	var overrides []Overrides

	for rows.Next() {
		var override Overrides
		if errScan := rows.Scan(&override.OverrideID, &override.Type, &override.Name, &override.Flags,
			&override.CreatedOn, &override.UpdatedOn); errScan != nil {
			return nil, database.Err(errScan)
		}

		overrides = append(overrides, override)
	}

	return overrides, nil
}
