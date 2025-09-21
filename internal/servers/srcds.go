package servers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/domain/ban"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"golang.org/x/exp/slices"
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
	BanType    ban.Type        `json:"ban_type"`
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

func MiddlewareServer(servers Servers, sentryDSN string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reqAuthHeader := ctx.GetHeader("Authorization")
		if reqAuthHeader == "" {
			ctx.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		if strings.HasPrefix(reqAuthHeader, "Bearer ") {
			parts := strings.Split(reqAuthHeader, " ")
			if len(parts) != 2 {
				ctx.AbortWithStatus(http.StatusUnauthorized)

				return
			}

			reqAuthHeader = parts[1]
		}

		var server Server
		if errServer := servers.GetByPassword(ctx, reqAuthHeader, &server, false, false); errServer != nil {
			slog.Error("Failed to load server during auth", log.ErrAttr(errServer), slog.String("token", reqAuthHeader), slog.String("IP", ctx.ClientIP()))
			ctx.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		ctx.Set("server_id", server.ServerID)

		if sentryDSN != "" {
			if hub := sentrygin.GetHubFromContext(ctx); hub != nil {
				hub.WithScope(func(scope *sentry.Scope) {
					scope.SetUser(sentry.User{
						ID:        strconv.Itoa(server.ServerID),
						IPAddress: server.Addr(),
						Name:      server.ShortName,
					})
				})
			}
		}

		ctx.Next()
	}
}

type SRCDS struct {
	repository SRCDSRepository
	config     *config.Configuration
	servers    Servers
	persons    PersonProvider
	notif      notification.Notifications
	cookie     string
	tfAPI      *thirdparty.TFAPI
}

func NewSRCDS(repository SRCDSRepository, config *config.Configuration, servers Servers,
	persons PersonProvider, tfAPI *thirdparty.TFAPI, notif notification.Notifications,
) *SRCDS {
	return &SRCDS{
		config:     config,
		servers:    servers,
		persons:    persons,
		repository: repository,
		cookie:     config.Config().HTTPCookieKey,
		tfAPI:      tfAPI,
		notif:      notif,
	}
}

func (h SRCDS) GetBanState(ctx context.Context, steamID steamid.SteamID, ip netip.Addr) (PlayerBanState, string, error) {
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
		appealURL = h.config.ExtURLRaw("/appeal/%d", banState.BanID)
	}

	if banState.BanID > 0 && banState.BanType >= ban.NoComm {
		switch banState.BanSource {
		case BanSourceSteam:
			if banState.BanType == ban.NoComm {
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

func (h SRCDS) GetOverride(ctx context.Context, overrideID int) (SMOverrides, error) {
	return h.repository.GetOverride(ctx, overrideID)
}

func (h SRCDS) GetGroupImmunityByID(ctx context.Context, groupImmunityID int) (SMGroupImmunity, error) {
	return h.repository.GetGroupImmunityByID(ctx, groupImmunityID)
}

func (h SRCDS) GetGroupImmunities(ctx context.Context) ([]SMGroupImmunity, error) {
	return h.repository.GetGroupImmunities(ctx)
}

func (h SRCDS) AddGroupImmunity(ctx context.Context, groupID int, otherID int) (SMGroupImmunity, error) {
	if groupID == otherID {
		return SMGroupImmunity{}, httphelper.ErrBadRequest // TODO fix error
	}

	group, errGroup := h.GetGroupByID(ctx, groupID)
	if errGroup != nil {
		return SMGroupImmunity{}, errGroup
	}

	other, errOther := h.GetGroupByID(ctx, otherID)
	if errOther != nil {
		return SMGroupImmunity{}, errOther
	}

	return h.repository.AddGroupImmunity(ctx, group, other)
}

func (h SRCDS) DelGroupImmunity(ctx context.Context, groupImmunityID int) error {
	immunity, errImmunity := h.GetGroupImmunityByID(ctx, groupImmunityID)
	if errImmunity != nil {
		return errImmunity
	}

	if err := h.repository.DelGroupImmunity(ctx, immunity); err != nil {
		return err
	}

	slog.Info("Deleted group immunity", slog.Int("group_immunity_id", immunity.GroupImmunityID))

	return nil
}

func (h SRCDS) AddGroupOverride(ctx context.Context, groupID int, name string, overrideType OverrideType, access OverrideAccess) (SMGroupOverrides, error) {
	if name == "" || overrideType == "" {
		return SMGroupOverrides{}, domain.ErrInvalidParameter
	}

	if access != OverrideAccessAllow && access != OverrideAccessDeny {
		return SMGroupOverrides{}, domain.ErrInvalidParameter
	}

	now := time.Now()

	override, err := h.repository.AddGroupOverride(ctx, SMGroupOverrides{
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

	slog.Info("Added group override", log.ErrAttr(err), slog.Int("group_id", groupID), slog.String("name", name))

	return override, nil
}

func (h SRCDS) DelGroupOverride(ctx context.Context, groupOverrideID int) error {
	override, errOverride := h.GetGroupOverride(ctx, groupOverrideID)
	if errOverride != nil {
		return errOverride
	}

	return h.repository.DelGroupOverride(ctx, override)
}

func (h SRCDS) GetGroupOverride(ctx context.Context, groupOverrideID int) (SMGroupOverrides, error) {
	return h.repository.GetGroupOverride(ctx, groupOverrideID)
}

func (h SRCDS) SaveGroupOverride(ctx context.Context, override SMGroupOverrides) (SMGroupOverrides, error) {
	if override.Name == "" || override.Type == "" {
		return SMGroupOverrides{}, domain.ErrInvalidParameter
	}

	if override.Access != OverrideAccessAllow && override.Access != OverrideAccessDeny {
		return SMGroupOverrides{}, domain.ErrInvalidParameter
	}

	return h.repository.SaveGroupOverride(ctx, override)
}

func (h SRCDS) GroupOverrides(ctx context.Context, groupID int) ([]SMGroupOverrides, error) {
	group, errGroup := h.GetGroupByID(ctx, groupID)
	if errGroup != nil {
		return []SMGroupOverrides{}, errGroup
	}

	return h.repository.GroupOverrides(ctx, group)
}

func (h SRCDS) Overrides(ctx context.Context) ([]SMOverrides, error) {
	return h.repository.Overrides(ctx)
}

func (h SRCDS) SaveOverride(ctx context.Context, override SMOverrides) (SMOverrides, error) {
	if override.Name == "" || override.Flags == "" || override.Type != OverrideTypeCommand && override.Type != OverrideTypeGroup {
		return SMOverrides{}, domain.ErrInvalidParameter
	}

	return h.repository.SaveOverride(ctx, override)
}

func (h SRCDS) AddOverride(ctx context.Context, name string, overrideType OverrideType, flags string) (SMOverrides, error) {
	if name == "" || flags == "" || overrideType != OverrideTypeCommand && overrideType != OverrideTypeGroup {
		return SMOverrides{}, domain.ErrInvalidParameter
	}

	now := time.Now()

	return h.repository.AddOverride(ctx, SMOverrides{
		Type:      overrideType,
		Name:      name,
		Flags:     flags,
		CreatedOn: now,
		UpdatedOn: now,
	})
}

func (h SRCDS) DelOverride(ctx context.Context, overrideID int) error {
	override, errOverride := h.repository.GetOverride(ctx, overrideID)
	if errOverride != nil {
		return errOverride
	}

	return h.repository.DelOverride(ctx, override)
}

func (h SRCDS) DelAdminGroup(ctx context.Context, adminID int, groupID int) (SMAdmin, error) {
	admin, errAdmin := h.GetAdminByID(ctx, adminID)
	if errAdmin != nil {
		return SMAdmin{}, errAdmin
	}

	group, errGroup := h.GetGroupByID(ctx, groupID)
	if errGroup != nil {
		return SMAdmin{}, errGroup
	}

	existing, errExisting := h.GetAdminGroups(ctx, admin)
	if errExisting != nil && !errors.Is(errExisting, database.ErrNoResult) {
		return admin, errExisting
	}

	if !slices.Contains(existing, group) {
		return admin, ErrSMAdminGroupExists
	}

	if err := h.repository.DeleteAdminGroup(ctx, admin, group); err != nil {
		return SMAdmin{}, err
	}

	admin.Groups = slices.DeleteFunc(admin.Groups, func(g SMGroups) bool {
		return g.GroupID == groupID
	})

	return admin, nil
}

func (h SRCDS) AddAdminGroup(ctx context.Context, adminID int, groupID int) (SMAdmin, error) {
	admin, errAdmin := h.GetAdminByID(ctx, adminID)
	if errAdmin != nil {
		return SMAdmin{}, errAdmin
	}

	group, errGroup := h.GetGroupByID(ctx, groupID)
	if errGroup != nil {
		return SMAdmin{}, errGroup
	}

	existing, errExisting := h.GetAdminGroups(ctx, admin)
	if errExisting != nil && !errors.Is(errExisting, database.ErrNoResult) {
		return admin, errExisting
	}

	if slices.Contains(existing, group) {
		return admin, ErrSMAdminGroupExists
	}

	if err := h.repository.InsertAdminGroup(ctx, admin, group, len(existing)+1); err != nil {
		return SMAdmin{}, err
	}

	admin.Groups = append(admin.Groups, group)

	return admin, nil
}

func (h SRCDS) GetAdminGroups(ctx context.Context, admin SMAdmin) ([]SMGroups, error) {
	return h.repository.GetAdminGroups(ctx, admin)
}

// func (h SRCDSUsecase) Report(ctx context.Context, currentUser domain.PersonInfo, req ban.RequestReportCreate) (ban.ReportWithAuthor, error) {
// 	if req.Description == "" || len(req.Description) < 10 {
// 		return ban.ReportWithAuthor{}, fmt.Errorf("%w: description", domain.ErrParamInvalid)
// 	}

// 	// ServerStore initiated requests will have a sourceID set by the server
// 	// Web based reports the source should not be set, the reporter will be taken from the
// 	// current session information instead
// 	if !req.SourceID.Valid() {
// 		req.SourceID = currentUser.GetSteamID()
// 	}

// 	if !req.SourceID.Valid() {
// 		return ban.ReportWithAuthor{}, domain.ErrSourceID
// 	}

// 	if !req.TargetID.Valid() {
// 		return ban.ReportWithAuthor{}, domain.ErrTargetID
// 	}

// 	if req.SourceID.Int64() == req.TargetID.Int64() {
// 		return ban.ReportWithAuthor{}, domain.ErrSelfReport
// 	}

// 	personSource, errCreateSource := h.persons.GetPersonBySteamID(ctx, nil, req.SourceID)
// 	if errCreateSource != nil {
// 		return ban.ReportWithAuthor{}, errCreateSource
// 	}

// 	personTarget, errCreateTarget := h.persons.GetOrCreatePersonBySteamID(ctx, nil, req.TargetID)
// 	if errCreateTarget != nil {
// 		return ban.ReportWithAuthor{}, errCreateTarget
// 	}

// 	if personTarget.Expired() {
// 		if err := person.UpdatePlayerSummary(ctx, &personTarget, h.tfAPI); err != nil {
// 			slog.Error("Failed to update target player", log.ErrAttr(err))
// 		} else {
// 			if errSave := h.persons.SavePerson(ctx, nil, &personTarget); errSave != nil {
// 				slog.Error("Failed to save target player update", log.ErrAttr(err))
// 			}
// 		}
// 	}

// 	// Ensure the user doesn't already have an open report against the user
// 	existing, errReports := h.reports.GetReportBySteamID(ctx, personSource.SteamID, req.TargetID)
// 	if errReports != nil {
// 		if !errors.Is(errReports, database.ErrNoResult) {
// 			return ban.ReportWithAuthor{}, errReports
// 		}
// 	}

// 	if existing.ReportID > 0 {
// 		return ban.ReportWithAuthor{}, domain.ErrReportExists
// 	}

// 	savedReport, errReportSave := h.reports.SaveReport(ctx, currentUser, req)
// 	if errReportSave != nil {
// 		return ban.ReportWithAuthor{}, errReportSave
// 	}

// 	// conf := h.config.Config()
// 	//
// 	// demoURL := ""
// 	//
// 	// h.notifications.Enqueue(ctx, notification.NewDiscordNotification(
// 	// 	discord.ChannelModLog,
// 	// 	discord.NewInGameReportResponse(savedReport, conf.ExtURL(savedReport), currentUser, conf.ExtURL(currentUser), demoURL)))

// 	return savedReport, nil
// }

func (h SRCDS) SetAdminGroups(ctx context.Context, authType AuthType, identity string, groups ...SMGroups) error {
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

func (h SRCDS) DelGroup(ctx context.Context, groupID int) error {
	group, errGroup := h.repository.GetGroupByID(ctx, groupID)
	if errGroup != nil {
		return errGroup
	}

	return h.repository.DeleteGroup(ctx, group)
}

const validFlags = "zabcdefghijklmnopqrst"

func (h SRCDS) AddGroup(ctx context.Context, name string, flags string, immunityLevel int) (SMGroups, error) {
	if name == "" {
		return SMGroups{}, ErrSMGroupName
	}

	if immunityLevel > 100 || immunityLevel < 0 {
		return SMGroups{}, ErrSMImmunity
	}

	for _, flag := range flags {
		if !strings.ContainsRune(validFlags, flag) {
			return SMGroups{}, ErrSMAdminFlagInvalid
		}
	}

	return h.repository.AddGroup(ctx, SMGroups{
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
			return "", domain.ErrInvalidSID
		}

		identity = steamID.String()
	case AuthTypeIP:
		if ip := net.ParseIP(identity); ip == nil || ip.To4() != nil {
			return "", domain.ErrInvalidIP
		}
	case AuthTypeName:
		if identity == "" {
			return "", ErrSMInvalidAuthName
		}

		if password == "" {
			return "", ErrSMRequirePassword
		}
	}

	return identity, nil
}

func (h SRCDS) DelAdmin(ctx context.Context, adminID int) error {
	admin, errAdmin := h.repository.GetAdminByID(ctx, adminID)
	if errAdmin != nil {
		return errAdmin
	}

	return h.repository.DelAdmin(ctx, admin)
}

func (h SRCDS) GetAdminByID(ctx context.Context, adminID int) (SMAdmin, error) {
	return h.repository.GetAdminByID(ctx, adminID)
}

func (h SRCDS) SaveAdmin(ctx context.Context, admin SMAdmin) (SMAdmin, error) {
	realIdentity, errValidate := validateAuthIdentity(ctx, admin.AuthType, admin.Identity, admin.Password)
	if errValidate != nil {
		return SMAdmin{}, errValidate
	}

	if admin.Immunity < 0 || admin.Immunity > 100 {
		return SMAdmin{}, ErrSMImmunity
	}

	var steamID steamid.SteamID
	if admin.AuthType == AuthTypeSteam {
		steamID = steamid.New(realIdentity)
		if _, err := h.persons.GetOrCreatePersonBySteamID(ctx, nil, steamID); err != nil {
			return SMAdmin{}, domain.ErrGetPerson
		}

		admin.Identity = string(steamID.Steam3())
		admin.SteamID = steamID
	}

	return h.repository.SaveAdmin(ctx, admin)
}

func (h SRCDS) AddAdmin(ctx context.Context, alias string, authType AuthType, identity string, flags string, immunity int, password string) (SMAdmin, error) {
	realIdentity, errValidate := validateAuthIdentity(ctx, authType, identity, password)
	if errValidate != nil {
		return SMAdmin{}, errValidate
	}

	if immunity < 0 || immunity > 100 {
		return SMAdmin{}, ErrSMImmunity
	}

	admin, errAdmin := h.repository.GetAdminByIdentity(ctx, authType, realIdentity)
	if errAdmin != nil && !errors.Is(errAdmin, database.ErrNoResult) {
		return SMAdmin{}, errAdmin
	}

	if errAdmin == nil {
		return admin, ErrSMAdminExists
	}

	var steamID steamid.SteamID
	if authType == AuthTypeSteam {
		steamID = steamid.New(realIdentity)
		if _, err := h.persons.GetOrCreatePersonBySteamID(ctx, nil, steamID); err != nil {
			return SMAdmin{}, domain.ErrGetPerson
		}

		identity = string(steamID.Steam3())
	}

	return h.repository.AddAdmin(ctx, SMAdmin{
		SteamID:  steamID,
		AuthType: authType,
		Identity: identity,
		Password: password,
		Flags:    flags,
		Name:     alias,
		Immunity: immunity,
		Groups:   []SMGroups{},
	})
}

func (h SRCDS) Admins(ctx context.Context) ([]SMAdmin, error) {
	return h.repository.Admins(ctx)
}

func (h SRCDS) Groups(ctx context.Context) ([]SMGroups, error) {
	return h.repository.Groups(ctx)
}

func (h SRCDS) GetGroupByID(ctx context.Context, groupID int) (SMGroups, error) {
	return h.repository.GetGroupByID(ctx, groupID)
}

func (h SRCDS) SaveGroup(ctx context.Context, group SMGroups) (SMGroups, error) {
	if group.Name == "" {
		return SMGroups{}, ErrSMGroupName
	}

	if group.ImmunityLevel > 100 || group.ImmunityLevel < 0 {
		return SMGroups{}, ErrSMImmunity
	}

	for _, flag := range group.Flags {
		if !strings.ContainsRune(validFlags, flag) {
			return SMGroups{}, ErrSMAdminFlagInvalid
		}
	}

	return h.repository.SaveGroup(ctx, group)
}
