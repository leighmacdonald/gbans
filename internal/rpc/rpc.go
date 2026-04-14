package rpc

import (
	"context"

	"connectrpc.com/authn"
	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type UserInfo struct {
	SteamID    steamid.SteamID      `json:"steam_id"`
	AvatarHash string               `json:"avatar_hash"`
	Name       string               `json:"name"`
	Privilege  permission.Privilege `json:"privilege"`
}

func (u UserInfo) HasPermission(privilege permission.Privilege) bool {
	return u.Privilege >= privilege
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
