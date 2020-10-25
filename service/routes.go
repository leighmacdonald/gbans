package service

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/memstore"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/config"
	"github.com/leighmacdonald/golib"
)

type Route string

const (
	routeDist           Route = "dist"
	routeHome           Route = "home"
	routeServerAPIBan   Route = "server_api_ban"
	routeServerAPICheck Route = "server_api_check"
)

func initRouter() {
	s := memstore.NewStore([]byte(golib.RandomString(64)))
	ses := sessions.Sessions("gbans", s)
	session := router.Group("")
	session.Use(ses)
	routesApply(router, session)
}

func routesApply(r *gin.Engine, session *gin.RouterGroup) {
	// Guest User
	r.Static(routeRaw("dist"), config.HTTP.StaticPath)
	r.GET(routeRaw("home"), onIndex())
	r.POST("/v1/auth", onPostAuth())
	// Authenticated User
	session.Use(authMiddleWare())

	// API
	authed := router.Group("/", checkServerAuth)
	authed.GET(string(routeServerAPIBan), onGetBan())
	authed.POST(string(routeServerAPICheck), onPostCheck())
}

func init() {
	routes = map[Route]string{
		routeHome:           "/",
		routeDist:           "/dist",
		routeServerAPIBan:   "/v1/ban",
		routeServerAPICheck: "/v1/check",
	}
}
