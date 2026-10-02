package notification

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/database"
	v1 "github.com/leighmacdonald/gbans/internal/notification/v1"
	"github.com/leighmacdonald/gbans/internal/notification/v1/notificationv1connect"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	// notificationv1connect.UnimplementedNotificationServiceHandler

	notifications *Notifications
}

func NewService(notifications *Notifications, roleAuth *rpc.RoleAuth, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := notificationv1connect.NewNotificationServiceHandler(Service{notifications: notifications}, option...)

	authMiddleware.UserRoute(notificationv1connect.NotificationServiceNotificationsProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_NOTIFICATIONS))
	authMiddleware.UserRoute(notificationv1connect.NotificationServiceMarkReadProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_NOTIFICATIONS))
	authMiddleware.UserRoute(notificationv1connect.NotificationServiceMarkReadAllProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_NOTIFICATIONS))
	authMiddleware.UserRoute(notificationv1connect.NotificationServiceDeleteAllProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_NOTIFICATIONS))
	authMiddleware.UserRoute(notificationv1connect.NotificationServiceDeleteProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_NOTIFICATIONS))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s Service) Notifications(ctx context.Context, _ *emptypb.Empty) (*v1.NotificationsResponse, error) {
	user := rpc.UserInfoFromCtx(ctx)
	notifications, err := s.notifications.GetPersonNotifications(ctx, user.GetSteamID())
	if err != nil && !errors.Is(err, database.ErrNoResult) {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.NotificationsResponse{Notifications: make([]*v1.UserNotification, len(notifications))}
	for idx, notif := range notifications {
		resp.Notifications[idx] = &v1.UserNotification{
			PersonNotificationId: &notif.PersonNotificationID,
			SteamId:              new(notif.SteamID.Int64()),
			Read:                 &notif.Read,
			Deleted:              &notif.Deleted,
			Severity:             new(v1.Severity(notif.Severity)), //nolint:gosec
			Message:              &notif.Message,
			Link:                 &notif.Link,
			Count:                &notif.Count,
			CreatedOn:            timestamppb.New(notif.CreatedOn),
		}
	}

	return &resp, nil
}

func (s Service) MarkRead(ctx context.Context, req *v1.MarkReadRequest) (*emptypb.Empty, error) {
	user := rpc.UserInfoFromCtx(ctx)
	if err := s.notifications.MarkMessagesRead(ctx, user.GetSteamID(), req.GetMessageId()); err != nil && !errors.Is(err, database.ErrNoResult) {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &emptypb.Empty{}, nil
}

func (s Service) MarkReadAll(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	user := rpc.UserInfoFromCtx(ctx)
	if err := s.notifications.MarkAllRead(ctx, user.GetSteamID()); err != nil && !errors.Is(err, database.ErrNoResult) {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &emptypb.Empty{}, nil
}

func (s Service) DeleteAll(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	user := rpc.UserInfoFromCtx(ctx)
	if err := s.notifications.DeleteAll(ctx, user.GetSteamID()); err != nil && !errors.Is(err, database.ErrNoResult) {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &emptypb.Empty{}, nil
}

func (s Service) Delete(ctx context.Context, req *v1.DeleteRequest) (*emptypb.Empty, error) {
	user := rpc.UserInfoFromCtx(ctx)
	if err := s.notifications.DeleteMessages(ctx, user.GetSteamID(), req.GetMessageId()); err != nil && !errors.Is(err, database.ErrNoResult) {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &emptypb.Empty{}, nil
}
