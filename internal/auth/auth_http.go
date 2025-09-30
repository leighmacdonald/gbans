package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/yohcop/openid-go"
)

type authHandler struct {
	authentication *Authentication
	config         *config.Configuration
	persons        *person.Persons
	tfAPI          *thirdparty.TFAPI
}

func NewAuthHandler(engine *gin.Engine, auth *Authentication, config *config.Configuration,
	person *person.Persons, tfAPI *thirdparty.TFAPI,
) {
	handler := &authHandler{
		authentication: auth,
		config:         config,
		persons:        person,
		tfAPI:          tfAPI,
	}

	engine.GET("/auth/callback", handler.onSteamOIDCCallback())

	authGrp := engine.Group("/")
	{
		// authed
		env := authGrp.Use(auth.Middleware(permission.PUser))

		env.GET("/api/auth/logout", handler.onAPILogout())
	}
}

func (h *authHandler) onSteamOIDCCallback() gin.HandlerFunc {
	var (
		handlerName    = log.HandlerName(1)
		nonceStore     = openid.NewSimpleNonceStore()
		discoveryCache = &noOpDiscoveryCache{}
		oidRx          = regexp.MustCompile(`^https://steamcommunity\.com/openid/id/(\d+)$`)
	)

	return func(ctx *gin.Context) {
		var idStr string

		referralURL := httphelper.Referral(ctx)
		conf := h.config.Config()
		fullURL := conf.ExternalURL + ctx.Request.URL.String()

		if conf.Debug.SkipOpenIDValidation {
			// Pull the sid out of the query without doing a signature check
			values, errParse := url.Parse(fullURL)
			if errParse != nil {
				ctx.Redirect(302, referralURL)
				slog.Error("Failed to parse url", log.ErrAttr(errParse), handlerName)

				return
			}

			idStr = values.Query().Get("openid.identity")
		} else {
			openID, errVerify := openid.Verify(fullURL, discoveryCache, nonceStore)
			if errVerify != nil {
				ctx.Redirect(302, referralURL)
				slog.Error("Error verifying openid auth response", log.ErrAttr(errVerify), handlerName)

				return
			}

			idStr = openID
		}

		match := oidRx.FindStringSubmatch(idStr)
		if match == nil || len(match) != 2 {
			ctx.Redirect(302, referralURL)
			slog.Error("Failed to match oid format provided", handlerName)

			return
		}

		sid := steamid.New(match[1])
		if !sid.Valid() {
			ctx.Redirect(302, referralURL)
			slog.Error("Received invalid steamid", handlerName)

			return
		}

		// TODO FIX
		_, _ = h.persons.GetOrCreatePersonBySteamID(ctx, nil, sid)

		fetchedPerson, errPerson := h.persons.GetPersonBySteamID(ctx, nil, sid)
		if errPerson != nil {
			ctx.Redirect(302, referralURL)
			slog.Error("Failed to create or load user profile", log.ErrAttr(errPerson), handlerName)
		}

		if fetchedPerson.Expired() {
			if errGetProfile := person.UpdatePlayerSummary(ctx, &fetchedPerson, h.tfAPI); errGetProfile != nil {
				slog.Error("Failed to fetch user profile on login", log.ErrAttr(errGetProfile), handlerName)
			} else {
				if errSave := h.persons.SavePerson(ctx, nil, &fetchedPerson); errSave != nil {
					slog.Error("Failed to save summary update", log.ErrAttr(errSave), handlerName)
				}
			}
		}

		token, errToken := h.authentication.MakeToken(ctx, conf.HTTPCookieKey, sid)
		if errToken != nil {
			ctx.Redirect(302, referralURL)
			slog.Error("Failed to create access token pair", log.ErrAttr(errToken), handlerName)

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
			slog.Error("Failed to parse ext url", log.ErrAttr(errExternal), handlerName)

			return
		}

		// TODO max age checks
		ctx.SetSameSite(http.SameSiteStrictMode)
		ctx.SetCookie(
			FingerprintCookieName,
			token.Fingerprint,
			int(AuthTokenDuration.Seconds()),
			"/api",
			parsedExternal.Hostname(),
			strings.HasPrefix(strings.ToLower(conf.ExternalURL), "https://"),
			true)

		ctx.Redirect(302, parsedURL.String())

		sentry.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "auth",
			Message:  "User logged in " + fetchedPerson.SteamID.String(),
			Level:    sentry.LevelWarning,
		})

		slog.Info("User logged in",
			slog.String("sid64", sid.String()),
			slog.String("name", fetchedPerson.PersonaName),
			slog.Int("permission_level", int(fetchedPerson.PermissionLevel)), handlerName)
	}
}

func (h *authHandler) onAPILogout() gin.HandlerFunc {
	conf := h.config.Config()

	return func(ctx *gin.Context) {
		fingerprint, errCookie := ctx.Cookie(FingerprintCookieName)
		if errCookie != nil {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(errCookie, httphelper.ErrBadRequest),
				"Failed to find fingerprint cookie: %s", FingerprintCookieName))

			return
		}

		parsedExternal, errExternal := url.Parse(conf.ExternalURL)
		if errExternal != nil {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(errExternal, httphelper.ErrInternal),
				"Invalid external url: %s", conf.ExternalURL))

			return
		}

		ctx.SetCookie(FingerprintCookieName, "", -1, "/api",
			parsedExternal.Hostname(), conf.General.Mode == config.ReleaseMode, true)

		personAuth := PersonAuth{}
		if errGet := h.authentication.GetPersonAuthByRefreshToken(ctx, fingerprint, &personAuth); errGet != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errGet, httphelper.ErrInternal)))

			return
		}

		if errDelete := h.authentication.DeletePersonAuth(ctx, personAuth.PersonAuthID); errDelete != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errDelete, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})

		sentry.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "auth",
			Message:  "User logged out " + personAuth.SteamID.String(),
			Level:    sentry.LevelWarning,
		})

		sentry.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetUser(sentry.User{})
		})
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
