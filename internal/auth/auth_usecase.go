package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

const ctxKeyUserProfile = "user_profile"

type authUsecase struct {
	authRepository domain.AuthRepository
	configUsecase  domain.ConfigUsecase
	personUsecase  domain.PersonUsecase
	banUsecase     domain.BanSteamUsecase
	serverUsecase  domain.ServersUsecase
	log            *zap.Logger
}

func NewAuthUsecase(log *zap.Logger, authRepository domain.AuthRepository, configUsecase domain.ConfigUsecase, personUsecase domain.PersonUsecase,
	banUsecase domain.BanSteamUsecase, serversUsecase domain.ServersUsecase,
) domain.AuthUsecase {
	return &authUsecase{
		authRepository: authRepository,
		configUsecase:  configUsecase,
		personUsecase:  personUsecase,
		banUsecase:     banUsecase,
		serverUsecase:  serversUsecase,
		log:            log.Named("auth"),
	}
}

func (u *authUsecase) Start(ctx context.Context) {
	var (
		log    = u.log.Named("cleanupTasks")
		ticker = time.NewTicker(time.Hour * 24)
	)

	for {
		select {
		case <-ticker.C:
			if err := u.authRepository.PrunePersonAuth(ctx); err != nil && !errors.Is(err, domain.ErrNoResult) {
				log.Error("Error pruning expired refresh tokens", zap.Error(err))
			}
		case <-ctx.Done():
			log.Debug("cleanupTasks shutting down")

			return
		}
	}
}

func (u *authUsecase) DeletePersonAuth(ctx context.Context, authID int64) error {
	return u.authRepository.DeletePersonAuth(ctx, authID)
}

func (u *authUsecase) GetPersonAuthByRefreshToken(ctx context.Context, token string, auth *domain.PersonAuth) error {
	return u.authRepository.GetPersonAuthByRefreshToken(ctx, token, auth)
}

// MakeTokens generates new jwt auth tokens
// fingerprint is a random string used to prevent side-jacking.
func (u *authUsecase) MakeTokens(ctx *gin.Context, cookieKey string, sid steamid.SID64, createRefresh bool) (domain.UserTokens, error) {
	if cookieKey == "" {
		return domain.UserTokens{}, domain.ErrCookieKeyMissing
	}

	fingerprint := util.SecureRandomString(40)

	accessToken, errJWT := u.NewUserToken(sid, cookieKey, fingerprint, domain.AuthTokenDuration)
	if errJWT != nil {
		return domain.UserTokens{}, errors.Join(errJWT, domain.ErrCreateToken)
	}

	refreshToken := ""

	if createRefresh {
		newRefreshToken, errRefresh := u.NewUserToken(sid, cookieKey, fingerprint, domain.RefreshTokenDuration)
		if errRefresh != nil {
			return domain.UserTokens{}, errors.Join(errRefresh, domain.ErrRefreshToken)
		}

		ipAddr := net.ParseIP(ctx.ClientIP())
		if ipAddr == nil {
			return domain.UserTokens{}, domain.ErrClientIP
		}

		personAuth := domain.NewPersonAuth(sid, ipAddr, fingerprint)
		// TODO move to authUsecase
		if saveErr := u.authRepository.SavePersonAuth(ctx, &personAuth); saveErr != nil {
			return domain.UserTokens{}, errors.Join(saveErr, domain.ErrSaveToken)
		}

		refreshToken = newRefreshToken
	}

	return domain.UserTokens{
		Access:      accessToken,
		Refresh:     refreshToken,
		Fingerprint: fingerprint,
	}, nil
}

func (u *authUsecase) AuthMiddleware(level domain.Privilege) gin.HandlerFunc {
	log := u.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	cookieKey := u.configUsecase.Config().HTTP.CookieKey

	return func(ctx *gin.Context) {
		var token string

		hdrToken, errToken := u.TokenFromHeader(ctx, level == domain.PGuest)
		if errToken != nil || hdrToken == "" {
			ctx.Set(ctxKeyUserProfile, domain.UserProfile{PermissionLevel: domain.PGuest, Name: "Guest"})
		} else {
			token = hdrToken

			if level >= domain.PGuest {
				sid, errFromToken := u.Sid64FromJWTToken(token, cookieKey)
				if errFromToken != nil {
					if errors.Is(errFromToken, domain.ErrExpired) {
						ctx.AbortWithStatus(http.StatusUnauthorized)

						return
					}

					log.Error("Failed to load sid from access token", zap.Error(errFromToken))
					ctx.AbortWithStatus(http.StatusForbidden)

					return
				}

				loggedInPerson := domain.NewPerson(sid)
				if errGetPerson := u.personUsecase.GetOrCreatePersonBySteamID(ctx, sid, &loggedInPerson); errGetPerson != nil {
					log.Error("Failed to load person during auth", zap.Error(errGetPerson))
					ctx.AbortWithStatus(http.StatusForbidden)

					return
				}

				if level > loggedInPerson.PermissionLevel {
					ctx.AbortWithStatus(http.StatusForbidden)

					return
				}

				bannedPerson := domain.NewBannedPerson()
				if errBan := u.banUsecase.GetBySteamID(ctx, sid, &bannedPerson, false); errBan != nil {
					if !errors.Is(errBan, domain.ErrNoResult) {
						log.Error("Failed to fetch authed user ban", zap.Error(errBan))
					}
				}

				profile := domain.UserProfile{
					SteamID:         loggedInPerson.SteamID,
					CreatedOn:       loggedInPerson.CreatedOn,
					UpdatedOn:       loggedInPerson.UpdatedOn,
					PermissionLevel: loggedInPerson.PermissionLevel,
					DiscordID:       loggedInPerson.DiscordID,
					Name:            loggedInPerson.PersonaName,
					Avatarhash:      loggedInPerson.AvatarHash,
					Muted:           loggedInPerson.Muted,
					BanID:           bannedPerson.BanID,
				}

				ctx.Set(ctxKeyUserProfile, profile)

				if hub := sentrygin.GetHubFromContext(ctx); hub != nil {
					hub.WithScope(func(scope *sentry.Scope) {
						scope.SetUser(sentry.User{
							ID:        sid.String(),
							IPAddress: ctx.ClientIP(),
							Username:  loggedInPerson.PersonaName,
						})
					})
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
	log := u.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	cookieKey := u.configUsecase.Config().HTTP.CookieKey

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

		claims := &domain.ServerAuthClaims{}

		parsedToken, errParseClaims := jwt.ParseWithClaims(reqAuthHeader, claims, u.MakeGetTokenKey(cookieKey))
		if errParseClaims != nil {
			if errors.Is(errParseClaims, jwt.ErrSignatureInvalid) {
				log.Error("jwt signature invalid!", zap.Error(errParseClaims))
				ctx.AbortWithStatus(http.StatusUnauthorized)

				return
			}

			ctx.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		if !parsedToken.Valid {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			log.Error("Invalid jwt token parsed")

			return
		}

		if claims.ServerID <= 0 {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			log.Error("Invalid jwt claim server")

			return
		}

		var server domain.Server
		if errGetServer := u.serverUsecase.GetServer(ctx, claims.ServerID, &server); errGetServer != nil {
			log.Error("Failed to load server during auth", zap.Error(errGetServer))
			ctx.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		ctx.Set("server_id", claims.ServerID)

		if hub := sentrygin.GetHubFromContext(ctx); hub != nil {
			hub.WithScope(func(scope *sentry.Scope) {
				scope.SetUser(sentry.User{
					ID:        fmt.Sprintf("%d", server.ServerID),
					IPAddress: server.Addr(),
					Name:      server.ShortName,
				})
			})
		}

		ctx.Next()
	}
}

func (u *authUsecase) NewUserToken(steamID steamid.SID64, cookieKey string, userContext string, validDuration time.Duration) (string, error) {
	nowTime := time.Now()

	claims := domain.UserAuthClaims{
		Fingerprint: FingerprintHash(userContext),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "gbans",
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

func (u *authUsecase) Sid64FromJWTToken(token string, cookieKey string) (steamid.SID64, error) {
	claims := &jwt.RegisteredClaims{}

	tkn, errParseClaims := jwt.ParseWithClaims(token, claims, u.MakeGetTokenKey(cookieKey))
	if errParseClaims != nil {
		if errors.Is(errParseClaims, jwt.ErrSignatureInvalid) {
			return "", domain.ErrAuthentication
		}

		if errors.Is(errParseClaims, jwt.ErrTokenExpired) {
			return "", domain.ErrExpired
		}

		return "", domain.ErrAuthentication
	}

	if !tkn.Valid {
		return "", domain.ErrAuthentication
	}

	sid := steamid.New(claims.Subject)
	if !sid.Valid() {
		return "", domain.ErrAuthentication
	}

	return sid, nil
}
