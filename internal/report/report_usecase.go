package report

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type reportUsecase struct {
	rr          domain.ReportRepository
	du          domain.DiscordUsecase
	cu          domain.ConfigUsecase
	pu          domain.PersonUsecase
	demoUsecase domain.DemoUsecase
}

func NewReportUsecase(repository domain.ReportRepository, discordUsecase domain.DiscordUsecase,
	configUsecase domain.ConfigUsecase, personUsecase domain.PersonUsecase, demoUsecase domain.DemoUsecase,
) domain.ReportUsecase {
	return &reportUsecase{
		du:          discordUsecase,
		rr:          repository,
		cu:          configUsecase,
		pu:          personUsecase,
		demoUsecase: demoUsecase,
	}
}

func (r reportUsecase) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Hour * 24)
	updateChan := make(chan any)

	go func() {
		time.Sleep(time.Second * 5)
		updateChan <- true
	}()

	admin := domain.UserProfile{SteamID: steamid.New(r.cu.Config().General.Owner)}

	for {
		select {
		case <-ticker.C:
			updateChan <- true
		case <-updateChan:
			reports, errReports := r.GetReports(ctx, admin, domain.ReportQueryFilter{})
			if errReports != nil {
				slog.Error("failed to fetch reports for report metadata", log.ErrAttr(errReports))

				continue
			}

			var (
				now  = time.Now()
				meta domain.ReportMeta
			)

			for _, report := range reports {
				if report.ReportStatus == domain.ClosedWithAction || report.ReportStatus == domain.ClosedWithoutAction {
					meta.TotalClosed++

					continue
				}

				meta.TotalOpen++

				if report.ReportStatus == domain.NeedMoreInfo {
					meta.NeedInfo++
				} else {
					meta.Open++
				}

				switch {
				case now.Sub(report.CreatedOn) > time.Hour*24*7:
					meta.OpenWeek++
				case now.Sub(report.CreatedOn) > time.Hour*24*3:
					meta.Open3Days++
				case now.Sub(report.CreatedOn) > time.Hour*24:
					meta.Open1Day++
				default:
					meta.OpenNew++
				}
			}

			r.du.SendPayload(domain.ChannelMod, discord.ReportStatsMessage(meta, r.cu.ExtURLRaw("/admin/reports")))
		case <-ctx.Done():
			slog.Debug("showReportMeta shutting down")

			return
		}
	}
}

func (r reportUsecase) GetReportBySteamID(ctx context.Context, authorID steamid.SteamID, steamID steamid.SteamID) (domain.Report, error) {
	return r.rr.GetReportBySteamID(ctx, authorID, steamID)
}

func (r reportUsecase) GetReports(ctx context.Context, user domain.PersonInfo, opts domain.ReportQueryFilter) ([]domain.ReportWithAuthor, error) {
	// Make sure the person requesting is either a moderator, or a user
	// only able to request their own reports
	var sourceID steamid.SteamID

	if !user.HasPermission(domain.PModerator) {
		sourceID = user.GetSteamID()
	} else {
		if sid, ok := opts.SourceSteamID(ctx); ok {
			sourceID = sid
		}
	}

	if sourceID.Valid() {
		opts.SourceID = sourceID.String()
	}

	reports, errReports := r.rr.GetReports(ctx, opts)
	if errReports != nil {
		if errors.Is(errReports, domain.ErrNoResult) {
			return nil, nil
		}

		return nil, errReports
	}

	var peopleIDs steamid.Collection
	for _, report := range reports {
		peopleIDs = append(peopleIDs, report.SourceID, report.TargetID)
	}

	people, errAuthors := r.pu.GetPeopleBySteamID(ctx, fp.Uniq(peopleIDs))
	if errAuthors != nil {
		return nil, errAuthors
	}

	peopleMap := people.AsMap()

	userReports := make([]domain.ReportWithAuthor, len(reports))

	for i, report := range reports {
		userReports[i] = domain.ReportWithAuthor{
			Author:  peopleMap[report.SourceID],
			Report:  report,
			Subject: peopleMap[report.TargetID],
		}
	}

	return userReports, nil
}

func (r reportUsecase) GetReport(ctx context.Context, curUser domain.PersonInfo, reportID int64) (domain.ReportWithAuthor, error) {
	report, err := r.rr.GetReport(ctx, reportID)
	if err != nil {
		return domain.ReportWithAuthor{}, err
	}

	author, errAuthor := r.pu.GetPersonBySteamID(ctx, report.SourceID)
	if errAuthor != nil {
		return domain.ReportWithAuthor{}, errAuthor
	}

	if !httphelper.HasPrivilege(curUser, steamid.Collection{author.SteamID}, domain.PModerator) {
		return domain.ReportWithAuthor{}, domain.ErrPermissionDenied
	}

	target, errTarget := r.pu.GetPersonBySteamID(ctx, report.TargetID)
	if errTarget != nil {
		return domain.ReportWithAuthor{}, errTarget
	}

	var demo domain.DemoFile
	if report.DemoID > 0 {
		if errDemo := r.demoUsecase.GetDemoByID(ctx, report.DemoID, &demo); errDemo != nil {
			slog.Error("Failed to load report demo", slog.Int64("report_id", report.ReportID))
		}
	}

	return domain.ReportWithAuthor{
		Author:  author,
		Subject: target,
		Report:  report,
		Demo:    demo,
	}, nil
}

func (r reportUsecase) GetReportMessages(ctx context.Context, reportID int64) ([]domain.ReportMessage, error) {
	return r.rr.GetReportMessages(ctx, reportID)
}

func (r reportUsecase) GetReportMessageByID(ctx context.Context, reportMessageID int64) (domain.ReportMessage, error) {
	return r.rr.GetReportMessageByID(ctx, reportMessageID)
}

func (r reportUsecase) DropReportMessage(ctx context.Context, message *domain.ReportMessage) error {
	return r.rr.DropReportMessage(ctx, message)
}

func (r reportUsecase) DropReport(ctx context.Context, report *domain.Report) error {
	return r.rr.DropReport(ctx, report)
}

func (r reportUsecase) SaveReport(ctx context.Context, report *domain.Report) error {
	return r.rr.SaveReport(ctx, report)
}

func (r reportUsecase) SaveReportMessage(ctx context.Context, message *domain.ReportMessage) error {
	return r.rr.SaveReportMessage(ctx, message)
}
