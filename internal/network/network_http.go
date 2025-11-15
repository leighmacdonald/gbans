package network

import (
	"errors"
	"log/slog"
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type NetworkHandler struct { //nolint:revive
	Networks
}

func NewNetworkHandler(engine *gin.Engine, authenticator httphelper.Authenticator, networks Networks) {
	handler := NetworkHandler{Networks: networks}

	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(authenticator.Middleware(permission.Moderator))
		mod.POST("/api/connections", handler.onAPIQueryConnections())
		mod.POST("/api/network", handler.onAPIQueryNetwork())
	}

	adminGrp := engine.Group("/")
	{
		admin := adminGrp.Use(authenticator.Middleware(permission.Admin))
		admin.GET("/api/network/update_db", handler.onAPIGetUpdateDB())
	}
}

func (h NetworkHandler) onAPIGetUpdateDB() gin.HandlerFunc {
	updateInProgress := atomic.Bool{}

	return func(ctx *gin.Context) {
		if !updateInProgress.Load() {
			go func() {
				updateInProgress.Store(true)

				if err := h.RefreshLocationData(ctx); err != nil {
					slog.Error("Failed to update location data", slog.String("error", err.Error()))
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

func (h NetworkHandler) onAPIQueryNetwork() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req, ok := httphelper.BindJSON[DetailsQuery](ctx)
		if !ok {
			return
		}

		details, err := h.QueryNetwork(ctx, req.IP)
		if err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, details)
	}
}

func (h NetworkHandler) onAPIQueryConnections() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req, ok := httphelper.BindJSON[ConnectionHistoryQuery](ctx)
		if !ok {
			return
		}

		ipHist, totalCount, errIPHist := h.QueryConnectionHistory(ctx, req)
		if errIPHist != nil && !errors.Is(errIPHist, database.ErrNoResult) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errIPHist, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, httphelper.NewLazyResult(totalCount, ipHist))
	}
}
