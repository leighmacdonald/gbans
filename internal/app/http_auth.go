package app

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"github.com/yohcop/openid-go"
	"go.uber.org/zap"
)

// noOpDiscoveryCache implements the DiscoveryCache interface and doesn't cache anything.
type noOpDiscoveryCache struct{}

// Put is a no op.
func (n *noOpDiscoveryCache) Put(_ string, _ openid.DiscoveredInfo) {}

// Get always returns nil.
//
//nolint:ireturn
func (n *noOpDiscoveryCache) Get(_ string) openid.DiscoveredInfo {
	return nil
}

func authServerMiddleWare(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

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

		claims := &serverAuthClaims{}

		parsedToken, errParseClaims := jwt.ParseWithClaims(reqAuthHeader, claims, makeGetTokenKey(app.conf.HTTP.CookieKey))
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

		var server store.Server
		if errGetServer := app.db.GetServer(ctx, claims.ServerID, &server); errGetServer != nil {
			log.Error("Failed to load server during auth", zap.Error(errGetServer))
			ctx.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		ctx.Set("server_id", claims.ServerID)

		ctx.Next()
	}
}

func referral(ctx *gin.Context) string {
	referralURL, found := ctx.GetQuery("return_url")
	if !found {
		referralURL = "/"
	}

	return referralURL
}

func onOpenIDCallback(app *App) gin.HandlerFunc {
	nonceStore := openid.NewSimpleNonceStore()
	discoveryCache := &noOpDiscoveryCache{}
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())
	oidRx := regexp.MustCompile(`^https://steamcommunity\.com/openid/id/(\d+)$`)

	return func(ctx *gin.Context) {
		var idStr string

		referralURL := referral(ctx)
		fullURL := app.conf.General.ExternalURL + ctx.Request.URL.String()

		if app.conf.Debug.SkipOpenIDValidation {
			// Pull the sid out of the query without doing a signature check
			values, errParse := url.Parse(fullURL)
			if errParse != nil {
				log.Error("Failed to parse url", zap.Error(errParse))
				ctx.Redirect(302, referralURL)

				return
			}

			idStr = values.Query().Get("openid.identity")
		} else {
			openID, errVerify := openid.Verify(fullURL, discoveryCache, nonceStore)
			if errVerify != nil {
				log.Error("Error verifying openid auth response", zap.Error(errVerify))
				ctx.Redirect(302, referralURL)

				return
			}
			idStr = openID
		}

		match := oidRx.FindStringSubmatch(idStr)
		if match == nil || len(match) != 2 {
			ctx.Redirect(302, referralURL)

			return
		}

		sid, errDecodeSid := steamid.SID64FromString(match[1])
		if errDecodeSid != nil {
			log.Error("Received invalid steamid", zap.Error(errDecodeSid))
			ctx.Redirect(302, referralURL)

			return
		}

		person := store.NewPerson(sid)
		if errGetProfile := app.PersonBySID(ctx, sid, &person); errGetProfile != nil {
			log.Error("Failed to fetch user profile", zap.Error(errGetProfile))
			ctx.Redirect(302, referralURL)

			return
		}

		tokens, errToken := makeTokens(ctx, app.db, app.conf.HTTP.CookieKey, sid, true)
		if errToken != nil {
			ctx.Redirect(302, referralURL)
			log.Error("Failed to create access token pair", zap.Error(errToken))

			return
		}

		parsedURL, errParse := url.Parse("/login/success")
		if errParse != nil {
			ctx.Redirect(302, referralURL)

			return
		}

		query := parsedURL.Query()
		query.Set("refresh", tokens.refresh)
		query.Set("token", tokens.access)
		query.Set("next_url", referralURL)
		parsedURL.RawQuery = query.Encode()

		parsedExternal, errExternal := url.Parse(app.conf.General.ExternalURL)
		if errExternal != nil {
			ctx.Redirect(302, referralURL)
			log.Error("Failed to parse ext url", zap.Error(errExternal))

			return
		}
		// TODO max age checks
		ctx.SetSameSite(http.SameSiteStrictMode)
		ctx.SetCookie(fingerprintCookieName, tokens.fingerprint, int(refreshTokenDuration.Seconds()), "/api", parsedExternal.Hostname(), app.conf.General.Mode == ReleaseMode, true)

		ctx.Redirect(302, parsedURL.String())
		log.Info("User logged in",
			zap.Int64("sid64", sid.Int64()),
			zap.String("name", person.PersonaName),
			zap.Int("permission_level", int(person.PermissionLevel)))
	}
}

type userTokens struct {
	access      string
	refresh     string
	fingerprint string
}

func fingerprintHash(fingerprint string) string {
	hasher := sha256.New()

	return fmt.Sprintf("%x", hasher.Sum([]byte(fingerprint)))
}

// makeTokens generates new jwt auth tokens
// fingerprint is a random string used to prevent side-jacking.
func makeTokens(ctx *gin.Context, database *store.Store, cookieKey string, sid steamid.SID64, createRefresh bool) (userTokens, error) {
	if cookieKey == "" {
		return userTokens{}, errors.New("cookieKey or fingerprint empty")
	}

	fingerprint := store.SecureRandomString(40)

	accessToken, errJWT := newUserToken(sid, cookieKey, fingerprint, authTokenDuration)
	if errJWT != nil {
		return userTokens{}, errors.Wrap(errJWT, "Failed to create new access token")
	}

	refreshToken := ""

	if createRefresh {
		newRefreshToken, errRefresh := newUserToken(sid, cookieKey, fingerprint, refreshTokenDuration)
		if errRefresh != nil {
			return userTokens{}, errors.Wrap(errRefresh, "Failed to create new refresh token")
		}

		ipAddr := net.ParseIP(ctx.ClientIP())
		if ipAddr == nil {
			return userTokens{}, errors.New("Failed to parse IP")
		}

		personAuth := store.NewPersonAuth(sid, ipAddr, fingerprint)
		if saveErr := database.SavePersonAuth(ctx, &personAuth); saveErr != nil {
			return userTokens{}, errors.Wrap(saveErr, "Failed to save new createRefresh token")
		}

		refreshToken = newRefreshToken
	}

	return userTokens{
		access:      accessToken,
		refresh:     refreshToken,
		fingerprint: fingerprint,
	}, nil
}

type userToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type serverAuthClaims struct {
	ServerID int `json:"server_id"`
	// A random string which is used to fingerprint and prevent sidejacking
	jwt.RegisteredClaims
}

type userAuthClaims struct {
	// user context to prevent side-jacking
	// https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html#token-sidejacking
	Fingerprint string `json:"fingerprint"`
	jwt.RegisteredClaims
}

const (
	authTokenDuration    = time.Minute * 15
	refreshTokenDuration = time.Hour * 24 * 31
)

func newUserToken(steamID steamid.SID64, cookieKey string, userContext string, validDuration time.Duration) (string, error) {
	nowTime := time.Now()

	claims := userAuthClaims{
		Fingerprint: fingerprintHash(userContext),
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
		return "", errors.Wrap(errSigned, "Failed create signed string")
	}

	return signedToken, nil
}

func newServerToken(serverID int, cookieKey string) (string, error) {
	curTime := time.Now()

	claims := &serverAuthClaims{
		ServerID: serverID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(curTime.Add(authTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(curTime),
			NotBefore: jwt.NewNumericDate(curTime),
		},
	}

	tokenWithClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, errSigned := tokenWithClaims.SignedString([]byte(cookieKey))
	if errSigned != nil {
		return "", errors.Wrap(errSigned, "Failed create signed string")
	}

	return signedToken, nil
}

type authHeader struct {
	Authorization string `header:"Authorization"`
}

func tokenFromHeader(ctx *gin.Context) (string, error) {
	hdr := authHeader{}
	if errBind := ctx.ShouldBindHeader(&hdr); errBind != nil {
		return "", errors.Wrap(errBind, "Failed to bind auth header")
	}

	pcs := strings.Split(hdr.Authorization, " ")
	if len(pcs) != 2 || pcs[1] == "" {
		ctx.AbortWithStatus(http.StatusForbidden)

		return "", errors.New("Invalid auth header")
	}

	return pcs[1], nil
}

// authMiddleware handles client authentication to the HTTP & websocket api.
// websocket clients must pass the key as a query parameter called "token".
func authMiddleware(app *App, level consts.Privilege) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var token string

		hdrToken, errToken := tokenFromHeader(ctx)
		if errToken != nil {
			ctx.Set(ctxKeyUserProfile, userProfile{PermissionLevel: consts.PGuest, Name: "Guest"})
			ctx.Next()

			return
		}

		token = hdrToken

		if level >= consts.PGuest {
			sid, errFromToken := sid64FromJWTToken(token, app.conf.HTTP.CookieKey)
			if errFromToken != nil {
				if errors.Is(errFromToken, consts.ErrExpired) {
					ctx.AbortWithStatus(http.StatusUnauthorized)

					return
				}

				log.Error("Failed to load sid from access token", zap.Error(errFromToken))
				ctx.AbortWithStatus(http.StatusForbidden)

				return
			}

			loggedInPerson := store.NewPerson(sid)
			if errGetPerson := app.PersonBySID(ctx, sid, &loggedInPerson); errGetPerson != nil {
				log.Error("Failed to load person during auth", zap.Error(errGetPerson))
				ctx.AbortWithStatus(http.StatusForbidden)

				return
			}

			if level > loggedInPerson.PermissionLevel {
				ctx.AbortWithStatus(http.StatusForbidden)

				return
			}

			bannedPerson := store.NewBannedPerson()
			if errBan := app.db.GetBanBySteamID(ctx, sid, &bannedPerson, false); errBan != nil {
				if !errors.Is(errBan, store.ErrNoResult) {
					log.Error("Failed to fetch authed user ban", zap.Error(errBan))
				}
			}

			profile := userProfile{
				SteamID:         loggedInPerson.SteamID,
				CreatedOn:       loggedInPerson.CreatedOn,
				UpdatedOn:       loggedInPerson.UpdatedOn,
				PermissionLevel: loggedInPerson.PermissionLevel,
				DiscordID:       loggedInPerson.DiscordID,
				Name:            loggedInPerson.PersonaName,
				Avatar:          loggedInPerson.Avatar,
				Avatarfull:      loggedInPerson.AvatarFull,
				Muted:           loggedInPerson.Muted,
				BanID:           bannedPerson.BanID,
			}
			ctx.Set(ctxKeyUserProfile, profile)
		} else {
			ctx.Set(ctxKeyUserProfile, userProfile{PermissionLevel: consts.PGuest, Name: "Guest"})
		}

		ctx.Next()
	}
}

func sid64FromJWTToken(token string, cookieKey string) (steamid.SID64, error) {
	claims := &jwt.RegisteredClaims{}

	tkn, errParseClaims := jwt.ParseWithClaims(token, claims, makeGetTokenKey(cookieKey))
	if errParseClaims != nil {
		if errors.Is(errParseClaims, jwt.ErrSignatureInvalid) {
			return "", consts.ErrAuthentication
		}

		if errors.Is(errParseClaims, jwt.ErrTokenExpired) {
			return "", consts.ErrExpired
		}

		return "", consts.ErrAuthentication
	}

	if !tkn.Valid {
		return "", consts.ErrAuthentication
	}

	sid := steamid.New(claims.Subject)
	if !sid.Valid() {
		return "", consts.ErrAuthentication
	}

	return sid, nil
}
