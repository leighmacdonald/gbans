package service

import (
	"github.com/gin-gonic/gin"
)

func onIndex() gin.HandlerFunc {
	return func(c *gin.Context) {
		render(c, "home", defaultArgs(c))
	}
}

func onServers() gin.HandlerFunc {
	return func(c *gin.Context) {
		serverStateMu.RLock()
		state := serverState
		serverStateMu.RUnlock()
		a := defaultArgs(c)
		a.V["servers"] = state
		render(c, "servers", a)
	}
}
