package auth

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const ctxKeyUserProfile = "user_profile"

type authUsecase struct {
	authRepository domain.AuthRepository
	configUsecase  domain.ConfigUsecase
	personUsecase  domain.PersonUsecase
	banUsecase     domain.BanSteamUsecase
	serverUsecase  domain.ServersUsecase
}

func NewAuthUsecase(authRepository domain.AuthRepository, configUsecase domain.ConfigUsecase, personUsecase domain.PersonUsecase,
	banUsecase domain.BanSteamUsecase, serversUsecase domain.ServersUsecase,
) domain.AuthUsecase {
	return &authUsecase{
		authRepository: authRepository,
		configUsecase:  configUsecase,
		personUsecase:  personUsecase,
		banUsecase:     banUsecase,
		serverUsecase:  serversUsecase,
	}
}

func (u *authUsecase) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Hour * 24)

	for {
		select {
		case <-ticker.C:
			if err := u.authRepository.PrunePersonAuth(ctx); err != nil && !errors.Is(err, domain.ErrNoResult) {
				slog.Error("Error pruning expired refresh tokens", log.ErrAttr(err))
			}
		case <-ctx.Done():
			slog.Debug("cleanupTasks shutting down")

			return
		}
	}
}

func (u *authUsecase) DeletePersonAuth(ctx context.Context, authID int64) error {
	return u.authRepository.DeletePersonAuth(ctx, authID)
}

func (u *authUsecase) GetPersonAuthByRefreshToken(ctx context.Context, token string, auth *domain.PersonAuth) error {
	return u.authRepository.GetPersonAuthByFingerprint(ctx, token, auth)
}

// MakeToken generates new jwt auth tokens
// fingerprint is a random string used to prevent side-jacking.
func (u *authUsecase) MakeToken(ctx *gin.Context, cookieKey string, sid steamid.SteamID) (domain.UserTokens, error) {
	if cookieKey == "" {
		return domain.UserTokens{}, domain.ErrCookieKeyMissing
	}

	fingerprint := util.SecureRandomString(40)

	accessToken, errAccess := u.NewUserToken(sid, cookieKey, fingerprint, domain.AuthTokenDuration)
	if errAccess != nil {
		return domain.UserTokens{}, errors.Join(errAccess, domain.ErrCreateToken)
	}

	ipAddr := net.ParseIP(ctx.ClientIP())
	if ipAddr == nil {
		return domain.UserTokens{}, domain.ErrClientIP
	}

	personAuth := domain.NewPersonAuth(sid, ipAddr, accessToken)

	if saveErr := u.authRepository.SavePersonAuth(ctx, &personAuth); saveErr != nil {
		return domain.UserTokens{}, errors.Join(saveErr, domain.ErrSaveToken)
	}

	return domain.UserTokens{Access: accessToken, Fingerprint: fingerprint}, nil
}

func (u *authUsecase) AuthMiddleware(level domain.Privilege) gin.HandlerFunc {
	cookieKey := u.configUsecase.Config().HTTPCookieKey

	return func(ctx *gin.Context) {
		var token string

		hdrToken, errToken := u.TokenFromHeader(ctx, level == domain.PGuest)
		if errToken != nil || hdrToken == "" {
			ctx.Set(ctxKeyUserProfile, domain.UserProfile{PermissionLevel: domain.PGuest, Name: "Guest"})
		} else {
			token = hdrToken

			if level >= domain.PGuest {
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

				loggedInPerson, errGetPerson := u.personUsecase.GetOrCreatePersonBySteamID(ctx, sid)
				if errGetPerson != nil {
					slog.Error("Failed to load person during auth", log.ErrAttr(errGetPerson))
					ctx.AbortWithStatus(http.StatusForbidden)

					return
				}

				if level > loggedInPerson.PermissionLevel {
					ctx.AbortWithStatus(http.StatusForbidden)

					return
				}

				bannedPerson, errBan := u.banUsecase.GetBySteamID(ctx, sid, false, true)
				if errBan != nil {
					if !errors.Is(errBan, domain.ErrNoResult) {
						slog.Error("Failed to fetch authed user ban", log.ErrAttr(errBan))
					}
				}

				profile := domain.UserProfile{
					SteamID:         loggedInPerson.SteamID,
					CreatedOn:       loggedInPerson.CreatedOn,
					UpdatedOn:       loggedInPerson.UpdatedOn,
					PermissionLevel: loggedInPerson.PermissionLevel,
					DiscordID:       loggedInPerson.DiscordID,
					PatreonID:       loggedInPerson.PatreonID,
					Name:            loggedInPerson.PersonaName,
					Avatarhash:      loggedInPerson.AvatarHash,
					Muted:           loggedInPerson.Muted,
					BanID:           bannedPerson.BanID,
				}

				ctx.Set(ctxKeyUserProfile, profile)

				if u.configUsecase.Config().Sentry.SentryDSN != "" {
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
				ctx.Set(ctxKeyUserProfile, domain.UserProfile{PermissionLevel: domain.PGuest, Name: "Guest"})
			}
		}

		ctx.Next()
	}
}

func (u *authUsecase) MakeGetTokenKey(cookieKey string) func(_ *jwt.Token) (any, error) {
	return func(_ *jwt.Token) (any, error) {
		return []byte(cookieKey), nil
	}
}

func (u *authUsecase) AuthServerMiddleWare() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reqAuthHeader := ctx.GetHeader("Authorization")
		if reqAuthHeader == "" {
			ctx.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		if strings.HasPrefix(reqAuthHeader, "Bearer ") {
			parts := strings.Split(reqAuthHeader, " ")
			if len(parts) != 2 {
				ctx.AbortWithStatus(http.StatusUnauthorized)

				return
			}

			reqAuthHeader = parts[1]
		}

		var server domain.Server
		if errServer := u.serverUsecase.GetServerByPassword(ctx, reqAuthHeader, &server, false, false); errServer != nil {
			slog.Error("Failed to load server during auth", log.ErrAttr(errServer), slog.String("token", reqAuthHeader), slog.String("IP", ctx.ClientIP()))
			ctx.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		ctx.Set("server_id", server.ServerID)

		if u.configUsecase.Config().Sentry.SentryDSN != "" {
			if hub := sentrygin.GetHubFromContext(ctx); hub != nil {
				hub.WithScope(func(scope *sentry.Scope) {
					scope.SetUser(sentry.User{
						ID:        strconv.Itoa(server.ServerID),
						IPAddress: server.Addr(),
						Name:      server.ShortName,
					})
				})
			}
		}

		ctx.Next()
	}
}

func (u *authUsecase) NewUserToken(steamID steamid.SteamID, cookieKey string, fingerPrint string, validDuration time.Duration) (string, error) {
	nowTime := time.Now()
	conf := u.configUsecase.Config()
	claims := domain.UserAuthClaims{
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

func (u *authUsecase) TokenFromHeader(ctx *gin.Context, emptyOK bool) (string, error) {
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

func (u *authUsecase) Sid64FromJWTToken(token string, cookieKey string, fingerprint string) (steamid.SteamID, error) {
	claims := &domain.UserAuthClaims{}

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
