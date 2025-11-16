package httphelper

import (
	"errors"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/gofrs/uuid/v5"
	"github.com/gorilla/schema"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type ServerAuthenticator interface {
	Middleware(ctx *gin.Context)
}

type Authenticator interface {
	Middleware(level permission.Privilege) gin.HandlerFunc
	MiddlewareWS(level permission.Privilege) gin.HandlerFunc
}

func BindJSON[T any](ctx *gin.Context) (T, bool) { //nolint:ireturn
	var value T
	if err := ctx.ShouldBindJSON(&value); err != nil {
		var validationErrs validator.ValidationErrors
		if errors.As(err, &validationErrs) {
			SetError(ctx, NewAPIError(http.StatusBadRequest, validationErrs))
		} else {
			SetError(ctx, NewAPIError(http.StatusBadRequest, ErrBadRequest))
		}

		return value, false
	}

	return value, true
}

// Decoder is a package global because it caches
// meta-data about structs, and an instance can be shared safely.
var Decoder = schema.NewDecoder() //nolint:gochecknoglobals

func BindQuery(ctx *gin.Context, target any) bool {
	if errBind := Decoder.Decode(target, ctx.Request.URL.Query()); errBind != nil {
		SetError(ctx,
			NewAPIErrorf(http.StatusInternalServerError,
				errors.Join(errBind, ErrBadRequest),
				"Could not decode query params"))

		return false
	}

	return true
}

// NewClient allocates a preconfigured *http.Client.
func NewClient() *http.Client {
	c := &http.Client{
		Timeout: time.Second * 10,
	}

	return c
}

func GetSID64Param(ctx *gin.Context, key string) (steamid.SteamID, bool) {
	i, found := GetInt64Param(ctx, key)
	if !found {
		return steamid.SteamID{}, false
	}

	sid := steamid.New(i)
	if !sid.Valid() {
		SetError(ctx, NewAPIErrorf(http.StatusBadRequest, steamid.ErrInvalidSID,
			"%s contains an invalid Steam ID: %s", key, ctx.Param(key)))

		return steamid.SteamID{}, false
	}

	return sid, true
}

func GetInt64Param(ctx *gin.Context, key string) (int64, bool) {
	valueStr := ctx.Param(key)
	if valueStr == "" {
		SetError(ctx, NewAPIErrorf(http.StatusBadRequest, ErrParamKeyMissing,
			"Cannot read value for param: %s", key))

		return 0, false
	}

	value, valueErr := strconv.ParseInt(valueStr, 10, 64)
	if valueErr != nil {
		SetError(ctx, NewAPIErrorf(http.StatusBadRequest, errors.Join(valueErr, ErrParamParse),
			"Must be a valid integer: %s", key))

		return 0, false
	}

	if value <= 0 {
		SetError(ctx, NewAPIErrorf(http.StatusBadRequest, ErrParamInvalid,
			"Integer value cannot be negative: %s", key))

		return 0, false
	}

	return value, true
}

func GetIntParam(ctx *gin.Context, key string) (int, bool) {
	valueStr := ctx.Param(key)
	if valueStr == "" {
		SetError(ctx, NewAPIErrorf(http.StatusBadRequest, ErrParamKeyMissing,
			"Cannot read value for param: %s", key))

		return 0, false
	}

	return stringutil.StringToIntOrZero(valueStr), true
}

func GetStringParam(ctx *gin.Context, key string) (string, bool) {
	valueStr := ctx.Param(key)
	if valueStr == "" {
		SetError(ctx, NewAPIErrorf(http.StatusBadRequest, ErrParamKeyMissing,
			"Cannot find param: %s", key))

		return "", false
	}

	return valueStr, true
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

func GetDefaultFloat64(s string, def float64) float64 {
	if s != "" {
		l, errLat := strconv.ParseFloat(s, 64)
		if errLat != nil {
			return def
		}

		return l
	}

	return def
}

// HasPrivilege first checks if the steamId matches one of the provided allowedSteamIds, otherwise it will check
// if the user has appropriate privilege levels.
// Error responses are handled by this function, no further action needs to take place in the handlers.
func HasPrivilege(person person.Info, allowedSteamIDs steamid.Collection, minPrivilege permission.Privilege) bool {
	if slices.Contains(allowedSteamIDs, person.GetSteamID()) {
		return true
	}

	return person.HasPermission(minPrivilege)
}

type ResultsCount struct {
	Count int64 `json:"count"`
}

func NewServer(listenAddr string, handler http.Handler) *http.Server {
	httpServer := &http.Server{
		Addr:           listenAddr,
		Handler:        handler,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   120 * time.Second,
		MaxHeaderBytes: 1 << 20,
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

type RequestQuery struct {
	Query string `json:"query" url:"query"`
}
