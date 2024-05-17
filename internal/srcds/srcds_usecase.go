package srcds

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type srcdsUsecase struct {
	cu     domain.ConfigUsecase
	sv     domain.ServersUsecase
	sr     srcdsRepository
	pu     domain.PersonUsecase
	ru     domain.ReportUsecase
	du     domain.DiscordUsecase
	cookie string
}

func NewSrcdsUsecase(configUsecase domain.ConfigUsecase, serversUsecase domain.ServersUsecase,
	personUsecase domain.PersonUsecase, reportUsecase domain.ReportUsecase, discordUsecase domain.DiscordUsecase,
) domain.SRCDSUsecase {
	return &srcdsUsecase{
		cu:     configUsecase,
		sv:     serversUsecase,
		pu:     personUsecase,
		ru:     reportUsecase,
		du:     discordUsecase,
		cookie: configUsecase.Config().HTTP.CookieKey,
	}
}

func (h srcdsUsecase) ServerAuth(ctx context.Context, req domain.ServerAuthReq) (string, error) {
	var server domain.Server

	errGetServer := h.sv.GetServerByPassword(ctx, req.Key, &server, true, false)
	if errGetServer != nil {
		return "", errGetServer
	}

	if server.Password != req.Key {
		return "", domain.ErrPermissionDenied
	}

	accessToken, errToken := newServerToken(server.ServerID, h.cookie)
	if errToken != nil {
		return "", errToken
	}

	server.TokenCreatedOn = time.Now()
	if errSaveServer := h.sv.SaveServer(ctx, &server); errSaveServer != nil {
		return "", errSaveServer
	}

	return accessToken, nil
}

func (h srcdsUsecase) Report(ctx context.Context, currentUser domain.UserProfile, req domain.CreateReportReq) (*domain.Report, error) {
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

	personSource, errCreatePerson := h.pu.GetPersonBySteamID(ctx, req.SourceID)
	if errCreatePerson != nil {
		return nil, domain.ErrInternal
	}

	personTarget, errCreatePerson := h.pu.GetOrCreatePersonBySteamID(ctx, req.TargetID)
	if errCreatePerson != nil {
		return nil, domain.ErrInternal
	}

	if personTarget.Expired() {
		if err := thirdparty.UpdatePlayerSummary(ctx, &personTarget); err != nil {
			slog.Error("Failed to update target player", log.ErrAttr(err))
		} else {
			if errSave := h.pu.SavePerson(ctx, &personTarget); errSave != nil {
				slog.Error("Failed to save target player update", log.ErrAttr(err))
			}
		}
	}

	// Ensure the user doesn't already have an open report against the user
	existing, errReports := h.ru.GetReportBySteamID(ctx, personSource.SteamID, req.TargetID)
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

	if errReportSave := h.ru.SaveReport(ctx, &report); errReportSave != nil {
		return nil, errReportSave
	}

	slog.Info("New report created successfully", slog.Int64("report_id", report.ReportID))

	conf := h.cu.Config()

	demoURL := ""

	msg := discord.NewInGameReportResponse(report, conf.ExtURL(report), currentUser, conf.ExtURL(currentUser), demoURL)

	h.du.SendPayload(domain.ChannelModLog, msg)

	return &report, nil
}

func (h srcdsUsecase) SetAdminGroups(ctx context.Context, authType domain.AuthType, identity string, groups ...domain.SMGroups) error {
	admin, errAdmin := h.sr.GetAdminByID(ctx, authType, identity)
	if errAdmin != nil {
		return errAdmin
	}

	// Delete existing groups.
	if errDelete := h.sr.DeleteAdminGroups(ctx, admin); errDelete != nil && !errors.Is(errDelete, domain.ErrNoResult) {
		return errDelete
	}

	// If no groups are given to add, this is treated purely as a delete function
	if len(groups) == 0 {
		return nil
	}

	for i := range groups {
		if errInsert := h.sr.InsertAdminGroup(ctx, admin, groups[i], i); errInsert != nil {
			return errInsert
		}
	}

	return nil
}

func (h srcdsUsecase) DelGroup(ctx context.Context, groupID int) error {
	group, errGroup := h.sr.GetGroupByID(ctx, groupID)
	if errGroup != nil {
		return errGroup
	}

	return h.sr.DeleteGroup(ctx, group)
}

func (h srcdsUsecase) AddGroup(ctx context.Context, name string, flags string, immunityLevel int) (domain.SMGroups, error) {
	if name == "" {
		return domain.SMGroups{}, domain.ErrSMGroupName
	}

	if immunityLevel > 100 || immunityLevel < 0 {
		return domain.SMGroups{}, domain.ErrSMImmunity
	}

	return h.sr.AddGroup(ctx, domain.SMGroups{
		Flags:         flags,
		Name:          name,
		ImmunityLevel: immunityLevel,
	})
}

func validateAuthTypeOpts(authType domain.AuthType, adminID string) error {
	switch {
	case authType == domain.AuthTypeSteam:
		sid := steamid.New(adminID)
		if !sid.Valid() {
			return domain.ErrInvalidSID
		}
	case authType == domain.AuthTypeIP:
		if net.ParseIP(adminID) == nil {
			return domain.ErrInvalidIP
		}
	case authType == domain.AuthTypeName:
		if adminID == "" {
			return domain.ErrSMInvalidAuthName
		}
	}

	return nil
}

func (h srcdsUsecase) DelAdmin(ctx context.Context, authType domain.AuthType, identity string) error {
	if errValidate := validateAuthTypeOpts(authType, identity); errValidate != nil {
		return errValidate
	}

	admin, errAdmin := h.sr.GetAdminByID(ctx, authType, identity)
	if errAdmin != nil {
		return errAdmin
	}

	return h.sr.DelAdmin(ctx, admin)
}

func (h srcdsUsecase) AddAdmin(ctx context.Context, alias string, authType domain.AuthType, identity string, flags string, immunity int, password string) (domain.SMAdmin, error) {
	if errValidate := validateAuthTypeOpts(authType, identity); errValidate != nil {
		return domain.SMAdmin{}, errValidate
	}

	if immunity < 0 || immunity > 100 {
		return domain.SMAdmin{}, domain.ErrSMImmunity
	}

	admin, errAdmin := h.sr.GetAdminByID(ctx, authType, identity)
	if errAdmin != nil && errors.Is(errAdmin, domain.ErrNoResult) {
		return domain.SMAdmin{}, errAdmin
	}

	if errAdmin == nil {
		return admin, domain.ErrSMAdminExists
	}

	var steamID steamid.SteamID
	if authType == domain.AuthTypeSteam {
		steamID = steamid.New(identity)
	}

	return h.sr.AddAdmin(ctx, domain.SMAdmin{
		SteamID:  steamID,
		AuthType: authType,
		Identity: identity,
		Password: password,
		Flags:    flags,
		Name:     alias,
		Immunity: immunity,
	})
}
