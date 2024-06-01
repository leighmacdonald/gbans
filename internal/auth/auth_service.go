package auth

import (
	"log/slog"
	"net/http"
	"net/url"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/log"
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

	engine.GET("/auth/callback", handler.onSteamOIDCCallback())

	authGrp := engine.Group("/")
	{
		// authed
		env := authGrp.Use(authUsecase.AuthMiddleware(domain.PUser))

		env.GET("/api/auth/logout", handler.onAPILogout())
	}
}

func (h authHandler) onSteamOIDCCallback() gin.HandlerFunc {
	nonceStore := openid.NewSimpleNonceStore()
	discoveryCache := &noOpDiscoveryCache{}
	oidRx := regexp.MustCompile(`^https://steamcommunity\.com/openid/id/(\d+)$`)

	return func(ctx *gin.Context) {
		var idStr string

		referralURL := httphelper.Referral(ctx)
		conf := h.configUsecase.Config()
		fullURL := conf.ExternalURL + ctx.Request.URL.String()

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
			slog.Error("Failed to match oid format provided")
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

		token, errToken := h.authUsecase.MakeToken(ctx, conf.HTTPCookieKey, sid)
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
		query.Set("token", token.Access)
		query.Set("next_url", referralURL)
		parsedURL.RawQuery = query.Encode()

		parsedExternal, errExternal := url.Parse(conf.ExternalURL)
		if errExternal != nil {
			ctx.Redirect(302, referralURL)
			slog.Error("Failed to parse ext url", log.ErrAttr(errExternal))

			return
		}

		// TODO max age checks
		ctx.SetSameSite(http.SameSiteStrictMode)
		ctx.SetCookie(
			fingerprintCookieName,
			token.Fingerprint,
			int(domain.AuthTokenDuration.Seconds()),
			"/api",
			parsedExternal.Hostname(),
			conf.General.Mode == domain.ReleaseMode,
			true)

		ctx.Redirect(302, parsedURL.String())
		slog.Info("User logged in",
			slog.String("sid64", sid.String()),
			slog.String("name", person.PersonaName),
			slog.Int("permission_level", int(person.PermissionLevel)))
	}
}

const fingerprintCookieName = "fingerprint"

func (h authHandler) onAPILogout() gin.HandlerFunc {
	conf := h.configUsecase.Config()

	return func(ctx *gin.Context) {
		fingerprint, errCookie := ctx.Cookie(fingerprintCookieName)
		if errCookie != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		parsedExternal, errExternal := url.Parse(conf.ExternalURL)
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
