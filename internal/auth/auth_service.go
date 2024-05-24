package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/yohcop/openid-go"
)

type authHandler struct {
	authUsecase   domain.AuthUsecase
	configUsecase domain.ConfigUsecase
	personUsecase domain.PersonUsecase
}

func NewAuthHandler(engine *gin.Engine, authUsecase domain.AuthUsecase, configUsecase domain.ConfigUsecase,
	personUsecase domain.PersonUsecase,
) {
	handler := &authHandler{
		authUsecase:   authUsecase,
		configUsecase: configUsecase,
		personUsecase: personUsecase,
	}

	engine.GET("/auth/callback", handler.onOpenIDCallback())

	authGrp := engine.Group("/")
	{
		// authed
		env := authGrp.Use(authUsecase.AuthMiddleware(domain.PUser))
		env.POST("/api/auth/refresh", handler.onTokenRefresh())
		env.GET("/api/auth/discord", handler.onOAuthDiscordCallback())
		env.GET("/api/auth/logout", handler.onAPILogout())
	}
}

func (h authHandler) onOpenIDCallback() gin.HandlerFunc {
	nonceStore := openid.NewSimpleNonceStore()
	discoveryCache := &noOpDiscoveryCache{}
	oidRx := regexp.MustCompile(`^https://steamcommunity\.com/openid/id/(\d+)$`)

	return func(ctx *gin.Context) {
		var idStr string

		referralURL := httphelper.Referral(ctx)
		conf := h.configUsecase.Config()
		fullURL := conf.General.ExternalURL + ctx.Request.URL.String()

		if conf.Debug.SkipOpenIDValidation {
			// Pull the sid out of the query without doing a signature check
			values, errParse := url.Parse(fullURL)
			if errParse != nil {
				slog.Error("Failed to parse url", log.ErrAttr(errParse))
				ctx.Redirect(302, referralURL)

				return
			}

			idStr = values.Query().Get("openid.identity")
		} else {
			openID, errVerify := openid.Verify(fullURL, discoveryCache, nonceStore)
			if errVerify != nil {
				slog.Error("Error verifying openid auth response", log.ErrAttr(errVerify))
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

		sid := steamid.New(match[1])
		if !sid.Valid() {
			slog.Error("Received invalid steamid")
			ctx.Redirect(302, referralURL)

			return
		}

		person, errPerson := h.personUsecase.GetOrCreatePersonBySteamID(ctx, sid)
		if errPerson != nil {
			slog.Error("Failed to create or load user profile", log.ErrAttr(errPerson))
			ctx.Redirect(302, referralURL)
		}

		if person.Expired() {
			if errGetProfile := thirdparty.UpdatePlayerSummary(ctx, &person); errGetProfile != nil {
				slog.Error("Failed to fetch user profile on login", log.ErrAttr(errGetProfile))
			} else {
				if errSave := h.personUsecase.SavePerson(ctx, &person); errSave != nil {
					slog.Error("Failed to save summary update", log.ErrAttr(errSave))
				}
			}
		}

		tokens, errToken := h.authUsecase.MakeTokens(ctx, conf.HTTP.CookieKey, sid, true)
		if errToken != nil {
			ctx.Redirect(302, referralURL)
			slog.Error("Failed to create access token pair", log.ErrAttr(errToken))

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
			slog.Error("Failed to parse ext url", log.ErrAttr(errExternal))

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
		slog.Info("User logged in",
			slog.Int64("sid64", sid.Int64()),
			slog.String("name", person.PersonaName),
			slog.Int("permission_level", int(person.PermissionLevel)))
	}
}

const fingerprintCookieName = "fingerprint"

// onTokenRefresh handles generating new token pairs to access the api
// NOTE: All error code paths must return 401 (Unauthorized).
func (h authHandler) onTokenRefresh() gin.HandlerFunc {
	conf := h.configUsecase.Config()

	return func(ctx *gin.Context) {
		fingerprint, errCookie := ctx.Cookie(fingerprintCookieName)
		if errCookie != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			slog.Warn("Failed to get fingerprint cookie", log.ErrAttr(errCookie))

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
				slog.Error("jwt signature invalid!", log.ErrAttr(errParseClaims))
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
			slog.Error("Failed to create access token pair", log.ErrAttr(errMakeToken))

			return
		}

		ctx.JSON(http.StatusOK, userToken{
			AccessToken: tokens.Access,
		})
	}
}

func (h authHandler) onOAuthDiscordCallback() gin.HandlerFunc {
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

	client := util.NewHTTPClient()
	conf := h.configUsecase.Config()

	fetchDiscordID := func(ctx context.Context, accessToken string) (string, error) {
		req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, "https://discord.com/api/users/@me", nil)
		if errReq != nil {
			return "", errors.Join(errReq, domain.ErrCreateRequest)
		}

		req.Header.Add("Authorization", "Bearer "+accessToken)
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
			httphelper.ResponseErr(ctx, http.StatusBadRequest, nil)
			slog.Error("Failed to get code from query")

			return
		}

		token, errToken := fetchToken(ctx, code)
		if errToken != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, nil)
			slog.Error("Failed to fetch token", log.ErrAttr(errToken))

			return
		}

		discordID, errID := fetchDiscordID(ctx, token)
		if errID != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, nil)
			slog.Error("Failed to fetch discord ID", log.ErrAttr(errID))

			return
		}

		if discordID == "" {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, nil)
			slog.Error("Empty discord id received")

			return
		}

		discordPerson, errDp := h.personUsecase.GetPersonByDiscordID(ctx, discordID)
		if errDp != nil {
			if !errors.Is(errDp, domain.ErrNoResult) {
				httphelper.ResponseErr(ctx, http.StatusInternalServerError, nil)

				return
			}
		}

		if discordPerson.DiscordID != "" {
			httphelper.ResponseErr(ctx, http.StatusConflict, nil)
			slog.Error("Failed to update persons discord id")

			return
		}

		sid := httphelper.CurrentUserProfile(ctx).SteamID

		person, errPerson := h.personUsecase.GetPersonBySteamID(ctx, sid)
		if errPerson != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		if person.Expired() {
			if errGetProfile := thirdparty.UpdatePlayerSummary(ctx, &person); errGetProfile != nil {
				slog.Error("Failed to fetch user profile", log.ErrAttr(errGetProfile))
			} else {
				if errSave := h.personUsecase.SavePerson(ctx, &person); errSave != nil {
					slog.Error("Failed to save player summary update", log.ErrAttr(errSave))
				}
			}
		}

		person.DiscordID = discordID

		if errSave := h.personUsecase.SavePerson(ctx, &person); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		ctx.JSON(http.StatusOK, nil)

		slog.Info("Discord account linked successfully",
			slog.String("discord_id", discordID), slog.Int64("sid64", sid.Int64()))
	}
}

func (h authHandler) onAPILogout() gin.HandlerFunc {
	conf := h.configUsecase.Config()

	return func(ctx *gin.Context) {
		fingerprint, errCookie := ctx.Cookie(fingerprintCookieName)
		if errCookie != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		parsedExternal, errExternal := url.Parse(conf.General.ExternalURL)
		if errExternal != nil {
			ctx.Status(http.StatusInternalServerError)
			slog.Error("Failed to parse ext url", log.ErrAttr(errExternal))

			return
		}

		ctx.SetCookie(fingerprintCookieName, "", -1, "/api",
			parsedExternal.Hostname(), conf.General.Mode == domain.ReleaseMode, true)

		personAuth := domain.PersonAuth{}
		if errGet := h.authUsecase.GetPersonAuthByRefreshToken(ctx, fingerprint, &personAuth); errGet != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, nil)
			slog.Warn("Failed to load person via fingerprint")

			return
		}

		if errDelete := h.authUsecase.DeletePersonAuth(ctx, personAuth.PersonAuthID); errDelete != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, nil)
			slog.Error("Failed to delete person auth on logout", log.ErrAttr(errDelete))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
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
