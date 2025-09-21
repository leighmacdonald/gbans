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
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const (
	AuthTokenDuration     = time.Hour * 24 * 31
	FingerprintCookieName = "fingerprint"
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

const ctxKeyUserProfile = "user_profile"

type Authentication struct {
	auth      Repository
	config    *config.Configuration
	persons   *person.Persons
	bans      ban.Bans
	servers   servers.Servers
	sentryDSN string
}

func NewAuthentication(repository Repository, config *config.Configuration, persons *person.Persons,
	bans ban.Bans, servers servers.Servers, sentryDSN string,
) *Authentication {
	return &Authentication{
		auth:      repository,
		config:    config,
		persons:   persons,
		bans:      bans,
		servers:   servers,
		sentryDSN: sentryDSN,
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
		return UserTokens{}, domain.ErrCookieKeyMissing
	}

	fingerprint := stringutil.SecureRandomString(40)

	accessToken, errAccess := u.NewUserToken(sid, cookieKey, fingerprint, AuthTokenDuration)
	if errAccess != nil {
		return UserTokens{}, errors.Join(errAccess, domain.ErrCreateToken)
	}

	ipAddr := net.ParseIP(ctx.ClientIP())
	if ipAddr == nil {
		return UserTokens{}, domain.ErrClientIP
	}

	personAuth := NewPersonAuth(sid, ipAddr, accessToken)

	if saveErr := u.auth.SavePersonAuth(ctx, &personAuth); saveErr != nil {
		return UserTokens{}, errors.Join(saveErr, domain.ErrSaveToken)
	}

	return UserTokens{Access: accessToken, Fingerprint: fingerprint}, nil
}

func (u *Authentication) Middleware(level permission.Privilege) gin.HandlerFunc {
	cookieKey := u.config.Config().HTTPCookieKey

	return func(ctx *gin.Context) {
		var token string

		hdrToken, errToken := u.TokenFromHeader(ctx, level == permission.Guest)
		if errToken != nil || hdrToken == "" {
			ctx.Set(ctxKeyUserProfile, domain.PersonCore{PermissionLevel: permission.Guest, Name: "Guest"})
		} else {
			token = hdrToken

			if level >= permission.Guest {
				fingerprint, errFingerprint := ctx.Cookie("fingerprint")
				if errFingerprint != nil {
					slog.Error("Failed to load fingerprint cookie", log.ErrAttr(errFingerprint))
					ctx.AbortWithStatus(http.StatusForbidden)

					return
				}

				sid, errFromToken := u.Sid64FromJWTToken(token, cookieKey, fingerprint)
				if errFromToken != nil {
					if errors.Is(errFromToken, domain.ErrExpired) {
						ctx.AbortWithStatus(http.StatusUnauthorized)

						return
					}

					slog.Error("Failed to load sid from access token", log.ErrAttr(errFromToken))
					ctx.AbortWithStatus(http.StatusForbidden)

					return
				}

				loggedInPerson, errGetPerson := u.persons.GetPersonBySteamID(ctx, nil, sid)
				if errGetPerson != nil {
					slog.Error("Failed to load person during auth", log.ErrAttr(errGetPerson))
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

				bannedPerson, errBan := u.bans.QueryOne(ctx, ban.QueryOpts{
					TargetID: sid,
					EvadeOk:  true,
				})
				if errBan != nil {
					if !errors.Is(errBan, database.ErrNoResult) {
						slog.Error("Failed to fetch authed user ban", log.ErrAttr(errBan))
					}
				}

				profile := domain.PersonCore{
					SteamID:         loggedInPerson.SteamID,
					PermissionLevel: loggedInPerson.PermissionLevel,
					DiscordID:       loggedInPerson.DiscordID,
					PatreonID:       loggedInPerson.PatreonID,
					Name:            loggedInPerson.PersonaName,
					Avatarhash:      loggedInPerson.AvatarHash,
					BanID:           bannedPerson.BanID,
				}

				ctx.Set(ctxKeyUserProfile, profile)

				if u.sentryDSN != "" {
					if hub := sentrygin.GetHubFromContext(ctx); hub != nil {
						hub.WithScope(func(scope *sentry.Scope) {
							scope.SetUser(sentry.User{
								ID:        sid.String(),
								IPAddress: ctx.ClientIP(),
								Username:  loggedInPerson.PersonaName,
							})
						})
					}
				}
			} else {
				ctx.Set(ctxKeyUserProfile, domain.PersonCore{PermissionLevel: permission.Guest, Name: "Guest"})
			}
		}

		ctx.Next()
	}
}

func (u *Authentication) TokenFromQuery(ctx *gin.Context) (string, error) {
	token, found := ctx.GetQuery("token")
	if !found || token == "" {
		ctx.AbortWithStatus(http.StatusForbidden)

		return "", domain.ErrMalformedAuthHeader
	}

	return token, nil
}

func (u *Authentication) MiddlewareWS(level permission.Privilege) gin.HandlerFunc {
	cookieKey := u.config.Config().HTTPCookieKey

	return func(ctx *gin.Context) {
		var token string

		queryToken, errToken := u.TokenFromQuery(ctx)
		if errToken != nil || queryToken == "" {
			ctx.Set(ctxKeyUserProfile, domain.PersonCore{PermissionLevel: permission.Guest, Name: "Guest"})
		} else {
			token = queryToken

			if level >= permission.Guest {
				sid, errFromToken := u.Sid64FromJWTTokenNoFP(token, cookieKey)
				if errFromToken != nil {
					if errors.Is(errFromToken, domain.ErrExpired) {
						ctx.AbortWithStatus(http.StatusUnauthorized)

						return
					}

					slog.Error("Failed to load sid from access token", log.ErrAttr(errFromToken))
					ctx.AbortWithStatus(http.StatusForbidden)

					return
				}

				loggedInPerson, errGetPerson := u.persons.GetPersonBySteamID(ctx, nil, sid)
				if errGetPerson != nil {
					slog.Error("Failed to load person during auth", log.ErrAttr(errGetPerson))
					ctx.AbortWithStatus(http.StatusForbidden)

					return
				}

				if level > loggedInPerson.PermissionLevel {
					ctx.AbortWithStatus(http.StatusForbidden)

					return
				}

				bannedPerson, errBan := u.bans.QueryOne(ctx, ban.QueryOpts{
					TargetID: sid,
					EvadeOk:  true,
				})
				if errBan != nil {
					if !errors.Is(errBan, database.ErrNoResult) {
						slog.Error("Failed to fetch authed user ban", log.ErrAttr(errBan))
					}
				}

				profile := domain.PersonCore{
					SteamID:         loggedInPerson.SteamID,
					PermissionLevel: loggedInPerson.PermissionLevel,
					DiscordID:       loggedInPerson.DiscordID,
					PatreonID:       loggedInPerson.PatreonID,
					Name:            loggedInPerson.PersonaName,
					Avatarhash:      loggedInPerson.AvatarHash,
					BanID:           bannedPerson.BanID,
				}

				ctx.Set(ctxKeyUserProfile, profile)

				if u.sentryDSN != "" {
					if hub := sentrygin.GetHubFromContext(ctx); hub != nil {
						hub.WithScope(func(scope *sentry.Scope) {
							scope.SetUser(sentry.User{
								ID:        sid.String(),
								IPAddress: ctx.ClientIP(),
								Username:  loggedInPerson.PersonaName,
							})
						})
					}
				}
			} else {
				ctx.Set(ctxKeyUserProfile, domain.PersonCore{PermissionLevel: permission.Guest, Name: "Guest"})
			}
		}

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
	conf := u.config.Config()
	claims := UserAuthClaims{
		Fingerprint: FingerprintHash(fingerPrint),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    conf.General.SiteName,
			Subject:   steamID.String(),
			ExpiresAt: jwt.NewNumericDate(nowTime.Add(validDuration)),
			IssuedAt:  jwt.NewNumericDate(nowTime),
			NotBefore: jwt.NewNumericDate(nowTime),
		},
	}
	tokenWithClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, errSigned := tokenWithClaims.SignedString([]byte(cookieKey))

	if errSigned != nil {
		return "", errors.Join(errSigned, domain.ErrSignToken)
	}

	return signedToken, nil
}

type authHeader struct {
	Authorization string `header:"Authorization"`
}

func (u *Authentication) TokenFromHeader(ctx *gin.Context, emptyOK bool) (string, error) {
	hdr := authHeader{}
	if errBind := ctx.ShouldBindHeader(&hdr); errBind != nil {
		return "", errors.Join(errBind, domain.ErrAuthHeader)
	}

	pcs := strings.Split(hdr.Authorization, " ")
	if len(pcs) != 2 || pcs[1] == "" {
		if emptyOK {
			return "", nil
		}

		ctx.AbortWithStatus(http.StatusForbidden)

		return "", domain.ErrMalformedAuthHeader
	}

	return pcs[1], nil
}

func (u *Authentication) Sid64FromJWTToken(token string, cookieKey string, fingerprint string) (steamid.SteamID, error) {
	claims := &UserAuthClaims{}

	tkn, errParseClaims := jwt.ParseWithClaims(token, claims, u.MakeGetTokenKey(cookieKey))
	if errParseClaims != nil {
		if errors.Is(errParseClaims, jwt.ErrSignatureInvalid) {
			return steamid.SteamID{}, domain.ErrAuthentication
		}

		if errors.Is(errParseClaims, jwt.ErrTokenExpired) {
			return steamid.SteamID{}, domain.ErrExpired
		}

		return steamid.SteamID{}, domain.ErrAuthentication
	}

	if !tkn.Valid {
		return steamid.SteamID{}, domain.ErrAuthentication
	}

	if claims.Fingerprint != FingerprintHash(fingerprint) {
		slog.Error("Invalid cookie fingerprint, token rejected")

		return steamid.SteamID{}, domain.ErrAuthentication
	}

	sid := steamid.New(claims.Subject)
	if !sid.Valid() {
		return steamid.SteamID{}, domain.ErrAuthentication
	}

	return sid, nil
}

func (u *Authentication) Sid64FromJWTTokenNoFP(token string, cookieKey string) (steamid.SteamID, error) {
	claims := &UserAuthClaims{}

	tkn, errParseClaims := jwt.ParseWithClaims(token, claims, u.MakeGetTokenKey(cookieKey))
	if errParseClaims != nil {
		if errors.Is(errParseClaims, jwt.ErrSignatureInvalid) {
			return steamid.SteamID{}, domain.ErrAuthentication
		}

		if errors.Is(errParseClaims, jwt.ErrTokenExpired) {
			return steamid.SteamID{}, domain.ErrExpired
		}

		return steamid.SteamID{}, domain.ErrAuthentication
	}

	if !tkn.Valid {
		return steamid.SteamID{}, domain.ErrAuthentication
	}

	sid := steamid.New(claims.Subject)
	if !sid.Valid() {
		return steamid.SteamID{}, domain.ErrAuthentication
	}

	return sid, nil
}
