package httphelper

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/gorilla/schema"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/convert"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

func recoveryHandler() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, err interface{}) {
		slog.Error("Recovery error:", slog.String("err", fmt.Sprintf("%v", err)))

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Something went wrong",
		})
	})
}

func Bind(ctx *gin.Context, target any) bool {
	if errBind := ctx.BindJSON(&target); errBind != nil {
		SetError(ctx, NewAPIError(http.StatusBadRequest, domain.ErrBadRequest))

		return false
	}

	return true
}

// Set a Decoder instance as a package global, because it caches
// meta-data about structs, and an instance can be shared safely.
var decoder = schema.NewDecoder() //nolint:gochecknoglobals

func BindQuery(ctx *gin.Context, target any) bool {
	if errBind := decoder.Decode(target, ctx.Request.URL.Query()); errBind != nil {
		SetError(ctx,
			NewAPIErrorf(http.StatusInternalServerError,
				errors.Join(errBind, domain.ErrBadRequest),
				"Could not decode query params"))

		return false
	}

	return true
}

func CurrentUserProfile(ctx *gin.Context) domain.UserProfile {
	maybePerson, found := ctx.Get(ctxKeyUserProfile)
	if !found {
		return domain.NewUserProfile(steamid.SteamID{})
	}

	person, ok := maybePerson.(domain.UserProfile)
	if !ok {
		return domain.NewUserProfile(steamid.SteamID{})
	}

	return person
}

func GetSID64Param(ctx *gin.Context, key string) (steamid.SteamID, bool) {
	i, found := GetInt64Param(ctx, key)
	if !found {
		return steamid.SteamID{}, false
	}

	sid := steamid.New(i)
	if !sid.Valid() {
		_ = ctx.Error(NewAPIErrorf(ctx, http.StatusBadRequest, nil, domain.ErrInvalidSID,
			"%s contains an invalid Steam ID: %s", key, ctx.Param(key)))

		return steamid.SteamID{}, false
	}

	return sid, true
}

func GetInt64Param(ctx *gin.Context, key string) (int64, bool) {
	valueStr := ctx.Param(key)
	if valueStr == "" {
		_ = ctx.Error(NewAPIErrorf(ctx, http.StatusBadRequest, domain.ErrParamKeyMissing,
			"Cannot read value for param: %s", key))

		return 0, false
	}

	value, valueErr := strconv.ParseInt(valueStr, 10, 64)
	if valueErr != nil {
		_ = ctx.Error(NewAPIErrorf(ctx, http.StatusBadRequest, domain.ErrParamParse,
			"Must be a valid integer: %s", key))

		return 0, false
	}

	if value <= 0 {
		_ = ctx.Error(NewAPIErrorf(ctx, http.StatusBadRequest, domain.ErrParamInvalid,
			"Integer value cannot be negative: %s", key))

		return 0, false
	}

	return value, true
}

func GetIntParam(ctx *gin.Context, key string) (int, bool) {
	valueStr := ctx.Param(key)
	if valueStr == "" {
		_ = ctx.Error(NewAPIErrorf(ctx, http.StatusBadRequest, domain.ErrParamKeyMissing,
			"Cannot read value for param: %s", key))

		return 0, false
	}

	return convert.StringToInt(valueStr), true
}

func GetStringParam(ctx *gin.Context, key string) (string, bool) {
	valueStr := ctx.Param(key)
	if valueStr == "" {
		_ = ctx.Error(NewAPIErrorf(ctx, http.StatusBadRequest, domain.ErrParamKeyMissing,
			"Cannot find param: %s", key))

		return "", false
	}

	return valueStr, true
}

func GetUUIDParam(ctx *gin.Context, key string) (uuid.UUID, bool) {
	valueStr := ctx.Param(key)
	if valueStr == "" {
		_ = ctx.Error(NewAPIErrorf(ctx, http.StatusBadRequest, domain.ErrParamKeyMissing,
			"Cannot find param: %s", key))

		return uuid.UUID{}, false
	}

	parsedUUID, errString := uuid.FromString(valueStr)
	if errString != nil {
		_ = ctx.Error(NewAPIErrorf(ctx, http.StatusBadRequest, domain.ErrParamParse, "Supplied value is not a valid UUID: %s", valueStr))

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
func HasPrivilege(person domain.PersonInfo, allowedSteamIDs steamid.Collection, minPrivilege domain.Privilege) bool {
	for _, steamID := range allowedSteamIDs {
		if steamID == person.GetSteamID() {
			return true
		}
	}

	return person.HasPermission(minPrivilege)
}

type ResultsCount struct {
	Count int64 `json:"count"`
}

const ctxKeyUserProfile = "user_profile"

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
