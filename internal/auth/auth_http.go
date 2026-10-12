package auth

import (
	"context"
	"log/slog"
	"net"
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

type TokenGenerator interface {
	MakeUserToken(id person.BaseUser) (string, string, error)
	ValidateUserToken(tokenStr string, fingerprint string) (steamid.SteamID, error)
}

// sessionCookieDomain scopes session cookies to the request's own host.
// Dotted non-IP hostnames keep a Domain cookie (existing production
// behaviour); localhost and IP literals use host-only cookies because
// browsers reject Domain cookies for them.
// isLocalHost reports whether hostname is a loopback name or address. Only
// there are Secure cookies optional (plain-http local dev); everywhere else
// Secure is always set, even if the backend itself only sees plain http
// behind a proxy that strips scheme headers.
func isLocalHost(hostname string) bool {
	name := strings.ToLower(strings.TrimSuffix(hostname, "."))

	if name == "localhost" || strings.HasSuffix(name, ".localhost") {
		return true
	}

	if ip := net.ParseIP(name); ip != nil {
		return ip.IsLoopback()
	}

	return false
}

func sessionCookieDomain(req *http.Request) (string, bool) {
	base, err := url.Parse(httphelper.RequestBaseURL(req))
	if err != nil {
		return "", false
	}

	hostname := base.Hostname()

	secure := true
	if isLocalHost(hostname) {
		secure = strings.EqualFold(base.Scheme, "https")
	}

	if net.ParseIP(hostname) != nil || !strings.Contains(hostname, ".") {
		return "", secure
	}

	return hostname, secure
}

// newSessionCookie builds a session cookie scoped to the request's own host
// so login/logout work on every host the backend serves (local dev ports,
// LAN IPs, proxied domains), not just the configured external URL.
func newSessionCookie(req *http.Request, name string, value string, maxAge int) *http.Cookie {
	domain, secure := sessionCookieDomain(req)

	return &http.Cookie{ //nolint:gosec
		Name:     name,
		Value:    value,
		MaxAge:   maxAge,
		Path:     "/",
		Domain:   domain,
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
}

// RoleAssigner assigns the default user role to steam IDs without role
// assignments. Implemented by roles.Roles.
type RoleAssigner interface {
	AssignUserRole(ctx context.Context, steamID steamid.SteamID) error
}

type authHandler struct {
	*Authentication

	config         *config.Configuration
	tfAPI          thirdparty.APIProvider
	notif          notification.Notifier
	tokenGenerator TokenGenerator
	roleAssigner   RoleAssigner
}

func NewAuthHandler(mux *http.ServeMux, auth *Authentication, config *config.Configuration,
	tfAPI thirdparty.APIProvider, notif notification.Notifier, tokenGenerator TokenGenerator, roleAssigner RoleAssigner,
) {
	handler := &authHandler{
		Authentication: auth,
		config:         config,
		tfAPI:          tfAPI,
		notif:          notif,
		tokenGenerator: tokenGenerator,
		roleAssigner:   roleAssigner,
	}

	mux.HandleFunc("GET /auth/callback", handler.onSteamOIDCCallback())
	mux.HandleFunc("POST /api/auth/logout", handler.onAPILogout())
}

func (h *authHandler) onSteamOIDCCallback() http.HandlerFunc {
	var (
		nonceStore     = openid.NewSimpleNonceStore()
		discoveryCache = &noOpDiscoveryCache{}
		oidRx          = regexp.MustCompile(`^https://steamcommunity\.com/openid/id/(\d+)$`)
	)

	return func(res http.ResponseWriter, req *http.Request) {
		var idStr string

		referralURL := httphelper.Referral(req)
		conf := h.config.Config()
		// Verify against the browser-visible URL, not the static external
		// URL: the signed return_to always points at the host the user
		// actually used, which may differ per deployment/host.
		fullURL := httphelper.RequestBaseURL(req) + req.URL.String()

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

		fetchedPerson, errPerson := h.persons.RefreshFromSteam(req.Context(), sid)
		if errPerson != nil {
			http.Redirect(res, req, referralURL, http.StatusFound) //nolint:gosec
			slog.Error("Failed to create or load user profile", slog.String("error", errPerson.Error()))
		}

		if errAssign := h.roleAssigner.AssignUserRole(req.Context(), sid); errAssign != nil {
			slog.Error("Failed to assign default user role", slog.String("error", errAssign.Error()))
		}

		accessToken, fingerprint, errToken := h.tokenGenerator.MakeUserToken(fetchedPerson)
		if errToken != nil {
			http.Redirect(res, req, referralURL, http.StatusFound) //nolint:gosec
			slog.Error("Failed to create access token pair", slog.String("error", errToken.Error()))

			return
		}

		if ip := parseIP(req.RemoteAddr); ip != nil {
			if errSave := h.SavePersonAuth(req.Context(), NewPersonAuth(fetchedPerson.SteamID, ip, fingerprint)); errSave != nil {
				slog.Error("Failed to save auth record for revocation", slog.String("error", errSave.Error()))
			}
		}

		parsedURL, errParse := url.Parse("/login/success")
		if errParse != nil {
			http.Redirect(res, req, referralURL, http.StatusFound) //nolint:gosec

			return
		}

		exchangeCode := h.CreateExchangeCode(accessToken)
		if exchangeCode == "" {
			http.Redirect(res, req, referralURL, http.StatusFound) //nolint:gosec
			slog.Error("Failed to create exchange code")

			return
		}

		query := parsedURL.Query()
		query.Set("code", exchangeCode)
		query.Set("next_url", referralURL)
		parsedURL.RawQuery = query.Encode()

		http.SetCookie(res, newSessionCookie(req, FingerprintCookieName, fingerprint, int(TokenDuration.Seconds())))
		http.SetCookie(res, newSessionCookie(req, JWTCookieName, accessToken, int(TokenDuration.Seconds())))

		http.Redirect(res, req, parsedURL.String(), http.StatusFound) //nolint:gosec

		sentry.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "auth",
			Message:  "" + fetchedPerson.SteamID.String(),
			Level:    sentry.LevelWarning,
		})

		go h.notif.Send(notification.NewDiscord(conf.Discord.LogChannelID, loginMessage(fetchedPerson)))

		slog.Debug("User logged in",
			slog.String("sid64", sid.String()),
			slog.String("name", fetchedPerson.GetName()))
	}
}

func (h *authHandler) onAPILogout() http.HandlerFunc {
	conf := h.config.Config()

	return func(res http.ResponseWriter, req *http.Request) {
		var sid steamid.SteamID

		fingerprint, errCookie := req.Cookie(FingerprintCookieName)
		if errCookie == nil {
			http.SetCookie(res, newSessionCookie(req, FingerprintCookieName, "", -1))
			http.SetCookie(res, newSessionCookie(req, JWTCookieName, "", -1))
		}

		authHeader := req.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") && errCookie == nil {
			token := strings.TrimPrefix(authHeader, "Bearer ")

			validatedSID, errValidation := h.tokenGenerator.ValidateUserToken(token, fingerprint.Value)
			if errValidation == nil {
				sid = validatedSID
			}
		}

		if sid.Valid() {
			if errDelete := h.DeletePersonAuthBySteamID(req.Context(), sid); errDelete != nil {
				slog.Error("Failed to delete person auth on logout", slog.String("error", errDelete.Error()))
			}
		}

		httphelper.RespondJSON(res, http.StatusOK, map[string]string{})

		if sid.Valid() {
			go func(steamID steamid.SteamID) { //nolint:gosec
				sentry.AddBreadcrumb(&sentry.Breadcrumb{
					Category: "auth",
					Message:  "User logged out " + steamID.String(),
					Level:    sentry.LevelWarning,
				})

				sentry.ConfigureScope(func(scope *sentry.Scope) {
					scope.SetUser(sentry.User{})
				})

				logoutCtx := context.Background()
				player, errPerson := h.persons.GetOrCreatePersonBySteamID(logoutCtx, steamID)
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

func parseIP(remoteAddr string) net.IP {
	host, _, errSplit := net.SplitHostPort(remoteAddr)
	if errSplit != nil {
		return net.ParseIP(remoteAddr)
	}

	return net.ParseIP(host)
}
