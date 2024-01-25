package http_helper

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

const ctxKeyUserProfile = "user_profile"

var (
	ErrParamKeyMissing = errors.New("param key not found")
	ErrParamParse      = errors.New("failed to parse param value")
	ErrParamInvalid    = errors.New("param value invalid")
)

type ApiError struct {
	Message string `json:"message"`
}

func ResponseErr(ctx *gin.Context, statusCode int, err error) {
	userErr := "API Error"
	if err != nil {
		userErr = err.Error()
	}

	ctx.JSON(statusCode, ApiError{Message: userErr})
}

func Bind(ctx *gin.Context, log *zap.Logger, target any) bool {
	if errBind := ctx.BindJSON(&target); errBind != nil {
		ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)
		log.Error("Failed to bind request", zap.Error(errBind))

		return false
	}

	return true
}

func CurrentUserProfile(ctx *gin.Context) domain.UserProfile {
	maybePerson, found := ctx.Get(ctxKeyUserProfile)
	if !found {
		return domain.NewUserProfile("")
	}

	person, ok := maybePerson.(domain.UserProfile)
	if !ok {
		return domain.NewUserProfile("")
	}

	return person
}

func GetSID64Param(c *gin.Context, key string) (steamid.SID64, error) {
	i, errGetParam := GetInt64Param(c, key)
	if errGetParam != nil {
		return "", errGetParam
	}

	sid := steamid.New(i)
	if !sid.Valid() {
		return "", errs.ErrInvalidSID
	}

	return sid, nil
}

func GetInt64Param(ctx *gin.Context, key string) (int64, error) {
	valueStr := ctx.Param(key)
	if valueStr == "" {
		return 0, fmt.Errorf("%w: %s", ErrParamKeyMissing, key)
	}

	value, valueErr := strconv.ParseInt(valueStr, 10, 64)
	if valueErr != nil {
		return 0, ErrParamParse
	}

	if value <= 0 {
		return 0, fmt.Errorf("%w: %s", ErrParamInvalid, key)
	}

	return value, nil
}

func GetIntParam(ctx *gin.Context, key string) (int, error) {
	valueStr := ctx.Param(key)
	if valueStr == "" {
		return 0, fmt.Errorf("%w: %s", ErrParamKeyMissing, key)
	}

	return util.StringToInt(valueStr), nil
}

func GetUUIDParam(ctx *gin.Context, key string) (uuid.UUID, error) {
	valueStr := ctx.Param(key)
	if valueStr == "" {
		return uuid.UUID{}, fmt.Errorf("%w: %s", ErrParamKeyMissing, key)
	}

	parsedUUID, errString := uuid.FromString(valueStr)
	if errString != nil {
		return uuid.UUID{}, errors.Join(errString, ErrParamParse)
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
