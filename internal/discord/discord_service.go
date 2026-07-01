package discord

import (
	"context"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	discordv1 "github.com/leighmacdonald/gbans/internal/discord/v1"
	"github.com/leighmacdonald/gbans/internal/discord/v1/discordv1connect"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

func NewService(connection Connection, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := discordv1connect.NewDiscordServiceHandler(Service{connection: connection}, option...)
	authMiddleware.UserRoute(discordv1connect.DiscordServiceSeedRoleIDsProcedure, rpc.WithMinPermissions(permission.Moderator))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

type Service struct {
	discordv1connect.UnimplementedDiscordServiceHandler

	connection Connection
}

func (s Service) SeedRoleIDs(ctx context.Context, req *emptypb.Empty) (*discordv1.SeedRoleIDsResponse, error) {
	roles, err := s.connection.Roles()
	if err != nil {
		return nil, err
	}

	resp := &discordv1.SeedRoleIDsResponse{
		Roles: make([]*discordv1.Role, len(roles)),
	}
	for idx := range roles {
		resp.Roles[idx] = &discordv1.Role{
			Id:           &roles[idx].ID,
			Name:         &roles[idx].Name,
			Managed:      &roles[idx].Managed,
			Mentionable:  &roles[idx].Mentionable,
			Hoist:        &roles[idx].Hoist,
			Color:        new(int32(roles[idx].Color)),
			Position:     new(int32(roles[idx].Position)),
			Permissions:  &roles[idx].Permissions,
			Icon:         &roles[idx].Icon,
			UnicodeEmoji: &roles[idx].UnicodeEmoji,
			Flags:        new(int32(roles[idx].Flags)),
		}
	}

	return resp, nil
}
