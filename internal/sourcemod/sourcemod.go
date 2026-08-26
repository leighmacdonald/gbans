package sourcemod

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/ban/bantype"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/rpc"
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
	ErrAdminNameExists  = errors.New("an admin group with this name already exists")
	ErrGetPerson        = errors.New("failed to fetch person result")
)

type Config struct {
	sync.RWMutex

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
	SteamID    steamid.SteamID
	BanSource  BanSource
	BanID      int
	BanType    bantype.Type
	Reason     reason.Reason
	EvadeOK    bool
	IP         netip.Addr
	ValidUntil time.Time
}

func (p PlayerBanState) Path() string {
	return fmt.Sprintf("/ban/%d", p.BanID)
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
	AdminID   int64
	SteamID   steamid.SteamID
	AuthType  AuthType // steam | name |ip
	Identity  string
	Password  string
	Flags     string
	Name      string
	Immunity  int32
	Groups    []Groups
	CreatedOn time.Time
	UpdatedOn time.Time
}

type Groups struct {
	GroupID       int32
	Flags         string
	Name          string
	ImmunityLevel int32
	CreatedOn     time.Time
	UpdatedOn     time.Time
}

type GroupImmunity struct {
	GroupImmunityID int32
	Group           Groups
	Other           Groups
	CreatedOn       time.Time
}

type GroupOverrides struct {
	GroupOverrideID int32
	GroupID         int32
	Type            OverrideType // command | group
	Name            string
	Access          OverrideAccess // allow | deny
	CreatedOn       time.Time
	UpdatedOn       time.Time
}

type Overrides struct {
	OverrideID int32
	Type       OverrideType // command | group
	Name       string
	Flags      string
	CreatedOn  time.Time
	UpdatedOn  time.Time
}

type ConfigEntry struct {
	CfgKey   string
	CfgValue string
}

func New(repository Repository, person person.Provider, notifier notification.Notifier, seedChannelID string, modPingChannelID string, modRoleID string, servers *servers.Servers) Sourcemod {
	return Sourcemod{
		seedChannelID:    seedChannelID,
		modPingChannelID: modPingChannelID,
		modRoleID:        modRoleID,
		repository:       repository,
		person:           person,
		notifier:         notifier,
		servers:          servers,
		seedQueue: &SeedQueue{
			minTime: time.Second * 300,
			servers: make(map[int32]seedRequest),
			mu:      &sync.Mutex{},
		},
	}
}

type Sourcemod struct {
	seedChannelID    string
	modPingChannelID string
	modRoleID        string
	repository       Repository
	person           person.Provider
	seedQueue        *SeedQueue
	notifier         notification.Notifier
	servers          *servers.Servers
}

func (h Sourcemod) PingMod(_ context.Context, _ steamid.SteamID, name string, reason string, clientID int32, serverName string) error {
	h.notifier.Send(notification.NewDiscord(h.modPingChannelID, discord.NewMessage(
		discord.Heading("Mod Request [%s]", serverName),
		discord.BodyText(fmt.Sprintf("Mod requested by %s (%s) @<%s> (cid %d)", name, reason, h.modRoleID, clientID)))))

	return nil
}

type seedRequestView struct {
	Name        string
	Short       string
	Connect     string
	Link        string
	CC          string
	Roles       []string
	PlayerCount int32
	MaxPlayers  int32
}

func (h Sourcemod) seedRequest(roleIDs []string, server servers.SafeServer, userID string) bool {
	if !h.seedQueue.Allowed(server.ServerID, userID) {
		return false
	}

	if len(roleIDs) > 0 {
		content, errContent := discord.RenderTemplate("seed_req", seedRequestView{
			Name:        server.Name,
			Short:       server.NameShort,
			CC:          server.CC,
			Connect:     server.Addr(),
			Link:        server.Connect(),
			Roles:       roleIDs,
			PlayerCount: server.Players,
			MaxPlayers:  server.MaxPlayerDisplay(),
		})

		if errContent != nil {
			slog.Error("Failed to render content", slog.String("error", errContent.Error()))

			return false
		}

		go h.notifier.Send(notification.NewDiscord(h.seedChannelID, discord.NewMessage(
			discordgo.Container{
				AccentColor: new(discord.ColourSuccess),
				Components: []discordgo.MessageComponent{
					discordgo.TextDisplay{Content: content},
				},
			}, discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label: "Connect",
						Style: discordgo.LinkButton,
						URL:   server.Connect(),
					},
				},
			})))

		return true
	}

	slog.Error("No seed channel found", slog.String("server", server.NameShort))

	return false
}

func (h Sourcemod) GetBanState(ctx context.Context, steamID steamid.SteamID, ipAddr netip.Addr) (PlayerBanState, string, error) {
	const format = "Banned\nReason: %s (%s)\nUntil: %s\nAppeal: %s"

	banState, errBanState := h.repository.QueryBanState(ctx, steamID, ipAddr)
	if errBanState != nil || banState.BanID == 0 {
		return banState, "", errBanState
	}
	banState.IP = ipAddr

	var msg string
	validUntil := banState.ValidUntil.Format(time.ANSIC)
	if banState.ValidUntil.After(time.Now().AddDate(5, 0, 0)) {
		validUntil = "Permanent"
	}

	appealURL := link.Raw(fmt.Sprintf("/appeal/%d", banState.BanID))
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

func (h Sourcemod) Override(ctx context.Context, overrideID int32) (Overrides, error) {
	return h.repository.GetOverride(ctx, overrideID)
}

func (h Sourcemod) GroupImmunityByID(ctx context.Context, groupImmunityID int32) (GroupImmunity, error) {
	return h.repository.GetGroupImmunityByID(ctx, groupImmunityID)
}

func (h Sourcemod) GroupImmunities(ctx context.Context) ([]GroupImmunity, error) {
	return h.repository.GetGroupImmunities(ctx)
}

func (h Sourcemod) AddGroupImmunity(ctx context.Context, groupID int32, otherID int32) (GroupImmunity, error) {
	if groupID == otherID {
		return GroupImmunity{}, rpc.ErrBadRequest // TODO fix error
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

func (h Sourcemod) DelGroupImmunity(ctx context.Context, groupImmunityID int32) error {
	immunity, errImmunity := h.GroupImmunityByID(ctx, groupImmunityID)
	if errImmunity != nil {
		return errImmunity
	}

	if err := h.repository.DelGroupImmunity(ctx, immunity); err != nil {
		return err
	}

	slog.Info("Deleted group immunity", slog.Int("group_immunity_id", int(immunity.GroupImmunityID)))

	return nil
}

func (h Sourcemod) AddGroupOverride(ctx context.Context, groupID int32, name string, overrideType OverrideType, access OverrideAccess) (GroupOverrides, error) {
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

	slog.Info("Added group override", slog.Int("group_id", int(groupID)), slog.String("name", name))

	return override, nil
}

func (h Sourcemod) DelGroupOverride(ctx context.Context, groupOverrideID int32) error {
	override, errOverride := h.GroupOverride(ctx, groupOverrideID)
	if errOverride != nil {
		return errOverride
	}

	return h.repository.DelGroupOverride(ctx, override)
}

func (h Sourcemod) GroupOverride(ctx context.Context, groupOverrideID int32) (GroupOverrides, error) {
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

func (h Sourcemod) GroupOverrides(ctx context.Context, groupID int32) ([]GroupOverrides, error) {
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

func (h Sourcemod) DelOverride(ctx context.Context, overrideID int32) error {
	override, errOverride := h.repository.GetOverride(ctx, overrideID)
	if errOverride != nil {
		return errOverride
	}

	return h.repository.DelOverride(ctx, override)
}

func (h Sourcemod) DelAdminGroup(ctx context.Context, adminID int64, groupID int32) (Admin, error) {
	admin, errAdmin := h.AdminByID(ctx, adminID)
	if errAdmin != nil {
		return Admin{}, errAdmin
	}

	group, errGroup := h.GetGroupByID(ctx, groupID)
	if errGroup != nil {
		return Admin{}, errGroup
	}

	if !slices.Contains(admin.Groups, group) {
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

func (h Sourcemod) AddAdminGroup(ctx context.Context, adminID int64, groupID int32) (Admin, error) {
	admin, errAdmin := h.AdminByID(ctx, adminID)
	if errAdmin != nil {
		return Admin{}, errAdmin
	}

	group, errGroup := h.GetGroupByID(ctx, groupID)
	if errGroup != nil {
		return Admin{}, errGroup
	}

	if slices.Contains(admin.Groups, group) {
		return admin, ErrAdminGroupExists
	}

	if err := h.repository.InsertAdminGroup(ctx, admin, group); err != nil {
		return Admin{}, err
	}

	admin.Groups = append(admin.Groups, group)

	return admin, nil
}

func (h Sourcemod) DelGroup(ctx context.Context, groupID int32) error {
	group, errGroup := h.repository.GetGroupByID(ctx, groupID)
	if errGroup != nil {
		return errGroup
	}

	return h.repository.DeleteGroup(ctx, group)
}

const validFlags = "zabcdefghijklmnopqrst"

func (h Sourcemod) AddGroup(ctx context.Context, name string, flags string, immunityLevel int32) (Groups, error) {
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

func (h Sourcemod) DelAdmin(ctx context.Context, adminID int64) error {
	admin, errAdmin := h.AdminByID(ctx, adminID)
	if errAdmin != nil {
		return errAdmin
	}

	return h.repository.DelAdmin(ctx, admin)
}

func (h Sourcemod) AdminByID(ctx context.Context, adminID int64) (Admin, error) {
	return h.repository.GetAdminByID(ctx, adminID)
}

func (h Sourcemod) SaveAdmin(ctx context.Context, admin Admin) (Admin, error) {
	if admin.AuthType != AuthTypeSteam {
		return Admin{}, ErrInvalidAuthName
	}

	if admin.Immunity < 0 || admin.Immunity > 100 {
		return Admin{}, ErrImmunity
	}

	steamID, errSteamID := steamid.Resolve(ctx, admin.Identity)
	if errSteamID != nil || !steamID.Valid() {
		return Admin{}, steamid.ErrDecodeSID
	}

	if err := h.person.EnsurePerson(ctx, steamID); err != nil {
		return Admin{}, ErrGetPerson
	}

	admin.SteamID = steamID
	admin.Identity = string(steamID.Steam3())
	admin.Password = ""

	return h.repository.SaveAdmin(ctx, admin)
}

func (h Sourcemod) AddAdmin(ctx context.Context, alias string, authType AuthType, identity string, flags string, immunity int32, _ string) (Admin, error) {
	if authType != AuthTypeSteam {
		return Admin{}, ErrInvalidAuthName
	}

	if immunity < 0 || immunity > 100 {
		return Admin{}, ErrImmunity
	}

	steamID, errSteamID := steamid.Resolve(ctx, identity)
	if errSteamID != nil || !steamID.Valid() {
		return Admin{}, steamid.ErrDecodeSID
	}

	if _, errAdmin := h.AdminBySteamID(ctx, steamID); errAdmin == nil {
		return Admin{}, ErrAdminExists
	}

	if err := h.person.EnsurePerson(ctx, steamID); err != nil {
		return Admin{}, ErrGetPerson
	}

	return h.repository.AddAdmin(ctx, Admin{
		SteamID:  steamID,
		AuthType: AuthTypeSteam,
		Identity: string(steamID.Steam3()),
		Flags:    flags,
		Name:     alias,
		Immunity: immunity,
		Groups:   []Groups{},
	})
}

func (h Sourcemod) Admins(ctx context.Context) ([]Admin, error) {
	return h.repository.Admins(ctx)
}

func (h Sourcemod) AdminBySteamID(ctx context.Context, steamID steamid.SteamID) (Admin, error) {
	return h.repository.GetAdminByID(ctx, steamID.Int64())
}

func (h Sourcemod) Groups(ctx context.Context) ([]Groups, error) {
	return h.repository.Groups(ctx)
}

func (h Sourcemod) GetGroupByID(ctx context.Context, groupID int32) (Groups, error) {
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
