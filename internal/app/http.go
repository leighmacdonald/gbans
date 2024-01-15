package app

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/common"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

const ctxKeyUserProfile = "user_profile"

type apiError struct {
	Message string `json:"message"`
}

func responseErr(ctx *gin.Context, statusCode int, err error) {
	userErr := "API Error"
	if err != nil {
		userErr = err.Error()
	}

	ctx.JSON(statusCode, apiError{Message: userErr})
}

func bind(ctx *gin.Context, log *zap.Logger, target any) bool {
	if errBind := ctx.BindJSON(&target); errBind != nil {
		responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)
		log.Error("Failed to bind request", zap.Error(errBind))

		return false
	}

	return true
}

func newLazyResult(count int64, data any) LazyResult {
	if count == 0 {
		return LazyResult{0, []interface{}{}}
	}

	return LazyResult{Count: count, Data: data}
}

func newHTTPServer(ctx context.Context, app *App) *http.Server {
	conf := app.config()

	httpServer := &http.Server{
		Addr:           conf.HTTP.Addr(),
		Handler:        createRouter(ctx, app),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   120 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if conf.HTTP.TLS {
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

// userProfile is the model used in the webui representing the logged-in user.
type userProfile struct {
	SteamID         steamid.SID64    `json:"steam_id"`
	CreatedOn       time.Time        `json:"created_on"`
	UpdatedOn       time.Time        `json:"updated_on"`
	PermissionLevel consts.Privilege `json:"permission_level"`
	DiscordID       string           `json:"discord_id"`
	Name            string           `json:"name"`
	Avatarhash      string           `json:"avatarhash"`
	BanID           int64            `json:"ban_id"`
	Muted           bool             `json:"muted"`
}

func (p userProfile) GetDiscordID() string {
	return p.DiscordID
}

func (p userProfile) GetName() string {
	return p.Name
}

func (p userProfile) GetAvatar() common.AvatarLinks {
	return common.NewAvatarLinks(p.Avatarhash)
}

func (p userProfile) GetSteamID() steamid.SID64 {
	return p.SteamID
}

func (p userProfile) Path() string {
	return fmt.Sprintf("/profile/%d", p.SteamID.Int64())
}

// newUserProfile allocates a new default person instance.
func newUserProfile(sid64 steamid.SID64) userProfile {
	t0 := time.Now()

	return userProfile{
		SteamID:         sid64,
		CreatedOn:       t0,
		UpdatedOn:       t0,
		PermissionLevel: consts.PUser,
		Name:            "Guest",
	}
}

func currentUserProfile(ctx *gin.Context) userProfile {
	maybePerson, found := ctx.Get(ctxKeyUserProfile)
	if !found {
		return newUserProfile("")
	}

	person, ok := maybePerson.(userProfile)
	if !ok {
		return newUserProfile("")
	}

	return person
}

// checkPrivilege first checks if the steamId matches one of the provided allowedSteamIds, otherwise it will check
// if the user has appropriate privilege levels.
// Error responses are handled by this function, no further action needs to take place in the handlers.
func checkPrivilege(ctx *gin.Context, person userProfile, allowedSteamIds steamid.Collection, minPrivilege consts.Privilege) bool {
	for _, steamID := range allowedSteamIds {
		if steamID == person.SteamID {
			return true
		}
	}

	if person.PermissionLevel >= minPrivilege {
		return true
	}

	ctx.JSON(http.StatusForbidden, consts.ErrPermissionDenied.Error())

	return false
}

type ResultsCount struct {
	Count int64 `json:"count"`
}

type LazyResult struct {
	Count int64 `json:"count"`
	Data  any   `json:"data"`
}
