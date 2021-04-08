package service

import (
	"crypto/tls"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

const baseLayout = `<!doctype html>
    <html class="no-js" lang="en">
    <head>
        <meta charset="utf-8"/>
        <meta http-equiv="x-ua-compatible" content="ie=edge">
        <meta name="viewport" content="minimum-scale=1, initial-scale=1, width=device-width"/>
		<link rel="apple-touch-icon" sizes="180x180" href="/static/dist/apple-touch-icon.png">
		<link rel="icon" type="image/png" sizes="32x32" href="/static/dist/favicon-32x32.png">
		<link rel="icon" type="image/png" sizes="16x16" href="/static/dist/favicon-16x16.png">
		<link rel="manifest" href="/static/dist/site.webmanifest">
		<link rel="mask-icon" href="/static/dist/safari-pinned-tab.svg" color="#5bbad5">
		<meta name="msapplication-TileColor" content="#941739">
		<meta name="theme-color" content="#ffffff">
		<link rel="stylesheet" href="https://fonts.googleapis.com/css?family=Roboto:300,400,500,700&display=swap" />
		<link rel="stylesheet" href="https://fonts.googleapis.com/icon?family=Material+Icons" />
        <title>gbans</title>
    </head>
    <body>
    <div id="root"></div>
    <script src="/static/dist/bundle.js"></script>
    </body>
    </html>`

type StatusResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func initHTTP() {
	log.Infof("Starting HTTP service")
	go func() {
		httpServer = &http.Server{
			Addr:           config.HTTP.Addr(),
			Handler:        router,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}
		if config.HTTP.TLS {
			tlsVar := &tls.Config{
				// Causes servers to use Go's default ciphersuite preferences,
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
		if err := httpServer.ListenAndServe(); err != nil {
			log.Errorf("Error shutting down service: %v", err)
		}
	}()
	<-gCtx.Done()
}

func currentPerson(c *gin.Context) *model.Person {
	p, found := c.Get("person")
	if !found {
		return model.NewPerson(0)
	}
	person, ok := p.(*model.Person)
	if !ok {
		log.Warnf("Total not cast store.Person from session")
		return model.NewPerson(0)
	}
	return person
}
