package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban"
	personDomain "github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const (
	TokenDuration         = time.Hour * 24 * 31
	FingerprintCookieName = "fingerprint"
)

var (
	ErrExpired             = errors.New("expired")
	ErrAuthentication      = errors.New("auth invalid")
	ErrSaveToken           = errors.New("failed to save new createRefresh token")
	ErrSignToken           = errors.New("failed create signed string")
	ErrAuthHeader          = errors.New("failed to bind auth header")
	ErrMalformedAuthHeader = errors.New("invalid auth header format")
	ErrCookieKeyMissing    = errors.New("cookie key missing, cannot generate token")
	ErrCreateToken         = errors.New("failed to create new Access token")
	ErrClientIP            = errors.New("failed to parse IP")
)

func FingerprintHash(fingerprint string) string {
	hasher := sha256.New()

	return hex.EncodeToString(hasher.Sum([]byte(fingerprint)))
}

type UserTokens struct {
	Access      string `json:"access"`
	Fingerprint string `json:"fingerprint"`
}

type UserAuthClaims struct {
	jwt.RegisteredClaims

	// user context to prevent side-jacking
	// https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html#token-sidejacking
	Fingerprint string `json:"fingerprint"`
}

type ServerAuthClaims struct {
	jwt.RegisteredClaims

	ServerID int `json:"server_id"`
}

type PersonAuth struct {
	PersonAuthID int64           `json:"person_auth_id"`
	SteamID      steamid.SteamID `json:"steam_id"`
	IPAddr       net.IP          `json:"ip_addr"`
	AccessToken  string          `json:"access_token"`
	CreatedOn    time.Time       `json:"created_on"`
}

func NewPersonAuth(sid64 steamid.SteamID, addr net.IP, accessToken string) PersonAuth {
	return PersonAuth{
		PersonAuthID: 0,
		SteamID:      sid64,
		IPAddr:       addr,
		AccessToken:  accessToken,
		CreatedOn:    time.Now(),
	}
}

const CtxKeyUserProfile = "user_profile"

type Authentication struct {
	auth      Repository
	persons   *person.Persons
	bans      ban.Bans
	servers   *servers.Servers
	sentryDSN string
	siteName  string
	cookieKey string
}

func NewAuthentication(repository Repository, siteName string, cookieKey string, persons *person.Persons,
	bans ban.Bans, servers *servers.Servers, sentryDSN string,
) *Authentication {
	return &Authentication{
		auth:      repository,
		persons:   persons,
		bans:      bans,
		servers:   servers,
		sentryDSN: sentryDSN,
		siteName:  siteName,
		cookieKey: cookieKey,
	}
}

func (u *Authentication) DeletePersonAuth(ctx context.Context, authID int64) error {
	return u.auth.DeletePersonAuth(ctx, authID)
}

func (u *Authentication) GetPersonAuthByRefreshToken(ctx context.Context, token string, auth *PersonAuth) error {
	return u.auth.GetPersonAuthByFingerprint(ctx, token, auth)
}

// MakeToken generates new jwt auth tokens
// fingerprint is a random string used to prevent side-jacking.
func (u *Authentication) MakeToken(ctx *gin.Context, cookieKey string, sid steamid.SteamID) (UserTokens, error) {
	if cookieKey == "" {
		return UserTokens{}, ErrCookieKeyMissing
	}

	fingerprint := stringutil.SecureRandomString(40)

	accessToken, errAccess := u.NewUserToken(sid, cookieKey, fingerprint, TokenDuration)
	if errAccess != nil {
		return UserTokens{}, errors.Join(errAccess, ErrCreateToken)
	}

	ipAddr := net.ParseIP(ctx.ClientIP())
	if ipAddr == nil {
		return UserTokens{}, ErrClientIP
	}

	personAuth := NewPersonAuth(sid, ipAddr, accessToken)

	if saveErr := u.auth.SavePersonAuth(ctx, &personAuth); saveErr != nil {
		return UserTokens{}, errors.Join(saveErr, ErrSaveToken)
	}

	return UserTokens{Access: accessToken, Fingerprint: fingerprint}, nil
}

func (u *Authentication) Middleware(level permission.Privilege) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var token string

		hdrToken, errToken := u.TokenFromHeader(ctx, level == permission.Guest)
		if errToken != nil || hdrToken == "" { //nolint:nestif
			ctx.Set(CtxKeyUserProfile, personDomain.Core{PermissionLevel: permission.Guest, Name: "Guest"})
		} else {
			token = hdrToken

			if level >= permission.Guest {
				fingerprint, errFingerprint := ctx.Cookie("fingerprint")
				if errFingerprint != nil {
					slog.Error("Failed to load fingerprint cookie", slog.String("error", errFingerprint.Error()))
					ctx.AbortWithStatus(http.StatusForbidden)

					return
				}

				sid, errFromToken := u.Sid64FromJWTToken(token, u.cookieKey, fingerprint)
				if errFromToken != nil {
					if errors.Is(errFromToken, ErrExpired) {
						ctx.AbortWithStatus(http.StatusUnauthorized)

						return
					}

					slog.Error("Failed to load sid from access token", slog.String("error", errFromToken.Error()))
					ctx.AbortWithStatus(http.StatusForbidden)

					return
				}

				u.loginSID(ctx, level, sid)
			} else {
				ctx.Set(CtxKeyUserProfile, personDomain.Core{PermissionLevel: permission.Guest, Name: "Guest"})
			}
		}

		ctx.Next()
	}
}

func (u *Authentication) TokenFromQuery(ctx *gin.Context) (string, error) {
	token, found := ctx.GetQuery("token")
	if !found || token == "" {
		ctx.AbortWithStatus(http.StatusForbidden)

		return "", ErrMalformedAuthHeader
	}

	return token, nil
}

func (u *Authentication) MiddlewareWS(level permission.Privilege) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var token string

		queryToken, errToken := u.TokenFromQuery(ctx)
		if errToken != nil || queryToken == "" {
			ctx.Set(CtxKeyUserProfile, personDomain.Core{PermissionLevel: permission.Guest, Name: "Guest"})
			ctx.Next()

			return
		}

		token = queryToken

		if level < permission.Guest {
			ctx.Set(CtxKeyUserProfile, personDomain.Core{PermissionLevel: permission.Guest, Name: "Guest"})
			ctx.Next()

			return
		}

		sid, errFromToken := u.Sid64FromJWTTokenNoFP(token, u.cookieKey)
		if errFromToken != nil {
			if errors.Is(errFromToken, ErrExpired) {
				ctx.AbortWithStatus(http.StatusUnauthorized)

				return
			}

			slog.Error("Failed to load sid from access token", slog.String("error", errFromToken.Error()))
			ctx.AbortWithStatus(http.StatusForbidden)

			return
		}

		u.loginSID(ctx, level, sid)

		ctx.Next()
	}
}

func (u *Authentication) MakeGetTokenKey(cookieKey string) func(_ *jwt.Token) (any, error) {
	return func(_ *jwt.Token) (any, error) {
		return []byte(cookieKey), nil
	}
}

func (u *Authentication) NewUserToken(steamID steamid.SteamID, cookieKey string, fingerPrint string, validDuration time.Duration) (string, error) {
	nowTime := time.Now()

	claims := UserAuthClaims{
		Fingerprint: FingerprintHash(fingerPrint),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    u.siteName,
			Subject:   steamID.String(),
			ExpiresAt: jwt.NewNumericDate(nowTime.Add(validDuration)),
			IssuedAt:  jwt.NewNumericDate(nowTime),
			NotBefore: jwt.NewNumericDate(nowTime),
		},
	}
	tokenWithClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, errSigned := tokenWithClaims.SignedString([]byte(cookieKey))

	if errSigned != nil {
		return "", errors.Join(errSigned, ErrSignToken)
	}

	return signedToken, nil
}

type authHeader struct {
	Authorization string `header:"Authorization"`
}

func (u *Authentication) TokenFromHeader(ctx *gin.Context, emptyOK bool) (string, error) {
	hdr := authHeader{}
	if errBind := ctx.ShouldBindHeader(&hdr); errBind != nil {
		return "", errors.Join(errBind, ErrAuthHeader)
	}

	pcs := strings.Split(hdr.Authorization, " ")
	if len(pcs) != 2 || pcs[1] == "" {
		if emptyOK {
			return "", nil
		}

		ctx.AbortWithStatus(http.StatusForbidden)

		return "", ErrMalformedAuthHeader
	}

	return pcs[1], nil
}

func (u *Authentication) Sid64FromJWTToken(token string, cookieKey string, fingerprint string) (steamid.SteamID, error) {
	claims := &UserAuthClaims{}

	tkn, errParseClaims := jwt.ParseWithClaims(token, claims, u.MakeGetTokenKey(cookieKey))
	if errParseClaims != nil {
		if errors.Is(errParseClaims, jwt.ErrSignatureInvalid) {
			return steamid.SteamID{}, ErrAuthentication
		}

		if errors.Is(errParseClaims, jwt.ErrTokenExpired) {
			return steamid.SteamID{}, ErrExpired
		}

		return steamid.SteamID{}, ErrAuthentication
	}

	if !tkn.Valid {
		return steamid.SteamID{}, ErrAuthentication
	}

	if claims.Fingerprint != FingerprintHash(fingerprint) {
		slog.Error("Invalid cookie fingerprint, token rejected")

		return steamid.SteamID{}, ErrAuthentication
	}

	sid := steamid.New(claims.Subject)
	if !sid.Valid() {
		return steamid.SteamID{}, ErrAuthentication
	}

	return sid, nil
}

func (u *Authentication) Sid64FromJWTTokenNoFP(token string, cookieKey string) (steamid.SteamID, error) {
	claims := &UserAuthClaims{}

	tkn, errParseClaims := jwt.ParseWithClaims(token, claims, u.MakeGetTokenKey(cookieKey))
	if errParseClaims != nil {
		if errors.Is(errParseClaims, jwt.ErrSignatureInvalid) {
			return steamid.SteamID{}, ErrAuthentication
		}

		if errors.Is(errParseClaims, jwt.ErrTokenExpired) {
			return steamid.SteamID{}, ErrExpired
		}

		return steamid.SteamID{}, ErrAuthentication
	}

	if !tkn.Valid {
		return steamid.SteamID{}, ErrAuthentication
	}

	sid := steamid.New(claims.Subject)
	if !sid.Valid() {
		return steamid.SteamID{}, ErrAuthentication
	}

	return sid, nil
}

func (u *Authentication) loginSID(ctx *gin.Context, level permission.Privilege, steamID steamid.SteamID) {
	loggedInPerson, errGetPerson := u.persons.BySteamID(ctx, steamID)
	if errGetPerson != nil {
		slog.Error("Failed to load person during auth", slog.String("error", errGetPerson.Error()))
		ctx.AbortWithStatus(http.StatusForbidden)

		return
	}
	if u.sentryDSN != "" {
		sentry.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetUser(sentry.User{
				ID:        loggedInPerson.SteamID.String(),
				IPAddress: ctx.ClientIP(),
				Username:  loggedInPerson.PersonaName,
			})
		})
	}
	if level > loggedInPerson.PermissionLevel {
		ctx.AbortWithStatus(http.StatusForbidden)

		return
	}

	bannedPerson, errBan := u.bans.QueryOne(ctx, ban.QueryOpts{TargetID: steamID, EvadeOk: true})
	if errBan != nil && !errors.Is(errBan, ban.ErrBanDoesNotExist) {
		slog.Error("Failed to fetch authed user ban", slog.String("error", errBan.Error()))
	}

	profile := personDomain.Core{
		SteamID:         loggedInPerson.SteamID,
		PermissionLevel: loggedInPerson.PermissionLevel,
		DiscordID:       loggedInPerson.DiscordID,
		PatreonID:       loggedInPerson.PatreonID,
		Name:            loggedInPerson.PersonaName,
		Avatarhash:      loggedInPerson.AvatarHash,
		BanID:           bannedPerson.BanID,
	}

	ctx.Set(CtxKeyUserProfile, profile)

	if u.sentryDSN != "" {
		if hub := sentrygin.GetHubFromContext(ctx); hub != nil {
			hub.WithScope(func(scope *sentry.Scope) {
				scope.SetUser(sentry.User{
					ID:        steamID.String(),
					IPAddress: ctx.ClientIP(),
					Username:  loggedInPerson.PersonaName,
				})
			})
		}
	}
}
