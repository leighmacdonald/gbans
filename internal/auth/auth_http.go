package auth

import (
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/yohcop/openid-go"
)

func safeRedirectURL(rawURL string) string {
	if rawURL == "" || rawURL[0] == '/' {
		return rawURL
	}

	return "/"
}

type TokenGenerator interface {
	MakeUserToken(id person.BaseUser) (string, string, error)
	ValidateUserToken(tokenStr string, fingerprint string) (steamid.SteamID, error)
}

type authHandler struct {
	*Authentication

	config         *config.Configuration
	tfAPI          thirdparty.APIProvider
	notif          notification.Notifier
	tokenGenerator TokenGenerator
}

func NewAuthHandler(mux *http.ServeMux, auth *Authentication, config *config.Configuration,
	tfAPI thirdparty.APIProvider, notif notification.Notifier, tokenGenerator TokenGenerator,
) {
	handler := &authHandler{
		Authentication: auth,
		config:         config,
		tfAPI:          tfAPI,
		notif:          notif,
		tokenGenerator: tokenGenerator,
	}

	mux.HandleFunc("GET /auth/callback", handler.onSteamOIDCCallback())
	mux.HandleFunc("GET /api/auth/logout", handler.onAPILogout())
}

func (h *authHandler) onSteamOIDCCallback() http.HandlerFunc {
	var (
		nonceStore     = openid.NewSimpleNonceStore()
		discoveryCache = &noOpDiscoveryCache{}
		oidRx          = regexp.MustCompile(`^https://steamcommunity\.com/openid/id/(\d+)$`)
	)

	return func(res http.ResponseWriter, req *http.Request) {
		var idStr string

		referralURL := safeRedirectURL(httphelper.Referral(req))
		conf := h.config.Config()
		fullURL := conf.ExternalURL + req.URL.String()

		if conf.Debug.SkipOpenIDValidation {
			values, errParse := url.Parse(fullURL)
			if errParse != nil {
				http.Redirect(res, req, referralURL, http.StatusFound) //nolint:gosec
				slog.Error("Failed to parse url", slog.String("error", errParse.Error()))

				return
			}

			idStr = values.Query().Get("openid.identity")
		} else {
			openID, errVerify := openid.Verify(fullURL, discoveryCache, nonceStore)
			if errVerify != nil {
				http.Redirect(res, req, referralURL, http.StatusFound) //nolint:gosec
				slog.Error("Error verifying openid auth response", slog.String("error", errVerify.Error()))

				return
			}

			idStr = openID
		}

		match := oidRx.FindStringSubmatch(idStr)
		if match == nil || len(match) != 2 {
			http.Redirect(res, req, referralURL, http.StatusFound) //nolint:gosec
			slog.Error("Failed to match oid format provided")

			return
		}

		sid := steamid.New(match[1])
		if !sid.Valid() {
			http.Redirect(res, req, referralURL, http.StatusFound) //nolint:gosec
			slog.Error("Received invalid steamid")

			return
		}

		fetchedPerson, errPerson := h.persons.GetOrCreatePersonBySteamID(req.Context(), sid)
		if errPerson != nil {
			http.Redirect(res, req, referralURL, http.StatusFound) //nolint:gosec
			slog.Error("Failed to create or load user profile", slog.String("error", errPerson.Error()))
		}

		accessToken, fingerprint, errToken := h.tokenGenerator.MakeUserToken(fetchedPerson)
		if errToken != nil {
			http.Redirect(res, req, referralURL, http.StatusFound) //nolint:gosec
			slog.Error("Failed to create access token pair", slog.String("error", errToken.Error()))

			return
		}

		parsedURL, errParse := url.Parse("/login/success")
		if errParse != nil {
			http.Redirect(res, req, referralURL, http.StatusFound) //nolint:gosec

			return
		}

		query := parsedURL.Query()
		query.Set("token", accessToken)
		query.Set("next_url", referralURL)
		parsedURL.RawQuery = query.Encode()

		parsedExternal, errExternal := url.Parse(conf.ExternalURL)
		if errExternal != nil {
			http.Redirect(res, req, referralURL, http.StatusFound) //nolint:gosec
			slog.Error("Failed to parse ext url", slog.String("error", errExternal.Error()))

			return
		}

		http.SetCookie(res, &http.Cookie{ //nolint:gosec
			Name:     FingerprintCookieName,
			Value:    fingerprint,
			MaxAge:   int(TokenDuration.Seconds()),
			Path:     "/connect",
			Domain:   parsedExternal.Hostname(),
			Secure:   strings.HasPrefix(strings.ToLower(conf.ExternalURL), "https://"),
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})

		http.Redirect(res, req, parsedURL.String(), http.StatusFound) //nolint:gosec

		sentry.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "auth",
			Message:  "" + fetchedPerson.SteamID.String(),
			Level:    sentry.LevelWarning,
		})

		go h.notif.Send(notification.NewDiscord(conf.Discord.LogChannelID, loginMessage(fetchedPerson)))

		slog.Debug("User logged in",
			slog.String("sid64", sid.String()),
			slog.String("name", fetchedPerson.GetName()),
			slog.Int("permission_level", int(fetchedPerson.PermissionLevel)))
	}
}

func (h *authHandler) onAPILogout() http.HandlerFunc {
	conf := h.config.Config()

	return func(res http.ResponseWriter, req *http.Request) {
		var sid steamid.SteamID

		fingerprint, errCookie := req.Cookie(FingerprintCookieName)
		if errCookie == nil {
			parsedExternal, errExternal := url.Parse(conf.ExternalURL)
			if errExternal == nil {
				http.SetCookie(res, &http.Cookie{ //nolint:gosec
					Name:     FingerprintCookieName,
					Value:    "",
					MaxAge:   -1,
					Path:     "/connect",
					Domain:   parsedExternal.Hostname(),
					Secure:   strings.HasPrefix(strings.ToLower(conf.ExternalURL), "https://"),
					HttpOnly: true,
					SameSite: http.SameSiteStrictMode,
				})
			}
		}

		authHeader := req.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") && errCookie == nil {
			token := strings.TrimPrefix(authHeader, "Bearer ")

			validatedSID, errValidation := h.tokenGenerator.ValidateUserToken(token, fingerprint.Value)
			if errValidation == nil {
				sid = validatedSID
			}
		}

		httphelper.RespondJSON(res, http.StatusOK, map[string]string{})

		if sid.Valid() {
			go func(steamID steamid.SteamID) {
				sentry.AddBreadcrumb(&sentry.Breadcrumb{
					Category: "auth",
					Message:  "User logged out " + steamID.String(),
					Level:    sentry.LevelWarning,
				})

				sentry.ConfigureScope(func(scope *sentry.Scope) {
					scope.SetUser(sentry.User{})
				})

				player, errPerson := h.persons.GetOrCreatePersonBySteamID(req.Context(), steamID)
				if errPerson != nil {
					slog.Error("Failed to load user for logout notification", slog.String("error", errPerson.Error()))

					return
				}

				h.notif.Send(notification.NewDiscord(conf.Discord.LogChannelID, logoutMessage(player)))
			}(sid)
		}
	}
}

type noOpDiscoveryCache struct{}

func (n *noOpDiscoveryCache) Put(_ string, _ openid.DiscoveredInfo) {}

func (n *noOpDiscoveryCache) Get(_ string) openid.DiscoveredInfo {
	return nil
}
