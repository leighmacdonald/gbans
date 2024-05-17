package srcds

import (
	"context"
	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"log/slog"
)

type AuthType string

const (
	AuthTypeSteam AuthType = "steam"
	AuthTypeName  AuthType = "name"
	AuthTypeIP    AuthType = "ip"
)

type OverrideType string

const (
	OverrideTypeCommand OverrideType = "command"
	OverrideTypeGroup   OverrideType = "group"
)

type OverrideAccess string

const (
	OverrideAccessAllow OverrideAccess = "allow"
	OverrideAccessDeny  OverrideAccess = "deny"
)

type SRCSDRepository interface {
	Admins()
	Groups()
}

type SMAdmin struct {
	AdminID  int
	SteamID  steamid.SteamID
	AuthType AuthType // steam | name |ip
	Identity string
	Password string
	Flags    string
	Name     string
	Immunity int
}

type SMGroups struct {
	GroupID       int
	Flags         string
	Name          string
	ImmunityLevel int
}

type SMGroupImmunity struct {
	GroupID int
	OtherID int
}

type SMGroupOverrides struct {
	GroupID int
	Type    OverrideType // command | group
	Name    string
	Access  OverrideAccess // allow | deny
}

type SMOverrides struct {
	Type  OverrideType // command | group
	Name  string
	Flags string
}

type SMAdminGroups struct {
	AdminID      int
	GroupID      int
	InheritOrder int
}

type SMConfig struct {
	CfgKey   string
	CfgValue string
}

type srcdsRepository struct {
	database database.Database
}

func newRepository(database database.Database) srcdsRepository {
	return srcdsRepository{database: database}
}

func (r srcdsRepository) GetAdminByID(ctx context.Context, authType AuthType, identity string) (SMAdmin, error) {
	var (
		admin SMAdmin
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

func (r srcdsRepository) GetGroupByName(ctx context.Context, groupName string) (SMGroups, error) {
	var group SMGroups
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

func (r srcdsRepository) DeleteAdminGroups(ctx context.Context, admin SMAdmin) error {
	if err := r.database.ExecDeleteBuilder(ctx, r.database.Builder().
		Delete("sm_admins_groups").
		Where(sq.Eq{"admin_id": admin.AdminID})); err != nil {
		return r.database.DBErr(err)
	}

	slog.Info("Deleted users admins groups", slog.String("steam_id", admin.SteamID.String()))

	return nil
}
