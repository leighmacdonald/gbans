package sourcemod

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type Repository struct {
	database database.Database
}

func NewRepository(database database.Database) Repository {
	return Repository{database: database}
}

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
		banType    *ban.Type
		reason     *ban.Reason
		evadeOK    *bool
		validUntil *time.Time
		banSteamID *int64
	)

	row := r.database.QueryRow(ctx, query, steamID.String(), ipAddr.String())
	if errScan := row.Scan(&banSource, &banID, &banType, &reason, &evadeOK, &validUntil, &banSteamID); errScan != nil {
		return banState, errors.Join(errScan, database.ErrScanResult)
	}

	if banSource != nil {
		banState.BanSource = *banSource
		banState.BanID = *banID
		banState.BanType = *banType
		banState.Reason = *reason
		banState.EvadeOK = *evadeOK
		banState.ValidUntil = *validUntil

		// TODO ensure the person record exists, this will panic otherwise.
		if banSteamID != nil {
			banState.SteamID = steamid.New(*banSteamID)
		}
	}

	return banState, nil
}

func (r Repository) GetGroupImmunities(ctx context.Context) ([]GroupImmunity, error) {
	var immunities []GroupImmunity

	rows, errRows := r.database.QueryBuilder(ctx, r.database.Builder().
		Select("gi.group_immunity_id", "gi.created_on",
			"g.id", "g.flags", "g.name", "g.immunity_level", "g.created_on", "g.updated_on",
			"o.id", "o.flags", "o.name", "o.immunity_level", "o.created_on", "o.updated_on").
		From("sm_group_immunity gi").
		LeftJoin("sm_groups g ON g.id = gi.group_id").
		LeftJoin("sm_groups o ON o.id = gi.other_id"))
	if errRows != nil {
		return nil, database.DBErr(errRows)
	}

	for rows.Next() {
		var immunity GroupImmunity

		if errScan := rows.Scan(&immunity.GroupImmunityID, &immunity.CreatedOn,
			&immunity.Group.GroupID, &immunity.Group.Flags, &immunity.Group.Name, &immunity.Group.ImmunityLevel,
			&immunity.Group.CreatedOn, &immunity.Group.UpdatedOn,
			&immunity.Other.GroupID, &immunity.Other.Flags, &immunity.Other.Name, &immunity.Other.ImmunityLevel,
			&immunity.Other.CreatedOn, &immunity.Other.UpdatedOn); errScan != nil {
			return nil, database.DBErr(errScan)
		}

		immunities = append(immunities, immunity)
	}

	return immunities, nil
}

func (r Repository) GetGroupImmunityByID(ctx context.Context, groupImmunityID int) (GroupImmunity, error) {
	var immunity GroupImmunity

	row, errRow := r.database.QueryRowBuilder(ctx, r.database.Builder().
		Select("gi.group_immunity_id", "gi.created_on",
			"g.id", "g.flags", "g.name", "g.immunity_level", "g.created_on", "g.updated_on",
			"o.id", "o.flags", "o.name", "o.immunity_level", "o.created_on", "o.updated_on").
		From("sm_group_immunity gi").
		LeftJoin("sm_groups g ON g.id = gi.group_id").
		LeftJoin("sm_groups o ON o.id = gi.other_id").
		Where(sq.Eq{"gi.group_immunity_id": groupImmunityID}))
	if errRow != nil {
		return GroupImmunity{}, database.DBErr(errRow)
	}

	if errScan := row.Scan(&immunity.GroupImmunityID, &immunity.CreatedOn,
		&immunity.Group.GroupID, &immunity.Group.Flags, &immunity.Group.Name, &immunity.Group.ImmunityLevel,
		&immunity.Group.CreatedOn, &immunity.Group.UpdatedOn,
		&immunity.Other.GroupID, &immunity.Other.Flags, &immunity.Other.Name, &immunity.Other.ImmunityLevel,
		&immunity.Other.CreatedOn, &immunity.Other.UpdatedOn); errScan != nil {
		return GroupImmunity{}, database.DBErr(errScan)
	}

	return immunity, nil
}

func (r Repository) AddGroupImmunity(ctx context.Context, group Groups, other Groups) (GroupImmunity, error) {
	immunity := GroupImmunity{
		GroupImmunityID: 0,
		Group:           group,
		Other:           other,
		CreatedOn:       time.Now(),
	}
	if err := r.database.ExecInsertBuilderWithReturnValue(ctx, r.database.Builder().
		Insert("sm_group_immunity").
		SetMap(map[string]any{
			"group_id":   immunity.Group.GroupID,
			"other_id":   immunity.Other.GroupID,
			"created_on": immunity.CreatedOn,
		}).
		Suffix("RETURNING group_immunity_id"), &immunity.GroupImmunityID); err != nil {
		return GroupImmunity{}, database.DBErr(err)
	}

	return immunity, nil
}

func (r Repository) DelGroupImmunity(ctx context.Context, groupImmunity GroupImmunity) error {
	return database.DBErr(r.database.ExecDeleteBuilder(ctx, r.database.Builder().
		Delete("sm_group_immunity").
		Where(sq.Eq{"group_immunity_id": groupImmunity.GroupImmunityID})))
}

func (r Repository) AddGroupOverride(ctx context.Context, override GroupOverrides) (GroupOverrides, error) {
	if err := r.database.ExecInsertBuilderWithReturnValue(ctx, r.database.Builder().
		Insert("sm_group_overrides").
		SetMap(map[string]any{
			"group_id":   override.GroupID,
			"type":       override.Type,
			"name":       override.Name,
			"access":     override.Access,
			"created_on": override.CreatedOn,
			"updated_on": override.UpdatedOn,
		}).
		Suffix("RETURNING group_override_id"), &override.GroupOverrideID); err != nil {
		return override, database.DBErr(err)
	}

	return override, nil
}

func (r Repository) DelGroupOverride(ctx context.Context, override GroupOverrides) error {
	return database.DBErr(r.database.ExecDeleteBuilder(ctx, r.database.Builder().
		Delete("sm_group_overrides").
		Where(sq.Eq{"group_override_id": override.GroupOverrideID})))
}

func (r Repository) GetGroupOverride(ctx context.Context, overrideID int) (GroupOverrides, error) {
	var override GroupOverrides

	row, errRow := r.database.QueryRowBuilder(ctx, r.database.Builder().
		Select("group_override_id", "group_id", "type", "name", "access", "created_on", "updated_on").
		From("sm_group_overrides").
		Where(sq.Eq{"group_override_id": overrideID}))
	if errRow != nil {
		return GroupOverrides{}, database.DBErr(errRow)
	}

	if errScan := row.Scan(&override.GroupOverrideID, &override.GroupID, &override.Type, &override.Name,
		&override.Access, &override.CreatedOn, &override.UpdatedOn); errScan != nil {
		return override, database.DBErr(errScan)
	}

	return override, nil
}

func (r Repository) SaveGroupOverride(ctx context.Context, override GroupOverrides) (GroupOverrides, error) {
	override.UpdatedOn = time.Now()

	if err := r.database.ExecUpdateBuilder(ctx, r.database.Builder().
		Update("sm_group_overrides").
		SetMap(map[string]any{
			"group_id":   override.GroupID,
			"type":       override.Type,
			"name":       override.Name,
			"access":     override.Access,
			"updated_on": override.UpdatedOn,
		}).
		Where(sq.Eq{"group_override_id": override.GroupOverrideID})); err != nil {
		return GroupOverrides{}, database.DBErr(err)
	}

	return override, nil
}

func (r Repository) SaveOverride(ctx context.Context, override Overrides) (Overrides, error) {
	override.UpdatedOn = time.Now()

	if err := r.database.ExecUpdateBuilder(ctx, r.database.Builder().
		Update("sm_overrides").
		SetMap(map[string]any{
			"type":       override.Type,
			"name":       override.Name,
			"flags":      override.Flags,
			"updated_on": override.UpdatedOn,
		}).
		Where(sq.Eq{"override_id": override.OverrideID})); err != nil {
		return Overrides{}, database.DBErr(err)
	}

	return override, nil
}

func (r Repository) GroupOverrides(ctx context.Context, group Groups) ([]GroupOverrides, error) {
	rows, errRows := r.database.QueryBuilder(ctx, r.database.Builder().
		Select("group_override_id", "group_id", "type", "name", "access", "created_on", "updated_on").
		From("sm_group_overrides").
		Where(sq.Eq{"group_id": group.GroupID}))
	if errRows != nil {
		return nil, database.DBErr(errRows)
	}

	var overrides []GroupOverrides

	for rows.Next() {
		var override GroupOverrides
		if errScan := rows.Scan(&override.GroupOverrideID, &override.GroupID, &override.Type, &override.Name, &override.Access,
			&override.CreatedOn, &override.UpdatedOn); errScan != nil {
			return nil, database.DBErr(errScan)
		}

		overrides = append(overrides, override)
	}

	return overrides, nil
}

func (r Repository) GetOverride(ctx context.Context, overrideID int) (Overrides, error) {
	var override Overrides

	row, err := r.database.QueryRowBuilder(ctx, r.database.Builder().
		Select("override_id", "type", "name", "flags", "created_on", "updated_on").
		From("sm_overrides").
		Where(sq.Eq{"override_id": overrideID}))
	if err != nil {
		return override, database.DBErr(err)
	}

	if errScan := row.Scan(&override.OverrideID, &override.Type, &override.Name,
		&override.Flags, &override.CreatedOn, &override.UpdatedOn); errScan != nil {
		return override, database.DBErr(errScan)
	}

	return override, nil
}

func (r Repository) AddOverride(ctx context.Context, overrides Overrides) (Overrides, error) {
	if err := r.database.ExecInsertBuilder(ctx, r.database.Builder().
		Insert("sm_overrides").SetMap(map[string]any{
		"type":       overrides.Type,
		"name":       overrides.Name,
		"flags":      overrides.Flags,
		"created_on": overrides.CreatedOn,
		"updated_on": overrides.UpdatedOn,
	})); err != nil {
		return overrides, database.DBErr(err)
	}

	return overrides, nil
}

func (r Repository) DelOverride(ctx context.Context, override Overrides) error {
	return database.DBErr(r.database.ExecDeleteBuilder(ctx, r.database.Builder().
		Delete("sm_overrides").
		Where(sq.Eq{"override_id": override.OverrideID}),
	))
}

func (r Repository) Overrides(ctx context.Context) ([]Overrides, error) {
	rows, errRows := r.database.QueryBuilder(ctx, r.database.Builder().
		Select("override_id", "type", "name", "flags", "created_on", "updated_on").
		From("sm_overrides"))
	if errRows != nil {
		return nil, database.DBErr(errRows)
	}

	var overrides []Overrides

	for rows.Next() {
		var override Overrides
		if errScan := rows.Scan(&override.OverrideID, &override.Type, &override.Name, &override.Flags,
			&override.CreatedOn, &override.UpdatedOn); errScan != nil {
			return nil, database.DBErr(errScan)
		}

		overrides = append(overrides, override)
	}

	return overrides, nil
}

func (r Repository) GetAdminGroups(ctx context.Context, admin Admin) ([]Groups, error) {
	rows, errRows := r.database.QueryBuilder(ctx, r.database.Builder().
		Select("g.id", "g.flags", "g.name", "g.immunity_level", "g.created_on", "g.updated_on").
		From("sm_groups g").
		LeftJoin("sm_admins_groups ag ON ag.group_id = g.id").
		Where(sq.Eq{"ag.admin_id": admin.AdminID}))
	if errRows != nil {
		return nil, errRows
	}

	var groups []Groups

	for rows.Next() {
		var group Groups
		if errScan := rows.Scan(&group.GroupID, &group.Flags, &group.Name, &group.ImmunityLevel,
			&group.CreatedOn, &group.UpdatedOn); errScan != nil {
			return nil, database.DBErr(errScan)
		}

		groups = append(groups, group)
	}

	if groups == nil {
		groups = []Groups{}
	}

	return groups, nil
}

func (r Repository) Admins(ctx context.Context) ([]Admin, error) {
	groups, errGroups := r.Groups(ctx)
	if errGroups != nil && !errors.Is(errGroups, database.ErrNoResult) {
		return nil, errGroups
	}

	rows, errRows := r.database.QueryBuilder(ctx, r.database.Builder().
		Select("a.id", "a.steam_id", "a.authtype", "a.identity", "a.password", "a.flags", "a.name", "a.immunity",
			"a.created_on", "a.updated_on", "array_agg(sag.group_id) as group_ids").
		From("sm_admins a").
		LeftJoin("sm_admins_groups sag on a.id = sag.admin_id").
		GroupBy("a.id"))
	if errRows != nil {
		if errors.Is(errRows, database.ErrNoResult) {
			return []Admin{}, nil
		}

		return nil, database.DBErr(errRows)
	}

	var admins []Admin

	for rows.Next() {
		var (
			// array_agg will return a {null} if no group entry exists
			groupIDs []*int
			admin    = Admin{Groups: []Groups{}}
		)

		if errScan := rows.Scan(&admin.AdminID, &admin.SteamID, &admin.AuthType, &admin.Identity, &admin.Password,
			&admin.Flags, &admin.Name, &admin.Immunity, &admin.CreatedOn, &admin.UpdatedOn, &groupIDs); errScan != nil {
			return nil, database.DBErr(errScan)
		}

		for _, groupID := range groupIDs {
			if groupID == nil {
				continue
			}

			for _, group := range groups {
				if group.GroupID == *groupID {
					admin.Groups = append(admin.Groups, group)
				}
			}
		}

		admins = append(admins, admin)
	}

	return admins, nil
}

func (r Repository) Groups(ctx context.Context) ([]Groups, error) {
	rows, errRows := r.database.QueryBuilder(ctx, r.database.Builder().
		Select("id", "flags", "name", "immunity_level", "created_on", "updated_on").
		From("sm_groups"))
	if errRows != nil {
		if errors.Is(errRows, database.ErrNoResult) {
			return []Groups{}, nil
		}

		return nil, database.DBErr(errRows)
	}

	var groups []Groups

	for rows.Next() {
		var group Groups
		if errScan := rows.Scan(&group.GroupID, &group.Flags, &group.Name, &group.ImmunityLevel,
			&group.CreatedOn, &group.UpdatedOn); errScan != nil {
			return nil, database.DBErr(errScan)
		}

		groups = append(groups, group)
	}

	if groups == nil {
		groups = []Groups{}
	}

	return groups, nil
}

func (r Repository) GetAdminByID(ctx context.Context, adminID int) (Admin, error) {
	var (
		admin Admin
		id64  *int64
	)

	row, errRow := r.database.QueryRowBuilder(ctx, r.database.Builder().
		Select("id", "steam_id", "authtype", "identity", "password", "flags", "name", "immunity", "created_on", "updated_on").
		From("sm_admins").
		Where(sq.And{sq.Eq{"id": adminID}}))
	if errRow != nil {
		return admin, database.DBErr(errRow)
	}

	if errScan := row.Scan(&admin.AdminID, &id64, &admin.AuthType, &admin.Identity,
		&admin.Password, &admin.Flags, &admin.Name, &admin.Immunity, &admin.CreatedOn, &admin.UpdatedOn); errScan != nil {
		return admin, database.DBErr(errScan)
	}

	if id64 != nil {
		admin.SteamID = steamid.New(*id64)
	}

	groups, errGroup := r.GetAdminGroups(ctx, admin)
	if errGroup != nil && !errors.Is(errGroup, database.ErrNoResult) {
		return Admin{}, errGroup
	}

	admin.Groups = groups

	return admin, nil
}

func (r Repository) SaveAdmin(ctx context.Context, admin Admin) (Admin, error) {
	admin.UpdatedOn = time.Now()

	var sid64 *int64

	if admin.SteamID.Valid() {
		sidValue := admin.SteamID.Int64()
		sid64 = &sidValue
	}

	if err := r.database.ExecUpdateBuilder(ctx, r.database.Builder().
		Update("sm_admins").
		SetMap(map[string]any{
			"steam_id":   sid64,
			"authtype":   admin.AuthType,
			"identity":   admin.Identity,
			"password":   admin.Password,
			"flags":      admin.Flags,
			"name":       admin.Name,
			"immunity":   admin.Immunity,
			"updated_on": admin.UpdatedOn,
		}).Where(sq.Eq{"id": admin.AdminID})); err != nil {
		return Admin{}, err
	}

	return admin, nil
}

func (r Repository) GetAdminByIdentity(ctx context.Context, authType AuthType, identity string) (Admin, error) {
	var (
		admin Admin
		id64  int64
	)

	row, errRow := r.database.QueryRowBuilder(ctx, r.database.Builder().
		Select("id", "steam_id", "authtype", "identity", "password", "flags", "name", "immunity", "created_on", "updated_on").
		From("sm_admins").
		Where(sq.And{sq.Eq{"authtype": authType}, sq.Eq{"identity": identity}}))
	if errRow != nil {
		return admin, database.DBErr(errRow)
	}

	if errScan := row.Scan(&admin.AdminID, &id64, &admin.AuthType, &admin.Identity,
		&admin.Password, &admin.Flags, &admin.Name, &admin.Immunity, &admin.CreatedOn, &admin.UpdatedOn); errScan != nil {
		return admin, database.DBErr(errScan)
	}

	admin.SteamID = steamid.New(id64)

	return admin, nil
}

func (r Repository) GetGroupByName(ctx context.Context, groupName string) (Groups, error) {
	var group Groups

	row, errRow := r.database.QueryRowBuilder(ctx, r.database.Builder().
		Select("id", "flags", "name", "immunity_level", "created_on", "updated_on").
		From("sm_admins").
		Where(sq.Eq{"name": groupName}))
	if errRow != nil {
		return group, database.DBErr(errRow)
	}

	if errScan := row.Scan(&group.GroupID, &group.Flags, &group.Name, &group.ImmunityLevel,
		&group.CreatedOn, &group.UpdatedOn); errScan != nil {
		return group, database.DBErr(errScan)
	}

	return group, nil
}

func (r Repository) GetGroupByID(ctx context.Context, groupID int) (Groups, error) {
	var group Groups

	row, errRow := r.database.QueryRowBuilder(ctx, r.database.Builder().
		Select("id", "flags", "name", "immunity_level", "created_on", "updated_on").
		From("sm_groups").
		Where(sq.Eq{"id": groupID}))
	if errRow != nil {
		return group, database.DBErr(errRow)
	}

	if errScan := row.Scan(&group.GroupID, &group.Flags, &group.Name, &group.ImmunityLevel,
		&group.CreatedOn, &group.UpdatedOn); errScan != nil {
		return group, database.DBErr(errScan)
	}

	return group, nil
}

func (r Repository) InsertAdminGroup(ctx context.Context, admin Admin, group Groups, inheritOrder int) error {
	now := time.Now()

	return database.DBErr(r.database.ExecInsertBuilder(ctx, r.database.Builder().
		Insert("sm_admins_groups").
		SetMap(map[string]any{
			"admin_id":      admin.AdminID,
			"group_id":      group.GroupID,
			"inherit_order": inheritOrder,
			"created_on":    now,
			"updated_on":    now,
		})))
}

func (r Repository) DeleteAdminGroups(ctx context.Context, admin Admin) error {
	if err := r.database.ExecDeleteBuilder(ctx, r.database.Builder().
		Delete("sm_admins_groups").
		Where(sq.Eq{"admin_id": admin.AdminID})); err != nil {
		return database.DBErr(err)
	}

	slog.Info("Deleted SM admin groups", slog.String("steam_id", admin.SteamID.String()))

	return nil
}

func (r Repository) DeleteAdminGroup(ctx context.Context, admin Admin, group Groups) error {
	return r.database.ExecDeleteBuilder(ctx, r.database.Builder().
		Delete("sm_admins_groups").
		Where(sq.And{sq.Eq{"admin_id": admin.AdminID}, sq.Eq{"group_id": group.GroupID}}))
}

func (r Repository) SaveGroup(ctx context.Context, group Groups) (Groups, error) {
	group.UpdatedOn = time.Now()
	if err := r.database.ExecUpdateBuilder(ctx, r.database.Builder().
		Update("sm_groups").
		SetMap(map[string]any{
			"name":           group.Name,
			"immunity_level": group.ImmunityLevel,
			"flags":          group.Flags,
		}).
		Where(sq.Eq{"id": group.GroupID})); err != nil {
		return group, database.DBErr(err)
	}

	return group, nil
}

func (r Repository) DeleteGroup(ctx context.Context, group Groups) error {
	if err := r.database.ExecDeleteBuilder(ctx, r.database.Builder().
		Delete("sm_admins_groups").
		Where(sq.Eq{"group_id": group.GroupID})); err != nil {
		return database.DBErr(err)
	}

	if err := r.database.ExecDeleteBuilder(ctx, r.database.Builder().
		Delete("sm_group_overrides").
		Where(sq.Eq{"group_id": group.GroupID})); err != nil {
		return database.DBErr(err)
	}

	if err := r.database.ExecDeleteBuilder(ctx, r.database.Builder().
		Delete("sm_group_immunity").
		Where(sq.Or{sq.Eq{"group_id": group.GroupID}, sq.Eq{"other_id": group.GroupID}})); err != nil {
		return database.DBErr(err)
	}

	if err := r.database.ExecDeleteBuilder(ctx, r.database.Builder().
		Delete("sm_groups").
		Where(sq.Eq{"id": group.GroupID})); err != nil {
		return database.DBErr(err)
	}

	slog.Info("Deleted SM Group", slog.Int("group_id", group.GroupID), slog.String("name", group.Name))

	return nil
}

func (r Repository) AddGroup(ctx context.Context, group Groups) (Groups, error) {
	now := time.Now()

	if err := r.database.ExecInsertBuilderWithReturnValue(ctx, r.database.
		Builder().
		Insert("sm_groups").
		SetMap(map[string]any{
			"name":           group.Name,
			"immunity_level": group.ImmunityLevel,
			"flags":          group.Flags,
			"created_on":     now,
			"updated_on":     now,
		}).
		Suffix("RETURNING id"), &group.GroupID); err != nil {
		return group, err
	}

	slog.Info("Created SM Group", slog.Int("group_id", group.GroupID), slog.String("name", group.Name))

	return group, nil
}

func (r Repository) DelAdmin(ctx context.Context, admin Admin) error {
	if err := r.database.ExecDeleteBuilder(ctx, r.database.Builder().
		Delete("sm_admins_groups").
		Where(sq.Eq{"admin_id": admin.AdminID})); err != nil {
		return database.DBErr(err)
	}

	if err := r.database.ExecDeleteBuilder(ctx, r.database.Builder().
		Delete("sm_admins").
		Where(sq.Eq{"id": admin.AdminID})); err != nil {
		return database.DBErr(err)
	}

	slog.Info("Deleted SM Admin", slog.Int("id", admin.AdminID),
		slog.String("steam_id", admin.SteamID.String()))

	return nil
}

func (r Repository) AddAdmin(ctx context.Context, admin Admin) (Admin, error) {
	var nullableSID64 *int64

	if admin.SteamID.Valid() {
		steamID := admin.SteamID.Int64()
		nullableSID64 = &steamID
	}

	now := time.Now()

	admin.CreatedOn = now
	admin.UpdatedOn = now

	if err := r.database.ExecInsertBuilderWithReturnValue(ctx, r.database.Builder().
		Insert("sm_admins").
		SetMap(map[string]any{
			"steam_id":   nullableSID64,
			"authtype":   admin.AuthType,
			"identity":   admin.Identity,
			"password":   admin.Password,
			"flags":      admin.Flags,
			"name":       admin.Name,
			"immunity":   admin.Immunity,
			"created_on": admin.CreatedOn,
			"updated_on": admin.UpdatedOn,
		}).
		Suffix("RETURNING id"), &admin.AdminID); err != nil {
		return admin, database.DBErr(err)
	}

	slog.Info("Added SM Admin", slog.Int("id", admin.AdminID), slog.String("identity", admin.Identity))

	return admin, nil
}
