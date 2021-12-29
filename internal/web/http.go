// Package web implements the HTTP and websocket services for the frontend client and backend server.
package web

import (
	"crypto/tls"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/action"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type WebHandler interface {
	ListenAndServe() error
}

type web struct {
	httpServer *http.Server
	executor   action.Executor
	db         store.Store
	bot        discord.ChatBot
}

func (w web) ListenAndServe() error {
	log.WithFields(log.Fields{"service": "web", "status": "ready"}).Infof("Service status changed")
	return w.httpServer.ListenAndServe()
}

// New sets up the router and starts the API HTTP handlers
// This function blocks on the context
func New(db store.Store, bot discord.ChatBot, exec action.Executor) (WebHandler, error) {
	var httpServer *http.Server
	if config.General.Mode == config.Release {
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
			// Causes servers to use Go's default cipher suite preferences,
			// which are tuned to avoid attacks. Does nothing on clients.
			PreferServerCipherSuites: true,
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
	w := web{httpServer: httpServer, executor: exec, bot: bot, db: db}
	w.setupRouter(router, db, bot)
	return w, nil
}

func currentPerson(c *gin.Context) model.Person {
	p, found := c.Get("person")
	if !found {
		return model.NewPerson(0)
	}
	person, ok := p.(model.Person)
	if !ok {
		log.Warnf("Total not cast store.Person from session")
		return model.NewPerson(0)
	}
	return person
}
