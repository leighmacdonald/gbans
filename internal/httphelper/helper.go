package httphelper

import (
	"net/http"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

// NewClient allocates a preconfigured *http.Client.
func NewClient() *http.Client {
	c := &http.Client{
		Timeout: time.Second * 10,
	}

	return c
}

func GetUUIDParam(ctx *gin.Context, key string) (uuid.UUID, bool) {
	valueStr := ctx.Param(key)
	if valueStr == "" {
		SetError(ctx, NewAPIErrorf(http.StatusBadRequest, ErrParamKeyMissing,
			"Cannot find param: %s", key))

		return uuid.UUID{}, false
	}

	parsedUUID, errString := uuid.FromString(valueStr)
	if errString != nil {
		SetError(ctx, NewAPIErrorf(http.StatusBadRequest, ErrParamParse, "Supplied value is not a valid UUID: %s", valueStr))

		return uuid.UUID{}, false
	}

	return parsedUUID, true
}

// HasPrivilege first checks if the steamId matches one of the provided allowedSteamIds, otherwise it will check
// if the user has appropriate privilege levels.
// Error responses are handled by this function, no further action needs to take place in the handlers.
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

func Referral(ctx *gin.Context) string {
	referralURL, found := ctx.GetQuery("return_url")
	if !found {
		referralURL = "/"
	}

	return referralURL
}
