package metrics

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type metricsHandler struct{}

func NewHandler(engine *gin.Engine) {
	handler := metricsHandler{}
	engine.GET("/metrics", handler.prometheusHandler())
}

func (h metricsHandler) prometheusHandler() gin.HandlerFunc {
	handler := promhttp.Handler()

	return func(ctx *gin.Context) {
		handler.ServeHTTP(ctx.Writer, ctx.Request)
	}
}
