package srcds

import (
	"context"
	"log/slog"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type srcdsRepository struct {
	database database.Database
}

func NewRepository(database database.Database) domain.SRCDSRepository {
	return srcdsRepository{database: database}
}

func (r srcdsRepository) GetAdminByID(ctx context.Context, authType domain.AuthType, identity string) (domain.SMAdmin, error) {
	var (
		admin domain.SMAdmin
		id64  int64
	)

	row, errRow := r.database.QueryRowBuilder(ctx, r.database.Builder().
		Select("id", "steam_id", "authtype", "identity", "password", "flags", "name", "immunity").
		From("sm_admins").
		Where(sq.And{sq.Eq{"authtype": authType}, sq.Eq{"identity": identity}}))
	if errRow != nil {
		return admin, r.database.DBErr(errRow)
	}

	if errScan := row.Scan(&admin.AdminID, &id64, &admin.AuthType, &admin.Identity,
		&admin.Password, &admin.Flags, &admin.Name, &admin.Immunity); errScan != nil {
		return admin, r.database.DBErr(errScan)
	}

	admin.SteamID = steamid.New(id64)

	return admin, nil
}

func (r srcdsRepository) GetGroupByName(ctx context.Context, groupName string) (domain.SMGroups, error) {
	var group domain.SMGroups

	row, errRow := r.database.QueryRowBuilder(ctx, r.database.Builder().
		Select("id", "flags", "name", "immunity_level").
		From("sm_admins").
		Where(sq.Eq{"name": groupName}))
	if errRow != nil {
		return group, r.database.DBErr(errRow)
	}

	if errScan := row.Scan(&group.GroupID, &group.Flags, &group.Name, &group.ImmunityLevel); errScan != nil {
		return group, r.database.DBErr(errScan)
	}

	return group, nil
}

func (r srcdsRepository) GetGroupByID(ctx context.Context, groupID int) (domain.SMGroups, error) {
	var group domain.SMGroups

	row, errRow := r.database.QueryRowBuilder(ctx, r.database.Builder().
		Select("id", "flags", "name", "immunity_level").
		From("sm_admins").
		Where(sq.Eq{"group_id": groupID}))
	if errRow != nil {
		return group, r.database.DBErr(errRow)
	}

	if errScan := row.Scan(&group.GroupID, &group.Flags, &group.Name, &group.ImmunityLevel); errScan != nil {
		return group, r.database.DBErr(errScan)
	}

	return group, nil
}

func (r srcdsRepository) InsertAdminGroup(ctx context.Context, admin domain.SMAdmin, group domain.SMGroups, inheritOrder int) error {
	return r.database.DBErr(r.database.ExecInsertBuilder(ctx, r.database.Builder().
		Insert("sm_admins_groups").
		SetMap(map[string]interface{}{
			"admin_id":      admin.AdminID,
			"group_id":      group.GroupID,
			"inherit_order": inheritOrder,
		})))
}

func (r srcdsRepository) DeleteAdminGroups(ctx context.Context, admin domain.SMAdmin) error {
	if err := r.database.ExecDeleteBuilder(ctx, r.database.Builder().
		Delete("sm_admins_groups").
		Where(sq.Eq{"admin_id": admin.AdminID})); err != nil {
		return r.database.DBErr(err)
	}

	slog.Info("Deleted SM admin groups", slog.String("steam_id", admin.SteamID.String()))

	return nil
}

func (r srcdsRepository) DeleteGroup(ctx context.Context, group domain.SMGroups) error {
	if err := r.database.ExecDeleteBuilder(ctx, r.database.Builder().
		Delete("sm_admins_groups").
		Where(sq.Eq{"group_id": group.GroupID})); err != nil {
		return r.database.DBErr(err)
	}

	if err := r.database.ExecDeleteBuilder(ctx, r.database.Builder().
		Delete("sm_group_overrides").
		Where(sq.Eq{"group_id": group.GroupID})); err != nil {
		return r.database.DBErr(err)
	}

	if err := r.database.ExecDeleteBuilder(ctx, r.database.Builder().
		Delete("sm_group_immunity").
		Where(sq.Or{sq.Eq{"group_id": group.GroupID}, sq.Eq{"other_id": group.GroupID}})); err != nil {
		return r.database.DBErr(err)
	}

	if err := r.database.ExecDeleteBuilder(ctx, r.database.Builder().
		Delete("sm_groups").
		Where(sq.Eq{"group_id": group.GroupID})); err != nil {
		return r.database.DBErr(err)
	}

	slog.Info("Deleted SM Group", slog.Int("group_id", group.GroupID), slog.String("name", group.Name))

	return nil
}

func (r srcdsRepository) AddGroup(ctx context.Context, group domain.SMGroups) (domain.SMGroups, error) {
	if err := r.database.ExecInsertBuilderWithReturnValue(ctx, r.database.Builder().
		Insert("sm_groups"), &group.GroupID); err != nil {
		return group, err
	}

	slog.Info("Created SM Group", slog.Int("group_id", group.GroupID), slog.String("name", group.Name))

	return group, nil
}

func (r srcdsRepository) DelAdmin(ctx context.Context, admin domain.SMAdmin) error {
	if err := r.database.ExecDeleteBuilder(ctx, r.database.Builder().
		Delete("sm_admins_group").
		Where(sq.Eq{"admin_id": admin.AdminID})); err != nil {
		return r.database.DBErr(err)
	}

	if err := r.database.ExecDeleteBuilder(ctx, r.database.Builder().
		Delete("sm_admins").
		Where(sq.Eq{"admin_id": admin.AdminID})); err != nil {
		return r.database.DBErr(err)
	}

	slog.Info("Deleted SM Admin", slog.Int("admin_id", admin.AdminID),
		slog.String("steam_id", admin.SteamID.String()))

	return nil
}

func (r srcdsRepository) AddAdmin(ctx context.Context, admin domain.SMAdmin) (domain.SMAdmin, error) {
	var nullableSID64 *int64

	if admin.SteamID.Valid() {
		steamID := admin.SteamID.Int64()
		nullableSID64 = &steamID
	}

	if err := r.database.ExecInsertBuilderWithReturnValue(ctx, r.database.Builder().
		Insert("sm_admins").
		SetMap(map[string]interface{}{
			"steam_id": nullableSID64,
			"authtype": admin.AuthType,
			"identity": admin.Identity,
			"password": admin.Password,
			"flags":    admin.Flags,
			"name":     admin.Name,
			"immunity": admin.Immunity,
		}).
		Suffix("RETURNING admin_id"), &admin.AdminID); err != nil {
		return admin, r.database.DBErr(err)
	}

	slog.Info("Added SM Admin", slog.Int("admin_id", admin.AdminID), slog.String("identity", admin.Identity))

	return admin, nil
}
