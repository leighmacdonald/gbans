package report

import (
	"context"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

type reportUsecase struct {
	rr domain.ReportRepository
}

func NewReportUsecase(rr domain.ReportRepository) domain.ReportUsecase {
	return &reportUsecase{rr: rr}
}

func (r reportUsecase) GetReportBySteamID(ctx context.Context, authorID steamid.SID64, steamID steamid.SID64, report *domain.Report) error {
	return r.rr.GetReportBySteamID(ctx, authorID, steamID, report)
}

func (r reportUsecase) GetReports(ctx context.Context, opts domain.ReportQueryFilter) ([]domain.Report, int64, error) {
	return r.rr.GetReports(ctx, opts)
}

func (r reportUsecase) GetReport(ctx context.Context, reportID int64, report *domain.Report) error {
	return r.rr.GetReport(ctx, reportID, report)
}

func (r reportUsecase) GetReportMessages(ctx context.Context, reportID int64) ([]domain.ReportMessage, error) {
	return r.rr.GetReportMessages(ctx, reportID)
}

func (r reportUsecase) GetReportMessageByID(ctx context.Context, reportMessageID int64, message *domain.ReportMessage) error {
	return r.rr.GetReportMessageByID(ctx, reportMessageID, message)
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
