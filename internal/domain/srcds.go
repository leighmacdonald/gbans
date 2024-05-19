package domain

import (
	"context"
	"errors"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	ErrSMInvalidAuthName  = errors.New("invalid auth name")
	ErrSMImmunity         = errors.New("invalid immunity level, must be between 0-100")
	ErrSMGroupName        = errors.New("group name cannot be empty")
	ErrSMAdminGroupExists = errors.New("admin group already exists")
	ErrSMAdminExists      = errors.New("admin already exists")
	ErrSMAdminFlagInvalid = errors.New("invalid admin flag")
	ErrSMRequirePassword  = errors.New("name auth type requires password")
)

type SRCDSRepository interface {
	AddAdmin(ctx context.Context, admin SMAdmin) (SMAdmin, error)
	SaveAdmin(ctx context.Context, admin SMAdmin) (SMAdmin, error)
	DelAdmin(ctx context.Context, admin SMAdmin) error
	AddGroup(ctx context.Context, group SMGroups) (SMGroups, error)
	DeleteGroup(ctx context.Context, group SMGroups) error
	DeleteAdminGroups(ctx context.Context, admin SMAdmin) error
	DeleteAdminGroup(ctx context.Context, admin SMAdmin, group SMGroups) error
	InsertAdminGroup(ctx context.Context, admin SMAdmin, group SMGroups, inheritOrder int) error
	GetGroupByID(ctx context.Context, groupID int) (SMGroups, error)
	GetGroupByName(ctx context.Context, groupName string) (SMGroups, error)
	GetAdminByID(ctx context.Context, adminID int) (SMAdmin, error)
	GetAdminByIdentity(ctx context.Context, authType AuthType, identity string) (SMAdmin, error)
	SaveGroup(ctx context.Context, group SMGroups) (SMGroups, error)
	GetAdminGroups(ctx context.Context, admin SMAdmin) ([]SMGroups, error)
	Admins(ctx context.Context) ([]SMAdmin, error)
	Groups(ctx context.Context) ([]SMGroups, error)
	AddGroupOverride(ctx context.Context, override SMGroupOverrides) error
	AddOverride(ctx context.Context, overrides SMOverrides) error
	GroupOverrides(ctx context.Context) ([]SMGroupOverrides, error)
	Overrides(ctx context.Context) ([]SMOverrides, error)
}

type SRCDSUsecase interface {
	ServerAuth(ctx context.Context, req ServerAuthReq) (string, error)
	Report(ctx context.Context, currentUser UserProfile, req CreateReportReq) (*Report, error)
	GetAdminByID(ctx context.Context, adminID int) (SMAdmin, error)
	AddAdmin(ctx context.Context, alias string, authType AuthType, identity string, flags string, immunity int, password string) (SMAdmin, error)
	DelAdmin(ctx context.Context, adminID int) error
	Admins(ctx context.Context) ([]SMAdmin, error)
	SaveAdmin(ctx context.Context, admin SMAdmin) (SMAdmin, error)
	AddGroup(ctx context.Context, name string, flags string, immunityLevel int) (SMGroups, error)
	DelGroup(ctx context.Context, groupID int) error
	GetGroupByID(ctx context.Context, groupID int) (SMGroups, error)
	Groups(ctx context.Context) ([]SMGroups, error)
	SaveGroup(ctx context.Context, group SMGroups) (SMGroups, error)
	GetAdminGroups(ctx context.Context, admin SMAdmin) ([]SMGroups, error)
	SetAdminGroups(ctx context.Context, authType AuthType, identity string, groups ...SMGroups) error
	AddAdminGroup(ctx context.Context, adminID int, groupID int) (SMAdmin, error)
	DelAdminGroup(ctx context.Context, adminID int, groupID int) (SMAdmin, error)
}

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

type SMAdmin struct {
	AdminID  int             `json:"admin_id"`
	SteamID  steamid.SteamID `json:"steam_id"`
	AuthType AuthType        `json:"auth_type"` // steam | name |ip
	Identity string          `json:"identity"`
	Password string          `json:"password"`
	Flags    string          `json:"flags"`
	Name     string          `json:"name"`
	Immunity int             `json:"immunity"`
	Groups   []SMGroups      `json:"groups"`
	TimeStamped
}

type SMGroups struct {
	GroupID       int    `json:"group_id"`
	Flags         string `json:"flags"`
	Name          string `json:"name"`
	ImmunityLevel int    `json:"immunity_level"`
	TimeStamped
}

type SMGroupImmunity struct {
	GroupID   int       `json:"group_id"`
	OtherID   int       `json:"other_id"`
	CreatedOn time.Time `json:"created_on"`
}

type SMGroupOverrides struct {
	GroupID int            `json:"group_id"`
	Type    OverrideType   `json:"type"` // command | group
	Name    string         `json:"name"`
	Access  OverrideAccess `json:"access"` // allow | deny
	TimeStamped
}

type SMOverrides struct {
	Type  OverrideType `json:"type"` // command | group
	Name  string       `json:"name"`
	Flags string       `json:"flags"`
	TimeStamped
}

type SMAdminGroups struct {
	AdminID      int `json:"admin_id"`
	GroupID      int `json:"group_id"`
	InheritOrder int `json:"inherit_order"`
	TimeStamped
}

type SMConfig struct {
	CfgKey   string `json:"cfg_key"`
	CfgValue string `json:"cfg_value"`
}

type ServerAuthReq struct {
	Key string `json:"key"`
}
