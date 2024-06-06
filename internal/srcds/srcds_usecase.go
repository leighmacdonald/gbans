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

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"golang.org/x/exp/slices"
)

type SRCDS struct {
	banSteam domain.BanSteamUsecase
	config   domain.ConfigUsecase
	servers  domain.ServersUsecase
	srcds    domain.SRCDSRepository
	person   domain.PersonUsecase
	report   domain.ReportUsecase
	discord  domain.DiscordUsecase
	cookie   string
}

func NewSRCDS(srcds domain.SRCDSRepository, config domain.ConfigUsecase, servers domain.ServersUsecase,
	person domain.PersonUsecase, report domain.ReportUsecase, discord domain.DiscordUsecase, banSteam domain.BanSteamUsecase,
) domain.SRCDSUsecase {
	return &SRCDS{
		config:   config,
		servers:  servers,
		person:   person,
		report:   report,
		discord:  discord,
		banSteam: banSteam,
		srcds:    srcds,
		cookie:   config.Config().HTTPCookieKey,
	}
}

func (h SRCDS) GetBanState(ctx context.Context, steamID steamid.SteamID, ip netip.Addr) (domain.PlayerBanState, string, error) {
	banState, errBanState := h.srcds.QueryBanState(ctx, steamID, ip)
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
	if banState.BanSource == domain.BanSourceSteam {
		appealURL = h.config.ExtURLRaw("/appeal/%d", banState.BanID)
	}

	if banState.BanID > 0 && banState.BanType >= domain.NoComm {
		switch banState.BanSource {
		case domain.BanSourceSteam:
			if banState.BanType == domain.NoComm {
				msg = fmt.Sprintf("You are muted & gagged. Expires: %s. Appeal: %s", banState.ValidUntil.Format(time.DateTime), appealURL)
			} else {
				msg = fmt.Sprintf(format, banState.Reason.String(), "Steam", validUntil, appealURL)
			}
		case domain.BanSourceASN:
			msg = fmt.Sprintf(format, banState.Reason.String(), "Special", "Permanent", appealURL)
		case domain.BanSourceCIDR:
			msg = "Blocked Network/VPN\nPlease disable your VPN"
		case domain.BanSourceSteamFriend:
			msg = "Friend Network Ban"
		case domain.BanSourceSteamGroup:
			msg = "Blocked Steam Group"
		case domain.BanSourceSteamNet:
			msg = fmt.Sprintf(format, banState.Reason.String(), "Special", "Permanent", appealURL)
		}
	}

	return banState, msg, nil
}

func (h SRCDS) GetOverride(ctx context.Context, overrideID int) (domain.SMOverrides, error) {
	return h.srcds.GetOverride(ctx, overrideID)
}

func (h SRCDS) GetGroupImmunityByID(ctx context.Context, groupImmunityID int) (domain.SMGroupImmunity, error) {
	return h.srcds.GetGroupImmunityByID(ctx, groupImmunityID)
}

func (h SRCDS) GetGroupImmunities(ctx context.Context) ([]domain.SMGroupImmunity, error) {
	return h.srcds.GetGroupImmunities(ctx)
}

func (h SRCDS) AddGroupImmunity(ctx context.Context, groupID int, otherID int) (domain.SMGroupImmunity, error) {
	if groupID == otherID {
		return domain.SMGroupImmunity{}, domain.ErrBadRequest
	}

	group, errGroup := h.GetGroupByID(ctx, groupID)
	if errGroup != nil {
		return domain.SMGroupImmunity{}, errGroup
	}

	other, errOther := h.GetGroupByID(ctx, otherID)
	if errOther != nil {
		return domain.SMGroupImmunity{}, errOther
	}

	return h.srcds.AddGroupImmunity(ctx, group, other)
}

func (h SRCDS) DelGroupImmunity(ctx context.Context, groupImmunityID int) error {
	immunity, errImmunity := h.GetGroupImmunityByID(ctx, groupImmunityID)
	if errImmunity != nil {
		return errImmunity
	}

	return h.srcds.DelGroupImmunity(ctx, immunity)
}

func (h SRCDS) AddGroupOverride(ctx context.Context, groupID int, name string, overrideType domain.OverrideType, access domain.OverrideAccess) (domain.SMGroupOverrides, error) {
	if name == "" || overrideType == "" {
		return domain.SMGroupOverrides{}, domain.ErrInvalidParameter
	}

	if access != domain.OverrideAccessAllow && access != domain.OverrideAccessDeny {
		return domain.SMGroupOverrides{}, domain.ErrInvalidParameter
	}

	now := time.Now()

	return h.srcds.AddGroupOverride(ctx, domain.SMGroupOverrides{
		GroupID: groupID,
		Type:    overrideType,
		Name:    name,
		Access:  access,
		TimeStamped: domain.TimeStamped{
			CreatedOn: now,
			UpdatedOn: now,
		},
	})
}

func (h SRCDS) DelGroupOverride(ctx context.Context, groupOverrideID int) error {
	override, errOverride := h.GetGroupOverride(ctx, groupOverrideID)
	if errOverride != nil {
		return errOverride
	}

	return h.srcds.DelGroupOverride(ctx, override)
}

func (h SRCDS) GetGroupOverride(ctx context.Context, groupOverrideID int) (domain.SMGroupOverrides, error) {
	return h.srcds.GetGroupOverride(ctx, groupOverrideID)
}

func (h SRCDS) SaveGroupOverride(ctx context.Context, override domain.SMGroupOverrides) (domain.SMGroupOverrides, error) {
	if override.Name == "" || override.Type == "" {
		return domain.SMGroupOverrides{}, domain.ErrInvalidParameter
	}

	if override.Access != domain.OverrideAccessAllow && override.Access != domain.OverrideAccessDeny {
		return domain.SMGroupOverrides{}, domain.ErrInvalidParameter
	}

	return h.srcds.SaveGroupOverride(ctx, override)
}

func (h SRCDS) GroupOverrides(ctx context.Context, groupID int) ([]domain.SMGroupOverrides, error) {
	group, errGroup := h.GetGroupByID(ctx, groupID)
	if errGroup != nil {
		return []domain.SMGroupOverrides{}, errGroup
	}

	return h.srcds.GroupOverrides(ctx, group)
}

func (h SRCDS) Overrides(ctx context.Context) ([]domain.SMOverrides, error) {
	return h.srcds.Overrides(ctx)
}

func (h SRCDS) SaveOverride(ctx context.Context, override domain.SMOverrides) (domain.SMOverrides, error) {
	if override.Name == "" || override.Flags == "" || override.Type != domain.OverrideTypeCommand && override.Type != domain.OverrideTypeGroup {
		return domain.SMOverrides{}, domain.ErrInvalidParameter
	}

	return h.srcds.SaveOverride(ctx, override)
}

func (h SRCDS) AddOverride(ctx context.Context, name string, overrideType domain.OverrideType, flags string) (domain.SMOverrides, error) {
	if name == "" || flags == "" || overrideType != domain.OverrideTypeCommand && overrideType != domain.OverrideTypeGroup {
		return domain.SMOverrides{}, domain.ErrInvalidParameter
	}

	now := time.Now()

	return h.srcds.AddOverride(ctx, domain.SMOverrides{
		Type:  overrideType,
		Name:  name,
		Flags: flags,
		TimeStamped: domain.TimeStamped{
			CreatedOn: now,
			UpdatedOn: now,
		},
	})
}

func (h SRCDS) DelOverride(ctx context.Context, overrideID int) error {
	override, errOverride := h.srcds.GetOverride(ctx, overrideID)
	if errOverride != nil {
		return errOverride
	}

	return h.srcds.DelOverride(ctx, override)
}

func (h SRCDS) DelAdminGroup(ctx context.Context, adminID int, groupID int) (domain.SMAdmin, error) {
	admin, errAdmin := h.GetAdminByID(ctx, adminID)
	if errAdmin != nil {
		return domain.SMAdmin{}, errAdmin
	}

	group, errGroup := h.GetGroupByID(ctx, groupID)
	if errGroup != nil {
		return domain.SMAdmin{}, errGroup
	}

	existing, errExisting := h.GetAdminGroups(ctx, admin)
	if errExisting != nil && !errors.Is(errExisting, domain.ErrNoResult) {
		return admin, errExisting
	}

	if !slices.Contains(existing, group) {
		return admin, domain.ErrSMAdminGroupExists
	}

	if err := h.srcds.DeleteAdminGroup(ctx, admin, group); err != nil {
		return domain.SMAdmin{}, err
	}

	admin.Groups = slices.DeleteFunc(admin.Groups, func(g domain.SMGroups) bool {
		return g.GroupID == groupID
	})

	return admin, nil
}

func (h SRCDS) AddAdminGroup(ctx context.Context, adminID int, groupID int) (domain.SMAdmin, error) {
	admin, errAdmin := h.GetAdminByID(ctx, adminID)
	if errAdmin != nil {
		return domain.SMAdmin{}, errAdmin
	}

	group, errGroup := h.GetGroupByID(ctx, groupID)
	if errGroup != nil {
		return domain.SMAdmin{}, errGroup
	}

	existing, errExisting := h.GetAdminGroups(ctx, admin)
	if errExisting != nil && !errors.Is(errExisting, domain.ErrNoResult) {
		return admin, errExisting
	}

	if slices.Contains(existing, group) {
		return admin, domain.ErrSMAdminGroupExists
	}

	if err := h.srcds.InsertAdminGroup(ctx, admin, group, len(existing)+1); err != nil {
		return domain.SMAdmin{}, err
	}

	admin.Groups = append(admin.Groups, group)

	return admin, nil
}

func (h SRCDS) GetAdminGroups(ctx context.Context, admin domain.SMAdmin) ([]domain.SMGroups, error) {
	return h.srcds.GetAdminGroups(ctx, admin)
}

func (h SRCDS) Report(ctx context.Context, currentUser domain.UserProfile, req domain.CreateReportReq) (*domain.Report, error) {
	if req.Description == "" || len(req.Description) < 10 {
		return nil, fmt.Errorf("%w: description", domain.ErrParamInvalid)
	}

	// ServerStore initiated requests will have a sourceID set by the server
	// Web based reports the source should not be set, the reporter will be taken from the
	// current session information instead
	if !req.SourceID.Valid() {
		req.SourceID = currentUser.SteamID
	}

	if !req.SourceID.Valid() {
		return nil, domain.ErrSourceID
	}

	if !req.TargetID.Valid() {
		return nil, domain.ErrTargetID
	}

	if req.SourceID.Int64() == req.TargetID.Int64() {
		return nil, domain.ErrSelfReport
	}

	personSource, errCreatePerson := h.person.GetPersonBySteamID(ctx, req.SourceID)
	if errCreatePerson != nil {
		return nil, domain.ErrInternal
	}

	personTarget, errCreatePerson := h.person.GetOrCreatePersonBySteamID(ctx, req.TargetID)
	if errCreatePerson != nil {
		return nil, domain.ErrInternal
	}

	if personTarget.Expired() {
		if err := thirdparty.UpdatePlayerSummary(ctx, &personTarget); err != nil {
			slog.Error("Failed to update target player", log.ErrAttr(err))
		} else {
			if errSave := h.person.SavePerson(ctx, &personTarget); errSave != nil {
				slog.Error("Failed to save target player update", log.ErrAttr(err))
			}
		}
	}

	// Ensure the user doesn't already have an open report against the user
	existing, errReports := h.report.GetReportBySteamID(ctx, personSource.SteamID, req.TargetID)
	if errReports != nil {
		if !errors.Is(errReports, domain.ErrNoResult) {
			return nil, errReports
		}
	}

	if existing.ReportID > 0 {
		return nil, domain.ErrReportExists
	}

	// TODO encapsulate all operations in single tx
	report := domain.NewReport()
	report.SourceID = req.SourceID
	report.ReportStatus = domain.Opened
	report.Description = req.Description
	report.TargetID = req.TargetID
	report.Reason = req.Reason
	report.ReasonText = req.ReasonText
	report.DemoTick = req.DemoTick
	report.PersonMessageID = req.PersonMessageID

	if errReportSave := h.report.SaveReport(ctx, &report); errReportSave != nil {
		return nil, errReportSave
	}

	slog.Info("New report created successfully", slog.Int64("report_id", report.ReportID))

	conf := h.config.Config()

	demoURL := ""

	msg := discord.NewInGameReportResponse(report, conf.ExtURL(report), currentUser, conf.ExtURL(currentUser), demoURL)

	h.discord.SendPayload(domain.ChannelModLog, msg)

	return &report, nil
}

func (h SRCDS) SetAdminGroups(ctx context.Context, authType domain.AuthType, identity string, groups ...domain.SMGroups) error {
	admin, errAdmin := h.srcds.GetAdminByIdentity(ctx, authType, identity)
	if errAdmin != nil {
		return errAdmin
	}

	// Delete existing groups.
	if errDelete := h.srcds.DeleteAdminGroups(ctx, admin); errDelete != nil && !errors.Is(errDelete, domain.ErrNoResult) {
		return errDelete
	}

	// If no groups are given to add, this is treated purely as a delete function
	if len(groups) == 0 {
		return nil
	}

	for i := range groups {
		if errInsert := h.srcds.InsertAdminGroup(ctx, admin, groups[i], i); errInsert != nil {
			return errInsert
		}
	}

	return nil
}

func (h SRCDS) DelGroup(ctx context.Context, groupID int) error {
	group, errGroup := h.srcds.GetGroupByID(ctx, groupID)
	if errGroup != nil {
		return errGroup
	}

	return h.srcds.DeleteGroup(ctx, group)
}

const validFlags = "zabcdefghijklmnopqrst"

func (h SRCDS) AddGroup(ctx context.Context, name string, flags string, immunityLevel int) (domain.SMGroups, error) {
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

	return h.srcds.AddGroup(ctx, domain.SMGroups{
		Flags:         flags,
		Name:          name,
		ImmunityLevel: immunityLevel,
	})
}

func validateAuthIdentity(ctx context.Context, authType domain.AuthType, identity string, password string) (string, error) {
	switch {
	case authType == domain.AuthTypeSteam:
		steamID, errSteamID := steamid.Resolve(ctx, identity)
		if errSteamID != nil {
			return "", domain.ErrInvalidSID
		}

		identity = steamID.String()
	case authType == domain.AuthTypeIP:
		if ip := net.ParseIP(identity); ip == nil || ip.To4() != nil {
			return "", domain.ErrInvalidIP
		}
	case authType == domain.AuthTypeName:
		if identity == "" {
			return "", domain.ErrSMInvalidAuthName
		}

		if password == "" {
			return "", domain.ErrSMRequirePassword
		}
	}

	return identity, nil
}

func (h SRCDS) DelAdmin(ctx context.Context, adminID int) error {
	admin, errAdmin := h.srcds.GetAdminByID(ctx, adminID)
	if errAdmin != nil {
		return errAdmin
	}

	return h.srcds.DelAdmin(ctx, admin)
}

func (h SRCDS) GetAdminByID(ctx context.Context, adminID int) (domain.SMAdmin, error) {
	return h.srcds.GetAdminByID(ctx, adminID)
}

func (h SRCDS) SaveAdmin(ctx context.Context, admin domain.SMAdmin) (domain.SMAdmin, error) {
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
		if _, err := h.person.GetOrCreatePersonBySteamID(ctx, steamID); err != nil {
			return domain.SMAdmin{}, domain.ErrGetPerson
		}

		admin.Identity = string(steamID.Steam3())
		admin.SteamID = steamID
	}

	return h.srcds.SaveAdmin(ctx, admin)
}

func (h SRCDS) AddAdmin(ctx context.Context, alias string, authType domain.AuthType, identity string, flags string, immunity int, password string) (domain.SMAdmin, error) {
	realIdentity, errValidate := validateAuthIdentity(ctx, authType, identity, password)
	if errValidate != nil {
		return domain.SMAdmin{}, errValidate
	}

	if immunity < 0 || immunity > 100 {
		return domain.SMAdmin{}, domain.ErrSMImmunity
	}

	admin, errAdmin := h.srcds.GetAdminByIdentity(ctx, authType, realIdentity)
	if errAdmin != nil && !errors.Is(errAdmin, domain.ErrNoResult) {
		return domain.SMAdmin{}, errAdmin
	}

	if errAdmin == nil {
		return admin, domain.ErrSMAdminExists
	}

	var steamID steamid.SteamID
	if authType == domain.AuthTypeSteam {
		steamID = steamid.New(realIdentity)
		if _, err := h.person.GetOrCreatePersonBySteamID(ctx, steamID); err != nil {
			return domain.SMAdmin{}, domain.ErrGetPerson
		}

		identity = string(steamID.Steam3())
	}

	return h.srcds.AddAdmin(ctx, domain.SMAdmin{
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

func (h SRCDS) Admins(ctx context.Context) ([]domain.SMAdmin, error) {
	return h.srcds.Admins(ctx)
}

func (h SRCDS) Groups(ctx context.Context) ([]domain.SMGroups, error) {
	return h.srcds.Groups(ctx)
}

func (h SRCDS) GetGroupByID(ctx context.Context, groupID int) (domain.SMGroups, error) {
	return h.srcds.GetGroupByID(ctx, groupID)
}

func (h SRCDS) SaveGroup(ctx context.Context, group domain.SMGroups) (domain.SMGroups, error) {
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

	return h.srcds.SaveGroup(ctx, group)
}
