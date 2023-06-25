// Package web implements the HTTP and websocket services for the frontend client and backend server.
package app

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"go.uber.org/zap"
)

const ctxKeyUserProfile = "user_profile"

func (app *App) StartHTTP(ctx context.Context) error {
	app.log.Info("Service status changed", zap.String("state", "ready"))
	defer app.log.Info("Service status changed", zap.String("state", "stopped"))
	if app.conf.General.Mode == config.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	httpServer := newHTTPServer(ctx, app)
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		if errShutdown := httpServer.Shutdown(shutdownCtx); errShutdown != nil { //nolint:contextcheck
			app.log.Error("Error shutting down http service", zap.Error(errShutdown))
		}
	}()
	return httpServer.ListenAndServe()
}

func bind(ctx *gin.Context, target any) bool {
	if errBind := ctx.BindJSON(&target); errBind != nil {
		responseErr(ctx, http.StatusBadRequest, gin.H{
			"error": "Invalid request parameters",
		})
		return false
	}
	return true
}

func newHTTPServer(ctx context.Context, app *App) *http.Server {
	httpServer := &http.Server{
		Addr:           app.conf.HTTP.Addr(),
		Handler:        createRouter(ctx, app),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	if app.conf.HTTP.TLS {
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

func currentUserProfile(ctx *gin.Context) model.UserProfile {
	maybePerson, found := ctx.Get(ctxKeyUserProfile)
	if !found {
		return model.NewUserProfile(0)
	}
	person, ok := maybePerson.(model.UserProfile)
	if !ok {
		return model.NewUserProfile(0)
	}
	return person
}

// checkPrivilege first checks if the steamId matches one of the provided allowedSteamIds, otherwise it will check
// if the user has appropriate privilege levels.
// Error responses are handled by this function, no further action needs to take place in the handlers.
func checkPrivilege(ctx *gin.Context, person model.UserProfile, allowedSteamIds steamid.Collection, minPrivilege consts.Privilege) bool {
	for _, steamID := range allowedSteamIds {
		if steamID == person.SteamID {
			return true
		}
	}
	if person.PermissionLevel >= minPrivilege {
		return true
	}
	responseErrUser(ctx, http.StatusUnauthorized, nil, consts.ErrPermissionDenied.Error())
	return false
}
