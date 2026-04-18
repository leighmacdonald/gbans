package ban

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	v1 "github.com/leighmacdonald/gbans/internal/ban/v1"
	"github.com/leighmacdonald/gbans/internal/ban/v1/banv1connect"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	personv1 "github.com/leighmacdonald/gbans/internal/person/v1"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ReportService struct {
	banv1connect.UnimplementedReportServiceHandler

	reports Reports
}

func NewReportService(reports Reports) ReportService {
	return ReportService{reports: reports}
}

func (s ReportService) ReportCreate(ctx context.Context, req *v1.CreateReportRequest) (*v1.CreateReportResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	report, errReportSave := s.reports.Save(ctx, user, RequestReportCreate{
		SourceID:        user.GetSteamID(),
		TargetID:        steamid.New(req.GetTargetId()),
		Description:     req.GetDescription(),
		Reason:          reason.Reason(req.GetReason()),
		ReasonText:      req.GetReasonText(),
		DemoID:          req.GetDemoId(),
		DemoTick:        req.GetDemoTick(),
		PersonMessageID: req.GetPersonMessageId(),
	})
	if errReportSave != nil {
		if errors.Is(errReportSave, ErrReportExists) {
			return nil, connect.NewError(connect.CodeAlreadyExists, rpc.ErrExists)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.CreateReportResponse{Report: toReportWithAuthor(report)}, nil
}

func (s ReportService) Report(ctx context.Context, req *v1.ReportRequest) (*v1.ReportResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	report, errReport := s.reports.Report(ctx, user, req.GetReportId())
	if errReport != nil {
		if errors.Is(errReport, database.ErrNoResult) {
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.ReportResponse{Report: toReportWithAuthor(report)}, nil
}

func (s ReportService) ReportStatusEdit(ctx context.Context, req *v1.ReportStatusEditRequest) (*emptypb.Empty, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	_, err := s.reports.SetReportStatus(ctx, req.GetReportId(), user, ReportStatus(req.GetReportStatus()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &emptypb.Empty{}, nil
}

func (s ReportService) UserReports(ctx context.Context, req *v1.UserReportsRequest) (*v1.UserReportsResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	reports, errReports := s.reports.BySteamID(ctx, user.GetSteamID())
	if errReports != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.UserReportsResponse{Reports: make([]*v1.ReportWithAuthor, len(reports))}
	for idx, report := range reports {
		resp.Reports[idx] = toReportWithAuthor(report)
	}

	return &resp, nil
}

func (s ReportService) ReportMessages(ctx context.Context, req *v1.ReportMessagesRequest) (*v1.ReportMessagesResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	report, errGetReport := s.reports.Report(ctx, user, req.GetReportId())
	if errGetReport != nil {
		if errors.Is(errGetReport, database.ErrNoResult) {
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	if !httphelper.HasPrivilege(user, steamid.Collection{report.SourceID, report.TargetID}, permission.Moderator) {
		return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
	}

	reportMessages, errGetReportMessages := s.reports.Messages(ctx, req.GetReportId())
	if errGetReportMessages != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.ReportMessagesResponse{Messages: make([]*v1.ReportMessage, len(reportMessages))}
	for idx, reportMessage := range reportMessages {
		resp.Messages[idx] = toReportMessage(reportMessage)
	}

	return &resp, nil
}

func (s ReportService) ReportMessageCreate(ctx context.Context, req *v1.CreateReportMessageRequest) (*v1.CreateReportMessageResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	msg, errSave := s.reports.CreateMessage(ctx, req.GetReportId(), user, RequestMessageBodyMD{BodyMD: req.GetBodyMd()})
	if errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.CreateReportMessageResponse{ReportMessage: toReportMessage(msg)}, nil
}

func (s ReportService) ReportMessageEdit(ctx context.Context, req *v1.ReportMessageEditRequest) (*v1.ReportMessageEditResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	msg, errMsg := s.reports.EditMessage(ctx, req.GetReportMessageId(), user, RequestMessageBodyMD{BodyMD: req.GetBodyMd()})
	if errMsg != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.ReportMessageEditResponse{Message: toReportMessage(msg)}, nil
}

func (s ReportService) ReportMessageDelete(ctx context.Context, req *v1.ReportMessageDeleteRequest) (*emptypb.Empty, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	if err := s.reports.DropMessage(ctx, user, req.GetReportMessageId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &emptypb.Empty{}, nil
}

func toReportMessage(msg ReportMessage) *v1.ReportMessage {
	return &v1.ReportMessage{
		ReportId:        &msg.ReportID,
		ReportMessageId: &msg.ReportMessageID,
		AuthorId:        ptr.To(msg.AuthorID.Int64()),
		MessageMd:       &msg.MessageMD,
		Deleted:         &msg.Deleted,
		CreatedOn:       timestamppb.New(msg.CreatedOn),
		UpdatedOn:       timestamppb.New(msg.UpdatedOn),
		PersonaName:     &msg.Personaname,
		AvatarHash:      &msg.Avatarhash,
		PermissionLevel: ptr.To(personv1.Privilege(msg.PermissionLevel)),
	}
}

func toReportWithAuthor(report ReportWithAuthor) *v1.ReportWithAuthor {
	return &v1.ReportWithAuthor{
		Report:  toReport(report.Report),
		Author:  toPersonCore(report.Author),
		Subject: toPersonCore(report.Subject),
	}
}

func toPersonCore(person person.Core) *personv1.PersonCore {
	return &personv1.PersonCore{
		SteamId:         ptr.To(person.SteamID.Int64()),
		PermissionLevel: ptr.To(personv1.Privilege(person.PermissionLevel)),
		Name:            ptr.To(person.GetName()),
		AvatarHash:      ptr.To(string(person.GetAvatar())),
		DiscordId:       ptr.To(person.GetDiscordID()),
		VacBans:         ptr.To(person.GetVACBans()),
		GameBans:        ptr.To(person.GetGameBans()),
		BanId:           &person.BanID,
		TimeCreated:     timestamppb.New(person.GetTimeCreated()),
	}
}

func toReport(report Report) *v1.Report {
	return &v1.Report{
		ReportId:        &report.ReportID,
		SourceId:        ptr.To(report.SourceID.Int64()),
		TargetId:        ptr.To(report.TargetID.Int64()),
		Description:     &report.Description,
		ReportStatus:    ptr.To(v1.ReportStatus(report.ReportStatus)),
		Reason:          ptr.To(v1.BanReason(report.Reason)),
		ReasonText:      &report.ReasonText,
		Deleted:         &report.Deleted,
		DemoTick:        &report.DemoTick,
		DemoId:          &report.DemoID,
		PersonMessageId: &report.PersonMessageID,
		CreatedOn:       timestamppb.New(report.CreatedOn),
		UpdatedOn:       timestamppb.New(report.UpdatedOn),
	}
}
