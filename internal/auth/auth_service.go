package auth

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/auth/v1/authv1connect"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Service struct{}

func NewService(roleAuth *rpc.RoleAuth, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := authv1connect.NewAuthServiceHandler(Service{}, option...)

	authMiddleware.UserRoute(authv1connect.AuthServiceLogoutProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_LOGIN))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s Service) Logout(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	user := rpc.UserInfoFromCtx(ctx)
	slog.Info("User logged out", slog.String("user", user.Name), slog.String("steamId", user.SteamID.String()))

	return &emptypb.Empty{}, nil
}
