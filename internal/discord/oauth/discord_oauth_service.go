package discordoauth

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/database"
	v1 "github.com/leighmacdonald/gbans/internal/discord/oauth/v1"
	"github.com/leighmacdonald/gbans/internal/discord/oauth/v1/oauthv1connect"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DiscordService struct {
	discordOAuth DiscordOAuth
}

func NewService(discord DiscordOAuth, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := oauthv1connect.NewDiscordOAuthServiceHandler(&DiscordService{discordOAuth: discord}, option...)
	authMiddleware.UserRoute(oauthv1connect.DiscordOAuthServiceLoginProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(oauthv1connect.DiscordOAuthServiceLogoutProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(oauthv1connect.DiscordOAuthServiceProfileProcedure, rpc.WithMinPermissions(permission.User))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s *DiscordService) Login(ctx context.Context, _ *emptypb.Empty) (*v1.LoginResponse, error) {
	currentUser := rpc.UserInfoFromCtx(ctx)
	sid := currentUser.GetSteamID()

	loginURL, errURL := s.discordOAuth.CreateStatefulLoginURL(sid)
	if errURL != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, rpc.ErrBadRequest)
	}

	slog.Debug("User tried to connect discord", slog.String("sid", sid.String()))

	return &v1.LoginResponse{LoginUrl: &loginURL}, nil
}

func (s *DiscordService) Logout(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	user := rpc.UserInfoFromCtx(ctx)

	errUser := s.discordOAuth.Logout(ctx, user.GetSteamID())
	if errUser != nil {
		if errors.Is(errUser, database.ErrNoResult) {
			return &emptypb.Empty{}, nil
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &emptypb.Empty{}, nil
}

func (s *DiscordService) Profile(ctx context.Context, _ *emptypb.Empty) (*v1.ProfileResponse, error) {
	user := rpc.UserInfoFromCtx(ctx)
	discord, errUser := s.discordOAuth.GetUserDetail(ctx, user.GetSteamID())
	if errUser != nil {
		if errors.Is(errUser, database.ErrNoResult) {
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.ProfileResponse{DiscordProfile: &v1.DiscordProfile{
		SteamId: new(discord.SteamID.Int64()),
		Id:      &discord.ID,
		Avatar:  &discord.Avatar,
		// AvatarDecoration: &discord.AvatarDecoration,
		Discriminator: &discord.Discriminator,
		Flags:         &discord.Flags,
		// Banner:           &discord.Banner,
		// BannerColour:     &discord.Banner,
		// AccentColour:     &discord.AccentColor,
		Locale:      &discord.Locale,
		MfaEnabled:  &discord.MfaEnabled,
		PremiumType: &discord.PremiumType,
		CreatedOn:   timestamppb.New(discord.CreatedOn),
		UpdatedOn:   timestamppb.New(discord.UpdatedOn),
	}}, nil
}
