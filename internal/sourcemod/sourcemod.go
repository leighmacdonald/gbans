package sourcemod

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/ban/bantype"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	ErrInvalidAuthName  = errors.New("invalid auth name")
	ErrImmunity         = errors.New("invalid immunity level, must be between 0-100")
	ErrGroupName        = errors.New("group name cannot be empty")
	ErrAdminGroupExists = errors.New("admin group already exists")
	ErrAdminExists      = errors.New("admin already exists")
	ErrAdminFlagInvalid = errors.New("invalid admin flag")
	ErrRequirePassword  = errors.New("name auth type requires password")
	ErrInvalidIP        = errors.New("invalid ip, could not parse")
	ErrGetPerson        = errors.New("failed to fetch person result")
)

type Config struct {
	CenterProjectiles bool `mapstructure:"center_projectiles"`
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
	BanType    bantype.Type    `json:"ban_type"`
	Reason     reason.Reason   `json:"reason"`
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

type Admin struct {
	AdminID   int             `json:"admin_id"`
	SteamID   steamid.SteamID `json:"steam_id"`
	AuthType  AuthType        `json:"auth_type"` // steam | name |ip
	Identity  string          `json:"identity"`
	Password  string          `json:"password"`
	Flags     string          `json:"flags"`
	Name      string          `json:"name"`
	Immunity  int             `json:"immunity"`
	Groups    []Groups        `json:"groups"`
	CreatedOn time.Time       `json:"created_on"`
	UpdatedOn time.Time       `json:"updated_on"`
}

type Groups struct {
	GroupID       int       `json:"group_id"`
	Flags         string    `json:"flags"`
	Name          string    `json:"name"`
	ImmunityLevel int       `json:"immunity_level"`
	CreatedOn     time.Time `json:"created_on"`
	UpdatedOn     time.Time `json:"updated_on"`
}

type GroupImmunity struct {
	GroupImmunityID int       `json:"group_immunity_id"`
	Group           Groups    `json:"group"`
	Other           Groups    `json:"other"`
	CreatedOn       time.Time `json:"created_on"`
}

type GroupOverrides struct {
	GroupOverrideID int            `json:"group_override_id"`
	GroupID         int            `json:"group_id"`
	Type            OverrideType   `json:"type"` // command | group
	Name            string         `json:"name"`
	Access          OverrideAccess `json:"access"` // allow | deny
	CreatedOn       time.Time      `json:"created_on"`
	UpdatedOn       time.Time      `json:"updated_on"`
}

type Overrides struct {
	OverrideID int          `json:"override_id"`
	Type       OverrideType `json:"type"` // command | group
	Name       string       `json:"name"`
	Flags      string       `json:"flags"`
	CreatedOn  time.Time    `json:"created_on"`
	UpdatedOn  time.Time    `json:"updated_on"`
}

type AdminGroups struct {
	AdminID      int       `json:"admin_id"`
	GroupID      int       `json:"group_id"`
	InheritOrder int       `json:"inherit_order"`
	CreatedOn    time.Time `json:"created_on"`
	UpdatedOn    time.Time `json:"updated_on"`
}

type ConfigEntry struct {
	CfgKey   string `json:"cfg_key"`
	CfgValue string `json:"cfg_value"`
}

func New(repository Repository, person person.Provider, notifier notification.Notifier, seedChannelID string) Sourcemod {
	return Sourcemod{
		seedChannelID: seedChannelID,
		repository:    repository,
		person:        person,
		notifier:      notifier,
		seedQueue: &seedQueue{
			minTime: time.Second * 300,
			servers: make(map[int]seedRequest),
			mu:      &sync.Mutex{},
		},
	}
}

type Sourcemod struct {
	seedChannelID string
	repository    Repository
	person        person.Provider
	seedQueue     *seedQueue
	notifier      notification.Notifier
}

func (h Sourcemod) seedRequest(server servers.Server, steamID steamid.SteamID) bool {
	const format = `# Seed Request
{{ .Name }}
connect {{ .Path }}

{{- range .Roles }}<@&{{ . }}> {{end}}
`

	if !h.seedQueue.allowed(server.ServerID, steamID) {
		return false
	}

	if len(server.DiscordSeedRoleIDs) > 0 {
		content, errContent := discord.Render("seed_req", format, struct {
			Name  string
			Path  string
			Roles []string
		}{
			Name:  server.ShortName,
			Path:  server.Addr(),
			Roles: server.DiscordSeedRoleIDs,
		})

		if errContent != nil {
			slog.Error("Failed to render content", slog.String("error", errContent.Error()))

			return false
		}

		msg := discord.NewMessageSend(discordgo.Container{
			AccentColor: ptr.To(discord.ColourSuccess),
			Components: []discordgo.MessageComponent{
				discordgo.TextDisplay{Content: content},
			},
		})

		go h.notifier.Send(notification.NewDiscord(h.seedChannelID, msg))

		return true
	}

	slog.Error("No seed channel found", slog.String("server", server.ShortName))

	return false
}

func (h Sourcemod) GetBanState(ctx context.Context, steamID steamid.SteamID, ip netip.Addr) (PlayerBanState, string, error) {
	banState, errBanState := h.repository.QueryBanState(ctx, steamID, ip)
	if errBanState != nil || banState.BanID == 0 {
		return banState, "", errBanState
	}

	const format = "Banned\nReason: %s (%s)\nUntil: %s\nAppeal: %s"

	var msg string

	validUntil := banState.ValidUntil.Format(time.ANSIC)
	if banState.ValidUntil.After(time.Now().AddDate(5, 0, 0)) {
		validUntil = "Permanent"
	}

	appealURL := "n/a"
	if banState.BanSource == BanSourceSteam {
		appealURL = fmt.Sprintf("/appeal/%d", banState.BanID)
	}

	if banState.BanID > 0 && banState.BanType >= bantype.NoComm {
		switch banState.BanSource {
		case BanSourceSteam:
			if banState.BanType == bantype.NoComm {
				msg = fmt.Sprintf("You are muted & gagged. Expires: %s. Appeal: %s", banState.ValidUntil.Format(time.DateTime), appealURL)
			} else {
				msg = fmt.Sprintf(format, banState.Reason.String(), "Steam", validUntil, appealURL)
			}
		case BanSourceASN:
			msg = fmt.Sprintf(format, banState.Reason.String(), "ASN", "Permanent", appealURL)
		case BanSourceCIDR:
			msg = "Blocked Network/VPN\nPlease disable your VPN if you are using one."
		case BanSourceSteamFriend:
			msg = "Friend Network Ban"
		case BanSourceSteamGroup:
			msg = "Blocked Steam Group"
		case BanSourceSteamNet:
			msg = fmt.Sprintf(format, banState.Reason.String(), "Steam Net", "Permanent", appealURL)
		}
	}

	return banState, msg, nil
}

func (h Sourcemod) Override(ctx context.Context, overrideID int) (Overrides, error) {
	return h.repository.GetOverride(ctx, overrideID)
}

func (h Sourcemod) GroupImmunityByID(ctx context.Context, groupImmunityID int) (GroupImmunity, error) {
	return h.repository.GetGroupImmunityByID(ctx, groupImmunityID)
}

func (h Sourcemod) GroupImmunities(ctx context.Context) ([]GroupImmunity, error) {
	return h.repository.GetGroupImmunities(ctx)
}

func (h Sourcemod) AddGroupImmunity(ctx context.Context, groupID int, otherID int) (GroupImmunity, error) {
	if groupID == otherID {
		return GroupImmunity{}, httphelper.ErrBadRequest // TODO fix error
	}

	group, errGroup := h.GetGroupByID(ctx, groupID)
	if errGroup != nil {
		return GroupImmunity{}, errGroup
	}

	other, errOther := h.GetGroupByID(ctx, otherID)
	if errOther != nil {
		return GroupImmunity{}, errOther
	}

	return h.repository.AddGroupImmunity(ctx, group, other)
}

func (h Sourcemod) DelGroupImmunity(ctx context.Context, groupImmunityID int) error {
	immunity, errImmunity := h.GroupImmunityByID(ctx, groupImmunityID)
	if errImmunity != nil {
		return errImmunity
	}

	if err := h.repository.DelGroupImmunity(ctx, immunity); err != nil {
		return err
	}

	slog.Info("Deleted group immunity", slog.Int("group_immunity_id", immunity.GroupImmunityID))

	return nil
}

func (h Sourcemod) AddGroupOverride(ctx context.Context, groupID int, name string, overrideType OverrideType, access OverrideAccess) (GroupOverrides, error) {
	if name == "" || overrideType == "" {
		return GroupOverrides{}, httphelper.ErrInvalidParameter
	}

	if access != OverrideAccessAllow && access != OverrideAccessDeny {
		return GroupOverrides{}, httphelper.ErrInvalidParameter
	}

	now := time.Now()

	override, err := h.repository.AddGroupOverride(ctx, GroupOverrides{
		GroupID:   groupID,
		Type:      overrideType,
		Name:      name,
		Access:    access,
		CreatedOn: now,
		UpdatedOn: now,
	})
	if err != nil {
		return override, err
	}

	slog.Info("Added group override", slog.Int("group_id", groupID), slog.String("name", name))

	return override, nil
}

func (h Sourcemod) DelGroupOverride(ctx context.Context, groupOverrideID int) error {
	override, errOverride := h.GroupOverride(ctx, groupOverrideID)
	if errOverride != nil {
		return errOverride
	}

	return h.repository.DelGroupOverride(ctx, override)
}

func (h Sourcemod) GroupOverride(ctx context.Context, groupOverrideID int) (GroupOverrides, error) {
	return h.repository.GetGroupOverride(ctx, groupOverrideID)
}

func (h Sourcemod) SaveGroupOverride(ctx context.Context, override GroupOverrides) (GroupOverrides, error) {
	if override.Name == "" || override.Type == "" {
		return GroupOverrides{}, httphelper.ErrInvalidParameter
	}

	if override.Access != OverrideAccessAllow && override.Access != OverrideAccessDeny {
		return GroupOverrides{}, httphelper.ErrInvalidParameter
	}

	return h.repository.SaveGroupOverride(ctx, override)
}

func (h Sourcemod) GroupOverrides(ctx context.Context, groupID int) ([]GroupOverrides, error) {
	group, errGroup := h.GetGroupByID(ctx, groupID)
	if errGroup != nil {
		return []GroupOverrides{}, errGroup
	}

	return h.repository.GroupOverrides(ctx, group)
}

func (h Sourcemod) Overrides(ctx context.Context) ([]Overrides, error) {
	return h.repository.Overrides(ctx)
}

func (h Sourcemod) SaveOverride(ctx context.Context, override Overrides) (Overrides, error) {
	if override.Name == "" || override.Flags == "" || override.Type != OverrideTypeCommand && override.Type != OverrideTypeGroup {
		return Overrides{}, httphelper.ErrInvalidParameter
	}

	return h.repository.SaveOverride(ctx, override)
}

func (h Sourcemod) AddOverride(ctx context.Context, name string, overrideType OverrideType, flags string) (Overrides, error) {
	if name == "" || flags == "" || overrideType != OverrideTypeCommand && overrideType != OverrideTypeGroup {
		return Overrides{}, httphelper.ErrInvalidParameter
	}

	now := time.Now()

	return h.repository.AddOverride(ctx, Overrides{
		Type:      overrideType,
		Name:      name,
		Flags:     flags,
		CreatedOn: now,
		UpdatedOn: now,
	})
}

func (h Sourcemod) DelOverride(ctx context.Context, overrideID int) error {
	override, errOverride := h.repository.GetOverride(ctx, overrideID)
	if errOverride != nil {
		return errOverride
	}

	return h.repository.DelOverride(ctx, override)
}

func (h Sourcemod) DelAdminGroup(ctx context.Context, adminID int, groupID int) (Admin, error) {
	admin, errAdmin := h.AdminByID(ctx, adminID)
	if errAdmin != nil {
		return Admin{}, errAdmin
	}

	group, errGroup := h.GetGroupByID(ctx, groupID)
	if errGroup != nil {
		return Admin{}, errGroup
	}

	existing, errExisting := h.AdminGroups(ctx, admin)
	if errExisting != nil && !errors.Is(errExisting, database.ErrNoResult) {
		return admin, errExisting
	}

	if !slices.Contains(existing, group) {
		return admin, ErrAdminGroupExists
	}

	if err := h.repository.DeleteAdminGroup(ctx, admin, group); err != nil {
		return Admin{}, err
	}

	admin.Groups = slices.DeleteFunc(admin.Groups, func(g Groups) bool {
		return g.GroupID == groupID
	})

	return admin, nil
}

func (h Sourcemod) AddAdminGroup(ctx context.Context, adminID int, groupID int) (Admin, error) {
	admin, errAdmin := h.AdminByID(ctx, adminID)
	if errAdmin != nil {
		return Admin{}, errAdmin
	}

	group, errGroup := h.GetGroupByID(ctx, groupID)
	if errGroup != nil {
		return Admin{}, errGroup
	}

	existing, errExisting := h.AdminGroups(ctx, admin)
	if errExisting != nil && !errors.Is(errExisting, database.ErrNoResult) {
		return admin, errExisting
	}

	if slices.Contains(existing, group) {
		return admin, ErrAdminGroupExists
	}

	if err := h.repository.InsertAdminGroup(ctx, admin, group, len(existing)+1); err != nil {
		return Admin{}, err
	}

	admin.Groups = append(admin.Groups, group)

	return admin, nil
}

func (h Sourcemod) AdminGroups(ctx context.Context, admin Admin) ([]Groups, error) {
	return h.repository.GetAdminGroups(ctx, admin)
}

func (h Sourcemod) SetAdminGroups(ctx context.Context, authType AuthType, identity string, groups ...Groups) error {
	admin, errAdmin := h.repository.GetAdminByIdentity(ctx, authType, identity)
	if errAdmin != nil {
		return errAdmin
	}

	// Delete existing groups.
	if errDelete := h.repository.DeleteAdminGroups(ctx, admin); errDelete != nil && !errors.Is(errDelete, database.ErrNoResult) {
		return errDelete
	}

	// If no groups are given to add, this is treated purely as a delete function
	if len(groups) == 0 {
		return nil
	}

	for i := range groups {
		if errInsert := h.repository.InsertAdminGroup(ctx, admin, groups[i], i); errInsert != nil {
			return errInsert
		}
	}

	return nil
}

func (h Sourcemod) DelGroup(ctx context.Context, groupID int) error {
	group, errGroup := h.repository.GetGroupByID(ctx, groupID)
	if errGroup != nil {
		return errGroup
	}

	return h.repository.DeleteGroup(ctx, group)
}

const validFlags = "zabcdefghijklmnopqrst"

func (h Sourcemod) AddGroup(ctx context.Context, name string, flags string, immunityLevel int) (Groups, error) {
	if name == "" {
		return Groups{}, ErrGroupName
	}

	if immunityLevel > 100 || immunityLevel < 0 {
		return Groups{}, ErrImmunity
	}

	for _, flag := range flags {
		if !strings.ContainsRune(validFlags, flag) {
			return Groups{}, ErrAdminFlagInvalid
		}
	}

	return h.repository.AddGroup(ctx, Groups{
		Flags:         flags,
		Name:          name,
		ImmunityLevel: immunityLevel,
	})
}

func validateAuthIdentity(ctx context.Context, authType AuthType, identity string, password string) (string, error) {
	switch authType {
	case AuthTypeSteam:
		steamID, errSteamID := steamid.Resolve(ctx, identity)
		if errSteamID != nil {
			return "", steamid.ErrDecodeSID
		}

		identity = steamID.String()
	case AuthTypeIP:
		if ip := net.ParseIP(identity); ip == nil || ip.To4() != nil {
			return "", ErrInvalidIP
		}
	case AuthTypeName:
		if identity == "" {
			return "", ErrInvalidAuthName
		}

		if password == "" {
			return "", ErrRequirePassword
		}
	}

	return identity, nil
}

func (h Sourcemod) DelAdmin(ctx context.Context, adminID int) error {
	admin, errAdmin := h.repository.GetAdminByID(ctx, adminID)
	if errAdmin != nil {
		return errAdmin
	}

	return h.repository.DelAdmin(ctx, admin)
}

func (h Sourcemod) AdminByID(ctx context.Context, adminID int) (Admin, error) {
	return h.repository.GetAdminByID(ctx, adminID)
}

func (h Sourcemod) SaveAdmin(ctx context.Context, admin Admin) (Admin, error) {
	realIdentity, errValidate := validateAuthIdentity(ctx, admin.AuthType, admin.Identity, admin.Password)
	if errValidate != nil {
		return Admin{}, errValidate
	}

	if admin.Immunity < 0 || admin.Immunity > 100 {
		return Admin{}, ErrImmunity
	}

	var steamID steamid.SteamID
	if admin.AuthType == AuthTypeSteam {
		steamID = steamid.New(realIdentity)
		if _, err := h.person.GetOrCreatePersonBySteamID(ctx, steamID); err != nil {
			return Admin{}, ErrGetPerson
		}

		admin.Identity = string(steamID.Steam3())
		admin.SteamID = steamID
	}

	return h.repository.SaveAdmin(ctx, admin)
}

func (h Sourcemod) AddAdmin(ctx context.Context, alias string, authType AuthType, identity string, flags string, immunity int, password string) (Admin, error) {
	realIdentity, errValidate := validateAuthIdentity(ctx, authType, identity, password)
	if errValidate != nil {
		return Admin{}, errValidate
	}

	if immunity < 0 || immunity > 100 {
		return Admin{}, ErrImmunity
	}

	admin, errAdmin := h.repository.GetAdminByIdentity(ctx, authType, realIdentity)
	if errAdmin != nil && !errors.Is(errAdmin, database.ErrNoResult) {
		return Admin{}, errAdmin
	}

	if errAdmin == nil {
		return admin, ErrAdminExists
	}

	var steamID steamid.SteamID
	if authType == AuthTypeSteam {
		steamID = steamid.New(realIdentity)
		if _, err := h.person.GetOrCreatePersonBySteamID(ctx, steamID); err != nil {
			return Admin{}, ErrGetPerson
		}

		identity = string(steamID.Steam3())
	}

	return h.repository.AddAdmin(ctx, Admin{
		SteamID:  steamID,
		AuthType: authType,
		Identity: identity,
		Password: password,
		Flags:    flags,
		Name:     alias,
		Immunity: immunity,
		Groups:   []Groups{},
	})
}

func (h Sourcemod) Admins(ctx context.Context) ([]Admin, error) {
	return h.repository.Admins(ctx)
}

func (h Sourcemod) Groups(ctx context.Context) ([]Groups, error) {
	return h.repository.Groups(ctx)
}

func (h Sourcemod) GetGroupByID(ctx context.Context, groupID int) (Groups, error) {
	return h.repository.GetGroupByID(ctx, groupID)
}

func (h Sourcemod) SaveGroup(ctx context.Context, group Groups) (Groups, error) {
	if group.Name == "" {
		return Groups{}, ErrGroupName
	}

	if group.ImmunityLevel > 100 || group.ImmunityLevel < 0 {
		return Groups{}, ErrImmunity
	}

	for _, flag := range group.Flags {
		if !strings.ContainsRune(validFlags, flag) {
			return Groups{}, ErrAdminFlagInvalid
		}
	}

	return h.repository.SaveGroup(ctx, group)
}
