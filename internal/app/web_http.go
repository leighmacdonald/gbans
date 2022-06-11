// Package web implements the HTTP and websocket services for the frontend client and backend server.
package app

import (
	"context"
	"crypto/tls"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

const ctxKeyUserProfile = "user_profile"

type WebHandler interface {
	ListenAndServe(context.Context) error
}

type web struct {
	httpServer         *http.Server
	botSendMessageChan chan discordPayload
}

func (web web) ListenAndServe(ctx context.Context) error {
	log.WithFields(log.Fields{"service": "web", "status": "ready"}).Infof("Service status changed")
	defer log.WithFields(log.Fields{"service": "web", "status": "stopped"}).Infof("Service status changed")
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		if errShutdown := web.httpServer.Shutdown(shutdownCtx); errShutdown != nil {
			log.Errorf("Error shutting down http service: %v", errShutdown)
		}
	}()
	return web.httpServer.ListenAndServe()
}

// NewWeb sets up the router and starts the API HTTP handlers
// This function blocks on the context
func NewWeb(database store.Store, botSendMessageChan chan discordPayload) (WebHandler, error) {
	var httpServer *http.Server
	if config.General.Mode == config.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	httpServer = &http.Server{
		Addr:           config.HTTP.Addr(),
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	if config.HTTP.TLS {
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
	webHandler := web{httpServer: httpServer, botSendMessageChan: botSendMessageChan}
	webHandler.setupRouter(database, router)
	return webHandler, nil
}

func currentUserProfile(ctx *gin.Context) model.UserProfile {
	maybePerson, found := ctx.Get(ctxKeyUserProfile)
	if !found {
		return model.NewUserProfile(0)
	}
	person, ok := maybePerson.(model.UserProfile)
	if !ok {
		log.Errorf("Count not cast store.Person from session")
		return model.NewUserProfile(0)
	}
	return person
}
