package srcds

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"strings"
	"time"

	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/report"
	"github.com/leighmacdonald/gbans/internal/steam"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"golang.org/x/exp/slices"
)

type srcds struct {
	repository    SRCDSRepository
	bans          ban.BanUsecase
	config        domain.ConfigUsecase
	servers       domain.ServersUsecase
	persons       domain.PersonUsecase
	reports       report.ReportUsecase
	notifications domain.NotificationUsecase
	cookie        string
	tfAPI         *thirdparty.TFAPI
}

func NewSrcdsUsecase(repository SRCDSRepository, config domain.ConfigUsecase, servers domain.ServersUsecase,
	persons domain.PersonUsecase, reports report.ReportUsecase, notifications domain.NotificationUsecase, bans ban.BanUsecase,
	tfAPI *thirdparty.TFAPI,
) SRCDSUsecase {
	return &srcds{
		config:        config,
		servers:       servers,
		persons:       persons,
		reports:       reports,
		notifications: notifications,
		bans:          bans,
		repository:    repository,
		cookie:        config.Config().HTTPCookieKey,
		tfAPI:         tfAPI,
	}
}

func (h srcds) GetBanState(ctx context.Context, steamID steamid.SteamID, ip netip.Addr) (PlayerBanState, string, error) {
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

func (h srcds) GetOverride(ctx context.Context, overrideID int) (SMOverrides, error) {
	return h.repository.GetOverride(ctx, overrideID)
}

func (h srcds) GetGroupImmunityByID(ctx context.Context, groupImmunityID int) (SMGroupImmunity, error) {
	return h.repository.GetGroupImmunityByID(ctx, groupImmunityID)
}

func (h srcds) GetGroupImmunities(ctx context.Context) ([]SMGroupImmunity, error) {
	return h.repository.GetGroupImmunities(ctx)
}

func (h srcds) AddGroupImmunity(ctx context.Context, groupID int, otherID int) (SMGroupImmunity, error) {
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

func (h srcds) DelGroupImmunity(ctx context.Context, groupImmunityID int) error {
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

func (h srcds) AddGroupOverride(ctx context.Context, groupID int, name string, overrideType OverrideType, access OverrideAccess) (SMGroupOverrides, error) {
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

func (h srcds) DelGroupOverride(ctx context.Context, groupOverrideID int) error {
	override, errOverride := h.GetGroupOverride(ctx, groupOverrideID)
	if errOverride != nil {
		return errOverride
	}

	return h.repository.DelGroupOverride(ctx, override)
}

func (h srcds) GetGroupOverride(ctx context.Context, groupOverrideID int) (SMGroupOverrides, error) {
	return h.repository.GetGroupOverride(ctx, groupOverrideID)
}

func (h srcds) SaveGroupOverride(ctx context.Context, override SMGroupOverrides) (SMGroupOverrides, error) {
	if override.Name == "" || override.Type == "" {
		return SMGroupOverrides{}, domain.ErrInvalidParameter
	}

	if override.Access != OverrideAccessAllow && override.Access != OverrideAccessDeny {
		return SMGroupOverrides{}, domain.ErrInvalidParameter
	}

	return h.repository.SaveGroupOverride(ctx, override)
}

func (h srcds) GroupOverrides(ctx context.Context, groupID int) ([]SMGroupOverrides, error) {
	group, errGroup := h.GetGroupByID(ctx, groupID)
	if errGroup != nil {
		return []SMGroupOverrides{}, errGroup
	}

	return h.repository.GroupOverrides(ctx, group)
}

func (h srcds) Overrides(ctx context.Context) ([]SMOverrides, error) {
	return h.repository.Overrides(ctx)
}

func (h srcds) SaveOverride(ctx context.Context, override SMOverrides) (SMOverrides, error) {
	if override.Name == "" || override.Flags == "" || override.Type != OverrideTypeCommand && override.Type != OverrideTypeGroup {
		return SMOverrides{}, domain.ErrInvalidParameter
	}

	return h.repository.SaveOverride(ctx, override)
}

func (h srcds) AddOverride(ctx context.Context, name string, overrideType OverrideType, flags string) (SMOverrides, error) {
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

func (h srcds) DelOverride(ctx context.Context, overrideID int) error {
	override, errOverride := h.repository.GetOverride(ctx, overrideID)
	if errOverride != nil {
		return errOverride
	}

	return h.repository.DelOverride(ctx, override)
}

func (h srcds) DelAdminGroup(ctx context.Context, adminID int, groupID int) (SMAdmin, error) {
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

func (h srcds) AddAdminGroup(ctx context.Context, adminID int, groupID int) (SMAdmin, error) {
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

func (h srcds) GetAdminGroups(ctx context.Context, admin SMAdmin) ([]SMGroups, error) {
	return h.repository.GetAdminGroups(ctx, admin)
}

func (h srcds) Report(ctx context.Context, currentUser domain.UserProfile, req RequestReportCreate) (ReportWithAuthor, error) {
	if req.Description == "" || len(req.Description) < 10 {
		return ReportWithAuthor{}, fmt.Errorf("%w: description", domain.ErrParamInvalid)
	}

	// ServerStore initiated requests will have a sourceID set by the server
	// Web based reports the source should not be set, the reporter will be taken from the
	// current session information instead
	if !req.SourceID.Valid() {
		req.SourceID = currentUser.SteamID
	}

	if !req.SourceID.Valid() {
		return ReportWithAuthor{}, domain.ErrSourceID
	}

	if !req.TargetID.Valid() {
		return ReportWithAuthor{}, domain.ErrTargetID
	}

	if req.SourceID.Int64() == req.TargetID.Int64() {
		return ReportWithAuthor{}, domain.ErrSelfReport
	}

	personSource, errCreateSource := h.persons.GetPersonBySteamID(ctx, nil, req.SourceID)
	if errCreateSource != nil {
		return ReportWithAuthor{}, errCreateSource
	}

	personTarget, errCreateTarget := h.persons.GetOrCreatePersonBySteamID(ctx, nil, req.TargetID)
	if errCreateTarget != nil {
		return ReportWithAuthor{}, errCreateTarget
	}

	if personTarget.Expired() {
		if err := steam.UpdatePlayerSummary(ctx, &personTarget, h.tfAPI); err != nil {
			slog.Error("Failed to update target player", log.ErrAttr(err))
		} else {
			if errSave := h.persons.SavePerson(ctx, nil, &personTarget); errSave != nil {
				slog.Error("Failed to save target player update", log.ErrAttr(err))
			}
		}
	}

	// Ensure the user doesn't already have an open report against the user
	existing, errReports := h.reports.GetReportBySteamID(ctx, personSource.SteamID, req.TargetID)
	if errReports != nil {
		if !errors.Is(errReports, database.ErrNoResult) {
			return ReportWithAuthor{}, errReports
		}
	}

	if existing.ReportID > 0 {
		return ReportWithAuthor{}, domain.ErrReportExists
	}

	savedReport, errReportSave := h.reports.SaveReport(ctx, currentUser, req)
	if errReportSave != nil {
		return ReportWithAuthor{}, errReportSave
	}

	conf := h.config.Config()

	demoURL := ""

	h.notifications.Enqueue(ctx, domain.NewDiscordNotification(
		domain.ChannelModLog,
		discord.NewInGameReportResponse(savedReport, conf.ExtURL(savedReport), currentUser, conf.ExtURL(currentUser), demoURL)))

	return savedReport, nil
}

func (h srcds) SetAdminGroups(ctx context.Context, authType domain.AuthType, identity string, groups ...domain.SMGroups) error {
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

func (h srcds) DelGroup(ctx context.Context, groupID int) error {
	group, errGroup := h.repository.GetGroupByID(ctx, groupID)
	if errGroup != nil {
		return errGroup
	}

	return h.repository.DeleteGroup(ctx, group)
}

const validFlags = "zabcdefghijklmnopqrst"

func (h srcds) AddGroup(ctx context.Context, name string, flags string, immunityLevel int) (domain.SMGroups, error) {
	if name == "" {
		return domain.SMGroups{}, domain.ErrSMGroupName
	}

	if immunityLevel > 100 || immunityLevel < 0 {
		return domain.SMGroups{}, domain.ErrSMImmunity
	}

	for _, flag := range flags {
		if !strings.ContainsRune(validFlags, flag) {
			return domain.SMGroups{}, domain.ErrSMAdminFlagInvalid
		}
	}

	return h.repository.AddGroup(ctx, domain.SMGroups{
		Flags:         flags,
		Name:          name,
		ImmunityLevel: immunityLevel,
	})
}

func validateAuthIdentity(ctx context.Context, authType domain.AuthType, identity string, password string) (string, error) {
	switch authType {
	case domain.AuthTypeSteam:
		steamID, errSteamID := steamid.Resolve(ctx, identity)
		if errSteamID != nil {
			return "", domain.ErrInvalidSID
		}

		identity = steamID.String()
	case domain.AuthTypeIP:
		if ip := net.ParseIP(identity); ip == nil || ip.To4() != nil {
			return "", domain.ErrInvalidIP
		}
	case domain.AuthTypeName:
		if identity == "" {
			return "", domain.ErrSMInvalidAuthName
		}

		if password == "" {
			return "", domain.ErrSMRequirePassword
		}
	}

	return identity, nil
}

func (h srcds) DelAdmin(ctx context.Context, adminID int) error {
	admin, errAdmin := h.repository.GetAdminByID(ctx, adminID)
	if errAdmin != nil {
		return errAdmin
	}

	return h.repository.DelAdmin(ctx, admin)
}

func (h srcds) GetAdminByID(ctx context.Context, adminID int) (domain.SMAdmin, error) {
	return h.repository.GetAdminByID(ctx, adminID)
}

func (h srcds) SaveAdmin(ctx context.Context, admin domain.SMAdmin) (domain.SMAdmin, error) {
	realIdentity, errValidate := validateAuthIdentity(ctx, admin.AuthType, admin.Identity, admin.Password)
	if errValidate != nil {
		return domain.SMAdmin{}, errValidate
	}

	if admin.Immunity < 0 || admin.Immunity > 100 {
		return domain.SMAdmin{}, domain.ErrSMImmunity
	}

	var steamID steamid.SteamID
	if admin.AuthType == domain.AuthTypeSteam {
		steamID = steamid.New(realIdentity)
		if _, err := h.persons.GetOrCreatePersonBySteamID(ctx, nil, steamID); err != nil {
			return domain.SMAdmin{}, domain.ErrGetPerson
		}

		admin.Identity = string(steamID.Steam3())
		admin.SteamID = steamID
	}

	return h.repository.SaveAdmin(ctx, admin)
}

func (h srcds) AddAdmin(ctx context.Context, alias string, authType domain.AuthType, identity string, flags string, immunity int, password string) (domain.SMAdmin, error) {
	realIdentity, errValidate := validateAuthIdentity(ctx, authType, identity, password)
	if errValidate != nil {
		return domain.SMAdmin{}, errValidate
	}

	if immunity < 0 || immunity > 100 {
		return domain.SMAdmin{}, domain.ErrSMImmunity
	}

	admin, errAdmin := h.repository.GetAdminByIdentity(ctx, authType, realIdentity)
	if errAdmin != nil && !errors.Is(errAdmin, database.ErrNoResult) {
		return domain.SMAdmin{}, errAdmin
	}

	if errAdmin == nil {
		return admin, domain.ErrSMAdminExists
	}

	var steamID steamid.SteamID
	if authType == domain.AuthTypeSteam {
		steamID = steamid.New(realIdentity)
		if _, err := h.persons.GetOrCreatePersonBySteamID(ctx, nil, steamID); err != nil {
			return domain.SMAdmin{}, domain.ErrGetPerson
		}

		identity = string(steamID.Steam3())
	}

	return h.repository.AddAdmin(ctx, domain.SMAdmin{
		SteamID:  steamID,
		AuthType: authType,
		Identity: identity,
		Password: password,
		Flags:    flags,
		Name:     alias,
		Immunity: immunity,
		Groups:   []domain.SMGroups{},
	})
}

func (h srcds) Admins(ctx context.Context) ([]domain.SMAdmin, error) {
	return h.repository.Admins(ctx)
}

func (h srcds) Groups(ctx context.Context) ([]domain.SMGroups, error) {
	return h.repository.Groups(ctx)
}

func (h srcds) GetGroupByID(ctx context.Context, groupID int) (domain.SMGroups, error) {
	return h.repository.GetGroupByID(ctx, groupID)
}

func (h srcds) SaveGroup(ctx context.Context, group domain.SMGroups) (domain.SMGroups, error) {
	if group.Name == "" {
		return domain.SMGroups{}, domain.ErrSMGroupName
	}

	if group.ImmunityLevel > 100 || group.ImmunityLevel < 0 {
		return domain.SMGroups{}, domain.ErrSMImmunity
	}

	for _, flag := range group.Flags {
		if !strings.ContainsRune(validFlags, flag) {
			return domain.SMGroups{}, domain.ErrSMAdminFlagInvalid
		}
	}

	return h.repository.SaveGroup(ctx, group)
}
