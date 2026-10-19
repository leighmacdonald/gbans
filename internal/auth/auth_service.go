package auth

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/auth/v1/authv1connect"
	"github.com/leighmacdonald/gbans/internal/opengraph"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Service struct{}

func NewService(openGraph *opengraph.Repository, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := authv1connect.NewAuthServiceHandler(Service{}, option...)

	openGraph.Register("/login", opengraph.Static("Login", "Sign in to your account."))
	openGraph.Register("/login/success", opengraph.Static("Login Successful", "You have signed in."))
	openGraph.Register("/logout", opengraph.Static("Logout", "Sign out of your account."))
	openGraph.Register("/page-not-found",
		opengraph.Static("Page Not Found", "The page you are looking for does not exist."))
	openGraph.Register("/permission", opengraph.Static("Permission Denied", "You do not have permission to view this page."))

	authMiddleware.AuthedRoute(authv1connect.AuthServiceLogoutProcedure)

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s Service) Logout(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	user := rpc.UserInfoFromCtx(ctx)
	slog.Info("User logged out", slog.String("user", user.Name), slog.String("steamId", user.SteamID.String()))

	return &emptypb.Empty{}, nil
}
