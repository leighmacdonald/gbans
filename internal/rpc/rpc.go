package rpc

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/authn"
	"connectrpc.com/connect"
	"connectrpc.com/validate"
	"github.com/leighmacdonald/gbans/internal/database/query"
	v1 "github.com/leighmacdonald/gbans/internal/database/query/v1"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stoewer/go-strcase"
)

var (
	ErrBadRequest  = errors.New("invalid request")
	ErrInternal    = errors.New("internal server error")
	ErrNotFound    = errors.New("entity not found")
	ErrPermission  = errors.New("permission denied")
	ErrExists      = errors.New("entity already exists")
	ErrNoProcedure = errors.New("no matching procedure for route")
)

type Service struct {
	Pattern string
	Handler http.Handler
}

type UserInfo struct {
	SteamID     steamid.SteamID
	AvatarHash  person.Avatar
	Name        string
	Permissions []rolesv1.Permission
}

func (u UserInfo) Path() string {
	return fmt.Sprintf("https://steamcommunity.com/profiles/%d", u.SteamID.Int64())
}

func (u UserInfo) GetSteamID() steamid.SteamID {
	return u.SteamID
}

func (u UserInfo) GetName() string {
	if u.Name == "" {
		return u.SteamID.String()
	}

	return u.Name
}

func (u UserInfo) GetAvatar() person.Avatar {
	if u.AvatarHash == "" {
		return "fef49e7fa7e1997310d705b2a6158ff8dc1cdfeb"
	}

	return u.AvatarHash
}

func (u UserInfo) GetPermissions() []rolesv1.Permission {
	return u.Permissions
}

type ServerAuthenticator interface {
	GetByPassword(ctx context.Context, password string) (int32, string, error)
}

func NewServerAuthenticator() ServerRouteAuthFn {
	return func(_ context.Context, _ *http.Request, server ServerInfo) bool {
		return server.ServerID > 0
	}
}

type ServerInfo struct {
	ServerID   int32
	ServerName string
}

// ServerInfoFromCtx retreives the ServerInfo struct that is set with the ServerAuthenticator middleware. Returns nil
// for a unauthenticated request.
func ServerInfoFromCtx(ctx context.Context) *ServerInfo {
	server, ok := authn.GetInfo(ctx).(ServerInfo)
	if !ok {
		return nil
	}

	return &server
}

// UserInfoFromCtx retreives the ServerInfo struct that is set with the UserAuthenticator middleware. Returns nil
// for a unauthenticated request.
func UserInfoFromCtx(ctx context.Context) *UserInfo {
	user, ok := authn.GetInfo(ctx).(UserInfo)
	if !ok {
		return nil
	}

	return &user
}

func FromRPC(filter *v1.Filter) query.Filter {
	return query.Filter{
		Offset:  filter.GetOffset(),
		Limit:   filter.GetLimit(),
		Desc:    filter.GetDesc(),
		OrderBy: strcase.SnakeCase(filter.GetOrderBy()),
	}
}

func CreateInterceptors() connect.Option {
	return connect.WithInterceptors(validate.NewInterceptor())
}
