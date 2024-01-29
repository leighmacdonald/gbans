package srcds

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"go.uber.org/zap"
)

type srcdsUsecase struct {
	cu     domain.ConfigUsecase
	sv     domain.ServersUsecase
	pu     domain.PersonUsecase
	ru     domain.ReportUsecase
	du     domain.DiscordUsecase
	log    *zap.Logger
	cookie string
}

func NewSrcdsUsecase(log *zap.Logger, cu domain.ConfigUsecase, sv domain.ServersUsecase,
	pu domain.PersonUsecase, ru domain.ReportUsecase, du domain.DiscordUsecase,
) domain.SRCDSUsecase {
	return &srcdsUsecase{
		log:    log,
		cu:     cu,
		sv:     sv,
		pu:     pu,
		ru:     ru,
		du:     du,
		cookie: cu.Config().HTTP.CookieKey,
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
	if req.SourceID == "" {
		req.SourceID = domain.StringSID(currentUser.SteamID.String())
	}

	sourceID, errSourceID := req.SourceID.SID64(ctx)
	if errSourceID != nil {
		return nil, domain.ErrSourceID
	}

	targetID, errTargetID := req.TargetID.SID64(ctx)
	if errTargetID != nil {
		return nil, domain.ErrTargetID
	}

	if sourceID == targetID {
		return nil, domain.ErrSelfReport
	}

	var personSource domain.Person
	if errCreatePerson := h.pu.GetPersonBySteamID(ctx, sourceID, &personSource); errCreatePerson != nil {
		return nil, domain.ErrInternal
	}

	var personTarget domain.Person
	if errCreatePerson := h.pu.GetOrCreatePersonBySteamID(ctx, targetID, &personTarget); errCreatePerson != nil {
		return nil, domain.ErrInternal
	}

	if personTarget.Expired() {
		if err := thirdparty.UpdatePlayerSummary(ctx, &personTarget); err != nil {
			h.log.Error("Failed to update target player", zap.Error(err))
		} else {
			if errSave := h.pu.SavePerson(ctx, &personTarget); errSave != nil {
				h.log.Error("Failed to save target player update", zap.Error(err))
			}
		}
	}

	// Ensure the user doesn't already have an open report against the user
	var existing domain.Report
	if errReports := h.ru.GetReportBySteamID(ctx, personSource.SteamID, targetID, &existing); errReports != nil {
		if !errors.Is(errReports, domain.ErrNoResult) {
			return nil, errReports
		}
	}

	if existing.ReportID > 0 {
		return nil, domain.ErrReportExists
	}

	// TODO encapsulate all operations in single tx
	report := domain.NewReport()
	report.SourceID = sourceID
	report.ReportStatus = domain.Opened
	report.Description = req.Description
	report.TargetID = targetID
	report.Reason = req.Reason
	report.ReasonText = req.ReasonText
	parts := strings.Split(req.DemoName, "/")
	report.DemoName = parts[len(parts)-1]
	report.DemoTick = req.DemoTick
	report.PersonMessageID = req.PersonMessageID

	if errReportSave := h.ru.SaveReport(ctx, &report); errReportSave != nil {
		return nil, errReportSave
	}

	h.log.Info("New report created successfully", zap.Int64("report_id", report.ReportID))

	conf := h.cu.Config()

	demoURL := ""

	if report.DemoName != "" {
		demoURL = conf.ExtURLRaw("/demos/name/%s", report.DemoName)
	}

	msg := discord.NewInGameReportResponse(report, conf.ExtURL(report), currentUser, conf.ExtURL(currentUser), demoURL)

	h.du.SendPayload(domain.ChannelModLog, msg)

	return &report, nil
}
