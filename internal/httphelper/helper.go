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
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type APIError struct {
	Message string `json:"message"`
}

func ResponseErr(ctx *gin.Context, statusCode int, err error) {
	userErr := "API Error"
	if err != nil {
		userErr = err.Error()
	}

	ctx.JSON(statusCode, APIError{Message: userErr})

	if err != nil {
		slog.Error("Response error", slog.Int("code", statusCode), log.ErrAttr(err))
	}
}

func Bind(ctx *gin.Context, target any) bool {
	if errBind := ctx.BindJSON(&target); errBind != nil {
		ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)
		slog.Error("Failed to bind request", log.ErrAttr(errBind))

		return false
	}

	return true
}

// Set a Decoder instance as a package global, because it caches
// meta-data about structs, and an instance can be shared safely.
var decoder = schema.NewDecoder() //nolint:gochecknoglobals

func BindQuery(ctx *gin.Context, target any) bool {
	if errBind := decoder.Decode(target, ctx.Request.URL.Query()); errBind != nil {
		ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)
		slog.Error("Failed to bind query request", log.ErrAttr(errBind))

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

func GetSID64Param(c *gin.Context, key string) (steamid.SteamID, error) {
	i, errGetParam := GetInt64Param(c, key)
	if errGetParam != nil {
		return steamid.SteamID{}, errGetParam
	}

	sid := steamid.New(i)
	if !sid.Valid() {
		return steamid.SteamID{}, domain.ErrInvalidSID
	}

	return sid, nil
}

func GetInt64Param(ctx *gin.Context, key string) (int64, error) {
	valueStr := ctx.Param(key)
	if valueStr == "" {
		HandleErrBadRequest(ctx)

		return 0, fmt.Errorf("%w: %s", domain.ErrParamKeyMissing, key)
	}

	value, valueErr := strconv.ParseInt(valueStr, 10, 64)
	if valueErr != nil {
		HandleErrBadRequest(ctx)

		return 0, domain.ErrParamParse
	}

	if value <= 0 {
		HandleErrBadRequest(ctx)

		return 0, fmt.Errorf("%w: %s", domain.ErrParamInvalid, key)
	}

	return value, nil
}

func GetIntParam(ctx *gin.Context, key string) (int, error) {
	valueStr := ctx.Param(key)
	if valueStr == "" {
		return 0, fmt.Errorf("%w: %s", domain.ErrParamKeyMissing, key)
	}

	return util.StringToInt(valueStr), nil
}

func GetStringParam(ctx *gin.Context, key string) (string, error) {
	valueStr := ctx.Param(key)
	if valueStr == "" {
		return "", fmt.Errorf("%w: %s", domain.ErrParamKeyMissing, key)
	}

	return valueStr, nil
}

func GetUUIDParam(ctx *gin.Context, key string) (uuid.UUID, error) {
	valueStr := ctx.Param(key)
	if valueStr == "" {
		return uuid.UUID{}, fmt.Errorf("%w: %s", domain.ErrParamKeyMissing, key)
	}

	parsedUUID, errString := uuid.FromString(valueStr)
	if errString != nil {
		return uuid.UUID{}, errors.Join(errString, domain.ErrParamParse)
	}

	return parsedUUID, nil
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

func ServerIDFromCtx(ctx *gin.Context) int {
	serverIDUntyped, ok := ctx.Get("server_id")
	if !ok {
		return 0
	}

	serverID, castOk := serverIDUntyped.(int)
	if !castOk {
		return 0
	}

	return serverID
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

func NewHTTPServer(listenAddr string, handler http.Handler) *http.Server {
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
