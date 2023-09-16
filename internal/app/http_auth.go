package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	"github.com/leighmacdonald/gbans/pkg/util"
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
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		claims := &serverAuthClaims{}

		parsedToken, errParseClaims := jwt.ParseWithClaims(authHeader, claims, makeGetTokenKey(app.conf.HTTP.CookieKey))
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

func onGetLogout(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		// TODO Logout key / mark as invalid manually
		log.Error("onGetLogout Unimplemented")
		ctx.Redirect(http.StatusTemporaryRedirect, "/")
	}
}

func referral(ctx *gin.Context) string {
	referralURL, found := ctx.GetQuery("return_url")
	if !found {
		referralURL = "/"
	}

	return referralURL
}

func onOAuthDiscordCallback(app *App) gin.HandlerFunc {
	type accessTokenResp struct {
		AccessToken  string `json:"access_token"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
		TokenType    string `json:"token_type"`
	}

	type discordUserDetail struct {
		ID               string      `json:"id"`
		Username         string      `json:"username"`
		Avatar           string      `json:"avatar"`
		AvatarDecoration interface{} `json:"avatar_decoration"`
		Discriminator    string      `json:"discriminator"`
		PublicFlags      int         `json:"public_flags"`
		Flags            int         `json:"flags"`
		Banner           interface{} `json:"banner"`
		BannerColor      interface{} `json:"banner_color"`
		AccentColor      interface{} `json:"accent_color"`
		Locale           string      `json:"locale"`
		MfaEnabled       bool        `json:"mfa_enabled"`
		PremiumType      int         `json:"premium_type"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())
	client := util.NewHTTPClient()

	fetchDiscordID := func(ctx context.Context, accessToken string) (string, error) {
		req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, "https://discord.com/api/users/@me", nil)
		if errReq != nil {
			return "", errors.Wrap(errReq, "Failed to create new request")
		}

		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
		resp, errResp := client.Do(req)

		if errResp != nil {
			return "", errors.Wrap(errResp, "Failed to perform http request")
		}

		defer func() {
			_ = resp.Body.Close()
		}()

		body, errBody := io.ReadAll(resp.Body)
		if errBody != nil {
			return "", errors.Wrap(errBody, "Failed to read response body")
		}

		var details discordUserDetail
		if errJSON := json.Unmarshal(body, &details); errJSON != nil {
			return "", errors.Wrap(errJSON, "Failed to unmarshal response")
		}

		return details.ID, nil
	}

	fetchToken := func(ctx context.Context, code string) (string, error) {
		// v, _ := go_oauth_pkce_code_verifier.CreateCodeVerifierFromBytes([]byte(code))
		form := url.Values{}
		form.Set("client_id", app.conf.Discord.AppID)
		form.Set("client_secret", app.conf.Discord.AppSecret)
		form.Set("redirect_uri", app.ExtURLRaw("/login/discord"))
		form.Set("code", code)
		form.Set("grant_type", "authorization_code")
		// form.Set("state", state.String())
		form.Set("scope", "identify")
		req, errReq := http.NewRequestWithContext(ctx, http.MethodPost, "https://discord.com/api/oauth2/token", strings.NewReader(form.Encode()))

		if errReq != nil {
			return "", errors.Wrap(errReq, "Failed to create new request")
		}

		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		resp, errResp := client.Do(req)
		if errResp != nil {
			return "", errors.Wrap(errResp, "Failed to perform http request")
		}

		defer func() {
			_ = resp.Body.Close()
		}()

		body, errBody := io.ReadAll(resp.Body)
		if errBody != nil {
			return "", errors.Wrap(errBody, "Failed to read response body")
		}

		var atr accessTokenResp
		if errJSON := json.Unmarshal(body, &atr); errJSON != nil {
			return "", errors.Wrap(errJSON, "Failed to decode response body")
		}

		if atr.AccessToken == "" {
			return "", errors.New("Empty token returned")
		}

		return atr.AccessToken, nil
	}

	return func(ctx *gin.Context) {
		code := ctx.Query("code")
		if code == "" {
			responseErr(ctx, http.StatusBadRequest, nil)
			log.Error("Failed to get code from query")

			return
		}

		token, errToken := fetchToken(ctx, code)
		if errToken != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			log.Error("Failed to fetch token", zap.Error(errToken))

			return
		}

		discordID, errID := fetchDiscordID(ctx, token)
		if errID != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			log.Error("Failed to fetch discord ID", zap.Error(errID))

			return
		}

		if discordID == "" {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Empty discord id received")

			return
		}

		var discordPerson store.Person
		if errDp := app.db.GetPersonByDiscordID(ctx, discordID, &discordPerson); errDp != nil {
			if !errors.Is(errDp, store.ErrNoResult) {
				responseErr(ctx, http.StatusInternalServerError, nil)

				return
			}
		}

		if discordPerson.DiscordID != "" {
			responseErr(ctx, http.StatusConflict, nil)
			log.Error("Failed to update persons discord id")

			return
		}

		sid := currentUserProfile(ctx).SteamID

		var person store.Person
		if errPerson := app.PersonBySID(ctx, sid, &person); errPerson != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		person.DiscordID = discordID

		if errSave := app.db.SavePerson(ctx, &person); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		ctx.JSON(http.StatusOK, nil)

		log.Info("Discord account linked successfully",
			zap.String("discord_id", discordID), zap.Int64("sid64", sid.Int64()))
	}
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

		accessToken, refreshToken, errToken := makeTokens(ctx, app.db, app.conf.HTTP.CookieKey, sid)
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
		query.Set("refresh", refreshToken)
		query.Set("token", accessToken)
		query.Set("next_url", referralURL)
		parsedURL.RawQuery = query.Encode()
		ctx.Redirect(302, parsedURL.String())
		log.Info("User logged in",
			zap.Int64("sid64", sid.Int64()),
			zap.String("name", person.PersonaName),
			zap.Int("permission_level", int(person.PermissionLevel)))
	}
}

func makeTokens(ctx *gin.Context, database *store.Store, cookieKey string, sid steamid.SID64) (string, string, error) {
	accessToken, errJWT := newUserJWT(sid, cookieKey)
	if errJWT != nil {
		return "", "", errors.Wrap(errJWT, "Failed to create new access token")
	}

	ipAddr := net.ParseIP(ctx.ClientIP())
	refreshToken := store.NewPersonAuth(sid, ipAddr)

	if errAuth := database.GetPersonAuth(ctx, sid, ipAddr, &refreshToken); errAuth != nil {
		if !errors.Is(errAuth, store.ErrNoResult) {
			return "", "", errors.Wrap(errAuth, "Failed to fetch refresh token")
		}

		if createErr := database.SavePersonAuth(ctx, &refreshToken); createErr != nil {
			return "", "", errors.Wrap(errAuth, "Failed to create new refresh token")
		}
	}

	return accessToken, refreshToken.RefreshToken, nil
}

func makeGetTokenKey(cookieKey string) func(_ *jwt.Token) (any, error) {
	return func(_ *jwt.Token) (any, error) {
		return []byte(cookieKey), nil
	}
}

// onTokenRefresh handles generating new token pairs to access the api
// NOTE: All error code paths must return 401 (Unauthorized).
func onTokenRefresh(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var usrToken userToken
		if errBind := ctx.BindJSON(&usrToken); errBind != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			log.Error("Malformed user token", zap.Error(errBind))

			return
		}

		if usrToken.RefreshToken == "" {
			ctx.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		var auth store.PersonAuth
		if authError := app.db.GetPersonAuthByRefreshToken(ctx, usrToken.RefreshToken, &auth); authError != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		newAccessToken, newRefreshToken, errToken := makeTokens(ctx, app.db, app.conf.HTTP.CookieKey, auth.SteamID)
		if errToken != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			log.Error("Failed to create access token pair", zap.Error(errToken))

			return
		}

		ctx.JSON(http.StatusOK, userToken{
			AccessToken:  newAccessToken,
			RefreshToken: newRefreshToken,
		})
	}
}

type userToken struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token"`
}

type serverAuthClaims struct {
	ServerID int `json:"server_id"`
	jwt.RegisteredClaims
}

const authTokenLifetimeDuration = time.Hour * 24 * 30 // 1 month

func newUserJWT(steamID steamid.SID64, cookieKey string) (string, error) {
	nowTime := time.Now()
	claims := &jwt.RegisteredClaims{
		Issuer:    "gbans",
		Subject:   steamID.String(),
		ExpiresAt: jwt.NewNumericDate(nowTime.Add(authTokenLifetimeDuration)),
		IssuedAt:  jwt.NewNumericDate(nowTime),
		NotBefore: jwt.NewNumericDate(nowTime),
	}

	tokenWithClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, errSigned := tokenWithClaims.SignedString([]byte(cookieKey))

	if errSigned != nil {
		return "", errors.Wrap(errSigned, "Failed create signed string")
	}

	return signedToken, nil
}

func newServerJWT(serverID int, cookieKey string) (string, error) {
	curTime := time.Now()

	claims := &serverAuthClaims{
		ServerID: serverID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(curTime.Add(authTokenLifetimeDuration)),
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

// authMiddleware handles client authentication to the HTTP & websocket api.
// websocket clients must pass the key as a query parameter called "token".
func authMiddleware(app *App, level consts.Privilege) gin.HandlerFunc {
	type header struct {
		Authorization string `header:"Authorization"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var token string
		if ctx.FullPath() == "/ws" {
			token = ctx.Query("token")
		} else {
			hdr := header{}
			if errBind := ctx.ShouldBindHeader(&hdr); errBind != nil {
				ctx.AbortWithStatus(http.StatusForbidden)

				return
			}
			pcs := strings.Split(hdr.Authorization, " ")
			if len(pcs) != 2 && level >= consts.PUser {
				ctx.AbortWithStatus(http.StatusForbidden)

				return
			}
			token = pcs[1]
		}

		if level >= consts.PUser {
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
				BanID:           bannedPerson.Ban.BanID,
			}
			ctx.Set(ctxKeyUserProfile, profile)
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
