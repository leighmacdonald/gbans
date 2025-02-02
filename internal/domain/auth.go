package domain

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const (
	AuthTokenDuration     = time.Hour * 24 * 31
	FingerprintCookieName = "fingerprint"
)

type AuthRepository interface {
	SavePersonAuth(ctx context.Context, auth *PersonAuth) error
	DeletePersonAuth(ctx context.Context, authID int64) error
	PrunePersonAuth(ctx context.Context) error
	GetPersonAuthByFingerprint(ctx context.Context, fingerprint string, auth *PersonAuth) error
}
type AuthUsecase interface {
	DeletePersonAuth(ctx context.Context, authID int64) error
	NewUserToken(steamID steamid.SteamID, cookieKey string, userContext string, validDuration time.Duration) (string, error)
	Sid64FromJWTToken(token string, cookieKey string, fingerprint string) (steamid.SteamID, error)
	AuthMiddleware(level Privilege) gin.HandlerFunc
	AuthMiddlewareWS(level Privilege) gin.HandlerFunc
	AuthServerMiddleWare() gin.HandlerFunc
	MakeToken(ctx *gin.Context, cookieKey string, sid steamid.SteamID) (UserTokens, error)
	TokenFromHeader(ctx *gin.Context, emptyOK bool) (string, error)
	MakeGetTokenKey(cookieKey string) func(_ *jwt.Token) (any, error)
	GetPersonAuthByRefreshToken(ctx context.Context, token string, auth *PersonAuth) error
}

type UserTokens struct {
	Access      string `json:"access"`
	Fingerprint string `json:"fingerprint"`
}

type UserAuthClaims struct {
	// user context to prevent side-jacking
	// https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html#token-sidejacking
	Fingerprint string `json:"fingerprint"`
	jwt.RegisteredClaims
}

type ServerAuthClaims struct {
	ServerID int `json:"server_id"`
	// A random string which is used to fingerprint and prevent side-jacking
	jwt.RegisteredClaims
}
