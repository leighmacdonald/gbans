package auth

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/auth/v1/authv1connect"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Service struct{}

func NewService(authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := authv1connect.NewAuthServiceHandler(Service{}, option...)

	authMiddleware.UserRoute(authv1connect.AuthServiceLogoutProcedure, rpc.WithMinPermissions(permission.User))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s Service) Logout(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	slog.Info("User logged out", slog.String("user", user.Name), slog.String("steamId", user.SteamID.String()))

	return &emptypb.Empty{}, nil
}
