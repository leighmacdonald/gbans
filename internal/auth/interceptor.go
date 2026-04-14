package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"connectrpc.com/authn"
	"github.com/golang-jwt/jwt/v5"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

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

type RouteAuthFn = func(ctx context.Context, req *http.Request) bool

func NewMiddleware() *Middleware {
	return &Middleware{
		allowList: map[string]RouteAuthFn{},
	}
}

type Middleware struct {
	allowList map[string]RouteAuthFn
}

func (m *Middleware) AuthedRoute(procedure string, authFunc RouteAuthFn) {
	m.allowList[procedure] = authFunc
}

func (m *Middleware) Authenticate(ctx context.Context, req *http.Request) (any, error) {
	procedure, found := authn.InferProcedure(req.URL)
	if !found {
		return nil, nil
	}

	var info rpc.UserInfo

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

	if !authFn(ctx, req) {
		return info, authn.Errorf("unauthorized")
	}

	return info, nil
}

func WithMinPermissions(permission permission.Privilege) RouteAuthFn {
	return func(ctx context.Context, req *http.Request) bool {
		info, err := rpc.UserInfoFromCtxWithCheck(ctx, permission)
		if err != nil {
			return false
		}

		return info.HasPermission(permission)
	}
}
