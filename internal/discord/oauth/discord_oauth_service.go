package discordoauth

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/database"
	v1 "github.com/leighmacdonald/gbans/internal/discord/v1"
	"github.com/leighmacdonald/gbans/internal/discord/v1/oauthv1connect"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DiscordService struct {
	oauthv1connect.UnimplementedDiscordOAuthServiceHandler

	discord DiscordOAuth
}

func (s *DiscordService) Login(ctx context.Context, _ *emptypb.Empty) (*v1.LoginResponse, error) {
	currentUser, _ := rpc.UserInfoFromCtx(ctx)
	sid := currentUser.GetSteamID()

	loginURL, errURL := s.discord.CreateStatefulLoginURL(sid)
	if errURL != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, rpc.ErrBadRequest)
	}

	slog.Debug("User tried to connect discord", slog.String("sid", sid.String()))

	return &v1.LoginResponse{LoginUrl: &loginURL}, nil
}

func (s *DiscordService) Logout(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)

	errUser := s.discord.Logout(ctx, user.GetSteamID())
	if errUser != nil {
		if errors.Is(errUser, database.ErrNoResult) {
			return &emptypb.Empty{}, nil
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &emptypb.Empty{}, nil
}

func (s *DiscordService) Profile(ctx context.Context, _ *emptypb.Empty) (*v1.ProfileResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	discord, errUser := s.discord.GetUserDetail(ctx, user.GetSteamID())
	if errUser != nil {
		if errors.Is(errUser, database.ErrNoResult) {
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.ProfileResponse{DiscordProfile: &v1.DiscordProfile{
		SteamId: ptr.To(discord.SteamID.Int64()),
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
