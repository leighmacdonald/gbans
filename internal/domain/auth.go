package domain

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"time"
)

type AuthUsecase interface {
	NewUserToken(steamID steamid.SID64, cookieKey string, userContext string, validDuration time.Duration) (string, error)
	Sid64FromJWTToken(token string, cookieKey string) (steamid.SID64, error)
	AuthMiddleware(level Privilege) gin.HandlerFunc
	AuthServerMiddleWare() gin.HandlerFunc
	MakeTokens(ctx *gin.Context, cookieKey string, sid steamid.SID64, createRefresh bool) (UserTokens, error)
}

type UserTokens struct {
	access      string
	refresh     string
	fingerprint string
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
