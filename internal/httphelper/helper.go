package httphelper

import (
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/steamid/v4/steamid"
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

func HasPrivilege(person person.BaseUser, allowedSteamIDs steamid.Collection, minPrivilege permission.Privilege) bool {
	if slices.Contains(allowedSteamIDs, person.GetSteamID()) {
		return true
	}

	return person.HasPermission(minPrivilege)
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
