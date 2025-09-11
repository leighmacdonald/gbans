package srcds

import (
	"context"
	"errors"
	"net/netip"
	"time"

	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	ErrSMInvalidAuthName   = errors.New("invalid auth name")
	ErrSMImmunity          = errors.New("invalid immunity level, must be between 0-100")
	ErrSMGroupName         = errors.New("group name cannot be empty")
	ErrSMAdminGroupExists  = errors.New("admin group already exists")
	ErrSMAdminExists       = errors.New("admin already exists")
	ErrSMAdminFlagInvalid  = errors.New("invalid admin flag")
	ErrSMRequirePassword   = errors.New("name auth type requires password")
	ErrInsufficientDetails = errors.New("insufficient details")
)

type SRCDSRepository interface { //nolint:interfacebloat
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
	AddOverride(ctx context.Context, overrides SMOverrides) (SMOverrides, error)
	Overrides(ctx context.Context) ([]SMOverrides, error)
	DelOverride(ctx context.Context, override SMOverrides) error
	GetOverride(ctx context.Context, overrideID int) (SMOverrides, error)
	SaveOverride(ctx context.Context, override SMOverrides) (SMOverrides, error)
	GroupOverrides(ctx context.Context, group SMGroups) ([]SMGroupOverrides, error)
	AddGroupOverride(ctx context.Context, override SMGroupOverrides) (SMGroupOverrides, error)
	DelGroupOverride(ctx context.Context, override SMGroupOverrides) error
	GetGroupOverride(ctx context.Context, overrideID int) (SMGroupOverrides, error)
	SaveGroupOverride(ctx context.Context, override SMGroupOverrides) (SMGroupOverrides, error)
	GetGroupImmunities(ctx context.Context) ([]SMGroupImmunity, error)
	GetGroupImmunityByID(ctx context.Context, groupImmunityID int) (SMGroupImmunity, error)
	AddGroupImmunity(ctx context.Context, group SMGroups, other SMGroups) (SMGroupImmunity, error)
	DelGroupImmunity(ctx context.Context, groupImmunity SMGroupImmunity) error
	QueryBanState(ctx context.Context, steamID steamid.SteamID, ipAddr netip.Addr) (PlayerBanState, error)
}

type SRCDSUsecase interface { //nolint:interfacebloat
	GetBanState(ctx context.Context, steamID steamid.SteamID, ip netip.Addr) (PlayerBanState, string, error)
	Report(ctx context.Context, currentUser UserProfile, req RequestReportCreate) (ReportWithAuthor, error)
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
	GroupOverrides(ctx context.Context, groupID int) ([]SMGroupOverrides, error)
	Overrides(ctx context.Context) ([]SMOverrides, error)
	AddOverride(ctx context.Context, name string, overrideType OverrideType, flags string) (SMOverrides, error)
	DelOverride(ctx context.Context, overrideID int) error
	GetOverride(ctx context.Context, overrideID int) (SMOverrides, error)
	SaveOverride(ctx context.Context, override SMOverrides) (SMOverrides, error)
	AddGroupOverride(ctx context.Context, groupID int, name string, overrideType OverrideType, access OverrideAccess) (SMGroupOverrides, error)
	DelGroupOverride(ctx context.Context, groupOverrideID int) error
	GetGroupOverride(ctx context.Context, groupOverrideID int) (SMGroupOverrides, error)
	SaveGroupOverride(ctx context.Context, override SMGroupOverrides) (SMGroupOverrides, error)
	GetGroupImmunities(ctx context.Context) ([]SMGroupImmunity, error)
	GetGroupImmunityByID(ctx context.Context, groupImmunityID int) (SMGroupImmunity, error)
	AddGroupImmunity(ctx context.Context, groupID int, otherID int) (SMGroupImmunity, error)
	DelGroupImmunity(ctx context.Context, groupImmunityID int) error
}

type BanSource string

const (
	BanSourceNone        BanSource = ""
	BanSourceSteam       BanSource = "ban_steam"
	BanSourceSteamFriend BanSource = "ban_steam_friend"
	BanSourceSteamGroup  BanSource = "steam_group"
	BanSourceSteamNet    BanSource = "ban_net"
	BanSourceCIDR        BanSource = "cidr_block"
	BanSourceASN         BanSource = "ban_asn"
)

type PlayerBanState struct {
	SteamID    steamid.SteamID `json:"steam_id"`
	BanSource  BanSource       `json:"ban_source"`
	BanID      int             `json:"ban_id"`
	BanType    ban.BanType     `json:"ban_type"`
	Reason     ban.Reason      `json:"reason"`
	EvadeOK    bool            `json:"evade_ok"`
	ValidUntil time.Time       `json:"valid_until"`
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
	AdminID   int             `json:"admin_id"`
	SteamID   steamid.SteamID `json:"steam_id"`
	AuthType  AuthType        `json:"auth_type"` // steam | name |ip
	Identity  string          `json:"identity"`
	Password  string          `json:"password"`
	Flags     string          `json:"flags"`
	Name      string          `json:"name"`
	Immunity  int             `json:"immunity"`
	Groups    []SMGroups      `json:"groups"`
	CreatedOn time.Time       `json:"created_on"`
	UpdatedOn time.Time       `json:"updated_on"`
}

type SMGroups struct {
	GroupID       int       `json:"group_id"`
	Flags         string    `json:"flags"`
	Name          string    `json:"name"`
	ImmunityLevel int       `json:"immunity_level"`
	CreatedOn     time.Time `json:"created_on"`
	UpdatedOn     time.Time `json:"updated_on"`
}

type SMGroupImmunity struct {
	GroupImmunityID int       `json:"group_immunity_id"`
	Group           SMGroups  `json:"group"`
	Other           SMGroups  `json:"other"`
	CreatedOn       time.Time `json:"created_on"`
}

type SMGroupOverrides struct {
	GroupOverrideID int            `json:"group_override_id"`
	GroupID         int            `json:"group_id"`
	Type            OverrideType   `json:"type"` // command | group
	Name            string         `json:"name"`
	Access          OverrideAccess `json:"access"` // allow | deny
	CreatedOn       time.Time      `json:"created_on"`
	UpdatedOn       time.Time      `json:"updated_on"`
}

type SMOverrides struct {
	OverrideID int          `json:"override_id"`
	Type       OverrideType `json:"type"` // command | group
	Name       string       `json:"name"`
	Flags      string       `json:"flags"`
	CreatedOn  time.Time    `json:"created_on"`
	UpdatedOn  time.Time    `json:"updated_on"`
}

type SMAdminGroups struct {
	AdminID      int       `json:"admin_id"`
	GroupID      int       `json:"group_id"`
	InheritOrder int       `json:"inherit_order"`
	CreatedOn    time.Time `json:"created_on"`
	UpdatedOn    time.Time `json:"updated_on"`
}

type SMConfig struct {
	CfgKey   string `json:"cfg_key"`
	CfgValue string `json:"cfg_value"`
}

type ServerAuthReq struct {
	Key string `json:"key"`
}
