package network

import (
	"errors"
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"go.uber.org/zap"
)

type NetworkHandler struct {
	nu  domain.NetworkUsecase
	log *zap.Logger
}

func NewNetworkHandler(log *zap.Logger, engine *gin.Engine, nu domain.NetworkUsecase, ath domain.AuthUsecase) {
	handler := NetworkHandler{log: log.Named("network"), nu: nu}

	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(ath.AuthMiddleware(domain.PModerator))
		mod.POST("/api/connections", handler.onAPIQueryPersonConnections())
	}
}

func (h NetworkHandler) onAPIQueryPersonConnections() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.ConnectionHistoryQueryFilter
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		ipHist, totalCount, errIPHist := h.nu.QueryConnectionHistory(ctx, req)
		if errIPHist != nil && !errors.Is(errIPHist, domain.ErrNoResult) {
			log.Error("Failed to query connection history", zap.Error(errIPHist))
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(totalCount, ipHist))
	}
}
