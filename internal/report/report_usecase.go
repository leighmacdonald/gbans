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
	"github.com/leighmacdonald/steamid/v3/steamid"
)

type reportUsecase struct {
	rr domain.ReportRepository
	du domain.DiscordUsecase
	cu domain.ConfigUsecase
	pu domain.PersonUsecase
}

func NewReportUsecase(repository domain.ReportRepository, discordUsecase domain.DiscordUsecase,
	configUsecase domain.ConfigUsecase, personUsecase domain.PersonUsecase,
) domain.ReportUsecase {
	return &reportUsecase{
		du: discordUsecase,
		rr: repository,
		cu: configUsecase,
		pu: personUsecase,
	}
}

func (r reportUsecase) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Hour * 24)
	updateChan := make(chan any)

	go func() {
		time.Sleep(time.Second * 5)
		updateChan <- true
	}()

	admin := domain.UserProfile{SteamID: r.cu.Config().General.Owner}

	for {
		select {
		case <-ticker.C:
			updateChan <- true
		case <-updateChan:
			reports, _, errReports := r.GetReports(ctx, admin, domain.ReportQueryFilter{
				QueryFilter: domain.QueryFilter{
					Limit: 0,
				},
			})
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

func (r reportUsecase) GetReportBySteamID(ctx context.Context, authorID steamid.SID64, steamID steamid.SID64) (domain.Report, error) {
	return r.rr.GetReportBySteamID(ctx, authorID, steamID)
}

func (r reportUsecase) GetReports(ctx context.Context, user domain.PersonInfo, opts domain.ReportQueryFilter) ([]domain.ReportWithAuthor, int64, error) {
	if opts.Limit <= 0 && opts.Limit > 100 {
		opts.Limit = 25
	}

	// Make sure the person requesting is either a moderator, or a user
	// only able to request their own reports
	var sourceID steamid.SID64

	if !user.HasPermission(domain.PModerator) {
		sourceID = user.GetSteamID()
	} else if opts.SourceID != "" {
		sid, errSourceID := opts.SourceID.SID64(ctx)
		if errSourceID != nil {
			return nil, 0, errSourceID
		}

		sourceID = sid
	}

	if sourceID.Valid() {
		opts.SourceID = domain.StringSID(sourceID.String())
	}

	reports, count, errReports := r.rr.GetReports(ctx, opts)
	if errReports != nil {
		if errors.Is(errReports, domain.ErrNoResult) {
			return nil, 0, nil
		}

		return nil, 0, errReports
	}

	var peopleIDs steamid.Collection
	for _, report := range reports {
		peopleIDs = append(peopleIDs, report.SourceID, report.TargetID)
	}

	people, errAuthors := r.pu.GetPeopleBySteamID(ctx, fp.Uniq(peopleIDs))
	if errAuthors != nil {
		return nil, 0, errAuthors
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

	return userReports, count, nil
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

	return domain.ReportWithAuthor{
		Author:  author,
		Subject: target,
		Report:  report,
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
