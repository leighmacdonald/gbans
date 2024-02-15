package network

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type networkHandler struct {
	nu domain.NetworkUsecase
}

func NewNetworkHandler(engine *gin.Engine, nu domain.NetworkUsecase, ath domain.AuthUsecase) {
	handler := networkHandler{nu: nu}

	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(ath.AuthMiddleware(domain.PModerator))
		mod.POST("/api/connections", handler.onAPIQueryPersonConnections())
	}
}

func (h networkHandler) onAPIQueryPersonConnections() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.ConnectionHistoryQueryFilter
		if !httphelper.Bind(ctx, &req) {
			return
		}

		ipHist, totalCount, errIPHist := h.nu.QueryConnectionHistory(ctx, req)
		if errIPHist != nil && !errors.Is(errIPHist, domain.ErrNoResult) {
			slog.Error("Failed to query connection history", log.ErrAttr(errIPHist))
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(totalCount, ipHist))
	}
}
