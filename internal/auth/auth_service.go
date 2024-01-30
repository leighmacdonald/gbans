package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/http_helper"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/yohcop/openid-go"
	"go.uber.org/zap"
)

type AuthHandler struct {
	authUsecase   domain.AuthUsecase
	configUsecase domain.ConfigUsecase
	personUsecase domain.PersonUsecase
	log           *zap.Logger
}

func NewAuthHandler(log *zap.Logger, engine *gin.Engine, au domain.AuthUsecase, cu domain.ConfigUsecase, pu domain.PersonUsecase) {
	handler := &AuthHandler{
		authUsecase:   au,
		configUsecase: cu,
		personUsecase: pu,
		log:           log.Named("auth"),
	}

	engine.GET("/auth/callback", handler.onOpenIDCallback())

	env := engine.Use(au.AuthMiddleware(domain.PUser))
	{
		// authed
		env.POST("/api/auth/refresh", handler.onTokenRefresh())
		env.GET("/api/auth/discord", handler.onOAuthDiscordCallback())
		env.GET("/api/auth/logout", handler.onAPILogout())
	}
}

func (h AuthHandler) onOpenIDCallback() gin.HandlerFunc {
	nonceStore := openid.NewSimpleNonceStore()
	discoveryCache := &noOpDiscoveryCache{}
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())
	oidRx := regexp.MustCompile(`^https://steamcommunity\.com/openid/id/(\d+)$`)

	return func(ctx *gin.Context) {
		var idStr string

		referralURL := http_helper.Referral(ctx)
		conf := h.configUsecase.Config()
		fullURL := conf.General.ExternalURL + ctx.Request.URL.String()

		if conf.Debug.SkipOpenIDValidation {
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

		person := domain.NewPerson(sid)
		if errPerson := h.personUsecase.GetOrCreatePersonBySteamID(ctx, sid, &person); errPerson != nil {
			log.Error("Failed to create or load user profile", zap.Error(errPerson))
			ctx.Redirect(302, referralURL)
		}

		if person.Expired() {
			if errGetProfile := thirdparty.UpdatePlayerSummary(ctx, &person); errGetProfile != nil {
				log.Error("Failed to fetch user profile", zap.Error(errGetProfile))
				ctx.Redirect(302, referralURL)

				return
			}

			if errSave := h.personUsecase.SavePerson(ctx, &person); errSave != nil {
				log.Error("Failed to save summary update", zap.Error(errSave))
			}
		}

		tokens, errToken := h.authUsecase.MakeTokens(ctx, conf.HTTP.CookieKey, sid, true)
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
		query.Set("refresh", tokens.Refresh)
		query.Set("token", tokens.Access)
		query.Set("next_url", referralURL)
		parsedURL.RawQuery = query.Encode()

		parsedExternal, errExternal := url.Parse(conf.General.ExternalURL)
		if errExternal != nil {
			ctx.Redirect(302, referralURL)
			log.Error("Failed to parse ext url", zap.Error(errExternal))

			return
		}
		// TODO max age checks
		ctx.SetSameSite(http.SameSiteStrictMode)
		ctx.SetCookie(
			fingerprintCookieName,
			tokens.Fingerprint,
			int(domain.RefreshTokenDuration.Seconds()),
			"/api",
			parsedExternal.Hostname(),
			conf.General.Mode == domain.ReleaseMode,
			true)

		ctx.Redirect(302, parsedURL.String())
		log.Info("User logged in",
			zap.Int64("sid64", sid.Int64()),
			zap.String("name", person.PersonaName),
			zap.Int("permission_level", int(person.PermissionLevel)))
	}
}

const fingerprintCookieName = "fingerprint"

// onTokenRefresh handles generating new token pairs to access the api
// NOTE: All error code paths must return 401 (Unauthorized).
func (h AuthHandler) onTokenRefresh() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())
	conf := h.configUsecase.Config()

	return func(ctx *gin.Context) {
		fingerprint, errCookie := ctx.Cookie(fingerprintCookieName)
		if errCookie != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			log.Warn("Failed to get fingerprint cookie", zap.Error(errCookie))

			return
		}

		refreshTokenString, errToken := h.authUsecase.TokenFromHeader(ctx, false)
		if errToken != nil {
			ctx.AbortWithStatus(http.StatusForbidden)

			return
		}

		userClaims := domain.UserAuthClaims{}

		refreshToken, errParseClaims := jwt.ParseWithClaims(refreshTokenString, &userClaims, h.authUsecase.MakeGetTokenKey(conf.HTTP.CookieKey))
		if errParseClaims != nil {
			if errors.Is(errParseClaims, jwt.ErrSignatureInvalid) {
				log.Error("jwt signature invalid!", zap.Error(errParseClaims))
				ctx.AbortWithStatus(http.StatusUnauthorized)

				return
			}

			ctx.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		claims, ok := refreshToken.Claims.(*domain.UserAuthClaims)
		if !ok || !refreshToken.Valid {
			ctx.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		hash := FingerprintHash(fingerprint)
		if claims.Fingerprint != hash {
			ctx.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		var personAuth domain.PersonAuth
		if authError := h.authUsecase.GetPersonAuthByRefreshToken(ctx, fingerprint, &personAuth); authError != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		tokens, errMakeToken := h.authUsecase.MakeTokens(ctx, conf.HTTP.CookieKey, personAuth.SteamID, false)
		if errMakeToken != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			log.Error("Failed to create access token pair", zap.Error(errMakeToken))

			return
		}

		ctx.JSON(http.StatusOK, userToken{
			AccessToken: tokens.Access,
		})
	}
}

func (h AuthHandler) onOAuthDiscordCallback() gin.HandlerFunc {
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

	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())
	client := util.NewHTTPClient()
	conf := h.configUsecase.Config()

	fetchDiscordID := func(ctx context.Context, accessToken string) (string, error) {
		req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, "https://discord.com/api/users/@me", nil)
		if errReq != nil {
			return "", errors.Join(errReq, domain.ErrCreateRequest)
		}

		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
		resp, errResp := client.Do(req)

		if errResp != nil {
			return "", errors.Join(errResp, domain.ErrRequestPerform)
		}

		defer func() {
			_ = resp.Body.Close()
		}()

		var details discordUserDetail
		if errJSON := json.NewDecoder(resp.Body).Decode(&details); errJSON != nil {
			return "", errors.Join(errJSON, domain.ErrRequestDecode)
		}

		return details.ID, nil
	}

	fetchToken := func(ctx context.Context, code string) (string, error) {
		// v, _ := go_oauth_pkce_code_verifier.CreateCodeVerifierFromBytes([]byte(code))
		form := url.Values{}
		form.Set("client_id", conf.Discord.AppID)
		form.Set("client_secret", conf.Discord.AppSecret)
		form.Set("redirect_uri", conf.ExtURLRaw("/login/discord"))
		form.Set("code", code)
		form.Set("grant_type", "authorization_code")
		// form.Set("state", state.String())
		form.Set("scope", "identify")
		req, errReq := http.NewRequestWithContext(ctx, http.MethodPost, "https://discord.com/api/oauth2/token", strings.NewReader(form.Encode()))

		if errReq != nil {
			return "", errors.Join(errReq, domain.ErrCreateRequest)
		}

		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		resp, errResp := client.Do(req)
		if errResp != nil {
			return "", errors.Join(errResp, domain.ErrRequestPerform)
		}

		defer func() {
			_ = resp.Body.Close()
		}()

		var atr accessTokenResp
		if errJSON := json.NewDecoder(resp.Body).Decode(&atr); errJSON != nil {
			return "", errors.Join(errJSON, domain.ErrRequestDecode)
		}

		if atr.AccessToken == "" {
			return "", domain.ErrEmptyToken
		}

		return atr.AccessToken, nil
	}

	return func(ctx *gin.Context) {
		code := ctx.Query("code")
		if code == "" {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, nil)
			log.Error("Failed to get code from query")

			return
		}

		token, errToken := fetchToken(ctx, code)
		if errToken != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, nil)
			log.Error("Failed to fetch token", zap.Error(errToken))

			return
		}

		discordID, errID := fetchDiscordID(ctx, token)
		if errID != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, nil)
			log.Error("Failed to fetch discord ID", zap.Error(errID))

			return
		}

		if discordID == "" {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Empty discord id received")

			return
		}

		var discordPerson domain.Person
		if errDp := h.personUsecase.GetPersonByDiscordID(ctx, discordID, &discordPerson); errDp != nil {
			if !errors.Is(errDp, domain.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusInternalServerError, nil)

				return
			}
		}

		if discordPerson.DiscordID != "" {
			http_helper.ResponseErr(ctx, http.StatusConflict, nil)
			log.Error("Failed to update persons discord id")

			return
		}

		sid := http_helper.CurrentUserProfile(ctx).SteamID

		var person domain.Person
		if errPerson := h.personUsecase.GetPersonBySteamID(ctx, sid, &person); errPerson != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		if person.Expired() {
			if errGetProfile := thirdparty.UpdatePlayerSummary(ctx, &person); errGetProfile != nil {
				log.Error("Failed to fetch user profile", zap.Error(errGetProfile))
				http_helper.ResponseErr(ctx, http.StatusInternalServerError, nil)

				return
			}

			if errSave := h.personUsecase.SavePerson(ctx, &person); errSave != nil {
				log.Error("Failed to save player summary update", zap.Error(errSave))
			}
		}

		person.DiscordID = discordID

		if errSave := h.personUsecase.SavePerson(ctx, &person); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		ctx.JSON(http.StatusOK, nil)

		log.Info("Discord account linked successfully",
			zap.String("discord_id", discordID), zap.Int64("sid64", sid.Int64()))
	}
}

func (h AuthHandler) onAPILogout() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())
	conf := h.configUsecase.Config()

	return func(ctx *gin.Context) {
		fingerprint, errCookie := ctx.Cookie(fingerprintCookieName)
		if errCookie != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		parsedExternal, errExternal := url.Parse(conf.General.ExternalURL)
		if errExternal != nil {
			ctx.Status(http.StatusInternalServerError)
			log.Error("Failed to parse ext url", zap.Error(errExternal))

			return
		}

		ctx.SetCookie(fingerprintCookieName, "", -1, "/api",
			parsedExternal.Hostname(), conf.General.Mode == domain.ReleaseMode, true)

		personAuth := domain.PersonAuth{}
		if errGet := h.authUsecase.GetPersonAuthByRefreshToken(ctx, fingerprint, &personAuth); errGet != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, nil)
			log.Warn("Failed to load person via fingerprint")

			return
		}

		if errDelete := h.authUsecase.DeletePersonAuth(ctx, personAuth.PersonAuthID); errDelete != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to delete person auth on logout", zap.Error(errDelete))

			return
		}

		ctx.Status(http.StatusNoContent)
	}
}

type userToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

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
