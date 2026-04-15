package rpc

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/authn"
	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	ErrBadRequest = errors.New("invalid request")
	ErrInternal   = errors.New("internal server error")
	ErrNotFound   = errors.New("entity not found")
	ErrPermission = errors.New("permission denied")
	ErrExists     = errors.New("entity already exists")
)

type UserInfo struct {
	SteamID    steamid.SteamID      `json:"steam_id"`
	AvatarHash person.Avatar        `json:"avatar_hash"`
	Name       string               `json:"name"`
	Privilege  permission.Privilege `json:"privilege"`
}

func (u UserInfo) Path() string {
	return fmt.Sprintf("https://steamcommunity.com/profiles/%d", u.SteamID.Int64())
}

func (u UserInfo) HasPermission(privilege permission.Privilege) bool {
	return u.Privilege >= privilege
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

func (u UserInfo) GetPrivilege() permission.Privilege {
	return u.Privilege
}

func (u UserInfo) GetAvatar() person.Avatar {
	if u.AvatarHash == "" {
		return "fef49e7fa7e1997310d705b2a6158ff8dc1cdfeb"
	}

	return u.AvatarHash
}

func UserInfoFromCtx(ctx context.Context) (*UserInfo, bool) {
	user, ok := authn.GetInfo(ctx).(UserInfo)
	if !ok {
		return nil, false
	}

	return &user, true
}

func UserInfoFromCtxWithCheck(ctx context.Context, privilege permission.Privilege) (*UserInfo, error) {
	user, authed := UserInfoFromCtx(ctx)

	if !authed || !user.HasPermission(privilege) {
		return nil, connect.NewError(connect.CodePermissionDenied, permission.ErrDenied)
	}

	return user, nil
}

func FromRPC(filter *internal.Filter) query.Filter {
	return query.Filter{
		Offset:  filter.GetOffset(),
		Limit:   filter.GetLimit(),
		Desc:    filter.GetDesc(),
		OrderBy: filter.GetOrderBy(),
	}
}
