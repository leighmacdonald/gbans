package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"connectrpc.com/authn"
	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type UserInfo struct {
	SteamID   steamid.SteamID      `json:"steam_id"`
	Privilege permission.Privilege `json:"privilege"`
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

func fingerprintFromRequest(req *http.Request) (string, error) {
	fp, errFP := req.Cookie("fingerprint")
	if errFP != nil {
		return "", authn.Errorf("invalid fingerprint cookie")
	}

	fingerprint := fp.String()
	if fingerprint == "" {
		return "", authn.Errorf("empty fingerprint")
	}

	return fingerprint, nil
}

func claimsFromRequest(req *http.Request) (*UserAuthClaims, error) {
	fingerprint, errFP := fingerprintFromRequest(req)
	if errFP != nil {
		return nil, errFP
	}

	token, ok := authn.BearerToken(req)
	if !ok {
		// TODO Make sure procedure is allowed
		return nil, authn.Errorf("invalid authorization")
	}

	claims := UserAuthClaims{}
	tkn, errParseClaims := jwt.ParseWithClaims(token, &claims, makeGetTokenKey(fingerprint))
	if errParseClaims != nil {
		if errors.Is(errParseClaims, jwt.ErrSignatureInvalid) {
			return nil, authn.Errorf("invalid authorization")
		}

		if errors.Is(errParseClaims, jwt.ErrTokenExpired) {
			return nil, authn.Errorf("expired authorization")
		}

		return nil, authn.Errorf("invalid authorization")
	}

	if !tkn.Valid {
		return nil, authn.Errorf("invalid token")
	}

	if claims.Fingerprint != FingerprintHash(fingerprint) {
		slog.Error("Invalid cookie fingerprint, token rejected")

		return nil, authn.Errorf("invalid token")
	}

	return &claims, nil
}

type RouteAuthFn = func(req *http.Request) bool

type Middleware struct {
	allowList map[string]RouteAuthFn
}

func (m *Middleware) AuthedRoute(procedure string, authFunc RouteAuthFn) {
	m.allowList[procedure] = authFunc
}

func (m *Middleware) Authenticate(_ context.Context, req *http.Request) (any, error) {
	procedure, found := authn.InferProcedure(req.URL)
	if !found {
		return nil, nil
	}

	var info UserInfo

	claims, errToken := claimsFromRequest(req)
	if errToken != nil {
		return info, errToken
	}

	sid := steamid.New(claims.Subject)
	if !sid.Valid() {
		return info, authn.Errorf("invalid authorization")
	}

	info.SteamID = sid
	info.Privilege = claims.Privilege

	authFn, ok := m.allowList[procedure]
	if !ok {
		return info, authn.Errorf("unknown procedure")
	}

	if !authFn(req) {
		return info, authn.Errorf("unauthorized")
	}

	return info, nil
}
