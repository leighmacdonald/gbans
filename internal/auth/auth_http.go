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
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/yohcop/openid-go"
)

type authHandler struct {
	*Authentication

	config *config.Configuration
	tfAPI  thirdparty.APIProvider
	notif  notification.Notifier
}

func NewAuthHandler(engine *gin.Engine, auth *Authentication, config *config.Configuration,
	tfAPI thirdparty.APIProvider, notif notification.Notifier,
) {
	handler := &authHandler{
		Authentication: auth,
		config:         config,
		tfAPI:          tfAPI,
		notif:          notif,
	}

	engine.GET("/auth/callback", handler.onSteamOIDCCallback())

	authGrp := engine.Group("/")
	{
		// authed
		env := authGrp.Use(auth.Middleware(permission.User))

		env.GET("/api/auth/logout", handler.onAPILogout())
	}
}

func (h *authHandler) onSteamOIDCCallback() gin.HandlerFunc {
	var (
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
				slog.Error("Failed to parse url", slog.String("error", errParse.Error()))

				return
			}

			idStr = values.Query().Get("openid.identity")
		} else {
			openID, errVerify := openid.Verify(fullURL, discoveryCache, nonceStore)
			if errVerify != nil {
				ctx.Redirect(302, referralURL)
				slog.Error("Error verifying openid auth response", slog.String("error", errVerify.Error()))

				return
			}

			idStr = openID
		}

		match := oidRx.FindStringSubmatch(idStr)
		if match == nil || len(match) != 2 {
			ctx.Redirect(302, referralURL)
			slog.Error("Failed to match oid format provided")

			return
		}

		sid := steamid.New(match[1])
		if !sid.Valid() {
			ctx.Redirect(302, referralURL)
			slog.Error("Received invalid steamid")

			return
		}

		fetchedPerson, errPerson := h.persons.GetOrCreatePersonBySteamID(ctx, sid)
		if errPerson != nil {
			ctx.Redirect(302, referralURL)
			slog.Error("Failed to create or load user profile", slog.String("error", errPerson.Error()))
		}

		token, errToken := h.MakeToken(ctx, conf.HTTPCookieKey, sid)
		if errToken != nil {
			ctx.Redirect(302, referralURL)
			slog.Error("Failed to create access token pair", slog.String("error", errToken.Error()))

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
			slog.Error("Failed to parse ext url", slog.String("error", errExternal.Error()))

			return
		}

		ctx.SetSameSite(http.SameSiteStrictMode)
		ctx.SetCookie(
			FingerprintCookieName,
			token.Fingerprint,
			int(TokenDuration.Seconds()),
			"/api",
			parsedExternal.Hostname(),
			strings.HasPrefix(strings.ToLower(conf.ExternalURL), "https://"),
			true)

		ctx.Redirect(302, parsedURL.String())

		sentry.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "auth",
			Message:  "" + fetchedPerson.SteamID.String(),
			Level:    sentry.LevelWarning,
		})

		go h.notif.Send(notification.NewDiscord(conf.Discord.LogChannelID, loginMessage(fetchedPerson)))

		slog.Info("User logged in",
			slog.String("sid64", sid.String()),
			slog.String("name", fetchedPerson.GetName()),
			slog.Int("permission_level", int(fetchedPerson.PermissionLevel)))
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

		token, errToken := h.TokenFromHeader(ctx, false)
		if errToken != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		sid, errFromToken := h.Sid64FromJWTToken(token, h.cookieKey, fingerprint)
		if errFromToken != nil {
			if errors.Is(errFromToken, ErrExpired) {
				ctx.AbortWithStatus(http.StatusUnauthorized)

				return
			}

			slog.Error("Failed to load sid from access token", slog.String("error", errFromToken.Error()))
			ctx.AbortWithStatus(http.StatusForbidden)

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})

		go func(steamID steamid.SteamID) {
			sentry.AddBreadcrumb(&sentry.Breadcrumb{
				Category: "auth",
				Message:  "User logged out " + steamID.String(),
				Level:    sentry.LevelWarning,
			})

			sentry.ConfigureScope(func(scope *sentry.Scope) {
				scope.SetUser(sentry.User{})
			})
			player, errPerson := h.persons.GetOrCreatePersonBySteamID(ctx, steamID)
			if errPerson != nil {
				slog.Error("Failed to create or load user profile", slog.String("error", errPerson.Error()))

				return
			}
			h.notif.Send(notification.NewDiscord(conf.Discord.LogChannelID, logoutMessage(player)))
		}(sid)
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
