package service

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type MetricsHandler struct {
	log *zap.Logger
}

func NewMetricsHandler(logger *zap.Logger, engine *gin.Engine) {
	handler := MetricsHandler{log: logger.Named("metrics")}
	engine.GET("/metrics", handler.prometheusHandler())
}

func (h MetricsHandler) prometheusHandler() gin.HandlerFunc {
	handler := promhttp.Handler()

	return func(ctx *gin.Context) {
		handler.ServeHTTP(ctx.Writer, ctx.Request)
	}
}
