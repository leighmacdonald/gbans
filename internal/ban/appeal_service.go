package ban

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	v1 "github.com/leighmacdonald/gbans/internal/ban/v1"
	"github.com/leighmacdonald/gbans/internal/ban/v1/banv1connect"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	personv1 "github.com/leighmacdonald/gbans/internal/person/v1"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AppealService struct {
	banv1connect.UnimplementedAppealServiceHandler

	appeals Appeals
}

func NewAppealService(appeals Appeals, authMiddleware *rpc.Middleware, options ...connect.HandlerOption) rpc.Service {
	pattern, handler := banv1connect.NewAppealServiceHandler(AppealService{appeals: appeals}, options...)

	authMiddleware.UserRoute(banv1connect.AppealServiceAppealsProcedure, rpc.WithMinPermissions(permission.Moderator))
	authMiddleware.UserRoute(banv1connect.AppealServiceSetAppealStateProcedure, rpc.WithMinPermissions(permission.Moderator))
	authMiddleware.UserRoute(banv1connect.AppealServiceMessagesProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(banv1connect.AppealServiceReplyProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(banv1connect.AppealServiceEditAppealMessageProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(banv1connect.AppealServiceDeleteAppealMessageProcedure, rpc.WithMinPermissions(permission.User))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s AppealService) Appeals(ctx context.Context, req *v1.AppealsRequest) (*v1.AppealsResponse, error) {
	appeals, errAppeals := s.appeals.GetAppealsByActivity(ctx, AppealQueryFilter{Deleted: req.GetDeleted()})
	if errAppeals != nil {
		return nil, connect.NewError(connect.CodeInternal, errAppeals)
	}

	resp := v1.AppealsResponse{Appeals: make([]*v1.AppealOverview, len(appeals))}
	for idx, appeal := range appeals {
		resp.Appeals[idx] = &v1.AppealOverview{
			Ban:               toBan(appeal.Ban),
			SourcePersonaName: &appeal.SourcePersonaname,
			SourceAvatarHash:  &appeal.SourceAvatarhash,
			TargetPersonaName: &appeal.TargetPersonaname,
			TargetAvatarHash:  &appeal.TargetAvatarhash,
		}
	}

	return &resp, nil
}

func (s AppealService) Messages(ctx context.Context, req *v1.MessagesRequest) (*v1.MessagesResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)

	banMessages, errGetBanMessages := s.appeals.Messages(ctx, user, req.GetBanId())
	if errGetBanMessages != nil && !errors.Is(errGetBanMessages, httphelper.ErrNotFound) {
		return nil, connect.NewError(connect.CodeInternal, errGetBanMessages)
	}

	resp := v1.MessagesResponse{Messages: make([]*v1.AppealMessage, len(banMessages))}
	for idx, message := range banMessages {
		resp.Messages[idx] = toAppealMessage(message)
	}

	return &resp, nil
}

func (s AppealService) Reply(ctx context.Context, req *v1.ReplyRequest) (*v1.ReplyResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	msg, errSave := s.appeals.CreateBanMessage(ctx, user, req.GetBanId(), req.GetBodyMd())
	if errSave != nil {
		if errors.Is(errSave, permission.ErrDenied) {
			return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
		}

		return nil, connect.NewError(connect.CodeInternal, errSave)
	}

	return &v1.ReplyResponse{Message: toAppealMessage(msg)}, nil
}

func (s AppealService) EditAppealMessage(ctx context.Context, req *v1.EditAppealMessageRequest) (*v1.EditAppealMessageResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	msg, errSave := s.appeals.EditBanMessage(ctx, user, req.GetBanMessageId(), req.GetBodyMd())
	if errSave != nil {
		switch {
		case errors.Is(errSave, httphelper.ErrParamInvalid):
			return nil, connect.NewError(connect.CodeInvalidArgument, rpc.ErrBadRequest)
		case errors.Is(errSave, permission.ErrDenied):
			return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
		case errors.Is(errSave, database.ErrDuplicate):
			return nil, connect.NewError(connect.CodeAlreadyExists, rpc.ErrExists)
		default:
			return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
		}
	}

	return &v1.EditAppealMessageResponse{Message: toAppealMessage(msg)}, nil
}

func (s AppealService) DeleteAppealMessage(ctx context.Context, req *v1.DeleteAppealMessageRequest) (*emptypb.Empty, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	if err := s.appeals.DropMessage(ctx, user, req.GetBanMessageId()); err != nil {
		switch {
		case errors.Is(err, permission.ErrDenied):
			return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
		case errors.Is(err, database.ErrNoResult):
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		default:
			return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
		}
	}

	return &emptypb.Empty{}, nil
}

func toAppealMessage(message AppealMessage) *v1.AppealMessage {
	return &v1.AppealMessage{
		BanId:        &message.BanID,
		BanMessageId: &message.BanMessageID,
		AuthorId:     ptr.To(message.AuthorID.Int64()),
		MessageMd:    &message.MessageMD,
		Deleted:      &message.Deleted,
		CreatedOn:    timestamppb.New(message.CreatedOn),
		UpdatedOn:    timestamppb.New(message.UpdatedOn),
		AvatarHash:   &message.Avatarhash,
		PersonaName:  &message.Personaname,
		Privilege:    ptr.To(personv1.Privilege(message.PermissionLevel)),
	}
}
