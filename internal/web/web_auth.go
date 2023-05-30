package web

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/leighmacdonald/gbans/internal/app"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"github.com/yohcop/openid-go"
	"go.uber.org/zap"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// noOpDiscoveryCache implements the DiscoveryCache interface and doesn't cache anything.
type noOpDiscoveryCache struct{}

// Put is a no op.
func (n *noOpDiscoveryCache) Put(_ string, _ openid.DiscoveredInfo) {}

// Get always returns nil.
func (n *noOpDiscoveryCache) Get(_ string) openid.DiscoveredInfo {
	return nil
}

var nonceStore = openid.NewSimpleNonceStore()
var discoveryCache = &noOpDiscoveryCache{}

func authServerMiddleWare() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		claims := &serverAuthClaims{}
		parsedToken, errParseClaims := jwt.ParseWithClaims(authHeader, claims, getTokenKey)
		if errParseClaims != nil {
			if errParseClaims == jwt.ErrSignatureInvalid {
				logger.Error("jwt signature invalid!", zap.Error(errParseClaims))
				ctx.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if !parsedToken.Valid {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			logger.Error("Invalid jwt token parsed")
			return
		}
		if claims.ServerId <= 0 {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			logger.Error("Invalid jwt claim ServerId")
			return
		}
		var server store.Server
		if errGetServer := store.GetServer(ctx, claims.ServerId, &server); errGetServer != nil {
			logger.Error("Failed to load server during auth", zap.Error(errGetServer))
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		ctx.Next()
	}
}

func onGetLogout() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// TODO Logout key / mark as invalid manually
		logger.Error("onGetLogout Unimplemented")
		ctx.Redirect(http.StatusTemporaryRedirect, "/")
	}
}

func referral(ctx *gin.Context) string {
	referralUrl, found := ctx.GetQuery("return_url")
	if !found {
		referralUrl = "/"
	}
	return referralUrl
}

func onOAuthDiscordCallback() gin.HandlerFunc {
	client := util.NewHTTPClient()
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

	var fetchDiscordId = func(ctx context.Context, accessToken string) (string, error) {
		req, errReq := http.NewRequestWithContext(ctx, "GET", "https://discord.com/api/users/@me", nil)
		if errReq != nil {
			return "", errReq
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
		resp, errResp := client.Do(req)
		if errResp != nil {
			return "", errResp
		}
		b, errBody := io.ReadAll(resp.Body)
		if errBody != nil {
			return "", errBody
		}
		defer util.LogCloser(resp.Body, logger)
		var details discordUserDetail
		if errJson := json.Unmarshal(b, &details); errJson != nil {
			return "", errJson
		}
		return details.ID, nil
	}

	var fetchToken = func(ctx context.Context, code string) (string, error) {
		form := url.Values{}
		form.Set("client_id", config.Discord.AppID)
		form.Set("client_secret", config.Discord.AppSecret)
		form.Set("redirect_uri", config.ExtURL("/login/discordutil"))
		form.Set("code", code)
		form.Set("grant_type", "authorization_code")
		form.Set("scope", "identify")
		req, errReq := http.NewRequestWithContext(ctx, "POST", "https://discord.com/api/oauth2/token", strings.NewReader(form.Encode()))
		if errReq != nil {
			return "", errReq
		}
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		resp, errResp := client.Do(req)
		if errResp != nil {
			return "", errResp
		}
		body, errBody := io.ReadAll(resp.Body)
		if errBody != nil {
			return "", errBody
		}
		defer util.LogCloser(resp.Body, logger)
		var atr accessTokenResp
		if errJson := json.Unmarshal(body, &atr); errJson != nil {
			return "", errJson
		}
		return atr.AccessToken, nil
	}

	return func(ctx *gin.Context) {
		code := ctx.Query("code")
		if code == "" {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		token, errToken := fetchToken(ctx, code)
		if errToken != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		discordId, errId := fetchDiscordId(ctx, token)
		if errId != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		if discordId == "" {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		var dp store.Person
		if errDp := store.GetPersonByDiscordID(ctx, discordId, &dp); errDp != nil {
			if !errors.Is(errDp, store.ErrNoResult) {
				responseErr(ctx, http.StatusInternalServerError, nil)
				return
			}
		}
		if dp.DiscordID != "" {
			responseErr(ctx, http.StatusConflict, nil)
			return
		}
		sid := currentUserProfile(ctx).SteamID
		var sp store.Person
		if errPerson := app.PersonBySID(ctx, sid, &sp); errPerson != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		sp.DiscordID = discordId
		if errSave := store.SavePerson(ctx, &sp); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusInternalServerError, nil)
		logger.Info("Discord account linked successfully",
			zap.String("discord_id", discordId), zap.Int64("sid64", sid.Int64()))
	}
}

func onOpenIDCallback() gin.HandlerFunc {
	oidRx := regexp.MustCompile(`^https://steamcommunity\.com/openid/id/(\d+)$`)
	return func(ctx *gin.Context) {
		var idStr string
		referralUrl := referral(ctx)
		fullURL := config.General.ExternalUrl + ctx.Request.URL.String()
		if config.Debug.SkipOpenIDValidation {
			// Pull the sid out of the query without doing a signature check
			values, errParse := url.Parse(fullURL)
			if errParse != nil {
				logger.Error("Failed to parse url", zap.Error(errParse))
				ctx.Redirect(302, referralUrl)
				return
			}
			idStr = values.Query().Get("openid.identity")
		} else {
			id, errVerify := openid.Verify(fullURL, discoveryCache, nonceStore)
			if errVerify != nil {
				logger.Error("Error verifying openid auth response", zap.Error(errVerify))
				ctx.Redirect(302, referralUrl)
				return
			}
			idStr = id
		}
		match := oidRx.FindStringSubmatch(idStr)
		if match == nil || len(match) != 2 {
			ctx.Redirect(302, referralUrl)
			return
		}
		sid, errDecodeSid := steamid.SID64FromString(match[1])
		if errDecodeSid != nil {
			logger.Error("Received invalid steamid", zap.Error(errDecodeSid))
			ctx.Redirect(302, referralUrl)
			return
		}
		person := store.NewPerson(sid)
		if errGetProfile := app.PersonBySID(ctx, sid, &person); errGetProfile != nil {
			logger.Error("Failed to fetch user profile", zap.Error(errGetProfile))
			ctx.Redirect(302, referralUrl)
			return
		}
		accessToken, refreshToken, errToken := makeTokens(ctx, sid)
		if errToken != nil {
			ctx.Redirect(302, referralUrl)
			logger.Error("Failed to create access token pair", zap.Error(errToken))
			return
		}
		parsedUrl, errParse := url.Parse("/login/success")
		if errParse != nil {
			ctx.Redirect(302, referralUrl)
			return
		}
		query := parsedUrl.Query()
		query.Set("refresh", refreshToken)
		query.Set("token", accessToken)
		query.Set("next_url", referralUrl)
		parsedUrl.RawQuery = query.Encode()
		ctx.Redirect(302, parsedUrl.String())
		logger.Info("User logged in",
			zap.Int64("sid64", sid.Int64()),
			zap.String("name", person.PersonaName),
			zap.Int("permission_level", int(person.PermissionLevel)))
	}
}

func makeTokens(ctx *gin.Context, sid steamid.SID64) (string, string, error) {
	accessToken, errJWT := newUserJWT(sid)
	if errJWT != nil {
		return "", "", errors.Wrap(errJWT, "Failed to create new access token")
	}
	ipAddr := net.ParseIP(ctx.ClientIP())
	refreshToken := store.NewPersonAuth(sid, ipAddr)
	if errAuth := store.GetPersonAuth(ctx, sid, ipAddr, &refreshToken); errAuth != nil {
		if !errors.Is(errAuth, store.ErrNoResult) {
			return "", "", errors.Wrap(errAuth, "Failed to fetch refresh token")
		}
		if createErr := store.SavePersonAuth(ctx, &refreshToken); createErr != nil {
			return "", "", errors.Wrap(errAuth, "Failed to create new refresh token")
		}
	}
	return accessToken, refreshToken.RefreshToken, nil
}

func getTokenKey(_ *jwt.Token) (any, error) {
	return []byte(config.HTTP.CookieKey), nil
}

// onTokenRefresh handles generating new token pairs to access the api
// NOTE: All error code paths must return 401 (Unauthorized)
func onTokenRefresh() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var rt userToken
		if errBind := ctx.BindJSON(&rt); errBind != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			logger.Error("Malformed user token", zap.Error(errBind))
			return
		}
		if rt.RefreshToken == "" {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		var auth store.PersonAuth
		if authError := store.GetPersonAuthByRefreshToken(ctx, rt.RefreshToken, &auth); authError != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		newAccessToken, newRefreshToken, errToken := makeTokens(ctx, auth.SteamId)
		if errToken != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			logger.Error("Failed to create access token pair", zap.Error(errToken))
			return
		}
		responseOK(ctx, http.StatusOK, userToken{
			AccessToken:  newAccessToken,
			RefreshToken: newRefreshToken,
		})
	}
}

type userToken struct {
	AccessToken  string `json:"accessToken,omitempty"`
	RefreshToken string `json:"refreshToken"`
}

type personAuthClaims struct {
	SteamID int64 `json:"steam_id"`
	jwt.StandardClaims
}

type serverAuthClaims struct {
	ServerId int `json:"server_id"`
	jwt.StandardClaims
}

const authTokenLifetimeDuration = time.Hour * 24 * 30 // 1 month

func newUserJWT(steamID steamid.SID64) (string, error) {
	t0 := config.Now()
	claims := &personAuthClaims{
		SteamID: steamID.Int64(),
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: t0.Add(authTokenLifetimeDuration).Unix(),
			IssuedAt:  t0.Unix(),
		},
	}
	tokenWithClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, errSigned := tokenWithClaims.SignedString([]byte(config.HTTP.CookieKey))
	if errSigned != nil {
		return "", errors.Wrap(errSigned, "Failed create signed string")
	}
	return signedToken, nil
}

func newServerJWT(serverId int) (string, error) {
	t0 := config.Now()
	claims := &serverAuthClaims{
		ServerId: serverId,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: t0.Add(authTokenLifetimeDuration).Unix(),
			IssuedAt:  t0.Unix(),
		},
	}
	tokenWithClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, errSigned := tokenWithClaims.SignedString([]byte(config.HTTP.CookieKey))
	if errSigned != nil {
		return "", errors.Wrap(errSigned, "Failed create signed string")
	}
	return signedToken, nil
}

// authMiddleware handles client authentication to the HTTP & websocket api.
// websocket clients must pass the key as a query parameter called "token"
func authMiddleware(level store.Privilege) gin.HandlerFunc {
	type header struct {
		Authorization string `header:"Authorization"`
	}
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
			if len(pcs) != 2 && level >= store.PUser {
				ctx.AbortWithStatus(http.StatusForbidden)
				return
			}
			token = pcs[1]
		}
		if level >= store.PUser {
			sid, errFromToken := sid64FromJWTToken(token)
			if errFromToken != nil {
				if errors.Is(errFromToken, consts.ErrExpired) {
					ctx.AbortWithStatus(http.StatusUnauthorized)
					return
				}
				logger.Error("Failed to load sid from access token", zap.Error(errFromToken))
				ctx.AbortWithStatus(http.StatusForbidden)
				return
			}
			loggedInPerson := store.NewPerson(sid)
			if errGetPerson := app.PersonBySID(ctx, sid, &loggedInPerson); errGetPerson != nil {
				logger.Error("Failed to load person during auth", zap.Error(errGetPerson))
				ctx.AbortWithStatus(http.StatusForbidden)
				return
			}
			if level > loggedInPerson.PermissionLevel {
				ctx.AbortWithStatus(http.StatusForbidden)
				return
			}
			bp := store.NewBannedPerson()
			if errBan := store.GetBanBySteamID(ctx, sid, &bp, false); errBan != nil {
				if !errors.Is(errBan, store.ErrNoResult) {
					logger.Error("Failed to fetch authed user ban", zap.Error(errBan))
				}
			}
			notifications, errNotifications := store.GetPersonNotifications(ctx, sid)
			if errNotifications != nil && !errors.Is(errNotifications, store.ErrNoResult) {
				logger.Error("Failed to fetch user notifications", zap.Error(errNotifications))
			}
			profile := model.UserProfile{
				SteamID:         loggedInPerson.SteamID,
				CreatedOn:       loggedInPerson.CreatedOn,
				UpdatedOn:       loggedInPerson.UpdatedOn,
				PermissionLevel: loggedInPerson.PermissionLevel,
				DiscordID:       loggedInPerson.DiscordID,
				Name:            loggedInPerson.PersonaName,
				Avatar:          loggedInPerson.Avatar,
				AvatarFull:      loggedInPerson.AvatarFull,
				Muted:           loggedInPerson.Muted,
				BanID:           bp.Ban.BanID,
				Notifications:   notifications,
			}
			ctx.Set(ctxKeyUserProfile, profile)
		}
		ctx.Next()
	}
}

func sid64FromJWTToken(token string) (steamid.SID64, error) {
	claims := &personAuthClaims{}
	tkn, errParseClaims := jwt.ParseWithClaims(token, claims, getTokenKey)
	if errParseClaims != nil {
		if errors.Is(errParseClaims, jwt.ErrSignatureInvalid) {
			return 0, consts.ErrAuthentication
		}
		e, ok := errParseClaims.(*jwt.ValidationError)
		if ok && e.Errors == jwt.ValidationErrorExpired {
			return 0, consts.ErrExpired
		}
		return 0, consts.ErrAuthentication
	}
	if !tkn.Valid {
		return 0, consts.ErrAuthentication
	}
	sid := steamid.SID64(claims.SteamID)
	if !sid.Valid() {
		return 0, consts.ErrAuthentication
	}
	return sid, nil
}
