package network

import (
	"errors"
	"log/slog"
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type networkHandler struct {
	networks domain.NetworkUsecase
}

func NewHandler(engine *gin.Engine, networks domain.NetworkUsecase, auth domain.AuthUsecase) {
	handler := networkHandler{networks: networks}

	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(auth.Middleware(domain.PModerator))
		mod.POST("/api/connections", handler.onAPIQueryConnections())
		mod.POST("/api/network", handler.onAPIQueryNetwork())
	}

	adminGrp := engine.Group("/")
	{
		admin := adminGrp.Use(auth.Middleware(domain.PAdmin))
		admin.GET("/api/network/update_db", handler.onAPIGetUpdateDB())
	}
}

func (h networkHandler) onAPIGetUpdateDB() gin.HandlerFunc {
	updateInProgress := atomic.Bool{}

	return func(ctx *gin.Context) {
		if !updateInProgress.Load() {
			go func() {
				updateInProgress.Store(true)

				if err := h.networks.RefreshLocationData(ctx); err != nil {
					slog.Error("Failed to update location data", log.ErrAttr(err))
				}

				updateInProgress.Store(false)
			}()
			ctx.JSON(http.StatusOK, gin.H{})
		} else {
			slog.Warn("Tried to start concurrent location update")
			ctx.JSON(http.StatusConflict, gin.H{})
		}
	}
}

func (h networkHandler) onAPIQueryNetwork() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.NetworkDetailsQuery
		if !httphelper.Bind(ctx, &req) {
			return
		}

		details, err := h.networks.QueryNetwork(ctx, req.IP)
		if err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, domain.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, details)
	}
}

func (h networkHandler) onAPIQueryConnections() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.ConnectionHistoryQuery
		if !httphelper.Bind(ctx, &req) {
			return
		}

		ipHist, totalCount, errIPHist := h.networks.QueryConnectionHistory(ctx, req)
		if errIPHist != nil && !errors.Is(errIPHist, domain.ErrNoResult) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errIPHist, domain.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, httphelper.NewLazyResult(totalCount, ipHist))
	}
}
