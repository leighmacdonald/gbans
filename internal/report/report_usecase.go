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
	repository domain.ReportRepository
	discord    domain.DiscordUsecase
	config     domain.ConfigUsecase
	persons    domain.PersonUsecase
	demos      domain.DemoUsecase
}

func NewReportUsecase(repository domain.ReportRepository, discord domain.DiscordUsecase,
	config domain.ConfigUsecase, persons domain.PersonUsecase, demos domain.DemoUsecase,
) domain.ReportUsecase {
	return &reportUsecase{
		discord:    discord,
		repository: repository,
		config:     config,
		persons:    persons,
		demos:      demos,
	}
}

func (r reportUsecase) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Hour * 24)
	updateChan := make(chan any)

	go func() {
		time.Sleep(time.Second * 5)
		updateChan <- true
	}()

	for {
		select {
		case <-ticker.C:
			updateChan <- true
		case <-updateChan:
			reports, errReports := r.GetReports(ctx)
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

			r.discord.SendPayload(domain.ChannelMod, discord.ReportStatsMessage(meta, r.config.ExtURLRaw("/admin/reports")))
		case <-ctx.Done():
			slog.Debug("showReportMeta shutting down")

			return
		}
	}
}

func (r reportUsecase) addAuthorsToReports(ctx context.Context, reports []domain.Report) ([]domain.ReportWithAuthor, error) {
	var peopleIDs steamid.Collection
	for _, report := range reports {
		peopleIDs = append(peopleIDs, report.SourceID, report.TargetID)
	}

	people, errAuthors := r.persons.GetPeopleBySteamID(ctx, fp.Uniq(peopleIDs))
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

func (r reportUsecase) GetReportsBySteamID(ctx context.Context, steamID steamid.SteamID) ([]domain.ReportWithAuthor, error) {
	if !steamID.Valid() {
		return nil, domain.ErrInvalidSID
	}

	reports, errReports := r.repository.GetReports(ctx, steamID)
	if errReports != nil {
		if errors.Is(errReports, domain.ErrNoResult) {
			return nil, nil
		}

		return nil, errReports
	}

	return r.addAuthorsToReports(ctx, reports)
}

func (r reportUsecase) GetReports(ctx context.Context) ([]domain.ReportWithAuthor, error) {
	reports, errReports := r.repository.GetReports(ctx, steamid.SteamID{})
	if errReports != nil {
		if errors.Is(errReports, domain.ErrNoResult) {
			return nil, nil
		}

		return nil, errReports
	}

	return r.addAuthorsToReports(ctx, reports)
}

func (r reportUsecase) GetReport(ctx context.Context, curUser domain.PersonInfo, reportID int64) (domain.ReportWithAuthor, error) {
	report, err := r.repository.GetReport(ctx, reportID)
	if err != nil {
		return domain.ReportWithAuthor{}, err
	}

	author, errAuthor := r.persons.GetPersonBySteamID(ctx, report.SourceID)
	if errAuthor != nil {
		return domain.ReportWithAuthor{}, errAuthor
	}

	if !httphelper.HasPrivilege(curUser, steamid.Collection{author.SteamID}, domain.PModerator) {
		return domain.ReportWithAuthor{}, domain.ErrPermissionDenied
	}

	target, errTarget := r.persons.GetPersonBySteamID(ctx, report.TargetID)
	if errTarget != nil {
		return domain.ReportWithAuthor{}, errTarget
	}

	var demo domain.DemoFile
	if report.DemoID > 0 {
		if errDemo := r.demos.GetDemoByID(ctx, report.DemoID, &demo); errDemo != nil {
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
	return r.repository.GetReportMessages(ctx, reportID)
}

func (r reportUsecase) GetReportMessageByID(ctx context.Context, reportMessageID int64) (domain.ReportMessage, error) {
	return r.repository.GetReportMessageByID(ctx, reportMessageID)
}

func (r reportUsecase) DropReportMessage(ctx context.Context, message *domain.ReportMessage) error {
	return r.repository.DropReportMessage(ctx, message)
}

func (r reportUsecase) DropReport(ctx context.Context, report *domain.Report) error {
	return r.repository.DropReport(ctx, report)
}

func (r reportUsecase) SaveReport(ctx context.Context, report *domain.Report) error {
	if err := r.repository.SaveReport(ctx, report); err != nil {
		return err
	}

	slog.Info("New report created", slog.Int64("report_id", report.ReportID))

	return nil
}

func (r reportUsecase) SaveReportMessage(ctx context.Context, message *domain.ReportMessage) error {
	return r.repository.SaveReportMessage(ctx, message)
}
