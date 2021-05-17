package web

import (
	"context"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v2/steamid"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"time"
)

func initRouter(r *gin.Engine, logMsgChan chan LogPayload) {
	defaultRoute := func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Status(http.StatusNotFound)
			return
		}
		c.Data(200, gin.MIMEHTML, []byte(baseLayout))
	}
	r.Use(gin.Logger())
	// Dont use session for static assets
	// Note that we only use embedded assets for !release modes
	// This is to allow us the ability to develop the frontend without needing to
	// compile+re-embed the assets on each change.
	if config.General.Mode == config.Release {
		r.StaticFS("/assets", http.FS(content))
	} else {
		r.StaticFS("/assets/dist", http.Dir(config.HTTP.StaticPath))
	}
	//router.GET(routeRaw(string(routeHome)), )
	authRequired := func(level model.Privilege) gin.HandlerFunc {
		type header struct {
			Authorization string `header:"Authorization"`
		}
		return func(c *gin.Context) {
			hdr := header{}
			if err := c.ShouldBindHeader(&hdr); err != nil {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			pcs := strings.Split(hdr.Authorization, " ")
			if len(pcs) != 2 {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			claims := &authClaims{}
			tkn, errC := jwt.ParseWithClaims(pcs[1], claims, getTokenKey)
			if errC != nil {
				if errC == jwt.ErrSignatureInvalid {
					c.AbortWithStatus(http.StatusForbidden)
					return
				}
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			if !tkn.Valid {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			if !steamid.SID64(claims.SteamID).Valid() {
				c.AbortWithStatus(http.StatusForbidden)
				log.Warnf("Invalid steamID")
				return
			}
			cx, cancel := context.WithTimeout(context.Background(), time.Second*6)
			defer cancel()
			loggedInPerson, err := store.GetPersonBySteamID(cx, steamid.SID64(claims.SteamID))
			if err != nil {
				log.Errorf("Failed to load persons session user: %v", err)
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			if level > loggedInPerson.PermissionLevel {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.Set("person", loggedInPerson)
			c.Next()
		}
	}
	r.NoRoute(defaultRoute)
	r.GET("/login/success", onLoginSuccess())
	r.GET("/auth/callback", onOpenIDCallback())
	r.GET("/api/ban/:ban_id", onAPIGetBanByID())
	r.POST("/api/bans", onAPIGetBans())
	r.GET("/api/profile", onAPIProfile())
	r.GET("/api/servers", onAPIGetServers())
	r.GET("/api/stats", onAPIGetStats())
	r.GET("/api/filtered_words", onAPIGetFilteredWords())
	r.GET("/api/players", onAPIGetPlayers())

	// Game server plugin routes
	r.POST("/api/server_auth", onSAPIPostServerAuth())
	// Server Auth Request
	serverAuth := r.Use(authMiddleWare())
	serverAuth.POST("/api/ping_mod", onPostPingMod())
	serverAuth.POST("/api/check", onPostServerCheck())

	// Relay
	r.POST("/api/log", onPostLogAdd(logMsgChan))

	// Basic logged in user
	authed := r.Use(authRequired(model.PAuthenticated))
	authed.GET("/api/current_profile", onAPICurrentProfile())
	authed.GET("/api/auth/refresh", onTokenRefresh())
	authed.GET("/api/auth/logout", onGetLogout())

	// Moderator access
	modRoute := r.Use(authRequired(model.PModerator))
	modRoute.POST("/api/ban", onAPIPostBanCreate())

	// Admin access
	modAdmin := r.Use(authRequired(model.PAdmin))
	modAdmin.POST("/api/server", onAPIPostServer())
}
