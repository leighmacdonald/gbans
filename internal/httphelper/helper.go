package httphelper

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
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
}

func Bind(ctx *gin.Context, target any) bool {
	if errBind := ctx.BindJSON(&target); errBind != nil {
		ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)
		slog.Error("Failed to bind request", log.ErrAttr(errBind))

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
func HasPrivilege(person domain.PersonInfo, allowedSteamIds steamid.Collection, minPrivilege domain.Privilege) bool {
	for _, steamID := range allowedSteamIds {
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

func NewHTTPServer(tlsEnabled bool, listenAddr string, handler http.Handler) *http.Server {
	httpServer := &http.Server{
		Addr:           listenAddr,
		Handler:        handler,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   120 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if tlsEnabled {
		tlsVar := &tls.Config{
			// Only use curves which have assembly implementations
			CurvePreferences: []tls.CurveID{
				tls.CurveP256,
				tls.X25519, // Go 1.8 only
			},
			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305, // Go 1.8 only
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,   // Go 1.8 only
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		}
		httpServer.TLSConfig = tlsVar
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
