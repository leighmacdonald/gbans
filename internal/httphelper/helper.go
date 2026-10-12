package httphelper

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
)

func NewClient() *http.Client {
	c := &http.Client{
		Timeout: time.Second * 10,
	}

	return c
}

func GetUUIDParam(r *http.Request, key string) (uuid.UUID, bool) {
	valueStr := r.PathValue(key)
	if valueStr == "" {
		return uuid.UUID{}, false
	}

	parsedUUID, errString := uuid.FromString(valueStr)
	if errString != nil {
		return uuid.UUID{}, false
	}

	return parsedUUID, true
}

func NewServer(listenAddr string, handler http.Handler) *http.Server {
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)

	httpServer := &http.Server{
		Addr:           listenAddr,
		Handler:        handler,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   120 * time.Second,
		MaxHeaderBytes: 1 << 20,
		Protocols:      protocols,
	}

	return httpServer
}

func Referral(r *http.Request) string {
	referralURL := r.URL.Query().Get("return_url")
	if referralURL == "" {
		referralURL = "/"
	}

	return safeRedirectURL(referralURL)
}

// RequestBaseURL rebuilds the public base URL (scheme://host[:port]) for the
// incoming request, honouring reverse-proxy headers. Auth flows (OpenID
// verification, session cookies, exchange origin checks) must use this
// instead of the static external URL: a single backend can serve several
// hosts (local dev ports, LAN IPs, Cloudflare-fronted domains) and the
// browser-visible URL is the only one signatures and cookies will match.
func RequestBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	if forwardedProto := r.Header.Get("X-Forwarded-Proto"); forwardedProto != "" {
		scheme = strings.ToLower(strings.TrimSpace(strings.Split(forwardedProto, ",")[0]))
	}

	host := r.Host
	if forwardedHost := r.Header.Get("X-Forwarded-Host"); forwardedHost != "" {
		host = strings.TrimSpace(strings.Split(forwardedHost, ",")[0])
	}

	if host == "" {
		host = "localhost"
	}

	return scheme + "://" + host
}

func safeRedirectURL(rawURL string) string {
	if rawURL == "" {
		return "/"
	}

	u, err := url.Parse(rawURL)
	if err != nil || u.Host != "" || !strings.HasPrefix(rawURL, "/") {
		return "/"
	}

	return rawURL
}
